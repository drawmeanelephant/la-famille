package search

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"unicode"
)

type Item struct {
	Title   string   `json:"t"`
	URL     string   `json:"u"`
	Tags    []string `json:"g"`
	Snippet string   `json:"s"`
}

var linkRe = regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)

func ExtractSnippet(rest []byte) string {
	s := string(rest)
	s = linkRe.ReplaceAllString(s, "$1")
	var sb strings.Builder
	sb.Grow(len(s))
	for _, r := range s {
		if r == '#' || r == '*' || r == '[' || r == ']' || r == '`' || r == '>' {
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
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "")
	return enc.Encode(data)
}
