package graphexplorer

import (
	"sort"
	"strings"
	"unicode"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/graph"
)

// HomepageID is the page id of the site homepage. It is exempt from the orphan
// rule: nothing links to the front page of a freshly seeded site, and flagging
// it as an orphan is noise rather than a finding.
const HomepageID = "index"

// NodeData is one page as the explorer page consumes it. Everything the browser
// needs is resolved here — classification, title, public URL, and both link
// directions — so the client renders the payload rather than re-deriving it.
type NodeData struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Type       string   `json:"type"`
	Render     bool     `json:"render"`
	Stub       bool     `json:"stub"`
	Orphan     bool     `json:"orphan"`
	URL        string   `json:"url,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Categories []string `json:"categories,omitempty"`
	Author     string   `json:"author,omitempty"`
	Date       string   `json:"date,omitempty"`
	WordCount  int      `json:"word_count"`
	Inbound    []string `json:"inbound"`
	Outbound   []string `json:"outbound"`
}

// Data is the full payload written to <output>/graph/data.json.
type Data struct {
	Nodes              []NodeData  `json:"nodes"`
	Edges              [][2]string `json:"edges"`
	LargeSiteThreshold int         `json:"large_site_threshold"`
	BasePath           string      `json:"base_path"`
}

// BuildData assembles the explorer payload from the generated graph, the page
// metadata, and the output path each rendered page was written to.
//
// pageOutputs maps a node id to the output-relative path of its rendered HTML
// (as returned by transform.GetOutputURL), which is what makes the emitted URLs
// slug-aware. Ids missing from the map get no URL, which is correct for raw
// markdown pages and missing-link stubs.
//
// The result is fully sorted and deduplicated so two builds of the same input
// emit identical bytes.
func BuildData(cfg config.Config, g graph.Graph, meta map[string]map[string]interface{}, pageOutputs map[string]string) Data {
	adj := graph.Adjacency(g)

	ids := make([]string, 0, len(adj))
	for id := range adj {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	nodes := make([]NodeData, 0, len(ids))
	for _, id := range ids {
		node, isContent := g.Nodes[id]
		neighbors := adj[id]
		pageMeta := meta[id]

		// A node that only ever appeared as a link target was never walked as
		// content, so it is a missing-link stub by definition.
		stub := !isContent || node.Type == "stub" || node.Missing
		render := isContent && node.Render

		nodeType := node.Type
		if nodeType == "" {
			nodeType = "page"
		}

		data := NodeData{
			ID:         id,
			Title:      titleFor(id, pageMeta),
			Type:       nodeType,
			Render:     render,
			Stub:       stub,
			Orphan:     len(neighbors.Inbound) == 0 && id != HomepageID,
			Tags:       stringSlice(pageMeta["tags"]),
			Categories: stringSlice(pageMeta["categories"]),
			Author:     stringValue(pageMeta["author"]),
			Date:       stringValue(pageMeta["date"]),
			WordCount:  intValue(pageMeta["word_count"]),
			Inbound:    neighbors.Inbound,
			Outbound:   neighbors.Outbound,
		}

		// Only rendered pages have a browsable URL. Raw markdown pages and
		// stubs deliberately get none so the client shows why instead of
		// linking somewhere that would 404.
		if render {
			if out, ok := pageOutputs[id]; ok && out != "" {
				data.URL = cfg.PublicPathForOutput(out)
			}
		}

		nodes = append(nodes, data)
	}

	return Data{
		Nodes:              nodes,
		Edges:              uniqueSortedEdges(g.Edges),
		LargeSiteThreshold: LargeSiteThreshold,
		BasePath:           cfg.BasePath(),
	}
}

// uniqueSortedEdges collapses repeated links between the same two pages and
// orders the result so the payload is stable across builds.
func uniqueSortedEdges(edges [][2]string) [][2]string {
	seen := make(map[[2]string]struct{}, len(edges))
	out := make([][2]string, 0, len(edges))
	for _, e := range edges {
		if e[0] == "" || e[1] == "" {
			continue
		}
		if _, dup := seen[e]; dup {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i][0] != out[j][0] {
			return out[i][0] < out[j][0]
		}
		return out[i][1] < out[j][1]
	})
	return out
}

func titleFor(id string, meta map[string]interface{}) string {
	if title := stringValue(meta["title"]); title != "" {
		return title
	}
	return TitleFromID(id)
}

// TitleFromID produces a readable fallback title for a page with no frontmatter
// title, e.g. "docs/getting-started.md" becomes "Getting Started".
func TitleFromID(id string) string {
	base := id
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}
	base = strings.TrimSuffix(base, ".md")

	var b strings.Builder
	startOfWord := true
	for _, r := range base {
		if r == '-' || r == '_' {
			// Collapse runs of separators into a single space.
			if b.Len() > 0 && !startOfWord {
				b.WriteRune(' ')
			}
			startOfWord = true
			continue
		}
		if startOfWord {
			b.WriteRune(unicode.ToUpper(r))
			startOfWord = false
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// stringSlice normalizes a frontmatter list value. Values arrive as []string in
// process, but YAML decoding can also yield []interface{}, so both are handled.
// The result is nil when empty so it is omitted from the payload entirely.
func stringSlice(v interface{}) []string {
	switch typed := v.(type) {
	case []string:
		if len(typed) == 0 {
			return nil
		}
		out := make([]string, len(typed))
		copy(out, typed)
		return out
	case []interface{}:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s := stringValue(item); s != "" {
				out = append(out, s)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	default:
		return nil
	}
}

func stringValue(v interface{}) string {
	s, _ := v.(string)
	return s
}

func intValue(v interface{}) int {
	switch typed := v.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}
