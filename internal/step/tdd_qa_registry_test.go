package step_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/step"
)

// TestTDDQAPhaseRegistry verifies TASK-18 acceptance criteria:
// TDD sub-phase QA entries are registered correctly.
// Traces: ARCH-034, ARCH-037

func TestTDDRedQARegistry(t *testing.T) {
	t.Run("tdd-red-qa phase is registered", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("tdd-red-qa")
		g.Expect(ok).To(BeTrue(), "tdd-red-qa should be registered")
		g.Expect(info.Producer).To(Equal("qa"))
		g.Expect(info.ProducerPath).To(Equal("skills/qa/SKILL.md"))
		g.Expect(info.QA).To(Equal("qa"))
		g.Expect(info.QAPath).To(Equal("skills/qa/SKILL.md"))
		g.Expect(info.ProducerModel).To(Equal("haiku"))
		g.Expect(info.QAModel).To(Equal("haiku"))
	})
}

// Traces: ARCH-034, ARCH-037
func TestTDDGreenQARegistry(t *testing.T) {
	t.Run("tdd-green-qa phase is registered", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("tdd-green-qa")
		g.Expect(ok).To(BeTrue(), "tdd-green-qa should be registered")
		g.Expect(info.Producer).To(Equal("qa"))
		g.Expect(info.ProducerPath).To(Equal("skills/qa/SKILL.md"))
		g.Expect(info.QA).To(Equal("qa"))
		g.Expect(info.QAPath).To(Equal("skills/qa/SKILL.md"))
		g.Expect(info.ProducerModel).To(Equal("haiku"))
		g.Expect(info.QAModel).To(Equal("haiku"))
	})
}

// Traces: ARCH-034, ARCH-037
func TestTDDRefactorQARegistry(t *testing.T) {
	t.Run("tdd-refactor-qa phase is registered", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("tdd-refactor-qa")
		g.Expect(ok).To(BeTrue(), "tdd-refactor-qa should be registered")
		g.Expect(info.Producer).To(Equal("qa"))
		g.Expect(info.ProducerPath).To(Equal("skills/qa/SKILL.md"))
		g.Expect(info.QA).To(Equal("qa"))
		g.Expect(info.QAPath).To(Equal("skills/qa/SKILL.md"))
		g.Expect(info.ProducerModel).To(Equal("haiku"))
		g.Expect(info.QAModel).To(Equal("haiku"))
	})
}

// Traces: ARCH-035, ARCH-037
func TestCommitRedQARegistry(t *testing.T) {
	t.Run("commit-red-qa phase is registered", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("commit-red-qa")
		g.Expect(ok).To(BeTrue(), "commit-red-qa should be registered")
		g.Expect(info.Producer).To(Equal("qa"))
		g.Expect(info.ProducerPath).To(Equal("skills/qa/SKILL.md"))
		g.Expect(info.QA).To(Equal("qa"))
		g.Expect(info.QAPath).To(Equal("skills/qa/SKILL.md"))
		g.Expect(info.ProducerModel).To(Equal("haiku"))
		g.Expect(info.QAModel).To(Equal("haiku"))
	})
}

// Traces: ARCH-035, ARCH-037
func TestCommitGreenQARegistry(t *testing.T) {
	t.Run("commit-green-qa phase is registered", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("commit-green-qa")
		g.Expect(ok).To(BeTrue(), "commit-green-qa should be registered")
		g.Expect(info.Producer).To(Equal("qa"))
		g.Expect(info.ProducerPath).To(Equal("skills/qa/SKILL.md"))
		g.Expect(info.QA).To(Equal("qa"))
		g.Expect(info.QAPath).To(Equal("skills/qa/SKILL.md"))
		g.Expect(info.ProducerModel).To(Equal("haiku"))
		g.Expect(info.QAModel).To(Equal("haiku"))
	})
}

// Traces: ARCH-035, ARCH-037
func TestCommitRefactorQARegistry(t *testing.T) {
	t.Run("commit-refactor-qa phase is registered", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("commit-refactor-qa")
		g.Expect(ok).To(BeTrue(), "commit-refactor-qa should be registered")
		g.Expect(info.Producer).To(Equal("qa"))
		g.Expect(info.ProducerPath).To(Equal("skills/qa/SKILL.md"))
		g.Expect(info.QA).To(Equal("qa"))
		g.Expect(info.QAPath).To(Equal("skills/qa/SKILL.md"))
		g.Expect(info.ProducerModel).To(Equal("haiku"))
		g.Expect(info.QAModel).To(Equal("haiku"))
	})
}

func TestRegistryPhasesIncludesQAPhases(t *testing.T) {
	t.Run("Registry.Phases() includes all TDD and commit QA phases", func(t *testing.T) {
		g := NewWithT(t)

		phases := step.Registry.Phases()
		g.Expect(phases).To(ContainElements(
			"tdd-red-qa", "tdd-green-qa", "tdd-refactor-qa",
			"commit-red-qa", "commit-green-qa", "commit-refactor-qa",
		))
	})
}
