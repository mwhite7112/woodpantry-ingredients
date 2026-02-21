package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeAliases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		winnerAliases []string
		loserName     string
		loserAliases  []string
		winnerName    string
		want          []string
	}{
		{
			name:          "basic merge with dedup",
			winnerAliases: []string{"clove"},
			loserName:     "garlic clove",
			loserAliases:  []string{"clove"},
			winnerName:    "garlic",
			want:          []string{"clove", "garlic clove"},
		},
		{
			name:          "empty inputs produce only loser name",
			winnerAliases: []string{},
			loserName:     "butter",
			loserAliases:  []string{},
			winnerName:    "unsalted butter",
			want:          []string{"butter"},
		},
		{
			name:          "all empty",
			winnerAliases: nil,
			loserName:     "onion",
			loserAliases:  nil,
			winnerName:    "yellow onion",
			want:          []string{"onion"},
		},
		{
			name:          "winner name excluded from result",
			winnerAliases: []string{},
			loserName:     "garlic",
			loserAliases:  []string{"garlic"},
			winnerName:    "garlic",
			want:          []string{},
		},
		{
			name:          "loser alias same as winner name is excluded",
			winnerAliases: []string{"minced garlic"},
			loserName:     "garlic clove",
			loserAliases:  []string{"garlic"},
			winnerName:    "garlic",
			want:          []string{"minced garlic", "garlic clove"},
		},
		{
			name:          "duplicate across winner and loser aliases",
			winnerAliases: []string{"a", "b"},
			loserName:     "c",
			loserAliases:  []string{"a", "d"},
			winnerName:    "winner",
			want:          []string{"a", "b", "c", "d"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := mergeAliases(tc.winnerAliases, tc.loserName, tc.loserAliases, tc.winnerName)
			assert.Equal(t, tc.want, got)
		})
	}
}
