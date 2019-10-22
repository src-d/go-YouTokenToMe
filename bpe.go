package bpe

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/sirupsen/logrus"
)

// TokenID is a numerical identitier of the subword token
type TokenID uint32

// EncodedToken is a sequence of subword tokens ids
type EncodedToken []TokenID

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
	recipe        map[TokenID]EncodedToken
	revRecipe     map[string]TokenID
	specialTokens specialTokens
}

func newModel(nRules int) *Model {
	return &Model{
		make(map[rune]TokenID),
		make(map[TokenID]rune),
		make([]rule, nRules),
		make(map[TokenID]EncodedToken),
		make(map[string]TokenID),
		specialTokens{-1, -1, -1, -1},
	}
}

// DecodeToken converts the sequence of chars' ids into the string -
// sequence of the corresponding chars
func DecodeToken(token EncodedToken, id2char map[TokenID]rune) (string, error) {
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
		model.recipe[charID] = EncodedToken{charID}
		model.revRecipe[string(char)] = charID
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
		model.rules[i] = rule
		if _, ok := model.recipe[rule.left]; !ok {
			logrus.Errorf("%d: token id not described before", rule.left)
			return model, errors.New("key not found in id2char")
		}
		if _, ok := model.recipe[rule.right]; !ok {
			logrus.Errorf("%d: token id not described before", rule.right)
			return model, errors.New("key not found in id2char")
		}
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
	model.specialTokens = specials
	return model, err
}
