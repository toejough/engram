package vaultgraph

import (
	"sort"

	"github.com/toejough/engram/internal/luhmann"
)

// SelectStartingPoints picks the starting points for one component:
//   - If the component contains one or more MOCs, returns every MOC (sorted).
//   - Otherwise returns the node(s) with the highest in-degree, tie-broken by
//     earliest Luhmann ID (luhmann.Less). If two or more tied nodes lack
//     Luhmann IDs (or the tie cannot be broken otherwise), returns all of them.
//
// Empty input yields nil. Result order is deterministic and stable for testing.
func SelectStartingPoints(component []string, g Graph) []string {
	if len(component) == 0 {
		return nil
	}

	mocs := mocsIn(component, g)
	if len(mocs) > 0 {
		sortBasenames(mocs)

		return mocs
	}

	return highestInDegreeWinners(component, g)
}

func highestInDegreeWinners(component []string, g Graph) []string {
	bestDeg := -1

	for _, name := range component {
		if deg := g.InDegree(name); deg > bestDeg {
			bestDeg = deg
		}
	}

	tied := make([]string, 0)

	for _, name := range component {
		if g.InDegree(name) == bestDeg {
			tied = append(tied, name)
		}
	}

	if len(tied) == 1 {
		return tied
	}

	return luhmannTieBreak(tied, g)
}

// luhmannTieBreak finds the earliest-Luhmann tied node(s). Nodes without
// Luhmann IDs are eligible: if every tied node lacks an ID, they all win.
// If some have IDs and some don't, only ID-bearing nodes can win (the
// earliest one) — IDless nodes are demoted.
func luhmannTieBreak(tied []string, g Graph) []string {
	withID := make([]string, 0, len(tied))
	withoutID := make([]string, 0, len(tied))

	for _, name := range tied {
		if g.Notes[name].LuhmannID != "" {
			withID = append(withID, name)
		} else {
			withoutID = append(withoutID, name)
		}
	}

	if len(withID) == 0 {
		sortBasenames(withoutID)

		return withoutID
	}

	earliestID := g.Notes[withID[0]].LuhmannID
	winners := make([]string, 0, len(withID))
	winners = append(winners, withID[0])

	for _, name := range withID[1:] {
		id := g.Notes[name].LuhmannID

		switch {
		case luhmann.Less(id, earliestID):
			earliestID = id
			winners = winners[:0]
			winners = append(winners, name)
		case id == earliestID:
			winners = append(winners, name)
		}
	}

	sortBasenames(winners)

	return winners
}

func mocsIn(component []string, g Graph) []string {
	out := make([]string, 0)

	for _, name := range component {
		if g.Notes[name].IsMOC {
			out = append(out, name)
		}
	}

	return out
}

func sortBasenames(names []string) {
	sort.Strings(names)
}
