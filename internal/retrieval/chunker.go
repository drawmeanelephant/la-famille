package retrieval

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// frontmatterRe matches a YAML frontmatter block delimited with ---.
// We use it to strip frontmatter before chunking so titles extracted below
// are not polluted by metadata.
var frontmatterRe = regexp.MustCompile("(?s)^---\n.*?\n---\n?")

// pageHeadingRE: ATX markdown headings (## Foo, ### Bar) at levels 2-3.
// We deliberately skip level-1 headings because every page starts with one;
// chunking at level-1 would produce just one giant chunk per file.
var pageHeadingRE = regexp.MustCompile(`(?m)^(#{2,4})\s+(.+?)\s*#*\s*$`)

// fmKeyValueRE was removed when parseFrontmatter switched to a line-based
// parser; the variable is intentionally absent here so future contributors
// reach for parseFrontmatter directly rather than re-introducing a regex.

// chunkFile splits a single source file into stable chunks bounded by
// heading-2/heading-3 markdown headings. Each chunk preserves its
// surrounding heading trail (so the UI can render "Page > Heading > Sub").
// Stable IDs come from the source path and the chunk position so reruns
// produce identical identifiers.
//
// `render: false` in frontmatter is documented as "exclude from the
// published site", which means "exclude from the assistant corpus"
// — same intent. We treat it as out-of-scope and return nil.
func chunkFile(text, sourcePath string) []Chunk {
	pageID := derivePageID(sourcePath)
	renderedURL := pageIDToURL(pageID)
	kind := sourceKind(sourcePath)
	title, body, render := stripFrontmatter(text)

	if !render {
		// Documented behaviour: content explicitly marked as not for
		// publication does not enter the assistant's corpus.
		return nil
	}

	// If the file is short and has no headings, produce a single chunk.
	if !pageHeadingRE.MatchString(body) {
		id := chunkID(sourcePath, 0, "")
		return []Chunk{{
			ID:          id,
			PageID:      pageID,
			Title:       title,
			HeadingText: "",
			URL:         renderedURL,
			SourcePath:  sourcePath,
			SourceKind:  kind,
			Text:        strings.TrimSpace(body),
			Position:    0,
			TokenCount:  len(body) / 4,
		}}
	}

	matches := pageHeadingRE.FindAllStringSubmatchIndex(body, -1)

	var out []Chunk
	position := 0

	// chunkBeforeFirstHeading — content that lives before the first heading.
	// We keep it as a single chunk titled "Introduction" so the chunker does
	// not lose opening prose (frontmatter, lede paragraphs).
	prelude := body[:matches[0][0]]
	if strings.TrimSpace(prelude) != "" {
		id := chunkID(sourcePath, position, "")
		out = append(out, Chunk{
			ID:         id,
			PageID:     pageID,
			Title:      title,
			URL:        renderedURL,
			SourcePath: sourcePath,
			SourceKind: kind,
			Text:       strings.TrimSpace(prelude),
			Position:   position,
			TokenCount: len(prelude) / 4,
		})
		position++
	}

	for i, m := range matches {
		heading := strings.TrimSpace(body[m[4]:m[5]])
		start := m[0]
		end := len(body)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		section := body[start:end]
		// Drop the heading line itself from the chunk text; we keep the
		// heading in HeadingText so it can be displayed separately. m[1] is
		// the byte offset of the end of the heading line, which is also the
		// start of the body prose for this section.
		sectionText := strings.TrimSpace(body[m[1]:end])
		headingTrail := []string{heading}
		if i > 0 {
			headingTrail = append(headingTrail, headingFromMatch(body, matches[i-1]))
		}

		id := chunkID(sourcePath, position, heading)
		out = append(out, Chunk{
			ID:          id,
			PageID:      pageID,
			Title:       title,
			HeadingPath: headingTrail,
			HeadingText: heading,
			URL:         renderedURL,
			SourcePath:  sourcePath,
			SourceKind:  kind,
			Text:        sectionText,
			Position:    position,
			TokenCount:  len(section) / 4,
		})
		position++
	}
	return out
}

