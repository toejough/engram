// Package keyword provides keyword filtering utilities for memory pipelines.
package keyword

// FilterByDocFrequency removes keywords that appear in more than
// maxRatio of existing memory keyword sets. Keywords not present
// in any existing memory always pass (0% frequency).
func FilterByDocFrequency(
	candidates []string,
	existingKeywordSets [][]string,
	maxRatio float64,
) []string {
	if len(existingKeywordSets) == 0 {
		return candidates
	}

	// Build document frequency: how many memories contain each keyword.
	docFreq := make(map[string]int, len(candidates))

	for _, keywordSet := range existingKeywordSets {
		seen := make(map[string]bool, len(keywordSet))

		for _, kw := range keywordSet {
			if !seen[kw] {
				docFreq[kw]++
				seen[kw] = true
			}
		}
	}

	corpusSize := float64(len(existingKeywordSets))
	filtered := make([]string, 0, len(candidates))

	for _, candidate := range candidates {
		ratio := float64(docFreq[candidate]) / corpusSize
		if ratio <= maxRatio {
			filtered = append(filtered, candidate)
		}
	}

	return filtered
}
