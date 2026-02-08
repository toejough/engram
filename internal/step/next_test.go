package step_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
	"github.com/toejough/projctl/internal/step"
	"pgregory.net/rapid"
)

func fixedTime() time.Time {
	return time.Date(2026, 1, 27, 12, 0, 0, 0, time.UTC)
}

func nowFunc() func() time.Time {
	return func() time.Time { return fixedTime() }
}

func TestNext(t *testing.T) {
	t.Run("returns spawn-producer for plan_produce phase", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Skill).To(Equal("plan-producer"))
		g.Expect(result.SkillPath).To(Equal("skills/plan-producer/SKILL.md"))
		g.Expect(result.Model).ToNot(BeEmpty())
		g.Expect(result.Context.Issue).To(Equal("ISSUE-89"))
	})

	t.Run("returns transition to plan_approve when producer complete at plan_produce", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.SetPair(dir, "plan", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("plan_approve"))
	})

	t.Run("returns spawn-qa at crosscut_qa state", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{
			"tasklist_create", "plan_produce", "plan_approve",
			"artifact_fork", "artifact_pm_produce", "artifact_join",
			"crosscut_qa",
		} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))
		g.Expect(result.Skill).To(Equal("qa"))
		g.Expect(result.SkillPath).To(Equal("skills/qa/SKILL.md"))
		g.Expect(result.Model).To(Equal("haiku"))
	})

	t.Run("returns commit at artifact_commit when not yet committed", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{
			"tasklist_create", "plan_produce", "plan_approve",
			"artifact_fork", "artifact_pm_produce", "artifact_join",
			"crosscut_qa", "crosscut_decide", "artifact_commit",
		} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Set approved verdict (decide routed here because of approved)
		_, err = state.SetPair(dir, "artifact", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "approved",
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("commit"))
	})

	t.Run("returns spawn-producer again after qa improvement-request", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// After improvement-request: ProducerComplete=false, ImprovementRequest set
		_, err = state.SetPair(dir, "plan", state.PairState{
			Iteration:          2,
			MaxIterations:      3,
			ProducerComplete:   false,
			QAVerdict:          "improvement-request",
			ImprovementRequest: "REQ-003 needs measurable criteria",
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Context.QAFeedback).To(Equal("REQ-003 needs measurable criteria"))
	})

	t.Run("returns transition after artifact_commit committed", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{
			"tasklist_create", "plan_produce", "plan_approve",
			"artifact_fork", "artifact_pm_produce", "artifact_join",
			"crosscut_qa", "crosscut_decide", "artifact_commit",
		} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		_, err = state.SetPair(dir, "artifact", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "committed",
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("breakdown_produce"))
	})

	t.Run("errors when no state file", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := step.Next(dir)
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("returns all-complete for terminal phase", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Walk to complete via the new workflow flat state machine
		phases := []string{
			"tasklist_create", "plan_produce", "plan_approve",
			"artifact_fork", "artifact_pm_produce", "artifact_join",
			"crosscut_qa", "crosscut_decide", "artifact_commit",
			"breakdown_produce", "breakdown_qa", "breakdown_decide", "breakdown_commit",
			"item_select", "item_fork", "worktree_create",
			"tdd_red_produce", "tdd_red_qa", "tdd_red_decide", "tdd_green_produce",
			"tdd_green_qa", "tdd_green_decide", "tdd_refactor_produce",
			"tdd_refactor_qa", "tdd_refactor_decide", "tdd_commit",
			"merge_acquire", "rebase", "merge", "worktree_cleanup", "item_join",
			"item_assess", "items_done",
			"documentation_produce", "documentation_qa", "documentation_decide", "documentation_commit",
			"alignment_produce", "alignment_qa", "alignment_decide", "alignment_commit",
			"evaluation_produce", "evaluation_interview", "evaluation_commit",
			"issue_update", "next_steps", "complete",
		}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred(), "failed to transition to %s", phase)
		}

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("all-complete"))
	})

	t.Run("returns transition for non-registered default state type", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "scoped",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// item_select is a "select" type state — default handler returns transition
		_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "item_select", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("item_fork"))
	})

	t.Run("scoped workflow returns spawn-producer for tdd_red_produce", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "scoped",
			Issue:    "ISSUE-42",
		})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{"tasklist_create", "item_select", "item_fork", "worktree_create", "tdd_red_produce"} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Skill).To(Equal("tdd-red-producer"))
		g.Expect(result.Model).ToNot(BeEmpty())
	})

	t.Run("init phase returns transition to workflow init_state (ISSUE-155)", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "scoped",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// State starts in "init" phase
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("init"))

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())

		// The init phase should transition to the workflow's init_state (tasklist_create)
		// not return all-complete (ISSUE-155: fix)
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tasklist_create"))
	})
}

