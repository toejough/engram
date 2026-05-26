package vaultgraph

// Graph is the wikilink graph over a vault: a node per .md file, an edge per
// resolved wikilink. Edges are directed (a wikilink from A to B is an outgoing
// edge from A and an incoming edge to B). Components treat edges as undirected.
//
// Broken wikilinks (target not in the note set) and self-links are dropped at
// build time. Duplicate wikilink targets within a single note collapse to one edge.
type Graph struct {
	Notes    map[string]Note                // basename → note metadata
	Outgoing map[string]map[string]struct{} // src basename → set of dst basenames
	Incoming map[string]map[string]struct{} // dst basename → set of src basenames
}

// InDegree returns the count of notes that wikilink to basename. Returns 0 for
// unknown basenames.
func (g Graph) InDegree(basename string) int {
	return len(g.Incoming[basename])
}

// InDegreeIn returns the count of notes within subset that wikilink to
// basename. Returns 0 for unknown basenames or empty subsets. The
// subset is used as a set (only keys are consulted).
func (g Graph) InDegreeIn(basename string, subset map[string]struct{}) int {
	if len(subset) == 0 {
		return 0
	}

	count := 0

	for source := range g.Incoming[basename] {
		if _, ok := subset[source]; ok {
			count++
		}
	}

	return count
}

// UndirectedNeighbors returns the set of basenames connected to basename by an
// edge in either direction. Order is unspecified.
func (g Graph) UndirectedNeighbors(basename string) []string {
	seen := make(map[string]struct{}, len(g.Outgoing[basename])+len(g.Incoming[basename]))

	for target := range g.Outgoing[basename] {
		seen[target] = struct{}{}
	}

	for source := range g.Incoming[basename] {
		seen[source] = struct{}{}
	}

	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}

	return out
}

// BuildGraph constructs the graph from a flat slice of notes (e.g. ScanVault output).
// Drops broken-target edges (target not in notes) and self-links.
func BuildGraph(notes []Note) Graph {
	noteByName := make(map[string]Note, len(notes))
	for _, note := range notes {
		noteByName[note.Basename] = note
	}

	outgoing := make(map[string]map[string]struct{}, len(notes))
	incoming := make(map[string]map[string]struct{}, len(notes))

	for _, note := range notes {
		for _, target := range note.Outgoing {
			if target == note.Basename {
				continue
			}

			if _, exists := noteByName[target]; !exists {
				continue
			}

			ensureSet(outgoing, note.Basename)[target] = struct{}{}
			ensureSet(incoming, target)[note.Basename] = struct{}{}
		}
	}

	return Graph{Notes: noteByName, Outgoing: outgoing, Incoming: incoming}
}

func ensureSet(m map[string]map[string]struct{}, key string) map[string]struct{} {
	set, ok := m[key]
	if !ok {
		set = make(map[string]struct{})
		m[key] = set
	}

	return set
}
