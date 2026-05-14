package vaultgraph

// Components partitions g's nodes into connected components, treating edges as undirected.
// Returns one slice per component; the inner slices are unordered. Isolated nodes form
// singleton components. The outer ordering is unspecified.
func Components(g Graph) [][]string {
	parent := make(map[string]string, len(g.Notes))

	for name := range g.Notes {
		parent[name] = name
	}

	for src, dsts := range g.Outgoing {
		for dst := range dsts {
			union(parent, src, dst)
		}
	}

	groups := make(map[string][]string, len(parent))
	for name := range parent {
		root := find(parent, name)
		groups[root] = append(groups[root], name)
	}

	out := make([][]string, 0, len(groups))
	for _, members := range groups {
		out = append(out, members)
	}

	return out
}

func find(parent map[string]string, name string) string {
	root := name
	for parent[root] != root {
		root = parent[root]
	}

	// Path compression.
	current := name
	for parent[current] != root {
		next := parent[current]
		parent[current] = root
		current = next
	}

	return root
}

func union(parent map[string]string, a, b string) {
	rootA := find(parent, a)
	rootB := find(parent, b)

	if rootA == rootB {
		return
	}

	parent[rootA] = rootB
}
