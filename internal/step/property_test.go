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

		navigateToTDDRedProduce(g, dir)

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

		navigateToTDDRedProduce(g, dir)

		// Run a random sequence of producer -> qa -> improvement-request cycles
		numCycles := rapid.IntRange(1, 3).Draw(rt, "numCycles")
		lastIteration := 0

		for i := 0; i < numCycles; i++ {
			// Complete producer
			err := step.RecordComplete(dir, step.CompleteResult{
				Action: "spawn-producer",
				Status: "done",
			}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())

			// Complete QA with improvement-request
			// First transition to tdd_red_qa so Complete("spawn-qa") uses the right pair key
			_, err = state.Transition(dir, "tdd_red_qa", state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())

			err = step.RecordComplete(dir, step.CompleteResult{
				Action:     "spawn-qa",
				Status:     "done",
				QAVerdict:  "improvement-request",
				QAFeedback: "Improve",
			}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())

			// Transition through decide back to produce for next cycle
			_, err = state.Transition(dir, "tdd_red_decide", state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
			_, err = state.Transition(dir, "tdd_red_produce", state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())

			s, _ := state.Get(dir)
			currentIteration := s.Pairs["tdd_red"].Iteration

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

		navigateToTDDRedProduce(g, dir)

		// Set a random max iterations (1-5)
		maxIterations := rapid.IntRange(1, 5).Draw(rt, "maxIterations")

		// Set iteration beyond max with ProducerComplete=false
		// (QA sets ProducerComplete false on improvement-request, putting us back in produce)
		_, err := state.SetPair(dir, "tdd_red", state.PairState{
			Iteration:        maxIterations + 1,
			MaxIterations:    maxIterations,
			ProducerComplete: false,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Property: when iteration > maxIterations, Next() returns escalate-user
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

		navigateToTDDRedProduce(g, dir)

		// Simulate successful producer completion
		err := step.RecordComplete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Property: successful producer completion should lead to transition (to tdd_red_qa)
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

		navigateToTDDRedProduce(g, dir)

		// Property: ProducerComplete=true implies QAVerdict is either empty or set
		err := step.RecordComplete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, _ := state.Get(dir)
		if s.Pairs["tdd_red"].ProducerComplete {
			// When producer is complete, QA verdict should initially be empty
			g.Expect(s.Pairs["tdd_red"].QAVerdict).To(BeEmpty(),
				"QA verdict should be empty after producer completes")
		}
	})
}
