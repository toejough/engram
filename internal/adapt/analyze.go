// Package adapt analyzes feedback patterns across memories and generates policy proposals.
package adapt

import (
	"fmt"
	"sort"
	"strings"

	"engram/internal/memory"
	"engram/internal/policy"
)

// Config holds thresholds for the analysis engine.
type Config struct {
	MinClusterSize             int
	MinFeedbackEvents          int
	MeasurementWindow          int
	MaintenanceMinOutcomes     int
	MaintenanceMinSuccess      float64
	MinNewFeedback             int
	ConsolidationMinConfidence float64
}

// AnalyzeAll runs all analysis functions, combines and sorts proposals by sample size descending.
// Returns the combined proposals and the IDs of policies that were validated.
func AnalyzeAll(
	memories []*memory.Stored,
	cfg Config,
	activePolicies []policy.Policy,
	measurableRecords []MeasurableRecord,
) ([]policy.Policy, []string) {
	contentProposals := AnalyzeContentPatterns(memories, cfg)
	structuralProposals := AnalyzeStructuralPatterns(memories, cfg)
	surfacingProposals := AnalyzeSurfacingPatterns(memories, activePolicies, cfg.MeasurementWindow)
	maintenanceProposals := AnalyzeMaintenanceOutcomes(measurableRecords, MaintenanceAnalysisConfig{
		MinMeasuredOutcomes: cfg.MaintenanceMinOutcomes,
		MinSuccessRate:      cfg.MaintenanceMinSuccess,
	})
	evalResult := EvaluateActivePolicies(memories, activePolicies, cfg.MeasurementWindow)

	capacity := len(contentProposals) + len(structuralProposals) + len(surfacingProposals) +
		len(maintenanceProposals) + len(evalResult.RetirementProposals)
	proposals := make([]policy.Policy, 0, capacity)
	proposals = append(proposals, contentProposals...)
	proposals = append(proposals, structuralProposals...)
	proposals = append(proposals, surfacingProposals...)
	proposals = append(proposals, maintenanceProposals...)
	proposals = append(proposals, evalResult.RetirementProposals...)
	sort.Slice(proposals, func(i, j int) bool {
		return proposals[i].Evidence.SampleSize > proposals[j].Evidence.SampleSize
	})

	return proposals, evalResult.ValidatedPolicyIDs
}

// AnalyzeContentPatterns clusters memories by shared keywords and generates
// extraction policy proposals for keyword clusters with high irrelevance rates.
func AnalyzeContentPatterns(memories []*memory.Stored, cfg Config) []policy.Policy {
	clusters := clusterByKeywords(memories)
	proposals := make([]policy.Policy, 0, len(clusters))

	for keyword, cluster := range clusters {
		if len(cluster) < cfg.MinClusterSize {
			continue
		}

		totalFeedback := 0
		totalIrrelevant := 0

		for _, mem := range cluster {
			feedback := feedbackTotal(mem)
			if feedback < cfg.MinFeedbackEvents {
				continue
			}

			totalFeedback += feedback
			totalIrrelevant += mem.IrrelevantCount
		}

		if totalFeedback == 0 {
			continue
		}

		irrelevanceRate := float64(totalIrrelevant) / float64(totalFeedback)
		if irrelevanceRate < highIrrelevanceThreshold {
			continue
		}

		proposals = append(proposals, policy.Policy{
			Dimension: policy.DimensionExtraction,
			Directive: fmt.Sprintf(
				"de-prioritize keyword %q: %.0f%% irrelevance rate across %d memories",
				keyword, irrelevanceRate*percentMultiplier, len(cluster),
			),
			Rationale: fmt.Sprintf(
				"keyword %q has irrelevance rate of %.0f%% (threshold %.0f%%)",
				keyword, irrelevanceRate*percentMultiplier, highIrrelevanceThreshold*percentMultiplier,
			),
			Evidence: policy.Evidence{
				IrrelevantRate: irrelevanceRate,
				SampleSize:     totalFeedback,
			},
			Status: policy.StatusProposed,
		})
	}

	return proposals
}

