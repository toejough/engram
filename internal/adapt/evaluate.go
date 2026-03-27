package adapt

import (
	"fmt"

	"engram/internal/memory"
	"engram/internal/policy"
)

// EvaluationResult holds the outcome of evaluating active policies.
type EvaluationResult struct {
	RetirementProposals []policy.Policy
	ValidatedPolicyIDs  []string
}

// EvaluateActivePolicies checks all active policies past their measurement
// window. Compares current corpus snapshot to before-snapshot.
// Returns retirement proposals for degraded policies and IDs of validated ones.
func EvaluateActivePolicies(
	memories []*memory.Stored,
	activePolicies []policy.Policy,
	measurementWindow int,
) EvaluationResult {
	result := EvaluationResult{
		RetirementProposals: make([]policy.Policy, 0),
		ValidatedPolicyIDs:  make([]string, 0),
	}

	current := ComputeCorpusSnapshot(memories)

	for _, pol := range activePolicies {
		if pol.Status != policy.StatusActive {
			continue
		}

		if pol.Effectiveness.Validated {
			continue
		}

		if pol.Effectiveness.MeasuredSessions < measurementWindow {
			continue
		}

		improved := current.FollowRate > pol.Effectiveness.BeforeFollowRate ||
			current.MeanEffectiveness > pol.Effectiveness.BeforeMeanEffectiveness

		if improved {
			result.ValidatedPolicyIDs = append(result.ValidatedPolicyIDs, pol.ID)

			continue
		}

		result.RetirementProposals = append(result.RetirementProposals, policy.Policy{
			Dimension: pol.Dimension,
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

	return result
}
