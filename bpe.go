package bpe

import (
	"bufio"
	"container/heap"
	"encoding/binary"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// TokenID is a numerical identifier of the subword token
type TokenID uint32

// TokenIDPair is a concatenation of two TokenIDs that is used as the key type in rule2id map.
type TokenIDPair uint64

// EncodedString is a sequence of subword token identifiers
type EncodedString []TokenID

const (
	unkToken = "<UNK>"
	padToken = "<PAD>"
	bosToken = "<BOS>"
	eosToken = "<EOS>"
)

// EncodingConfig is a configuration for encoding of strings
type EncodingConfig struct {
	bos     bool
	eos     bool
	reverse bool
}

type rule struct {
	left   TokenID
	right  TokenID
	result TokenID
}

type specialTokens struct {
	unk int32
	pad int32
	bos int32
	eos int32
}

// Model is a Byte-Pair encoding model, which supports encoding and decoding text into sequences
// of most frequent subword tokens
type Model struct {
	char2id       map[rune]TokenID
	id2char       map[TokenID]rune
	rules         []rule
	rule2id       map[TokenIDPair]int
	recipe        map[TokenID]EncodedString
	revRecipe     map[string]TokenID
	specialTokens specialTokens
	spaceID       TokenID
}

func newModel(nRules int) *Model {
	return &Model{
		make(map[rune]TokenID),
		make(map[TokenID]rune),
		make([]rule, nRules),
		make(map[TokenIDPair]int),
		make(map[TokenID]EncodedString),
		make(map[string]TokenID),
		specialTokens{-1, -1, -1, -1},
		0,
	}
}

func newTokenIDPair(left, right TokenID) TokenIDPair {
	return (TokenIDPair(left) << 32) + TokenIDPair(right)
}

// DecodeToken converts the sequence of chars' ids into the string -
// sequence of the corresponding chars
func DecodeToken(token EncodedString, id2char map[TokenID]rune) (string, error) {
	word := ""
	for _, id := range token {
		if char, ok := id2char[id]; ok {
			word = word + string(char)
		} else {
			logrus.Errorf("Decode failure: %d token id has no corresponding char", id)
			return "", errors.New("key not found in id2char")
		}
	}
	return word, nil
}

func (s specialTokens) toBinary() []byte {
	bytesArray := make([]byte, 16)
	binary.BigEndian.PutUint32(bytesArray, uint32(s.unk))
	binary.BigEndian.PutUint32(bytesArray[4:], uint32(s.pad))
	binary.BigEndian.PutUint32(bytesArray[8:], uint32(s.bos))
	binary.BigEndian.PutUint32(bytesArray[12:], uint32(s.eos))
	return bytesArray
}

func binaryToSpecialTokens(bytesArray []byte) (specialTokens, error) {
	var s specialTokens
	if len(bytesArray) < 16 {
		logrus.Error("Bytes array length is too small")
		return s, errors.New("bytes array is too small")
	}
	s.unk = int32(binary.BigEndian.Uint32(bytesArray))
	s.pad = int32(binary.BigEndian.Uint32(bytesArray[4:]))
	s.bos = int32(binary.BigEndian.Uint32(bytesArray[8:]))
	s.eos = int32(binary.BigEndian.Uint32(bytesArray[12:]))
	return s, nil
}

func (r rule) toBinary() []byte {
	bytesArray := make([]byte, 12)
	binary.BigEndian.PutUint32(bytesArray, uint32(r.left))
	binary.BigEndian.PutUint32(bytesArray[4:], uint32(r.right))
	binary.BigEndian.PutUint32(bytesArray[8:], uint32(r.result))
	return bytesArray
}

func binaryToRule(bytesArray []byte) (rule, error) {
	var r rule
	if len(bytesArray) < 12 {
		logrus.Error("Bytes array length is too small")
		return r, errors.New("bytes array is too small")
	}
	r.left = TokenID(binary.BigEndian.Uint32(bytesArray))
	r.right = TokenID(binary.BigEndian.Uint32(bytesArray[4:]))
	r.result = TokenID(binary.BigEndian.Uint32(bytesArray[8:]))
	return r, nil
}

// ReadModel loads the BPE model from the binary dump
func ReadModel(reader io.Reader) (*Model, error) {
	buf := make([]byte, 4)
	var nChars, nRules int
	if _, err := io.ReadFull(reader, buf); err != nil {
		logrus.Error("Broken input: ", err)
		return &Model{}, err
	}
	nChars = int(binary.BigEndian.Uint32(buf))
	if _, err := io.ReadFull(reader, buf); err != nil {
		logrus.Error("Broken input: ", err)
		return &Model{}, err
	}
	nRules = int(binary.BigEndian.Uint32(buf))

	model := newModel(nRules)
	minCharID := TokenID(0)
	for i := 0; i < nChars; i++ {
		var char rune
		var charID TokenID
		if _, err := io.ReadFull(reader, buf); err != nil {
			logrus.Error("Broken input: ", err)
			return &Model{}, err
		}
		char = rune(binary.BigEndian.Uint32(buf))
		if _, err := io.ReadFull(reader, buf); err != nil {
			logrus.Error("Broken input: ", err)
			return &Model{}, err
		}
		charID = TokenID(binary.BigEndian.Uint32(buf))
		model.char2id[char] = charID
		model.id2char[charID] = char
		model.recipe[charID] = EncodedString{charID}
		model.revRecipe[string(char)] = charID
		if charID < minCharID || minCharID == 0 {
			minCharID = charID
			model.spaceID = charID
		}
	}
	ruleBuf := make([]byte, 12)
	for i := 0; i < nRules; i++ {
		if _, err := io.ReadFull(reader, ruleBuf); err != nil {
			logrus.Error("Broken input: ", err)
			return &Model{}, err
		}
		rule, err := binaryToRule(ruleBuf)
		if err != nil {
			return model, err
		}
		if _, ok := model.recipe[rule.left]; !ok {
			logrus.Errorf("%d: token id not described before", rule.left)
			return model, errors.New("token id is impossible")
		}
		if _, ok := model.recipe[rule.right]; !ok {
			logrus.Errorf("%d: token id not described before", rule.right)
			return model, errors.New("token id is impossible")
		}
		model.rules[i] = rule
		model.rule2id[newTokenIDPair(rule.left, rule.right)] = i
		model.recipe[rule.result] = append(model.recipe[rule.left], model.recipe[rule.right]...)
		resultString, err := DecodeToken(model.recipe[rule.result], model.id2char)
		if err != nil {
			logrus.Error("Unexpected token id inside the rules: ", err)
			return model, err
		}
		model.revRecipe[resultString] = rule.result
	}
	specialTokensBuf := make([]byte, 16)
	if _, err := io.ReadFull(reader, specialTokensBuf); err != nil {
		logrus.Error("Broken input: ", err)
		return &Model{}, err
	}
	specials, err := binaryToSpecialTokens(specialTokensBuf)
	if err != nil {
		return model, err
	}
	model.specialTokens = specials
	model.revRecipe[bosToken] = TokenID(specials.bos)
	model.revRecipe[eosToken] = TokenID(specials.eos)
	model.revRecipe[unkToken] = TokenID(specials.unk)
	model.revRecipe[padToken] = TokenID(specials.pad)
	return model, err
}

// IDToToken returns string token corresponding to the given token id.
// If replaceSpace is true, special space token that is used for marking starts of words
// will be replaced with space.
func (m Model) IDToToken(id TokenID, replaceSpace bool) (string, error) {
	if _, ok := m.recipe[id]; !ok {
		switch id {
		case TokenID(m.specialTokens.unk):
			return unkToken, nil
		case TokenID(m.specialTokens.pad):
			return padToken, nil
		case TokenID(m.specialTokens.bos):
			return bosToken, nil
		case TokenID(m.specialTokens.eos):
			return eosToken, nil
		default:
			logrus.Errorf("%d: token id is impossible", id)
			return "", errors.New("token id is impossible")
		}
	}
	encodedToken, _ := m.recipe[id]
	if encodedToken[0] == m.spaceID && replaceSpace {
		token, err := DecodeToken(encodedToken[1:], m.id2char)
		if err != nil {
			return "", err
		}
		return " " + token, nil
	}
	return DecodeToken(encodedToken, m.id2char)
}

// DecodeSentence decodes a sequence of token ids in a text sentence - string of words
// with spaces in between
func (m Model) DecodeSentence(encodedSentence EncodedString) (string, error) {
	sentence := ""
	for _, tokenID := range encodedSentence {
		token, err := m.IDToToken(tokenID, true)
		if err != nil {
			return sentence, err
		}
		sentence += token
	}
	if string(sentence[0]) == " " {
		sentence = sentence[1:]
	}
	if sentence[:len(bosToken)+1] == bosToken+" " {
		sentence = bosToken + sentence[len(bosToken)+1:]
	}
	return sentence, nil
}

// DecodeSentences decodes a sequence of encoded sentences - sequences of token ids -
// into a sequence of corresponding text sentences
func (m Model) DecodeSentences(encodedSentences []EncodedString) ([]string, error) {
	sentences := make([]string, len(encodedSentences))
	for i, encodedSentence := range encodedSentences {
		sentence, err := m.DecodeSentence(encodedSentence)
		if err != nil {
			return sentences, err
		}
		sentences[i] = sentence
	}
	return sentences, nil
}

// DecodeFromStream decodes a sequence of encoded sentences written in an input stream
// using Model.DecodeSentences
func (m Model) DecodeFromStream(reader io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(reader)
	var sentences []string
	for scanner.Scan() {
		numbers := strings.Fields(scanner.Text())
		var encodedSentence = make([]TokenID, len(numbers))
		for i, number := range numbers {
			id, err := strconv.Atoi(number)
			if err != nil {
				return nil, err
			}
			encodedSentence[i] = TokenID(id)
		}
		sentence, err := m.DecodeSentence(encodedSentence)
		if err != nil {
			return sentences, err
		}
		sentences = append(sentences, sentence)
	}
	if err := scanner.Err(); err != nil {
		return sentences, err
	}
	return sentences, nil
}

type encodingToken struct {
	id   TokenID
	prev int
	next int
}

type mergeEvent struct {
	priority int
	pos      int
}

type mergeQueue []*mergeEvent

func (mq mergeQueue) Len() int { return len(mq) }

func (mq mergeQueue) Less(i, j int) bool {
	return mq[i].priority < mq[j].priority ||
		mq[i].priority == mq[j].priority && mq[i].pos < mq[j].pos
}

func (mq mergeQueue) Swap(i, j int) {
	mq[i], mq[j] = mq[j], mq[i]
}

func (mq *mergeQueue) Push(x interface{}) {
	*mq = append(*mq, x.(*mergeEvent))
}

func (mq *mergeQueue) Pop() interface{} {
	old := *mq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	*mq = old[0 : n-1]
	return item
}

// EncodeSentence takes a string of space-separated words and tokenizes each word
// according to the BPE rules. Through encodingConfig one can state whether to add BOS, EOS tokens
// and whether to reverse the output sequences. EncodeSentence returns the numerical encoding
// of the sentence.
func (m Model) EncodeSentence(sentence string, encodingConfig EncodingConfig,
) (EncodedString, error) {
	var encodedSentence EncodedString

	if encodingConfig.bos {
		if m.specialTokens.bos == -1 {
			logrus.Error("Cannot use bos - model was trained without it")
			return encodedSentence, errors.New("model was trained withous bos")
		}
		encodedSentence = append(encodedSentence, TokenID(m.specialTokens.bos))
	}
	for _, word := range strings.Fields(sentence) {
		if len(word) == 0 {
			continue
		}
		var encodedWord = []encodingToken{{m.spaceID, -1, 1}}
		var pendingMerges mergeQueue
		// Check whether two consecutive tokens can be merged and if so add merge suggestion to
		// the priority queue
		pushIfRuleExists := func(leftPos int) {
			rightPos := encodedWord[leftPos].next
			ruleCandidate := newTokenIDPair(encodedWord[leftPos].id, encodedWord[rightPos].id)
			if priority, ok := m.rule2id[ruleCandidate]; ok {
				heap.Push(&pendingMerges, &mergeEvent{priority, leftPos})
			}
		}
		// Build linked list corresponding to the word's split on known chars and unknown tokens
		unknownToken := false
		for _, char := range word {
			if charID, ok := m.char2id[char]; ok {
				if unknownToken {
					encodedWord = append(encodedWord,
						encodingToken{TokenID(m.specialTokens.unk), len(encodedWord) - 1,
							len(encodedWord) + 1})
					unknownToken = false
				}
				encodedWord = append(encodedWord,
					encodingToken{charID, len(encodedWord) - 1, len(encodedWord) + 1})
				pushIfRuleExists(len(encodedWord) - 2)
			} else {
				unknownToken = true
			}
		}
		if unknownToken {
			encodedWord = append(encodedWord,
				encodingToken{TokenID(m.specialTokens.unk), len(encodedWord) - 1,
					len(encodedWord) + 1})
		}
		encodedWord[len(encodedWord)-1].next = -1
		// Perform merges of subword tokens in the word according to the BPE model rules
		for len(pendingMerges) > 0 {
			event := heap.Pop(&pendingMerges).(*mergeEvent)
			proposedRule := m.rules[event.priority]
			leftPos := event.pos
			leftToken := encodedWord[leftPos]
			rightPos := leftToken.next
			if rightPos == -1 {
				continue
			}
			rightToken := encodedWord[rightPos]
			// Check that the tokens suggested for the merge have not changed
			if proposedRule.left != leftToken.id || proposedRule.right != rightToken.id {
				continue
			}
			// Create token as a merge of the right and the left ones
			leftToken.next = rightToken.next
			leftToken.id = proposedRule.result
			// Put merged token on the place of the left token
			encodedWord[leftPos] = leftToken
			// Put 'empty' token on the place of the right token
			encodedWord[rightPos] = encodingToken{0, -1, -1}
			// Add suggestions for merges for the new merged token
			if rightToken.next != -1 {
				encodedWord[rightToken.next].prev = leftPos
				pushIfRuleExists(leftPos)
			}
			if leftToken.prev != -1 {
				pushIfRuleExists(leftToken.prev)
			}
		}
		// Retrieve all tokens that are left and append them to the result for the whole sentence
		for pos := 0; pos > -1; {
			encodedSentence = append(encodedSentence, encodedWord[pos].id)
			pos = encodedWord[pos].next
		}
	}
	if encodingConfig.eos {
		if m.specialTokens.eos == -1 {
			logrus.Error("Cannot use eos - model was trained without it")
			return encodedSentence, errors.New("model was trained withous eos")
		}
		encodedSentence = append(encodedSentence, TokenID(m.specialTokens.eos))
	}
	if encodingConfig.reverse {
		for i := 0; i < len(encodedSentence)/2; i++ {
			encodedSentence[i], encodedSentence[len(encodedSentence)-i-1] =
				encodedSentence[len(encodedSentence)-i-1], encodedSentence[i]
		}
	}
	return encodedSentence, nil
}

// EncodeSentences takes a sequence of strings which consist of space-separated words and tokenizes
// each word according to the BPE rules. Through encodingConfig one can state whether to add BOS
// and EOS tokens (beginning and end of sentence) and whether to reverse the output sequences.
// EncodeSentences returns the numerical encodings of the sentences.
func (m Model) EncodeSentences(sentences []string, encodingConfig EncodingConfig) ([]EncodedString,
	error) {
	encodedSentence := make([]EncodedString, len(sentences))
	for i, sentence := range sentences {
		sentenceIds, err := m.EncodeSentence(sentence, encodingConfig)
		if err != nil {
			return encodedSentence, err
		}
		encodedSentence[i] = sentenceIds
	}
	return encodedSentence, nil
}

// EncodeStream reads a sequence of strings which consist of space-separated words from the given
// stream and tokenizes each word according to the BPE rules. Through encodingConfig one can state
// whether to add BOS and EOS tokens (beginning and end of sentence) and whether to reverse the
// output sequences. EncodeStream returns the numerical encodings of the sentences.
func (m Model) EncodeStream(reader io.Reader, encodingConfig EncodingConfig) ([]EncodedString,
	error) {
	scanner := bufio.NewScanner(reader)
	var encodedSentence []EncodedString
	for scanner.Scan() {
		sentenceIds, err := m.EncodeSentence(scanner.Text(), encodingConfig)
		if err != nil {
			return encodedSentence, err
		}
		encodedSentence = append(encodedSentence, sentenceIds)
	}
	err := scanner.Err()
	return encodedSentence, err
}
