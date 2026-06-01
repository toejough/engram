package cluster

// BestMatch returns the index of the candidate most similar (cosine) to centroid
// and that similarity, but only if the best meets threshold; otherwise it returns
// (-1, bestSeenSimilarity). Similarity = 1 - CosineDistance. Used to decide whether
// a fresh L2 cluster maps onto an existing L3 (update) or is a new topic (create).
func BestMatch(centroid []float32, candidates [][]float32, threshold float32) (int, float32) {
	bestIdx, bestSim := -1, float32(-1)

	for i, c := range candidates {
		sim := 1 - CosineDistance(centroid, c)
		if sim > bestSim {
			bestSim = sim

			if sim >= threshold {
				bestIdx = i
			}
		}
	}

	return bestIdx, bestSim
}