func TestComplete(t *testing.T) {
	t.Run("producer completion updates pair state", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		err = step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs).To(HaveKey("plan"))
		g.Expect(s.Pairs["plan"].ProducerComplete).To(BeTrue())
	})

	t.Run("records qa verdict", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{
			"tasklist_create", "plan_produce", "plan_approve",
			"artifact_fork", "artifact_pm_produce", "artifact_join",
			"crosscut_qa",
		} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		err = step.Complete(dir, step.CompleteResult{
			Action:    "spawn-qa",
			Status:    "done",
			QAVerdict: "approved",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["crosscut"].QAVerdict).To(Equal("approved"))
	})

	t.Run("records commit and marks committed", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{
			"tasklist_create", "plan_produce", "plan_approve",
			"artifact_fork", "artifact_pm_produce", "artifact_join",
			"crosscut_qa", "crosscut_decide", "artifact_commit",
		} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		_, err = state.SetPair(dir, "artifact", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "approved",
		})
		g.Expect(err).ToNot(HaveOccurred())

		err = step.Complete(dir, step.CompleteResult{
			Action: "commit",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["artifact"].QAVerdict).To(Equal("committed"))
	})

	t.Run("transition clears pair on cross-group boundary", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{
			"tasklist_create", "plan_produce", "plan_approve",
			"artifact_fork", "artifact_pm_produce", "artifact_join",
			"crosscut_qa", "crosscut_decide", "artifact_commit",
		} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		_, err = state.SetPair(dir, "artifact", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "committed",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Transition from artifact_commit to breakdown_produce (cross-group: "artifact" -> "breakdown")
		err = step.Complete(dir, step.CompleteResult{
			Action: "transition",
			Phase:  "breakdown_produce",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("breakdown_produce"))
		g.Expect(s.Pairs).ToNot(HaveKey("artifact"))
	})

	t.Run("transition preserves pair within same group", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{
			"tasklist_create", "plan_produce", "plan_approve",
			"artifact_fork", "artifact_pm_produce", "artifact_join",
			"crosscut_qa",
		} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		_, err = state.SetPair(dir, "crosscut", state.PairState{
			Iteration:        2,
			MaxIterations:    3,
			ProducerComplete: true,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Transition from crosscut_qa to crosscut_decide (same group: "crosscut" -> "crosscut")
		err = step.Complete(dir, step.CompleteResult{
			Action: "transition",
			Phase:  "crosscut_decide",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("crosscut_decide"))
		g.Expect(s.Pairs).To(HaveKey("crosscut"))
		g.Expect(s.Pairs["crosscut"].Iteration).To(Equal(2))
	})

	t.Run("errors on invalid action", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		err = step.Complete(dir, step.CompleteResult{
			Action: "invalid-action",
			Status: "done",
		}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("unknown action"))
	})

	t.Run("commit rejects improvement-request verdict", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{
			"tasklist_create", "plan_produce", "plan_approve",
			"artifact_fork", "artifact_pm_produce", "artifact_join",
			"crosscut_qa", "crosscut_decide", "artifact_commit",
		} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		_, err = state.SetPair(dir, "artifact", state.PairState{
			Iteration:          1,
			MaxIterations:      3,
			ProducerComplete:   true,
			QAVerdict:          "improvement-request",
			ImprovementRequest: "Needs more tests",
		})
		g.Expect(err).ToNot(HaveOccurred())

		err = step.Complete(dir, step.CompleteResult{
			Action: "commit",
			Status: "done",
		}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("QA"))
	})
}

