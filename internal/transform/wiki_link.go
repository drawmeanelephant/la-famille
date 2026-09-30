package transform

import (
	"bytes"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/tbuddy/la-famille/internal/content"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

const wikiLinkScheme = "lafamille-wiki"

// WikiLinkParser recognizes [[target]], [[target|alias]], and
// [[target#heading]] inline links.
type WikiLinkParser struct{}

func (*WikiLinkParser) Trigger() []byte {
	return []byte{'['}
}

func (*WikiLinkParser) Parse(_ ast.Node, reader text.Reader, _ parser.Context) ast.Node {
	line, _ := reader.PeekLine()
	if len(line) < 4 || line[0] != '[' || line[1] != '[' {
		return nil
	}
	closeAt := bytes.Index(line[2:], []byte("]]"))
	if closeAt < 0 {
		return nil
	}

	body := strings.TrimSpace(string(line[2 : 2+closeAt]))
	if body == "" {
		return nil
	}
	target, alias := body, body
	if separator := strings.IndexByte(body, '|'); separator >= 0 {
		target = strings.TrimSpace(body[:separator])
		alias = strings.TrimSpace(body[separator+1:])
		if target == "" || alias == "" {
			return nil
		}
	}
	pageTarget, heading := target, ""
	if hash := strings.IndexByte(target, '#'); hash >= 0 {
		pageTarget = strings.TrimSpace(target[:hash])
		heading = strings.TrimSpace(target[hash+1:])
		if pageTarget == "" {
			return nil
		}
	}

	reader.Advance(closeAt + 4)
	link := ast.NewLink()
	link.Destination = []byte(wikiLinkDestination(pageTarget, heading))
	link.AppendChild(link, ast.NewString([]byte(alias)))
	return link
}

// ParseWikiLinkDestination decodes the private destination used between the
// wiki-link parser and LinkTransformer.
func ParseWikiLinkDestination(destination string) (target, heading string, ok bool) {
	u, err := url.Parse(destination)
	if err != nil || u.Scheme != wikiLinkScheme || u.Opaque == "" {
		return "", "", false
	}
	target, err = url.PathUnescape(u.Opaque)
	if err != nil || strings.TrimSpace(target) == "" {
		return "", "", false
	}
	return target, u.Fragment, true
}

func wikiLinkDestination(target, heading string) string {
	u := url.URL{
		Scheme:   wikiLinkScheme,
		Opaque:   url.PathEscape(target),
		Fragment: heading,
	}
	return u.String()
}

// ResolveWikiTarget finds a content page by a relative/root-relative Markdown
// path, filename, or case-insensitive page title. Ambiguous title matches are
// left unresolved instead of choosing an arbitrary page.
func ResolveWikiTarget(currentFile, target string, fileMap map[string]*content.FileMeta) (string, *content.FileMeta, bool) {
	target = strings.TrimSpace(strings.ReplaceAll(target, "\\", "/"))
	if target == "" {
		return "", nil, false
	}

	rootRelative := strings.HasPrefix(target, "/")
	target = strings.TrimPrefix(target, "/")
	candidates := []string{target}
	if !rootRelative {
		if dir := filepath.ToSlash(filepath.Dir(currentFile)); dir != "." {
			candidates = append([]string{path.Join(dir, target)}, candidates...)
		}
	}
	for _, candidate := range candidates {
		candidate, ok := wikiSourcePath(candidate)
		if !ok {
			continue
		}
		if meta, exists := fileMap[candidate]; exists {
			return candidate, meta, true
		}
	}

	key := wikiKey(strings.TrimSuffix(target, ".md"))
	matches := make(map[string]*content.FileMeta)
	for sourcePath, meta := range fileMap {
		sourceName := path.Base(strings.TrimSuffix(sourcePath, ".md"))
		if wikiKey(strings.TrimSuffix(sourcePath, ".md")) == key ||
			wikiKey(sourceName) == key ||
			(meta != nil && wikiKey(meta.Title) == key) {
			matches[sourcePath] = meta
		}
	}
	if len(matches) != 1 {
		return "", nil, false
	}
	for sourcePath, meta := range matches {
		return sourcePath, meta, true
	}
	return "", nil, false
}

// UnresolvedWikiTargetPath returns a safe source-style identity for a target
// that does not yet exist, allowing it to pass through the normal stub path.
func UnresolvedWikiTargetPath(currentFile, target string) string {
	target = strings.TrimSpace(strings.TrimSuffix(strings.ReplaceAll(target, "\\", "/"), ".md"))
	rootRelative := strings.HasPrefix(target, "/")
	target = strings.Trim(target, "/")
	if !rootRelative {
		if currentDir := filepath.ToSlash(filepath.Dir(currentFile)); currentDir != "." {
			target = path.Join(currentDir, target)
		}
	}
	parts := strings.Split(target, "/")
	for i, part := range parts {
		parts[i] = wikiSlug(part)
	}
	name := strings.Trim(strings.Join(parts, "/"), "/")
	if name == "" {
		name = "unresolved-note"
	}
	return name + ".md"
}

// WikiTargetTitle returns the readable page name supplied in a wiki link.
func WikiTargetTitle(target string) string {
	target = strings.TrimSpace(strings.TrimSuffix(target, ".md"))
	if slash := strings.LastIndexAny(target, "/\\"); slash >= 0 {
		target = target[slash+1:]
	}
	return strings.TrimSpace(target)
}

// WikiHeadingFragment applies Goldmark's configured heading-id rules to a
// heading label.
func WikiHeadingFragment(heading string) string {
	if strings.TrimSpace(heading) == "" {
		return ""
	}
	return string(parser.NewContext().IDs().Generate([]byte(heading), ast.KindHeading))
}

func wikiSourcePath(candidate string) (string, bool) {
	candidate = filepath.ToSlash(filepath.Clean(filepath.FromSlash(candidate)))
	if candidate == "." || !filepath.IsLocal(filepath.FromSlash(candidate)) ||
		strings.Contains(candidate, "%2e%2e") {
		return "", false
	}
	if !strings.HasSuffix(strings.ToLower(candidate), ".md") {
		candidate += ".md"
	}
	return candidate, true
}

func wikiKey(value string) string {
	value = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ".md")
	value = strings.NewReplacer("-", " ", "_", " ").Replace(value)
	return strings.Join(strings.Fields(value), " ")
}

func wikiSlug(value string) string {
	var out strings.Builder
	dash := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			out.WriteRune(r)
			dash = false
		case !dash && out.Len() > 0:
			out.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(out.String(), "-")
}
