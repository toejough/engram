package adapt_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/adapt"
	"engram/internal/memory"
	"engram/internal/policy"
)

func TestAnalyzeAll_CombinesAndSorts(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Use enough memories to trigger at least one proposal
	memories := []*memory.Stored{
		{FilePath: "a.toml", Keywords: []string{"tool"}, IrrelevantCount: 8, SurfacedCount: 10},
		{FilePath: "b.toml", Keywords: []string{"tool"}, IrrelevantCount: 7, SurfacedCount: 9},
		{FilePath: "c.toml", Keywords: []string{"tool"}, IrrelevantCount: 6, SurfacedCount: 9},
		{FilePath: "d.toml", Keywords: []string{"tool"}, IrrelevantCount: 9, SurfacedCount: 10},
		{FilePath: "e.toml", Keywords: []string{"tool"}, IrrelevantCount: 5, SurfacedCount: 8},
	}

	config := adapt.Config{MinClusterSize: 5, MinFeedbackEvents: 3}
	proposals := adapt.AnalyzeAll(memories, config)
	g.Expect(proposals).NotTo(BeEmpty())
}

func TestAnalyze_ContentPattern_BelowMinCluster(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{FilePath: "a.toml", Keywords: []string{"build-tool"}, IrrelevantCount: 8, SurfacedCount: 10},
		{FilePath: "b.toml", Keywords: []string{"build-tool"}, IrrelevantCount: 7, SurfacedCount: 9},
	}

	config := adapt.Config{MinClusterSize: 5, MinFeedbackEvents: 3}
	proposals := adapt.AnalyzeContentPatterns(memories, config)
	g.Expect(proposals).To(BeEmpty())
}

func TestAnalyze_ContentPattern_HighIrrelevanceCluster(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			FilePath:        "a.toml",
			Keywords:        []string{"build-tool", "targ"},
			IrrelevantCount: 8,
			FollowedCount:   1,
			SurfacedCount:   10,
		},
		{
			FilePath:        "b.toml",
			Keywords:        []string{"build-tool", "compilation"},
			IrrelevantCount: 7,
			FollowedCount:   1,
			SurfacedCount:   9,
		},
		{
			FilePath:        "c.toml",
			Keywords:        []string{"build-tool", "lint"},
			IrrelevantCount: 6,
			FollowedCount:   2,
			SurfacedCount:   9,
		},
		{
			FilePath:        "d.toml",
			Keywords:        []string{"build-tool", "test-runner"},
			IrrelevantCount: 9,
			FollowedCount:   0,
			SurfacedCount:   10,
		},
		{
			FilePath:        "e.toml",
			Keywords:        []string{"build-tool", "coverage"},
			IrrelevantCount: 5,
			FollowedCount:   2,
			SurfacedCount:   8,
		},
	}

	config := adapt.Config{MinClusterSize: 5, MinFeedbackEvents: 3}
	proposals := adapt.AnalyzeContentPatterns(memories, config)
	g.Expect(proposals).NotTo(BeEmpty())
	g.Expect(proposals[0].Dimension).To(Equal(policy.DimensionExtraction))
	g.Expect(proposals[0].Directive).To(ContainSubstring("build-tool"))
}

func TestAnalyze_ContentPattern_LowIrrelevance_NoProposal(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{FilePath: "a.toml", Keywords: []string{"design"}, FollowedCount: 8, IrrelevantCount: 1, SurfacedCount: 10},
		{FilePath: "b.toml", Keywords: []string{"design"}, FollowedCount: 7, IrrelevantCount: 1, SurfacedCount: 9},
		{FilePath: "c.toml", Keywords: []string{"design"}, FollowedCount: 6, IrrelevantCount: 2, SurfacedCount: 9},
		{FilePath: "d.toml", Keywords: []string{"design"}, FollowedCount: 9, IrrelevantCount: 0, SurfacedCount: 10},
		{FilePath: "e.toml", Keywords: []string{"design"}, FollowedCount: 5, IrrelevantCount: 2, SurfacedCount: 8},
	}

	config := adapt.Config{MinClusterSize: 5, MinFeedbackEvents: 3}
	proposals := adapt.AnalyzeContentPatterns(memories, config)
	g.Expect(proposals).To(BeEmpty())
}

