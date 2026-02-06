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
	t.Run("returns spawn-producer for pm phase with pending sub-phase", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Transition to pm
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Skill).To(Equal("pm-interview-producer"))
		g.Expect(result.SkillPath).To(Equal("skills/pm-interview-producer/SKILL.md"))
		g.Expect(result.Model).To(Equal("sonnet"))
		g.Expect(result.Context.Issue).To(Equal("ISSUE-89"))
	})

	t.Run("returns spawn-qa after producer sub-phase", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Set sub-phase to producer (meaning producer is done)
		_, err = state.SetPair(dir, "pm", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))
		g.Expect(result.Skill).To(Equal("qa"))
		g.Expect(result.SkillPath).To(Equal("skills/qa/SKILL.md"))
		g.Expect(result.Model).To(Equal("haiku"))
	})

	t.Run("returns commit after qa approved", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Set sub-phase to qa approved
		_, err = state.SetPair(dir, "pm", state.PairState{
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

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Set sub-phase to qa improvement-request
		_, err = state.SetPair(dir, "pm", state.PairState{
			Iteration:          1,
			MaxIterations:      3,
			ProducerComplete:   true,
			QAVerdict:          "improvement-request",
			ImprovementRequest: "REQ-003 needs measurable criteria",
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Context.QAFeedback).To(Equal("REQ-003 needs measurable criteria"))
	})

	t.Run("returns transition after commit sub-phase", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create a skills directory structure for artifact path resolution
		skillsDir := filepath.Join(dir, "skills", "pm-interview-producer")
		g.Expect(os.MkdirAll(skillsDir, 0o755)).To(Succeed())

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Simulate: producer done, qa approved, commit needed
		// After commit, step next should say "transition" to advance phase
		_, err = state.SetPair(dir, "pm:committed", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "approved",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Actually use a dedicated state field: set the pair to show committed
		_, err = state.SetPair(dir, "pm", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "committed",
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("pm-complete"))
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

		// Walk to complete
		phases := []string{
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red",
			"commit-red", "tdd-green", "commit-green", "tdd-refactor",
			"commit-refactor", "task-audit", "task-complete",
			"implementation-complete", "documentation", "documentation-complete",
			"alignment", "alignment-complete", "retro", "retro-complete",
			"summary", "summary-complete", "issue-update", "next-steps", "complete",
		}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred(), "failed to transition to %s", phase)
		}

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("all-complete"))
	})

	t.Run("returns transition for non-registered phase", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// pm-complete is not registered (it's a transition marker)
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm-complete", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("design"))
	})

	t.Run("task workflow returns spawn-producer for tdd-red", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "task",
			Issue:    "ISSUE-42",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Walk to tdd-red phase via task workflow
		_, err = state.Transition(dir, "task-implementation", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "task-start", state.TransitionOpts{Task: "TASK-001"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tdd-red", state.TransitionOpts{Task: "TASK-001"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Skill).To(Equal("tdd-red-producer"))
		g.Expect(result.Model).To(Equal("sonnet"))
	})
}

