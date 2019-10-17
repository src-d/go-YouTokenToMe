package bpe

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewModel(t *testing.T) {
	model := newModel(10)
	require.Equal(t, 10, len(model.rules))
}

func TestDecodedTokenToString(t *testing.T) {
	id2char := map[TokenID]rune{1: []rune("a")[0], 2: []rune("b")[0], 3: []rune("c")[0]}
	word, err := DecodeToken(EncodedToken{1, 2, 1, 3, 3}, id2char)
	require.NoError(t, err)
	require.Equal(t, "abacc", word)
}

func TestReadModel(t *testing.T) {
	reader := strings.NewReader(`5 4
99 6
98 7
95 4
100 5
97 8
4 8 9
4 6 10
4 5 11
4 7 12
1 0 2 4`)
	expected := Model{
		map[rune]TokenID{97: 8, 98: 7, 99: 6, 100: 5, 95: 4},
		map[TokenID]rune{4: 95, 5: 100, 6: 99, 7: 98, 8: 97},
		[]rule{{4, 8, 9}, {4, 6, 10}, {4, 5, 11}, {4, 7, 12}},
		map[TokenID]EncodedToken{4: {4}, 5: {5}, 6: {6}, 7: {7}, 8: {8}, 9: {4, 8}, 10: {4, 6}, 11: {4, 5}, 12: {4, 7}},
		map[string]TokenID{"a": 8, "b": 7, "c": 6, "d": 5, "_": 4,
			"_a": 9, "_b": 12, "_c": 10, "_d": 11},
		specialTokens{1, 0, 2, 4},
	}
	model, _ := ReadModelFromText(reader)
	require.Equal(t, expected, *model)
}
