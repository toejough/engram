package maintain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"engram/internal/maintain"
)

// T-P6e-10: ApplyEscalationProposal propagates applier error.
func TestP6e10_ApplyEscalationProposalApplierError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	applier := &fakeEnforcementApplier{
		setFn: func(_, _, _ string) error {
			return errors.New("registry unavailable")
		},
	}

	proposal := maintain.EscalationProposal{
		MemoryPath:    "mem/foo.toml",
		ProposedLevel: "emphasized_advisory",
	}

	err := maintain.ApplyEscalationProposal(proposal, "content", applier, nil, nil)
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("setting enforcement level")))
}

// T-P6e-1: LevelGraduated exists as 4th escalation level.
func TestP6e1_LevelGraduatedIs4thLevel(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	engine := maintain.NewEscalationEngine(nil, fixedNow)

	// At reminder level, next should be graduated.
	leeches := []maintain.EscalationMemory{
		{
			Path:            "mem-reminder",
			Content:         "use descriptive names",
			EscalationLevel: maintain.LevelReminder,
			Effectiveness:   0.10,
		},
	}

	proposals, err := engine.Analyze(leeches)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].ProposalType).To(gomega.Equal("escalate"))
	g.Expect(proposals[0].ProposedLevel).To(gomega.Equal(string(maintain.LevelGraduated)))
}

// T-P6e-2: ApplyEscalationProposal calls SetEnforcementLevel with correct args.
func TestP6e2_ApplyEscalationProposalCallsRegistry(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var (
		gotID     string
		gotLevel  string
		gotReason string
	)

	applier := &fakeEnforcementApplier{
		setFn: func(id, level, reason string) error {
			gotID = id
			gotLevel = level
			gotReason = reason

			return nil
		},
	}

	proposal := maintain.EscalationProposal{
		MemoryPath:      "mem/foo.toml",
		ProposalType:    "escalate",
		CurrentLevel:    "advisory",
		ProposedLevel:   "emphasized_advisory",
		Rationale:       "Memory ineffective at advisory level",
		PredictedImpact: "unknown",
	}

	err := maintain.ApplyEscalationProposal(proposal, "some content", applier, nil, nil)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(gotID).To(gomega.Equal("mem/foo.toml"))
	g.Expect(gotLevel).To(gomega.Equal("emphasized_advisory"))
	g.Expect(gotReason).To(gomega.Equal("Memory ineffective at advisory level"))
}

// T-P6e-3: ApplyEscalationProposal with nil applier is a no-op.
func TestP6e3_ApplyEscalationProposalNilApplierNoOp(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	proposal := maintain.EscalationProposal{
		MemoryPath:    "mem/foo.toml",
		ProposalType:  "escalate",
		ProposedLevel: "emphasized_advisory",
		Rationale:     "ineffective",
	}

	err := maintain.ApplyEscalationProposal(proposal, "content", nil, nil, nil)
	g.Expect(err).NotTo(gomega.HaveOccurred())
}

// T-P6e-4: ApplyEscalationProposal to graduated emits graduation signal.
func TestP6e4_ApplyEscalationProposalGraduatedEmitsSignal(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	applier := &fakeEnforcementApplier{
		setFn: func(_, _, _ string) error { return nil },
	}

	var (
		gotPath           string
		gotRecommendation string
		gotAt             time.Time
	)

	emitter := &fakeGraduationEmitter{
		emitFn: func(path, recommendation string, at time.Time) error {
			gotPath = path
			gotRecommendation = recommendation
			gotAt = at

			return nil
		},
	}

	now := fixedNow()
	proposal := maintain.EscalationProposal{
		MemoryPath:    "mem/graduated.toml",
		ProposalType:  "escalate",
		ProposedLevel: string(maintain.LevelGraduated),
		Rationale:     "top of ladder",
	}

	err := maintain.ApplyEscalationProposal(
		proposal,
		"always use linter settings",
		applier,
		emitter,
		func() time.Time { return now },
	)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(gotPath).To(gomega.Equal("mem/graduated.toml"))
	g.Expect(gotRecommendation).NotTo(gomega.BeEmpty())
	g.Expect(gotAt).To(gomega.BeTemporally("~", now, time.Second))
}

// T-P6e-5: ApplyEscalationProposal to non-graduated does NOT emit graduation signal.
func TestP6e5_ApplyEscalationProposalNonGraduatedNoSignal(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	applier := &fakeEnforcementApplier{
		setFn: func(_, _, _ string) error { return nil },
	}

	emitCount := 0
	emitter := &fakeGraduationEmitter{
		emitFn: func(_, _ string, _ time.Time) error {
			emitCount++

			return nil
		},
	}

	proposal := maintain.EscalationProposal{
		MemoryPath:    "mem/foo.toml",
		ProposedLevel: "emphasized_advisory",
		Rationale:     "escalation",
	}

	err := maintain.ApplyEscalationProposal(proposal, "content", applier, emitter, nil)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(emitCount).To(gomega.Equal(0))
}

// T-P6e-6: ClassifyContent returns "settings.json" for linter/config content.
func TestP6e6_ClassifyContent_Linter(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	result := maintain.ClassifyContent("always configure golangci linter settings")
	g.Expect(result).To(gomega.Equal("settings.json"))
}

// T-P6e-7: ClassifyContent returns ".claude/rules/" for file-scoped content.
func TestP6e7_ClassifyContent_FileScoped(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	result := maintain.ClassifyContent("apply rule file to all .go files via glob pattern")
	g.Expect(result).To(gomega.Equal(".claude/rules/"))
}

// T-P6e-8: ClassifyContent returns "skill" for procedural content.
func TestP6e8_ClassifyContent_Procedural(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	result := maintain.ClassifyContent("follow this workflow step-by-step procedure")
	g.Expect(result).To(gomega.Equal("skill"))
}

// T-P6e-9: ClassifyContent returns "CLAUDE.md" as behavioral default.
func TestP6e9_ClassifyContent_BehavioralDefault(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	result := maintain.ClassifyContent("always write tests before implementing features")
	g.Expect(result).To(gomega.Equal("CLAUDE.md"))
}

// --- Fakes ---

type fakeEnforcementApplier struct {
	setFn func(id, level, reason string) error
}

func (f *fakeEnforcementApplier) SetEnforcementLevel(id, level, reason string) error {
	return f.setFn(id, level, reason)
}

type fakeGraduationEmitter struct {
	emitFn func(memoryPath, recommendation string, detectedAt time.Time) error
}

func (f *fakeGraduationEmitter) EmitGraduation(memoryPath, recommendation string, detectedAt time.Time) error {
	return f.emitFn(memoryPath, recommendation, detectedAt)
}
