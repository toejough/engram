//go:build integration

package skills_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

// TestOldQASkillsDeleted verifies all 13 phase-specific QA skill directories are deleted
func TestOldQASkillsDeleted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skillsDir := filepath.Join("..", "..", "skills")
	oldQASkills := []string{
		"pm-qa",
		"design-qa",
		"arch-qa",
		"breakdown-qa",
		"tdd-qa",
		"tdd-red-qa",
		"tdd-green-qa",
		"tdd-refactor-qa",
		"doc-qa",
		"context-qa",
		"alignment-qa",
		"retro-qa",
		"summary-qa",
	}

	for _, skillName := range oldQASkills {
		skillPath := filepath.Join(skillsDir, skillName)
		_, err := os.Stat(skillPath)
		g.Expect(os.IsNotExist(err)).To(BeTrue(),
			"Old QA skill directory %s should not exist", skillName)
	}
}

// TestUniversalQASkillExists verifies the universal QA skill exists and is functional
func TestUniversalQASkillExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skillPath := filepath.Join("..", "..", "skills", "qa", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).NotTo(HaveOccurred(), "Universal QA skill should exist at skills/qa/SKILL.md")

	contentStr := string(content)
	g.Expect(contentStr).To(ContainSubstring("name: qa"), "Should have correct skill name")
	g.Expect(contentStr).To(ContainSubstring("Universal QA"), "Should be documented as universal")
	g.Expect(contentStr).To(ContainSubstring("## Contract"), "Should reference contract standard")
}

// TestNoOldQAReferencesInSkills verifies no broken references to old QA skills in skill files
func TestNoOldQAReferencesInSkills(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skillsDir := filepath.Join("..", "..", "skills")

	// Old QA skill names that should not be referenced
	oldQANames := []string{
		"pm-qa",
		"design-qa",
		"arch-qa",
		"breakdown-qa",
		"tdd-qa",
		"tdd-red-qa",
		"tdd-green-qa",
		"tdd-refactor-qa",
		"doc-qa",
		"context-qa",
		"alignment-qa",
		"retro-qa",
		"summary-qa",
	}

	// Walk through all SKILL.md files
	err := filepath.Walk(skillsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "SKILL.md" {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			contentStr := string(content)

			// Check for old QA references
			for _, oldQA := range oldQANames {
				// Look for the old QA name as a standalone reference (not in historical context)
				if strings.Contains(contentStr, oldQA) {
					// Check if it's in a context that suggests current usage
					lines := strings.Split(contentStr, "\n")
					for i, line := range lines {
						if strings.Contains(line, oldQA) {
							// Skip if it's clearly historical/deprecated context
							if strings.Contains(line, "DEPRECATED") ||
								strings.Contains(line, "was:") ||
								strings.Contains(line, "previously") ||
								(i > 0 && strings.Contains(lines[i-1], "DEPRECATED")) {
								continue
							}

							// This is a current reference that should use universal QA
							t.Errorf("File %s contains reference to old QA skill '%s' at line %d: %s",
								path, oldQA, i+1, strings.TrimSpace(line))
						}
					}
				}
			}
		}
		return nil
	})
	g.Expect(err).NotTo(HaveOccurred())
}

// TestQATemplateHandled verifies QA-TEMPLATE.md is properly updated or removed
func TestQATemplateHandled(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	templatePath := filepath.Join("..", "..", "skills", "shared", "QA-TEMPLATE.md")
	_, err := os.Stat(templatePath)

	if err == nil {
		// If it exists, it should reference the universal QA approach
		content, err := os.ReadFile(templatePath)
		g.Expect(err).NotTo(HaveOccurred())
		contentStr := string(content)
		g.Expect(contentStr).To(Or(
			ContainSubstring("universal"),
			ContainSubstring("deprecated"),
		), "QA-TEMPLATE.md should reference universal QA or be marked deprecated")
	} else {
		// If it doesn't exist, that's also acceptable (deleted)
		g.Expect(os.IsNotExist(err)).To(BeTrue(),
			"QA-TEMPLATE.md should either exist with updates or be deleted")
	}
}

