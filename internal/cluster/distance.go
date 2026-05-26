// Package cluster implements k-means clustering with silhouette-based
// automatic k selection, intended for the engram query subgraph
// clustering step. Pure Go, no third-party dependencies; the algorithms
// are small enough (~50-100 LOC each) to maintain inline.
package cluster

import "math"

// CosineDistance returns 1 - cosine_similarity(a, b), clamped to [0, 2].
// Returns 1 (max) for zero-magnitude or mismatched-length inputs — those
// have no direction so the safest default is "no signal" treated as max
// distance, mirroring engram's embedding contract.
func CosineDistance(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 1
	}

	var dot, normA, normB float64

	for i := range a {
		af := float64(a[i])
		bf := float64(b[i])
		dot += af * bf
		normA += af * af
		normB += bf * bf
	}

	if normA == 0 || normB == 0 {
		return 1
	}

	similarity := dot / (math.Sqrt(normA) * math.Sqrt(normB))

	return float32(1 - similarity)
}