func TestNextTaskParams(t *testing.T) {
	t.Run("spawn-producer populates TaskParams with correct fields", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.TaskParams).ToNot(BeNil())
		g.Expect(result.TaskParams.SubagentType).To(Equal("general-purpose"))
		g.Expect(result.TaskParams.Name).To(Equal("plan-producer"))
		g.Expect(result.TaskParams.Model).To(Equal(result.Model))
		g.Expect(result.TaskParams.Prompt).To(HavePrefix(step.HandshakeInstruction))
		g.Expect(result.ExpectedModel).To(Equal(result.Model))
	})

	t.Run("spawn-qa populates TaskParams with QA fields", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{
			"tasklist_create", "plan_produce", "plan_approve",
			"artifact_fork", "artifact_pm_produce", "artifact_join",
			"crosscut_qa",
		} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))
		g.Expect(result.TaskParams).ToNot(BeNil())
		g.Expect(result.TaskParams.SubagentType).To(Equal("general-purpose"))
		g.Expect(result.TaskParams.Name).To(Equal("qa"))
		g.Expect(result.TaskParams.Model).To(Equal("haiku"))
		g.Expect(result.ExpectedModel).To(Equal("haiku"))
	})

	t.Run("improvement-request re-spawn populates TaskParams", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.SetPair(dir, "plan", state.PairState{
			Iteration:          2,
			MaxIterations:      3,
			ProducerComplete:   false,
			QAVerdict:          "improvement-request",
			ImprovementRequest: "needs work",
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.TaskParams).ToNot(BeNil())
		g.Expect(result.TaskParams.SubagentType).To(Equal("general-purpose"))
		g.Expect(result.TaskParams.Name).To(Equal("plan-producer"))
		g.Expect(result.TaskParams.Model).To(Equal(result.Model))
		g.Expect(result.ExpectedModel).To(Equal(result.Model))
	})

	t.Run("non-spawn actions have nil TaskParams and empty ExpectedModel", func(t *testing.T) {
		g := NewWithT(t)

		// Test commit action
		dir := t.TempDir()
		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-89"})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{
			"tasklist_create", "plan_produce", "plan_approve",
			"artifact_fork", "artifact_pm_produce", "artifact_join",
			"crosscut_qa", "crosscut_decide", "artifact_commit",
		} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}
		_, err = state.SetPair(dir, "artifact", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "approved",
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("commit"))
		g.Expect(result.TaskParams).To(BeNil())
		g.Expect(result.ExpectedModel).To(BeEmpty())
	})

	t.Run("transition action has nil TaskParams", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "scoped",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// item_select is a default-type state that returns transition
		_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "item_select", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.TaskParams).To(BeNil())
		g.Expect(result.ExpectedModel).To(BeEmpty())
	})

	t.Run("all-complete action has nil TaskParams", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		phases := []string{
			"tasklist_create", "plan_produce", "plan_approve",
			"artifact_fork", "artifact_pm_produce", "artifact_join",
			"crosscut_qa", "crosscut_decide", "artifact_commit",
			"breakdown_produce", "breakdown_qa", "breakdown_decide", "breakdown_commit",
			"item_select", "item_fork", "worktree_create",
			"tdd_red_produce", "tdd_red_qa", "tdd_red_decide", "tdd_green_produce",
			"tdd_green_qa", "tdd_green_decide", "tdd_refactor_produce",
			"tdd_refactor_qa", "tdd_refactor_decide", "tdd_commit",
			"merge_acquire", "rebase", "merge", "worktree_cleanup", "item_join",
			"item_assess", "items_done",
			"documentation_produce", "documentation_qa", "documentation_decide", "documentation_commit",
			"alignment_produce", "alignment_qa", "alignment_decide", "alignment_commit",
			"evaluation_produce", "evaluation_interview", "evaluation_commit",
			"issue_update", "next_steps", "complete",
		}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("all-complete"))
		g.Expect(result.TaskParams).To(BeNil())
		g.Expect(result.ExpectedModel).To(BeEmpty())
	})

	t.Run("tdd_red_produce spawn-producer has correct TaskParams", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "scoped",
			Issue:    "ISSUE-42",
		})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{"tasklist_create", "item_select", "item_fork", "worktree_create", "tdd_red_produce"} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.TaskParams).ToNot(BeNil())
		g.Expect(result.TaskParams.SubagentType).To(Equal("general-purpose"))
		g.Expect(result.TaskParams.Name).To(Equal("tdd-red-producer"))
		g.Expect(result.TaskParams.Model).To(Equal(result.Model))
		g.Expect(result.ExpectedModel).To(Equal(result.Model))
	})
}

