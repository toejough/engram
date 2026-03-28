// Package dedup filters candidate learnings to remove near-duplicates of existing memories.
// P5c: Merge-on-Write — candidates with >50% overlap are returned as merge pairs.
package dedup

import (
	"engram/internal/keyword"
	"engram/internal/memory"
)

// ClassifyResult holds the results of dedup classification (UC-33).
type ClassifyResult struct {
	Surviving  []memory.CandidateLearning // candidates to create as new memories
	MergePairs []MergePair                // candidate-existing pairs to merge
}

// KeywordDeduplicator filters candidates by keyword overlap against existing memories.
type KeywordDeduplicator struct{}

// New returns a new KeywordDeduplicator.
func New() *KeywordDeduplicator {
	return &KeywordDeduplicator{}
}

// Classify partitions candidates into survivors (new memories) and merge pairs (UC-33).
// Candidates with >50% keyword overlap with any existing memory are returned as merge pairs.
// Others are returned as survivors (to be created as new memories).
func (d *KeywordDeduplicator) Classify(
	candidates []memory.CandidateLearning,
	existing []*memory.Stored,
) ClassifyResult {
	result := ClassifyResult{
		Surviving:  make([]memory.CandidateLearning, 0, len(candidates)),
		MergePairs: make([]MergePair, 0),
	}

	for _, candidate := range candidates {
		match := findMergeMatch(candidate, existing)
		if match != nil {
			result.MergePairs = append(result.MergePairs, MergePair{
				Candidate: candidate,
				Existing:  match,
			})
		} else {
			result.Surviving = append(result.Surviving, candidate)
		}
	}

	return result
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

// MergePair pairs a candidate learning with an existing memory for merge (UC-33).
// (Capitalized as a public type for use across packages.)
type MergePair struct {
	Candidate memory.CandidateLearning
	Existing  *memory.Stored
}

// unexported constants.
const (
	overlapThreshold = 0.5
)

// findMergeMatch returns the first existing memory with >50% keyword overlap with the candidate.
// Returns nil if no match found.
func findMergeMatch(candidate memory.CandidateLearning, existing []*memory.Stored) *memory.Stored {
	candidateKeys := keywordSet(candidate.Keywords)
	if len(candidateKeys) == 0 {
		return nil // empty keywords never merge
	}

	for _, stored := range existing {
		if stored == nil {
			continue
		}

		storedKeys := keywordSet(stored.Keywords)
		overlap := intersectionSize(candidateKeys, storedKeys)

		if float64(overlap)/float64(len(candidateKeys)) > overlapThreshold {
			return stored
		}
	}

	return nil
}

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
	for _, kw := range keywords {
		set[keyword.Normalize(kw)] = struct{}{}
	}

	return set
}
