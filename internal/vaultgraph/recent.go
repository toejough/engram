package vaultgraph

import (
	"regexp"
	"sort"
)

// Recent returns up to limit basenames sorted by their YYYY-MM-DD filename
// date prefix in descending order (newest first). Basenames lacking a valid
// date prefix are skipped — MEMORY and index files do not surface here.
// Ties on date break by basename ascending for determinism.
func Recent(notes []Note, limit int) []string {
	type dated struct {
		basename string
		date     string
	}

	candidates := make([]dated, 0, len(notes))

	for _, n := range notes {
		match := datePrefixRE.FindStringSubmatch(n.Basename)
		if match == nil {
			continue
		}

		candidates = append(candidates, dated{basename: n.Basename, date: match[1]})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].date == candidates[j].date {
			return candidates[i].basename < candidates[j].basename
		}

		return candidates[i].date > candidates[j].date
	})

	if limit > 0 && len(candidates) > limit {
		candidates = candidates[:limit]
	}

	out := make([]string, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, c.basename)
	}

	return out
}

// unexported variables.
var (
	// datePrefixRE matches a leading "<luhmann>.YYYY-MM-DD." prefix and captures
	// the date. The basename format is "<luhmann-id>.YYYY-MM-DD.<slug>" (no .md suffix —
	// basenames are stripped of their extension by ParseBasename).
	datePrefixRE = regexp.MustCompile(`^[^.]+\.(\d{4}-\d{2}-\d{2})\.`)
)