func TestPromptAssembly(t *testing.T) {
	// Helper to navigate to plan_produce in new workflow
	navigateToPlanProduce := func(g Gomega, dir string) {
		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
	}

	t.Run("prompt starts with HandshakeInstruction", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		navigateToPlanProduce(g, dir)

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.TaskParams).ToNot(BeNil())
		g.Expect(result.TaskParams.Prompt).To(HavePrefix(step.HandshakeInstruction))
	})

	t.Run("prompt contains skill invocation instruction", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		navigateToPlanProduce(g, dir)

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.TaskParams.Prompt).To(ContainSubstring("Then invoke /plan-producer"))
	})

	t.Run("prompt includes issue reference when present", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		navigateToPlanProduce(g, dir)

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.TaskParams.Prompt).To(ContainSubstring("ISSUE-98"))
	})

	t.Run("prompt includes QA feedback for improvement-request", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		navigateToPlanProduce(g, dir)

		_, err := state.SetPair(dir, "plan", state.PairState{
			Iteration:          2,
			MaxIterations:      3,
			ProducerComplete:   false,
			QAVerdict:          "improvement-request",
			ImprovementRequest: "REQ-003 needs measurable criteria",
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.TaskParams.Prompt).To(ContainSubstring("REQ-003 needs measurable criteria"))
	})

	t.Run("prompt does NOT contain handshake for non-spawn actions", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{
			"tasklist_create", "plan_produce", "plan_approve",
			"artifact_fork", "artifact_pm_produce", "artifact_join",
			"crosscut_qa", "crosscut_decide", "artifact_commit",
		} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}
		_, err = state.SetPair(dir, "artifact", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "approved",
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("commit"))
		g.Expect(result.TaskParams).To(BeNil())
	})

	t.Run("spawn-qa prompt contains QA skill invocation", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{
			"tasklist_create", "plan_produce", "plan_approve",
			"artifact_fork", "artifact_pm_produce", "artifact_join",
			"crosscut_qa",
		} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))
		g.Expect(result.TaskParams.Prompt).To(HavePrefix(step.HandshakeInstruction))
		g.Expect(result.TaskParams.Prompt).To(ContainSubstring("Then invoke /qa"))
	})
}

func TestPromptAssemblyWithTask(t *testing.T) {
	t.Run("prompt includes current task when set", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue:    "ISSUE-98",
			Workflow: "scoped",
		})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{"tasklist_create", "item_select", "item_fork", "worktree_create", "tdd_red_produce"} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Set current task
		_, err = state.Set(dir, state.SetOpts{Task: "TASK-3"})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Context.CurrentTask).To(Equal("TASK-3"))
		g.Expect(result.TaskParams.Prompt).To(ContainSubstring("Task: TASK-3"))
	})

	t.Run("prompt omits task when not set", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue:    "ISSUE-98",
			Workflow: "scoped",
		})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{"tasklist_create", "item_select", "item_fork", "worktree_create", "tdd_red_produce"} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Context.CurrentTask).To(BeEmpty())
		g.Expect(result.TaskParams.Prompt).ToNot(ContainSubstring("Task:"))
	})

	t.Run("task context appears in StepContext JSON", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue:    "ISSUE-42",
			Workflow: "scoped",
		})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{"tasklist_create", "item_select", "item_fork", "worktree_create", "tdd_red_produce"} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		_, err = state.Set(dir, state.SetOpts{Task: "TASK-7"})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Context.CurrentTask).To(Equal("TASK-7"))
	})
}

