package bpe

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var BPE = Model{
	map[rune]TokenID{97: 8, 98: 7, 99: 6, 100: 5, 95: 4},
	map[TokenID]rune{4: 95, 5: 100, 6: 99, 7: 98, 8: 97},
	[]rule{{4, 8, 9}, {4, 6, 10}, {4, 5, 11}, {4, 7, 12}, {8, 7, 13}, {8, 8, 14}},
	map[PairTokenID]int{PairTokenID((4 << 32) + 8): 0, PairTokenID((4 << 32) + 6): 1,
		PairTokenID((4 << 32) + 5): 2, PairTokenID((4 << 32) + 7): 3,
		PairTokenID((8 << 32) + 7): 4, PairTokenID((8 << 32) + 8): 5},
	map[TokenID]EncodedString{4: {4}, 5: {5}, 6: {6}, 7: {7}, 8: {8}, 9: {4, 8},
		10: {4, 6}, 11: {4, 5}, 12: {4, 7}, 13: {8, 7}, 14: {8, 8}},
	map[string]TokenID{"a": 8, "b": 7, "c": 6, "d": 5, "_": 4, "_a": 9, "_b": 12,
		"_c": 10, "_d": 11, "ab": 13, "aa": 14, "<PAD>": 0, "<UNK>": 1, "<BOS>": 2, "<EOS>": 3},
	specialTokens{1, 0, 2, 3},
	4,
}

func TestNewModel(t *testing.T) {
	model := newModel(10)
	require.Equal(t, 10, len(model.rules))
}

func TestDecodeToken(t *testing.T) {
	req := require.New(t)
	id2char := map[TokenID]rune{1: []rune("a")[0], 2: []rune("b")[0], 3: []rune("c")[0]}
	word, err := DecodeToken(EncodedString{1, 2, 1, 3, 3}, id2char)
	req.NoError(err)
	req.Equal("abacc", word)

	word, err = DecodeToken(EncodedString{1, 2, 4, 3, 3}, id2char)
	req.Error(err)
}

func TestSpecialTokens_ToBinary(t *testing.T) {
	specials := specialTokens{1, 259, 2*256*256 + 37*256 + 2, -256 * 256 * 256 * 127}
	bytesArray := []byte{0, 0, 0, 1, 0, 0, 1, 3, 0, 2, 37, 2, 129, 0, 0, 0}
	require.Equal(t, bytesArray, specials.toBinary())
}

func TestBinaryToSpecialTokens(t *testing.T) {
	req := require.New(t)
	bytesArray := []byte{0, 0, 0, 1, 0, 0, 1, 3, 0, 2, 37, 2, 129, 0, 0, 0}
	expected := specialTokens{1, 259, 2*256*256 + 37*256 + 2, -256 * 256 * 256 * 127}
	specials, err := binaryToSpecialTokens(bytesArray)
	req.NoError(err)
	req.Equal(expected, specials)
	bytesArray = []byte{0, 0, 0, 1, 0, 0, 1, 3, 0, 2, 37, 2, 129, 0, 0}
	specials, err = binaryToSpecialTokens(bytesArray)
	req.Error(err)
	bytesArray = []byte{}
	specials, err = binaryToSpecialTokens(bytesArray)
	req.Error(err)
}

func TestRule_ToBinary(t *testing.T) {
	rule := rule{1, 2, 257}
	bytesArray := []byte{0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 1, 1}
	require.Equal(t, bytesArray, rule.toBinary())
}

func TestBinaryToRule(t *testing.T) {
	req := require.New(t)
	expected := rule{1, 2, 257}
	bytesArray := []byte{0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 1, 1}
	rule, err := binaryToRule(bytesArray)
	req.NoError(err)
	req.Equal(expected, rule)
	bytesArray = []byte{0, 0, 0, 0, 0, 0, 2, 0, 0, 1, 1}
	rule, err = binaryToRule(bytesArray)
	req.Error(err)
	bytesArray = []byte{}
	rule, err = binaryToRule(bytesArray)
	req.Error(err)
}

