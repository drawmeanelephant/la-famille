package search

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"unicode"
)

type Item struct {
	Tags    []string `json:"g"`
	Title   string   `json:"t"`
	URL     string   `json:"u"`
	Snippet string   `json:"s"`
}

var (
	linkRe      = regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)
	codeBlockRe = regexp.MustCompile("(?s)```[a-zA-Z0-9]*\\n(.*?)(\\n)```")
	htmlTagRe   = regexp.MustCompile(`<[^>]*>`)
)

// ExtractSnippet cleans up Markdown and HTML content to produce a clean text snippet.
// It explicitly filters out code blocks, tags, and formatting markers to keep search data lightweight and clean.
func ExtractSnippet(rest []byte) string {
	s := string(rest)

	// 1. Strip Markdown code blocks
	s = codeBlockRe.ReplaceAllString(s, "")

	// 2. Strip Markdown links, preserving only anchor text
	s = linkRe.ReplaceAllString(s, "$1")

	// 3. Strip raw HTML tags to prevent indexing styling classes or scripts
	s = htmlTagRe.ReplaceAllString(s, "")

	var sb strings.Builder
	sb.Grow(len(s))
	for _, r := range s {
		// Strip common Markdown syntactic noise
		if r == '#' || r == '*' || r == '[' || r == ']' || r == '`' || r == '>' || r == '_' || r == '~' {
			continue
		}
		if unicode.IsSpace(r) {
			sb.WriteRune(' ')
		} else {
			sb.WriteRune(r)
		}
	}

	cleaned := strings.Join(strings.Fields(sb.String()), " ")
	runes := []rune(cleaned)
	if len(runes) > 160 {
		return string(runes[:160]) + "..."
	}
	if len(runes) > 0 {
		return string(runes)
	}
	return ""
}

func WriteMinifiedJSON(path string, data interface{}) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "")
	return enc.Encode(data)
}
