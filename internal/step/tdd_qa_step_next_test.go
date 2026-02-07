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

// navigateToPhaseForStepNext is a helper to reach a specific phase for step next testing.
func navigateToPhaseForStepNext(t *testing.T, dir string, targetPhase string) {
	t.Helper()
	g := NewWithT(t)

	allPhases := []string{
		"pm", "pm-complete", "design", "design-complete",
		"architect", "architect-complete", "breakdown", "breakdown-complete",
		"implementation", "task-start", "tdd-red", "tdd-red-qa",
		"tdd-green", "tdd-green-qa",
		"tdd-refactor", "tdd-refactor-qa",
	}

	targetIdx := -1
	for i, phase := range allPhases {
		if phase == targetPhase {
			targetIdx = i
			break
		}
	}

	if targetIdx == -1 {
		t.Fatalf("target phase %s not found in sequence", targetPhase)
	}

	for i := 0; i <= targetIdx; i++ {
		_, err := state.Transition(dir, allPhases[i], state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred(), "failed to transition to %s", allPhases[i])
	}
}

func TestStepNextTDDRedQA(t *testing.T) {
	t.Run("tdd-red-qa returns spawn-qa action with qa skill", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		navigateToPhaseForStepNext(t, dir, "tdd-red-qa")

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))
		g.Expect(result.Skill).To(Equal("qa"))
		g.Expect(result.SkillPath).To(Equal("skills/qa/SKILL.md"))
	})
}

func TestStepNextTDDGreenQA(t *testing.T) {
	t.Run("tdd-green-qa returns spawn-qa action with qa skill", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		navigateToPhaseForStepNext(t, dir, "tdd-green-qa")

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))
		g.Expect(result.Skill).To(Equal("qa"))
		g.Expect(result.SkillPath).To(Equal("skills/qa/SKILL.md"))
	})
}

func TestStepNextTDDRefactorQA(t *testing.T) {
	t.Run("tdd-refactor-qa returns spawn-qa action with qa skill", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		navigateToPhaseForStepNext(t, dir, "tdd-refactor-qa")

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))
		g.Expect(result.Skill).To(Equal("qa"))
		g.Expect(result.SkillPath).To(Equal("skills/qa/SKILL.md"))
	})
}
