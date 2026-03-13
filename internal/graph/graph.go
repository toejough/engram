// Package graph implements link building and maintenance for memory entries (P3).
package graph

import (
	"regexp"
	"strings"

	"engram/internal/bm25"
	"engram/internal/registry"
)

// Builder constructs links between memory entries.
type Builder struct{}

// New creates a new link builder.
func New() *Builder {
	return &Builder{}
}

// BuildConceptOverlap computes Jaccard similarity links between a new entry and existing entries.
// Pairs with Jaccard >= conceptOverlapMinJaccard produce a concept_overlap link.
func (b *Builder) BuildConceptOverlap(
	entry registry.InstructionEntry,
	existing []registry.InstructionEntry,
) []registry.Link {
	var links []registry.Link

	newTokens := tokenize(entry.Title + " " + entry.Content)

	for _, other := range existing {
		if other.ID == entry.ID {
			continue // No self-links
		}

		otherTokens := tokenize(other.Title + " " + other.Content)
		jac := jaccard(newTokens, otherTokens)

		if jac >= conceptOverlapMinJaccard {
			links = append(links, registry.Link{
				Target: other.ID,
				Weight: jac,
				Basis:  "concept_overlap",
			})
		}
	}

	return links
}

// BuildContentSimilarity computes BM25 relevance links between a new entry and existing entries.
// Pairs with BM25 score >= contentSimilarityMinScore produce a content_similarity link with normalized weight.
func (b *Builder) BuildContentSimilarity(
	entry registry.InstructionEntry,
	existing []registry.InstructionEntry,
) []registry.Link {
	var links []registry.Link

	// Build BM25 scorer with existing entries as corpus
	docs := make([]bm25.Document, len(existing))
	for i, other := range existing {
		docs[i] = bm25.Document{
			ID:   other.ID,
			Text: other.Title + " " + other.Content,
		}
	}

	scorer := bm25.New()
	query := entry.Title + " " + entry.Content
	scored := scorer.Score(query, docs)

	for _, scoredDoc := range scored {
		if scoredDoc.ID == entry.ID {
			continue
		}

		if scoredDoc.Score >= contentSimilarityMinScore {
			// Normalize: weight = min(1.0, raw / contentSimilarityNormDivisor)
			weight := scoredDoc.Score / contentSimilarityNormDivisor
			if weight > 1.0 {
				weight = 1.0
			}

			links = append(links, registry.Link{
				Target: scoredDoc.ID,
				Weight: weight,
				Basis:  "content_similarity",
			})
		}
	}

	return links
}

// Prune removes links with weight < pruneWeightThreshold and CoSurfacingCount >= pruneMinCount.
// Returns a new slice with remaining links.
func Prune(links []registry.Link) []registry.Link {
	var result []registry.Link

	for _, link := range links {
		if link.Weight < pruneWeightThreshold && link.CoSurfacingCount >= pruneMinCount {
			continue // Prune this link
		}

		result = append(result, link)
	}

	return result
}

// UpdateCoSurfacing updates co_surfacing links for a pair of memory IDs.
// Increments weight (+coSurfacingIncrement, capped at 1.0) and CoSurfacingCount (+1).
// Returns the updated links slice.
func UpdateCoSurfacing(links []registry.Link, targetID string) []registry.Link {
	for i, link := range links {
		if link.Target == targetID && link.Basis == "co_surfacing" {
			links[i].Weight += coSurfacingIncrement
			if links[i].Weight > 1.0 {
				links[i].Weight = 1.0
			}

			links[i].CoSurfacingCount++

			return links
		}
	}

	// No existing link, create new one
	return append(links, registry.Link{
		Target:           targetID,
		Weight:           coSurfacingIncrement,
		Basis:            "co_surfacing",
		CoSurfacingCount: 1,
	})
}

// UpdateEvaluationCorrelation updates evaluation_correlation links for a pair.
// Increments weight (+evalCorrelationIncrement, capped at 1.0).
// Returns the updated links slice.
func UpdateEvaluationCorrelation(links []registry.Link, targetID string) []registry.Link {
	for i, link := range links {
		if link.Target == targetID && link.Basis == "evaluation_correlation" {
			links[i].Weight += evalCorrelationIncrement
			if links[i].Weight > 1.0 {
				links[i].Weight = 1.0
			}

			return links
		}
	}

	// No existing link, create new one
	return append(links, registry.Link{
		Target: targetID,
		Weight: evalCorrelationIncrement,
		Basis:  "evaluation_correlation",
	})
}

// unexported constants.
const (
	coSurfacingIncrement         = 0.1
	conceptOverlapMinJaccard     = 0.15
	contentSimilarityMinScore    = 0.05
	contentSimilarityNormDivisor = 5.0
	evalCorrelationIncrement     = 0.05
	pruneMinCount                = 10
	pruneWeightThreshold         = 0.1
)

// jaccard computes Jaccard similarity: |A∩B| / |A∪B|
func jaccard(a, b map[string]bool) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}

	intersection := 0

	for key := range a {
		if b[key] {
			intersection++
		}
	}

	// Union size
	union := make(map[string]bool)
	for key := range a {
		union[key] = true
	}

	for key := range b {
		union[key] = true
	}

	if len(union) == 0 {
		return 0
	}

	return float64(intersection) / float64(len(union))
}

// tokenize returns a set of lowercase word tokens from text.
func tokenize(text string) map[string]bool {
	// Split on non-word characters
	re := regexp.MustCompile(`\W+`)
	parts := re.Split(strings.ToLower(text), -1)

	set := make(map[string]bool)

	for _, part := range parts {
		if part != "" {
			set[part] = true
		}
	}

	return set
}
