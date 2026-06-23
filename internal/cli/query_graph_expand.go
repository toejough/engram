package cli

import (
	"sort"

	"github.com/toejough/engram/internal/vaultgraph"
)

// graphBridgeBasenames performs GraphRAG-local-search seed expansion: it
// traverses the vault wikilink graph from the cosine-matched note seeds
// (undirected, hops-bounded, capacity-bounded) and returns the BRIDGE
// basenames — visited nodes that are not themselves seeds. Pure. Returns
// nil when hops <= 0 or seeds is empty.
func graphBridgeBasenames(notes []vaultgraph.Note, seeds []string, hops, capacity int) []string {
	if hops <= 0 || len(seeds) == 0 {
		return nil
	}

	graph := vaultgraph.BuildGraph(notes)
	result := vaultgraph.BFSWithCap(graph, seeds, hops, capacity)

	seedSet := make(map[string]struct{}, len(seeds))
	for _, seed := range seeds {
		seedSet[seed] = struct{}{}
	}

	bridges := make([]string, 0, len(result.Visited))

	for basename := range result.Visited {
		if _, isSeed := seedSet[basename]; !isSeed {
			bridges = append(bridges, basename)
		}
	}

	sort.Strings(bridges)

	return bridges
}
