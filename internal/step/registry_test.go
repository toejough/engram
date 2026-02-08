package step_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/step"
	"github.com/toejough/projctl/internal/workflow"
	"pgregory.net/rapid"
)

func TestRegistryLookup(t *testing.T) {
	t.Run("returns phase info for known produce state", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("pm_produce")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("pm-interview-producer"))
		g.Expect(info.ProducerPath).To(Equal("skills/pm-interview-producer/SKILL.md"))
		g.Expect(info.Artifact).To(Equal("requirements.md"))
		g.Expect(info.IDFormat).To(Equal("REQ"))
		g.Expect(info.ProducerModel).ToNot(BeEmpty())
		g.Expect(info.StateType).To(Equal(workflow.StateTypeProduce))
	})

	t.Run("returns phase info for known qa state", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("pm_qa")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.QA).To(Equal("qa"))
		g.Expect(info.QAPath).To(Equal("skills/qa/SKILL.md"))
		g.Expect(info.QAModel).ToNot(BeEmpty())
		g.Expect(info.StateType).To(Equal(workflow.StateTypeQA))
	})

	t.Run("returns false for unknown phase", func(t *testing.T) {
		g := NewWithT(t)

		_, ok := step.Registry.Lookup("nonexistent")
		g.Expect(ok).To(BeFalse())
	})

	t.Run("design produce state has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("design_produce")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("design-interview-producer"))
		g.Expect(info.Artifact).To(Equal("design.md"))
		g.Expect(info.IDFormat).To(Equal("DES"))
	})

	t.Run("arch produce state has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("arch_produce")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("arch-interview-producer"))
		g.Expect(info.Artifact).To(Equal("architecture.md"))
		g.Expect(info.IDFormat).To(Equal("ARCH"))
	})

	t.Run("breakdown produce state has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("breakdown_produce")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("breakdown-producer"))
		g.Expect(info.Artifact).To(Equal("tasks.md"))
		g.Expect(info.IDFormat).To(Equal("TASK"))
	})

	t.Run("tdd_red_produce state has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("tdd_red_produce")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("tdd-red-producer"))
		g.Expect(info.StateType).To(Equal(workflow.StateTypeProduce))
	})

	t.Run("tdd_green_produce state has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("tdd_green_produce")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("tdd-green-producer"))
	})

	t.Run("tdd_refactor_produce state has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("tdd_refactor_produce")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("tdd-refactor-producer"))
	})

	t.Run("retro produce state has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("retro_produce")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("retro-producer"))
		g.Expect(info.Artifact).To(Equal("retro.md"))
	})

	t.Run("documentation produce state has correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("documentation_produce")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("doc-producer"))
	})

	t.Run("align infer states have correct info", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("align_infer_tests_produce")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("tdd-red-infer-producer"))

		info, ok = step.Registry.Lookup("align_infer_arch_produce")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("arch-infer-producer"))
		g.Expect(info.Artifact).To(Equal("architecture.md"))

		info, ok = step.Registry.Lookup("align_infer_design_produce")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("design-infer-producer"))
		g.Expect(info.Artifact).To(Equal("design.md"))

		info, ok = step.Registry.Lookup("align_infer_reqs_produce")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.Producer).To(Equal("pm-infer-producer"))
		g.Expect(info.Artifact).To(Equal("requirements.md"))
	})

	t.Run("decide states have correct type", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("pm_decide")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.StateType).To(Equal(workflow.StateTypeDecide))
	})

	t.Run("commit states have correct type", func(t *testing.T) {
		g := NewWithT(t)

		info, ok := step.Registry.Lookup("pm_commit")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.StateType).To(Equal(workflow.StateTypeCommit))
	})
}

func TestRegistryPhases(t *testing.T) {
	t.Run("returns all registered phases from TOML", func(t *testing.T) {
		g := NewWithT(t)

		phases := step.Registry.Phases()
		// New project workflow produce states
		g.Expect(phases).To(ContainElements("pm_produce", "design_produce", "arch_produce", "breakdown_produce"))
		// TDD produce states
		g.Expect(phases).To(ContainElements("tdd_red_produce", "tdd_green_produce", "tdd_refactor_produce"))
		// QA states
		g.Expect(phases).To(ContainElements("pm_qa", "design_qa", "tdd_red_qa"))
		// Decide states
		g.Expect(phases).To(ContainElements("pm_decide", "tdd_red_decide"))
		// Commit states
		g.Expect(phases).To(ContainElements("pm_commit", "tdd_commit"))
		// Align workflow
		g.Expect(phases).To(ContainElements("align_infer_tests_produce", "align_infer_arch_produce"))
	})
}

