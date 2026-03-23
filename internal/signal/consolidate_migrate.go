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

// MigrationRunner handles the migrate-scores subcommand pipeline.
type MigrationRunner struct {
	lister MemoryRecordLister
	scorer GeneralizabilityScorer
	writer MigrationWriter
	stderr io.Writer
}

// MigrationWriter writes a MemoryRecord back to its source path.
type MigrationWriter interface {
	WriteRecord(record *memory.MemoryRecord) error
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
				logMigrateStderrf(r.stderr, "[engram] write failed for %q: %v\n", rec.Title, writeErr)

				continue
			}

			scored++
		}
	}

	return scored, nil
}

func logMigrateStderrf(stderr io.Writer, format string, args ...any) {
	if stderr == nil {
		return
	}

	_, _ = fmt.Fprintf(stderr, format, args...)
}
