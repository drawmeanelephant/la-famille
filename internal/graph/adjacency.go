package graph

import "sort"

// Neighbors holds the resolved one-degree links of a single node.
type Neighbors struct {
	// Inbound lists the ids of nodes that link to this node.
	Inbound []string `json:"inbound"`
	// Outbound lists the ids of nodes this node links to.
	Outbound []string `json:"outbound"`
}

// Adjacency resolves Edges into per-node inbound and outbound neighbor lists.
//
// Edges are recorded as [source, target], so for the edge [a, b] the node b is
// outbound from a and the node a is inbound to b. Deriving both directions from
// Edges here keeps the two lists consistent with each other and with the graph
// they came from; consumers no longer re-derive adjacency themselves.
//
// A page may link to the same target several times, so both lists are
// deduplicated, and both are sorted so repeated builds emit identical bytes.
// Every node in g.Nodes gets an entry, including isolated ones, so callers can
// index the result without nil checks.
func Adjacency(g Graph) map[string]Neighbors {
	inSets := make(map[string]map[string]struct{}, len(g.Nodes))
	outSets := make(map[string]map[string]struct{}, len(g.Nodes))

	ids := make(map[string]struct{}, len(g.Nodes))
	for id := range g.Nodes {
		ids[id] = struct{}{}
	}

	for _, e := range g.Edges {
		source, target := e[0], e[1]
		if source == "" || target == "" {
			continue
		}
		ids[source] = struct{}{}
		ids[target] = struct{}{}
		addTo(outSets, source, target)
		addTo(inSets, target, source)
	}

	result := make(map[string]Neighbors, len(ids))
	for id := range ids {
		result[id] = Neighbors{
			Inbound:  sortedMembers(inSets[id]),
			Outbound: sortedMembers(outSets[id]),
		}
	}
	return result
}

func addTo(sets map[string]map[string]struct{}, key, value string) {
	set, ok := sets[key]
	if !ok {
		set = make(map[string]struct{})
		sets[key] = set
	}
	set[value] = struct{}{}
}

// sortedMembers returns the set's members in sorted order. The result is always
// non-nil so it marshals to [] rather than null.
func sortedMembers(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for member := range set {
		out = append(out, member)
	}
	sort.Strings(out)
	return out
}
