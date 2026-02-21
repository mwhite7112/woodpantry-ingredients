package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "trims leading and trailing whitespace",
			input: "  garlic  ",
			want:  "garlic",
		},
		{
			name:  "lowercases uppercase letters",
			input: "Garlic Powder",
			want:  "garlic powder",
		},
		{
			name:  "trims and lowercases combined",
			input: "  OLIVE OIL  ",
			want:  "olive oil",
		},
		{
			name:  "empty string returns empty",
			input: "",
			want:  "",
		},
		{
			name:  "whitespace-only string returns empty",
			input: "   ",
			want:  "",
		},
		{
			name:  "already normalized string is unchanged",
			input: "garlic",
			want:  "garlic",
		},
		{
			name:  "tabs and newlines are trimmed",
			input: "\t\n garlic \n\t",
			want:  "garlic",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Normalize(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}
