package bpe

import (
	"bufio"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
)

type TokenId int32

type DecodedToken []TokenId

type Rule struct {
	left   TokenId
	right  TokenId
	result TokenId
}

type SpecialTokens struct {
	unk TokenId
	pad TokenId
	bos TokenId
	eos TokenId
}

type Model struct {
	char2id       map[rune]TokenId
	id2char       map[TokenId]rune
	rules         []Rule
	recipe        map[TokenId]DecodedToken
	revRecipe     map[string]TokenId
	specialTokens SpecialTokens
}

func NewModel(nRules int) *Model {
	return &Model{
		make(map[rune]TokenId),
		make(map[TokenId]rune),
		make([]Rule, nRules),
		make(map[TokenId]DecodedToken),
		make(map[string]TokenId),
		SpecialTokens{-1, -1, -1, -1},
	}
}

func DecodedTokenToString(token DecodedToken, id2char map[TokenId]rune) (string, error) {
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

func ReadModel(reader io.Reader) (*Model, error) {
	scanner := bufio.NewScanner(reader)
	var nChars, nRules int
	scanner.Scan()
	_, err := fmt.Sscanf(scanner.Text(), "%d %d", &nChars, &nRules)
	if err != nil {
		logrus.Fatal("Wrong input format: ", err)
		return &Model{}, err
	}
	model := NewModel(nRules)
	model.rules = make([]Rule, nRules)
	for i := 0; i < nChars; i++ {
		var char rune
		var charId TokenId
		scanner.Scan()
		_, err = fmt.Sscanf(scanner.Text(), "%d %d", &char, &charId)
		if err != nil {
			logrus.Fatal("Wrong input format: ", err)
			return model, err
		}
		model.char2id[char] = charId
		model.id2char[charId] = char
		model.recipe[charId] = DecodedToken{charId}
		model.revRecipe[string(char)] = charId
	}
	for i := 0; i < nRules; i++ {
		var rule Rule
		scanner.Scan()
		_, err = fmt.Sscanf(scanner.Text(), "%d %d %d", &rule.left, &rule.right, &rule.result)
		if err != nil {
			logrus.Fatal("Wrong input format: ", err)
			return model, err
		}
		model.rules[i] = rule
		model.recipe[rule.result] = append(model.recipe[rule.left], model.recipe[rule.right]...)
		resultString, err := DecodedTokenToString(model.recipe[rule.result], model.id2char)
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
