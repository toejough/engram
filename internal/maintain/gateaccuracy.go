package maintain

import (
	"fmt"

	"engram/internal/memory"
	"engram/internal/policy"
)

// CheckGateAccuracy computes aggregate surfacing irrelevance rate across all memories.
// If the rate exceeds the threshold, returns a recommend proposal for prompt review.
// Returns nil if rate is at or below the threshold, or if no surfacings have occurred.
func CheckGateAccuracy(records []memory.StoredRecord, threshold float64) *Proposal {
	totalSurfaced, _, _, totalIrrelevant := aggregateMetrics(records)

	if totalSurfaced == 0 {
		return nil
	}

	irrelevanceRate := float64(totalIrrelevant) / float64(totalSurfaced) * percentScale

	if irrelevanceRate <= threshold {
		return nil
	}

	return &Proposal{
		ID:     "gate-accuracy",
		Action: ActionRecommend,
		Target: policy.Filename,
		Field:  "SurfaceGateHaikuPrompt",
		Rationale: fmt.Sprintf(
			"aggregate irrelevance rate %.0f%% exceeds %.0f%% threshold — "+
				"SurfaceGateHaikuPrompt may need revision",
			irrelevanceRate, threshold,
		),
	}
}
