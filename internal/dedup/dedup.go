// Package dedup filters candidate learnings to remove near-duplicates of existing memories.
package dedup

import (
	"strings"

	"engram/internal/memory"
)

// KeywordDeduplicator filters candidates by keyword overlap against existing memories.
type KeywordDeduplicator struct{}

// New returns a new KeywordDeduplicator.
func New() *KeywordDeduplicator {
	return &KeywordDeduplicator{}
}

// Filter returns only candidates whose keyword overlap with any existing memory is ≤50%.
func (d *KeywordDeduplicator) Filter(
	candidates []memory.CandidateLearning,
	existing []*memory.Stored,
) []memory.CandidateLearning {
	result := make([]memory.CandidateLearning, 0, len(candidates))
	for _, candidate := range candidates {
		if !isDuplicate(candidate, existing) {
			result = append(result, candidate)
		}
	}

	return result
}

// unexported constants.
const (
	overlapThreshold = 0.5
)

func intersectionSize(a, b map[string]struct{}) int {
	count := 0

	for key := range a {
		if _, ok := b[key]; ok {
			count++
		}
	}

	return count
}

func isDuplicate(candidate memory.CandidateLearning, existing []*memory.Stored) bool {
	candidateKeys := keywordSet(candidate.Keywords)
	if len(candidateKeys) == 0 {
		return false
	}

	for _, stored := range existing {
		if stored == nil {
			continue
		}

		storedKeys := keywordSet(stored.Keywords)

		overlap := intersectionSize(candidateKeys, storedKeys)
		if float64(overlap)/float64(len(candidateKeys)) > overlapThreshold {
			return true
		}
	}

	return false
}

func keywordSet(keywords []string) map[string]struct{} {
	set := make(map[string]struct{}, len(keywords))
	for _, keyword := range keywords {
		set[strings.ToLower(keyword)] = struct{}{}
	}

	return set
}
