//go:build integration

package skills_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

// TEST-162-003 traces: ISSUE-162
// Test that arch-interview-producer SKILL.md mentions plan awareness
func TestArchInterviewProducer_MentionsPlanAwareness(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Read arch-interview-producer SKILL.md
	skillPath := filepath.Join("..", "..", "skills", "arch-interview-producer", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	skillContent := string(content)

	// Verify it contains plan awareness keywords
	g.Expect(skillContent).To(ContainSubstring("plan.md"),
		"arch-interview-producer should mention plan.md")
}

// TEST-162-002 traces: ISSUE-162
// Test that design-interview-producer SKILL.md mentions plan awareness
func TestDesignInterviewProducer_MentionsPlanAwareness(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Read design-interview-producer SKILL.md
	skillPath := filepath.Join("..", "..", "skills", "design-interview-producer", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	skillContent := string(content)

	// Verify it contains plan awareness keywords
	g.Expect(skillContent).To(ContainSubstring("plan.md"),
		"design-interview-producer should mention plan.md")
}

// TEST-162-001 traces: ISSUE-162
// Test that pm-interview-producer SKILL.md mentions plan awareness
func TestPMInterviewProducer_MentionsPlanAwareness(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Read pm-interview-producer SKILL.md
	skillPath := filepath.Join("..", "..", "skills", "pm-interview-producer", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	skillContent := string(content)

	// Verify it contains plan awareness keywords
	// Should mention checking for plan.md or approved plan
	g.Expect(skillContent).To(ContainSubstring("plan.md"),
		"pm-interview-producer should mention plan.md")
}
