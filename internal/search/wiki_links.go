package search

import (
	"sort"
	"strings"

	"github.com/tbuddy/la-famille/internal/markdown"
	"github.com/tbuddy/la-famille/internal/transform"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// ExtractWikiLinkTargets returns the unique wiki-link destinations in Markdown
// source, excluding aliases, headings, and code examples.
func ExtractWikiLinkTargets(source []byte) []string {
	if len(source) == 0 {
		return nil
	}
	doc := markdown.NewEngine(nil).Parser().Parse(text.NewReader(source))
	seen := make(map[string]struct{})
	var targets []string
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		link, ok := node.(*ast.Link)
		if !ok {
			return ast.WalkContinue, nil
		}
		target, _, isWikiLink := transform.ParseWikiLinkDestination(string(link.Destination))
		if !isWikiLink {
			return ast.WalkContinue, nil
		}
		target = strings.TrimSpace(target)
		if target == "" {
			return ast.WalkContinue, nil
		}
		if _, exists := seen[target]; !exists {
			seen[target] = struct{}{}
			targets = append(targets, target)
		}
		return ast.WalkContinue, nil
	})
	sort.Strings(targets)
	return targets
}