func TestCompleteFailedSpawn(t *testing.T) {
	// Helper to navigate to plan_produce in new workflow
	navigateToPlanProduce := func(g Gomega, dir string) {
		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
	}

	t.Run("failed spawn-producer increments SpawnAttempts and appends FailedModels", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		navigateToPlanProduce(g, dir)

		err := step.Complete(dir, step.CompleteResult{
			Action:        "spawn-producer",
			Status:        "failed",
			ReportedModel: "haiku",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["plan"].SpawnAttempts).To(Equal(1))
		g.Expect(s.Pairs["plan"].FailedModels).To(Equal([]string{"haiku"}))
	})

	t.Run("failed spawn-producer does NOT set ProducerComplete", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		navigateToPlanProduce(g, dir)

		err := step.Complete(dir, step.CompleteResult{
			Action:        "spawn-producer",
			Status:        "failed",
			ReportedModel: "haiku",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["plan"].ProducerComplete).To(BeFalse())
	})

	t.Run("failed spawn-qa does NOT set QAVerdict", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{
			"tasklist_create", "plan_produce", "plan_approve",
			"artifact_fork", "artifact_pm_produce", "artifact_join",
			"crosscut_qa",
		} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		err = step.Complete(dir, step.CompleteResult{
			Action:        "spawn-qa",
			Status:        "failed",
			ReportedModel: "sonnet",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["crosscut"].QAVerdict).To(BeEmpty())
		g.Expect(s.Pairs["crosscut"].SpawnAttempts).To(Equal(1))
		g.Expect(s.Pairs["crosscut"].FailedModels).To(Equal([]string{"sonnet"}))
	})

	t.Run("done spawn-producer resets SpawnAttempts and FailedModels", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		navigateToPlanProduce(g, dir)

		_, err := state.SetPair(dir, "plan", state.PairState{
			SpawnAttempts: 2,
			FailedModels:  []string{"haiku", "opus"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		err = step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["plan"].SpawnAttempts).To(Equal(0))
		g.Expect(s.Pairs["plan"].FailedModels).To(BeNil())
		g.Expect(s.Pairs["plan"].ProducerComplete).To(BeTrue())
	})

	t.Run("empty status works as done path", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		navigateToPlanProduce(g, dir)

		err := step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["plan"].ProducerComplete).To(BeTrue())
		g.Expect(s.Pairs["plan"].SpawnAttempts).To(Equal(0))
	})
}

func TestNextRetryEscalation(t *testing.T) {
	// Helper to navigate to plan_produce in new workflow
	navigateToPlanProduce := func(g Gomega, dir string) {
		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
	}

	t.Run("SpawnAttempts == 0 emits normal spawn", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		navigateToPlanProduce(g, dir)

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Details).To(BeEmpty())
	})

	t.Run("SpawnAttempts == 1 emits same spawn (retry)", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		navigateToPlanProduce(g, dir)

		_, err := state.SetPair(dir, "plan", state.PairState{
			SpawnAttempts: 1,
			FailedModels:  []string{"haiku"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Details).To(BeEmpty())
	})

	t.Run("SpawnAttempts == 2 emits same spawn (retry)", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		navigateToPlanProduce(g, dir)

		_, err := state.SetPair(dir, "plan", state.PairState{
			SpawnAttempts: 2,
			FailedModels:  []string{"haiku", "haiku"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
	})

	t.Run("SpawnAttempts >= 3 emits escalate-user with Details", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		navigateToPlanProduce(g, dir)

		_, err := state.SetPair(dir, "plan", state.PairState{
			SpawnAttempts: 3,
			FailedModels:  []string{"haiku", "haiku", "haiku"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("escalate-user"))
		g.Expect(result.Details).To(ContainSubstring("expected model"))
		g.Expect(result.Details).To(ContainSubstring("haiku"))
	})

	t.Run("SpawnAttempts >= 3 for spawn-qa emits escalate-user", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{
			"tasklist_create", "plan_produce", "plan_approve",
			"artifact_fork", "artifact_pm_produce", "artifact_join",
			"crosscut_qa",
		} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		_, err = state.SetPair(dir, "crosscut", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
			SpawnAttempts:    3,
			FailedModels:     []string{"sonnet", "sonnet", "sonnet"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("escalate-user"))
		g.Expect(result.Details).To(ContainSubstring("haiku"))
		g.Expect(result.Details).To(ContainSubstring("sonnet"))
	})
}

func TestNextResult_JSON(t *testing.T) {
	t.Run("NextResult contains all required fields", func(t *testing.T) {
		g := NewWithT(t)

		result := step.NextResult{
			Action:    "spawn-producer",
			Skill:     "pm-interview-producer",
			SkillPath: "skills/pm-interview-producer/SKILL.md",
			Model:     "opus",
			Artifact:  "requirements.md",
			Phase:     "pm_produce",
			Context: step.StepContext{
				Issue:          "ISSUE-89",
				PriorArtifacts: []string{},
				QAFeedback:     "",
			},
		}

		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Skill).To(Equal("pm-interview-producer"))
		g.Expect(result.Model).ToNot(BeEmpty())
		g.Expect(result.Context.Issue).To(Equal("ISSUE-89"))
	})
}

// reachable returns the set of all phases reachable from the given starting phase
// via the workflow transition graph.
func reachable(from string, transitions map[string][]string) map[string]bool {
	visited := map[string]bool{}
	queue := []string{from}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		for _, target := range transitions[current] {
			if !visited[target] {
				queue = append(queue, target)
			}
		}
	}

	return visited
}

// onAllPathsTo returns true if every path from `from` to `target` must pass through `required`.
// Uses the property: `required` is on all paths from `from` to `target` iff
// `target` is not reachable from `from` when `required` is removed from the graph.
func onAllPathsTo(from, target, required string, transitions map[string][]string) bool {
	if from == required || from == target {
		return true
	}

	// BFS from `from` to `target`, skipping `required`
	visited := map[string]bool{}
	queue := []string{from}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		if current == target {
			return false // Reached target without going through required
		}

		for _, next := range transitions[current] {
			if next != required && !visited[next] {
				queue = append(queue, next)
			}
		}
	}

	return true // Target not reachable without required
}

