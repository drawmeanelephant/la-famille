package retrieval

import (
	"slices"
	"sort"
	"strings"

	"github.com/tbuddy/la-famille/internal/graph"
)

const (
	maxGraphHops    = 4
	maxGraphVisited = 256
)

// PathNode identifies the exact chunk grounding one page on a graph route.
type PathNode struct {
	PageID  string `json:"page_id"`
	ChunkID string `json:"chunk_id"`
	Title   string `json:"title"`
	URL     string `json:"url,omitempty"`
	Key     string `json:"key,omitempty"`
}

// GraphPath orders traversal nodes, while Edges retain actual link direction.
// A backlink can be traversed without asserting that its target links back.
type GraphPath struct {
	Nodes []PathNode  `json:"nodes"`
	Edges [][2]string `json:"edges"`
}

type GraphResult struct {
	Scored []Scored
	Paths  []GraphPath
}

// GraphRanker deliberately owns a lexical ranker, never a hybrid scorer.
// Expansion is bounded and opt-in and respects the lexical coverage guard.
type GraphRanker struct {
	lexical *Ranker
	pages   map[string][]Chunk
	adj     map[string]graph.Neighbors
}

func NewGraphRanker(c Corpus) *GraphRanker {
	g := &GraphRanker{lexical: NewRanker(c), pages: make(map[string][]Chunk)}
	for _, ch := range c.Chunks {
		node, ok := c.Graph.Nodes[ch.PageID]
		if ok && node.Render && !node.Missing && node.Type == "page" &&
			ch.SourceKind == "rag-content" && hasSourceEvidence(ch) {
			g.pages[ch.PageID] = append(g.pages[ch.PageID], ch)
		}
	}
	filtered := graph.Graph{Nodes: c.Graph.Nodes}
	for _, edge := range c.Graph.Edges {
		if len(g.pages[edge[0]]) > 0 && len(g.pages[edge[1]]) > 0 && edge[0] != edge[1] {
			filtered.Edges = append(filtered.Edges, edge)
		}
	}
	g.adj = graph.Adjacency(filtered)
	return g
}

func (g *GraphRanker) Rank(query string, topK int) []Scored {
	return g.Retrieve(query, topK).Scored
}

// Retrieve preserves the lexical head unless two independently matching pages
// justify a complete connecting route. Otherwise only reciprocal neighbors
// with a strong lexical score and complementary query evidence are promoted.
// It never appends a zero-relevance neighbor merely because a link exists.
func (g *GraphRanker) Retrieve(query string, topK int) GraphResult {
	if topK <= 0 {
		topK = 5
	}
	all := g.lexical.Rank(query, len(g.lexical.corpus.Chunks))
	result := GraphResult{Scored: slices.Clone(all[:min(topK, len(all))])}
	if len(all) == 0 {
		return result
	}
	best := make(map[string]Scored)
	var seeds []Scored
	for _, match := range all {
		id := match.Chunk.PageID
		if len(g.pages[id]) == 0 || !hasSourceEvidence(match.Chunk) {
			continue
		}
		if _, ok := best[id]; !ok {
			best[id] = match
			seeds = append(seeds, match)
		}
	}
	if len(seeds) == 0 {
		return result
	}
	terms := evidenceTerms(query)
	var named []Scored
	for _, seed := range seeds {
		if namesPage(query, seed.Chunk) {
			named = append(named, seed)
		}
	}
	// Explicit page names are stronger endpoint evidence than a shared generic
	// word. Otherwise require independently covered query terms and scores.
	endpoints := seeds[:min(4, len(seeds))]
	explicit := len(named) == 2
	if explicit {
		sort.SliceStable(named, func(i, j int) bool {
			return pageNamePosition(query, named[i].Chunk) < pageNamePosition(query, named[j].Chunk)
		})
		endpoints = named
	}
	for i, a := range endpoints {
		for _, b := range endpoints[i+1:] {
			if !explicit && (b.Score < seeds[0].Score*0.55 || !complementary(terms, a.Chunk, b.Chunk)) {
				continue
			}
			route := g.shortestPath(a.Chunk.PageID, b.Chunk.PageID)
			if len(route) < 3 || len(route) > topK {
				continue
			}
			path := GraphPath{}
			var selected []Scored
			for j, id := range route {
				match, ok := best[id]
				if !ok {
					// A bridge need not match the question lexically. Prefer a
					// chunk witnessing its links, then stable position/ID.
					match = Scored{Chunk: g.bridgeChunk(id, route)}
				}
				selected = append(selected, match)
				path.Nodes = append(path.Nodes, pathNode(match.Chunk))
				if j > 0 {
					edge := [2]string{route[j-1], id}
					if !slices.Contains(g.adj[edge[0]].Outbound, edge[1]) {
						edge[0], edge[1] = edge[1], edge[0]
					}
					path.Edges = append(path.Edges, edge)
				}
			}
			result.Scored = fillGraphSelection(selected, all, topK)
			result.Paths = []GraphPath{path}
			return result // one complete route, not a combinatorial bag of paths
		}
	}
	a := seeds[0]
	selected := []Scored{a}
	for _, b := range seeds[1:] {
		neighbors := g.adj[a.Chunk.PageID]
		if b.Score >= a.Score*0.55 && complementary(terms, a.Chunk, b.Chunk) &&
			slices.Contains(neighbors.Outbound, b.Chunk.PageID) &&
			slices.Contains(neighbors.Inbound, b.Chunk.PageID) {
			selected = append(selected, b)
			if len(selected) == 3 {
				break
			}
		}
	}
	if len(selected) > 1 {
		result.Scored = fillGraphSelection(selected, all, topK)
	}
	return result
}

