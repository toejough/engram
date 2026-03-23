package signal

import (
	"context"
	"fmt"
	"io"

	"engram/internal/memory"
)

// GeneralizabilityScorer scores a batch of memories for generalizability (1-5).
type GeneralizabilityScorer interface {
	ScoreBatch(ctx context.Context, memories []*memory.MemoryRecord) ([]int, error)
}

// MigrationResult holds stats from a batch consolidation run.
type MigrationResult struct {
	ScoredCount        int
	ClusterCount       int
	ConsolidatedCount  int
	UnclusterableCount int
}

// MigrationRunner handles the migrate-scores subcommand pipeline.
type MigrationRunner struct {
	lister       MemoryRecordLister
	scorer       GeneralizabilityScorer
	writer       MigrationWriter
	stderr       io.Writer
	consolidator *Consolidator
}

// NewMigrationRunner creates a MigrationRunner with the given dependencies.
func NewMigrationRunner(
	lister MemoryRecordLister,
	scorer GeneralizabilityScorer,
	writer MigrationWriter,
	stderr io.Writer,
) *MigrationRunner {
	return &MigrationRunner{
		lister: lister,
		scorer: scorer,
		writer: writer,
		stderr: stderr,
	}
}

// ConsolidateBatch scans the full corpus for semantic clusters.
// In dry-run mode, reports what would happen. In apply mode, executes consolidation.
//
//nolint:cyclop,funlen // scan + filter + cluster + dry-run/apply pipeline: inherent branching
func (r *MigrationRunner) ConsolidateBatch(
	ctx context.Context,
	dryRun bool,
) (*MigrationResult, error) {
	if r.consolidator == nil {
		return &MigrationResult{}, nil
	}

	records, err := r.lister.ListAllRecords(ctx)
	if err != nil {
		return nil, fmt.Errorf("migration: listing records: %w", err)
	}

	// Track assigned titles to prevent overlap between clusters.
	assigned := make(map[string]struct{})

	var clusters []ConfirmedCluster

	for _, rec := range records {
		// Skip already-absorbed memories.
		if len(rec.Absorbed) > 0 {
			continue
		}

		// Skip already-assigned memories.
		if _, ok := assigned[rec.Title]; ok {
			continue
		}

		// Build exclude list from assigned titles.
		excludeList := make([]string, 0, len(assigned))
		for slug := range assigned {
			excludeList = append(excludeList, slug)
		}

		cluster := r.consolidator.findCluster(ctx, rec, excludeList)
		if cluster == nil {
			continue
		}

		// Mark all members as assigned.
		for _, mem := range cluster.Members {
			assigned[mem.Title] = struct{}{}
		}

		clusters = append(clusters, *cluster)
	}

	result := &MigrationResult{
		ClusterCount: len(clusters),
	}

	if dryRun {
		for _, cluster := range clusters {
			logMigrateStderrf(r.stderr,
				"[engram] Cluster: %s (%d members)\n",
				cluster.Principle, len(cluster.Members))
		}

		return result, nil
	}

	// Apply consolidation.
	for _, cluster := range clusters {
		action, consErr := r.consolidator.consolidateCluster(ctx, &cluster)
		if consErr != nil {
			logMigrateStderrf(r.stderr,
				"[engram] consolidation error: %v\n", consErr)

			continue
		}

		if action.Type == Consolidated {
			result.ConsolidatedCount++

			if r.writer != nil {
				writeErr := r.writer.WriteRecord(action.ConsolidatedMem)
				if writeErr != nil {
					logMigrateStderrf(r.stderr,
						"[engram] write failed: %v\n", writeErr)
				}
			}
		}
	}

	return result, nil
}

// ScoreUnscored loads all memories, filters those with generalizability==0,
// batch-scores them via LLM, and writes scores back.
func (r *MigrationRunner) ScoreUnscored(ctx context.Context) (int, error) {
	records, err := r.lister.ListAllRecords(ctx)
	if err != nil {
		return 0, fmt.Errorf("migration: listing records: %w", err)
	}

	unscored := make([]*memory.MemoryRecord, 0, len(records))

	for _, rec := range records {
		if rec.Generalizability == 0 {
			unscored = append(unscored, rec)
		}
	}

	if len(unscored) == 0 {
		return 0, nil
	}

	scores, err := r.scorer.ScoreBatch(ctx, unscored)
	if err != nil {
		return 0, fmt.Errorf("migration: scoring batch: %w", err)
	}

	scored := 0

	for idx, rec := range unscored {
		if idx < len(scores) {
			rec.Generalizability = scores[idx]

			writeErr := r.writer.WriteRecord(rec)
			if writeErr != nil {
				logMigrateStderrf(r.stderr,
					"[engram] write failed for %q: %v\n",
					rec.Title, writeErr)

				continue
			}

			scored++
		}
	}

	return scored, nil
}

// SetConsolidator sets the consolidator for batch consolidation.
func (r *MigrationRunner) SetConsolidator(c *Consolidator) {
	r.consolidator = c
}

// MigrationWriter writes a MemoryRecord back to its source path.
type MigrationWriter interface {
	WriteRecord(record *memory.MemoryRecord) error
}

func logMigrateStderrf(stderr io.Writer, format string, args ...any) {
	if stderr == nil {
		return
	}

	_, _ = fmt.Fprintf(stderr, format, args...)
}
