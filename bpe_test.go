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
	[]rule{{4, 8, 9}, {4, 6, 10}, {4, 5, 11}, {4, 7, 12}},
	map[TokenID]EncodedString{4: {4}, 5: {5}, 6: {6}, 7: {7}, 8: {8}, 9: {4, 8},
		10: {4, 6}, 11: {4, 5}, 12: {4, 7}},
	map[string]TokenID{"a": 8, "b": 7, "c": 6, "d": 5, "_": 4,
		"_a": 9, "_b": 12, "_c": 10, "_d": 11},
	specialTokens{1, 0, 2, 3},
	4,
}

func TestNewModel(t *testing.T) {
	model := newModel(10)
	require.Equal(t, 10, len(model.rules))
}

func TestDecodeToken(t *testing.T) {
	id2char := map[TokenID]rune{1: []rune("a")[0], 2: []rune("b")[0], 3: []rune("c")[0]}
	word, err := DecodeToken(EncodedString{1, 2, 1, 3, 3}, id2char)
	require.NoError(t, err)
	require.Equal(t, "abacc", word)

	word, err = DecodeToken(EncodedString{1, 2, 4, 3, 3}, id2char)
	require.Error(t, err)
}

func TestSpecialTokens_ToBinary(t *testing.T) {
	specials := specialTokens{1, 259, 2*256*256 + 37*256 + 2, -256 * 256 * 256 * 127}
	bytesArray := []byte{0, 0, 0, 1, 0, 0, 1, 3, 0, 2, 37, 2, 129, 0, 0, 0}
	require.Equal(t, bytesArray, specials.toBinary())
}

func TestBinaryToSpecialTokens(t *testing.T) {
	bytesArray := []byte{0, 0, 0, 1, 0, 0, 1, 3, 0, 2, 37, 2, 129, 0, 0, 0}
	expected := specialTokens{1, 259, 2*256*256 + 37*256 + 2, -256 * 256 * 256 * 127}
	specials, err := binaryToSpecialTokens(bytesArray)
	require.NoError(t, err)
	require.Equal(t, expected, specials)
	bytesArray = []byte{0, 0, 0, 1, 0, 0, 1, 3, 0, 2, 37, 2, 129, 0, 0}
	specials, err = binaryToSpecialTokens(bytesArray)
	require.Error(t, err)
	bytesArray = []byte{}
	specials, err = binaryToSpecialTokens(bytesArray)
	require.Error(t, err)
}

func TestRule_ToBinary(t *testing.T) {
	rule := rule{1, 2, 257}
	bytesArray := []byte{0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 1, 1}
	require.Equal(t, bytesArray, rule.toBinary())
}

func TestBinaryToRule(t *testing.T) {
	expected := rule{1, 2, 257}
	bytesArray := []byte{0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 1, 1}
	rule, err := binaryToRule(bytesArray)
	require.NoError(t, err)
	require.Equal(t, expected, rule)
	bytesArray = []byte{0, 0, 0, 0, 0, 0, 2, 0, 0, 1, 1}
	rule, err = binaryToRule(bytesArray)
	require.Error(t, err)
	bytesArray = []byte{}
	rule, err = binaryToRule(bytesArray)
	require.Error(t, err)
}

