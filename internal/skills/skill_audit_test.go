package skills_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"
)

// WorkflowsConfig mirrors the structure of workflows.toml
type WorkflowsConfig struct {
	States map[string]StateConfig `toml:"states"`
}

// StateConfig represents a state configuration in workflows.toml
type StateConfig struct {
	Type         string `toml:"type"`
	Skill        string `toml:"skill"`
	SkillPath    string `toml:"skill_path"`
	DefaultModel string `toml:"default_model"`
}

// TestSkillAudit_AllSkillsHaveSKILLmd verifies every skill in workflows.toml has a SKILL.md
// Traces to: ISSUE-163
func TestSkillAudit_AllSkillsHaveSKILLmd(t *testing.T) {
	g := NewWithT(t)

	// Parse workflows.toml
	workflowPath := filepath.Join("..", "..", "internal", "workflow", "workflows.toml")
	var config WorkflowsConfig
	_, err := toml.DecodeFile(workflowPath, &config)
	g.Expect(err).ToNot(HaveOccurred(), "should parse workflows.toml")

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	// Collect all unique skills from workflows.toml
	skillSet := make(map[string]bool)
	for _, state := range config.States {
		if state.Skill != "" {
			skillSet[state.Skill] = true
		}
	}

	g.Expect(len(skillSet)).To(BeNumerically(">", 0), "workflows.toml should reference at least one skill")

	// Verify each skill has a SKILL.md
	for skillName := range skillSet {
		skillPath := filepath.Join(homeDir, ".claude", "skills", skillName, "SKILL.md")
		_, err := os.ReadFile(skillPath)
		g.Expect(err).ToNot(HaveOccurred(), "skill %s should have SKILL.md at %s", skillName, skillPath)
	}
}

// TestSkillAudit_AllSkillsHaveWorkflowContextSection verifies each SKILL.md has "## Workflow Context"
// Traces to: ISSUE-163
func TestSkillAudit_AllSkillsHaveWorkflowContextSection(t *testing.T) {
	g := NewWithT(t)

	// Parse workflows.toml
	workflowPath := filepath.Join("..", "..", "internal", "workflow", "workflows.toml")
	var config WorkflowsConfig
	_, err := toml.DecodeFile(workflowPath, &config)
	g.Expect(err).ToNot(HaveOccurred(), "should parse workflows.toml")

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	// Collect all unique skills from workflows.toml
	skillSet := make(map[string]bool)
	for _, state := range config.States {
		if state.Skill != "" {
			skillSet[state.Skill] = true
		}
	}

	// Verify each skill's SKILL.md has "## Workflow Context" section
	for skillName := range skillSet {
		skillPath := filepath.Join(homeDir, ".claude", "skills", skillName, "SKILL.md")
		content, err := os.ReadFile(skillPath)
		g.Expect(err).ToNot(HaveOccurred(), "should read SKILL.md for %s", skillName)

		text := string(content)
		g.Expect(text).To(ContainSubstring("## Workflow Context"),
			"skill %s should have '## Workflow Context' section", skillName)
	}
}

// TestSkillAudit_ModelDeclarationsMatch verifies model declarations in SKILL.md match workflows.toml
// Traces to: ISSUE-163
func TestSkillAudit_ModelDeclarationsMatch(t *testing.T) {
	g := NewWithT(t)

	// Parse workflows.toml
	workflowPath := filepath.Join("..", "..", "internal", "workflow", "workflows.toml")
	var config WorkflowsConfig
	_, err := toml.DecodeFile(workflowPath, &config)
	g.Expect(err).ToNot(HaveOccurred(), "should parse workflows.toml")

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	// Build map: skill name -> expected default_model
	skillModels := make(map[string]string)
	for _, state := range config.States {
		if state.Skill != "" && state.DefaultModel != "" {
			// If we've seen this skill before, verify consistency
			if existing, ok := skillModels[state.Skill]; ok {
				if existing != state.DefaultModel {
					t.Logf("WARNING: skill %s has inconsistent default_model in workflows.toml: %s vs %s",
						state.Skill, existing, state.DefaultModel)
				}
			}
			skillModels[state.Skill] = state.DefaultModel
		}
	}

	// For each skill with a default_model, check SKILL.md frontmatter
	frontmatterPattern := regexp.MustCompile(`(?m)^---\n(.*?)\n---`)
	modelPattern := regexp.MustCompile(`(?m)^model:\s*(\w+)`)

	for skillName, expectedModel := range skillModels {
		skillPath := filepath.Join(homeDir, ".claude", "skills", skillName, "SKILL.md")
		content, err := os.ReadFile(skillPath)
		g.Expect(err).ToNot(HaveOccurred(), "should read SKILL.md for %s", skillName)

		text := string(content)

		// Extract frontmatter
		matches := frontmatterPattern.FindStringSubmatch(text)
		if len(matches) < 2 {
			t.Logf("WARNING: skill %s SKILL.md has no frontmatter", skillName)
			continue
		}

		frontmatter := matches[1]

		// Extract model declaration
		modelMatches := modelPattern.FindStringSubmatch(frontmatter)
		if len(modelMatches) < 2 {
			t.Logf("WARNING: skill %s SKILL.md has no model: declaration in frontmatter", skillName)
			continue
		}

		actualModel := modelMatches[1]

		// Verify model matches (or at least doesn't conflict)
		g.Expect(actualModel).To(Equal(expectedModel),
			"skill %s model declaration should match workflows.toml: expected %s, got %s",
			skillName, expectedModel, actualModel)
	}
}

