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
