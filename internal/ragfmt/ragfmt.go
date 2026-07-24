// Package ragfmt owns the one escaping rule shared by the RAG archive writer
// (internal/ragexport) and its reader (internal/retrieval).
//
// The archive wraps each file verbatim:
//
//	<file path="internal/foo.go">
//	<content>
//	...bytes...
//	</content>
//	</file>
//
// The reader is a line-prefix state machine, so a *content* line that begins
// with one of those markers is indistinguishable from real structure. That is
// not hypothetical: the archive bundles the project's own Go sources and every
// file under content/, so any source file or Markdown page that documents the
// format — including this package's tests — silently corrupts the bundle and
// makes it unparseable.
//
// Escaping is therefore applied at write time and undone at read time. A
// content line whose first characters are a marker (optionally already preceded
// by backslashes) gains one leading backslash; the reader removes exactly one.
// The transform is reversible for arbitrary input, and lines that merely start
// with a backslash are left alone, so ordinary prose and code are untouched.
package ragfmt

import "strings"

// markers are the line prefixes the archive reader treats as structure.
var markers = []string{"<file ", "<content>", "</content>", "</file>"}

// hasMarker reports whether s begins with an archive structure marker.
func hasMarker(s string) bool {
	for _, m := range markers {
		if strings.HasPrefix(s, m) {
			return true
		}
	}
	return false
}

// isEscapableMarker reports whether the line is a structure marker, or an
// already-escaped one (any run of leading backslashes followed by a marker).
// Those are exactly the lines whose meaning depends on escaping, which keeps
// the transform reversible: `</file>` and `\</file>` both need a backslash
// added, while `\hello` needs nothing.
func isEscapableMarker(line string) bool {
	return hasMarker(strings.TrimLeft(line, `\`))
}

// EscapeLine escapes a single content line. Safe to call on any input.
func EscapeLine(line string) string {
	if isEscapableMarker(line) {
		return `\` + line
	}
	return line
}

// UnescapeLine reverses EscapeLine for a single line.
func UnescapeLine(line string) string {
	if len(line) == 0 || line[0] != '\\' || !isEscapableMarker(line) {
		return line
	}
	return line[1:]
}

// EscapeContent escapes every line of a file body destined for the archive.
// Line endings are preserved: only a leading marker is affected.
func EscapeContent(content string) string {
	if content == "" {
		return content
	}
	// Fast path: most files contain no markers at all.
	if !strings.Contains(content, "<file ") &&
		!strings.Contains(content, "<content>") &&
		!strings.Contains(content, "</content>") &&
		!strings.Contains(content, "</file>") {
		return content
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		// Tolerate CRLF: escape based on the line without its trailing CR,
		// then put the CR back.
		if cr := strings.HasSuffix(line, "\r"); cr {
			lines[i] = EscapeLine(strings.TrimSuffix(line, "\r")) + "\r"
			continue
		}
		lines[i] = EscapeLine(line)
	}
	return strings.Join(lines, "\n")
}
