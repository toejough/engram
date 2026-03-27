package adapt

import "engram/internal/memory"

// MeasurableRecord pairs a memory file path with its loaded record.
type MeasurableRecord struct {
	Path   string
	Record memory.MemoryRecord
}

// MeasuredResult identifies a specific MaintenanceAction that is now measurable.
type MeasuredResult struct {
	Path               string
	ActionIndex        int
	EffectivenessAfter float64
	SurfacedCountAfter int
}

// MeasureOutcomes scans records for unmeasured MaintenanceActions that have
// accumulated at least minNewFeedback new feedback events since the action.
func MeasureOutcomes(records []MeasurableRecord, minNewFeedback int) []MeasuredResult {
	results := make([]MeasuredResult, 0)

	for _, rec := range records {
		currentFeedback := rec.Record.FollowedCount + rec.Record.ContradictedCount +
			rec.Record.IgnoredCount + rec.Record.IrrelevantCount

		var effectivenessNow float64
		if currentFeedback > 0 {
			effectivenessNow = float64(rec.Record.FollowedCount) /
				float64(currentFeedback) * percentMultiplier
		}

		for idx, action := range rec.Record.MaintenanceHistory {
			if action.Measured {
				continue
			}

			newFeedback := currentFeedback - action.FeedbackCountBefore
			if newFeedback < minNewFeedback {
				continue
			}

			results = append(results, MeasuredResult{
				Path:               rec.Path,
				ActionIndex:        idx,
				EffectivenessAfter: effectivenessNow,
				SurfacedCountAfter: rec.Record.SurfacedCount,
			})
		}
	}

	return results
}
