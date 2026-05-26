package vaultgraph

// BFSResult bundles the outcome of a capped breadth-first traversal:
// the set of visited basenames, the deepest hop reached, and whether
// the cap forced expansion to stop.
type BFSResult struct {
	Visited     map[string]struct{}
	HopsReached int
	Capped      bool
}

// BFSWithCap traverses g starting from the given seeds using
// undirected wikilink edges (outgoing ∪ incoming), bounded by
// maxDepth hops and a hard cap on the visited-set size.
//
// Behavior:
//   - Each seed is at depth 0 and is added to Visited (assuming it is a
//     known basename in g).
//   - Unknown seeds (not in g.Notes) are skipped silently.
//   - When adding a neighbor would push the visited count above cap, the
//     neighbor is *not* added and Capped flips to true. Already-queued
//     work continues to drain at the current depth; the next depth is
//     not entered.
//   - HopsReached is the deepest hop fully completed (0 if seeds only,
//     up to maxDepth).
//   - Empty seeds returns an empty Visited and HopsReached == 0.
func BFSWithCap(graph Graph, seeds []string, maxDepth, capacity int) BFSResult {
	visited := make(map[string]struct{})

	if len(seeds) == 0 {
		return BFSResult{Visited: visited}
	}

	frontier, seedCapped := admitSeeds(graph, seeds, visited, capacity)
	if seedCapped {
		return BFSResult{Visited: visited, Capped: true}
	}

	hopsReached := 0
	capped := false

	for hop := 1; hop <= maxDepth && !capped && len(frontier) > 0; hop++ {
		var next []string

		next, capped = expandOneHop(graph, frontier, visited, capacity)
		hopsReached = hop
		frontier = next
	}

	return BFSResult{Visited: visited, HopsReached: hopsReached, Capped: capped}
}

// admitSeeds adds each seed to visited (subject to capacity and
// existence-in-graph checks) and returns the initial frontier plus a
// flag for whether the cap fired during seeding.
func admitSeeds(
	graph Graph, seeds []string, visited map[string]struct{}, capacity int,
) ([]string, bool) {
	frontier := make([]string, 0, len(seeds))

	for _, seed := range seeds {
		if _, ok := graph.Notes[seed]; !ok {
			continue
		}

		if _, dup := visited[seed]; dup {
			continue
		}

		if len(visited) >= capacity {
			return frontier, true
		}

		visited[seed] = struct{}{}
		frontier = append(frontier, seed)
	}

	return frontier, false
}

// expandOneHop visits every neighbor of every node in frontier, admits
// novel ones into visited (subject to capacity), and returns the next
// frontier plus a flag indicating whether the cap was reached this hop.
func expandOneHop(
	graph Graph, frontier []string, visited map[string]struct{}, capacity int,
) ([]string, bool) {
	next := make([]string, 0)

	for _, node := range frontier {
		for _, neighbor := range graph.UndirectedNeighbors(node) {
			if _, dup := visited[neighbor]; dup {
				continue
			}

			if len(visited) >= capacity {
				return next, true
			}

			visited[neighbor] = struct{}{}
			next = append(next, neighbor)
		}
	}

	return next, false
}
