package memory_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/memory"
)

func TestExtractSkillDescription(t *testing.T) {
	t.Run("extracts first non-empty non-header line for simple content", func(t *testing.T) {
		g := NewWithT(t)
		content := `# Header

This is the description line.
More content here.`
		result := memory.ExtractSkillDescription(content, 100)
		// New behavior: collects multiple lines up to maxLen
		g.Expect(result).To(ContainSubstring("This is the description line."))
		g.Expect(result).To(ContainSubstring("More content here."))
	})

	t.Run("extracts multiple lines up to maxLen when no structured markers", func(t *testing.T) {
		g := NewWithT(t)
		content := `# Header

First line of description.
Second line of description.
Third line of description.`
		result := memory.ExtractSkillDescription(content, 100)
		// Should collect multiple lines up to maxLen
		g.Expect(result).To(ContainSubstring("First line"))
		g.Expect(result).To(ContainSubstring("Second line"))
		g.Expect(len(result)).To(BeNumerically("<=", 100))
	})

	t.Run("extracts structured section when Core marker present", func(t *testing.T) {
		g := NewWithT(t)
		content := `---
description: |
  Core: Produces test files for acceptance criteria.
  Triggers: write tests, tdd red, failing tests.
  Domains: testing, tdd.
---

# Full Body

More content here.`
		result := memory.ExtractSkillDescription(content, 1500)
		g.Expect(result).To(ContainSubstring("Core: Produces test files"))
		g.Expect(result).To(ContainSubstring("Triggers:"))
		g.Expect(result).To(ContainSubstring("Domains:"))
	})

	t.Run("extracts all structured lines from first to last marker", func(t *testing.T) {
		g := NewWithT(t)
		content := `Some preamble
Core: Does something useful.
Triggers: action1, action2.
Domains: domain1, domain2.
Anti-patterns: NOT for X.
Related: other-skill.
More content after.`
		result := memory.ExtractSkillDescription(content, 1500)
		g.Expect(result).To(ContainSubstring("Core:"))
		g.Expect(result).To(ContainSubstring("Triggers:"))
		g.Expect(result).To(ContainSubstring("Domains:"))
		g.Expect(result).To(ContainSubstring("Anti-patterns:"))
		g.Expect(result).To(ContainSubstring("Related:"))
		g.Expect(result).ToNot(ContainSubstring("Some preamble"))
		g.Expect(result).ToNot(ContainSubstring("More content after"))
	})

	t.Run("truncates at maxLen if structured section is too long", func(t *testing.T) {
		g := NewWithT(t)
		longTriggers := "trigger1, trigger2, trigger3, trigger4, trigger5, trigger6, trigger7, trigger8, trigger9, trigger10"
		content := `Core: Test.
Triggers: ` + longTriggers + `.
Domains: testing.`
		result := memory.ExtractSkillDescription(content, 50)
		g.Expect(len(result)).To(BeNumerically("<=", 50))
		g.Expect(result).To(ContainSubstring("Core:"))
	})

	t.Run("handles content with only whitespace", func(t *testing.T) {
		g := NewWithT(t)
		content := "\n\n   \n\t\n"
		result := memory.ExtractSkillDescription(content, 100)
		g.Expect(result).To(Equal(""))
	})

	t.Run("handles content shorter than maxLen", func(t *testing.T) {
		g := NewWithT(t)
		content := "Short description."
		result := memory.ExtractSkillDescription(content, 1000)
		g.Expect(result).To(Equal("Short description."))
	})
}
