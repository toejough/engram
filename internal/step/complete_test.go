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

func TestStepCompleteProducerDone(t *testing.T) {
	t.Run("producer completion updates state correctly", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Initialize and navigate to tdd-red
		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		phases := []string{"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red"}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Complete producer action
		err = step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred(), "producer completion should succeed")

		// Verify state reflects completion
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd-red"), "phase should remain tdd-red until transitioned")
	})

	t.Run("producer completion with failed status returns error", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		phases := []string{"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red"}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Complete with failed status
		err = step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "failed",
		}, nowFunc())
		g.Expect(err).To(HaveOccurred(), "failed producer should return error")
	})
}

func TestStepCompleteQAVerdict(t *testing.T) {
	t.Run("QA approved verdict updates state", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		phases := []string{"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red", "tdd-red-qa"}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// QA approves
		err = step.Complete(dir, step.CompleteResult{
			Action:    "spawn-qa",
			Status:    "done",
			QAVerdict: "approved",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred(), "QA approved should succeed")

		// Verify pair state updated
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		pairState, exists := s.Pairs["tdd-red"]
		g.Expect(exists).To(BeTrue(), "pair state should exist for tdd-red")
		g.Expect(pairState.QAVerdict).To(Equal("approved"), "QA verdict should be recorded")
	})

	t.Run("QA improvement-request verdict records feedback", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		phases := []string{"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red", "tdd-red-qa"}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// QA requests improvement
		feedbackText := "Missing trace to REQ-042"
		err = step.Complete(dir, step.CompleteResult{
			Action:     "spawn-qa",
			Status:     "done",
			QAVerdict:  "improvement-request",
			QAFeedback: feedbackText,
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred(), "QA improvement-request should succeed")

		// Verify feedback stored
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		pairState, exists := s.Pairs["tdd-red"]
		g.Expect(exists).To(BeTrue())
		g.Expect(pairState.QAVerdict).To(Equal("improvement-request"))
		g.Expect(pairState.ImprovementRequest).To(Equal(feedbackText), "QA feedback should be stored")
		g.Expect(pairState.Iteration).To(Equal(1), "iteration should increment on improvement-request")
	})

	t.Run("QA escalate-user verdict is recorded", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		phases := []string{"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red", "tdd-red-qa"}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// QA escalates to user
		err = step.Complete(dir, step.CompleteResult{
			Action:     "spawn-qa",
			Status:     "done",
			QAVerdict:  "escalate-user",
			QAFeedback: "Max iterations reached",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred(), "QA escalate-user should succeed")

		// Verify verdict recorded
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		pairState, exists := s.Pairs["tdd-red"]
		g.Expect(exists).To(BeTrue())
		g.Expect(pairState.QAVerdict).To(Equal("escalate-user"))
	})
}

func TestStepCompleteTransitionAction(t *testing.T) {
	t.Run("transition action updates phase", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		phases := []string{"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red", "tdd-red-qa"}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Set approved verdict first
		err = step.Complete(dir, step.CompleteResult{
			Action:    "spawn-qa",
			Status:    "done",
			QAVerdict: "approved",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Complete transition to commit-red
		err = step.Complete(dir, step.CompleteResult{
			Action: "transition",
			Status: "done",
			Phase:  "commit-red",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred(), "transition should succeed")

		// Verify phase changed
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("commit-red"), "phase should be updated to commit-red")
	})

	t.Run("illegal transition is rejected", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		phases := []string{"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red"}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Try to transition directly to tdd-green (skipping tdd-red-qa and commit-red)
		err = step.Complete(dir, step.CompleteResult{
			Action: "transition",
			Status: "done",
			Phase:  "tdd-green",
		}, nowFunc())
		g.Expect(err).To(HaveOccurred(), "illegal transition should be rejected")
		g.Expect(err.Error()).To(ContainSubstring("not a legal transition"), "error should mention illegal transition")
	})
}

func TestStepCompleteEscalationResolution(t *testing.T) {
	t.Run("escalation resolution allows manual intervention", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		phases := []string{"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red", "tdd-red-qa"}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Escalate to user
		err = step.Complete(dir, step.CompleteResult{
			Action:     "spawn-qa",
			Status:     "done",
			QAVerdict:  "escalate-user",
			QAFeedback: "Unresolvable issue",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// User manually fixes and continues
		err = step.Complete(dir, step.CompleteResult{
			Action: "escalate-user-resolved",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred(), "user escalation resolution should succeed")

		// Verify state allows continuation
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).ToNot(Equal("error"), "phase should not be in error state after resolution")
	})
}

func TestStepCompleteInvalidAction(t *testing.T) {
	t.Run("unknown action type returns error", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		err = step.Complete(dir, step.CompleteResult{
			Action: "invalid-action-type",
			Status: "done",
		}, nowFunc())
		g.Expect(err).To(HaveOccurred(), "invalid action type should return error")
		g.Expect(err.Error()).To(ContainSubstring("unknown action"), "error should mention unknown action")
	})

	t.Run("empty status returns error", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		err = step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "",
		}, nowFunc())
		g.Expect(err).To(HaveOccurred(), "empty status should return error")
	})
}

func TestStepCompleteStatePersistence(t *testing.T) {
	t.Run("state changes are persisted to disk", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		phases := []string{"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red", "tdd-red-qa"}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		feedbackText := "Test feedback"
		err = step.Complete(dir, step.CompleteResult{
			Action:     "spawn-qa",
			Status:     "done",
			QAVerdict:  "improvement-request",
			QAFeedback: feedbackText,
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Reload state from disk
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		pairState, exists := s.Pairs["tdd-red"]
		g.Expect(exists).To(BeTrue())
		g.Expect(pairState.ImprovementRequest).To(Equal(feedbackText), "persisted state should contain QA feedback")
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

		// Attempt completion should fail gracefully
		err = step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).To(HaveOccurred(), "corrupted state should return error")
	})
}

func TestStepCompleteModelMismatch(t *testing.T) {
	t.Run("model mismatch on spawn failure is recorded", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		phases := []string{"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red"}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Report spawn failure due to model mismatch
		err = step.Complete(dir, step.CompleteResult{
			Action:        "spawn-producer",
			Status:        "failed",
			ReportedModel: "haiku",
		}, nowFunc())
		g.Expect(err).To(HaveOccurred(), "failed spawn should return error")
		// The error should contain information about model mismatch
		g.Expect(err.Error()).To(ContainSubstring("model"), "error should mention model issue")
	})
}

// Note: nowFunc() helper is already defined in next_test.go
