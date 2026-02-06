package state_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
)

// TestPerPhaseQAInTDDLoop verifies ISSUE-92 acceptance criteria:
// Each TDD sub-phase (red, green, refactor) has its own QA phase.

func TestTDDRedToRedQATransition(t *testing.T) {
	t.Run("tdd-red can transition to tdd-red-qa", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd-red phase
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm-complete", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "design", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "design-complete", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "architect", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "architect-complete", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "breakdown", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "breakdown-complete", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "implementation", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "task-start", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tdd-red", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Test: tdd-red → tdd-red-qa transition
		s, err := state.Transition(dir, "tdd-red-qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd-red-qa"))
	})

	t.Run("tdd-red cannot skip to commit-red", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd-red phase
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm-complete", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "design", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "design-complete", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "architect", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "architect-complete", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "breakdown", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "breakdown-complete", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "implementation", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "task-start", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tdd-red", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Test: tdd-red → commit-red should be illegal (must go through tdd-red-qa)
		_, err = state.Transition(dir, "commit-red", state.TransitionOpts{}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("illegal transition"))
	})
}

func TestTDDRedQAToCommitRedTransition(t *testing.T) {
	t.Run("tdd-red-qa can transition to commit-red", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd-red-qa phase
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm-complete", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "design", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "design-complete", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "architect", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "architect-complete", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "breakdown", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "breakdown-complete", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "implementation", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "task-start", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tdd-red", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tdd-red-qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Test: tdd-red-qa → commit-red transition
		s, err := state.Transition(dir, "commit-red", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("commit-red"))
	})
}

func TestTDDGreenToGreenQATransition(t *testing.T) {
	t.Run("tdd-green can transition to tdd-green-qa", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd-green phase (via full chain)
		phases := []string{
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red", "tdd-red-qa",
			"commit-red", "commit-red-qa", "tdd-green",
		}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Test: tdd-green → tdd-green-qa transition
		s, err := state.Transition(dir, "tdd-green-qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd-green-qa"))
	})

	t.Run("tdd-green cannot skip to commit-green", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd-green phase
		phases := []string{
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red", "tdd-red-qa",
			"commit-red", "commit-red-qa", "tdd-green",
		}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Test: tdd-green → commit-green should be illegal
		_, err = state.Transition(dir, "commit-green", state.TransitionOpts{}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("illegal transition"))
	})
}

func TestTDDGreenQAToCommitGreenTransition(t *testing.T) {
	t.Run("tdd-green-qa can transition to commit-green", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd-green-qa phase
		phases := []string{
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red", "tdd-red-qa",
			"commit-red", "commit-red-qa", "tdd-green", "tdd-green-qa",
		}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Test: tdd-green-qa → commit-green transition
		s, err := state.Transition(dir, "commit-green", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("commit-green"))
	})
}

func TestTDDRefactorToRefactorQATransition(t *testing.T) {
	t.Run("tdd-refactor can transition to tdd-refactor-qa", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd-refactor phase
		phases := []string{
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red", "tdd-red-qa",
			"commit-red", "commit-red-qa", "tdd-green", "tdd-green-qa", "commit-green", "commit-green-qa",
			"tdd-refactor",
		}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Test: tdd-refactor → tdd-refactor-qa transition
		s, err := state.Transition(dir, "tdd-refactor-qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd-refactor-qa"))
	})

	t.Run("tdd-refactor cannot skip to commit-refactor", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd-refactor phase
		phases := []string{
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red", "tdd-red-qa",
			"commit-red", "commit-red-qa", "tdd-green", "tdd-green-qa", "commit-green", "commit-green-qa",
			"tdd-refactor",
		}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Test: tdd-refactor → commit-refactor should be illegal
		_, err = state.Transition(dir, "commit-refactor", state.TransitionOpts{}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("illegal transition"))
	})
}

func TestTDDRefactorQAToCommitRefactorTransition(t *testing.T) {
	t.Run("tdd-refactor-qa can transition to commit-refactor", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd-refactor-qa phase
		phases := []string{
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red", "tdd-red-qa",
			"commit-red", "commit-red-qa", "tdd-green", "tdd-green-qa", "commit-green", "commit-green-qa",
			"tdd-refactor", "tdd-refactor-qa",
		}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Test: tdd-refactor-qa → commit-refactor transition
		s, err := state.Transition(dir, "commit-refactor", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("commit-refactor"))
	})
}

