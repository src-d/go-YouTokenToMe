package bpe

import (
	"bufio"
	"fmt"
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
	unk int
	pad int
	bos int
	eos int
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

// ReadModelFromText loads the BPE model from the text dump
func ReadModelFromText(reader io.Reader) (*Model, error) {
	scanner := bufio.NewScanner(reader)
	var nChars, nRules int
	scanner.Scan()
	_, err := fmt.Sscanf(scanner.Text(), "%d %d", &nChars, &nRules)
	if err != nil {
		logrus.Fatal("Wrong input format: ", err)
		return &Model{}, err
	}
	model := newModel(nRules)
	for i := 0; i < nChars; i++ {
		var char rune
		var charID TokenID
		scanner.Scan()
		_, err = fmt.Sscanf(scanner.Text(), "%d %d", &char, &charID)
		if err != nil {
			logrus.Fatal("Wrong input format: ", err)
			return model, err
		}
		model.char2id[char] = charID
		model.id2char[charID] = char
		model.recipe[charID] = EncodedToken{charID}
		model.revRecipe[string(char)] = charID
	}
	for i := 0; i < nRules; i++ {
		var rule rule
		scanner.Scan()
		_, err = fmt.Sscanf(scanner.Text(), "%d %d %d", &rule.left, &rule.right, &rule.result)
		if err != nil {
			logrus.Fatal("Wrong input format: ", err)
			return model, err
		}
		model.rules[i] = rule
		model.recipe[rule.result] = append(model.recipe[rule.left], model.recipe[rule.right]...)
		resultString, err := DecodeToken(model.recipe[rule.result], model.id2char)
		if err != nil {
			logrus.Fatal("Unexpected token id inside the rules: ", err)
			return model, err
		}
		model.revRecipe[resultString] = rule.result
	}
	scanner.Scan()
	_, err = fmt.Sscanf(scanner.Text(), "%d %d %d %d", &model.specialTokens.unk,
		&model.specialTokens.pad, &model.specialTokens.bos, &model.specialTokens.eos)
	if err != nil {
		logrus.Fatal("Wrong input format: ", err)
		return model, err
	}
	return model, nil
}
