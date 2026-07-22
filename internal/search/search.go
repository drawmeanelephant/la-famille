package search

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"unicode"
)

type Item struct {
	Title    string   `json:"t"`
	URL      string   `json:"u"`
	Tags     []string `json:"g"`
	Snippet  string   `json:"s"`
	Headings []string `json:"h,omitempty"`
}

var (
	linkRe      = regexp.MustCompile(`!?\[([^\]]*)\]\([^\)]+\)`)
	codeBlockRe = regexp.MustCompile("(?s)```[^\\n]*\\n(.*?)```")
	htmlTagRe   = regexp.MustCompile(`<[^>]*>`)
)

// ExtractSnippet cleans up Markdown and HTML content to produce a clean text snippet.
// It explicitly filters out code blocks, tags, and formatting markers to keep search data lightweight and clean.
func ExtractSnippet(rest []byte) string {
	s := string(rest)

	// 1. Strip Markdown code blocks
	s = codeBlockRe.ReplaceAllString(s, "")

	// 2. Strip Markdown links and images, preserving anchor text
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

// ExtractHeadings extracts ATX heading texts (# to ######) from raw Markdown content.
// It excludes code blocks and strips inline Markdown formatting to return clean heading signals.
func ExtractHeadings(rest []byte) []string {
	var headings []string
	seen := make(map[string]bool)

	lines := strings.Split(string(rest), "\n")
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock || !strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Count leading '#'
		i := 0
		for i < len(trimmed) && trimmed[i] == '#' {
			i++
		}
		if i < 1 || i > 6 {
			continue
		}
		// ATX heading requires space or tab or end of line after '#'
		if i < len(trimmed) && trimmed[i] != ' ' && trimmed[i] != '\t' {
			continue
		}

		headingContent := strings.TrimSpace(trimmed[i:])
		// Strip trailing '#' if ATX heading closes with '#'
		headingContent = strings.TrimRight(headingContent, "# \t")

		if headingContent == "" {
			continue
		}

		clean := cleanHeadingText(headingContent)
		if clean != "" && !seen[clean] {
			seen[clean] = true
			headings = append(headings, clean)
		}
	}

	return headings
}

func cleanHeadingText(s string) string {
	s = linkRe.ReplaceAllString(s, "$1")
	s = htmlTagRe.ReplaceAllString(s, "")
	var sb strings.Builder
	sb.Grow(len(s))
	for _, r := range s {
		if r == '#' || r == '*' || r == '[' || r == ']' || r == '`' || r == '>' || r == '_' || r == '~' {
			continue
		}
		if unicode.IsSpace(r) {
			sb.WriteRune(' ')
		} else {
			sb.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(sb.String()), " ")
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
