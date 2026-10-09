package transform

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

func newFigureEngine() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithParserOptions(
			parser.WithASTTransformers(
				util.Prioritized(&FigureTransformer{}, 200),
			),
			parser.WithInlineParsers(
				util.Prioritized(&EmojiKitchenParser{}, 100),
			),
		),
		goldmark.WithRendererOptions(
			renderer.WithNodeRenderers(
				util.Prioritized(&FigureRenderer{}, 100),
			),
		),
	)
}

func renderMarkdown(t *testing.T, md goldmark.Markdown, input string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := md.Convert([]byte(input), &buf); err != nil {
		t.Fatalf("convert: %v", err)
	}
	return buf.String()
}

func TestFigureTransformerPromotesStandaloneImages(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantSub     string
		notWantSubs []string
	}{
		{
			name:        "titled image becomes figure with caption",
			input:       "![A lighthouse at dusk](/assets/img/lighthouse.jpg \"The lighthouse at Point Reyes\")",
			wantSub:     `<figure class="lf-figure"><img src="/assets/img/lighthouse.jpg" alt="A lighthouse at dusk" loading="lazy" decoding="async" title="The lighthouse at Point Reyes"><figcaption>The lighthouse at Point Reyes</figcaption></figure>`,
			notWantSubs: []string{"<p><img"},
		},
		{
			name:        "untitled image becomes figure without caption",
			input:       "![A lighthouse](lighthouse.jpg)",
			wantSub:     `<figure class="lf-figure"><img src="lighthouse.jpg" alt="A lighthouse" loading="lazy" decoding="async"></figure>`,
			notWantSubs: []string{"<figcaption>", "<p><img"},
		},
		{
			name:    "alt text wrapped across lines still promotes",
			input:   "![alt\ntext](/assets/img/a.png)",
			wantSub: `<figure class="lf-figure"><img src="/assets/img/a.png" alt="alttext" loading="lazy" decoding="async"></figure>`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := renderMarkdown(t, newFigureEngine(), tc.input)
			if !strings.Contains(got, tc.wantSub) {
				t.Errorf("expected output to contain %q, got:\n%s", tc.wantSub, got)
			}
			for _, sub := range tc.notWantSubs {
				if strings.Contains(got, sub) {
					t.Errorf("expected output to omit %q, got:\n%s", sub, got)
				}
			}
		})
	}
}

func TestFigureTransformerLeavesMixedContentInline(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"image with surrounding text", "Look at this ![cat](cat.jpg) in the garden."},
		{"two images in one paragraph", "![a](a.png) and ![b](b.png)"},
		{"linked image", "[![alt](pic.png)](https://example.com)"},
		{"image with code span", "![alt](pic.png) `code`"},
		{"emoji kitchen glyph alone", "!ek[🐢+🔥]"},
		{"emoji kitchen with text", "Mix !ek[🐢+🔥] together."},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := renderMarkdown(t, newFigureEngine(), tc.input)
			if strings.Contains(got, "<figure") {
				t.Errorf("expected no figure promotion, got:\n%s", got)
			}
		})
	}
}

func TestFigureRendererEscapesHostileAttributes(t *testing.T) {
	input := "![x](/a.png?q=\"onmouseover='x'\" \"Title with <tags> & quotes\")"
	got := renderMarkdown(t, newFigureEngine(), input)
	if strings.Contains(got, "<tags>") || strings.Contains(got, `"onmouseover`) {
		t.Errorf("hostile content not escaped:\n%s", got)
	}
	if !strings.Contains(got, "%22onmouseover") {
		t.Errorf("URL destination not percent-encoded:\n%s", got)
	}
	if !strings.Contains(got, "<figcaption>Title with &lt;tags&gt; &amp; quotes</figcaption>") {
		t.Errorf("caption escaping wrong:\n%s", got)
	}
}

// TestFigureTransformerToleratesUnicodeWhitespaceSiblings covers the panic
// from issue #632: bytes.TrimSpace strips Unicode whitespace, so a
// whitespace-only *ast.Text node (U+00A0, U+2009, U+3000, ...) may precede or
// follow the image and must not crash promotion.
func TestFigureTransformerToleratesUnicodeWhitespaceSiblings(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"NBSP line before image", " \n![alt](x.png)\n"},
		{"NBSP line after image", "![alt](x.png)\n \n"},
		{"NBSP around image on one line", " ![alt](x.png) \n"},
		{"ideographic space before image", "　\n![a](a.png)\n"},
		{"thin space before image", " \n![a](a.png)\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := renderMarkdown(t, newFigureEngine(), tc.input)
			if !strings.Contains(got, `<figure class="lf-figure">`) {
				t.Errorf("expected figure promotion, got:\n%s", got)
			}
			if strings.Contains(got, "<p>") {
				t.Errorf("expected no paragraph wrapper, got:\n%s", got)
			}
		})
	}
}

// TestFigureInsideBlockContainers ensures promotion works within lists and
// blockquotes, not just top-level paragraphs.
func TestFigureInsideBlockContainers(t *testing.T) {
	input := "> ![quoted diagram](diagram.png)"
	got := renderMarkdown(t, newFigureEngine(), input)
	if !strings.Contains(got, "<figure class=\"lf-figure\">") {
		t.Errorf("expected figure inside blockquote, got:\n%s", got)
	}
}

var _ renderer.NodeRenderer = (*FigureRenderer)(nil)
