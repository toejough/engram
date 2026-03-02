// Package reclassify decreases impact scores for memories surfaced before a correction (ARCH-13).
package reclassify

import (
	"context"
	"fmt"

	"engram/internal/audit"
)

// AuditLog records reclassification events.
type AuditLog interface {
	Log(entry audit.Entry) error
}

// Store provides session surfacing log queries and impact score updates.
type Store interface {
	GetSessionSurfacings(ctx context.Context) ([]string, error)
	DecreaseImpact(ctx context.Context, memoryID string, factor float64) error
}

// Run reclassifies all memories surfaced in the current session by applying
// multiplicative decay to their impact scores. Returns the count of reclassified memories.
func Run(ctx context.Context, st Store, auditLog AuditLog, decayFactor float64) (int, error) {
	ids, err := st.GetSessionSurfacings(ctx)
	if err != nil {
		return 0, fmt.Errorf("reclassify: get surfacings: %w", err)
	}

	if len(ids) == 0 {
		return 0, nil
	}

	for _, id := range ids {
		err = st.DecreaseImpact(ctx, id, decayFactor)
		if err != nil {
			return 0, fmt.Errorf("reclassify: decrease impact %s: %w", id, err)
		}

		err = auditLog.Log(audit.Entry{
			Operation: "reclass",
			Action:    "decreased",
			Fields:    map[string]string{"memory_id": id},
		})
		if err != nil {
			return 0, fmt.Errorf("reclassify: audit: %w", err)
		}
	}

	return len(ids), nil
}
