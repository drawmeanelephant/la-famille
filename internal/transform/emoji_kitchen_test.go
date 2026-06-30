package transform

import (
	"bytes"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/util"
)

func TestEmojiKitchenParser(t *testing.T) {
	md := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithInlineParsers(
				util.Prioritized(&EmojiKitchenParser{}, 100),
			),
		),
	)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Turtle and Fire",
			input:    "Look at this !ek[🐢+🔥]",
			expected: "<p>Look at this <img src=\"https://www.gstatic.com/android/keyboard/emojikitchen/20201001/u1f422/u1f422_u1f525.png\" alt=\"Emoji Kitchen combination of 🐢 and 🔥\" title=\"Emoji Kitchen combination of 🐢 and 🔥\"></p>\n",
		},
		{
			name:     "Turtle and Turtle",
			input:    "!ek[🐢+🐢]",
			expected: "<p><img src=\"https://www.gstatic.com/android/keyboard/emojikitchen/20201001/u1f422/u1f422_u1f422.png\" alt=\"Emoji Kitchen combination of 🐢 and 🐢\" title=\"Emoji Kitchen combination of 🐢 and 🐢\"></p>\n",
		},
		{
			name:     "Invalid syntax",
			input:    "!ek[🐢+]",
			expected: "<p>!ek[🐢+]</p>\n",
		},
		{
			name:     "Spaces around emojis",
			input:    "!ek[ 🐢 + 🔥 ]",
			expected: "<p><img src=\"https://www.gstatic.com/android/keyboard/emojikitchen/20201001/u1f422/u1f422_u1f525.png\" alt=\"Emoji Kitchen combination of 🐢 and 🔥\" title=\"Emoji Kitchen combination of 🐢 and 🔥\"></p>\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := md.Convert([]byte(tc.input), &buf); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if buf.String() != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, buf.String())
			}
		})
	}
}