func TestReadModel(t *testing.T) {
	req := require.New(t)
	reader := bytes.NewReader([]byte{0, 0, 0, 5, 0, 0, 0, 6,
		0, 0, 0, 99, 0, 0, 0, 6,
		0, 0, 0, 98, 0, 0, 0, 7,
		0, 0, 0, 95, 0, 0, 0, 4,
		0, 0, 0, 100, 0, 0, 0, 5,
		0, 0, 0, 97, 0, 0, 0, 8,
		0, 0, 0, 4, 0, 0, 0, 8, 0, 0, 0, 9,
		0, 0, 0, 4, 0, 0, 0, 6, 0, 0, 0, 10,
		0, 0, 0, 4, 0, 0, 0, 5, 0, 0, 0, 11,
		0, 0, 0, 4, 0, 0, 0, 7, 0, 0, 0, 12,
		0, 0, 0, 8, 0, 0, 0, 7, 0, 0, 0, 13,
		0, 0, 0, 8, 0, 0, 0, 8, 0, 0, 0, 14,
		0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 3})
	model, err := ReadModel(reader)
	req.NoError(err)
	req.Equal(BPE, *model)

	reader = bytes.NewReader([]byte{0, 0, 0, 5, 0, 0, 0, 6,
		0, 0, 0, 99, 0, 0, 0, 6,
		0, 0, 0, 98, 0, 0, 0, 7,
		0, 0, 0, 95, 0, 0, 0, 4,
		0, 0, 0, 100, 0, 0, 0, 5,
		0, 0, 0, 97, 0, 0, 0, 8,
		0, 0, 0, 4, 0, 0, 0, 8, 0, 0, 0, 9,
		0, 0, 0, 4, 0, 0, 0, 6, 0, 0, 0, 10,
		0, 0, 0, 4, 0, 0, 0, 5, 0, 0, 0, 11,
		0, 0, 0, 4, 0, 0, 0, 7, 0, 0, 0, 12,
		0, 0, 0, 8, 0, 0, 0, 7, 0, 0, 0, 13,
		0, 0, 0, 8, 0, 0, 0, 8, 0, 0, 0, 14,
		0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 3,
		0, 0, 0, 4, 0, 0, 0, 5, 0, 0, 0, 11,
		0, 0, 0, 4, 0, 0, 0, 7, 0, 0, 0, 12})
	model, err = ReadModel(reader)
	req.NoError(err)
	req.Equal(BPE, *model)

	reader = bytes.NewReader([]byte{0, 0, 0, 5, 0, 0, 0, 6,
		0, 0, 0, 99, 0, 0, 0, 6,
		0, 0, 0, 98, 0, 0, 0, 7,
		0, 0, 0, 95, 0, 0, 0, 4,
		0, 0, 0, 100, 0, 0, 0, 5,
		0, 0, 0, 97, 0, 0, 0, 8,
		0, 0, 0, 4, 0, 0, 0, 8, 0, 0, 0, 9,
		0, 0, 0, 4, 0, 0, 0, 6, 0, 0, 0, 10,
		0, 0, 0, 4, 0, 0, 0, 5, 0, 0, 0, 11,
		0, 0, 0, 4, 0, 0, 0, 7, 0, 0, 0, 12,
		0, 0, 0, 8, 0, 0, 0, 7, 0, 0, 0, 13,
		0, 0, 0, 8, 0, 0, 0, 8, 0, 0, 0, 14,
		0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0})
	model, err = ReadModel(reader)
	req.Error(err)

	reader = bytes.NewReader([]byte{0, 0, 0, 5, 0, 0, 0, 6,
		0, 0, 0, 99, 0, 0, 0, 6,
		0, 0, 0, 98, 0, 0, 0, 7,
		0, 0, 0, 95, 0, 0, 0, 4,
		0, 0, 0, 100, 0, 0, 0, 5,
		0, 0, 0, 97, 0, 0, 0, 8,
		0, 0, 0, 4, 0, 0, 0, 20, 0, 0, 0, 9,
		0, 0, 0, 4, 0, 0, 0, 6, 0, 0, 0, 10,
		0, 0, 0, 4, 0, 0, 0, 5, 0, 0, 0, 11,
		0, 0, 0, 4, 0, 0, 0, 7, 0, 0, 0, 12,
		0, 0, 0, 8, 0, 0, 0, 7, 0, 0, 0, 13,
		0, 0, 0, 8, 0, 0, 0, 8, 0, 0, 0, 14,
		0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 3})
	model, err = ReadModel(reader)
	req.Error(err)

	reader = bytes.NewReader([]byte{0, 0, 0, 5, 0, 0, 0, 8,
		0, 0, 0, 99, 0, 0, 0, 6,
		0, 0, 0, 98, 0, 0, 0, 7,
		0, 0, 0, 95, 0, 0, 0, 4,
		0, 0, 0, 100, 0, 0, 0, 5,
		0, 0, 0, 97, 0, 0, 0, 8,
		0, 0, 0, 4, 0, 0, 0, 8, 0, 0, 0, 9,
		0, 0, 0, 4, 0, 0, 0, 6, 0, 0, 0, 10,
		0, 0, 0, 4, 0, 0, 0, 5, 0, 0, 0, 11,
		0, 0, 0, 4, 0, 0, 0, 7, 0, 0, 0, 12,
		0, 0, 0, 8, 0, 0, 0, 7, 0, 0, 0, 13,
		0, 0, 0, 8, 0, 0, 0, 8, 0, 0, 0, 14,
		0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 3})
	model, err = ReadModel(reader)
	req.Error(err)
}

