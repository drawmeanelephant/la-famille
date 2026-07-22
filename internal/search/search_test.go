package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractSnippet(t *testing.T) {
	md := []byte(`
# Hello World
This is a **bold** and *italic* text.
Here is a [link](https://example.com).
And a code block:
` + "```\nfoo = bar\n```" + `
Inline code: ` + "`fmt.Println()`" + `
> Blockquote text!

Let's see if this works nicely without those characters.
This text needs to be long enough to exceed the one hundred and sixty character limit so that we can verify the truncation logic correctly appends the ellipsis at the very end of the string.
`)
	snippet := ExtractSnippet(md)
	expected := "Hello World This is a bold and italic text. Here is a link. And a code block: Inline code: fmt.Println() Blockquote text! Let's see if this works nicely without..."
	if snippet != expected {
		t.Errorf("expected %q, got %q", expected, snippet)
	}
}

func TestWriteMinifiedJSON(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "search.json")
	items := []Item{
		{Title: "Test", URL: "/test", Tags: []string{"a"}, Snippet: "snip"},
	}
	err := WriteMinifiedJSON(path, items)
	if err != nil {
		t.Fatalf("WriteMinifiedJSON failed: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	str := string(b)
	if str != `[{"t":"Test","u":"/test","g":["a"],"s":"snip"}]`+"\n" {
		t.Errorf("unexpected json output: %q", str)
	}
}

func TestExtractHeadings(t *testing.T) {
	md := []byte("# Main Title\n" +
		"Intro text here.\n\n" +
		"```go\n" +
		"# Not a heading inside code block\n" +
		"```\n\n" +
		"## Section 1: **Features** & [Docs](https://example.com)\n" +
		"Details for section 1.\n\n" +
		"### SubSection 1.1 ###\n" +
		"More details.\n\n" +
		"# Main Title\n\n" +
		"####### Invalid Level 7\n" +
		"##### Level 5 Heading\n")

	got := ExtractHeadings(md)
	expected := []string{
		"Main Title",
		"Section 1: Features & Docs",
		"SubSection 1.1",
		"Level 5 Heading",
	}

	if len(got) != len(expected) {
		t.Fatalf("expected %d headings, got %d: %v", len(expected), len(got), got)
	}
	for i, h := range expected {
		if got[i] != h {
			t.Errorf("heading %d: expected %q, got %q", i, h, got[i])
		}
	}
}

func TestItemJSONSerialization(t *testing.T) {
	itemWithHeadings := Item{
		Title:    "Page Title",
		URL:      "/page.html",
		Tags:     []string{"go", "search"},
		Snippet:  "Snippet text",
		Headings: []string{"H1", "H2"},
	}
	itemWithoutHeadings := Item{
		Title:   "No Headings Page",
		URL:     "/none.html",
		Tags:    nil,
		Snippet: "Snippet text",
	}

	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "search.json")
	if err := WriteMinifiedJSON(path, []Item{itemWithHeadings, itemWithoutHeadings}); err != nil {
		t.Fatalf("failed to write minified json: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read json: %v", err)
	}

	expectedJSON := `[{"t":"Page Title","u":"/page.html","g":["go","search"],"s":"Snippet text","h":["H1","H2"]},{"t":"No Headings Page","u":"/none.html","s":"Snippet text"}]` + "\n"
	if string(b) != expectedJSON {
		t.Errorf("expected JSON:\n%s\ngot:\n%s", expectedJSON, string(b))
	}
}
