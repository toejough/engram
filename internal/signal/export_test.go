package signal

import (
	"context"

	"engram/internal/memory"
)

// ConsolidateClusterForTest exposes consolidateCluster for blackbox tests.
func (c *Consolidator) ConsolidateClusterForTest(
	ctx context.Context,
	cluster *ConfirmedCluster,
) (Action, error) {
	return c.consolidateCluster(ctx, cluster)
}

// FindClusterForTest exposes findCluster for blackbox tests.
func (c *Consolidator) FindClusterForTest(
	ctx context.Context,
	mem *memory.MemoryRecord,
	exclude []string,
) *ConfirmedCluster {
	return c.findCluster(ctx, mem, exclude)
}
