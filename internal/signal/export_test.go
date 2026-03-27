package signal

import (
	"context"
	"io"

	"engram/internal/memory"
)

// Exported variables.
var (
	ExportLogMigrateStderrf = logMigrateStderrf
)

// ConsolidateClusterForTest exposes consolidateCluster for blackbox tests.
func (c *Consolidator) ConsolidateClusterForTest(
	ctx context.Context,
	cluster *ConfirmedCluster,
) (Action, error) {
	return c.consolidateCluster(ctx, cluster)
}

// ExportLogStderrf exposes logStderrf for coverage tests.
func (c *Consolidator) ExportLogStderrf(format string, args ...any) {
	c.logStderrf(format, args...)
}

// FindClusterForTest exposes findCluster for blackbox tests.
func (c *Consolidator) FindClusterForTest(
	ctx context.Context,
	mem *memory.MemoryRecord,
	exclude []string,
) *ConfirmedCluster {
	return c.findCluster(ctx, mem, exclude)
}

// ExportNewConsolidatorWithStderr creates a Consolidator with the stderr writer for testing.
func ExportNewConsolidatorWithStderr(w io.Writer) *Consolidator {
	return NewConsolidator(WithStderr(w))
}
