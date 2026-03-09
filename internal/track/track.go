// Package track provides surfacing instrumentation logic for memories.
package track

import (
	"time"
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

// ComputeUpdate calculates new tracking values after a surfacing event.
// Takes explicit current values rather than reading from memory.Stored,
// since tracking fields are no longer stored inline in TOMLs (UC-23).
func ComputeUpdate(
	currentCount int, currentContexts []string, mode string, now time.Time,
) SurfacingUpdate {
	newCount := currentCount + 1

	// Build new contexts slice: append mode, then FIFO evict if over max.
	newContexts := make([]string, 0, MaxContextEntries)
	newContexts = append(newContexts, currentContexts...)
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
