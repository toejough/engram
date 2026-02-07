package step_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
	"github.com/toejough/projctl/internal/step"
)

// TestStepNextActionGeneration verifies TASK-6/TASK-13 acceptance criteria:
// step next returns correct actions for all TDD sub-phases.

func TestStepNextTDDRedProducerAction(t *testing.T) {
	t.Run("returns spawn-producer action for tdd-red phase", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "task",
			Issue:    "ISSUE-105",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd-red
		_, err = state.Transition(dir, "task-implementation", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "task-start", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tdd-red", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Skill).To(Equal("tdd-red-producer"))
		g.Expect(result.SkillPath).To(Equal("skills/tdd-red-producer/SKILL.md"))
		g.Expect(result.Model).To(Equal("sonnet"))
		g.Expect(result.Phase).To(Equal("tdd-red"))
		g.Expect(result.Context.Issue).To(Equal("ISSUE-105"))
		g.Expect(result.TaskParams).ToNot(BeNil())
		g.Expect(result.TaskParams.SubagentType).To(Equal("general-purpose"))
		g.Expect(result.TaskParams.Model).To(Equal("sonnet"))
	})
}

func TestStepNextTDDRedQAAction(t *testing.T) {
	t.Run("returns spawn-qa action after tdd-red producer completes", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "task",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "task-implementation", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "task-start", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tdd-red", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Mark producer complete
		_, err = state.SetPair(dir, "tdd-red", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))
		g.Expect(result.Skill).To(Equal("qa"))
		g.Expect(result.Model).To(Equal("haiku"))
	})
}

func TestStepNextCommitRedAction(t *testing.T) {
	t.Run("returns transition to tdd-red-qa after tdd-red committed", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "task",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "task-implementation", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "task-start", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tdd-red", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Mark as committed
		_, err = state.SetPair(dir, "tdd-red", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "committed",
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		// Fixed: Use state graph targets[0] instead of registry CompletionPhase
		g.Expect(result.Phase).To(Equal("tdd-red-qa"))
	})
}

func TestStepNextCommitRedProducerAction(t *testing.T) {
	t.Run("returns spawn-producer for commit-red phase", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "task",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "task-implementation", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "task-start", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tdd-red", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tdd-red-qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "commit-red", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Skill).To(Equal("commit-producer"))
		g.Expect(result.SkillPath).To(Equal("skills/commit-producer/SKILL.md"))
		g.Expect(result.Model).To(Equal("haiku"))
		g.Expect(result.Phase).To(Equal("commit-red"))
	})
}

// TestStepNextAllCompleteAction tests terminal phase behavior.
// Note: This is implicitly tested by the existing Next tests which check
// for len(targets) == 0 in the step.Next implementation.
