package step_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
	"github.com/toejough/projctl/internal/step"
)

// TestStepComplete verifies TASK-14 acceptance criteria:
// projctl step complete correctly updates state.toml for various completion scenarios
//
// Traces to: REQ-105-003, ARCH-105-003, ARCH-105-004, TASK-7, TASK-14, ISSUE-105

// navigateToTDDRedProduceViaScoped initializes a scoped workflow and transitions
// to tdd_red_produce via the flat state machine path:
// init -> item_select -> item_fork -> worktree_create -> tdd_red_produce
func navigateToTDDRedProduceViaScoped(g Gomega, dir string) {
	_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
		Workflow: "scoped",
	})
	g.Expect(err).ToNot(HaveOccurred())

	for _, phase := range []string{"tasklist_create", "item_select", "item_fork", "worktree_create", "tdd_red_produce"} {
		_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
	}
}

func TestStepCompleteProducerDone(t *testing.T) {
	t.Run("producer completion updates state correctly", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		navigateToTDDRedProduceViaScoped(g, dir)

		err := step.RecordComplete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd_red_produce"))
		g.Expect(s.Pairs["tdd_red"].ProducerComplete).To(BeTrue())
	})

	t.Run("producer completion with failed status records failure", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		navigateToTDDRedProduceViaScoped(g, dir)

		err := step.RecordComplete(dir, step.CompleteResult{
			Action:        "spawn-producer",
			Status:        "failed",
			ReportedModel: "haiku",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd_red"].SpawnAttempts).To(Equal(1))
		g.Expect(s.Pairs["tdd_red"].FailedModels).To(ContainElement("haiku"))
	})
}

func TestStepCompleteQAVerdict(t *testing.T) {
	t.Run("QA approved verdict updates state", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		navigateToTDDRedProduceViaScoped(g, dir)
		_, err := state.Transition(dir, "tdd_red_qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		err = step.RecordComplete(dir, step.CompleteResult{
			Action:    "spawn-qa",
			Status:    "done",
			QAVerdict: "approved",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		// pairKey("tdd_red_qa") = "tdd_red"
		g.Expect(s.Pairs["tdd_red"].QAVerdict).To(Equal("approved"))
	})

	t.Run("QA improvement-request verdict records feedback", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		navigateToTDDRedProduceViaScoped(g, dir)
		_, err := state.Transition(dir, "tdd_red_qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		feedbackText := "Missing trace to REQ-042"
		err = step.RecordComplete(dir, step.CompleteResult{
			Action:     "spawn-qa",
			Status:     "done",
			QAVerdict:  "improvement-request",
			QAFeedback: feedbackText,
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd_red"].QAVerdict).To(Equal("improvement-request"))
		g.Expect(s.Pairs["tdd_red"].ImprovementRequest).To(Equal(feedbackText))
		g.Expect(s.Pairs["tdd_red"].ProducerComplete).To(BeFalse())
		g.Expect(s.Pairs["tdd_red"].Iteration).To(Equal(1))
	})

	t.Run("QA escalate-user verdict is recorded", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		navigateToTDDRedProduceViaScoped(g, dir)
		_, err := state.Transition(dir, "tdd_red_qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		err = step.RecordComplete(dir, step.CompleteResult{
			Action:     "spawn-qa",
			Status:     "done",
			QAVerdict:  "escalate-user",
			QAFeedback: "Max iterations reached",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd_red"].QAVerdict).To(Equal("escalate-user"))
		g.Expect(s.Project.Phase).To(Equal("tdd_red_qa"))
	})
}

func TestStepCompleteTransitionAction(t *testing.T) {
	t.Run("transition action updates phase", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		navigateToTDDRedProduceViaScoped(g, dir)

		// Transition from tdd_red_produce to tdd_red_qa (same group)
		err := step.RecordComplete(dir, step.CompleteResult{
			Action: "transition",
			Phase:  "tdd_red_qa",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd_red_qa"))
	})

	t.Run("illegal transition is rejected", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		navigateToTDDRedProduceViaScoped(g, dir)

		// Try to transition directly to tdd_green_produce (skipping tdd_red_qa, tdd_red_decide)
		err := step.RecordComplete(dir, step.CompleteResult{
			Action: "transition",
			Phase:  "tdd_green_produce",
		}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("illegal transition"))
	})
}

func TestStepCompleteEscalationResolution(t *testing.T) {
	t.Run("escalation verdict is recorded in state", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		navigateToTDDRedProduceViaScoped(g, dir)
		_, err := state.Transition(dir, "tdd_red_qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		err = step.RecordComplete(dir, step.CompleteResult{
			Action:     "spawn-qa",
			Status:     "done",
			QAVerdict:  "escalate-user",
			QAFeedback: "Unresolvable issue",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd_red"].QAVerdict).To(Equal("escalate-user"))
		g.Expect(s.Project.Phase).To(Equal("tdd_red_qa"))
	})
}

func TestStepCompleteInvalidAction(t *testing.T) {
	t.Run("unknown action type returns error", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		err = step.RecordComplete(dir, step.CompleteResult{
			Action: "invalid-action-type",
			Status: "done",
		}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("unknown action"))
	})

	t.Run("empty status is treated as done", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		navigateToTDDRedProduceViaScoped(g, dir)

		err := step.RecordComplete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd_red"].ProducerComplete).To(BeTrue())
	})
}

func TestStepCompleteStatePersistence(t *testing.T) {
	t.Run("state changes are persisted to disk", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		navigateToTDDRedProduceViaScoped(g, dir)
		_, err := state.Transition(dir, "tdd_red_qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		feedbackText := "Test feedback"
		err = step.RecordComplete(dir, step.CompleteResult{
			Action:     "spawn-qa",
			Status:     "done",
			QAVerdict:  "improvement-request",
			QAFeedback: feedbackText,
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Reload state from disk
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd_red"].ImprovementRequest).To(Equal(feedbackText))
	})

	t.Run("state file corruption is detected", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Corrupt the state file
		statePath := filepath.Join(dir, "state.toml")
		err = os.WriteFile(statePath, []byte("corrupted invalid toml {{{"), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		err = step.RecordComplete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).To(HaveOccurred())
	})
}

func TestStepCompleteModelMismatch(t *testing.T) {
	t.Run("model mismatch on spawn failure is recorded", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		navigateToTDDRedProduceViaScoped(g, dir)

		err := step.RecordComplete(dir, step.CompleteResult{
			Action:        "spawn-producer",
			Status:        "failed",
			ReportedModel: "haiku",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd_red"].FailedModels).To(ContainElement("haiku"))
		g.Expect(s.Pairs["tdd_red"].SpawnAttempts).To(Equal(1))
	})
}