func fillGraphSelection(selected, lexical []Scored, k int) []Scored {
	seen := make(map[string]bool)
	out := make([]Scored, 0, k)
	for _, list := range [][]Scored{selected, lexical} {
		for _, match := range list {
			if !seen[match.Chunk.ID] && len(out) < k {
				seen[match.Chunk.ID] = true
				out = append(out, match)
			}
		}
	}
	return out
}

func (g *GraphRanker) shortestPath(start, end string) []string {
	type visit struct {
		id    string
		route []string
	}
	queue := []visit{{start, []string{start}}}
	seen := map[string]bool{start: true}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if len(current.route)-1 >= maxGraphHops {
			continue
		}
		adj := g.adj[current.id]
		neighbors := append(slices.Clone(adj.Outbound), adj.Inbound...)
		sort.Strings(neighbors)
		for _, id := range slices.Compact(neighbors) {
			if seen[id] || len(g.pages[id]) == 0 {
				continue
			}
			route := append(slices.Clone(current.route), id)
			if id == end {
				return route
			}
			if len(seen) >= maxGraphVisited {
				return nil
			}
			seen[id] = true
			queue = append(queue, visit{id, route})
		}
	}
	return nil
}

func (g *GraphRanker) bridgeChunk(id string, route []string) Chunk {
	chunks := slices.Clone(g.pages[id])
	witnesses := func(ch Chunk) int {
		n := 0
		for _, page := range route {
			if page != id && strings.Contains(ch.Text, page+".md") {
				n++
			}
		}
		return n
	}
	sort.Slice(chunks, func(i, j int) bool {
		if a, b := witnesses(chunks[i]), witnesses(chunks[j]); a != b {
			return a > b
		}
		if chunks[i].Position != chunks[j].Position {
			return chunks[i].Position < chunks[j].Position
		}
		return chunks[i].ID < chunks[j].ID
	})
	return chunks[0]
}

func pathNode(ch Chunk) PathNode {
	title := ch.Title
	if title == "" {
		title = ch.PageID
	}
	return PathNode{PageID: ch.PageID, ChunkID: ch.ID, Title: title, URL: ch.URL}
}

func namesPage(query string, ch Chunk) bool {
	return pageNamePosition(query, ch) >= 0
}

func pageNamePosition(query string, ch Chunk) int {
	q := " " + strings.Join(tokenize(query), " ") + " "
	position := -1
	for _, name := range []string{ch.Title, ch.PageID} {
		tokens := tokenize(name)
		if len(tokens) > 0 && len(evidenceTerms(name)) > 0 {
			i := strings.Index(q, " "+strings.Join(tokens, " ")+" ")
			if i >= 0 && (position < 0 || i < position) {
				position = i
			}
		}
	}
	return position
}

// Heading-only chunks can score highly from their duplicated title, but they
// cannot support an endpoint fact or a bridge. Leave them in lexical ranking;
// graph grounding must choose actual source material.
func hasSourceEvidence(ch Chunk) bool {
	for _, line := range strings.Split(ch.Text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			return true
		}
	}
	return false
}

func evidenceTerms(text string) []string {
	const stop = " a an and are as at be between by can connect connected connection describe did do does explain for from got has have heading headings how i in is it its link linked me my of on or our page pages path please relate related relationship route say says should site that the their them there these they this through to via was we what when where which who with you your "
	var terms []string
	for _, term := range tokenize(text) {
		if !strings.Contains(stop, " "+term+" ") {
			terms = append(terms, term)
		}
	}
	return slices.Compact(terms)
}

func complementary(terms []string, a, b Chunk) bool {
	covered := func(ch Chunk) map[string]bool {
		m := make(map[string]bool)
		for _, term := range tokenize(ch.Text + " " + ch.Title) {
			m[term] = true
		}
		return m
	}
	ma, mb := covered(a), covered(b)
	onlyA, onlyB := false, false
	for _, term := range terms {
		onlyA = onlyA || (ma[term] && !mb[term])
		onlyB = onlyB || (mb[term] && !ma[term])
	}
	return onlyA && onlyB
}
