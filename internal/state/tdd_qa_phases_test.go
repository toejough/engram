package state_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
)

// TestPerPhaseQAInTDDLoop verifies ISSUE-92 acceptance criteria:
// Each TDD sub-phase (red, green, refactor) has its own QA phase.
// Traces: ARCH-034, ARCH-036

// navigateToPhase is a helper that transitions through a sequence of phases.
// It fails the test if any transition fails.
func navigateToPhase(t *testing.T, dir string, targetPhase string) {
	t.Helper()
	g := NewWithT(t)

	// Define the full phase sequence from init to tdd-refactor-qa
	allPhases := []string{
		"pm", "pm-complete", "design", "design-complete",
		"architect", "architect-complete", "breakdown", "breakdown-complete",
		"implementation", "task-start", "tdd-red", "tdd-red-qa",
		"tdd-green", "tdd-green-qa",
		"tdd-refactor", "tdd-refactor-qa",
	}

	// Find the index of the target phase
	targetIdx := -1
	for i, phase := range allPhases {
		if phase == targetPhase {
			targetIdx = i
			break
		}
	}

	if targetIdx == -1 {
		t.Fatalf("target phase %s not found in sequence", targetPhase)
	}

	// Transition through phases up to and including the target
	for i := 0; i <= targetIdx; i++ {
		_, err := state.Transition(dir, allPhases[i], state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred(), "failed to transition to %s", allPhases[i])
	}
}

// Traces: ARCH-036
func TestTDDRedToRedQATransition(t *testing.T) {
	t.Run("tdd-red can transition to tdd-red-qa", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		navigateToPhase(t, dir, "tdd-red")

		// Test: tdd-red -> tdd-red-qa transition
		s, err := state.Transition(dir, "tdd-red-qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd-red-qa"))
	})
}

// Traces: ARCH-036
func TestTDDRedQAToTDDGreenTransition(t *testing.T) {
	t.Run("tdd-red-qa can transition to tdd-green", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		navigateToPhase(t, dir, "tdd-red-qa")

		// Test: tdd-red-qa -> tdd-green transition
		s, err := state.Transition(dir, "tdd-green", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd-green"))
	})
}

// Traces: ARCH-036
func TestTDDGreenToGreenQATransition(t *testing.T) {
	t.Run("tdd-green can transition to tdd-green-qa", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		navigateToPhase(t, dir, "tdd-green")

		// Test: tdd-green -> tdd-green-qa transition
		s, err := state.Transition(dir, "tdd-green-qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd-green-qa"))
	})
}

// Traces: ARCH-036
func TestTDDGreenQAToTDDRefactorTransition(t *testing.T) {
	t.Run("tdd-green-qa can transition to tdd-refactor", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		navigateToPhase(t, dir, "tdd-green-qa")

		// Test: tdd-green-qa -> tdd-refactor transition
		s, err := state.Transition(dir, "tdd-refactor", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd-refactor"))
	})
}

// Traces: ARCH-036
func TestTDDRefactorToRefactorQATransition(t *testing.T) {
	t.Run("tdd-refactor can transition to tdd-refactor-qa", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		navigateToPhase(t, dir, "tdd-refactor")

		// Test: tdd-refactor -> tdd-refactor-qa transition
		s, err := state.Transition(dir, "tdd-refactor-qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd-refactor-qa"))
	})
}

// Traces: ARCH-036
func TestTDDRefactorQAToTaskCompleteTransition(t *testing.T) {
	t.Run("tdd-refactor-qa can transition to task-complete", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		navigateToPhase(t, dir, "tdd-refactor-qa")

		// Test: tdd-refactor-qa -> task-complete transition
		s, err := state.Transition(dir, "task-complete", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("task-complete"))
	})
}

// Traces: ARCH-036
func TestLegalTargetsWithPerPhaseQA(t *testing.T) {
	t.Run("tdd-red legal targets include tdd-red-qa", func(t *testing.T) {
		g := NewWithT(t)

		targets := state.LegalTargets("tdd-red")
		g.Expect(targets).To(ContainElement("tdd-red-qa"))
	})

	t.Run("tdd-red-qa legal targets include tdd-green and tdd-red", func(t *testing.T) {
		g := NewWithT(t)

		targets := state.LegalTargets("tdd-red-qa")
		g.Expect(targets).To(ContainElement("tdd-green"))
		g.Expect(targets).To(ContainElement("tdd-red"))
	})

	t.Run("tdd-green legal targets include tdd-green-qa", func(t *testing.T) {
		g := NewWithT(t)

		targets := state.LegalTargets("tdd-green")
		g.Expect(targets).To(ContainElement("tdd-green-qa"))
	})

	t.Run("tdd-green-qa legal targets include tdd-refactor and tdd-green", func(t *testing.T) {
		g := NewWithT(t)

		targets := state.LegalTargets("tdd-green-qa")
		g.Expect(targets).To(ContainElement("tdd-refactor"))
		g.Expect(targets).To(ContainElement("tdd-green"))
	})

	t.Run("tdd-refactor legal targets include tdd-refactor-qa", func(t *testing.T) {
		g := NewWithT(t)

		targets := state.LegalTargets("tdd-refactor")
		g.Expect(targets).To(ContainElement("tdd-refactor-qa"))
	})

	t.Run("tdd-refactor-qa legal targets include task-complete", func(t *testing.T) {
		g := NewWithT(t)

		targets := state.LegalTargets("tdd-refactor-qa")
		g.Expect(targets).To(ContainElement("task-complete"))
		g.Expect(targets).To(ContainElement("task-retry"))
		g.Expect(targets).To(ContainElement("task-escalated"))
		g.Expect(targets).To(ContainElement("tdd-refactor"))
	})
}
