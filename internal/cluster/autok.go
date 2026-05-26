package cluster

// AutoKResult is the final clustering picked by AutoK.
//
// K == 0 means: no k in [minK, maxK] achieved silhouette >= threshold,
// or len(vectors) is too small to attempt clustering. Assignments and
// Centroids are nil in that case.
type AutoKResult struct {
	K           int
	Silhouette  float64
	Assignments []int
	Centroids   [][]float32
}

// AutoK loops k from minK to maxK (inclusive), running KMeans + Silhouette
// for each, and returns the result with the highest silhouette score.
//
// If max silhouette across the tried k values is below threshold, returns
// AutoKResult{K: 0} (no meaningful structure).
//
// Singleton clusters are absorbed into their nearest non-singleton cluster
// after kmeans converges — the spec's minimum cluster size is 2.
//
// maxK is automatically capped at len(vectors) (you can't have more
// clusters than points). If minK > maxK after that cap, returns K: 0.
//
// seed is forwarded to KMeans for deterministic results.
func AutoK(vectors [][]float32, minK, maxK int, threshold float64, seed uint64) (AutoKResult, error) {
	if len(vectors) < minK {
		return AutoKResult{}, nil
	}

	cappedMaxK := min(maxK, len(vectors))

	if minK > cappedMaxK {
		return AutoKResult{}, nil
	}

	best := AutoKResult{}

	for clusterCount := minK; clusterCount <= cappedMaxK; clusterCount++ {
		candidate, err := KMeans(vectors, clusterCount, seed)
		if err != nil {
			return AutoKResult{}, err
		}

		absorbed := absorbSingletons(vectors, candidate.Assignments, candidate.Centroids)

		effectiveK := countDistinct(absorbed)
		if effectiveK < MinClustersK {
			continue
		}

		// Re-pack cluster ids to be contiguous [0, effectiveK) so
		// silhouette sees a clean k.
		packed, packedK := repackAssignments(absorbed)

		score := Silhouette(vectors, packed, packedK)
		if score > best.Silhouette {
			best = AutoKResult{
				K:           packedK,
				Silhouette:  score,
				Assignments: packed,
				Centroids:   computeCentroidsForAssignments(vectors, packed, packedK),
			}
		}
	}

	if best.Silhouette < threshold {
		return AutoKResult{}, nil
	}

	return best, nil
}

// absorbSingletons reassigns members of singleton clusters (size == 1)
// to their nearest non-singleton cluster by cosine distance to centroid.
// Returns a new assignments slice; original is not modified.
func absorbSingletons(vectors [][]float32, assignments []int, centroids [][]float32) []int {
	counts := make(map[int]int, len(centroids))
	for _, clusterIdx := range assignments {
		counts[clusterIdx]++
	}

	out := make([]int, len(assignments))
	copy(out, assignments)

	for i, clusterIdx := range out {
		if counts[clusterIdx] != 1 {
			continue
		}

		nearest := nearestNonSingletonCluster(vectors[i], centroids, counts)
		if nearest == -1 {
			continue
		}

		out[i] = nearest
	}

	return out
}

// computeCentroidsForAssignments returns the mean vector per cluster
// over the given assignments. Used after singleton absorption to give
// the caller centroids consistent with the final assignments.
func computeCentroidsForAssignments(vectors [][]float32, assignments []int, clusterCount int) [][]float32 {
	if len(vectors) == 0 {
		return nil
	}

	dims := len(vectors[0])
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

	centroids := make([][]float32, clusterCount)

	for clusterIdx := range clusterCount {
		centroids[clusterIdx] = make([]float32, dims)
		if counts[clusterIdx] == 0 {
			continue
		}

		for dim := range dims {
			centroids[clusterIdx][dim] = float32(sums[clusterIdx][dim] / float64(counts[clusterIdx]))
		}
	}

	return centroids
}

// countDistinct returns the number of distinct cluster ids in assignments.
func countDistinct(assignments []int) int {
	seen := make(map[int]struct{})
	for _, clusterIdx := range assignments {
		seen[clusterIdx] = struct{}{}
	}

	return len(seen)
}

// nearestNonSingletonCluster returns the index of the cluster with the
// smallest cosine distance whose member count is > 1. Returns -1 if no
// non-singleton clusters exist.
func nearestNonSingletonCluster(vec []float32, centroids [][]float32, counts map[int]int) int {
	best := -1
	bestDist := float32(0)

	for clusterIdx, centroid := range centroids {
		if counts[clusterIdx] <= 1 {
			continue
		}

		dist := CosineDistance(vec, centroid)
		if best == -1 || dist < bestDist {
			best = clusterIdx
			bestDist = dist
		}
	}

	return best
}

// repackAssignments rewrites assignments so cluster ids are contiguous
// starting at 0. Returns the packed slice and the new k. Cluster id
// mapping is deterministic: earliest-appearance ordering.
func repackAssignments(assignments []int) ([]int, int) {
	mapping := make(map[int]int)

	next := 0
	out := make([]int, len(assignments))

	for i, clusterIdx := range assignments {
		id, ok := mapping[clusterIdx]
		if !ok {
			id = next
			mapping[clusterIdx] = id
			next++
		}

		out[i] = id
	}

	return out, next
}
