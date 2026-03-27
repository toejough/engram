package adapt

import (
	"fmt"

	"engram/internal/policy"
)

// MaintenanceAnalysisConfig holds thresholds for maintenance outcome analysis.
type MaintenanceAnalysisConfig struct {
	MinMeasuredOutcomes int
	MinSuccessRate      float64
}

// AnalyzeMaintenanceOutcomes groups measured MaintenanceHistory entries by action
// type and generates maintenance policy proposals for action types with low success
// rates (effectiveness didn't improve).
func AnalyzeMaintenanceOutcomes(
	records []MeasurableRecord,
	cfg MaintenanceAnalysisConfig,
) []policy.Policy {
	type actionStats struct {
		total    int
		improved int
	}

	stats := make(map[string]*actionStats)

	for _, rec := range records {
		for _, action := range rec.Record.MaintenanceHistory {
			if !action.Measured {
				continue
			}

			entry, exists := stats[action.Action]
			if !exists {
				entry = &actionStats{}
				stats[action.Action] = entry
			}

			entry.total++

			if action.EffectivenessAfter > action.EffectivenessBefore {
				entry.improved++
			}
		}
	}

	proposals := make([]policy.Policy, 0)

	for actionType, stat := range stats {
		if stat.total < cfg.MinMeasuredOutcomes {
			continue
		}

		successRate := float64(stat.improved) / float64(stat.total)
		if successRate >= cfg.MinSuccessRate {
			continue
		}

		proposals = append(proposals, policy.Policy{
			Dimension: policy.DimensionMaintenance,
			Directive: fmt.Sprintf(
				"%s has %.0f%% success rate (%d/%d improved); consider alternative actions",
				actionType,
				successRate*percentMultiplier,
				stat.improved, stat.total,
			),
			Rationale: fmt.Sprintf(
				"action type %q improved effectiveness in only %d of %d measured outcomes (%.0f%% < %.0f%% threshold)",
				actionType, stat.improved, stat.total,
				successRate*percentMultiplier, cfg.MinSuccessRate*percentMultiplier,
			),
			Evidence: policy.Evidence{
				FollowRate: successRate,
				SampleSize: stat.total,
			},
			Status: policy.StatusProposed,
		})
	}

	return proposals
}