// AnalyzeStructuralPatterns groups memories by tier and generates surfacing
// policy proposals when Tier B outperforms Tier A by more than tierDifferenceThreshold.
func AnalyzeStructuralPatterns(memories []*memory.Stored, cfg Config) []policy.Policy {
	tiers := map[string][]*memory.Stored{
		"A": {},
		"B": {},
		"C": {},
	}

	for _, mem := range memories {
		if _, known := tiers[mem.Tier]; known {
			tiers[mem.Tier] = append(tiers[mem.Tier], mem)
		}
	}

	proposals := make([]policy.Policy, 0, 1)

	tierA := tiers["A"]
	tierB := tiers["B"]

	if len(tierA) < cfg.MinClusterSize || len(tierB) < cfg.MinClusterSize {
		return proposals
	}

	followRateA := tierFollowRate(tierA, cfg.MinFeedbackEvents)
	followRateB := tierFollowRate(tierB, cfg.MinFeedbackEvents)

	if followRateB-followRateA <= tierDifferenceThreshold {
		return proposals
	}

	sampleSize := 0
	for _, mem := range append(tierA, tierB...) {
		sampleSize += feedbackTotal(mem)
	}

	proposals = append(proposals, policy.Policy{
		Dimension: policy.DimensionSurfacing,
		Directive: fmt.Sprintf(
			"adjust tier boosts: Tier B follow rate (%.0f%%) exceeds Tier A (%.0f%%) by %.0f%%",
			followRateB*percentMultiplier,
			followRateA*percentMultiplier,
			(followRateB-followRateA)*percentMultiplier,
		),
		Rationale: fmt.Sprintf(
			"Tier B outperforms Tier A by %.0f%% (threshold %.0f%%); tier boost calibration may be misconfigured",
			(followRateB-followRateA)*percentMultiplier,
			tierDifferenceThreshold*percentMultiplier,
		),
		Evidence: policy.Evidence{
			FollowRate: followRateB,
			SampleSize: sampleSize,
		},
		Status: policy.StatusProposed,
	})

	return proposals
}

// AnalyzeSurfacingPatterns evaluates active surfacing policies that have
// exceeded their measurement window. Compares current corpus snapshot to
// the before-snapshot stored on each policy. Returns retirement proposals
// for policies that degraded or didn't improve metrics.
func AnalyzeSurfacingPatterns(
	memories []*memory.Stored,
	activePolicies []policy.Policy,
	measurementWindow int,
) []policy.Policy {
	proposals := make([]policy.Policy, 0)

	for _, pol := range activePolicies {
		if pol.Dimension != policy.DimensionSurfacing {
			continue
		}

		if pol.Effectiveness.Validated {
			continue
		}

		if pol.Effectiveness.MeasuredSessions < measurementWindow {
			continue
		}

		current := ComputeCorpusSnapshot(memories)

		improved := current.FollowRate > pol.Effectiveness.BeforeFollowRate ||
			current.MeanEffectiveness > pol.Effectiveness.BeforeMeanEffectiveness

		if improved {
			continue
		}

		proposals = append(proposals, policy.Policy{
			Dimension: policy.DimensionSurfacing,
			Directive: fmt.Sprintf(
				"retire %s: follow rate %.0f%% (was %.0f%%), mean effectiveness %.1f (was %.1f)",
				pol.ID,
				current.FollowRate*percentMultiplier,
				pol.Effectiveness.BeforeFollowRate*percentMultiplier,
				current.MeanEffectiveness,
				pol.Effectiveness.BeforeMeanEffectiveness,
			),
			Rationale: fmt.Sprintf(
				"policy %s did not improve corpus metrics after %d sessions",
				pol.ID, pol.Effectiveness.MeasuredSessions,
			),
			Evidence: policy.Evidence{
				FollowRate:       current.FollowRate,
				SampleSize:       len(memories),
				SessionsObserved: pol.Effectiveness.MeasuredSessions,
			},
			Status: policy.StatusProposed,
		})
	}

	return proposals
}

// unexported constants.
const (
	highIrrelevanceThreshold = 0.6
	percentMultiplier        = 100
	tierDifferenceThreshold  = 0.15
)

// clusterByKeywords groups memories by keyword (lowercased).
// Each memory appears in multiple clusters (one per keyword).
func clusterByKeywords(memories []*memory.Stored) map[string][]*memory.Stored {
	clusters := make(map[string][]*memory.Stored)

	for _, mem := range memories {
		for _, keyword := range mem.Keywords {
			lowered := strings.ToLower(keyword)
			clusters[lowered] = append(clusters[lowered], mem)
		}
	}

	return clusters
}

// feedbackTotal returns the sum of all feedback event counts for a memory.
func feedbackTotal(mem *memory.Stored) int {
	return mem.FollowedCount + mem.ContradictedCount + mem.IgnoredCount + mem.IrrelevantCount
}

// tierFollowRate computes the aggregate follow rate for a group of memories.
// Memories below minFeedbackEvents are excluded from the aggregate.
func tierFollowRate(mems []*memory.Stored, minFeedbackEvents int) float64 {
	totalFollowed := 0
	totalFeedback := 0

	for _, mem := range mems {
		feedback := feedbackTotal(mem)
		if feedback < minFeedbackEvents {
			continue
		}

		totalFollowed += mem.FollowedCount
		totalFeedback += feedback
	}

	if totalFeedback == 0 {
		return 0
	}

	return float64(totalFollowed) / float64(totalFeedback)
}
