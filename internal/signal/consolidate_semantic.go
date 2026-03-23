package signal

import (
	"context"
	"sort"

	"engram/internal/memory"
)

// findCluster attempts to find a semantic cluster for the given memory.
// Returns nil if no cluster is found or if the pipeline is unavailable.
func (c *Consolidator) findCluster(
	ctx context.Context,
	mem *memory.MemoryRecord,
	exclude []string,
) *ConfirmedCluster {
	if c.scorer == nil || c.confirmer == nil {
		return nil
	}

	candidates, err := c.scorer.FindSimilar(ctx, mem, exclude)
	if err != nil || len(candidates) < minSemanticClusterSize-1 {
		return nil
	}

	clusters, err := c.confirmer.ConfirmClusters(ctx, mem, candidates)
	if err != nil || len(clusters) == 0 {
		return nil
	}

	// Sort smallest first — protect fragile clusters (per spec).
	sort.Slice(clusters, func(i, j int) bool {
		return len(clusters[i].Members) < len(clusters[j].Members)
	})

	for idx := range clusters {
		if len(clusters[idx].Members) >= minSemanticClusterSize {
			return &clusters[idx]
		}
	}

	return nil
}

// unexported constants.
const (
	minSemanticClusterSize = 3
)
