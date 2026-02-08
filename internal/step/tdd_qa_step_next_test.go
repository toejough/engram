package step_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
	"github.com/toejough/projctl/internal/step"
)

// TestTDDQAStepNext verifies TASK-19 acceptance criteria:
// `step next` returns QA actions for TDD sub-phase QA phases.
// Traces: ARCH-037

// navigateToPhaseForStepNext walks the scoped workflow state machine
// to reach a target state via direct state.Transition calls.
//
// Scoped workflow path:
//
//	init -> item_select -> item_fork -> worktree_create ->
//	  tdd_red_produce -> tdd_red_qa -> tdd_red_decide -> (approved: tdd_green_produce)
//	  tdd_green_produce -> tdd_green_qa -> tdd_green_decide -> (approved: tdd_refactor_produce)
//	  tdd_refactor_produce -> tdd_refactor_qa
func navigateToPhaseForStepNext(t *testing.T, dir string, targetPhase string) {
	t.Helper()
	g := NewWithT(t)

	// The scoped workflow linear path through the TDD loop.
	// For decide states, the "approved" transition is targets[1] which goes
	// to the next produce state (e.g., tdd_red_decide -> tdd_green_produce).
	allPhases := []string{
		"tasklist_create", "item_select", "item_fork", "worktree_create",
		"tdd_red_produce", "tdd_red_qa", "tdd_red_decide", "tdd_green_produce",
		"tdd_green_qa", "tdd_green_decide", "tdd_refactor_produce",
		"tdd_refactor_qa",
	}

	targetIdx := -1
	for i, phase := range allPhases {
		if phase == targetPhase {
			targetIdx = i
			break
		}
	}

	if targetIdx == -1 {
		t.Fatalf("target phase %s not found in scoped workflow sequence", targetPhase)
	}

	for i := 0; i <= targetIdx; i++ {
		_, err := state.Transition(dir, allPhases[i], state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred(), "failed to transition to %s", allPhases[i])
	}
}

func TestStepNextTDDRedQA(t *testing.T) {
	t.Run("tdd_red_qa returns spawn-qa action with qa skill", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "scoped",
		})
		g.Expect(err).ToNot(HaveOccurred())

		navigateToPhaseForStepNext(t, dir, "tdd_red_qa")

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))
		g.Expect(result.Skill).To(Equal("qa"))
		g.Expect(result.SkillPath).To(Equal("skills/qa/SKILL.md"))
	})
}

func TestStepNextTDDGreenQA(t *testing.T) {
	t.Run("tdd_green_qa returns spawn-qa action with qa skill", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "scoped",
		})
		g.Expect(err).ToNot(HaveOccurred())

		navigateToPhaseForStepNext(t, dir, "tdd_green_qa")

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))
		g.Expect(result.Skill).To(Equal("qa"))
		g.Expect(result.SkillPath).To(Equal("skills/qa/SKILL.md"))
	})
}

func TestStepNextTDDRefactorQA(t *testing.T) {
	t.Run("tdd_refactor_qa returns spawn-qa action with qa skill", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "scoped",
		})
		g.Expect(err).ToNot(HaveOccurred())

		navigateToPhaseForStepNext(t, dir, "tdd_refactor_qa")

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))
		g.Expect(result.Skill).To(Equal("qa"))
		g.Expect(result.SkillPath).To(Equal("skills/qa/SKILL.md"))
	})
}
