package retrieval

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// BuildGraphAnswerPrompt gives every selected chunk a share of the source
// budget, rather than using the lexical UI's 160-rune preview as evidence.
// Only keys with actual source text are returned. Incomplete routes are not
// presented to the model or the UI as grounded paths.
func BuildGraphAnswerPrompt(question string, result GraphResult, budget PromptBudget) (string, []CitationHint, []GraphPath) {
	if budget.MaxChunks <= 0 {
		budget.MaxChunks = 8
	}
	if budget.MaxContextChars <= 0 {
		budget.MaxContextChars = 6000
	}
	scored := result.Scored[:min(len(result.Scored), budget.MaxChunks)]
	var chunks []Chunk
	remaining := budget.MaxContextChars
	for i, match := range scored {
		share := remaining / (len(scored) - i)
		overhead := len(fmt.Sprintf("\n[%d]\n\n", i+1))
		if share <= overhead {
			break
		}
		text := strings.TrimSpace(match.Chunk.Text)
		if len(text) > share-overhead {
			text = text[:share-overhead]
			for !utf8.ValidString(text) {
				text = text[:len(text)-1]
			}
		}
		if text == "" {
			break
		}
		ch := match.Chunk
		ch.Text = text
		chunks = append(chunks, ch)
		remaining -= len(text) + overhead
	}
	cites := NewCitations(chunks)
	hints := cites.Hints()
	for i := range hints {
		hints[i].Excerpt = chunks[i].Text
	}
	paths := KeyedPaths(result.Paths, cites)
	var sb strings.Builder
	sb.WriteString(systemPrompt())
	sb.WriteString("\n\nCitation key map:\n")
	for _, hint := range hints {
		fmt.Fprintf(&sb, "  [%s] %s\n", hint.Key, fallbackTitle(hint))
	}
	sb.WriteString("\nSource material:\n")
	for _, hint := range hints {
		fmt.Fprintf(&sb, "\n[%s]\n%s\n", hint.Key, hint.Excerpt)
	}
	if len(paths) > 0 {
		sb.WriteString("\nVerified site-link routes (links show navigation, not proof of causation):\n")
		for _, path := range paths {
			var labels []string
			for _, node := range path.Nodes {
				labels = append(labels, fmt.Sprintf("%s [%s]", node.Title, node.Key))
			}
			sb.WriteString(strings.Join(labels, " → ") + "\n")
			for _, edge := range path.Edges {
				fmt.Fprintf(&sb, "  Outbound link: %s -> %s\n", edge[0], edge[1])
			}
		}
		sb.WriteString("Explain the connecting route, naming and citing every page including intermediate pages. Use source text for facts; a link alone does not imply a factual relationship.\n")
	}
	sb.WriteString("\nQuestion: " + strings.TrimSpace(question) + "\nAnswer:")
	return sb.String(), hints, paths
}

// KeyedPaths drops a whole route if any grounding chunk was excluded from the
// prompt. It copies nodes so request-local citation keys never mutate the graph.
func KeyedPaths(paths []GraphPath, cites *Citations) []GraphPath {
	var out []GraphPath
	for _, path := range paths {
		copyPath := GraphPath{Nodes: slices.Clone(path.Nodes), Edges: slices.Clone(path.Edges)}
		complete := len(path.Nodes) >= 2
		for i := range copyPath.Nodes {
			copyPath.Nodes[i].Key = cites.KeyFor(copyPath.Nodes[i].ChunkID)
			complete = complete && copyPath.Nodes[i].Key != ""
		}
		if complete {
			out = append(out, copyPath)
		}
	}
	return out
}

// CitedPaths only exposes routes for which the answer cited every page's exact
// grounding chunk. Citation membership does not verify the model's prose.
func CitedPaths(paths []GraphPath, verifiedKeys []string) []GraphPath {
	var out []GraphPath
	for _, path := range paths {
		complete := true
		for _, node := range path.Nodes {
			complete = complete && slices.Contains(verifiedKeys, node.Key)
		}
		if complete {
			out = append(out, path)
		}
	}
	return out
}
