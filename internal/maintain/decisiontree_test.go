package maintain_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/maintain"
	"engram/internal/memory"
)

func TestDiagnoseAll_BatchProcessing(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	records := []memory.StoredRecord{
		{
			Path: "memories/insufficient.toml",
			Record: memory.MemoryRecord{
				SurfacedCount: 2, FollowedCount: 1,
			},
		},
		{
			Path: "memories/bad.toml",
			Record: memory.MemoryRecord{
				SurfacedCount: 10, FollowedCount: 1,
				NotFollowedCount: 3, IrrelevantCount: 6,
			},
		},
		{
			Path: "memories/good.toml",
			Record: memory.MemoryRecord{
				SurfacedCount: 10, FollowedCount: 8,
				NotFollowedCount: 1, IrrelevantCount: 1,
			},
		},
		{
			Path: "memories/needs-escalation.toml",
			Record: memory.MemoryRecord{
				SurfacedCount: 12, FollowedCount: 3,
				NotFollowedCount: 7, IrrelevantCount: 2,
			},
		},
	}

	cfg := defaultDiagnosisConfig()
	proposals := maintain.DiagnoseAll(records, cfg)

	// insufficient → nil (skipped), bad → delete, good → nil (working),
	// needs-escalation → recommend
	g.Expect(proposals).To(HaveLen(2))
	g.Expect(proposals[0].Action).To(Equal(maintain.ActionDelete))
	g.Expect(proposals[0].Target).To(Equal("memories/bad.toml"))
	g.Expect(proposals[1].Action).To(Equal(maintain.ActionRecommend))
	g.Expect(proposals[1].Target).To(Equal("memories/needs-escalation.toml"))
}

func TestDiagnose_Ambiguous(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// effectiveness = 3/10 * 100 = 30% (below threshold)
	// irrelevant_rate = 2/10 * 100 = 20% (below threshold)
	// not_followed_rate = 3/10 * 100 = 30% (below threshold)
	record := &memory.MemoryRecord{
		SurfacedCount:    10,
		FollowedCount:    3,
		NotFollowedCount: 3,
		IrrelevantCount:  2,
	}

	cfg := defaultDiagnosisConfig()
	proposal := maintain.Diagnose("memories/meh-memory.toml", record, cfg)

	g.Expect(proposal).To(BeNil())
}

func TestDiagnose_InsufficientData(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	record := &memory.MemoryRecord{
		SurfacedCount:    3,
		FollowedCount:    2,
		NotFollowedCount: 1,
		IrrelevantCount:  0,
	}

	cfg := defaultDiagnosisConfig()
	proposal := maintain.Diagnose("memories/use-targ.toml", record, cfg)

	g.Expect(proposal).To(BeNil())
}

func TestDiagnose_NarrowSituation(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// effectiveness = 4/10 * 100 = 40% (below threshold but not combined with high irrelevance for delete)
	// irrelevant_rate = 5/10 * 100 = 50% (above threshold)
	// Since effectiveness < threshold AND irrelevant >= threshold, this hits priority 2 (delete).
	// To get priority 3 (narrow situation), we need effectiveness >= threshold OR
	// just irrelevant >= threshold without low effectiveness.
	// effectiveness = 7/10 * 100 = 70% (above threshold), irrelevant_rate = 5/10 * 100 = 50%
	record := &memory.MemoryRecord{
		SurfacedCount:    10,
		FollowedCount:    7,
		NotFollowedCount: 0,
		IrrelevantCount:  5,
	}

	cfg := defaultDiagnosisConfig()
	proposal := maintain.Diagnose("memories/use-targ.toml", record, cfg)

	g.Expect(proposal).NotTo(BeNil())

	if proposal == nil {
		return
	}

	g.Expect(proposal.Action).To(Equal(maintain.ActionUpdate))
	g.Expect(proposal.Field).To(Equal("situation"))
	g.Expect(proposal.Target).To(Equal("memories/use-targ.toml"))
	g.Expect(proposal.ID).To(Equal("diag-use-targ-narrow"))
	g.Expect(proposal.Rationale).NotTo(BeEmpty())
}

