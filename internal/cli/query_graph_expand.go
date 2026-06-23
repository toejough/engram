package cli

import (
	"sort"

	"github.com/toejough/engram/internal/vaultgraph"
)

// appendGraphBridges expands the matched set in place with wikilink-reachable
// bridge notes (GraphRAG local search). hops is the already-resolved BFS depth
// (the caller applies the default); a value <= 0 disables expansion. Bridges
// are bounded so the matched set stays under matchSetCap, keeping clustering
// O(n^2)-bounded. BuildGraph is rebuilt per query (CPU only — notes are already
// scanned); acceptable at current vault sizes, a caching target if it ever bites.
func appendGraphBridges(
	matchSet *matchedSet,
	notes []vaultgraph.Note,
	hits []compatibleSidecar,
	noteUnion []scoredCandidate,
	hops int,
) {
	if hops <= 0 || len(matchSet.members) >= matchSetCap {
		return
	}

	seeds := make([]string, 0, len(noteUnion))
	for _, candidate := range noteUnion {
		seeds = append(seeds, candidate.basename)
	}

	hitByBasename := make(map[string]compatibleSidecar, len(hits))
	for _, hit := range hits {
		hitByBasename[hit.note.Basename] = hit
	}

	// Invariant: seeds are exactly the note members already in matchSet (both
	// derive from noteUnion), and BFS counts seeds in its Visited cap. Budgeting
	// matchSetCap total nodes therefore leaves matchSetCap-len(members) bridges,
	// so the post-append set stays <= matchSetCap.
	capacity := matchSetCap - len(matchSet.members) + len(seeds)
	bridges := graphBridgeBasenames(notes, seeds, hops, capacity)
	matchSet.members = append(matchSet.members, buildBridgeMembers(bridges, hitByBasename)...)
}

// buildBridgeMembers turns graph-bridge basenames into matchedMembers using
// already-loaded sidecars. Bridges have no query cosine: the cluster coordinate
// is the situation axis, score is 0, and content is empty (the recall skill
// fetches it via `engram show`). Only bridges with a compatible sidecar are
// included, since clustering needs a vector.
func buildBridgeMembers(
	bridges []string,
	hitByBasename map[string]compatibleSidecar,
) []matchedMember {
	members := make([]matchedMember, 0, len(bridges))

	for _, basename := range bridges {
		hit, ok := hitByBasename[basename]
		if !ok {
			continue
		}

		members = append(members, matchedMember{
			basename: basename,
			notePath: pathOf(basename),
			// No query cosine for a bridge, so the "winning coord" invariant on
			// matchedMember.vector doesn't apply; use the situation axis as the
			// clustering coordinate (same axis notes embed on).
			vector:        hit.sidecar.SituationVector,
			sitVec:        hit.sidecar.SituationVector,
			bodyVec:       hit.sidecar.BodyVector,
			score:         0,
			content:       "",
			graphExpanded: true,
		})
	}

	return members
}

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
