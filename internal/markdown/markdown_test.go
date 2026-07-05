package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/transform"
)

func TestNewEngine(t *testing.T) {
	// Provide a dummy transformer
	transformer := &transform.LinkTransformer{}
	engine := NewEngine(transformer)

	if engine == nil {
		t.Fatal("expected engine to not be nil")
	}

	// Test a simple conversion to ensure it is configured properly
	source := []byte("# Hello World\n\nThis is a test.")
	var buf bytes.Buffer
	if err := engine.Convert(source, &buf); err != nil {
		t.Fatalf("failed to convert markdown: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "<h1>Hello World</h1>") {
		t.Errorf("expected output to contain <h1>Hello World</h1>, got: %s", result)
	}
	if !strings.Contains(result, "<p>This is a test.</p>") {
		t.Errorf("expected output to contain <p>This is a test.</p>, got: %s", result)
	}
}

func TestNewEngine_GFMAndTypographer(t *testing.T) {
	transformer := &transform.LinkTransformer{}
	engine := NewEngine(transformer)

	tests := []struct {
		name     string
		markdown string
		expected []string
	}{
		{
			name: "GFM Table Rendering",
			markdown: `| Header 1 | Header 2 |
| --- | --- |
| Row 1 Col 1 | Row 1 Col 2 |`,
			expected: []string{
				"<table>",
				"<thead>",
				"<th>Header 1</th>",
				"<tbody>",
				"<td>Row 1 Col 1</td>",
			},
		},
		{
			name:     "GFM Strikethrough",
			markdown: "This is ~~bad~~ formatting.",
			expected: []string{
				"<del>bad</del>",
			},
		},
		{
			name: "GFM Task Lists",
			markdown: `- [ ] Incomplete task
- [x] Completed task`,
			expected: []string{
				`<li><input disabled="" type="checkbox"> Incomplete task</li>`,
				`<li><input checked="" disabled="" type="checkbox"> Completed task</li>`,
			},
		},
		{
			name:     "Typographer Smart Punctuation",
			markdown: `"Hello" -- writing text...`,
			expected: []string{
				"&ldquo;Hello&rdquo;", // Curly double quotes
				"&ndash;",              // En-dash (default behavior for --)
				"&hellip;",              // Ellipsis
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := engine.Convert([]byte(tc.markdown), &buf); err != nil {
				t.Fatalf("failed to convert markdown for %s: %v", tc.name, err)
			}

			result := buf.String()
			for _, exp := range tc.expected {
				if !strings.Contains(result, exp) {
					t.Errorf("Expected output to contain %q, but it did not.\nGot:\n%s", exp, result)
				}
			}
		})
	}
}
