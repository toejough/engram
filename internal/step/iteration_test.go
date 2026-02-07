package step_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
	"github.com/toejough/projctl/internal/step"
)

// TestIterationEnforcement verifies TASK-5 acceptance criteria:
// Iteration counter increments on improvement-request, resets on phase transition,
// and returns escalate-user when max iterations exceeded.

func TestIterationIncrementsOnImprovementRequest(t *testing.T) {
	t.Run("iteration increments when QA requests improvements", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "task",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Use valid state machine path: init -> task-implementation -> task-start -> tdd-red
		_, err = state.Transition(dir, "task-implementation", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "task-start", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "tdd-red", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Complete producer
		err = step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// QA requests improvement
		err = step.Complete(dir, step.CompleteResult{
			Action:     "spawn-qa",
			Status:     "done",
			QAVerdict:  "improvement-request",
			QAFeedback: "Fix the tests",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Check iteration incremented
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd-red"].Iteration).To(Equal(2))
	})
}

func TestIterationResetsOnPhaseTransition(t *testing.T) {
	t.Run("iteration resets when transitioning to next phase", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "task",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Use valid state machine path: init -> task-implementation -> task-start -> tdd-red
		_, err = state.Transition(dir, "task-implementation", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "task-start", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "tdd-red", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Set iteration to 2
		_, err = state.SetPair(dir, "tdd-red", state.PairState{
			Iteration:        2,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "approved",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Mark QA as approved (required before commit can happen)
		_, err = state.SetPair(dir, "tdd-red", state.PairState{
			Iteration:        2,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "approved",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Complete commit
		err = step.Complete(dir, step.CompleteResult{
			Action: "commit",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Transition to next phase (tdd-red-qa, following the state machine)
		err = step.Complete(dir, step.CompleteResult{
			Action: "transition",
			Phase:  "tdd-red-qa",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Check tdd-red pair state is cleared
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		_, exists := s.Pairs["tdd-red"]
		g.Expect(exists).To(BeFalse(), "tdd-red pair state should be cleared on transition")
	})
}

func TestMaxIterationEnforcement(t *testing.T) {
	t.Run("returns escalate-user when max iterations exceeded", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "task",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Use valid state machine path: init -> task-implementation -> task-start -> tdd-red
		_, err = state.Transition(dir, "task-implementation", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "task-start", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "tdd-red", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Set iteration beyond max (4 > 3)
		_, err = state.SetPair(dir, "tdd-red", state.PairState{
			Iteration:          4,
			MaxIterations:      3,
			ProducerComplete:   true,
			QAVerdict:          "improvement-request",
			ImprovementRequest: "Try again",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Next should return escalate-user
		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("escalate-user"))
		g.Expect(result.Details).To(ContainSubstring("max iterations (4) exceeded"))
	})
}

func TestIterationNeverDecreases(t *testing.T) {
	t.Run("iteration counter never decreases during improvement cycles", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "task",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Use valid state machine path: init -> task-implementation -> task-start -> tdd-red
		_, err = state.Transition(dir, "task-implementation", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "task-start", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "tdd-red", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// First iteration: producer -> qa -> improvement-request
		err = step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		err = step.Complete(dir, step.CompleteResult{
			Action:     "spawn-qa",
			Status:     "done",
			QAVerdict:  "improvement-request",
			QAFeedback: "Fix test 1",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, _ := state.Get(dir)
		iteration1 := s.Pairs["tdd-red"].Iteration
		g.Expect(iteration1).To(Equal(2))

		// Second iteration: producer -> qa -> improvement-request
		err = step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		err = step.Complete(dir, step.CompleteResult{
			Action:     "spawn-qa",
			Status:     "done",
			QAVerdict:  "improvement-request",
			QAFeedback: "Fix test 2",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, _ = state.Get(dir)
		iteration2 := s.Pairs["tdd-red"].Iteration
		g.Expect(iteration2).To(Equal(3))
		g.Expect(iteration2).To(BeNumerically(">", iteration1))
	})
}