func headingFromMatch(body string, m []int) string {
	return strings.TrimSpace(body[m[4]:m[5]])
}

// stripFrontmatter removes the leading YAML frontmatter block (if any) and
// returns the first title it found, the remaining markdown body, and a
// render flag (defaults to true). The render flag is read from
// `render: false` in the YAML block so authors can opt pages out of the
// assistant corpus; the field is part of the project-wide convention.
func stripFrontmatter(text string) (title, body string, render bool) {
	render = true
	body = text
	if loc := frontmatterRe.FindStringIndex(text); loc != nil {
		fm := text[loc[0]+3 : loc[1]]
		body = strings.TrimSpace(text[loc[1]:])
		title, render = parseFrontmatter(fm)
	}
	if title == "" {
		// Fall back to the first level-1 heading if present.
		if h1 := regexp.MustCompile(`(?m)^#\s+(.+?)\s*$`).FindStringSubmatch(body); h1 != nil {
			title = strings.TrimSpace(h1[1])
		}
	}
	return title, body, render
}

// parseFrontmatter returns the page title and an explicit render flag.
// It is intentionally tiny: it splits the frontmatter on newlines, trims
// surrounding whitespace from each value, and strips any matching leading
// or trailing double or single quote. The hand-rolled approach avoids the
// subtle issues with matching `render: "false"` using a single multiline
// regex — quoted YAML booleans are common in editor output.
func parseFrontmatter(fm string) (title string, render bool) {
	render = true
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.IndexByte(line, ':')
		if idx < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:idx]))
		value := strings.TrimSpace(line[idx+1:])
		value = strings.Trim(value, `"'`)
		switch key {
		case "title":
			title = value
		case "render":
			switch strings.ToLower(value) {
			case "false", "no", "0", "off":
				render = false
			}
		}
	}
	return title, render
}

func derivePageID(sourcePath string) string {
	base := filepath.ToSlash(sourcePath)
	base = strings.TrimPrefix(base, "./")
	// drop leading "content/"
	base = strings.TrimPrefix(base, "content/")
	// drop extension
	if ext := filepath.Ext(base); ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	if base == "" {
		return "index"
	}
	return base
}

// pageIDToURL maps a PageID like "docs/rag" to "/docs/rag/". Empty IDs map
// to "/". Callers can replace the prefix with the configured site URL later.
func pageIDToURL(id string) string {
	if id == "" {
		return "/"
	}
	return "/" + strings.TrimLeft(id, "/") + "/"
}

// sourceKind categorises a RAG bundle path so the corpus can later reject
// non-public artifacts (e.g. we don't expose auth secrets from
// rag-system.md even if they happen to contain citations).
func sourceKind(path string) string {
	base := filepath.Base(filepath.ToSlash(path))
	switch {
	case strings.HasPrefix(base, "rag-content"):
		return "rag-content"
	case strings.HasPrefix(base, "rag-system"):
		return "rag-system"
	case strings.HasPrefix(base, "rag-config"):
		return "rag-config"
	default:
		return "content"
	}
}

// chunkID produces a deterministic chunk identifier. The same input always
// yields the same ID so reruns (including across Ask sessions) compare
// cleanly. The slug is computed from the heading text only if non-empty.
func chunkID(sourcePath string, position int, heading string) string {
	page := derivePageID(sourcePath)
	slug := "h0"
	if heading != "" {
		slug = "h" + fmt.Sprintf("%d", position+1) + "-" + slugify(heading)
	}
	return page + "#" + slug
}

// slugify converts a heading into a stable kebab-case slug. We avoid
// dependencies on third-party slug libraries.
func slugify(s string) string {
	s = strings.ToLower(s)
	var sb strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			sb.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && sb.Len() > 0 {
				sb.WriteRune('-')
				lastDash = true
			}
		}
	}
	out := strings.TrimRight(sb.String(), "-")
	if out == "" {
		return "section"
	}
	// Cap length so chunk IDs stay compact for the UI / JSON payloads.
	if len(out) > 48 {
		out = out[:48]
		out = strings.TrimRight(out, "-")
	}
	return out
}
