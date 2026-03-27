package signal_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/signal"
)

func TestTransferFields_AbsorbedRecords(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	originals := []*memory.MemoryRecord{
		{
			SourcePath:        "/data/memories/mem-a.toml",
			SurfacedCount:     3,
			FollowedCount:     2,
			ContradictedCount: 1,
			IgnoredCount:      4,
			ContentHash:       "abc123",
		},
		{
			SourcePath:        "/data/memories/mem-b.toml",
			SurfacedCount:     7,
			FollowedCount:     5,
			ContradictedCount: 0,
			IgnoredCount:      2,
			ContentHash:       "def456",
		},
	}
	base := &memory.MemoryRecord{}

	signal.TransferFields(base, originals, transferTestNow)

	g.Expect(base.Absorbed).To(HaveLen(2))

	expectedMergedAt := transferTestNow.Format(time.RFC3339)

	g.Expect(base.Absorbed[0]).To(Equal(memory.AbsorbedRecord{
		From:          "/data/memories/mem-a.toml",
		SurfacedCount: 3,
		Evaluations: memory.EvaluationCounters{
			Followed:     2,
			Contradicted: 1,
			Ignored:      4,
		},
		ContentHash: "abc123",
		MergedAt:    expectedMergedAt,
	}))

	g.Expect(base.Absorbed[1]).To(Equal(memory.AbsorbedRecord{
		From:          "/data/memories/mem-b.toml",
		SurfacedCount: 7,
		Evaluations: memory.EvaluationCounters{
			Followed:     5,
			Contradicted: 0,
			Ignored:      2,
		},
		ContentHash: "def456",
		MergedAt:    expectedMergedAt,
	}))
}

func TestTransferFields_ConfidenceAlwaysB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	originals := []*memory.MemoryRecord{
		{Confidence: "A"},
		{Confidence: "C"},
	}
	base := &memory.MemoryRecord{Confidence: "A"}

	signal.TransferFields(base, originals, transferTestNow)

	g.Expect(base.Confidence).To(Equal("B"))
}

func TestTransferFields_EmptyProjectSlug(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	originals := []*memory.MemoryRecord{
		{ProjectSlug: "project-a"},
		{ProjectSlug: "project-b"},
	}
	base := &memory.MemoryRecord{ProjectSlug: "project-a"}

	signal.TransferFields(base, originals, transferTestNow)

	g.Expect(base.ProjectSlug).To(BeEmpty())
}

func TestTransferFields_ResetsIgnoredCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	originals := []*memory.MemoryRecord{
		{IgnoredCount: 4},
		{IgnoredCount: 6},
	}
	base := &memory.MemoryRecord{IgnoredCount: 8}

	signal.TransferFields(base, originals, transferTestNow)

	g.Expect(base.IgnoredCount).To(Equal(0))
}

func TestTransferFields_ResetsIrrelevantCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	originals := []*memory.MemoryRecord{
		{IrrelevantCount: 5},
		{IrrelevantCount: 3},
	}
	base := &memory.MemoryRecord{IrrelevantCount: 10}

	signal.TransferFields(base, originals, transferTestNow)

	g.Expect(base.IrrelevantCount).To(Equal(0))
}

func TestTransferFields_ResetsSurfacedCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	originals := []*memory.MemoryRecord{
		{SurfacedCount: 10},
		{SurfacedCount: 20},
	}
	base := &memory.MemoryRecord{SurfacedCount: 15}

	signal.TransferFields(base, originals, transferTestNow)

	g.Expect(base.SurfacedCount).To(Equal(0))
}

func TestTransferFields_SumsContradictedCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	originals := []*memory.MemoryRecord{
		{ContradictedCount: 1},
		{ContradictedCount: 4},
		{ContradictedCount: 7},
	}
	base := &memory.MemoryRecord{}

	signal.TransferFields(base, originals, transferTestNow)

	g.Expect(base.ContradictedCount).To(Equal(12))
}

func TestTransferFields_SumsFollowedCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	originals := []*memory.MemoryRecord{
		{FollowedCount: 2},
		{FollowedCount: 5},
		{FollowedCount: 3},
	}
	base := &memory.MemoryRecord{}

	signal.TransferFields(base, originals, transferTestNow)

	g.Expect(base.FollowedCount).To(Equal(10))
}

// unexported variables.
var (
	transferTestNow = time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
)
