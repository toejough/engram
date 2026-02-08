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
	t.Run("returns spawn-producer action for tdd_red_produce phase", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "scoped",
			Issue:    "ISSUE-105",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd_red_produce via scoped workflow
		for _, phase := range []string{"tasklist_create", "item_select", "item_fork", "worktree_create", "tdd_red_produce"} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Skill).To(Equal("tdd-red-producer"))
		g.Expect(result.SkillPath).To(Equal("skills/tdd-red-producer/SKILL.md"))
		g.Expect(result.Model).ToNot(BeEmpty())
		g.Expect(result.Phase).To(Equal("tdd_red_produce"))
		g.Expect(result.Context.Issue).To(Equal("ISSUE-105"))
		g.Expect(result.TaskParams).ToNot(BeNil())
		g.Expect(result.TaskParams.SubagentType).To(Equal("general-purpose"))
		g.Expect(result.TaskParams.Model).To(Equal(result.Model))
	})
}

func TestStepNextTDDRedQAAction(t *testing.T) {
	t.Run("returns spawn-qa action at tdd_red_qa state", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "scoped",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd_red_qa via scoped workflow
		for _, phase := range []string{"tasklist_create", "item_select", "item_fork", "worktree_create", "tdd_red_produce", "tdd_red_qa"} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))
		g.Expect(result.Skill).To(Equal("qa"))
		g.Expect(result.Model).To(Equal("haiku"))
	})
}

func TestStepNextTDDRedProduceCompleteTransitions(t *testing.T) {
	t.Run("returns transition to tdd_red_qa when producer complete at tdd_red_produce", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "scoped",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd_red_produce
		for _, phase := range []string{"tasklist_create", "item_select", "item_fork", "worktree_create", "tdd_red_produce"} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Mark producer complete
		_, err = state.SetPair(dir, "tdd_red", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tdd_red_qa"))
	})
}

func TestStepNextCommittedAction(t *testing.T) {
	t.Run("returns transition to item_select after breakdown_commit committed in new workflow", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "new",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to breakdown_commit via new workflow
		for _, phase := range []string{
			"tasklist_create", "plan_produce", "plan_approve",
			"artifact_fork", "artifact_pm_produce", "artifact_join",
			"crosscut_qa", "crosscut_decide", "artifact_commit",
			"breakdown_produce", "breakdown_qa", "breakdown_decide", "breakdown_commit",
		} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Mark as committed
		_, err = state.SetPair(dir, "breakdown", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "committed",
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("item_select"))
	})
}
