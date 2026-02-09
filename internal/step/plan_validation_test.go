package step_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
	"github.com/toejough/projctl/internal/step"
)

// TestStepComplete_PlanProduce_RequiresEnterPlanMode verifies ISSUE-170 AC-3
func TestStepComplete_PlanProduce_RequiresEnterPlanMode(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Initialize state
	_, err := state.Init(dir, "test-project", func() time.Time { return time.Now() })
	g.Expect(err).ToNot(HaveOccurred())

	// Transition through legal states to plan_produce
	_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{Force: true}, time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{Force: true}, time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	// Try to complete step without EnterPlanMode tool call
	result, err := step.Complete(dir, step.CompleteOpts{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(result.Valid).To(BeFalse())
	g.Expect(result.Error).To(ContainSubstring("EnterPlanMode"))
	g.Expect(result.Error).To(ContainSubstring("required"))
}

// TestStepComplete_PlanProduce_RequiresExitPlanMode verifies ISSUE-170 AC-3
func TestStepComplete_PlanProduce_RequiresExitPlanMode(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	_, err := state.Init(dir, "test-project", func() time.Time { return time.Now() })
	g.Expect(err).ToNot(HaveOccurred())

	_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{Force: true}, time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{Force: true}, time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	// Log EnterPlanMode but not ExitPlanMode
	err = state.LogToolCall(dir, "EnterPlanMode", time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	// Try to complete step - should fail because ExitPlanMode is missing
	result, err := step.Complete(dir, step.CompleteOpts{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(result.Valid).To(BeFalse())
	g.Expect(result.Error).To(ContainSubstring("ExitPlanMode"))
	g.Expect(result.Error).To(ContainSubstring("required"))
}

// TestStepComplete_PlanProduce_SucceedsWithBothTools verifies ISSUE-170 AC-3
func TestStepComplete_PlanProduce_SucceedsWithBothTools(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	_, err := state.Init(dir, "test-project", func() time.Time { return time.Now() })
	g.Expect(err).ToNot(HaveOccurred())

	_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{Force: true}, time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{Force: true}, time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	// Log both required tool calls
	err = state.LogToolCall(dir, "EnterPlanMode", time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	err = state.LogToolCall(dir, "ExitPlanMode", time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	// Complete step - should succeed
	result, err := step.Complete(dir, step.CompleteOpts{})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Valid).To(BeTrue())
	g.Expect(result.Error).To(BeEmpty())
}

// TestStepComplete_PlanProduce_ErrorMessageIsActionable verifies ISSUE-170 AC-4
func TestStepComplete_PlanProduce_ErrorMessageIsActionable(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	_, err := state.Init(dir, "test-project", func() time.Time { return time.Now() })
	g.Expect(err).ToNot(HaveOccurred())

	_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{Force: true}, time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{Force: true}, time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	// Try to complete without any tool calls
	result, err := step.Complete(dir, step.CompleteOpts{})
	g.Expect(err).To(HaveOccurred())

	// Error message should be actionable
	g.Expect(result.Error).To(ContainSubstring("plan mode"))
	g.Expect(result.Error).To(ContainSubstring("EnterPlanMode"))
	g.Expect(result.Error).To(ContainSubstring("ExitPlanMode"))

	// Error should explain what the producer should have done
	g.Expect(result.Error).To(Or(
		ContainSubstring("must call"),
		ContainSubstring("required"),
		ContainSubstring("missing"),
	))
}

// TestStepComplete_PlanProduce_ClearsToolCallsAfterSuccess verifies tool calls are cleared
func TestStepComplete_PlanProduce_ClearsToolCallsAfterSuccess(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	_, err := state.Init(dir, "test-project", func() time.Time { return time.Now() })
	g.Expect(err).ToNot(HaveOccurred())

	_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{Force: true}, time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{Force: true}, time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	// Log required tool calls
	err = state.LogToolCall(dir, "EnterPlanMode", time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	err = state.LogToolCall(dir, "ExitPlanMode", time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	// Complete step successfully
	_, err = step.Complete(dir, step.CompleteOpts{})
	g.Expect(err).ToNot(HaveOccurred())

	// Tool calls should be cleared for next phase
	calls, err := state.GetToolCalls(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(calls).To(HaveLen(0))
}

// TestStepComplete_NonPlanPhases_DoNotRequireToolCalls verifies other phases unaffected
func TestStepComplete_NonPlanPhases_DoNotRequireToolCalls(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Test a simple non-plan phase
	_, err := state.Init(dir, "test-project", func() time.Time { return time.Now() })
	g.Expect(err).ToNot(HaveOccurred())

	// Use tasklist_create which is a simple non-plan phase
	_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{Force: true}, time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	// Complete step without any tool calls - should succeed
	result, err := step.Complete(dir, step.CompleteOpts{})
	// Validation should pass (no error) because tasklist_create is not plan_produce
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Valid).To(BeTrue())
	g.Expect(result.Error).To(BeEmpty())
}

// TestStepComplete_PlanProduce_OrderDoesNotMatter verifies tool calls can be in any order
func TestStepComplete_PlanProduce_OrderDoesNotMatter(t *testing.T) {
	testCases := []struct {
		name  string
		order []string
	}{
		{
			name:  "Enter then Exit",
			order: []string{"EnterPlanMode", "ExitPlanMode"},
		},
		{
			name:  "Exit then Enter (unusual but valid)",
			order: []string{"ExitPlanMode", "EnterPlanMode"},
		},
		{
			name:  "With other tools in between",
			order: []string{"EnterPlanMode", "Read", "Grep", "ExitPlanMode"},
		},
		{
			name:  "Multiple Enter/Exit pairs",
			order: []string{"EnterPlanMode", "ExitPlanMode", "EnterPlanMode", "ExitPlanMode"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			dir := t.TempDir()

			_, err := state.Init(dir, "test-project", func() time.Time { return time.Now() })
			g.Expect(err).ToNot(HaveOccurred())

	_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{Force: true}, time.Now)
	g.Expect(err).ToNot(HaveOccurred())

			_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{Force: true}, time.Now)
			g.Expect(err).ToNot(HaveOccurred())

			// Log tools in specified order
			for _, tool := range tc.order {
				err = state.LogToolCall(dir, tool, time.Now)
				g.Expect(err).ToNot(HaveOccurred())
			}

			// Complete should succeed as long as both required tools are present
			result, err := step.Complete(dir, step.CompleteOpts{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(result.Valid).To(BeTrue())
		})
	}
}
