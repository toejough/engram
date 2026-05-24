// Package embed wires the engram embedder, sidecar format, and staleness
// detection for the embed-on-write semantic-search pipeline.
package embed

import "math"

// Cosine returns the cosine similarity of a and b. Returns 0 when either
// vector has zero magnitude or when lengths differ — callers should treat
// that as "no signal" rather than a strong match.
func Cosine(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
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
		return 0
	}

	return float32(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
}
