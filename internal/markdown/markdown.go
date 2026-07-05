package markdown

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension" // Required for GFM and Typographer
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"

	"github.com/tbuddy/la-famille/internal/transform"
)

// NewEngine creates a new configured goldmark.Markdown instance
func NewEngine(transformer *transform.LinkTransformer) goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,              // Adds Tables, Strikethrough, Linkify, and Task Lists
			extension.NewTypographer(), // Adds smart punctuation, curly quotes, and em-dashes
		),
		goldmark.WithParserOptions(
			parser.WithASTTransformers(
				util.Prioritized(transformer, 100),
			),
			parser.WithInlineParsers(
				util.Prioritized(&transform.EmojiKitchenParser{}, 100),
			),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
}
