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

// navigateToPhaseForIteration walks the scoped workflow state machine
// to reach a target state via direct state.Transition calls.
//
// Scoped workflow path:
//
//	init -> item_select -> item_fork -> worktree_create ->
//	  tdd_red_produce -> tdd_red_qa -> tdd_red_decide
func navigateToPhaseForIteration(t *testing.T, dir string, targetPhase string) {
	t.Helper()
	g := NewWithT(t)

	allPhases := []string{
		"item_select", "item_fork", "worktree_create",
		"tdd_red_produce", "tdd_red_qa", "tdd_red_decide",
	}

	targetIdx := -1
	for i, phase := range allPhases {
		if phase == targetPhase {
			targetIdx = i
			break
		}
	}

	if targetIdx == -1 {
		t.Fatalf("target phase %s not found in iteration test sequence", targetPhase)
	}

	for i := 0; i <= targetIdx; i++ {
		_, err := state.Transition(dir, allPhases[i], state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred(), "failed to transition to %s", allPhases[i])
	}
}

func TestIterationIncrementsOnImprovementRequest(t *testing.T) {
	t.Run("iteration increments when QA requests improvements", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "scoped",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate: init -> item_select -> item_fork -> worktree_create -> tdd_red_produce
		navigateToPhaseForIteration(t, dir, "tdd_red_produce")

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
		g.Expect(s.Pairs["tdd_red"].Iteration).To(Equal(2))
	})
}

func TestIterationResetsOnPhaseTransition(t *testing.T) {
	t.Run("iteration resets when transitioning across pair groups", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "scoped",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd_red_decide (through produce -> qa -> decide)
		navigateToPhaseForIteration(t, dir, "tdd_red_decide")

		// Set pair state with iteration=2 and approved verdict
		_, err = state.SetPair(dir, "tdd_red", state.PairState{
			Iteration:        2,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "approved",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Transition from tdd_red_decide to tdd_green_produce (targets[1] for approved).
		// This crosses pair group boundary: pairKey("tdd_green_produce") = "tdd_green" != "tdd_red",
		// so the "tdd_red" pair state should be cleared.
		err = step.Complete(dir, step.CompleteResult{
			Action: "transition",
			Phase:  "tdd_green_produce",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Check tdd_red pair state is cleared
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		_, exists := s.Pairs["tdd_red"]
		g.Expect(exists).To(BeFalse(), "tdd_red pair state should be cleared on cross-group transition")
	})
}

func TestMaxIterationEnforcement(t *testing.T) {
	t.Run("returns escalate-user when max iterations exceeded", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "scoped",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd_red_produce
		navigateToPhaseForIteration(t, dir, "tdd_red_produce")

		// Set iteration beyond max (4 > 3) with ProducerComplete=false
		// and ImprovementRequest set (QA has sent back for rework)
		_, err = state.SetPair(dir, "tdd_red", state.PairState{
			Iteration:          4,
			MaxIterations:      3,
			ProducerComplete:   false,
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
			Workflow: "scoped",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to tdd_red_produce
		navigateToPhaseForIteration(t, dir, "tdd_red_produce")

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
		iteration1 := s.Pairs["tdd_red"].Iteration
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
		iteration2 := s.Pairs["tdd_red"].Iteration
		g.Expect(iteration2).To(Equal(3))
		g.Expect(iteration2).To(BeNumerically(">", iteration1))
	})
}
