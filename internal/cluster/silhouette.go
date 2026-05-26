package cluster

// PointSilhouette returns the silhouette score for a single point given
// pre-computed cluster member lists. Used by callers that need per-point
// or per-cluster aggregations rather than the global mean.
//
// own is the cluster id of the point; selfIdx is its index in vectors.
// Returns 0 for singleton clusters or when no other non-empty cluster
// exists.
func PointSilhouette(
	vec []float32,
	vectors [][]float32,
	members [][]int,
	own, selfIdx int,
) float64 {
	if own < 0 || own >= len(members) {
		return 0
	}

	if len(members[own]) <= 1 {
		return 0
	}

	intraDist := meanIntraDistance(vec, vectors, members[own], selfIdx)
	interDist := nearestOtherClusterMeanDistance(vec, vectors, members, own)

	denom := max64(intraDist, interDist)
	if denom == 0 {
		return 0
	}

	return float64(interDist-intraDist) / float64(denom)
}

// Silhouette returns the mean silhouette score across all points,
// using cosine distance and the provided cluster assignments. A
// silhouette value lies in [-1, 1]; higher = better separation.
//
// The score for a point in a cluster of size 1 is defined as 0
// (silhouette is undefined for singletons; engram absorbs them
// into the nearest cluster before scoring, but the formula's
// safety net is loud here).
//
// Returns 0 if vectors is empty.
func Silhouette(vectors [][]float32, assignments []int, clusterCount int) float64 {
	if len(vectors) == 0 {
		return 0
	}

	// Pre-build cluster index → member positions to avoid quadratic re-scans.
	members := groupByCluster(assignments, clusterCount)

	total := float64(0)
	counted := 0

	for i, vec := range vectors {
		own := assignments[i]

		if len(members[own]) <= 1 {
			// Singleton: contributes 0 per the silhouette convention.
			counted++

			continue
		}

		intraDist := meanIntraDistance(vec, vectors, members[own], i)
		interDist := nearestOtherClusterMeanDistance(vec, vectors, members, own)

		denom := max64(intraDist, interDist)
		if denom == 0 {
			counted++

			continue
		}

		total += float64(interDist-intraDist) / float64(denom)
		counted++
	}

	if counted == 0 {
		return 0
	}

	return total / float64(counted)
}

// groupByCluster returns members[c] = sorted slice of point indices in cluster c.
func groupByCluster(assignments []int, clusterCount int) [][]int {
	members := make([][]int, clusterCount)
	for clusterIdx := range members {
		members[clusterIdx] = make([]int, 0)
	}

	for idx, clusterIdx := range assignments {
		if clusterIdx < 0 || clusterIdx >= clusterCount {
			continue
		}

		members[clusterIdx] = append(members[clusterIdx], idx)
	}

	return members
}

// max64 returns the larger of a and b in float32 precision.
func max64(a, b float32) float32 {
	if a > b {
		return a
	}

	return b
}

// meanGroupDistance returns the mean cosine distance from vec to every
// point in group (no self-exclusion: callers pass groups that don't
// contain vec).
func meanGroupDistance(vec []float32, vectors [][]float32, group []int) float32 {
	var total float32

	for _, idx := range group {
		total += CosineDistance(vec, vectors[idx])
	}

	return total / float32(len(group))
}

// meanIntraDistance computes a(i): the mean cosine distance from vec
// to all OTHER points in its own cluster (excludes the point itself,
// at position selfIdx).
func meanIntraDistance(vec []float32, vectors [][]float32, ownMembers []int, selfIdx int) float32 {
	if len(ownMembers) <= 1 {
		return 0
	}

	var total float32

	count := 0

	for _, idx := range ownMembers {
		if idx == selfIdx {
			continue
		}

		total += CosineDistance(vec, vectors[idx])
		count++
	}

	if count == 0 {
		return 0
	}

	return total / float32(count)
}

// nearestOtherClusterMeanDistance computes b(i): the minimum, across
// all OTHER clusters, of the mean cosine distance from vec to that
// cluster's members. Returns 0 if there's no other non-empty cluster
// (caller treats this with the max(a, b) guard).
func nearestOtherClusterMeanDistance(
	vec []float32,
	vectors [][]float32,
	members [][]int,
	own int,
) float32 {
	best := float32(-1)

	for clusterIdx, group := range members {
		if clusterIdx == own || len(group) == 0 {
			continue
		}

		mean := meanGroupDistance(vec, vectors, group)

		if best < 0 || mean < best {
			best = mean
		}
	}

	if best < 0 {
		return 0
	}

	return best
}