func TestComplete(t *testing.T) {
	t.Run("advances sub-phase from pending to producer started", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Complete the producer step
		err = step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify pair state updated
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs).To(HaveKey("pm"))
		g.Expect(s.Pairs["pm"].ProducerComplete).To(BeTrue())
	})

	t.Run("records qa verdict", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Set producer as complete
		_, err = state.SetPair(dir, "pm", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Complete QA with approval
		err = step.Complete(dir, step.CompleteResult{
			Action:    "spawn-qa",
			Status:    "done",
			QAVerdict: "approved",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify pair state updated
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["pm"].QAVerdict).To(Equal("approved"))
	})

	t.Run("records commit and transitions phase", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Set pair to approved state
		_, err = state.SetPair(dir, "pm", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "approved",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Complete commit step
		err = step.Complete(dir, step.CompleteResult{
			Action: "commit",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify pair state shows committed
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["pm"].QAVerdict).To(Equal("committed"))
	})

	t.Run("transition complete advances the state machine phase", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Set pair as committed
		_, err = state.SetPair(dir, "pm", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "committed",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Complete the transition
		err = step.Complete(dir, step.CompleteResult{
			Action: "transition",
			Phase:  "pm-complete",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify phase transitioned
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("pm-complete"))

		// Verify pair cleared
		g.Expect(s.Pairs).ToNot(HaveKey("pm"))
	})

	t.Run("errors on invalid action", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		err = step.Complete(dir, step.CompleteResult{
			Action: "invalid-action",
			Status: "done",
		}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("unknown action"))
	})

	t.Run("qa cannot be skipped - no commit without qa pass", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Try to complete commit without producer or qa
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

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.TaskParams).ToNot(BeNil())
		g.Expect(result.TaskParams.SubagentType).To(Equal("code"))
		g.Expect(result.TaskParams.Name).To(Equal("pm-interview-producer"))
		g.Expect(result.TaskParams.Model).To(Equal("sonnet"))
		g.Expect(result.TaskParams.Prompt).To(HavePrefix(step.HandshakeInstruction))
		g.Expect(result.ExpectedModel).To(Equal("sonnet"))
	})

	t.Run("spawn-qa populates TaskParams with QA fields", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-89",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.SetPair(dir, "pm", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))
		g.Expect(result.TaskParams).ToNot(BeNil())
		g.Expect(result.TaskParams.SubagentType).To(Equal("code"))
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

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.SetPair(dir, "pm", state.PairState{
			Iteration:          1,
			MaxIterations:      3,
			ProducerComplete:   true,
			QAVerdict:          "improvement-request",
			ImprovementRequest: "needs work",
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.TaskParams).ToNot(BeNil())
		g.Expect(result.TaskParams.SubagentType).To(Equal("code"))
		g.Expect(result.TaskParams.Name).To(Equal("pm-interview-producer"))
		g.Expect(result.TaskParams.Model).To(Equal("sonnet"))
		g.Expect(result.ExpectedModel).To(Equal("sonnet"))
	})

	t.Run("non-spawn actions have nil TaskParams and empty ExpectedModel", func(t *testing.T) {
		g := NewWithT(t)

		// Test commit action
		dir := t.TempDir()
		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-89"})
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.SetPair(dir, "pm", state.PairState{
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

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm-complete", state.TransitionOpts{}, nowFunc())
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
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red",
			"commit-red", "tdd-green", "commit-green", "tdd-refactor",
			"commit-refactor", "task-audit", "task-complete",
			"implementation-complete", "documentation", "documentation-complete",
			"alignment", "alignment-complete", "retro", "retro-complete",
			"summary", "summary-complete", "issue-update", "next-steps", "complete",
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

	t.Run("tdd-red spawn-producer has correct TaskParams", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "task",
			Issue:    "ISSUE-42",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "task-implementation", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "task-start", state.TransitionOpts{Task: "TASK-001"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tdd-red", state.TransitionOpts{Task: "TASK-001"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.TaskParams).ToNot(BeNil())
		g.Expect(result.TaskParams.SubagentType).To(Equal("code"))
		g.Expect(result.TaskParams.Name).To(Equal("tdd-red-producer"))
		g.Expect(result.TaskParams.Model).To(Equal("sonnet"))
		g.Expect(result.ExpectedModel).To(Equal("sonnet"))
	})
}

func TestPromptAssembly(t *testing.T) {
	t.Run("prompt starts with HandshakeInstruction", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.TaskParams).ToNot(BeNil())
		g.Expect(result.TaskParams.Prompt).To(HavePrefix(step.HandshakeInstruction))
	})

	t.Run("prompt contains skill invocation instruction", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.TaskParams.Prompt).To(ContainSubstring("Then invoke /pm-interview-producer"))
	})

	t.Run("prompt includes issue reference when present", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.TaskParams.Prompt).To(ContainSubstring("ISSUE-98"))
	})

	t.Run("prompt includes QA feedback for improvement-request", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.SetPair(dir, "pm", state.PairState{
			Iteration:          1,
			MaxIterations:      3,
			ProducerComplete:   true,
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
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.SetPair(dir, "pm", state.PairState{
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
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.SetPair(dir, "pm", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))
		g.Expect(result.TaskParams.Prompt).To(HavePrefix(step.HandshakeInstruction))
		g.Expect(result.TaskParams.Prompt).To(ContainSubstring("Then invoke /qa"))
	})
}

func TestCompleteFailedSpawn(t *testing.T) {
	t.Run("failed spawn-producer increments SpawnAttempts and appends FailedModels", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		err = step.Complete(dir, step.CompleteResult{
			Action:        "spawn-producer",
			Status:        "failed",
			ReportedModel: "haiku",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["pm"].SpawnAttempts).To(Equal(1))
		g.Expect(s.Pairs["pm"].FailedModels).To(Equal([]string{"haiku"}))
	})

	t.Run("failed spawn-producer does NOT set ProducerComplete", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		err = step.Complete(dir, step.CompleteResult{
			Action:        "spawn-producer",
			Status:        "failed",
			ReportedModel: "haiku",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["pm"].ProducerComplete).To(BeFalse())
	})

	t.Run("failed spawn-qa does NOT set QAVerdict", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.SetPair(dir, "pm", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
		})
		g.Expect(err).ToNot(HaveOccurred())

		err = step.Complete(dir, step.CompleteResult{
			Action:        "spawn-qa",
			Status:        "failed",
			ReportedModel: "sonnet",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["pm"].QAVerdict).To(BeEmpty())
		g.Expect(s.Pairs["pm"].SpawnAttempts).To(Equal(1))
		g.Expect(s.Pairs["pm"].FailedModels).To(Equal([]string{"sonnet"}))
	})

	t.Run("done spawn-producer resets SpawnAttempts and FailedModels", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Set some prior failed attempts
		_, err = state.SetPair(dir, "pm", state.PairState{
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
		g.Expect(s.Pairs["pm"].SpawnAttempts).To(Equal(0))
		g.Expect(s.Pairs["pm"].FailedModels).To(BeNil())
		g.Expect(s.Pairs["pm"].ProducerComplete).To(BeTrue())
	})

	t.Run("empty status works as done path (backward compat)", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		err = step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["pm"].ProducerComplete).To(BeTrue())
		g.Expect(s.Pairs["pm"].SpawnAttempts).To(Equal(0))
	})
}

