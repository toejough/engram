package maintain_test

import (
	"errors"
	"testing"

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

	err := maintain.ApplyEscalationProposal(proposal, applier)
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("setting enforcement level")))
}

// T-P6e-1: Reminder is the top of the escalation ladder.
func TestP6e1_ReminderIsTopLevel(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	engine := maintain.NewEscalationEngine(nil, fixedNow)

	// At reminder level, no further escalation should be proposed.
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

	g.Expect(proposals).To(gomega.BeEmpty())
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

	err := maintain.ApplyEscalationProposal(proposal, applier)
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

	err := maintain.ApplyEscalationProposal(proposal, nil)
	g.Expect(err).NotTo(gomega.HaveOccurred())
}

// --- Fakes ---

type fakeEnforcementApplier struct {
	setFn func(id, level, reason string) error
}

func (f *fakeEnforcementApplier) SetEnforcementLevel(id, level, reason string) error {
	return f.setFn(id, level, reason)
}
