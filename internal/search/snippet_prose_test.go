package search

import (
	"strings"
	"testing"
)

// TestExtractSnippetKeepsProseAroundAngleBrackets covers the difference between
// markup and prose that merely looks like it. ExtractSnippet reads raw Markdown,
// where a bare '<' is an ordinary character, so a pattern that deleted
// everything between '<' and the next '>' removed the author's words along with
// it — and the deletion was invisible, because the snippet still read as a
// sentence.
func TestExtractSnippetKeepsProseAroundAngleBrackets(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "spaced comparison survives",
			in:   "Swap when x < 5 and y > 3 holds.",
			want: "Swap when x < 5 and y > 3 holds.",
		},
		{
			name: "unspaced comparison survives",
			in:   "If a<b then swap them.",
			want: "If a<b then swap them.",
		},
		{
			name: "generic types survive",
			in:   "Use Vec<String> and List<int> here.",
			want: "Use Vec<String> and List<int> here.",
		},
		{
			name: "real markup is still removed",
			in:   "Use <em>emphasis</em> and <strong>weight</strong>.",
			want: "Use emphasis and weight.",
		},
		{
			name: "unknown element name is kept as prose",
			in:   "Compare <foo bar> against baz.",
			want: "Compare <foo bar> against baz.",
		},
		{
			name: "comments are removed",
			in:   "Before <!-- a note --> after.",
			want: "Before after.",
		},
		{
			name: "self-closing markup is removed",
			in:   "Line one<br/>line two.",
			want: "Line oneline two.",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ExtractSnippet([]byte(c.in)); got != c.want {
				t.Errorf("ExtractSnippet(%q)\n got  %q\n want %q", c.in, got, c.want)
			}
		})
	}
}

// TestExtractSnippetBlockquoteMarkerOnly pins '>' handling: a marker at the
// start of a line is Markdown, the same character mid-sentence is an operator.
func TestExtractSnippetBlockquoteMarkerOnly(t *testing.T) {
	got := ExtractSnippet([]byte("> Quoted line.\n\nAnd 5 > 4 is true."))
	want := "Quoted line. And 5 > 4 is true."
	if got != want {
		t.Errorf("ExtractSnippet\n got  %q\n want %q", got, want)
	}
}

// TestExtractSnippetTagDoesNotSpanLines is the guard against the specific way
// this went wrong: an unterminated '<' running on to a '>' further down the
// document and swallowing every word between them.
func TestExtractSnippetTagDoesNotSpanLines(t *testing.T) {
	got := ExtractSnippet([]byte("Compare a<b for ordering.\n\n> A quoted line.\n\nDone."))
	want := "Compare a<b for ordering. A quoted line. Done."
	if got != want {
		t.Errorf("a match spanned a line break\n got  %q\n want %q", got, want)
	}
}

func TestCleanHeadingTextKeepsProse(t *testing.T) {
	headings := ExtractHeadings([]byte("# Using Vec<String>\n\n## Why 5 > 4\n\n### With <em>markup</em>\n"))
	want := []string{"Using Vec<String>", "Why 5 > 4", "With markup"}

	if len(headings) != len(want) {
		t.Fatalf("ExtractHeadings = %q, want %q", headings, want)
	}
	for i := range want {
		if headings[i] != want[i] {
			t.Errorf("heading %d = %q, want %q", i, headings[i], want[i])
		}
	}
}

// TestExtractSnippetStripsInlineSVG covers the gap an element allowlist is
// prone to: only <svg> and <path> were listed, so a diagram's children survived
// as "prose" and their geometry attributes consumed the whole snippet budget,
// leaving none of the article's text indexed.
func TestExtractSnippetStripsInlineSVG(t *testing.T) {
	in := `# Pipeline

<svg viewBox="0 0 320 80" role="img">
  <rect x="4" y="20" width="90" height="40" fill="#eee" />
  <text x="20" y="45">ingest</text>
  <line x1="94" y1="40" x2="130" y2="40" stroke="#333" />
  <circle cx="150" cy="40" r="16" />
</svg>

Events arrive on a Kafka topic and land in the warehouse.`

	got := ExtractSnippet([]byte(in))
	for _, leaked := range []string{"<rect", "<line", "<circle", "viewBox", "stroke", "x1="} {
		if strings.Contains(got, leaked) {
			t.Errorf("SVG markup %q leaked into the snippet: %q", leaked, got)
		}
	}
	if !strings.Contains(got, "Events arrive on a Kafka topic") {
		t.Errorf("the article prose was pushed out of the snippet: %q", got)
	}
}

// TestExtractSnippetStripsUnlistedMarkupWithAttributes keeps the allowlist from
// having to be exhaustive: a span carrying a quoted attribute is markup even
// when its element name is not one we enumerated.
func TestExtractSnippetStripsUnlistedMarkupWithAttributes(t *testing.T) {
	got := ExtractSnippet([]byte(`Before <my-widget data-id="7" class="x"> after.`))
	if strings.Contains(got, "my-widget") || strings.Contains(got, "data-id") {
		t.Errorf("custom element with attributes should be treated as markup: %q", got)
	}
	if !strings.Contains(got, "Before") || !strings.Contains(got, "after") {
		t.Errorf("surrounding prose was lost: %q", got)
	}
}

// TestExtractSnippetAttributeHeuristicSparesProse is the other half: the
// heuristic must not fire on ordinary text that happens to contain quotes.
func TestExtractSnippetAttributeHeuristicSparesProse(t *testing.T) {
	cases := []string{
		`Compare a<b for ordering.`,
		`Use Vec<String> and List<int> here.`,
		`Swap when x < 5 and y > 3 holds.`,
	}
	for _, in := range cases {
		got := ExtractSnippet([]byte(in))
		if got != in {
			t.Errorf("prose was altered\n in   %q\n got  %q", in, got)
		}
	}
}
