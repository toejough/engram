package adapt_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/adapt"
	"engram/internal/memory"
)

func TestComputeCorpusSnapshot_AggregatesMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			FilePath:          "mem-1.toml",
			FollowedCount:     3,
			ContradictedCount: 1,
			IgnoredCount:      1,
			IrrelevantCount:   5,
			UpdatedAt:         time.Now(),
		},
		{
			FilePath:          "mem-2.toml",
			FollowedCount:     8,
			ContradictedCount: 0,
			IgnoredCount:      0,
			IrrelevantCount:   2,
			UpdatedAt:         time.Now(),
		},
	}

	snap := adapt.ComputeCorpusSnapshot(memories)

	// Total feedback: 10 + 10 = 20, followed: 11, irrelevant: 7
	g.Expect(snap.FollowRate).To(BeNumerically("~", 0.55, 0.001))
	g.Expect(snap.IrrelevanceRatio).To(BeNumerically("~", 0.35, 0.001))
	// Effectiveness: mem1=30, mem2=80, mean=55
	g.Expect(snap.MeanEffectiveness).To(BeNumerically("~", 55.0, 0.001))
}

func TestComputeCorpusSnapshot_EmptyMemories(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	snap := adapt.ComputeCorpusSnapshot(nil)
	g.Expect(snap.FollowRate).To(BeNumerically("~", 0.0, 0.001))
	g.Expect(snap.IrrelevanceRatio).To(BeNumerically("~", 0.0, 0.001))
	g.Expect(snap.MeanEffectiveness).To(BeNumerically("~", 0.0, 0.001))
}

func TestComputeCorpusSnapshot_SkipsZeroFeedback(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	memories := []*memory.Stored{
		{FilePath: "no-feedback.toml"},
		{FilePath: "has-feedback.toml", FollowedCount: 4, IgnoredCount: 6, UpdatedAt: time.Now()},
	}
	snap := adapt.ComputeCorpusSnapshot(memories)
	g.Expect(snap.FollowRate).To(BeNumerically("~", 0.4, 0.001))
	g.Expect(snap.MeanEffectiveness).To(BeNumerically("~", 40.0, 0.001))
}
