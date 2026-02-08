package state_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
)

func TestIsLegalTransitionDelegatesToWorkflowConfig(t *testing.T) {
	t.Run("valid transition in new workflow", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(state.IsLegalTransition("plan_produce", "plan_approve", "new")).To(BeTrue())
	})

	t.Run("invalid transition in new workflow", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(state.IsLegalTransition("plan_produce", "artifact_commit", "new")).To(BeFalse())
	})

	t.Run("TDD transitions in new workflow", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(state.IsLegalTransition("tdd_red_produce", "tdd_red_qa", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("tdd_red_qa", "tdd_red_decide", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("tdd_red_decide", "tdd_red_produce", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("tdd_red_decide", "tdd_green_produce", "new")).To(BeTrue())
	})

	t.Run("TDD transitions in scoped workflow", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(state.IsLegalTransition("tdd_red_produce", "tdd_red_qa", "scoped")).To(BeTrue())
		g.Expect(state.IsLegalTransition("tdd_green_decide", "tdd_refactor_produce", "scoped")).To(BeTrue())
	})

	t.Run("plan transitions not in scoped workflow", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(state.IsLegalTransition("plan_produce", "plan_approve", "scoped")).To(BeFalse())
	})

	t.Run("unknown workflow returns false", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(state.IsLegalTransition("plan_produce", "plan_approve", "nonexistent")).To(BeFalse())
	})

	t.Run("unknown state returns false", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(state.IsLegalTransition("nonexistent", "plan_approve", "new")).To(BeFalse())
	})
}

func TestLegalTargetsDelegatesToWorkflowConfig(t *testing.T) {
	t.Run("returns targets for plan_produce in new workflow", func(t *testing.T) {
		g := NewWithT(t)
		targets := state.LegalTargets("plan_produce", "new")
		g.Expect(targets).To(Equal([]string{"plan_approve"}))
	})

	t.Run("returns nil for unknown state", func(t *testing.T) {
		g := NewWithT(t)
		targets := state.LegalTargets("nonexistent", "new")
		g.Expect(targets).To(BeNil())
	})

	t.Run("returns nil for unknown workflow", func(t *testing.T) {
		g := NewWithT(t)
		targets := state.LegalTargets("plan_produce", "nonexistent")
		g.Expect(targets).To(BeNil())
	})

	t.Run("decide states have multiple targets", func(t *testing.T) {
		g := NewWithT(t)
		targets := state.LegalTargets("tdd_red_decide", "new")
		g.Expect(len(targets)).To(BeNumerically(">=", 2))
		g.Expect(targets).To(ContainElement("tdd_red_produce"))
		g.Expect(targets).To(ContainElement("tdd_green_produce"))
	})

	t.Run("complete state has no targets", func(t *testing.T) {
		g := NewWithT(t)
		targets := state.LegalTargets("complete", "new")
		g.Expect(targets).To(BeEmpty())
	})
}
