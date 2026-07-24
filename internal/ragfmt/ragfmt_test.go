package ragfmt

import (
	"strings"
	"testing"
)

func TestEscapeLine(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"ordinary prose", "hello world", "hello world"},
		{"empty", "", ""},
		{"leading backslash but no marker", `\hello`, `\hello`},
		{"indented marker is not structure", "  </file>", "  </file>"},
		{"open file", `<file path="a.go">`, `\<file path="a.go">`},
		{"open content", "<content>", `\<content>`},
		{"close content", "</content>", `\</content>`},
		{"close file", "</file>", `\</file>`},
		{"already escaped gains one", `\</file>`, `\\</file>`},
		{"double escaped gains one", `\\</file>`, `\\\</file>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := EscapeLine(tc.in); got != tc.want {
				t.Fatalf("EscapeLine(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestEscapeUnescapeRoundTrip(t *testing.T) {
	// Every one of these must survive a write/read cycle byte-for-byte,
	// including lines that already look escaped.
	lines := []string{
		"",
		"package main",
		`\hello`,
		`\\hello`,
		"<file path=\"content/pub.md\">",
		"<content>",
		"</content>",
		"</file>",
		`\</file>`,
		`\\</content>`,
		"  </file>",
		"text </file> mid-line",
	}
	for _, line := range lines {
		if got := UnescapeLine(EscapeLine(line)); got != line {
			t.Fatalf("round trip failed for %q: got %q", line, got)
		}
	}
}

// TestEscapeContentDefusesEmbeddedBundle is the regression that matters: a Go
// source file containing a raw-string archive fixture must not be able to
// inject structure into the bundle that embeds it.
func TestEscapeContentDefusesEmbeddedBundle(t *testing.T) {
	source := strings.Join([]string{
		"func TestSomething(t *testing.T) {",
		"\tbundle := `<file path=\"content/pub.md\">",
		"<content>",
		"# Pub",
		"</content>",
		"</file>",
		"`",
		"}",
	}, "\n")

	escaped := EscapeContent(source)

	for i, line := range strings.Split(escaped, "\n") {
		if hasMarker(line) {
			t.Fatalf("line %d still reads as structure: %q", i+1, line)
		}
	}

	// And it must decode back to the original source exactly.
	var restored []string
	for _, line := range strings.Split(escaped, "\n") {
		restored = append(restored, UnescapeLine(line))
	}
	if got := strings.Join(restored, "\n"); got != source {
		t.Fatalf("content round trip failed:\n got: %q\nwant: %q", got, source)
	}
}

func TestEscapeContentPreservesCRLFAndCleanFiles(t *testing.T) {
	clean := "package main\n\nfunc main() {}\n"
	if got := EscapeContent(clean); got != clean {
		t.Fatalf("clean content was modified: %q", got)
	}

	crlf := "ok\r\n</file>\r\ndone\r\n"
	got := EscapeContent(crlf)
	if !strings.Contains(got, "\\</file>\r\n") {
		t.Fatalf("CRLF marker not escaped: %q", got)
	}
	var restored []string
	for _, line := range strings.Split(got, "\n") {
		if strings.HasSuffix(line, "\r") {
			restored = append(restored, UnescapeLine(strings.TrimSuffix(line, "\r"))+"\r")
			continue
		}
		restored = append(restored, UnescapeLine(line))
	}
	if out := strings.Join(restored, "\n"); out != crlf {
		t.Fatalf("CRLF round trip failed: %q", out)
	}
}
