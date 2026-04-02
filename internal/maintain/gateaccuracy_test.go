package maintain_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/maintain"
	"engram/internal/memory"
)

func TestCheckGateAccuracy_AboveThreshold_GeneratesProposal(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	records := []memory.StoredRecord{
		{Record: memory.MemoryRecord{SurfacedCount: 50, IrrelevantCount: 10}},
		{Record: memory.MemoryRecord{SurfacedCount: 50, IrrelevantCount: 8}},
	}

	// 18/100 = 18% — above 10% threshold.
	const threshold = 10.0

	proposal := maintain.CheckGateAccuracy(records, threshold)
	g.Expect(proposal).NotTo(BeNil())

	if proposal == nil {
		return
	}

	g.Expect(proposal.Action).To(Equal(maintain.ActionRecommend))
	g.Expect(proposal.Target).To(Equal("policy.toml"))
	g.Expect(proposal.Rationale).To(ContainSubstring("18%"))
	g.Expect(proposal.Rationale).To(ContainSubstring("SurfaceGateHaikuPrompt"))
}

func TestCheckGateAccuracy_BelowThreshold_NoProposal(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	records := []memory.StoredRecord{
		{Record: memory.MemoryRecord{SurfacedCount: 100, IrrelevantCount: 5}},
		{Record: memory.MemoryRecord{SurfacedCount: 50, IrrelevantCount: 2}},
	}

	// 7/150 = 4.7% — below 10% threshold.
	const threshold = 10.0

	proposal := maintain.CheckGateAccuracy(records, threshold)
	g.Expect(proposal).To(BeNil())
}

func TestCheckGateAccuracy_ExactlyAtThreshold_NoProposal(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	records := []memory.StoredRecord{
		{Record: memory.MemoryRecord{SurfacedCount: 100, IrrelevantCount: 10}},
	}

	// 10/100 = 10% — exactly at threshold, should NOT trigger (strictly greater).
	const threshold = 10.0

	proposal := maintain.CheckGateAccuracy(records, threshold)
	g.Expect(proposal).To(BeNil())
}

func TestCheckGateAccuracy_NoSurfacings_NoProposal(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	records := []memory.StoredRecord{
		{Record: memory.MemoryRecord{SurfacedCount: 0, IrrelevantCount: 0}},
	}

	const threshold = 10.0

	proposal := maintain.CheckGateAccuracy(records, threshold)
	g.Expect(proposal).To(BeNil())
}
