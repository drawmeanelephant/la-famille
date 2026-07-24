package graph

import (
	"reflect"
	"testing"
)

// nodesFor builds a Graph with the given ids and edges. Edges use the same
// [source, target] convention that transform.LinkTransformer writes.
func nodesFor(ids []string, edges [][2]string) Graph {
	g := Graph{Nodes: map[string]Node{}, Edges: edges}
	for _, id := range ids {
		g.Nodes[id] = Node{Type: "page", Render: true}
	}
	return g
}

// TestAdjacencyDirection is the regression guard for the direction of an edge.
// For the edge [a, b] — "a links to b" — b must appear in a's Outbound and a
// must appear in b's Inbound, never the reverse. An implementation that reads
// the edge backwards makes Inbound and Outbound mirror each other, which is
// exactly what this asserts against.
func TestAdjacencyDirection(t *testing.T) {
	g := nodesFor([]string{"a", "b"}, [][2]string{{"a", "b"}})
	adj := Adjacency(g)

	if got, want := adj["a"].Outbound, []string{"b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("a.Outbound = %v, want %v", got, want)
	}
	if got := adj["a"].Inbound; len(got) != 0 {
		t.Errorf("a.Inbound = %v, want empty (nothing links to a)", got)
	}
	if got, want := adj["b"].Inbound, []string{"a"}; !reflect.DeepEqual(got, want) {
		t.Errorf("b.Inbound = %v, want %v", got, want)
	}
	if got := adj["b"].Outbound; len(got) != 0 {
		t.Errorf("b.Outbound = %v, want empty (b links to nothing)", got)
	}
}

// TestAdjacencyInboundAndOutboundDiffer guards the specific symptom of an
// inverted edge read: every node reporting the same set both ways.
func TestAdjacencyInboundAndOutboundDiffer(t *testing.T) {
	// hub links out to leaf1 and leaf2; only root links in to hub.
	g := nodesFor(
		[]string{"root", "hub", "leaf1", "leaf2"},
		[][2]string{{"root", "hub"}, {"hub", "leaf1"}, {"hub", "leaf2"}},
	)
	adj := Adjacency(g)

	hub := adj["hub"]
	if reflect.DeepEqual(hub.Inbound, hub.Outbound) {
		t.Fatalf("hub Inbound and Outbound are identical (%v) — edges are being read in one direction only", hub.Inbound)
	}
	if got, want := hub.Inbound, []string{"root"}; !reflect.DeepEqual(got, want) {
		t.Errorf("hub.Inbound = %v, want %v", got, want)
	}
	if got, want := hub.Outbound, []string{"leaf1", "leaf2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("hub.Outbound = %v, want %v", got, want)
	}
}

// TestAdjacencyDeduplicatesRepeatedEdges covers a page linking to the same
// target more than once, which the walk records as several identical edges.
func TestAdjacencyDeduplicatesRepeatedEdges(t *testing.T) {
	g := nodesFor([]string{"a", "b"}, [][2]string{{"a", "b"}, {"a", "b"}, {"a", "b"}})
	adj := Adjacency(g)

	if got, want := adj["a"].Outbound, []string{"b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("a.Outbound = %v, want %v (repeated links must collapse)", got, want)
	}
	if got, want := adj["b"].Inbound, []string{"a"}; !reflect.DeepEqual(got, want) {
		t.Errorf("b.Inbound = %v, want %v (repeated links must collapse)", got, want)
	}
}

func TestAdjacencySortsNeighbors(t *testing.T) {
	g := nodesFor(
		[]string{"src", "x", "m", "a"},
		[][2]string{{"src", "x"}, {"src", "m"}, {"src", "a"}},
	)
	adj := Adjacency(g)

	if got, want := adj["src"].Outbound, []string{"a", "m", "x"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Outbound = %v, want sorted %v", got, want)
	}
}

// TestAdjacencyCoversIsolatedNodes keeps the result safe to index directly:
// a node with no links at all still gets an entry with empty, non-nil lists.
func TestAdjacencyCoversIsolatedNodes(t *testing.T) {
	g := nodesFor([]string{"lonely"}, nil)
	adj := Adjacency(g)

	n, ok := adj["lonely"]
	if !ok {
		t.Fatal("isolated node missing from adjacency result")
	}
	if n.Inbound == nil || n.Outbound == nil {
		t.Errorf("isolated node lists must be non-nil so they marshal to [], got in=%v out=%v", n.Inbound, n.Outbound)
	}
	if len(n.Inbound) != 0 || len(n.Outbound) != 0 {
		t.Errorf("isolated node should have no neighbors, got in=%v out=%v", n.Inbound, n.Outbound)
	}
}

// TestAdjacencyIncludesEdgeOnlyTargets covers stub targets: a link can point at
// an id that never became a content node.
func TestAdjacencyIncludesEdgeOnlyTargets(t *testing.T) {
	g := nodesFor([]string{"a"}, [][2]string{{"a", "missing/target"}})
	adj := Adjacency(g)

	n, ok := adj["missing/target"]
	if !ok {
		t.Fatal("target that only appears in an edge is missing from adjacency result")
	}
	if got, want := n.Inbound, []string{"a"}; !reflect.DeepEqual(got, want) {
		t.Errorf("missing/target Inbound = %v, want %v", got, want)
	}
}

func TestAdjacencyHandlesSelfLink(t *testing.T) {
	g := nodesFor([]string{"a"}, [][2]string{{"a", "a"}})
	adj := Adjacency(g)

	if got, want := adj["a"].Inbound, []string{"a"}; !reflect.DeepEqual(got, want) {
		t.Errorf("self-link Inbound = %v, want %v", got, want)
	}
	if got, want := adj["a"].Outbound, []string{"a"}; !reflect.DeepEqual(got, want) {
		t.Errorf("self-link Outbound = %v, want %v", got, want)
	}
}

// TestAdjacencyInboundMatchesBacklinks pins Adjacency's inbound direction to
// the same convention internal/transform uses when it builds the backlinks map
// (Backlinks[target] gets the source appended), so the two cannot drift apart.
func TestAdjacencyInboundMatchesBacklinks(t *testing.T) {
	edges := [][2]string{{"a", "b"}, {"c", "b"}, {"a", "c"}}

	backlinks := map[string][]string{}
	for _, e := range edges {
		source, target := e[0], e[1]
		backlinks[target] = append(backlinks[target], source)
	}

	adj := Adjacency(nodesFor([]string{"a", "b", "c"}, edges))

	for target, sources := range backlinks {
		got := adj[target].Inbound
		if len(got) != len(sources) {
			t.Errorf("%s: Inbound = %v, backlinks = %v", target, got, sources)
			continue
		}
		for _, source := range sources {
			found := false
			for _, g := range got {
				if g == source {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s: Inbound %v missing backlink source %q", target, got, source)
			}
		}
	}
}
