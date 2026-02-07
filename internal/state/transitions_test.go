package state_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
)

func TestTDDSubPhaseTransitions(t *testing.T) {
	t.Run("TDD red sub-phase transitions", func(t *testing.T) {
		g := NewWithT(t)
		// Forward path
		g.Expect(state.IsLegalTransition("tdd-red", "tdd-red-qa")).To(BeTrue())
		g.Expect(state.IsLegalTransition("tdd-red-qa", "commit-red")).To(BeTrue())
		g.Expect(state.IsLegalTransition("commit-red", "commit-red-qa")).To(BeTrue())

		// QA improvement loop: commit-red-qa can loop back to tdd-red
		g.Expect(state.IsLegalTransition("commit-red-qa", "tdd-red")).To(BeTrue())

		// Forward to next phase
		g.Expect(state.IsLegalTransition("commit-red-qa", "tdd-green")).To(BeTrue())
	})

	t.Run("TDD green sub-phase transitions", func(t *testing.T) {
		g := NewWithT(t)
		// Forward path
		g.Expect(state.IsLegalTransition("tdd-green", "tdd-green-qa")).To(BeTrue())
		g.Expect(state.IsLegalTransition("tdd-green-qa", "commit-green")).To(BeTrue())
		g.Expect(state.IsLegalTransition("commit-green", "commit-green-qa")).To(BeTrue())

		// QA improvement loop: commit-green-qa can loop back to tdd-green
		g.Expect(state.IsLegalTransition("commit-green-qa", "tdd-green")).To(BeTrue())

		// Forward to next phase
		g.Expect(state.IsLegalTransition("commit-green-qa", "tdd-refactor")).To(BeTrue())
	})

	t.Run("TDD refactor sub-phase transitions", func(t *testing.T) {
		g := NewWithT(t)
		// Forward path
		g.Expect(state.IsLegalTransition("tdd-refactor", "tdd-refactor-qa")).To(BeTrue())
		g.Expect(state.IsLegalTransition("tdd-refactor-qa", "commit-refactor")).To(BeTrue())
		g.Expect(state.IsLegalTransition("commit-refactor", "commit-refactor-qa")).To(BeTrue())

		// QA improvement loop: commit-refactor-qa can loop back to tdd-refactor
		g.Expect(state.IsLegalTransition("commit-refactor-qa", "tdd-refactor")).To(BeTrue())

		// Forward to next phase (task completion paths)
		g.Expect(state.IsLegalTransition("commit-refactor-qa", "task-complete")).To(BeTrue())
		g.Expect(state.IsLegalTransition("commit-refactor-qa", "task-retry")).To(BeTrue())
		g.Expect(state.IsLegalTransition("commit-refactor-qa", "task-escalated")).To(BeTrue())
	})
}

func TestTDDSubPhaseIllegalTransitions(t *testing.T) {
	t.Run("cannot skip QA phases", func(t *testing.T) {
		g := NewWithT(t)
		// Can't skip from producer directly to commit
		g.Expect(state.IsLegalTransition("tdd-red", "commit-red")).To(BeFalse())
		g.Expect(state.IsLegalTransition("tdd-green", "commit-green")).To(BeFalse())
		g.Expect(state.IsLegalTransition("tdd-refactor", "commit-refactor")).To(BeFalse())

		// Can't skip from commit directly to next producer phase
		g.Expect(state.IsLegalTransition("commit-red", "tdd-green")).To(BeFalse())
		g.Expect(state.IsLegalTransition("commit-green", "tdd-refactor")).To(BeFalse())
		g.Expect(state.IsLegalTransition("commit-refactor", "task-complete")).To(BeFalse())
	})

	t.Run("cannot jump between TDD phases", func(t *testing.T) {
		g := NewWithT(t)
		// Can't jump from red to refactor
		g.Expect(state.IsLegalTransition("tdd-red", "tdd-refactor")).To(BeFalse())
		g.Expect(state.IsLegalTransition("tdd-red-qa", "tdd-refactor-qa")).To(BeFalse())

		// Can't go backwards through the main flow
		g.Expect(state.IsLegalTransition("tdd-green", "tdd-red")).To(BeFalse())
		g.Expect(state.IsLegalTransition("tdd-refactor", "tdd-green")).To(BeFalse())
	})

	t.Run("QA phases cannot loop to wrong producer phase", func(t *testing.T) {
		g := NewWithT(t)
		// commit-red-qa can only loop to tdd-red, not tdd-green or tdd-refactor
		g.Expect(state.IsLegalTransition("commit-red-qa", "tdd-refactor")).To(BeFalse())

		// commit-green-qa can only loop to tdd-green, not tdd-red or tdd-refactor
		g.Expect(state.IsLegalTransition("commit-green-qa", "tdd-red")).To(BeFalse())

		// commit-refactor-qa can only loop to tdd-refactor, not tdd-red or tdd-green
		g.Expect(state.IsLegalTransition("commit-refactor-qa", "tdd-red")).To(BeFalse())
		g.Expect(state.IsLegalTransition("commit-refactor-qa", "tdd-green")).To(BeFalse())
	})
}

