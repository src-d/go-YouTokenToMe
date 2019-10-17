package bpe

import (
	"bufio"
	"encoding/binary"
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
			logrus.Fatalf("%d key not found in id2char", id)
		}
	}
	return word, nil
}

func specialTokensToBin(specials specialTokens) []byte {
	bytesArray := make([]byte, 16)
	binary.BigEndian.PutUint32(bytesArray, uint32(specials.unk))
	binary.BigEndian.PutUint32(bytesArray[4:], uint32(specials.pad))
	binary.BigEndian.PutUint32(bytesArray[8:], uint32(specials.bos))
	binary.BigEndian.PutUint32(bytesArray[12:], uint32(specials.eos))
	return bytesArray
}

func binToSpecialTokens(bytesArray []byte) specialTokens {
	var s specialTokens
	s.unk = int32(binary.BigEndian.Uint32(bytesArray))
	s.pad = int32(binary.BigEndian.Uint32(bytesArray[4:]))
	s.bos = int32(binary.BigEndian.Uint32(bytesArray[8:]))
	s.eos = int32(binary.BigEndian.Uint32(bytesArray[12:]))
	return s
}

func ruleToBin(rule rule) []byte {
	bytesArray := make([]byte, 12)
	binary.BigEndian.PutUint32(bytesArray, uint32(rule.left))
	binary.BigEndian.PutUint32(bytesArray[4:], uint32(rule.right))
	binary.BigEndian.PutUint32(bytesArray[8:], uint32(rule.result))
	return bytesArray
}

func binToRule(bytesArray []byte) rule {
	var r rule
	r.left = TokenID(binary.BigEndian.Uint32(bytesArray))
	r.right = TokenID(binary.BigEndian.Uint32(bytesArray[4:]))
	r.result = TokenID(binary.BigEndian.Uint32(bytesArray[8:]))
	return r
}

// ReadModelFromBinary loads the BPE model from the binary dump
func ReadModelFromBinary(reader io.Reader) (*Model, error) {
	bytesReader := bufio.NewReader(reader)
	buf := make([]byte, 4)
	var nChars, nRules int
	_, err := bytesReader.Read(buf)
	if err != nil {
		logrus.Fatal("Broken input: ", err)
		return &Model{}, err
	}
	nChars = int(binary.BigEndian.Uint32(buf))
	_, err = bytesReader.Read(buf)
	if err != nil {
		logrus.Fatal("Broken input: ", err)
		return &Model{}, err
	}
	nRules = int(binary.BigEndian.Uint32(buf))

	model := newModel(nRules)
	for i := 0; i < nChars; i++ {
		var char rune
		var charID TokenID
		_, err = bytesReader.Read(buf)
		if err != nil {
			logrus.Fatal("Broken input: ", err)
			return &Model{}, err
		}
		char = rune(binary.BigEndian.Uint32(buf))
		_, err = bytesReader.Read(buf)
		if err != nil {
			logrus.Fatal("Broken input: ", err)
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
		_, err = bytesReader.Read(ruleBuf)
		if err != nil {
			logrus.Fatal("Broken input: ", err)
			return &Model{}, err
		}
		rule := binToRule(ruleBuf)
		model.rules[i] = rule
		model.recipe[rule.result] = append(model.recipe[rule.left], model.recipe[rule.right]...)
		resultString, err := DecodeToken(model.recipe[rule.result], model.id2char)
		if err != nil {
			logrus.Fatal("Unexpected token id inside the rules: ", err)
			return model, err
		}
		model.revRecipe[resultString] = rule.result
	}
	specialTokensBuf := make([]byte, 16)
	_, err = bytesReader.Read(specialTokensBuf)
	if err != nil {
		logrus.Fatal("Broken input: ", err)
		return &Model{}, err
	}
	model.specialTokens = binToSpecialTokens(specialTokensBuf)
	return model, nil
}
