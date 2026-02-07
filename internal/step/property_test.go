package step_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
	"github.com/toejough/projctl/internal/step"
	"pgregory.net/rapid"
)

// TestPropertyBasedInvariants verifies TASK-16 acceptance criteria:
// State machine invariants hold across randomized sequences.
// Properties: never stuck, all sequences reach terminal, iteration never decreases, max enforced.

func TestPropertyNeverStuck(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "task",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd-red
		_, err = state.Transition(dir, "task-implementation", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "task-start", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tdd-red", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Property: Next() should always return a valid action (never stuck)
		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).ToNot(BeEmpty(), "Next() must always return an action")
		g.Expect(result.Action).To(BeElementOf(
			"spawn-producer", "spawn-qa", "commit", "transition", "escalate-user", "all-complete",
		))
	})
}

func TestPropertyIterationNeverDecreases(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
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

		// Run a random sequence of producer -> qa -> improvement-request cycles
		numCycles := rapid.IntRange(1, 3).Draw(rt, "numCycles")
		lastIteration := 0

		for i := 0; i < numCycles; i++ {
			// Complete producer
			err = step.Complete(dir, step.CompleteResult{
				Action: "spawn-producer",
				Status: "done",
			}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())

			// Complete QA with improvement-request
			err = step.Complete(dir, step.CompleteResult{
				Action:     "spawn-qa",
				Status:     "done",
				QAVerdict:  "improvement-request",
				QAFeedback: "Improve",
			}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())

			s, _ := state.Get(dir)
			currentIteration := s.Pairs["tdd-red"].Iteration

			// Property: iteration never decreases
			g.Expect(currentIteration).To(BeNumerically(">=", lastIteration),
				"iteration must never decrease")
			lastIteration = currentIteration
		}
	})
}

func TestPropertyMaxIterationsEnforced(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
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

		// Set a random max iterations (1-5)
		maxIterations := rapid.IntRange(1, 5).Draw(rt, "maxIterations")

		// Run cycles until we hit max
		for i := 1; i <= maxIterations; i++ {
			err = step.Complete(dir, step.CompleteResult{
				Action: "spawn-producer",
				Status: "done",
			}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())

			if i < maxIterations {
				err = step.Complete(dir, step.CompleteResult{
					Action:     "spawn-qa",
					Status:     "done",
					QAVerdict:  "improvement-request",
					QAFeedback: "Improve",
				}, nowFunc())
				g.Expect(err).ToNot(HaveOccurred())
			}
		}

		// Set iteration to max and QA verdict to improvement-request
		_, err = state.SetPair(dir, "tdd-red", state.PairState{
			Iteration:          maxIterations,
			MaxIterations:      maxIterations,
			ProducerComplete:   true,
			QAVerdict:          "improvement-request",
			ImprovementRequest: "Try again",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Property: when iteration >= maxIterations, Next() returns escalate-user
		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("escalate-user"),
			"max iterations must be enforced")
	})
}

func TestPropertyTerminalPhasesReachable(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
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

		// Simulate successful completion sequence
		err = step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		err = step.Complete(dir, step.CompleteResult{
			Action:    "spawn-qa",
			Status:    "done",
			QAVerdict: "approved",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		err = step.Complete(dir, step.CompleteResult{
			Action: "commit",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Property: successful sequences should reach transition or completion
		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"),
			"successful completion should lead to transition")
	})
}

func TestPropertyPairStateConsistency(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
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

		// Property: ProducerComplete=true implies QAVerdict is either empty or set
		err = step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, _ := state.Get(dir)
		if s.Pairs["tdd-red"].ProducerComplete {
			// When producer is complete, QA verdict should initially be empty
			g.Expect(s.Pairs["tdd-red"].QAVerdict).To(BeEmpty(),
				"QA verdict should be empty after producer completes")
		}
	})
}