func TestCommitPairLoops(t *testing.T) {
	t.Run("commit-red can transition to commit-red-qa", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to commit-red phase
		phases := []string{
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red", "tdd-red-qa",
			"commit-red",
		}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Test: commit-red → commit-red-qa transition
		s, err := state.Transition(dir, "commit-red-qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("commit-red-qa"))
	})

	t.Run("commit-red-qa can transition to tdd-green", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to commit-red-qa phase
		phases := []string{
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red", "tdd-red-qa",
			"commit-red", "commit-red-qa",
		}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Test: commit-red-qa → tdd-green transition
		s, err := state.Transition(dir, "tdd-green", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd-green"))
	})

	t.Run("commit-green can transition to commit-green-qa", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to commit-green phase
		phases := []string{
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red", "tdd-red-qa",
			"commit-red", "commit-red-qa", "tdd-green", "tdd-green-qa",
			"commit-green",
		}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Test: commit-green → commit-green-qa transition
		s, err := state.Transition(dir, "commit-green-qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("commit-green-qa"))
	})

	t.Run("commit-green-qa can transition to tdd-refactor", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to commit-green-qa phase
		phases := []string{
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red", "tdd-red-qa",
			"commit-red", "commit-red-qa", "tdd-green", "tdd-green-qa",
			"commit-green", "commit-green-qa",
		}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Test: commit-green-qa → tdd-refactor transition
		s, err := state.Transition(dir, "tdd-refactor", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd-refactor"))
	})

	t.Run("commit-refactor can transition to commit-refactor-qa", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to commit-refactor phase
		phases := []string{
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red", "tdd-red-qa",
			"commit-red", "commit-red-qa", "tdd-green", "tdd-green-qa",
			"commit-green", "commit-green-qa", "tdd-refactor", "tdd-refactor-qa",
			"commit-refactor",
		}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Test: commit-refactor → commit-refactor-qa transition
		s, err := state.Transition(dir, "commit-refactor-qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("commit-refactor-qa"))
	})

	t.Run("commit-refactor-qa can transition to task-audit", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to commit-refactor-qa phase
		phases := []string{
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red", "tdd-red-qa",
			"commit-red", "commit-red-qa", "tdd-green", "tdd-green-qa",
			"commit-green", "commit-green-qa", "tdd-refactor", "tdd-refactor-qa",
			"commit-refactor", "commit-refactor-qa",
		}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Test: commit-refactor-qa → task-audit transition
		s, err := state.Transition(dir, "task-audit", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("task-audit"))
	})
}

func TestLegalTargetsWithPerPhaseQA(t *testing.T) {
	t.Run("tdd-red legal targets include tdd-red-qa", func(t *testing.T) {
		g := NewWithT(t)

		targets := state.LegalTargets("tdd-red")
		g.Expect(targets).To(ContainElement("tdd-red-qa"))
		g.Expect(targets).ToNot(ContainElement("commit-red"))
	})

	t.Run("tdd-red-qa legal targets include commit-red", func(t *testing.T) {
		g := NewWithT(t)

		targets := state.LegalTargets("tdd-red-qa")
		g.Expect(targets).To(ContainElement("commit-red"))
	})

	t.Run("commit-red legal targets include commit-red-qa", func(t *testing.T) {
		g := NewWithT(t)

		targets := state.LegalTargets("commit-red")
		g.Expect(targets).To(ContainElement("commit-red-qa"))
		g.Expect(targets).ToNot(ContainElement("tdd-green"))
	})

	t.Run("commit-red-qa legal targets include tdd-green", func(t *testing.T) {
		g := NewWithT(t)

		targets := state.LegalTargets("commit-red-qa")
		g.Expect(targets).To(ContainElement("tdd-green"))
	})

	t.Run("tdd-green legal targets include tdd-green-qa", func(t *testing.T) {
		g := NewWithT(t)

		targets := state.LegalTargets("tdd-green")
		g.Expect(targets).To(ContainElement("tdd-green-qa"))
		g.Expect(targets).ToNot(ContainElement("commit-green"))
	})

	t.Run("tdd-green-qa legal targets include commit-green", func(t *testing.T) {
		g := NewWithT(t)

		targets := state.LegalTargets("tdd-green-qa")
		g.Expect(targets).To(ContainElement("commit-green"))
	})

	t.Run("commit-green legal targets include commit-green-qa", func(t *testing.T) {
		g := NewWithT(t)

		targets := state.LegalTargets("commit-green")
		g.Expect(targets).To(ContainElement("commit-green-qa"))
		g.Expect(targets).ToNot(ContainElement("tdd-refactor"))
	})

	t.Run("commit-green-qa legal targets include tdd-refactor", func(t *testing.T) {
		g := NewWithT(t)

		targets := state.LegalTargets("commit-green-qa")
		g.Expect(targets).To(ContainElement("tdd-refactor"))
	})

	t.Run("tdd-refactor legal targets include tdd-refactor-qa", func(t *testing.T) {
		g := NewWithT(t)

		targets := state.LegalTargets("tdd-refactor")
		g.Expect(targets).To(ContainElement("tdd-refactor-qa"))
		g.Expect(targets).ToNot(ContainElement("commit-refactor"))
	})

	t.Run("tdd-refactor-qa legal targets include commit-refactor", func(t *testing.T) {
		g := NewWithT(t)

		targets := state.LegalTargets("tdd-refactor-qa")
		g.Expect(targets).To(ContainElement("commit-refactor"))
	})

	t.Run("commit-refactor legal targets include commit-refactor-qa", func(t *testing.T) {
		g := NewWithT(t)

		targets := state.LegalTargets("commit-refactor")
		g.Expect(targets).To(ContainElement("commit-refactor-qa"))
		g.Expect(targets).ToNot(ContainElement("task-audit"))
	})

	t.Run("commit-refactor-qa legal targets include task-audit", func(t *testing.T) {
		g := NewWithT(t)

		targets := state.LegalTargets("commit-refactor-qa")
		g.Expect(targets).To(ContainElement("task-audit"))
	})
}
