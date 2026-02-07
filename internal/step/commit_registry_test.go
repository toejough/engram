package step_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/step"
)

// TestCommitPhaseRegistry verifies TASK-3 acceptance criteria:
// Commit phase entries are registered correctly.
// Traces: ARCH-034, ARCH-035

func TestCommitRedRegistry(t *testing.T) {
	t.Run("commit-red phase is registered", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("commit-red")
		g.Expect(ok).To(BeTrue(), "commit-red should be registered")
		g.Expect(info.Producer).To(Equal("commit-producer"))
		g.Expect(info.ProducerPath).To(Equal("skills/commit-producer/SKILL.md"))
		g.Expect(info.QA).To(Equal("qa"))
		g.Expect(info.QAPath).To(Equal("skills/qa/SKILL.md"))
		g.Expect(info.ProducerModel).To(Equal("haiku"))
		g.Expect(info.QAModel).To(Equal("haiku"))
		g.Expect(info.CompletionPhase).To(Equal("commit-red-qa"))
	})
}

func TestCommitGreenRegistry(t *testing.T) {
	t.Run("commit-green phase is registered", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("commit-green")
		g.Expect(ok).To(BeTrue(), "commit-green should be registered")
		g.Expect(info.Producer).To(Equal("commit-producer"))
		g.Expect(info.ProducerPath).To(Equal("skills/commit-producer/SKILL.md"))
		g.Expect(info.QA).To(Equal("qa"))
		g.Expect(info.QAPath).To(Equal("skills/qa/SKILL.md"))
		g.Expect(info.ProducerModel).To(Equal("haiku"))
		g.Expect(info.QAModel).To(Equal("haiku"))
		g.Expect(info.CompletionPhase).To(Equal("commit-green-qa"))
	})
}

func TestCommitRefactorRegistry(t *testing.T) {
	t.Run("commit-refactor phase is registered", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("commit-refactor")
		g.Expect(ok).To(BeTrue(), "commit-refactor should be registered")
		g.Expect(info.Producer).To(Equal("commit-producer"))
		g.Expect(info.ProducerPath).To(Equal("skills/commit-producer/SKILL.md"))
		g.Expect(info.QA).To(Equal("qa"))
		g.Expect(info.QAPath).To(Equal("skills/qa/SKILL.md"))
		g.Expect(info.ProducerModel).To(Equal("haiku"))
		g.Expect(info.QAModel).To(Equal("haiku"))
		g.Expect(info.CompletionPhase).To(Equal("commit-refactor-qa"))
	})
}

func TestRegistryPhasesIncludesCommitPhases(t *testing.T) {
	t.Run("Registry.Phases() includes all commit phases", func(t *testing.T) {
		g := NewWithT(t)

		phases := step.Registry.Phases()
		g.Expect(phases).To(ContainElements(
			"commit-red", "commit-green", "commit-refactor",
		))
	})
}
