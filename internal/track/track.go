// Package track provides surfacing instrumentation logic for memories.
package track

import (
	"time"
)

// SurfacingUpdate holds the computed tracking values after a surfacing event.
type SurfacingUpdate struct {
	SurfacedCount int
	LastSurfaced  time.Time
}

// ComputeUpdate calculates new tracking values after a surfacing event.
// Takes explicit current values for pure computation.
func ComputeUpdate(currentCount int, now time.Time) SurfacingUpdate {
	return SurfacingUpdate{
		SurfacedCount: currentCount + 1,
		LastSurfaced:  now,
	}
}
