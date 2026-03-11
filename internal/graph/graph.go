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
// Pairs with Jaccard >= 0.15 produce a concept_overlap link.
func (b *Builder) BuildConceptOverlap(
	entry registry.InstructionEntry,
	existing []registry.InstructionEntry,
) []registry.Link {
	var links []registry.Link

	newTokens := tokenize(entry.Title + " " + entry.Content)

	for _, candidate := range existing {
		if candidate.ID == entry.ID {
			continue // No self-links
		}

		exTokens := tokenize(candidate.Title + " " + candidate.Content)
		jac := jaccard(newTokens, exTokens)

		if jac >= conceptOverlapThreshold {
			links = append(links, registry.Link{
				Target: candidate.ID,
				Weight: jac,
				Basis:  "concept_overlap",
			})
		}
	}

	return links
}

// BuildContentSimilarity computes BM25 relevance links between a new entry and existing entries.
// Pairs with BM25 score >= 0.05 (raw) produce a content_similarity link with normalized weight.
func (b *Builder) BuildContentSimilarity(
	entry registry.InstructionEntry,
	existing []registry.InstructionEntry,
) []registry.Link {
	var links []registry.Link

	// Build BM25 scorer with existing entries as corpus
	docs := make([]bm25.Document, len(existing))

	for i, candidate := range existing {
		docs[i] = bm25.Document{
			ID:   candidate.ID,
			Text: candidate.Title + " " + candidate.Content,
		}
	}

	scorer := bm25.New()
	query := entry.Title + " " + entry.Content
	scored := scorer.Score(query, docs)

	for _, result := range scored {
		if result.ID == entry.ID {
			continue
		}

		if result.Score >= contentSimilarityThreshold {
			// Normalize: weight = min(1.0, raw / 5.0)
			weight := result.Score / contentSimilarityNormDiv

			if weight > maxLinkWeight {
				weight = maxLinkWeight
			}

			links = append(links, registry.Link{
				Target: result.ID,
				Weight: weight,
				Basis:  "content_similarity",
			})
		}
	}

	return links
}

// Prune removes links with weight < 0.1 and CoSurfacingCount >= 10.
// Returns a new slice with remaining links.
func Prune(links []registry.Link) []registry.Link {
	var result []registry.Link

	for _, link := range links {
		if link.Weight < pruneWeightThreshold && link.CoSurfacingCount >= pruneCountThreshold {
			continue // Prune this link
		}

		result = append(result, link)
	}

	return result
}

// UpdateCoSurfacing updates co_surfacing links for a pair of memory IDs.
// Increments weight (+0.1, capped at 1.0) and CoSurfacingCount (+1).
// Returns the updated links slice.
func UpdateCoSurfacing(links []registry.Link, targetID string) []registry.Link {
	for i, link := range links {
		if link.Target == targetID && link.Basis == "co_surfacing" {
			links[i].Weight += coSurfacingIncrement

			if links[i].Weight > maxLinkWeight {
				links[i].Weight = maxLinkWeight
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
// Increments weight (+0.05, capped at 1.0).
// Returns the updated links slice.
func UpdateEvaluationCorrelation(links []registry.Link, targetID string) []registry.Link {
	for i, link := range links {
		if link.Target == targetID && link.Basis == "evaluation_correlation" {
			links[i].Weight += evalCorrIncrement

			if links[i].Weight > maxLinkWeight {
				links[i].Weight = maxLinkWeight
			}

			return links
		}
	}

	// No existing link, create new one
	return append(links, registry.Link{
		Target: targetID,
		Weight: evalCorrIncrement,
		Basis:  "evaluation_correlation",
	})
}

// unexported constants.
const (
	coSurfacingIncrement       = 0.1
	conceptOverlapThreshold    = 0.15
	contentSimilarityNormDiv   = 5.0
	contentSimilarityThreshold = 0.05
	evalCorrIncrement          = 0.05
	maxLinkWeight              = 1.0
	pruneCountThreshold        = 10
	pruneWeightThreshold       = 0.1
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