func TestTDDFullPhaseChain(t *testing.T) {
	t.Run("complete TDD cycle from task-start to task-complete", func(t *testing.T) {
		g := NewWithT(t)

		phases := []string{
			"task-start",
			"tdd-red",
			"tdd-red-qa",
			"commit-red",
			"commit-red-qa",
			"tdd-green",
			"tdd-green-qa",
			"commit-green",
			"commit-green-qa",
			"tdd-refactor",
			"tdd-refactor-qa",
			"commit-refactor",
			"commit-refactor-qa",
			"task-complete",
		}

		// Verify each transition in the chain is legal
		for i := 0; i < len(phases)-1; i++ {
			from := phases[i]
			to := phases[i+1]
			g.Expect(state.IsLegalTransition(from, to)).To(BeTrue(),
				"transition %s → %s should be legal", from, to)
		}
	})

	t.Run("TDD cycle with improvement iteration in red phase", func(t *testing.T) {
		g := NewWithT(t)

		// Red phase with one improvement loop
		phases := []string{
			"task-start",
			"tdd-red",
			"tdd-red-qa",
			"commit-red",
			"commit-red-qa",
			"tdd-red",           // Loop back for improvement
			"tdd-red-qa",
			"commit-red",
			"commit-red-qa",
			"tdd-green",         // Continue forward
			"tdd-green-qa",
			"commit-green",
			"commit-green-qa",
			"tdd-refactor",
			"tdd-refactor-qa",
			"commit-refactor",
			"commit-refactor-qa",
			"task-complete",
		}

		for i := 0; i < len(phases)-1; i++ {
			from := phases[i]
			to := phases[i+1]
			g.Expect(state.IsLegalTransition(from, to)).To(BeTrue(),
				"transition %s → %s should be legal", from, to)
		}
	})

	t.Run("TDD cycle with improvement iterations in all phases", func(t *testing.T) {
		g := NewWithT(t)

		// Each phase with one improvement loop
		phases := []string{
			"task-start",
			"tdd-red",
			"tdd-red-qa",
			"commit-red",
			"commit-red-qa",
			"tdd-red",           // Red improvement loop
			"tdd-red-qa",
			"commit-red",
			"commit-red-qa",
			"tdd-green",
			"tdd-green-qa",
			"commit-green",
			"commit-green-qa",
			"tdd-green",         // Green improvement loop
			"tdd-green-qa",
			"commit-green",
			"commit-green-qa",
			"tdd-refactor",
			"tdd-refactor-qa",
			"commit-refactor",
			"commit-refactor-qa",
			"tdd-refactor",      // Refactor improvement loop
			"tdd-refactor-qa",
			"commit-refactor",
			"commit-refactor-qa",
			"task-complete",
		}

		for i := 0; i < len(phases)-1; i++ {
			from := phases[i]
			to := phases[i+1]
			g.Expect(state.IsLegalTransition(from, to)).To(BeTrue(),
				"transition %s → %s should be legal", from, to)
		}
	})
}

func TestTDDTransitionSequentialOrdering(t *testing.T) {
	t.Run("phases maintain sequential ordering", func(t *testing.T) {
		g := NewWithT(t)

		// The main forward path should always be the first target (index 0)
		// This ensures consistency in automation
		targets := state.LegalTargets("commit-red-qa")
		g.Expect(targets).To(HaveLen(2)) // Should have forward and loop-back
		g.Expect(targets[0]).To(Equal("tdd-green")) // Forward is first
		g.Expect(targets[1]).To(Equal("tdd-red"))    // Loop-back is second

		targets = state.LegalTargets("commit-green-qa")
		g.Expect(targets).To(HaveLen(2))
		g.Expect(targets[0]).To(Equal("tdd-refactor"))
		g.Expect(targets[1]).To(Equal("tdd-green"))

		targets = state.LegalTargets("commit-refactor-qa")
		g.Expect(targets).To(HaveLen(4))
		g.Expect(targets).To(ContainElement("task-complete"))
		g.Expect(targets).To(ContainElement("task-retry"))
		g.Expect(targets).To(ContainElement("task-escalated"))
		g.Expect(targets).To(ContainElement("tdd-refactor"))
	})
}

func TestLegalTargetsForTDDPhases(t *testing.T) {
	t.Run("returns correct targets for each TDD phase", func(t *testing.T) {
		g := NewWithT(t)

		tests := []struct {
			phase   string
			targets []string
		}{
			{"tdd-red", []string{"tdd-red-qa"}},
			{"tdd-red-qa", []string{"commit-red"}},
			{"commit-red", []string{"commit-red-qa"}},
			{"commit-red-qa", []string{"tdd-green", "tdd-red"}},
			{"tdd-green", []string{"tdd-green-qa"}},
			{"tdd-green-qa", []string{"commit-green"}},
			{"commit-green", []string{"commit-green-qa"}},
			{"commit-green-qa", []string{"tdd-refactor", "tdd-green"}},
			{"tdd-refactor", []string{"tdd-refactor-qa"}},
			{"tdd-refactor-qa", []string{"commit-refactor"}},
			{"commit-refactor", []string{"commit-refactor-qa"}},
			{"commit-refactor-qa", []string{"task-complete", "task-retry", "task-escalated", "tdd-refactor"}},
		}

		for _, tt := range tests {
			targets := state.LegalTargets(tt.phase)
			g.Expect(targets).To(Equal(tt.targets),
				"LegalTargets(%s) should return %v", tt.phase, tt.targets)
		}
	})
}
