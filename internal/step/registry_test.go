package step_test

import (
	"fmt"
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
		g.Expect(info.ProducerModel).ToNot(BeEmpty())
		g.Expect(info.QAModel).ToNot(BeEmpty())
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

		// Every phase must have non-empty models
		g.Expect(info.ProducerModel).ToNot(BeEmpty(), "phase %q must have a producer model", phase)
		g.Expect(info.QAModel).ToNot(BeEmpty(), "phase %q must have a QA model", phase)
	})
}

// === ParseSkillModel tests ===

func TestParseSkillModel(t *testing.T) {
	t.Run("extracts model from valid frontmatter", func(t *testing.T) {
		g := NewWithT(t)

		content := []byte("---\nname: test-skill\nmodel: opus\n---\n# Body")
		g.Expect(step.ParseSkillModel(content)).To(Equal("opus"))
	})

	t.Run("returns empty for missing model field", func(t *testing.T) {
		g := NewWithT(t)

		content := []byte("---\nname: test-skill\ndescription: no model here\n---\n# Body")
		g.Expect(step.ParseSkillModel(content)).To(BeEmpty())
	})

	t.Run("returns empty for no frontmatter", func(t *testing.T) {
		g := NewWithT(t)

		content := []byte("# Just markdown, no frontmatter")
		g.Expect(step.ParseSkillModel(content)).To(BeEmpty())
	})

	t.Run("returns empty for unclosed frontmatter", func(t *testing.T) {
		g := NewWithT(t)

		content := []byte("---\nmodel: sonnet\n# Missing closing delimiter")
		g.Expect(step.ParseSkillModel(content)).To(BeEmpty())
	})

	t.Run("returns empty for empty content", func(t *testing.T) {
		g := NewWithT(t)

		g.Expect(step.ParseSkillModel([]byte(""))).To(BeEmpty())
	})

	t.Run("handles model with extra whitespace", func(t *testing.T) {
		g := NewWithT(t)

		content := []byte("---\nmodel:   sonnet  \n---\n")
		g.Expect(step.ParseSkillModel(content)).To(Equal("sonnet"))
	})

	t.Run("returns empty for model field with empty value", func(t *testing.T) {
		g := NewWithT(t)

		content := []byte("---\nmodel:\n---\n")
		g.Expect(step.ParseSkillModel(content)).To(BeEmpty())
	})

	t.Run("does not match model-like keys", func(t *testing.T) {
		g := NewWithT(t)

		// "model_name" starts with "model" but should not match "model:"
		content := []byte("---\nmodel_name: opus\n---\n")
		g.Expect(step.ParseSkillModel(content)).To(BeEmpty())
	})
}

// === NewRegistry tests ===

func TestNewRegistry(t *testing.T) {
	t.Run("resolves model from SKILL.md frontmatter", func(t *testing.T) {
		g := NewWithT(t)

		readFunc := func(path string) ([]byte, error) {
			switch path {
			case "skills/pm-interview-producer/SKILL.md":
				return []byte("---\nname: pm-producer\nmodel: opus\n---\n"), nil
			case "skills/qa/SKILL.md":
				return []byte("---\nname: qa\nmodel: haiku\n---\n"), nil
			default:
				return nil, fmt.Errorf("not found: %s", path)
			}
		}

		reg := step.NewRegistry(readFunc)
		info, ok := reg.Lookup("pm")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.ProducerModel).To(Equal("opus"))
		g.Expect(info.QAModel).To(Equal("haiku"))
	})

	t.Run("falls back to default when file cannot be read", func(t *testing.T) {
		g := NewWithT(t)

		readFunc := func(path string) ([]byte, error) {
			return nil, fmt.Errorf("file not found: %s", path)
		}

		reg := step.NewRegistry(readFunc)
		info, ok := reg.Lookup("pm")
		g.Expect(ok).To(BeTrue())
		// Fallback values from phaseDefinitions
		g.Expect(info.ProducerModel).To(Equal("opus"))
		g.Expect(info.QAModel).To(Equal("haiku"))
	})

	t.Run("falls back to default when model field missing", func(t *testing.T) {
		g := NewWithT(t)

		readFunc := func(path string) ([]byte, error) {
			return []byte("---\nname: some-skill\n---\n"), nil
		}

		reg := step.NewRegistry(readFunc)
		info, ok := reg.Lookup("tdd-green")
		g.Expect(ok).To(BeTrue())
		// Fallback for tdd-green producer is "sonnet"
		g.Expect(info.ProducerModel).To(Equal("sonnet"))
	})

	t.Run("frontmatter model overrides fallback", func(t *testing.T) {
		g := NewWithT(t)

		readFunc := func(path string) ([]byte, error) {
			// Return a custom model that differs from fallback
			return []byte("---\nname: skill\nmodel: custom-model\n---\n"), nil
		}

		reg := step.NewRegistry(readFunc)
		info, ok := reg.Lookup("pm")
		g.Expect(ok).To(BeTrue())
		// Both producer and QA paths should resolve to custom-model
		g.Expect(info.ProducerModel).To(Equal("custom-model"))
		g.Expect(info.QAModel).To(Equal("custom-model"))
	})
}

func TestNewRegistryProperty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		// Generate a random model name
		model := rapid.StringMatching(`[a-z]+-[0-9]+`).Draw(rt, "model")

		readFunc := func(path string) ([]byte, error) {
			return []byte("---\nmodel: " + model + "\n---\n"), nil
		}

		reg := step.NewRegistry(readFunc)
		phases := reg.Phases()

		// Property: all phases resolve to the model from frontmatter
		for _, phase := range phases {
			info, ok := reg.Lookup(phase)
			g.Expect(ok).To(BeTrue())
			g.Expect(info.ProducerModel).To(Equal(model),
				"phase %q producer model should come from frontmatter", phase)
			g.Expect(info.QAModel).To(Equal(model),
				"phase %q QA model should come from frontmatter", phase)
		}
	})
}

func TestNewRegistryAllPhasesResolveNonEmptyModel(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		// Randomly decide if files are readable or not
		readable := rapid.Bool().Draw(rt, "readable")

		readFunc := func(path string) ([]byte, error) {
			if readable {
				return []byte("---\nmodel: sonnet\n---\n"), nil
			}
			return nil, fmt.Errorf("not found")
		}

		reg := step.NewRegistry(readFunc)
		phases := reg.Phases()

		phase := rapid.SampledFrom(phases).Draw(rt, "phase")
		info, ok := reg.Lookup(phase)
		g.Expect(ok).To(BeTrue())
		g.Expect(info.ProducerModel).ToNot(BeEmpty(),
			"phase %q must always have a non-empty producer model", phase)
		g.Expect(info.QAModel).ToNot(BeEmpty(),
			"phase %q must always have a non-empty QA model", phase)
	})
}
