package step_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
	"github.com/toejough/projctl/internal/step"
)

// TestStepComplete verifies TASK-7/TASK-14 acceptance criteria:
// step.Complete() updates state correctly based on completed actions.

// navigateToTDDRedProduce initializes a scoped workflow and transitions
// to tdd_red_produce via the flat state machine path:
// init -> item_select -> item_fork -> worktree_create -> tdd_red_produce
func navigateToTDDRedProduce(g Gomega, dir string) {
	_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
		Workflow: "scoped",
	})
	g.Expect(err).ToNot(HaveOccurred())

	for _, phase := range []string{"tasklist_create", "item_select", "item_fork", "worktree_create", "tdd_red_produce"} {
		_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
	}
}

func TestCompleteSpawnProducer(t *testing.T) {
	t.Run("marks producer complete and resets spawn attempts on success", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		navigateToTDDRedProduce(g, dir)

		err := step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd_red"].ProducerComplete).To(BeTrue())
		g.Expect(s.Pairs["tdd_red"].SpawnAttempts).To(Equal(0))
		g.Expect(s.Pairs["tdd_red"].Iteration).To(Equal(1))
	})

	t.Run("increments spawn attempts on failure", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		navigateToTDDRedProduce(g, dir)

		err := step.Complete(dir, step.CompleteResult{
			Action:        "spawn-producer",
			Status:        "failed",
			ReportedModel: "opus",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd_red"].SpawnAttempts).To(Equal(1))
		g.Expect(s.Pairs["tdd_red"].FailedModels).To(ContainElement("opus"))
	})
}

func TestCompleteSpawnQA(t *testing.T) {
	t.Run("records QA verdict and resets spawn attempts on success", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		navigateToTDDRedProduce(g, dir)

		_, err := state.SetPair(dir, "tdd_red", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
		})
		g.Expect(err).ToNot(HaveOccurred())

		err = step.Complete(dir, step.CompleteResult{
			Action:    "spawn-qa",
			Status:    "done",
			QAVerdict: "approved",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd_red"].QAVerdict).To(Equal("approved"))
		g.Expect(s.Pairs["tdd_red"].SpawnAttempts).To(Equal(0))
	})

	t.Run("increments iteration on improvement-request", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		navigateToTDDRedProduce(g, dir)

		_, err := state.SetPair(dir, "tdd_red", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
		})
		g.Expect(err).ToNot(HaveOccurred())

		err = step.Complete(dir, step.CompleteResult{
			Action:     "spawn-qa",
			Status:     "done",
			QAVerdict:  "improvement-request",
			QAFeedback: "Fix the tests",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd_red"].QAVerdict).To(Equal("improvement-request"))
		g.Expect(s.Pairs["tdd_red"].ImprovementRequest).To(Equal("Fix the tests"))
		g.Expect(s.Pairs["tdd_red"].ProducerComplete).To(BeFalse())
		g.Expect(s.Pairs["tdd_red"].Iteration).To(Equal(2))
	})
}

func TestCompleteCommit(t *testing.T) {
	t.Run("marks QA verdict as committed", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		navigateToTDDRedProduce(g, dir)

		_, err := state.SetPair(dir, "tdd_red", state.PairState{
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
		g.Expect(s.Pairs["tdd_red"].QAVerdict).To(Equal("committed"))
	})

	t.Run("fails if QA has improvement-request verdict", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		navigateToTDDRedProduce(g, dir)

		// Set QA verdict to improvement-request (commit should reject this)
		_, err := state.SetPair(dir, "tdd_red", state.PairState{
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
		g.Expect(err.Error()).To(ContainSubstring("QA has not approved"))
	})
}

func TestCompleteTransition(t *testing.T) {
	t.Run("preserves pair state when transitioning within same phase group", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		navigateToTDDRedProduce(g, dir)

		// Set some pair state under "tdd_red"
		_, err := state.SetPair(dir, "tdd_red", state.PairState{
			Iteration:        2,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "committed",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Transition from tdd_red_produce -> tdd_red_qa (same pairKey "tdd_red")
		err = step.Complete(dir, step.CompleteResult{
			Action: "transition",
			Phase:  "tdd_red_qa",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd_red_qa"))
		pair, exists := s.Pairs["tdd_red"]
		g.Expect(exists).To(BeTrue(), "pair state should be preserved within same phase group")
		g.Expect(pair.Iteration).To(Equal(2))
		g.Expect(pair.ProducerComplete).To(BeTrue())
	})

	t.Run("clears pair state when transitioning across phase groups", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		navigateToTDDRedProduce(g, dir)

		// Set pair state under "tdd_red"
		_, err := state.SetPair(dir, "tdd_red", state.PairState{
			Iteration:        2,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "approved",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate tdd_red_produce -> tdd_red_qa -> tdd_red_decide
		_, err = state.Transition(dir, "tdd_red_qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tdd_red_decide", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Transition from tdd_red_decide -> tdd_green_produce (pairKey changes: "tdd_red" -> "tdd_green")
		err = step.Complete(dir, step.CompleteResult{
			Action: "transition",
			Phase:  "tdd_green_produce",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd_green_produce"))
		_, exists := s.Pairs["tdd_red"]
		g.Expect(exists).To(BeFalse(), "pair state should be cleared when transitioning across phase groups")
	})
}