func TestMainFlowEndingMandatory(t *testing.T) {
	transitions := state.TransitionsForWorkflow("new")

	// Main flow ending phases in order. Every workflow must pass through all of
	// these before reaching "complete".
	mainFlowEnding := []string{
		"alignment_produce", "alignment_commit",
		"evaluation_produce", "evaluation_commit",
		"issue_update", "next_steps",
	}

	// Phases that are part of (or after) the main flow ending sequence.
	mainFlowEndingSet := map[string]bool{}
	for _, p := range mainFlowEnding {
		mainFlowEndingSet[p] = true
	}
	// Also include the intermediate QA/decide/interview states
	for _, p := range []string{
		"alignment_qa", "alignment_decide",
		"evaluation_interview",
	} {
		mainFlowEndingSet[p] = true
	}

	// Collect all phases that feed into the main flow ending but are not part of it.
	var preEndingPhases []string
	for phase, targets := range transitions {
		if len(targets) > 0 && phase != "complete" && !mainFlowEndingSet[phase] {
			preEndingPhases = append(preEndingPhases, phase)
		}
	}

	t.Run("pre-ending phases must traverse all main flow ending phases to reach complete", func(t *testing.T) {
		g := NewWithT(t)

		rapid.Check(t, func(rt *rapid.T) {
			phase := rapid.SampledFrom(preEndingPhases).Draw(rt, "phase")

			reached := reachable(phase, transitions)
			if !reached["complete"] {
				return
			}

			for _, required := range mainFlowEnding {
				g.Expect(onAllPathsTo(phase, "complete", required, transitions)).To(BeTrue(),
					"phase %q can reach 'complete' without passing through %q", phase, required)
			}
		})
	})

	t.Run("within main flow ending every subsequent phase is mandatory", func(t *testing.T) {
		g := NewWithT(t)

		// For each main flow ending phase, all *later* ending phases must be on
		// the path to complete
		for i, phase := range mainFlowEnding {
			for j := i + 1; j < len(mainFlowEnding); j++ {
				later := mainFlowEnding[j]
				g.Expect(onAllPathsTo(phase, "complete", later, transitions)).To(BeTrue(),
					"phase %q can reach 'complete' without passing through later ending phase %q",
					phase, later)
			}
		}
	})

}

// producerTranscriptFile creates a temporary transcript file for testing.
func producerTranscriptFile(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "transcript.txt")
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return f
}
