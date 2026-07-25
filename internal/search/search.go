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
	Tags     []string `json:"g,omitempty"`
	Snippet  string   `json:"s,omitempty"`
	Headings []string `json:"h,omitempty"`
}

var (
	linkRe      = regexp.MustCompile(`!?\[([^\]]*)\]\([^\)]+\)`)
	codeBlockRe = regexp.MustCompile("(?s)```[^\\n]*\\n(.*?)```")

	// htmlMarkupRe covers comments, declarations and processing instructions —
	// never prose, always safe to drop.
	htmlMarkupRe = regexp.MustCompile(`(?s)<!--.*?-->|<![^<>]*>|<\?[^<>]*>`)

	// htmlTagCandidateRe matches something *shaped* like a tag. Shape alone is
	// not enough to delete it: this input is raw Markdown, so "x<y and y>z" and
	// "Vec<String>" match this too. The element name decides — see htmlElements.
	//
	// The body excludes newlines deliberately. Without that, "a<b then swap."
	// runs on to the next '>' anywhere later in the document — a blockquote
	// marker two lines down will do — and swallows every word in between. The
	// cost is that a tag written across several lines is left in the snippet,
	// which is the safer direction: text survives, markup leaks.
	htmlTagCandidateRe = regexp.MustCompile(`</?[a-zA-Z][a-zA-Z0-9-]*[^<>\n]*>`)

	tagNameRe = regexp.MustCompile(`^</?([a-zA-Z][a-zA-Z0-9-]*)`)

	// leadingQuoteRe strips the blockquote marker at the start of a line. The
	// same character mid-sentence is a greater-than sign and belongs to the
	// prose.
	leadingQuoteRe = regexp.MustCompile(`(?m)^[ \t]*>+[ \t]?`)
)

// htmlElements is the set of names treated as real markup. Anything else
// keeps its angle brackets and stays in the snippet, so a comparison or a
// generic type is indexed as written rather than silently deleted along with
// the words between the brackets.
var htmlElements = map[string]bool{
	"a": true, "abbr": true, "address": true, "area": true, "article": true,
	"aside": true, "audio": true, "b": true, "base": true, "bdi": true,
	"bdo": true, "blockquote": true, "body": true, "br": true, "button": true,
	"canvas": true, "caption": true, "cite": true, "code": true, "col": true,
	"colgroup": true, "data": true, "datalist": true, "dd": true, "del": true,
	"details": true, "dfn": true, "dialog": true, "div": true, "dl": true,
	"dt": true, "em": true, "embed": true, "fieldset": true, "figcaption": true,
	"figure": true, "footer": true, "form": true, "h1": true, "h2": true,
	"h3": true, "h4": true, "h5": true, "h6": true, "head": true, "header": true,
	"hgroup": true, "hr": true, "html": true, "i": true, "iframe": true,
	"img": true, "input": true, "ins": true, "kbd": true, "label": true,
	"legend": true, "li": true, "link": true, "main": true, "map": true,
	"mark": true, "menu": true, "meta": true, "meter": true, "nav": true,
	"noscript": true, "object": true, "ol": true, "optgroup": true,
	"option": true, "output": true, "p": true, "param": true, "picture": true,
	"pre": true, "progress": true, "q": true, "rp": true, "rt": true,
	"ruby": true, "s": true, "samp": true, "script": true, "section": true,
	"select": true, "slot": true, "small": true, "source": true, "span": true,
	"strong": true, "style": true, "sub": true, "summary": true, "sup": true,
	"svg": true, "path": true, "table": true, "tbody": true, "td": true,
	"template": true, "textarea": true, "tfoot": true, "th": true,
	"thead": true, "time": true, "title": true, "tr": true, "track": true,
	"u": true, "ul": true, "var": true, "video": true, "wbr": true,
}

// stripHTMLTags removes markup while leaving prose that merely looks like it.
func stripHTMLTags(s string) string {
	s = htmlMarkupRe.ReplaceAllString(s, "")
	return htmlTagCandidateRe.ReplaceAllStringFunc(s, func(match string) string {
		name := tagNameRe.FindStringSubmatch(match)
		if name == nil {
			return match
		}
		if htmlElements[strings.ToLower(name[1])] {
			return ""
		}
		return match
	})
}

// ExtractSnippet cleans up Markdown and HTML content to produce a clean text snippet.
// It explicitly filters out code blocks, tags, and formatting markers to keep search data lightweight and clean.
func ExtractSnippet(rest []byte) string {
	s := string(rest)

	// 1. Strip Markdown code blocks
	s = codeBlockRe.ReplaceAllString(s, "")

	// 2. Strip Markdown links and images, preserving anchor text
	s = linkRe.ReplaceAllString(s, "$1")

	// 3. Strip raw HTML tags to prevent indexing styling classes or scripts
	s = stripHTMLTags(s)

	// 4. Strip blockquote markers, which are only markers at the start of a
	//    line. Removing every '>' would take the operator out of "y > 3".
	s = leadingQuoteRe.ReplaceAllString(s, "")

	var sb strings.Builder
	sb.Grow(len(s))
	for _, r := range s {
		// Strip common Markdown syntactic noise
		if r == '#' || r == '*' || r == '[' || r == ']' || r == '`' || r == '_' || r == '~' {
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
	s = stripHTMLTags(s)
	var sb strings.Builder
	sb.Grow(len(s))
	for _, r := range s {
		// '>' is kept: a heading has no blockquote marker, so it is a
		// greater-than sign and part of the text.
		if r == '#' || r == '*' || r == '[' || r == ']' || r == '`' || r == '_' || r == '~' {
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
