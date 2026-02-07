package step_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
	"github.com/toejough/projctl/internal/step"
)

// TestStepComplete verifies TASK-7/TASK-14 acceptance criteria:
// step.Complete() updates state correctly based on completed actions.

func TestCompleteSpawnProducer(t *testing.T) {
	t.Run("marks producer complete and resets spawn attempts on success", func(t *testing.T) {
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

		// Complete producer
		err = step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify state
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd-red"].ProducerComplete).To(BeTrue())
		g.Expect(s.Pairs["tdd-red"].SpawnAttempts).To(Equal(0))
		g.Expect(s.Pairs["tdd-red"].Iteration).To(Equal(1))
	})

	t.Run("increments spawn attempts on failure", func(t *testing.T) {
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

		// Fail producer spawn
		err = step.Complete(dir, step.CompleteResult{
			Action:        "spawn-producer",
			Status:        "failed",
			ReportedModel: "opus",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify state
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd-red"].SpawnAttempts).To(Equal(1))
		g.Expect(s.Pairs["tdd-red"].FailedModels).To(ContainElement("opus"))
	})
}

func TestCompleteSpawnQA(t *testing.T) {
	t.Run("records QA verdict and resets spawn attempts on success", func(t *testing.T) {
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

		// Set producer complete
		_, err = state.SetPair(dir, "tdd-red", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Complete QA with approved
		err = step.Complete(dir, step.CompleteResult{
			Action:    "spawn-qa",
			Status:    "done",
			QAVerdict: "approved",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify state
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd-red"].QAVerdict).To(Equal("approved"))
		g.Expect(s.Pairs["tdd-red"].SpawnAttempts).To(Equal(0))
	})

	t.Run("increments iteration on improvement-request", func(t *testing.T) {
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

		// Set producer complete
		_, err = state.SetPair(dir, "tdd-red", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Complete QA with improvement-request
		err = step.Complete(dir, step.CompleteResult{
			Action:     "spawn-qa",
			Status:     "done",
			QAVerdict:  "improvement-request",
			QAFeedback: "Fix the tests",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify state
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd-red"].QAVerdict).To(Equal("improvement-request"))
		g.Expect(s.Pairs["tdd-red"].ImprovementRequest).To(Equal("Fix the tests"))
		g.Expect(s.Pairs["tdd-red"].ProducerComplete).To(BeFalse())
		g.Expect(s.Pairs["tdd-red"].Iteration).To(Equal(2))
	})
}

func TestCompleteCommit(t *testing.T) {
	t.Run("marks QA verdict as committed", func(t *testing.T) {
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

		// Set QA approved
		_, err = state.SetPair(dir, "tdd-red", state.PairState{
			Iteration:        1,
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

		// Verify state
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd-red"].QAVerdict).To(Equal("committed"))
	})

	t.Run("fails if QA not approved", func(t *testing.T) {
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

		// Try to commit without QA approval
		err = step.Complete(dir, step.CompleteResult{
			Action: "commit",
			Status: "done",
		}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("QA has not approved"))
	})
}

func TestCompleteTransition(t *testing.T) {
	t.Run("clears pair state and transitions to target phase", func(t *testing.T) {
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

		// Set some pair state
		_, err = state.SetPair(dir, "tdd-red", state.PairState{
			Iteration:        2,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "committed",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Transition to next phase
		err = step.Complete(dir, step.CompleteResult{
			Action: "transition",
			Phase:  "tdd-red-qa",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify state
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd-red-qa"))
		_, exists := s.Pairs["tdd-red"]
		g.Expect(exists).To(BeFalse(), "pair state should be cleared on transition")
	})
}
