package surface

import (
	"errors"
	"fmt"
	"time"

	"engram/internal/memory"
)

// WithPendingEvalModifier sets the modifier for writing pending evaluations.
func WithPendingEvalModifier(fn memory.ModifyFunc) SurfacerOption {
	return func(s *Surfacer) { s.pendingEvalModifier = fn }
}

// WritePendingEvaluations appends a pending evaluation entry to each surfaced memory.
// Continues on error so all memories get an evaluation record; returns combined errors.
func WritePendingEvaluations(
	memories []*memory.Stored,
	modify memory.ModifyFunc,
	sessionID, projectSlug, userPrompt string,
	now time.Time,
) error {
	var errs []error

	for _, mem := range memories {
		writeErr := modify(mem.FilePath, func(record *memory.MemoryRecord) {
			record.PendingEvaluations = append(record.PendingEvaluations, memory.PendingEvaluation{
				SurfacedAt:  now.Format(time.RFC3339),
				UserPrompt:  userPrompt,
				SessionID:   sessionID,
				ProjectSlug: projectSlug,
			})
		})
		if writeErr != nil {
			errs = append(
				errs,
				fmt.Errorf("writing pending evaluation for %s: %w", mem.FilePath, writeErr),
			)
		}
	}

	return errors.Join(errs...)
}