// TestSkillAudit_DormantSkillsDocumented verifies skills with SKILL.md but not in workflows.toml are marked DORMANT
// Traces to: ISSUE-163
func TestSkillAudit_DormantSkillsDocumented(t *testing.T) {
	g := NewWithT(t)

	// Parse workflows.toml
	workflowPath := filepath.Join("..", "..", "internal", "workflow", "workflows.toml")
	var config WorkflowsConfig
	_, err := toml.DecodeFile(workflowPath, &config)
	g.Expect(err).ToNot(HaveOccurred(), "should parse workflows.toml")

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	// Collect all skills referenced in workflows.toml
	activeSkills := make(map[string]bool)
	for _, state := range config.States {
		if state.Skill != "" {
			activeSkills[state.Skill] = true
		}
	}

	// Find all SKILL.md files
	skillsDir := filepath.Join(homeDir, ".claude", "skills")
	entries, err := os.ReadDir(skillsDir)
	g.Expect(err).ToNot(HaveOccurred(), "should read ~/.claude/skills directory")

	// Check each skill directory
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillName := entry.Name()
		skillPath := filepath.Join(skillsDir, skillName, "SKILL.md")

		// Check if SKILL.md exists
		_, err := os.Stat(skillPath)
		if os.IsNotExist(err) {
			continue
		}

		// If skill is not in workflows.toml, it should have DORMANT comment
		if !activeSkills[skillName] {
			content, err := os.ReadFile(skillPath)
			g.Expect(err).ToNot(HaveOccurred(), "should read SKILL.md for %s", skillName)

			text := string(content)

			// Look for DORMANT comment near the top of the file
			lines := strings.Split(text, "\n")
			foundDormant := false
			for i := 0; i < 20 && i < len(lines); i++ {
				if strings.Contains(lines[i], "DORMANT") {
					foundDormant = true
					break
				}
			}

			g.Expect(foundDormant).To(BeTrue(),
				"skill %s has SKILL.md but is not referenced in workflows.toml - should have DORMANT comment",
				skillName)
		}
	}
}

// TestSkillAudit_WorkflowContextHasRequiredFields verifies Workflow Context sections have required fields
// Traces to: ISSUE-163
func TestSkillAudit_WorkflowContextHasRequiredFields(t *testing.T) {
	g := NewWithT(t)

	// Parse workflows.toml
	workflowPath := filepath.Join("..", "..", "internal", "workflow", "workflows.toml")
	var config WorkflowsConfig
	_, err := toml.DecodeFile(workflowPath, &config)
	g.Expect(err).ToNot(HaveOccurred(), "should parse workflows.toml")

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	// Collect all unique skills from workflows.toml
	skillSet := make(map[string]bool)
	for _, state := range config.States {
		if state.Skill != "" {
			skillSet[state.Skill] = true
		}
	}

	// For each skill, verify Workflow Context section has required fields
	for skillName := range skillSet {
		skillPath := filepath.Join(homeDir, ".claude", "skills", skillName, "SKILL.md")
		content, err := os.ReadFile(skillPath)
		g.Expect(err).ToNot(HaveOccurred(), "should read SKILL.md for %s", skillName)

		text := string(content)

		// Extract Workflow Context section
		workflowContextPattern := regexp.MustCompile(`(?s)## Workflow Context\n(.*?)(?:\n##|\z)`)
		matches := workflowContextPattern.FindStringSubmatch(text)
		if len(matches) < 2 {
			t.Fatalf("skill %s has no Workflow Context section", skillName)
		}

		contextSection := matches[1]

		// Required fields: Phase, Upstream, Downstream, Model
		requiredFields := []string{"Phase", "Upstream", "Downstream", "Model"}
		for _, field := range requiredFields {
			// Look for field as bullet point or bold header
			fieldPattern := regexp.MustCompile(fmt.Sprintf(`(?i)(\*\*%s\*\*|-%s|•%s)`, field, field, field))
			g.Expect(fieldPattern.MatchString(contextSection)).To(BeTrue(),
				"skill %s Workflow Context should have '%s' field", skillName, field)
		}
	}
}