func TestModel_IDToToken(t *testing.T) {
	req := require.New(t)
	token, err := BPE.IDToToken(11, false)
	req.NoError(err)
	req.Equal("_d", token)

	token, err = BPE.IDToToken(12, true)
	req.NoError(err)
	req.Equal(" b", token)

	token, err = BPE.IDToToken(1, false)
	req.NoError(err)
	req.Equal(unkToken, token)

	token, err = BPE.IDToToken(5, true)
	req.NoError(err)
	req.Equal("d", token)

	token, err = BPE.IDToToken(30, true)
	req.Error(err)
}

func TestModel_DecodeSentence(t *testing.T) {
	req := require.New(t)
	sentence, err := BPE.DecodeSentence(EncodedString{2, 10, 7, 12, 6, 6, 11, 9, 13, 3, 0})
	req.NoError(err)
	req.Equal("<BOS>cb bcc d aab<EOS><PAD>", sentence)

	sentence, err = BPE.DecodeSentence(EncodedString{12, 8, 6, 5, 11, 6, 9, 9, 5, 5, 8, 11, 7})
	req.NoError(err)
	req.Equal("bacd dc a adda db", sentence)

	sentence, err = BPE.DecodeSentence(EncodedString{12, 8, 25, 5, 11, 6, 9, 9, 5, 5, 8, 11, 7})
	req.Error(err)
}

func TestModel_DecodeSentences(t *testing.T) {
	req := require.New(t)
	encodedSentences := []EncodedString{
		{2, 10, 7, 12, 6, 6, 11, 9, 13, 3, 0},
		{12, 8, 6, 5, 11, 6, 9, 9, 5, 5, 8, 11, 7}}
	sentences, err := BPE.DecodeSentences(encodedSentences)
	req.NoError(err)
	req.Equal([]string{"<BOS>cb bcc d aab<EOS><PAD>", "bacd dc a adda db"}, sentences)

	encodedSentences = []EncodedString{
		{2, 10, 7, 12, 6, 6, 11, 9, 8, 7, 3, 0},
		{12, 8, 6, 5, 30, 6, 9, 9, 5, 5, 8, 11, 7}}
	sentences, err = BPE.DecodeSentences(encodedSentences)
	req.Error(err)
}

func TestModel_DecodeFromStream(t *testing.T) {
	req := require.New(t)
	reader := strings.NewReader(`2 10 7 12 6 6 11 9 13 3 0
12 8 6 5 11 6 9 9 5 5 8 11 7`)
	sentences, err := BPE.DecodeFromStream(reader)
	req.NoError(err)
	req.Equal([]string{"<BOS>cb bcc d aab<EOS><PAD>", "bacd dc a adda db"}, sentences)

	reader = strings.NewReader(`2 20 7 12 6 6 11 9 8 7 3 0
12 8 6 5 11 6 9 9 5 5 8 11 7`)
	sentences, err = BPE.DecodeFromStream(reader)
	req.Error(err)
}

func TestModel_EncodeSentence(t *testing.T) {
	req := require.New(t)
	ids, tokens, err := BPE.EncodeSentence("abcda bdhsab acad aaab baaaab",
		EncodingConfig{true, true, false})
	req.NoError(err)
	req.Equal(EncodedString{2, 9, 7, 6, 5, 8, 12, 5, 1, 13, 9, 6, 8, 5, 9, 8, 13, 12, 14, 8, 13,
		3}, ids)
	req.Equal([]string{"<BOS>", "_a", "b", "c", "d", "a", "_b", "d", "hs", "ab", "_a", "c", "a",
		"d", "_a", "a", "ab", "_b", "aa", "a", "ab", "<EOS>"}, tokens)

	ids, tokens, err = BPE.EncodeSentence("gjhcbsd kbs;.jakjcdljk ajbabk,l kjaajlkj kj",
		EncodingConfig{false, false, false})
	req.NoError(err)
	req.Equal(EncodedString{4, 1, 6, 7, 1, 5, 4, 1, 7, 1, 8, 1, 6, 5, 1, 9, 1, 7, 13, 1, 4, 1, 14,
		1, 4, 1}, ids)
	req.Equal([]string{"_", "gjh", "c", "b", "s", "d", "_", "k", "b", "s;.j", "a", "kj", "c", "d",
		"ljk", "_a", "j", "b", "ab", "k,l", "_", "kj", "aa", "jlkj", "_", "kj"}, tokens)

	ids, tokens, err = BPE.EncodeSentence("sd;,fjha nbvmcnk';uegoqw kskg abckjgdhg kjsgf",
		EncodingConfig{true, true, true})
	req.NoError(err)
	derivedIds := make(EncodedString, len(tokens))
	for i, token := range tokens {
		if id, ok := BPE.revRecipe[token]; ok {
			derivedIds[i] = id
		} else {
			derivedIds[i] = TokenID(BPE.specialTokens.unk)
		}
	}
	req.Equal(ids, derivedIds)
}

