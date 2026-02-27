//go:build sqlite_fts5

package memory_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/memory"
)

func TestGenerateSkillContentFallback4Sections(t *testing.T) {
	t.Run("fallback template contains all 4 required sections", func(t *testing.T) {
		g := NewWithT(t)
		cluster := []memory.ClusterEntry{
			{ID: 1, Content: "Always check error returns"},
			{ID: 2, Content: "Handle nil pointer dereferences"},
			{ID: 3, Content: "Log errors with context"},
		}
		content, err := memory.GenerateSkillContentForTest("error handling", cluster, nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(content).To(ContainSubstring("## Overview"))
		g.Expect(content).To(ContainSubstring("## When to Use"))
		g.Expect(content).To(ContainSubstring("## Quick Reference"))
		g.Expect(content).To(ContainSubstring("## Common Mistakes"))
	})
}

// ============================================================================
// T010: Template 4-section body structure
// ============================================================================

func TestGenerateSkillTemplate4Sections(t *testing.T) {
	t.Run("template output contains all 4 required sections in order", func(t *testing.T) {
		g := NewWithT(t)
		content := memory.GenerateSkillTemplateForTest("error handling", "Always handle errors explicitly.")
		g.Expect(content).To(ContainSubstring("## Overview"))
		g.Expect(content).To(ContainSubstring("## When to Use"))
		g.Expect(content).To(ContainSubstring("## Quick Reference"))
		g.Expect(content).To(ContainSubstring("## Common Mistakes"))

		// Verify order
		overviewIdx := strings.Index(content, "## Overview")
		whenIdx := strings.Index(content, "## When to Use")
		quickIdx := strings.Index(content, "## Quick Reference")
		mistakesIdx := strings.Index(content, "## Common Mistakes")

		g.Expect(overviewIdx).To(BeNumerically("<", whenIdx))
		g.Expect(whenIdx).To(BeNumerically("<", quickIdx))
		g.Expect(quickIdx).To(BeNumerically("<", mistakesIdx))
	})
}

// ============================================================================
// T003: isGeneratedSkill() — dual guard check
// ============================================================================

func TestIsGeneratedSkill(t *testing.T) {
	t.Run("returns true for memory. prefix and generated=true", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(memory.IsGeneratedSkillForTest("memory.foo", true)).To(BeTrue())
	})

	t.Run("returns false for memory. prefix but generated=false", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(memory.IsGeneratedSkillForTest("memory.foo", false)).To(BeFalse())
	})

	t.Run("returns false for non-memory prefix with generated=true", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(memory.IsGeneratedSkillForTest("custom.foo", true)).To(BeFalse())
	})

	t.Run("returns false for empty name and generated=false", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(memory.IsGeneratedSkillForTest("", false)).To(BeFalse())
	})

	t.Run("returns false for empty name and generated=true", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(memory.IsGeneratedSkillForTest("", true)).To(BeFalse())
	})

	t.Run("returns false for mem: prefix (old format)", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(memory.IsGeneratedSkillForTest("mem:foo", true)).To(BeFalse())
	})
}

// ============================================================================
// T011: LLM CompileSkill JSON parsing
// ============================================================================

