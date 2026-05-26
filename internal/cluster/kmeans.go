package cluster

import (
	"errors"
	"fmt"
	"math/rand/v2"
)

// Exported constants.
const (
	// MaxKMeansIterations bounds the Lloyd-iteration loop. Per the spec,
	// non-convergence returns the best assignment found rather than failing.
	MaxKMeansIterations = 100
	// MinClustersK is the smallest valid k value for k-means; a 1-cluster
	// "clustering" is degenerate (trivial mean) and not useful for engram.
	MinClustersK = 2
)

// Exported variables.
var (
	ErrKMeansEmptyInput = errors.New("kmeans: empty input")
	ErrKMeansKTooLarge  = errors.New("kmeans: k must be <= len(vectors)")
	ErrKMeansKTooSmall  = errors.New("kmeans: k must be >= 2")
)

// KMeansResult bundles the per-vector cluster assignment with the
// final centroids, in cluster-index order.
type KMeansResult struct {
	Assignments []int       // assignments[i] = cluster id (0..k-1) for vectors[i]
	Centroids   [][]float32 // centroids[c] = centroid vector for cluster c
}

// KMeans runs Lloyd's algorithm with cosine distance for `clusterCount`
// clusters over the given vectors. The PRNG is seeded from `seed` for
// determinism; same seed + same vectors yields the same assignments
// and centroids.
//
// Errors:
//   - ErrKMeansEmptyInput if len(vectors) == 0.
//   - ErrKMeansKTooSmall if clusterCount < MinClustersK.
//   - ErrKMeansKTooLarge if clusterCount > len(vectors).
//
// Non-convergence (no change in assignments before MaxKMeansIterations)
// is not an error: returns the best assignment found.
func KMeans(vectors [][]float32, clusterCount int, seed uint64) (KMeansResult, error) {
	if len(vectors) == 0 {
		return KMeansResult{}, ErrKMeansEmptyInput
	}

	if clusterCount < MinClustersK {
		return KMeansResult{}, fmt.Errorf("%w (got k=%d)", ErrKMeansKTooSmall, clusterCount)
	}

	if clusterCount > len(vectors) {
		return KMeansResult{}, fmt.Errorf("%w (k=%d, n=%d)", ErrKMeansKTooLarge, clusterCount, len(vectors))
	}

	dims := len(vectors[0])

	prng := rand.New(rand.NewPCG(seed, seed^kmeansPCGSalt)) //nolint:gosec // determinism, not security

	centroids := initialCentroids(vectors, clusterCount, prng)
	assignments := make([]int, len(vectors))

	for range MaxKMeansIterations {
		changed := assignAll(vectors, centroids, assignments)
		recomputeCentroids(vectors, assignments, clusterCount, dims, centroids)

		if !changed {
			break
		}
	}

	return KMeansResult{Assignments: assignments, Centroids: centroids}, nil
}

// unexported constants.
const (
	// kmeansPCGSalt diversifies the second 64 bits of the PCG seed so
	// the same seed in both halves doesn't produce a degenerate stream.
	kmeansPCGSalt = uint64(0xA5A5_A5A5_5A5A_5A5A)
)

// assignAll updates assignments to nearest centroid by cosine distance.
// Returns true if any assignment changed.
func assignAll(vectors [][]float32, centroids [][]float32, assignments []int) bool {
	changed := false

	for i, vec := range vectors {
		best := nearestCentroid(vec, centroids)
		if assignments[i] != best {
			assignments[i] = best
			changed = true
		}
	}

	return changed
}

// cloneVec returns a fresh copy of vec so that mutation of centroids
// doesn't aliase the source vectors.
func cloneVec(vec []float32) []float32 {
	out := make([]float32, len(vec))
	copy(out, vec)

	return out
}

// initialCentroids picks clusterCount starting centroids via k-means++ seeding:
// the first centroid is chosen uniformly at random, each subsequent
// centroid is chosen with probability proportional to the squared
// distance from the nearest already-chosen centroid. Pure-Go,
// deterministic given the seeded PRNG.
func initialCentroids(vectors [][]float32, clusterCount int, prng *rand.Rand) [][]float32 {
	centroids := make([][]float32, 0, clusterCount)

	firstIdx := prng.IntN(len(vectors))
	centroids = append(centroids, cloneVec(vectors[firstIdx]))

	dists := make([]float32, len(vectors))

	for clusterIdx := 1; clusterIdx < clusterCount; clusterIdx++ {
		var total float64

		for i, vec := range vectors {
			minDist := minDistanceToCentroids(vec, centroids)
			dists[i] = minDist * minDist
			total += float64(dists[i])
		}

		if total == 0 {
			// All remaining points coincide with chosen centroids; pick
			// any unique-vector fallback.
			centroids = append(centroids, cloneVec(vectors[(firstIdx+clusterIdx)%len(vectors)]))

			continue
		}

		target := prng.Float64() * total
		cumulative := float64(0)

		picked := len(vectors) - 1

		for i, dist := range dists {
			cumulative += float64(dist)
			if cumulative >= target {
				picked = i

				break
			}
		}

		centroids = append(centroids, cloneVec(vectors[picked]))
	}

	return centroids
}

// minDistanceToCentroids returns the minimum cosine distance from vec
// to any centroid in the list. Assumes len(centroids) >= 1.
func minDistanceToCentroids(vec []float32, centroids [][]float32) float32 {
	best := CosineDistance(vec, centroids[0])

	for i := 1; i < len(centroids); i++ {
		dist := CosineDistance(vec, centroids[i])
		if dist < best {
			best = dist
		}
	}

	return best
}

// nearestCentroid returns the index of the centroid with smallest cosine
// distance to vec. Ties go to the lower index.
func nearestCentroid(vec []float32, centroids [][]float32) int {
	best := 0
	bestDist := CosineDistance(vec, centroids[0])

	for clusterIdx := 1; clusterIdx < len(centroids); clusterIdx++ {
		dist := CosineDistance(vec, centroids[clusterIdx])
		if dist < bestDist {
			best = clusterIdx
			bestDist = dist
		}
	}

	return best
}

// recomputeCentroids replaces each centroid with the mean of its
// assigned members. Empty clusters keep their previous centroid (rare
// once Lloyd stabilizes; defensive).
func recomputeCentroids(
	vectors [][]float32, assignments []int, clusterCount, dims int, centroids [][]float32,
) {
	sums := make([][]float64, clusterCount)
	counts := make([]int, clusterCount)

	for clusterIdx := range sums {
		sums[clusterIdx] = make([]float64, dims)
	}

	for i, vec := range vectors {
		assigned := assignments[i]
		counts[assigned]++

		for dim := range dims {
			sums[assigned][dim] += float64(vec[dim])
		}
	}

	for clusterIdx := range clusterCount {
		if counts[clusterIdx] == 0 {
			continue
		}

		next := make([]float32, dims)
		for dim := range dims {
			next[dim] = float32(sums[clusterIdx][dim] / float64(counts[clusterIdx]))
		}

		centroids[clusterIdx] = next
	}
}
