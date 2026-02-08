package state_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
)

// TestFullTDDCycleWithPerPhaseQA verifies the full TDD loop with per-phase QA
// executes correctly using the flat state machine.

func TestFullTDDCycleWithPerPhaseQA(t *testing.T) {
	t.Run("complete TDD cycle through all QA phases", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd_red_produce via BFS
		navigateToState(t, dir, "tdd_red_produce", "new")

		// Full TDD cycle: red → green → refactor → commit
		fullSequence := []string{
			"tdd_red_qa",
			"tdd_red_decide",
			"tdd_green_produce",
			"tdd_green_qa",
			"tdd_green_decide",
			"tdd_refactor_produce",
			"tdd_refactor_qa",
			"tdd_refactor_decide",
			"tdd_commit",
		}

		for _, phase := range fullSequence {
			s, err := state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred(), "transition to %s failed", phase)
			g.Expect(s.Project.Phase).To(Equal(phase), "state should reflect phase %s", phase)
		}
	})

	t.Run("no shortcuts allowed in full TDD cycle", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd_red_produce
		navigateToState(t, dir, "tdd_red_produce", "new")

		// 1. Cannot skip from tdd_red_produce to tdd_green_produce (must go through qa/decide)
		_, err = state.Transition(dir, "tdd_green_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("illegal transition"))

		// 2. Cannot skip from tdd_green_produce to tdd_refactor_produce
		dir2 := t.TempDir()
		_, err = state.Init(dir2, "test-project", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())
		navigateToState(t, dir2, "tdd_green_produce", "new")

		_, err = state.Transition(dir2, "tdd_refactor_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("illegal transition"))

		// 3. Cannot skip from tdd_refactor_produce to tdd_commit
		dir3 := t.TempDir()
		_, err = state.Init(dir3, "test-project", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())
		navigateToState(t, dir3, "tdd_refactor_produce", "new")

		_, err = state.Transition(dir3, "tdd_commit", state.TransitionOpts{}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("illegal transition"))
	})

	t.Run("output shows full phase progression", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		navigateToState(t, dir, "tdd_red_produce", "new")

		progression := []string{"tdd_red_produce"}

		fullSequence := []string{
			"tdd_red_qa",
			"tdd_red_decide",
			"tdd_green_produce",
			"tdd_green_qa",
			"tdd_green_decide",
			"tdd_refactor_produce",
			"tdd_refactor_qa",
			"tdd_refactor_decide",
			"tdd_commit",
		}

		for _, phase := range fullSequence {
			s, err := state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
			progression = append(progression, s.Project.Phase)
		}

		expectedProgression := []string{
			"tdd_red_produce",
			"tdd_red_qa",
			"tdd_red_decide",
			"tdd_green_produce",
			"tdd_green_qa",
			"tdd_green_decide",
			"tdd_refactor_produce",
			"tdd_refactor_qa",
			"tdd_refactor_decide",
			"tdd_commit",
		}

		g.Expect(progression).To(Equal(expectedProgression))
	})
}
