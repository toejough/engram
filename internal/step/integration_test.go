package step_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
	"github.com/toejough/projctl/internal/step"
)

// TestFullTDDWorkflow verifies integration of the full TDD cycle
// driven by step next / step complete through the flat state machine.
//
// Scoped workflow path through TDD:
// item_select -> item_fork -> worktree_create ->
// tdd_red_produce -> tdd_red_qa -> tdd_red_decide -> tdd_green_produce ->
// tdd_green_qa -> tdd_green_decide -> tdd_refactor_produce ->
// tdd_refactor_qa -> tdd_refactor_decide -> tdd_commit
func TestFullTDDWorkflow(t *testing.T) {
	t.Run("state machine drives full TDD cycle", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-tdd-workflow", nowFunc(), state.InitOpts{
			Workflow: "scoped",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd_red_produce
		for _, phase := range []string{"tasklist_create", "item_select", "item_fork", "worktree_create", "tdd_red_produce"} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// === tdd_red_produce: spawn producer ===
		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Skill).To(Equal("tdd-red-producer"))
		g.Expect(result.Phase).To(Equal("tdd_red_produce"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-producer", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// ProducerComplete=true -> transition to tdd_red_qa
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tdd_red_qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_red_qa"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === tdd_red_qa: spawn QA ===
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))
		g.Expect(result.Skill).To(Equal("qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-qa", Status: "done", QAVerdict: "approved"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// QAVerdict set -> transition to tdd_red_decide
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tdd_red_decide"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_red_decide"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === tdd_red_decide: approved -> targets[1] = tdd_green_produce ===
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tdd_green_produce"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_green_produce"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify tdd_red pair state cleared (cross-group: tdd_red -> tdd_green)
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		_, exists := s.Pairs["tdd_red"]
		g.Expect(exists).To(BeFalse(), "tdd_red pair should be cleared on cross-group transition")

		// === tdd_green_produce: spawn producer ===
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Skill).To(Equal("tdd-green-producer"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-producer", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// transition to tdd_green_qa
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tdd_green_qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_green_qa"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === tdd_green_qa: spawn QA ===
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-qa", Status: "done", QAVerdict: "approved"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// transition to tdd_green_decide
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tdd_green_decide"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_green_decide"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === tdd_green_decide: approved -> targets[1] = tdd_refactor_produce ===
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tdd_refactor_produce"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_refactor_produce"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === tdd_refactor_produce: spawn producer ===
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Skill).To(Equal("tdd-refactor-producer"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-producer", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// transition to tdd_refactor_qa
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tdd_refactor_qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_refactor_qa"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === tdd_refactor_qa: spawn QA ===
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-qa", Status: "done", QAVerdict: "approved"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// transition to tdd_refactor_decide
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tdd_refactor_decide"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_refactor_decide"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === tdd_refactor_decide: approved -> targets[1] = tdd_commit ===
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tdd_commit"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_commit"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === tdd_commit: commit action ===
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("commit"))

		err = step.Complete(dir, step.CompleteResult{Action: "commit", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// tdd_commit committed -> transition to merge_acquire
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("merge_acquire"))
	})
}

// TestQAIterationWithFeedback verifies the improvement-request loop:
// QA requests improvement -> iteration increments -> producer re-spawned with feedback
// -> max iterations triggers escalation
func TestQAIterationWithFeedback(t *testing.T) {
	t.Run("QA improvement-request triggers iteration loop", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-qa-iteration", nowFunc(), state.InitOpts{
			Workflow: "scoped",
		})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{"tasklist_create", "item_select", "item_fork", "worktree_create", "tdd_red_produce"} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// === Iteration 1: Producer -> QA -> improvement-request ===
		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-producer", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Transition to tdd_red_qa
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_red_qa"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// QA requests improvement
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		err = step.Complete(dir, step.CompleteResult{
			Action:     "spawn-qa",
			Status:     "done",
			QAVerdict:  "improvement-request",
			QAFeedback: "Add edge case tests for empty input",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify iteration incremented
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd_red"].Iteration).To(Equal(2))
		g.Expect(s.Pairs["tdd_red"].ImprovementRequest).To(Equal("Add edge case tests for empty input"))
		g.Expect(s.Pairs["tdd_red"].ProducerComplete).To(BeFalse())

		// QA verdict set -> transition to tdd_red_decide
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tdd_red_decide"))
		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_red_decide"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// tdd_red_decide: improvement-request (not "approved") -> targets[0] = tdd_red_produce
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tdd_red_produce"))
		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_red_produce"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === Iteration 2: Producer re-spawned with QA feedback ===
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Context.QAFeedback).To(Equal("Add edge case tests for empty input"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-producer", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// transition to tdd_red_qa
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_red_qa"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// QA requests another improvement
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		err = step.Complete(dir, step.CompleteResult{
			Action:     "spawn-qa",
			Status:     "done",
			QAVerdict:  "improvement-request",
			QAFeedback: "Add boundary tests for max values",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err = state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd_red"].Iteration).To(Equal(3))
		g.Expect(s.Pairs["tdd_red"].MaxIterations).To(Equal(3))

		// Navigate back to tdd_red_produce via decide
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_red_decide"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_red_produce"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === Iteration 3: At max, one more attempt ===
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Context.QAFeedback).To(Equal("Add boundary tests for max values"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-producer", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// transition to qa
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_red_qa"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		// QA requests yet another improvement (pushes iteration to 4)
		err = step.Complete(dir, step.CompleteResult{
			Action:     "spawn-qa",
			Status:     "done",
			QAVerdict:  "improvement-request",
			QAFeedback: "One more thing...",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err = state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd_red"].Iteration).To(Equal(4))

		// Navigate back to produce via decide
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_red_decide"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_red_produce"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === Escalation: iteration 4 > max 3 ===
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("escalate-user"))
		g.Expect(result.Details).To(ContainSubstring("max iterations (4) exceeded"))
		g.Expect(result.Details).To(ContainSubstring("tdd_red_produce"))
	})

	t.Run("QA approved verdict advances through decide to next phase", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-qa-approved", nowFunc(), state.InitOpts{
			Workflow: "scoped",
		})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{"tasklist_create", "item_select", "item_fork", "worktree_create", "tdd_red_produce"} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Producer -> QA -> approved
		err = step.Complete(dir, step.CompleteResult{Action: "spawn-producer", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_red_qa"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-qa", Status: "done", QAVerdict: "approved"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// transition to decide
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tdd_red_decide"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_red_decide"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// decide: approved -> tdd_green_produce (targets[1])
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tdd_green_produce"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd_green_produce"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify tdd_red pair state is cleared (cross-group boundary)
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		_, exists := s.Pairs["tdd_red"]
		g.Expect(exists).To(BeFalse(), "tdd_red pair should be cleared on cross-group transition")
		g.Expect(s.Project.Phase).To(Equal("tdd_green_produce"))
	})
}
