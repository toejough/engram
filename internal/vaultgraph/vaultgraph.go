package vaultgraph

import (
	"fmt"
	"sort"

	"engram/internal/luhmann"
)

// StartingPoints returns the deduped, globally Luhmann-sorted set of vault
// graph entry points: every MOC plus the per-component winner of every
// MOC-less connected component. Output is a slice of note basenames (no
// `[[ ]]` wrapping); callers format for display.
//
// Sort order:
//
//   - Basenames whose leading segment is a valid Luhmann ID sort by that ID
//     in tree order (`1 < 1a < 1a1 < 1b < 2`).
//   - Basenames without a Luhmann ID sort lexicographically *after* every
//     Luhmann-ID-bearing basename.
//
// The output is deterministic: same vault state → identical output across runs.
func StartingPoints(fs VaultFS, vaultPath string) ([]string, error) {
	notes, err := ScanVault(fs, vaultPath)
	if err != nil {
		return nil, fmt.Errorf("scanning vault: %w", err)
	}

	graph := BuildGraph(notes)
	comps := Components(graph)

	seen := make(map[string]struct{}, len(notes))

	out := make([]string, 0, len(notes))

	for _, comp := range comps {
		for _, name := range SelectStartingPoints(comp, graph) {
			if _, dup := seen[name]; dup {
				continue
			}

			seen[name] = struct{}{}

			out = append(out, name)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return basenameLess(out[i], out[j])
	})

	return out, nil
}

// basenameLess sorts basenames by leading Luhmann ID using tree order. Basenames
// without a valid Luhmann ID sort after every basename that has one; among
// IDless basenames, fall back to lexical order.
func basenameLess(a, b string) bool {
	aID, aOK := LuhmannFromBasename(a)
	bID, bOK := LuhmannFromBasename(b)

	switch {
	case aOK && bOK:
		if aID == bID {
			return a < b
		}

		return luhmann.Less(aID, bID)
	case aOK && !bOK:
		return true
	case !aOK && bOK:
		return false
	default:
		return a < b
	}
}
