package cluster_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cluster"
)

func TestBestMatch_BestCandidateMeetsThreshold(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	centroid := []float32{1, 0, 0}
	candidates := [][]float32{
		{0, 1, 0},
		{0.95, 0.05, 0},
		{1, 0, 0},
	}

	const threshold = float32(0.9)

	idx, sim := cluster.BestMatch(centroid, candidates, threshold)

	g.Expect(idx).To(Equal(2))
	g.Expect(sim).To(BeNumerically(">=", threshold))
}

func TestBestMatch_NoCandidateMeetsThreshold(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	centroid := []float32{1, 0, 0}
	candidates := [][]float32{
		{0, 1, 0},
		{0, 0, 1},
	}
	// Threshold higher than any similarity achievable with orthogonal vectors.
	const threshold = float32(0.99)

	idx, _ := cluster.BestMatch(centroid, candidates, threshold)

	g.Expect(idx).To(Equal(-1))
}

// TestBestMatch_PropertyBestIdxHasmaxSim asserts: when BestMatch returns
// idx >= 0, the similarity at that index equals the maximum similarity over
// all candidates AND meets the threshold.
func TestBestMatch_PropertyBestIdxHasMaxSim(t *testing.T) {
	t.Parallel()

	const dims = 8

	const floatTolerance = float32(1e-5)

	rapid.Check(t, func(t *rapid.T) {
		centroid := rapid.SliceOfN(rapid.Float32Range(-1, 1), dims, dims).Draw(t, "centroid")
		numCandidates := rapid.IntRange(1, 10).Draw(t, "numCandidates")
		candidates := make([][]float32, numCandidates)

		for i := range candidates {
			candidates[i] = rapid.SliceOfN(rapid.Float32Range(-1, 1), dims, dims).Draw(t, "candidate")
		}

		threshold := rapid.Float32Range(0, 1).Draw(t, "threshold")

		idx, sim := cluster.BestMatch(centroid, candidates, threshold)

		if idx < 0 {
			// Nothing met the threshold; result is valid — no further assertions.
			return
		}

		// idx >= 0: sim must meet threshold.
		if sim < threshold-floatTolerance {
			t.Fatalf("BestMatch returned idx=%d with sim=%v below threshold=%v", idx, sim, threshold)
		}

		// sim must be the maximum similarity across all candidates.
		for i, candidate := range candidates {
			candidateSim := 1 - cluster.CosineDistance(centroid, candidate)
			if candidateSim > sim+floatTolerance {
				t.Fatalf(
					"candidate %d has sim=%v > best sim=%v; BestMatch did not return the best candidate",
					i, candidateSim, sim,
				)
			}
		}
	})
}
