package content

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

func newBodyTagParser() goldmark.Markdown {
	return goldmark.New(goldmark.WithExtensions(extension.GFM))
}

func extractBodyTags(md goldmark.Markdown, source []byte) []string {
	if len(source) == 0 || !bytes.Contains(source, []byte{'#'}) {
		return nil
	}

	document := md.Parser().Parse(text.NewReader(source))
	var tags []string
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch current := node.(type) {
		case *ast.Heading, *ast.CodeBlock, *ast.FencedCodeBlock, *ast.CodeSpan,
			*ast.Link, *ast.Image, *ast.AutoLink, *ast.RawHTML, *ast.HTMLBlock:
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			tags = append(tags, extractHashtags(current.Value(source))...)
		}
		return ast.WalkContinue, nil
	})
	return tags
}

func extractHashtags(source []byte) []string {
	var tags []string
	for i := 0; i < len(source); {
		r, size := utf8.DecodeRune(source[i:])
		if r != '#' || (i > 0 && !isHashtagBoundary(lastRune(source[:i]))) {
			i += size
			continue
		}

		start := i + size
		end := start
		hasWordRune := false
		hasLetter := false
		for end < len(source) {
			next, nextSize := utf8.DecodeRune(source[end:])
			switch {
			case unicode.IsLetter(next):
				hasWordRune = true
				hasLetter = true
				end += nextSize
			case unicode.IsDigit(next):
				hasWordRune = true
				end += nextSize
			case unicode.IsMark(next) && hasWordRune:
				end += nextSize
			case (next == '-' || next == '_') && hasWordRune:
				end += nextSize
			default:
				goto tagEnd
			}
		}

	tagEnd:
		if hasLetter {
			tag := strings.TrimRight(string(source[start:end]), "-_")
			if tag != "" {
				tags = append(tags, tag)
				i = end
				continue
			}
		}
		i += size
	}
	return tags
}

func isHashtagBoundary(previous rune) bool {
	if unicode.IsLetter(previous) || unicode.IsDigit(previous) || unicode.IsMark(previous) {
		return false
	}
	switch previous {
	case '#', '/', '\\', ':', '?', '&', '=', '%', '_', '-', '@':
		return false
	default:
		return true
	}
}

func lastRune(source []byte) rune {
	r, _ := utf8.DecodeLastRune(source)
	return r
}
