package adapt_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/adapt"
	"engram/internal/memory"
	"engram/internal/policy"
)

func TestEvaluateActivePolicies_ProposesRetirementForDegraded(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{FilePath: "mem-1.toml", FollowedCount: 2, IrrelevantCount: 8},
	}

	policies := []policy.Policy{
		{
			ID: "pol-001", Dimension: policy.DimensionExtraction, Status: policy.StatusActive,
			Effectiveness: policy.Effectiveness{
				BeforeFollowRate: 0.50, BeforeMeanEffectiveness: 60.0, MeasuredSessions: 10,
			},
		},
		{
			ID: "pol-002", Dimension: policy.DimensionMaintenance, Status: policy.StatusActive,
			Effectiveness: policy.Effectiveness{
				BeforeFollowRate: 0.50, BeforeMeanEffectiveness: 60.0, MeasuredSessions: 10,
			},
		},
	}

	const measurementWindow = 10

	result := adapt.EvaluateActivePolicies(memories, policies, measurementWindow)

	// Current follow rate = 2/10 = 0.2, worse than 0.50
	g.Expect(result.RetirementProposals).To(HaveLen(2))
	g.Expect(result.ValidatedPolicyIDs).To(BeEmpty())
}

func TestEvaluateActivePolicies_ValidatesImproved(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{FilePath: "mem-1.toml", FollowedCount: 9, IgnoredCount: 1},
	}

	policies := []policy.Policy{{
		ID: "pol-001", Dimension: policy.DimensionExtraction, Status: policy.StatusActive,
		Effectiveness: policy.Effectiveness{
			BeforeFollowRate: 0.50, BeforeMeanEffectiveness: 40.0, MeasuredSessions: 10,
		},
	}}

	const measurementWindow = 10

	result := adapt.EvaluateActivePolicies(memories, policies, measurementWindow)

	g.Expect(result.RetirementProposals).To(BeEmpty())
	g.Expect(result.ValidatedPolicyIDs).To(ConsistOf("pol-001"))
}

func TestEvaluateActivePolicies_SkipsValidatedAndBelowWindow(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	policies := []policy.Policy{
		{
			ID: "pol-001", Dimension: policy.DimensionSurfacing, Status: policy.StatusActive,
			Effectiveness: policy.Effectiveness{MeasuredSessions: 5},
		},
		{
			ID: "pol-002", Dimension: policy.DimensionExtraction, Status: policy.StatusActive,
			Effectiveness: policy.Effectiveness{MeasuredSessions: 10, Validated: true},
		},
	}

	const measurementWindow = 10

	result := adapt.EvaluateActivePolicies(nil, policies, measurementWindow)

	g.Expect(result.RetirementProposals).To(BeEmpty())
	g.Expect(result.ValidatedPolicyIDs).To(BeEmpty())
}