func TestModel_EncodeSentences(t *testing.T) {
	req := require.New(t)
	ids, tokens, err := BPE.EncodeSentences([]string{"abcda bdhsab acad aaab baaaab",
		"gjhcbsd kbs;.jakjcdljk ajbabk,l kjaajlkj kj"},
		EncodingConfig{true, true, false})
	req.NoError(err)
	req.Equal([]EncodedString{{2, 9, 7, 6, 5, 8, 12, 5, 1, 13, 9, 6, 8, 5, 9, 8, 13, 12, 14, 8, 13,
		3}, {2, 4, 1, 6, 7, 1, 5, 4, 1, 7, 1, 8, 1, 6, 5, 1, 9, 1, 7, 13, 1, 4, 1, 14, 1, 4, 1,
		3}}, ids)
	req.Equal([][]string{{"<BOS>", "_a", "b", "c", "d", "a", "_b", "d", "hs", "ab", "_a", "c", "a",
		"d", "_a", "a", "ab", "_b", "aa", "a", "ab", "<EOS>"}, {"<BOS>", "_", "gjh", "c", "b", "s",
		"d", "_", "k", "b", "s;.j", "a", "kj", "c", "d", "ljk", "_a", "j", "b", "ab", "k,l",
		"_", "kj", "aa", "jlkj", "_", "kj", "<EOS>"}}, tokens)

	ids, tokens, err = BPE.EncodeSentences([]string{"abcda bdhsab acad aaab baaaab",
		"sd;,fjha nbvmcnk';uegoqw kskg abckjgdhg kjsgf"},
		EncodingConfig{true, true, false})
	req.NoError(err)
	derivedIds := make([]EncodedString, len(tokens))
	for j, splitSentence := range tokens {
		derivedIds[j] = make(EncodedString, len(splitSentence))
		for i, token := range splitSentence {
			if id, ok := BPE.revRecipe[token]; ok {
				derivedIds[j][i] = id
			} else {
				derivedIds[j][i] = TokenID(BPE.specialTokens.unk)
			}
		}
	}
	req.Equal(ids, derivedIds)
}

func TestModel_EncodeStream(t *testing.T) {
	req := require.New(t)
	reader := strings.NewReader(`abcda bdhsab acad aaab baaaab
gjhcbsd kbs;.jakjcdljk ajbabk,l kjaajlkj kj`)
	ids, tokens, err := BPE.EncodeStream(reader, EncodingConfig{true, true, false})
	req.NoError(err)
	req.Equal([]EncodedString{{2, 9, 7, 6, 5, 8, 12, 5, 1, 13, 9, 6, 8, 5, 9, 8, 13, 12, 14, 8, 13,
		3}, {2, 4, 1, 6, 7, 1, 5, 4, 1, 7, 1, 8, 1, 6, 5, 1, 9, 1, 7, 13, 1, 4, 1, 14, 1, 4, 1,
		3}}, ids)
	req.Equal([][]string{{"<BOS>", "_a", "b", "c", "d", "a", "_b", "d", "hs", "ab", "_a", "c", "a",
		"d", "_a", "a", "ab", "_b", "aa", "a", "ab", "<EOS>"}, {"<BOS>", "_", "gjh", "c", "b", "s",
		"d", "_", "k", "b", "s;.j", "a", "kj", "c", "d", "ljk", "_a", "j", "b", "ab", "k,l",
		"_", "kj", "aa", "jlkj", "_", "kj", "<EOS>"}}, tokens)

	reader = strings.NewReader(`abcda bdhsab acad aaab baaaab
sd;,fjha nbvmcnk';uegoqw kskg abckjgdhg kjsgf`)
	ids, tokens, err = BPE.EncodeStream(reader, EncodingConfig{false, false, true})
	req.NoError(err)
	derivedIds := make([]EncodedString, len(tokens))
	for j, splitSentence := range tokens {
		derivedIds[j] = make(EncodedString, len(splitSentence))
		for i, token := range splitSentence {
			if id, ok := BPE.revRecipe[token]; ok {
				derivedIds[j][i] = id
			} else {
				derivedIds[j][i] = TokenID(BPE.specialTokens.unk)
			}
		}
	}
	req.Equal(ids, derivedIds)
}
