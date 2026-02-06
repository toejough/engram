package step_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/step"
	"pgregory.net/rapid"
)

func TestRegistryLookup(t *testing.T) {
	t.Run("returns phase info for known phase", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("pm")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("pm-interview-producer"))
		g.Expect(info.ProducerPath).To(Equal("skills/pm-interview-producer/SKILL.md"))
		g.Expect(info.QA).To(Equal("qa"))
		g.Expect(info.QAPath).To(Equal("skills/qa/SKILL.md"))
		g.Expect(info.Artifact).To(Equal("requirements.md"))
		g.Expect(info.IDFormat).To(Equal("REQ"))
		g.Expect(info.ProducerModel).To(Equal("sonnet"))
		g.Expect(info.QAModel).To(Equal("haiku"))
	})

	t.Run("returns false for unknown phase", func(t *testing.T) {
		g := NewWithT(t)

		_, ok := step.Registry.Lookup("nonexistent")
		g.Expect(ok).To(BeFalse())
	})

	t.Run("design phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("design")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("design-interview-producer"))
		g.Expect(info.Artifact).To(Equal("design.md"))
		g.Expect(info.IDFormat).To(Equal("DES"))
	})

	t.Run("architect phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("architect")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("arch-interview-producer"))
		g.Expect(info.Artifact).To(Equal("architecture.md"))
		g.Expect(info.IDFormat).To(Equal("ARCH"))
	})

	t.Run("breakdown phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("breakdown")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("breakdown-producer"))
		g.Expect(info.Artifact).To(Equal("tasks.md"))
		g.Expect(info.IDFormat).To(Equal("TASK"))
	})

	t.Run("tdd-red phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("tdd-red")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("tdd-red-producer"))
	})

	t.Run("tdd-green phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("tdd-green")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("tdd-green-producer"))
	})

	t.Run("tdd-refactor phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("tdd-refactor")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("tdd-refactor-producer"))
	})

	t.Run("alignment phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("alignment")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("alignment-producer"))
	})

	t.Run("retro phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("retro")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("retro-producer"))
	})

	t.Run("summary phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("summary")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("summary-producer"))
	})

	t.Run("documentation phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("documentation")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("doc-producer"))
	})

	t.Run("adopt-infer-tests phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("adopt-infer-tests")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("tdd-red-infer-producer"))
		g.Expect(info.CompletionPhase).To(Equal("adopt-infer-arch"))
	})

	t.Run("adopt-infer-arch phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("adopt-infer-arch")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("arch-infer-producer"))
		g.Expect(info.Artifact).To(Equal("architecture.md"))
		g.Expect(info.CompletionPhase).To(Equal("adopt-infer-design"))
	})

	t.Run("adopt-infer-design phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("adopt-infer-design")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("design-infer-producer"))
		g.Expect(info.Artifact).To(Equal("design.md"))
		g.Expect(info.CompletionPhase).To(Equal("adopt-infer-reqs"))
	})

	t.Run("adopt-infer-reqs phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("adopt-infer-reqs")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("pm-infer-producer"))
		g.Expect(info.Artifact).To(Equal("requirements.md"))
		g.Expect(info.CompletionPhase).To(Equal("adopt-escalations"))
	})

	t.Run("adopt-documentation phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("adopt-documentation")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("doc-producer"))
		g.Expect(info.CompletionPhase).To(Equal("alignment"))
	})

	t.Run("align-infer-tests phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("align-infer-tests")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("tdd-red-infer-producer"))
		g.Expect(info.CompletionPhase).To(Equal("align-infer-arch"))
	})

	t.Run("align-infer-arch phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("align-infer-arch")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("arch-infer-producer"))
		g.Expect(info.CompletionPhase).To(Equal("align-infer-design"))
	})

	t.Run("align-infer-design phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("align-infer-design")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("design-infer-producer"))
		g.Expect(info.CompletionPhase).To(Equal("align-infer-reqs"))
	})

	t.Run("align-infer-reqs phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("align-infer-reqs")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("pm-infer-producer"))
		g.Expect(info.CompletionPhase).To(Equal("align-escalations"))
	})

	t.Run("align-documentation phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("align-documentation")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("doc-producer"))
		g.Expect(info.CompletionPhase).To(Equal("alignment"))
	})

	t.Run("task-documentation phase has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("task-documentation")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("doc-producer"))
		g.Expect(info.CompletionPhase).To(Equal("alignment"))
	})
}

func TestRegistryPhases(t *testing.T) {
	t.Run("returns all registered phases", func(t *testing.T) {
		g := NewWithT(t)

		phases := step.Registry.Phases()
		// New project workflow
		g.Expect(phases).To(ContainElements("pm", "design", "architect", "breakdown"))
		g.Expect(phases).To(ContainElements("tdd-red", "tdd-green", "tdd-refactor"))
		g.Expect(phases).To(ContainElements("alignment", "retro", "summary", "documentation"))
		// Adopt workflow
		g.Expect(phases).To(ContainElements(
			"adopt-infer-tests", "adopt-infer-arch", "adopt-infer-design",
			"adopt-infer-reqs", "adopt-documentation",
		))
		// Align workflow
		g.Expect(phases).To(ContainElements(
			"align-infer-tests", "align-infer-arch", "align-infer-design",
			"align-infer-reqs", "align-documentation",
		))
		// Task workflow
		g.Expect(phases).To(ContainElement("task-documentation"))
	})
}

func TestRegistryProperty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		phases := step.Registry.Phases()
		phase := rapid.SampledFrom(phases).Draw(rt, "phase")

		info, ok := step.Registry.Lookup(phase)
		g.Expect(ok).To(BeTrue(), "registered phase %q should be found via Lookup", phase)

		// Every phase must have a producer skill
		g.Expect(info.Producer).ToNot(BeEmpty(), "phase %q must have a producer", phase)
		g.Expect(info.ProducerPath).To(HaveSuffix("SKILL.md"), "phase %q producer path must end with SKILL.md", phase)

		// Every phase must have a QA skill
		g.Expect(info.QA).To(Equal("qa"), "phase %q must use the universal qa skill", phase)
		g.Expect(info.QAPath).To(Equal("skills/qa/SKILL.md"), "phase %q QA path must be skills/qa/SKILL.md", phase)

		// Every phase must have models
		g.Expect(info.ProducerModel).ToNot(BeEmpty(), "phase %q must have a producer model", phase)
		g.Expect(info.QAModel).To(Equal("haiku"), "phase %q must use haiku for QA", phase)
	})
}