func TestRegistryProduceStatesProperty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		phases := step.Registry.Phases()
		phase := rapid.SampledFrom(phases).Draw(rt, "phase")

		info, ok := step.Registry.Lookup(phase)
		g.Expect(ok).To(BeTrue(), "registered phase %q should be found via Lookup", phase)

		// Produce states must have a producer skill
		if info.StateType == workflow.StateTypeProduce {
			g.Expect(info.Producer).ToNot(BeEmpty(), "produce state %q must have a producer", phase)
			g.Expect(info.ProducerPath).To(HaveSuffix("SKILL.md"), "produce state %q producer path must end with SKILL.md", phase)
			g.Expect(info.ProducerModel).ToNot(BeEmpty(), "produce state %q must have a producer model", phase)
		}

		// QA states must have a QA skill
		if info.StateType == workflow.StateTypeQA {
			g.Expect(info.QA).ToNot(BeEmpty(), "qa state %q must have a QA skill", phase)
			g.Expect(info.QAPath).To(HaveSuffix("SKILL.md"), "qa state %q QA path must end with SKILL.md", phase)
			g.Expect(info.QAModel).ToNot(BeEmpty(), "qa state %q must have a QA model", phase)
		}
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
	t.Run("resolves model from SKILL.md frontmatter for produce states", func(t *testing.T) {
		g := NewWithT(t)

		readFunc := func(path string) ([]byte, error) {
			switch path {
			case "skills/pm-interview-producer/SKILL.md":
				return []byte("---\nname: pm-producer\nmodel: opus\n---\n"), nil
			default:
				return nil, fmt.Errorf("not found: %s", path)
			}
		}

		reg := step.NewRegistry(readFunc)
		info, ok := reg.Lookup("pm_produce")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.ProducerModel).To(Equal("opus"))
	})

	t.Run("falls back to TOML default_model when file cannot be read", func(t *testing.T) {
		g := NewWithT(t)

		readFunc := func(path string) ([]byte, error) {
			return nil, fmt.Errorf("file not found: %s", path)
		}

		reg := step.NewRegistry(readFunc)
		info, ok := reg.Lookup("pm_produce")
		g.Expect(ok).To(BeTrue())
		// Default value from TOML: default_model = "opus"
		g.Expect(info.ProducerModel).To(Equal("opus"))
	})

	t.Run("falls back when model field missing in SKILL.md", func(t *testing.T) {
		g := NewWithT(t)

		readFunc := func(path string) ([]byte, error) {
			return []byte("---\nname: some-skill\n---\n"), nil
		}

		reg := step.NewRegistry(readFunc)
		info, ok := reg.Lookup("tdd_green_produce")
		g.Expect(ok).To(BeTrue())
		// Default for tdd_green_produce is "sonnet"
		g.Expect(info.ProducerModel).To(Equal("sonnet"))
	})

	t.Run("frontmatter model overrides default", func(t *testing.T) {
		g := NewWithT(t)

		readFunc := func(path string) ([]byte, error) {
			return []byte("---\nname: skill\nmodel: custom-model\n---\n"), nil
		}

		reg := step.NewRegistry(readFunc)
		info, ok := reg.Lookup("pm_produce")
		g.Expect(ok).To(BeTrue())
		g.Expect(info.ProducerModel).To(Equal("custom-model"))
	})
}

func TestNewRegistryProduceModelsAlwaysNonEmpty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

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

		if info.StateType == workflow.StateTypeProduce {
			g.Expect(info.ProducerModel).ToNot(BeEmpty(),
				"produce state %q must always have a non-empty producer model", phase)
		}
		if info.StateType == workflow.StateTypeQA {
			g.Expect(info.QAModel).ToNot(BeEmpty(),
				"qa state %q must always have a non-empty QA model", phase)
		}
	})
}