func TestNextRetryEscalation(t *testing.T) {
	t.Run("SpawnAttempts == 0 emits normal spawn", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Details).To(BeEmpty())
	})

	t.Run("SpawnAttempts == 1 emits same spawn (retry)", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.SetPair(dir, "pm", state.PairState{
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

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.SetPair(dir, "pm", state.PairState{
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

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.SetPair(dir, "pm", state.PairState{
			SpawnAttempts: 3,
			FailedModels:  []string{"haiku", "haiku", "haiku"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("escalate-user"))
		g.Expect(result.Details).To(ContainSubstring("sonnet"))
		g.Expect(result.Details).To(ContainSubstring("haiku"))
	})

	t.Run("SpawnAttempts >= 3 for spawn-qa emits escalate-user", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Issue: "ISSUE-98"})
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.SetPair(dir, "pm", state.PairState{
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
			Model:     "sonnet",
			Artifact:  "requirements.md",
			Phase:     "pm",
			Context: step.StepContext{
				Issue:          "ISSUE-89",
				PriorArtifacts: []string{},
				QAFeedback:     "",
			},
		}

		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Skill).To(Equal("pm-interview-producer"))
		g.Expect(result.Model).To(Equal("sonnet"))
		g.Expect(result.Context.Issue).To(Equal("ISSUE-89"))
	})
}

// reachable returns the set of all phases reachable from the given starting phase
// via the LegalTransitions graph.
func reachable(from string) map[string]bool {
	visited := map[string]bool{}
	queue := []string{from}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		for _, target := range state.LegalTransitions[current] {
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
func onAllPathsTo(from, target, required string) bool {
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

		for _, next := range state.LegalTransitions[current] {
			if next != required && !visited[next] {
				queue = append(queue, next)
			}
		}
	}

	return true // Target not reachable without required
}

func TestMainFlowEndingMandatory(t *testing.T) {
	// Main flow ending phases in order. Every workflow must pass through all of
	// these before reaching "complete".
	mainFlowEnding := []string{
		"alignment", "retro", "summary", "issue-update", "next-steps",
	}

	// Phases that are part of (or after) the main flow ending sequence.
	// These naturally skip earlier ending phases they've already passed.
	mainFlowEndingSet := map[string]bool{}
	for _, p := range mainFlowEnding {
		mainFlowEndingSet[p] = true
	}
	// Also include the completion markers between ending phases
	for _, suffix := range []string{"alignment-complete", "retro-complete", "summary-complete"} {
		mainFlowEndingSet[suffix] = true
	}

	// Collect all phases that feed into the main flow ending but are not part of it.
	// These are the phases we want to verify the property on.
	var preEndingPhases []string
	for phase, targets := range state.LegalTransitions {
		if len(targets) > 0 && phase != "complete" && !mainFlowEndingSet[phase] {
			preEndingPhases = append(preEndingPhases, phase)
		}
	}

	t.Run("pre-ending phases must traverse all main flow ending phases to reach complete", func(t *testing.T) {
		g := NewWithT(t)

		rapid.Check(t, func(rt *rapid.T) {
			phase := rapid.SampledFrom(preEndingPhases).Draw(rt, "phase")

			reached := reachable(phase)
			if !reached["complete"] {
				return
			}

			for _, required := range mainFlowEnding {
				g.Expect(onAllPathsTo(phase, "complete", required)).To(BeTrue(),
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
				g.Expect(onAllPathsTo(phase, "complete", later)).To(BeTrue(),
					"phase %q can reach 'complete' without passing through later ending phase %q",
					phase, later)
			}
		}
	})
}
