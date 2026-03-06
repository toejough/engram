// Package track provides surfacing instrumentation logic for memories.
package track

import (
	"time"

	"engram/internal/memory"
)

// Exported constants.
const (
	MaxContextEntries = 10
)

// SurfacingUpdate holds the computed tracking values after a surfacing event.
type SurfacingUpdate struct {
	SurfacedCount     int
	LastSurfaced      time.Time
	SurfacingContexts []string
}

// ComputeUpdate calculates new tracking values for a memory after being surfaced.
// It does not mutate the input — it returns a new SurfacingUpdate.
func ComputeUpdate(current *memory.Stored, mode string, now time.Time) SurfacingUpdate {
	newCount := current.SurfacedCount + 1

	// Build new contexts slice: append mode, then FIFO evict if over max.
	existing := current.SurfacingContexts
	newContexts := make([]string, 0, MaxContextEntries)
	newContexts = append(newContexts, existing...)
	newContexts = append(newContexts, mode)

	if len(newContexts) > MaxContextEntries {
		newContexts = newContexts[len(newContexts)-MaxContextEntries:]
	}

	return SurfacingUpdate{
		SurfacedCount:     newCount,
		LastSurfaced:      now,
		SurfacingContexts: newContexts,
	}
}
