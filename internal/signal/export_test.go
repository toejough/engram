package signal

import (
	"context"

	"engram/internal/memory"
)

// FindClusterForTest exposes findCluster for blackbox tests.
func (c *Consolidator) FindClusterForTest(
	ctx context.Context,
	mem *memory.MemoryRecord,
	exclude []string,
) *ConfirmedCluster {
	return c.findCluster(ctx, mem, exclude)
}