func TestDiagnose_Recommend_PersistentNotFollowed(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// not_followed_rate = 7/12 * 100 = 58.3% (above threshold)
	// surfaced_count = 12 >= 2*5 = 10 → priority 4 (recommend)
	record := &memory.MemoryRecord{
		SurfacedCount:    12,
		FollowedCount:    3,
		NotFollowedCount: 7,
		IrrelevantCount:  2,
	}

	// irrelevant_rate = 2/12 = 16.7% (below threshold)

	cfg := defaultDiagnosisConfig()
	proposal := maintain.Diagnose("memories/use-targ.toml", record, cfg)

	g.Expect(proposal).NotTo(BeNil())

	if proposal == nil {
		return
	}

	g.Expect(proposal.Action).To(Equal(maintain.ActionRecommend))
	g.Expect(proposal.Target).To(Equal("memories/use-targ.toml"))
	g.Expect(proposal.ID).To(Equal("diag-use-targ-escalate"))
	g.Expect(proposal.Rationale).NotTo(BeEmpty())
}

func TestDiagnose_Remove(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// effectiveness = 1/10 * 100 = 10%, irrelevant_rate = 6/10 * 100 = 60%
	record := &memory.MemoryRecord{
		SurfacedCount:    10,
		FollowedCount:    1,
		NotFollowedCount: 3,
		IrrelevantCount:  6,
	}

	cfg := defaultDiagnosisConfig()
	proposal := maintain.Diagnose("memories/bad-memory.toml", record, cfg)

	g.Expect(proposal).NotTo(BeNil())

	if proposal == nil {
		return
	}

	g.Expect(proposal.Action).To(Equal(maintain.ActionDelete))
	g.Expect(proposal.Target).To(Equal("memories/bad-memory.toml"))
	g.Expect(proposal.ID).To(Equal("diag-bad-memory-remove"))
	g.Expect(proposal.Rationale).NotTo(BeEmpty())
}

func TestDiagnose_RewriteAction(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// not_followed_rate = 5/8 * 100 = 62.5% (above threshold)
	// surfaced_count = 8 < 2*5 = 10, so NOT >= 2*min -- priority 4b
	// irrelevant_rate = 1/8 = 12.5% (below threshold)
	record := &memory.MemoryRecord{
		SurfacedCount:    8,
		FollowedCount:    2,
		NotFollowedCount: 5,
		IrrelevantCount:  1,
	}

	cfg := defaultDiagnosisConfig()
	proposal := maintain.Diagnose("memories/check-tests.toml", record, cfg)

	g.Expect(proposal).NotTo(BeNil())

	if proposal == nil {
		return
	}

	g.Expect(proposal.Action).To(Equal(maintain.ActionUpdate))
	g.Expect(proposal.Field).To(Equal("action"))
	g.Expect(proposal.Target).To(Equal("memories/check-tests.toml"))
	g.Expect(proposal.ID).To(Equal("diag-check-tests-rewrite"))
	g.Expect(proposal.Rationale).NotTo(BeEmpty())
}

func TestDiagnose_Working(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// effectiveness = 8/10 * 100 = 80% (above threshold)
	record := &memory.MemoryRecord{
		SurfacedCount:    10,
		FollowedCount:    8,
		NotFollowedCount: 1,
		IrrelevantCount:  1,
	}

	cfg := defaultDiagnosisConfig()
	proposal := maintain.Diagnose("memories/good-memory.toml", record, cfg)

	g.Expect(proposal).To(BeNil())
}

func defaultDiagnosisConfig() maintain.DiagnosisConfig {
	return maintain.DiagnosisConfig{
		MinSurfaced:            5,
		EffectivenessThreshold: 60.0,
		IrrelevanceThreshold:   40.0,
		NotFollowedThreshold:   50.0,
	}
}