func TestReadModel(t *testing.T) {
	reader := bytes.NewReader([]byte{0, 0, 0, 5, 0, 0, 0, 4,
		0, 0, 0, 99, 0, 0, 0, 6,
		0, 0, 0, 98, 0, 0, 0, 7,
		0, 0, 0, 95, 0, 0, 0, 4,
		0, 0, 0, 100, 0, 0, 0, 5,
		0, 0, 0, 97, 0, 0, 0, 8,
		0, 0, 0, 4, 0, 0, 0, 8, 0, 0, 0, 9,
		0, 0, 0, 4, 0, 0, 0, 6, 0, 0, 0, 10,
		0, 0, 0, 4, 0, 0, 0, 5, 0, 0, 0, 11,
		0, 0, 0, 4, 0, 0, 0, 7, 0, 0, 0, 12,
		0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 3})
	model, err := ReadModel(reader)
	require.NoError(t, err)
	require.Equal(t, BPE, *model)

	reader = bytes.NewReader([]byte{0, 0, 0, 5, 0, 0, 0, 4,
		0, 0, 0, 99, 0, 0, 0, 6,
		0, 0, 0, 98, 0, 0, 0, 7,
		0, 0, 0, 95, 0, 0, 0, 4,
		0, 0, 0, 100, 0, 0, 0, 5,
		0, 0, 0, 97, 0, 0, 0, 8,
		0, 0, 0, 4, 0, 0, 0, 8, 0, 0, 0, 9,
		0, 0, 0, 4, 0, 0, 0, 6, 0, 0, 0, 10,
		0, 0, 0, 4, 0, 0, 0, 5, 0, 0, 0, 11,
		0, 0, 0, 4, 0, 0, 0, 7, 0, 0, 0, 12,
		0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 3,
		0, 0, 0, 4, 0, 0, 0, 5, 0, 0, 0, 11,
		0, 0, 0, 4, 0, 0, 0, 7, 0, 0, 0, 12})
	model, err = ReadModel(reader)
	require.NoError(t, err)
	require.Equal(t, BPE, *model)

	reader = bytes.NewReader([]byte{0, 0, 0, 5, 0, 0, 0, 4,
		0, 0, 0, 99, 0, 0, 0, 6,
		0, 0, 0, 98, 0, 0, 0, 7,
		0, 0, 0, 95, 0, 0, 0, 4,
		0, 0, 0, 100, 0, 0, 0, 5,
		0, 0, 0, 97, 0, 0, 0, 8,
		0, 0, 0, 4, 0, 0, 0, 8, 0, 0, 0, 9,
		0, 0, 0, 4, 0, 0, 0, 6, 0, 0, 0, 10,
		0, 0, 0, 4, 0, 0, 0, 5, 0, 0, 0, 11,
		0, 0, 0, 4, 0, 0, 0, 7, 0, 0, 0, 12,
		0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0})
	model, err = ReadModel(reader)
	require.Error(t, err)

	reader = bytes.NewReader([]byte{0, 0, 0, 5, 0, 0, 0, 4,
		0, 0, 0, 99, 0, 0, 0, 6,
		0, 0, 0, 98, 0, 0, 0, 7,
		0, 0, 0, 95, 0, 0, 0, 4,
		0, 0, 0, 100, 0, 0, 0, 5,
		0, 0, 0, 97, 0, 0, 0, 8,
		0, 0, 0, 4, 0, 0, 0, 20, 0, 0, 0, 9,
		0, 0, 0, 4, 0, 0, 0, 6, 0, 0, 0, 10,
		0, 0, 0, 4, 0, 0, 0, 5, 0, 0, 0, 11,
		0, 0, 0, 4, 0, 0, 0, 7, 0, 0, 0, 12,
		0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 3})
	model, err = ReadModel(reader)
	require.Error(t, err)
}

func TestModel_IdToToken(t *testing.T) {
	token, err := BPE.IdToToken(11, false)
	require.NoError(t, err)
	require.Equal(t, "_d", token)

	token, err = BPE.IdToToken(12, true)
	require.NoError(t, err)
	require.Equal(t, " b", token)

	token, err = BPE.IdToToken(1, false)
	require.NoError(t, err)
	require.Equal(t, UnkToken, token)

	token, err = BPE.IdToToken(5, true)
	require.NoError(t, err)
	require.Equal(t, "d", token)

	token, err = BPE.IdToToken(13, true)
	require.Error(t, err)
}

func TestModel_DecodeSentence(t *testing.T) {
	sentence, err := BPE.DecodeSentence(EncodedString{2, 10, 7, 12, 6, 6, 11, 9, 8, 7, 3, 0})
	require.NoError(t, err)
	require.Equal(t, "<BOS>cb bcc d aab<EOS><PAD>", sentence)

	sentence, err = BPE.DecodeSentence(EncodedString{12, 8, 6, 5, 11, 6, 9, 9, 5, 5, 8, 11, 7})
	require.NoError(t, err)
	require.Equal(t, "bacd dc a adda db", sentence)

	sentence, err = BPE.DecodeSentence(EncodedString{12, 8, 13, 5, 11, 6, 9, 9, 5, 5, 8, 11, 7})
	require.Error(t, err)
}

func TestModel_DecodeSentences(t *testing.T) {
	encodedSentences := []EncodedString{
		{2, 10, 7, 12, 6, 6, 11, 9, 8, 7, 3, 0},
		{12, 8, 6, 5, 11, 6, 9, 9, 5, 5, 8, 11, 7}}
	sentences, err := BPE.DecodeSentences(encodedSentences)
	require.NoError(t, err)
	require.Equal(t, []string{"<BOS>cb bcc d aab<EOS><PAD>", "bacd dc a adda db"}, sentences)

	encodedSentences = []EncodedString{
		{2, 10, 7, 12, 6, 6, 11, 9, 8, 7, 3, 0},
		{12, 8, 6, 5, 13, 6, 9, 9, 5, 5, 8, 11, 7}}
	sentences, err = BPE.DecodeSentences(encodedSentences)
	require.Error(t, err)
}

func TestModel_DecodeFromStream(t *testing.T) {
	reader := strings.NewReader(`2 10 7 12 6 6 11 9 8 7 3 0
12 8 6 5 11 6 9 9 5 5 8 11 7`)
	sentences, err := BPE.DecodeFromStream(reader)
	require.NoError(t, err)
	require.Equal(t, []string{"<BOS>cb bcc d aab<EOS><PAD>", "bacd dc a adda db"}, sentences)

	reader = strings.NewReader(`2 20 7 12 6 6 11 9 8 7 3 0
12 8 6 5 11 6 9 9 5 5 8 11 7`)
	sentences, err = BPE.DecodeFromStream(reader)
	require.Error(t, err)
}
