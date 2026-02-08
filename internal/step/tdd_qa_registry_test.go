package step_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/step"
	"github.com/toejough/projctl/internal/workflow"
)

// TestTDDQAPhaseRegistry verifies TDD sub-phase QA entries are registered correctly.

func TestTDDRedQARegistry(t *testing.T) {
	t.Run("tdd_red_qa state is registered as qa type", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("tdd_red_qa")
		g.Expect(ok).To(BeTrue(), "tdd_red_qa should be registered")
		g.Expect(info.StateType).To(Equal(workflow.StateTypeQA))
		g.Expect(info.QA).To(Equal("qa"))
		g.Expect(info.QAPath).To(Equal("skills/qa/SKILL.md"))
		g.Expect(info.QAModel).ToNot(BeEmpty())
	})
}

func TestTDDGreenQARegistry(t *testing.T) {
	t.Run("tdd_green_qa state is registered as qa type", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("tdd_green_qa")
		g.Expect(ok).To(BeTrue(), "tdd_green_qa should be registered")
		g.Expect(info.StateType).To(Equal(workflow.StateTypeQA))
		g.Expect(info.QA).To(Equal("qa"))
		g.Expect(info.QAPath).To(Equal("skills/qa/SKILL.md"))
	})
}

func TestTDDRefactorQARegistry(t *testing.T) {
	t.Run("tdd_refactor_qa state is registered as qa type", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("tdd_refactor_qa")
		g.Expect(ok).To(BeTrue(), "tdd_refactor_qa should be registered")
		g.Expect(info.StateType).To(Equal(workflow.StateTypeQA))
		g.Expect(info.QA).To(Equal("qa"))
		g.Expect(info.QAPath).To(Equal("skills/qa/SKILL.md"))
	})
}

func TestRegistryPhasesIncludesQAPhases(t *testing.T) {
	t.Run("Registry.Phases() includes all TDD QA phases", func(t *testing.T) {
		g := NewWithT(t)

		phases := step.Registry.Phases()
		g.Expect(phases).To(ContainElements(
			"tdd_red_qa", "tdd_green_qa", "tdd_refactor_qa",
		))
	})
}
