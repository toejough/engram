//go:build sqlite_fts5

package memory_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestApplyClaudeMDProposal_ConsolidateInsertBeforeNextSection verifies appendToClaudeMDWithFS
// inserts learnings before the next section when the file has multiple sections.
func TestApplyClaudeMDProposal_ConsolidateInsertBeforeNextSection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")

	// Write a CLAUDE.md with Promoted Learnings followed by another section
	content := "## Promoted Learnings\n\n- entry one\n- entry two\n\n## Another Section\n\n- item\n"
	err := os.WriteFile(claudeMDPath, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	proposal := memory.MaintenanceProposal{
		Action:  "consolidate",
		Tier:    "claude-md",
		Target:  "entry one|entry two",
		Preview: "combined learning",
	}

	err = memory.ApplyClaudeMDProposal(memory.RealFS{}, claudeMDPath, proposal)

	g.Expect(err).ToNot(HaveOccurred())

	result, err := os.ReadFile(claudeMDPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(result)).To(ContainSubstring("combined learning"))
	// Another Section should still be present after insertion
	g.Expect(string(result)).To(ContainSubstring("## Another Section"))
}

// TestApplyClaudeMDProposal_RewriteNoNewlineBeforeSection verifies appendToClaudeMDWithFS
// adds "\n\n" before the Promoted Learnings section when content doesn't end with it.
func TestApplyClaudeMDProposal_RewriteContentNoTrailingNewlines(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")

	// Write content that ends with a single newline (not "\n\n")
	err := os.WriteFile(claudeMDPath, []byte("# My Config\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	proposal := memory.MaintenanceProposal{
		Action:  "rewrite",
		Tier:    "claude-md",
		Target:  "nonexistent entry",
		Preview: "prefer explicit over implicit",
	}

	err = memory.ApplyClaudeMDProposal(memory.RealFS{}, claudeMDPath, proposal)

	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(claudeMDPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("## Promoted Learnings"))
}

// TestApplyClaudeMDProposal_RewriteNoPromotedSection verifies appendToClaudeMDWithFS
// handles a CLAUDE.md that has no "## Promoted Learnings" section (idx == -1 branch).
func TestApplyClaudeMDProposal_RewriteNoPromotedSection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")

	// Write a CLAUDE.md without any "## Promoted Learnings" section
	err := os.WriteFile(claudeMDPath, []byte("# My Config\n\n- some global setting\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	proposal := memory.MaintenanceProposal{
		Action:  "rewrite",
		Tier:    "claude-md",
		Target:  "nonexistent entry",
		Preview: "always use TDD",
	}

	err = memory.ApplyClaudeMDProposal(memory.RealFS{}, claudeMDPath, proposal)

	g.Expect(err).ToNot(HaveOccurred())

	// Verify the Promoted Learnings section was added
	content, err := os.ReadFile(claudeMDPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("## Promoted Learnings"))
	g.Expect(string(content)).To(ContainSubstring("always use TDD"))
}