func TestParseCompileSkillJSON(t *testing.T) {
	t.Run("valid JSON extracts description and body separately", func(t *testing.T) {
		g := NewWithT(t)
		input := `{"description":"Use when handling errors","body":"## Overview\nError handling best practices."}`
		desc, body, err := memory.ParseCompileSkillJSONForTest(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(desc).To(Equal("Use when handling errors"))
		g.Expect(body).To(Equal("## Overview\nError handling best practices."))
	})

	t.Run("invalid JSON (plain markdown) returns error", func(t *testing.T) {
		g := NewWithT(t)
		input := "# Error Handling\n\nThis is just markdown."
		_, _, err := memory.ParseCompileSkillJSONForTest(input)
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("valid JSON with empty description returns empty string", func(t *testing.T) {
		g := NewWithT(t)
		input := `{"description":"","body":"## Overview\nContent here."}`
		desc, body, err := memory.ParseCompileSkillJSONForTest(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(desc).To(Equal(""))
		g.Expect(body).To(Equal("## Overview\nContent here."))
	})

	t.Run("valid JSON with null description returns empty string", func(t *testing.T) {
		g := NewWithT(t)
		input := `{"description":null,"body":"## Overview\nContent."}`
		desc, body, err := memory.ParseCompileSkillJSONForTest(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(desc).To(Equal(""))
		g.Expect(body).To(Equal("## Overview\nContent."))
	})
}

// ============================================================================
// T018: Integration test — pipeline produces compliant skills
// ============================================================================

func TestPipelineProducesCompliantSkills(t *testing.T) {
	t.Run("generateTriggerDescription + generateSkillTemplate produce compliant skill", func(t *testing.T) {
		g := NewWithT(t)

		theme := "error handling"
		content := "Always handle errors explicitly in Go."

		// Simulate the template path (no LLM)
		body := memory.GenerateSkillTemplateForTest(theme, content)
		description := memory.GenerateTriggerDescriptionForTest(theme, body)

		skill := &memory.GeneratedSkill{
			Slug:        "error-handling",
			Theme:       theme,
			Description: description,
			Content:     body,
		}

		result := memory.ValidateSkillComplianceForTest(skill)
		g.Expect(result.Issues).To(BeEmpty(), "template-path skill should pass all compliance checks")
		g.Expect(result.DescriptionOK).To(BeTrue())
		g.Expect(result.BodyStructureOK).To(BeTrue())
		g.Expect(result.BodyLengthOK).To(BeTrue())
		g.Expect(result.NamingOK).To(BeTrue())
	})

	t.Run("LLM JSON path with valid response produces compliant skill", func(t *testing.T) {
		g := NewWithT(t)

		llmOutput := `{"description":"Use when the user encounters error handling patterns or needs guidance on error handling.","body":"## Overview\nError handling best practices.\n\n## When to Use\nApply when writing Go code that returns errors.\n\n## Quick Reference\n1. Always check error returns.\n2. Wrap errors with context.\n\n## Common Mistakes\n- Ignoring returned errors.\n- Not wrapping errors with context."}`
		desc, body, err := memory.ParseCompileSkillJSONForTest(llmOutput)
		g.Expect(err).ToNot(HaveOccurred())

		skill := &memory.GeneratedSkill{
			Slug:        "error-handling",
			Theme:       "error handling",
			Description: desc,
			Content:     body,
		}

		result := memory.ValidateSkillComplianceForTest(skill)
		g.Expect(result.Issues).To(BeEmpty(), "LLM JSON-path skill should pass all compliance checks")
	})

	t.Run("LLM JSON path with empty description falls back to trigger description", func(t *testing.T) {
		g := NewWithT(t)

		llmOutput := `{"description":"","body":"## Overview\nContent.\n\n## When to Use\nWhen needed.\n\n## Quick Reference\n1. Do this.\n\n## Common Mistakes\n- Don't do that."}`
		desc, body, err := memory.ParseCompileSkillJSONForTest(llmOutput)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(desc).To(Equal(""))

		// Simulate the fallback: empty desc → generateTriggerDescription
		theme := "testing"
		desc = memory.GenerateTriggerDescriptionForTest(theme, body)

		skill := &memory.GeneratedSkill{
			Slug:        "testing",
			Theme:       theme,
			Description: desc,
			Content:     body,
		}

		result := memory.ValidateSkillComplianceForTest(skill)
		g.Expect(result.Issues).To(BeEmpty(), "fallback description should still pass compliance")
	})

	t.Run("LLM plain markdown fallback produces compliant skill via template", func(t *testing.T) {
		g := NewWithT(t)

		// LLM returns plain markdown (not JSON) — parseCompileSkillJSON fails
		plainOutput := "# Error Handling\n\nJust some markdown content."
		_, _, err := memory.ParseCompileSkillJSONForTest(plainOutput)
		g.Expect(err).To(HaveOccurred())

		// Simulate fallback: use content as-is, generate trigger description
		// But since content lacks 4 sections, this would fail validation.
		// The pipeline would use the content from generateSkillContent which
		// falls back to the template when compiler is nil.
		theme := "error handling"
		cluster := []memory.ClusterEntry{
			{ID: 1, Content: "Always check error returns"},
			{ID: 2, Content: "Handle nil pointer dereferences"},
			{ID: 3, Content: "Log errors with context"},
		}
		body, err := memory.GenerateSkillContentForTest(theme, cluster, nil)
		g.Expect(err).ToNot(HaveOccurred())

		desc := memory.GenerateTriggerDescriptionForTest(theme, body)
		skill := &memory.GeneratedSkill{
			Slug:        "error-handling",
			Theme:       theme,
			Description: desc,
			Content:     body,
		}

		result := memory.ValidateSkillComplianceForTest(skill)
		g.Expect(result.Issues).To(BeEmpty(), "template fallback should produce compliant skill")
	})

	t.Run("non-compliant skill is correctly blocked", func(t *testing.T) {
		g := NewWithT(t)

		skill := &memory.GeneratedSkill{
			Slug:        "bad-skill",
			Theme:       "bad",
			Description: "This description does not start with Use when.",
			Content:     "# Bad\n\nNo required sections here.",
		}

		result := memory.ValidateSkillComplianceForTest(skill)
		g.Expect(result.Issues).ToNot(BeEmpty(), "non-compliant skill should have issues")
		g.Expect(result.DescriptionOK).To(BeFalse())
		g.Expect(result.BodyStructureOK).To(BeFalse())
	})
}

func TestSkillTestingIntegration(t *testing.T) {
	t.Run("skill mutation blocked when tests fail", func(t *testing.T) {
		g := NewWithT(t)

		// Create mock test harness that returns failing tests
		mockHarness := &mockSkillTester{
			shouldPass: false,
			reasoning:  "RED: 2/3 failures, GREEN: 1/3 successes → FAIL",
		}

		opts := memory.OptimizeOpts{
			TestSkills:  true,
			TestRuns:    3,
			SkillTester: mockHarness,
		}

		// Attempt to compile a skill - should be blocked
		err := memory.TestAndCompileSkill(opts, testSkillCandidate())

		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("skill test failed"))
		g.Expect(mockHarness.called).To(BeTrue())
	})

	t.Run("skill mutation proceeds when tests pass", func(t *testing.T) {
		g := NewWithT(t)

		// Create mock test harness that returns passing tests
		mockHarness := &mockSkillTester{
			shouldPass: true,
			reasoning:  "RED: 3/3 failures, GREEN: 3/3 successes → PASS",
		}

		opts := memory.OptimizeOpts{
			TestSkills:  true,
			TestRuns:    3,
			SkillTester: mockHarness,
		}

		// Attempt to compile a skill - should succeed
		err := memory.TestAndCompileSkill(opts, testSkillCandidate())

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(mockHarness.called).To(BeTrue())
	})

	t.Run("skill testing skipped when TestSkills=false", func(t *testing.T) {
		g := NewWithT(t)

		// Create mock test harness
		mockHarness := &mockSkillTester{
			shouldPass: false,
		}

		opts := memory.OptimizeOpts{
			TestSkills:  false,
			SkillTester: mockHarness,
		}

		// Attempt to compile a skill - should skip testing
		err := memory.TestAndCompileSkill(opts, testSkillCandidate())

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(mockHarness.called).To(BeFalse())
	})
}

// ============================================================================
// T015: ValidateSkillCompliance() — V1-V8 checks
// ============================================================================

func TestValidateSkillCompliance(t *testing.T) {
	compliantSkill := func() *memory.GeneratedSkill {
		return &memory.GeneratedSkill{
			Slug:        "test-skill",
			Theme:       "testing",
			Description: "Use when the user needs testing guidance.",
			Content:     "## Overview\nTest.\n## When to Use\nTest.\n## Quick Reference\n1. Test.\n## Common Mistakes\n- Test.",
		}
	}

	t.Run("V1: description must start with 'Use when'", func(t *testing.T) {
		g := NewWithT(t)

		// Pass
		result := memory.ValidateSkillComplianceForTest(compliantSkill())
		g.Expect(result.DescriptionOK).To(BeTrue())

		// Fail
		skill := compliantSkill()
		skill.Description = "This skill helps with testing."
		result = memory.ValidateSkillComplianceForTest(skill)
		g.Expect(result.DescriptionOK).To(BeFalse())
		g.Expect(result.Issues).To(ContainElement(ContainSubstring("Use when")))
	})

	t.Run("V2: description must be <= 1024 chars", func(t *testing.T) {
		g := NewWithT(t)

		// Fail
		skill := compliantSkill()
		skill.Description = "Use when " + strings.Repeat("x", 1020)
		result := memory.ValidateSkillComplianceForTest(skill)
		g.Expect(result.DescriptionOK).To(BeFalse())
		g.Expect(result.Issues).To(ContainElement(ContainSubstring("1024")))
	})

	t.Run("V4: description must be third person", func(t *testing.T) {
		g := NewWithT(t)

		// Fail - "you"
		skill := compliantSkill()
		skill.Description = "Use when you need testing guidance."
		result := memory.ValidateSkillComplianceForTest(skill)
		g.Expect(result.DescriptionOK).To(BeFalse())
		g.Expect(result.Issues).To(ContainElement(ContainSubstring("third person")))
	})

	t.Run("V5: body must contain required sections", func(t *testing.T) {
		g := NewWithT(t)

		// Pass
		result := memory.ValidateSkillComplianceForTest(compliantSkill())
		g.Expect(result.BodyStructureOK).To(BeTrue())

		// Fail — missing sections
		skill := compliantSkill()
		skill.Content = "# Just a heading\n\nSome content."
		result = memory.ValidateSkillComplianceForTest(skill)
		g.Expect(result.BodyStructureOK).To(BeFalse())
	})

	t.Run("V6: body must be <= 500 lines", func(t *testing.T) {
		g := NewWithT(t)

		// Fail
		skill := compliantSkill()
		skill.Content = strings.Repeat("line\n", 501)
		result := memory.ValidateSkillComplianceForTest(skill)
		g.Expect(result.BodyLengthOK).To(BeFalse())
		g.Expect(result.Issues).To(ContainElement(ContainSubstring("500")))
	})

	t.Run("V7: name must start with memory.", func(t *testing.T) {
		g := NewWithT(t)

		// Pass (slug-based check: skill with memory. prefix would pass)
		result := memory.ValidateSkillComplianceForTest(compliantSkill())
		g.Expect(result.NamingOK).To(BeTrue())
	})

	t.Run("all checks pass for compliant skill", func(t *testing.T) {
		g := NewWithT(t)
		result := memory.ValidateSkillComplianceForTest(compliantSkill())
		g.Expect(result.DescriptionOK).To(BeTrue())
		g.Expect(result.BodyStructureOK).To(BeTrue())
		g.Expect(result.BodyLengthOK).To(BeTrue())
		g.Expect(result.NamingOK).To(BeTrue())
		g.Expect(result.Issues).To(BeEmpty())
	})
}

// mockSkillTester implements SkillTester for testing
type mockSkillTester struct {
	shouldPass bool
	reasoning  string
	called     bool
}

func (m *mockSkillTester) TestAndEvaluate(scenario memory.TestScenario, runs int) (bool, string, error) {
	m.called = true
	return m.shouldPass, m.reasoning, nil
}

func testSkillCandidate() memory.SkillCandidate {
	return memory.SkillCandidate{
		Theme:   "test-skill",
		Content: "Test skill content",
	}
}
