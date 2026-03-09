package registry

import (
	"math"
	"time"
)

// Effectiveness computes followed/(followed+contradicted+ignored) as a
// percentage. Returns nil if the entry has fewer than 3 total evaluations.
func Effectiveness(entry *InstructionEntry) *float64 {
	total := entry.Evaluations.Total()
	if total < minEvaluationsForEffectiveness {
		return nil
	}

	score := float64(entry.Evaluations.Followed) /
		float64(total) * percentMultiplier

	return &score
}

// Frecency computes surfaced_count * exp(-t/halfLife) where t is the
// number of days since the entry was last surfaced (or updated, as fallback).
func Frecency(
	entry *InstructionEntry,
	now time.Time,
	halfLifeDays float64,
) float64 {
	refTime := entry.UpdatedAt
	if entry.LastSurfaced != nil {
		refTime = *entry.LastSurfaced
	}

	const hoursPerDay = 24.0

	daysSince := now.Sub(refTime).Hours() / hoursPerDay
	if daysSince < 0 {
		daysSince = 0
	}

	return float64(entry.SurfacedCount) * math.Exp(-daysSince/halfLifeDays)
}

// unexported constants.
const (
	minEvaluationsForEffectiveness = 3
	percentMultiplier              = 100.0
)
