package vaultgraph

import "sort"

// Follow expands the cascade frontier. Given a set of files to follow and a
// set of already-read files, returns the deduplicated union of outgoing
// wikilinks AND backlinks from the follow set, minus the follow set itself
// AND the already-read set. Output is sorted ascending for determinism.
//
// Both follow and alreadyRead are basename slices (no [[ ]] wrapping);
// the returned slice is also basenames. Unknown basenames in the input are
// ignored silently.
func Follow(graph Graph, follow, alreadyRead []string) []string {
	excluded := make(map[string]struct{}, len(follow)+len(alreadyRead))

	for _, name := range follow {
		excluded[name] = struct{}{}
	}

	for _, name := range alreadyRead {
		excluded[name] = struct{}{}
	}

	result := make(map[string]struct{})

	for _, source := range follow {
		for _, neighbor := range graph.UndirectedNeighbors(source) {
			if _, skip := excluded[neighbor]; skip {
				continue
			}

			result[neighbor] = struct{}{}
		}
	}

	out := make([]string, 0, len(result))

	for name := range result {
		out = append(out, name)
	}

	sort.Strings(out)

	return out
}
