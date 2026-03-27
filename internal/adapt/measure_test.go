package adapt_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/adapt"
	"engram/internal/memory"
)

func TestMeasureOutcomes_FillsAfterScores(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	records := []adapt.MeasurableRecord{
		{
			Path: "mem-1.toml",
			Record: memory.MemoryRecord{
				FollowedCount:     8,
				ContradictedCount: 0,
				IgnoredCount:      2,
				IrrelevantCount:   5,
				SurfacedCount:     20,
				MaintenanceHistory: []memory.MaintenanceAction{{
					Action:              "rewrite",
					AppliedAt:           "2026-03-27T10:00:00Z",
					EffectivenessBefore: 30.0,
					SurfacedCountBefore: 10,
					FeedbackCountBefore: 10,
					Measured:            false,
				}},
			},
		},
	}

	// Current feedback = 15, before = 10, diff = 5 >= minNewFeedback
	const minNewFeedback = 5

	results := adapt.MeasureOutcomes(records, minNewFeedback)

	g.Expect(results).To(HaveLen(1))
	g.Expect(results[0].Path).To(Equal("mem-1.toml"))
	g.Expect(results[0].ActionIndex).To(Equal(0))
	// Effectiveness: 8/15 * 100 = 53.33
	g.Expect(results[0].EffectivenessAfter).To(BeNumerically("~", 53.33, 0.01))
	g.Expect(results[0].SurfacedCountAfter).To(Equal(20))
}

func TestMeasureOutcomes_SkipsAlreadyMeasured(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	records := []adapt.MeasurableRecord{
		{
			Path: "mem-1.toml",
			Record: memory.MemoryRecord{
				FollowedCount: 20,
				MaintenanceHistory: []memory.MaintenanceAction{{
					Action: "rewrite", FeedbackCountBefore: 5, Measured: true,
				}},
			},
		},
	}

	const minNewFeedback = 5

	results := adapt.MeasureOutcomes(records, minNewFeedback)
	g.Expect(results).To(BeEmpty())
}

func TestMeasureOutcomes_SkipsInsufficientFeedback(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	records := []adapt.MeasurableRecord{
		{
			Path: "mem-1.toml",
			Record: memory.MemoryRecord{
				FollowedCount: 4, IgnoredCount: 2, SurfacedCount: 8,
				MaintenanceHistory: []memory.MaintenanceAction{{
					Action: "rewrite", FeedbackCountBefore: 5, Measured: false,
				}},
			},
		},
	}

	const minNewFeedback = 5

	results := adapt.MeasureOutcomes(records, minNewFeedback)
	g.Expect(results).To(BeEmpty())
}
