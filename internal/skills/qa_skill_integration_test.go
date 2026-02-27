//go:build integration

package skills_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

// TEST-002-013 traces: TASK-2
// Test that full checklist output format is documented
func TestQASkill_ChecklistFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())
	skillPath := filepath.Join(homeDir, ".claude", "skills", "qa", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document output format per DES-003
	g.Expect(text).To(ContainSubstring("DES-003"), "should reference DES-003 output format")
	g.Expect(text).To(ContainSubstring("checklist"), "should mention checklist output")

	// Should show example with [x] and [ ] markers
	g.Expect(text).To(ContainSubstring("[x]"), "should show checked item format")
	g.Expect(text).To(ContainSubstring("[ ]"), "should show unchecked item format")
}

// TEST-002-006 traces: TASK-2
// Test that contract extraction algorithm is documented
func TestQASkill_ContractExtraction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())
	skillPath := filepath.Join(homeDir, ".claude", "skills", "qa", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document contract extraction per ARCH-021
	g.Expect(text).To(ContainSubstring("## Contract"), "should mention contract section search")
	g.Expect(text).To(ContainSubstring("ARCH-021"), "should reference ARCH-021 algorithm")
}

// TEST-002-009 traces: TASK-2
// Test that escalate-user on max iterations is documented
func TestQASkill_EscalateOnMaxIterations(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())
	skillPath := filepath.Join(homeDir, ".claude", "skills", "qa", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should escalate to user when max iterations reached
	// Find sections that discuss max iterations and escalation together
	lowerText := strings.ToLower(text)
	g.Expect(lowerText).To(ContainSubstring("escalate-user"), "should mention escalate-user outcome")

	// Check that max iterations leads to escalation
	hasMaxIterations := strings.Contains(lowerText, "max") && (strings.Contains(lowerText, "iteration") || strings.Contains(lowerText, "attempt"))
	g.Expect(hasMaxIterations).To(BeTrue(), "should document max iterations behavior")
}

// TEST-002-001 traces: TASK-2
// Test that qa/SKILL.md file exists at expected location
func TestQASkill_FileExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Skills are in user's home directory
	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "qa", "SKILL.md")
	_, err = os.Stat(skillPath)
	g.Expect(err).ToNot(HaveOccurred(), "qa/SKILL.md should exist")
}

// TEST-002-002 traces: TASK-2
// Test that qa/SKILL.md has correct frontmatter
func TestQASkill_Frontmatter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "qa", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	// Extract frontmatter between --- markers
	parts := strings.Split(string(content), "---")
	g.Expect(len(parts)).To(BeNumerically(">=", 3), "should have frontmatter")

	frontmatter := parts[1]

	// Parse as YAML
	var fm map[string]interface{}
	err = yaml.Unmarshal([]byte(frontmatter), &fm)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify required fields
	g.Expect(fm["name"]).To(Equal("qa"))
	g.Expect(fm["model"]).To(Equal("haiku"))
	g.Expect(fm["role"]).To(Equal("qa"))
}

// TEST-002-008 traces: TASK-2
// Test that iteration tracking is documented
func TestQASkill_IterationTracking(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())
	skillPath := filepath.Join(homeDir, ".claude", "skills", "qa", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document iteration tracking per ARCH-028
	g.Expect(text).To(ContainSubstring("iteration"), "should mention iteration tracking")
	g.Expect(text).To(MatchRegexp("max.*3|3.*iterations"), "should mention max 3 iterations")
}

// TEST-002-003 traces: TASK-2
// Test that LOAD phase is documented
func TestQASkill_LoadPhase(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())
	skillPath := filepath.Join(homeDir, ".claude", "skills", "qa", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document LOAD phase
	g.Expect(text).To(ContainSubstring("LOAD"), "should document LOAD phase")

	// Should mention key LOAD phase activities
	g.Expect(text).To(ContainSubstring("Extract contract"), "should mention contract extraction")
	g.Expect(text).To(ContainSubstring("Read artifacts"), "should mention reading artifacts")
}

// TEST-002-010 traces: TASK-2
// Test that malformed output handling is documented
func TestQASkill_MalformedOutputHandling(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())
	skillPath := filepath.Join(homeDir, ".claude", "skills", "qa", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document error handling per DES-006
	g.Expect(text).To(ContainSubstring("DES-006"), "should reference DES-006 malformed output handling")
	g.Expect(text).To(MatchRegexp("(?i)malformed|invalid"), "should mention malformed output handling")
}

// TEST-002-011 traces: TASK-2
// Test that missing artifacts handling is documented
func TestQASkill_MissingArtifactsHandling(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())
	skillPath := filepath.Join(homeDir, ".claude", "skills", "qa", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document error handling per DES-007
	g.Expect(text).To(ContainSubstring("DES-007"), "should reference DES-007 missing artifacts handling")
	g.Expect(text).To(ContainSubstring("missing artifact"), "should mention missing artifact handling")
}

// TEST-002-007 traces: TASK-2
// Test that prose fallback is documented
func TestQASkill_ProseFallback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())
	skillPath := filepath.Join(homeDir, ".claude", "skills", "qa", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document fallback behavior per ARCH-024
	g.Expect(text).To(ContainSubstring("fallback"), "should mention fallback behavior")
	g.Expect(text).To(ContainSubstring("ARCH-024"), "should reference ARCH-024 prose extraction")
}

// TEST-002-005 traces: TASK-2
// Test that RETURN phase is documented
func TestQASkill_ReturnPhase(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())
	skillPath := filepath.Join(homeDir, ".claude", "skills", "qa", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document RETURN phase
	g.Expect(text).To(ContainSubstring("RETURN"), "should document RETURN phase")

	// Should mention return outcomes per acceptance criteria
	g.Expect(text).To(ContainSubstring("approved"), "should mention approved outcome")
	g.Expect(text).To(ContainSubstring("improvement-request"), "should mention improvement-request outcome")
	g.Expect(text).To(ContainSubstring("escalate-phase"), "should mention escalate-phase outcome")
	g.Expect(text).To(ContainSubstring("escalate-user"), "should mention escalate-user outcome")
	g.Expect(text).To(ContainSubstring("error"), "should mention error outcome")
}

// TEST-002-012 traces: TASK-2
// Test that unreadable producer SKILL.md handling is documented
func TestQASkill_UnreadableSkillHandling(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())
	skillPath := filepath.Join(homeDir, ".claude", "skills", "qa", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document error handling per DES-009
	g.Expect(text).To(ContainSubstring("DES-009"), "should reference DES-009 unreadable SKILL.md handling")
	g.Expect(text).To(ContainSubstring("Unreadable"), "should mention unreadable SKILL.md handling")
}

// TEST-002-004 traces: TASK-2
// Test that VALIDATE phase is documented
func TestQASkill_ValidatePhase(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())
	skillPath := filepath.Join(homeDir, ".claude", "skills", "qa", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document VALIDATE phase
	g.Expect(text).To(ContainSubstring("VALIDATE"), "should document VALIDATE phase")

	// Should mention executing checks
	g.Expect(text).To(ContainSubstring("checks"), "should mention checking against contract")
}
