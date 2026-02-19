//go:build sqlite_fts5

package memory_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// T002: writeSkillFile() — memory- prefix and memory. frontmatter name
// ============================================================================

func TestWriteSkillFileMemoryPrefix(t *testing.T) {
	t.Run("creates memory-{slug} directory", func(t *testing.T) {
		g := NewWithT(t)

		skillsDir := t.TempDir()
		skill := &memory.GeneratedSkill{
			Slug:        "test-skill",
			Theme:       "test skill",
			Description: "Use when testing skill generation.",
			Content:     "# Test\n\nContent here.",
			Alpha:       1.0,
			Beta:        1.0,
		}

		err := memory.WriteSkillFileForTest(skillsDir, skill)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify directory uses memory- prefix
		expectedDir := filepath.Join(skillsDir, "memory-test-skill")
		info, err := os.Stat(expectedDir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(info.IsDir()).To(BeTrue())

		// Verify old mem- directory was NOT created
		oldDir := filepath.Join(skillsDir, "mem-test-skill")
		_, err = os.Stat(oldDir)
		g.Expect(os.IsNotExist(err)).To(BeTrue())
	})

	t.Run("writes name: memory.{slug} in frontmatter", func(t *testing.T) {
		g := NewWithT(t)

		skillsDir := t.TempDir()
		skill := &memory.GeneratedSkill{
			Slug:        "my-skill",
			Theme:       "my skill",
			Description: "Use when doing things.",
			Content:     "# My Skill\n\nContent.",
			Alpha:       1.0,
			Beta:        1.0,
		}

		err := memory.WriteSkillFileForTest(skillsDir, skill)
		g.Expect(err).ToNot(HaveOccurred())

		// Read the written file
		content, err := os.ReadFile(filepath.Join(skillsDir, "memory-my-skill", "SKILL.md"))
		g.Expect(err).ToNot(HaveOccurred())

		// Verify name uses memory. prefix (dot separator)
		g.Expect(string(content)).To(ContainSubstring("name: memory.my-skill"))
		// Verify old mem: prefix is NOT present
		g.Expect(string(content)).ToNot(ContainSubstring("name: mem:"))
		// Verify generated flag still present
		g.Expect(string(content)).To(ContainSubstring("generated: true"))
	})
}

// ============================================================================
// T006: generateTriggerDescription() — "Use when..." trigger descriptions
// ============================================================================

func TestGenerateTriggerDescription(t *testing.T) {
	t.Run("starts with Use when", func(t *testing.T) {
		g := NewWithT(t)
		desc := memory.GenerateTriggerDescriptionForTest("error handling", "Some body content about error handling.")
		g.Expect(desc).To(HavePrefix("Use when"))
	})

	t.Run("is at most 1024 chars", func(t *testing.T) {
		g := NewWithT(t)
		longTheme := strings.Repeat("very-long-theme-name ", 100)
		desc := memory.GenerateTriggerDescriptionForTest(longTheme, "Content body.")
		g.Expect(len(desc)).To(BeNumerically("<=", 1024))
	})

	t.Run("uses third person (no I/you/we as subjects)", func(t *testing.T) {
		g := NewWithT(t)
		desc := memory.GenerateTriggerDescriptionForTest("testing patterns", "Content about testing.")
		// Should not start any sentence with prohibited pronouns
		g.Expect(desc).ToNot(MatchRegexp(`(?i)\bI\b`))
		g.Expect(desc).ToNot(MatchRegexp(`(?i)\byou\b`))
		g.Expect(desc).ToNot(MatchRegexp(`(?i)\bwe\b`))
		g.Expect(desc).ToNot(MatchRegexp(`(?i)\bmy\b`))
	})

	t.Run("is not a substring of content body", func(t *testing.T) {
		g := NewWithT(t)
		content := "This is the full body content about error handling patterns and practices."
		desc := memory.GenerateTriggerDescriptionForTest("error handling", content)
		g.Expect(content).ToNot(ContainSubstring(desc))
	})

	t.Run("handles short theme gracefully", func(t *testing.T) {
		g := NewWithT(t)
		desc := memory.GenerateTriggerDescriptionForTest("go", "Some content about Go.")
		g.Expect(desc).To(HavePrefix("Use when"))
		g.Expect(len(desc)).To(BeNumerically(">", 10))
	})
}

// ============================================================================
// T007: writeSkillFile() description cap and YAML safety
// ============================================================================

func TestWriteSkillFileDescriptionCap(t *testing.T) {
	t.Run("truncates description exceeding 1024 chars to 1024 on disk", func(t *testing.T) {
		g := NewWithT(t)

		skillsDir := t.TempDir()
		longDesc := "Use when " + strings.Repeat("x", 2000)
		skill := &memory.GeneratedSkill{
			Slug:        "long-desc",
			Theme:       "long desc",
			Description: longDesc,
			Content:     "# Content\n\nBody.",
			Alpha:       1.0,
			Beta:        1.0,
		}

		err := memory.WriteSkillFileForTest(skillsDir, skill)
		g.Expect(err).ToNot(HaveOccurred())

		content, err := os.ReadFile(filepath.Join(skillsDir, "memory-long-desc", "SKILL.md"))
		g.Expect(err).ToNot(HaveOccurred())

		// Find the description line in frontmatter
		lines := strings.Split(string(content), "\n")
		var descLine string
		for _, line := range lines {
			if strings.HasPrefix(line, "description: ") {
				descLine = strings.TrimPrefix(line, "description: ")
				break
			}
		}
		g.Expect(len(descLine)).To(BeNumerically("<=", 1024))
	})

	t.Run("YAML-quotes description with colons", func(t *testing.T) {
		g := NewWithT(t)

		skillsDir := t.TempDir()
		skill := &memory.GeneratedSkill{
			Slug:        "yaml-test",
			Theme:       "yaml test",
			Description: "Use when handling key: value pairs in YAML configs.",
			Content:     "# Content\n\nBody.",
			Alpha:       1.0,
			Beta:        1.0,
		}

		err := memory.WriteSkillFileForTest(skillsDir, skill)
		g.Expect(err).ToNot(HaveOccurred())

		content, err := os.ReadFile(filepath.Join(skillsDir, "memory-yaml-test", "SKILL.md"))
		g.Expect(err).ToNot(HaveOccurred())

		// Description with colon should be YAML-quoted
		g.Expect(string(content)).To(ContainSubstring(`description: "Use when handling key: value pairs`))
	})
}
