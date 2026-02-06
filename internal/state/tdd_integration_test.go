package state_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
	"github.com/toejough/projctl/internal/step"
)

// TestFullTDDCycleWithPerPhaseQA verifies TASK-22 acceptance criteria:
// Full TDD loop with per-phase QA executes correctly from tdd-red through task-audit.

func TestFullTDDCycleWithPerPhaseQA(t *testing.T) {
	t.Run("complete TDD cycle from tdd-red to task-audit with QA phases", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Initialize and navigate to tdd-red
		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd-red phase
		phases := []string{
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red",
		}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred(), "failed to transition to %s", phase)
		}

		// Full sequence: tdd-red → tdd-red-qa → commit-red → commit-red-qa →
		// tdd-green → tdd-green-qa → commit-green → commit-green-qa →
		// tdd-refactor → tdd-refactor-qa → commit-refactor → commit-refactor-qa →
		// task-audit
		fullSequence := []string{
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
			"task-audit",
		}

		for _, phase := range fullSequence {
			s, err := state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred(), "transition to %s failed", phase)
			g.Expect(s.Project.Phase).To(Equal(phase), "state should reflect phase %s", phase)
		}
	})

	t.Run("step next returns correct actions at each phase", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd-red
		phases := []string{
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red",
		}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Test step next at tdd-red-qa returns spawn-qa
		_, err = state.Transition(dir, "tdd-red-qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"), "tdd-red-qa should spawn QA")
		g.Expect(result.Skill).To(Equal("qa"))

		// Test step next at commit-red-qa returns spawn-qa
		_, err = state.Transition(dir, "commit-red", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "commit-red-qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"), "commit-red-qa should spawn QA")

		// Test step next at tdd-green-qa returns spawn-qa
		_, err = state.Transition(dir, "tdd-green", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tdd-green-qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"), "tdd-green-qa should spawn QA")

		// Test step next at commit-green-qa returns spawn-qa
		_, err = state.Transition(dir, "commit-green", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "commit-green-qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"), "commit-green-qa should spawn QA")

		// Test step next at tdd-refactor-qa returns spawn-qa
		_, err = state.Transition(dir, "tdd-refactor", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tdd-refactor-qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"), "tdd-refactor-qa should spawn QA")

		// Test step next at commit-refactor-qa returns spawn-qa
		_, err = state.Transition(dir, "commit-refactor", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "commit-refactor-qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"), "commit-refactor-qa should spawn QA")
	})

	t.Run("no shortcuts allowed in full TDD cycle", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd-red
		phases := []string{
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red",
		}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Verify illegal shortcuts are blocked:

		// 1. Cannot skip from tdd-red-qa to tdd-green (must go through commit-red and commit-red-qa)
		_, err = state.Transition(dir, "tdd-red-qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "tdd-green", state.TransitionOpts{}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("illegal transition"))

		// 2. Cannot skip from commit-red to tdd-green (must go through commit-red-qa)
		dir2 := t.TempDir()
		_, err = state.Init(dir2, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range append(phases, "tdd-red-qa", "commit-red") {
			_, err = state.Transition(dir2, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		_, err = state.Transition(dir2, "tdd-green", state.TransitionOpts{}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("illegal transition"))

		// 3. Cannot skip from tdd-green-qa to tdd-refactor (must go through commit-green and commit-green-qa)
		dir3 := t.TempDir()
		_, err = state.Init(dir3, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range append(phases, "tdd-red-qa", "commit-red", "commit-red-qa", "tdd-green", "tdd-green-qa") {
			_, err = state.Transition(dir3, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		_, err = state.Transition(dir3, "tdd-refactor", state.TransitionOpts{}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("illegal transition"))

		// 4. Cannot skip from commit-refactor to task-audit (must go through commit-refactor-qa)
		dir4 := t.TempDir()
		_, err = state.Init(dir4, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range append(phases, "tdd-red-qa", "commit-red", "commit-red-qa",
			"tdd-green", "tdd-green-qa", "commit-green", "commit-green-qa",
			"tdd-refactor", "tdd-refactor-qa", "commit-refactor") {
			_, err = state.Transition(dir4, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		_, err = state.Transition(dir4, "task-audit", state.TransitionOpts{}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("illegal transition"))
	})

	t.Run("output shows full phase progression", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd-red
		phases := []string{
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red",
		}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Track phase progression
		progression := []string{"tdd-red"}

		fullSequence := []string{
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
			"task-audit",
		}

		for _, phase := range fullSequence {
			s, err := state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
			progression = append(progression, s.Project.Phase)
		}

		// Verify progression includes all expected phases in order
		expectedProgression := []string{
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
			"task-audit",
		}

		g.Expect(progression).To(Equal(expectedProgression))
	})
}
