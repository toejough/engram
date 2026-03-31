package surface

import (
	"fmt"
	"time"

	"engram/internal/memory"
)

// ModifyFunc reads a memory TOML, applies a mutation, and writes it back.
type ModifyFunc func(path string, mutate func(*memory.MemoryRecord)) error

// WithPendingEvalModifier sets the modifier for writing pending evaluations.
func WithPendingEvalModifier(fn ModifyFunc) SurfacerOption {
	return func(s *Surfacer) { s.pendingEvalModifier = fn }
}

// WritePendingEvaluations appends a pending evaluation entry to each surfaced memory.
func WritePendingEvaluations(
	memories []*memory.Stored,
	modify ModifyFunc,
	sessionID, projectSlug, userPrompt string,
	now time.Time,
) error {
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
			return fmt.Errorf("writing pending evaluation for %s: %w", mem.FilePath, writeErr)
		}
	}

	return nil
}