// TestAllProducerContractsExist verifies all producers have Contract sections (prerequisite)
func TestAllProducerContractsExist(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	producersWithContracts := []string{
		"pm-interview-producer",
		"pm-infer-producer",
		"design-interview-producer",
		"design-infer-producer",
		"arch-interview-producer",
		"arch-infer-producer",
		"breakdown-producer",
		"tdd-red-producer",
		"tdd-red-infer-producer",
		"tdd-green-producer",
		"tdd-refactor-producer",
		"doc-producer",
		"alignment-producer",
		"evaluation-producer",
		"plan-producer",
	}

	for _, producer := range producersWithContracts {
		skillPath := filepath.Join("..", "..", "skills", producer, "SKILL.md")
		content, err := os.ReadFile(skillPath)
		g.Expect(err).NotTo(HaveOccurred(), "Producer %s SKILL.md should exist", producer)

		contentStr := string(content)
		g.Expect(contentStr).To(ContainSubstring("## Contract"),
			"Producer %s should have Contract section", producer)
	}
}

// TestGapAnalysisComplete verifies gap analysis was completed (prerequisite)
func TestGapAnalysisComplete(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	gapAnalysisPath := filepath.Join("..", "..", "docs", "gap-analysis.md")
	content, err := os.ReadFile(gapAnalysisPath)
	g.Expect(err).NotTo(HaveOccurred(), "Gap analysis document should exist")

	contentStr := string(content)

	// Verify all 13 QA skills were analyzed
	expectedAnalyses := []string{
		"pm-qa",
		"design-qa",
		"arch-qa",
		"breakdown-qa",
		"tdd-qa",
		"tdd-red-qa",
		"tdd-green-qa",
		"tdd-refactor-qa",
		"doc-qa",
		"context-qa",
		"alignment-qa",
		"retro-qa",
		"summary-qa",
	}

	for _, qa := range expectedAnalyses {
		g.Expect(contentStr).To(ContainSubstring(qa),
			"Gap analysis should include %s", qa)
	}
}

// Property test: any skill file that mentions "qa" should use the universal "qa" skill
func TestPropertyQAReferencesUseUniversalSkill(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)
		skillsDir := filepath.Join("..", "..", "skills")

		// Get all SKILL.md files
		var skillFiles []string
		err := filepath.Walk(skillsDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && info.Name() == "SKILL.md" {
				skillFiles = append(skillFiles, path)
			}
			return nil
		})
		g.Expect(err).NotTo(HaveOccurred())

		if len(skillFiles) == 0 {
			t.Skip("No skill files found")
		}

		// Pick a random skill file
		skillFile := rapid.SampledFrom(skillFiles).Draw(t, "skillFile")
		content, err := os.ReadFile(skillFile)
		g.Expect(err).NotTo(HaveOccurred())

		contentStr := string(content)

		// If the file mentions QA in a dispatch/invocation context, it should use "qa" not "<phase>-qa"
		if strings.Contains(contentStr, "qa = ") ||
			strings.Contains(contentStr, "QA:") ||
			strings.Contains(contentStr, "spawn-qa") {

			// Verify it uses generic "qa" references, not phase-specific
			oldQAPattern := []string{
				`"pm-qa"`, `"design-qa"`, `"arch-qa"`, `"breakdown-qa"`,
				`"tdd-qa"`, `"tdd-red-qa"`, `"tdd-green-qa"`, `"tdd-refactor-qa"`,
				`"doc-qa"`, `"context-qa"`, `"alignment-qa"`, `"retro-qa"`, `"summary-qa"`,
			}

			for _, pattern := range oldQAPattern {
				if strings.Contains(contentStr, pattern) {
					// Check if it's in a deprecated/historical context
					if !strings.Contains(contentStr, "DEPRECATED") &&
						!strings.Contains(contentStr, "was:") {
						t.Fatalf("File %s uses old QA reference: %s", skillFile, pattern)
					}
				}
			}
		}
	})
}