func TestAnalyze_StructuralPattern_TierBoostMismatch(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{FilePath: "a.toml", Tier: "A", FollowedCount: 2, IgnoredCount: 8, SurfacedCount: 10},
		{FilePath: "b.toml", Tier: "A", FollowedCount: 1, IgnoredCount: 7, SurfacedCount: 10},
		{FilePath: "c.toml", Tier: "A", FollowedCount: 3, IgnoredCount: 7, SurfacedCount: 10},
		{FilePath: "d.toml", Tier: "B", FollowedCount: 8, IgnoredCount: 2, SurfacedCount: 10},
		{FilePath: "e.toml", Tier: "B", FollowedCount: 7, IgnoredCount: 1, SurfacedCount: 10},
		{FilePath: "f.toml", Tier: "B", FollowedCount: 9, IgnoredCount: 1, SurfacedCount: 10},
	}

	config := adapt.Config{MinClusterSize: 3, MinFeedbackEvents: 3}
	proposals := adapt.AnalyzeStructuralPatterns(memories, config)
	g.Expect(proposals).NotTo(BeEmpty())
	g.Expect(proposals[0].Dimension).To(Equal(policy.DimensionSurfacing))
}

func TestAnalyzeSurfacingPatterns_ProposesRetirement(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Low follow rate: 2 followed out of 10 total = 0.2
	memories := []*memory.Stored{
		{FilePath: "a.toml", FollowedCount: 1, IgnoredCount: 4, SurfacedCount: 5},
		{FilePath: "b.toml", FollowedCount: 1, IgnoredCount: 4, SurfacedCount: 5},
	}

	activePolicies := []policy.Policy{
		{
			ID:        "pol-001",
			Dimension: policy.DimensionSurfacing,
			Status:    policy.StatusActive,
			Effectiveness: policy.Effectiveness{
				BeforeFollowRate:        0.50,
				BeforeMeanEffectiveness: 50.0,
				MeasuredSessions:        10,
				Validated:               false,
			},
		},
	}

	proposals := adapt.AnalyzeSurfacingPatterns(memories, activePolicies, 5)
	g.Expect(proposals).To(HaveLen(1))
	g.Expect(proposals[0].Dimension).To(Equal(policy.DimensionSurfacing))
	g.Expect(proposals[0].Directive).To(ContainSubstring("pol-001"))
	g.Expect(proposals[0].Status).To(Equal(policy.StatusProposed))
}

func TestAnalyzeSurfacingPatterns_ValidatesImproved(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// High follow rate: 8 followed out of 10 total = 0.8
	memories := []*memory.Stored{
		{FilePath: "a.toml", FollowedCount: 4, IgnoredCount: 1, SurfacedCount: 5},
		{FilePath: "b.toml", FollowedCount: 4, IgnoredCount: 1, SurfacedCount: 5},
	}

	activePolicies := []policy.Policy{
		{
			ID:        "pol-002",
			Dimension: policy.DimensionSurfacing,
			Status:    policy.StatusActive,
			Effectiveness: policy.Effectiveness{
				BeforeFollowRate:        0.50,
				BeforeMeanEffectiveness: 50.0,
				MeasuredSessions:        10,
				Validated:               false,
			},
		},
	}

	proposals := adapt.AnalyzeSurfacingPatterns(memories, activePolicies, 5)
	g.Expect(proposals).To(BeEmpty())
}

func TestAnalyzeSurfacingPatterns_SkipsBelowWindow(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{FilePath: "a.toml", FollowedCount: 1, IgnoredCount: 4, SurfacedCount: 5},
	}

	activePolicies := []policy.Policy{
		{
			ID:        "pol-003",
			Dimension: policy.DimensionSurfacing,
			Status:    policy.StatusActive,
			Effectiveness: policy.Effectiveness{
				BeforeFollowRate: 0.50,
				MeasuredSessions: 3,
				Validated:        false,
			},
		},
	}

	proposals := adapt.AnalyzeSurfacingPatterns(memories, activePolicies, 5)
	g.Expect(proposals).To(BeEmpty())
}

func TestAnalyzeSurfacingPatterns_SkipsValidated(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{FilePath: "a.toml", FollowedCount: 1, IgnoredCount: 4, SurfacedCount: 5},
	}

	activePolicies := []policy.Policy{
		{
			ID:        "pol-004",
			Dimension: policy.DimensionSurfacing,
			Status:    policy.StatusActive,
			Effectiveness: policy.Effectiveness{
				BeforeFollowRate: 0.50,
				MeasuredSessions: 10,
				Validated:        true,
			},
		},
	}

	proposals := adapt.AnalyzeSurfacingPatterns(memories, activePolicies, 5)
	g.Expect(proposals).To(BeEmpty())
}
