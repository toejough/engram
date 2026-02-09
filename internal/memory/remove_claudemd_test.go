package memory_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// Unit tests for removeFromClaudeMD
// traces: ISSUE-184
// ============================================================================

// TEST-1110: RemoveFromClaudeMD removes matching entry
func TestRemoveFromClaudeMDRemovesEntry(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	claudeMDPath := filepath.Join(tempDir, "CLAUDE.md")

	content := `# Working With Joe

## Promoted Learnings

- 2026-02-08 21:40: important pattern for review
- 2026-02-08 21:40: learning number A
- 2026-02-08 21:41: learning number B

## Other Section

Some content here.
`
	g.Expect(os.WriteFile(claudeMDPath, []byte(content), 0644)).To(Succeed())

	err := memory.RemoveFromClaudeMD(claudeMDPath, []string{"learning number A"})
	g.Expect(err).ToNot(HaveOccurred())

	result, err := os.ReadFile(claudeMDPath)
	g.Expect(err).ToNot(HaveOccurred())

	resultStr := string(result)
	g.Expect(resultStr).To(ContainSubstring("important pattern for review"))
	g.Expect(resultStr).ToNot(ContainSubstring("learning number A"))
	g.Expect(resultStr).To(ContainSubstring("learning number B"))
	g.Expect(resultStr).To(ContainSubstring("Other Section"))
	g.Expect(resultStr).To(ContainSubstring("Some content here"))
}

// TEST-1111: RemoveFromClaudeMD with nonexistent entry is a no-op
func TestRemoveFromClaudeMDNonexistentEntry(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	claudeMDPath := filepath.Join(tempDir, "CLAUDE.md")

	content := `## Promoted Learnings

- 2026-02-08 21:40: important pattern for review
`
	g.Expect(os.WriteFile(claudeMDPath, []byte(content), 0644)).To(Succeed())

	err := memory.RemoveFromClaudeMD(claudeMDPath, []string{"nonexistent entry"})
	g.Expect(err).ToNot(HaveOccurred())

	result, err := os.ReadFile(claudeMDPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(result)).To(ContainSubstring("important pattern for review"))
}

// TEST-1112: RemoveFromClaudeMD leaves other sections untouched
func TestRemoveFromClaudeMDOtherSectionsUntouched(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	claudeMDPath := filepath.Join(tempDir, "CLAUDE.md")

	content := `# Main Title

## Core Principles

1. Be good

## Promoted Learnings

- 2026-02-08 21:40: learning to remove

## Code Quality

- Run tests
`
	g.Expect(os.WriteFile(claudeMDPath, []byte(content), 0644)).To(Succeed())

	err := memory.RemoveFromClaudeMD(claudeMDPath, []string{"learning to remove"})
	g.Expect(err).ToNot(HaveOccurred())

	result, err := os.ReadFile(claudeMDPath)
	g.Expect(err).ToNot(HaveOccurred())

	resultStr := string(result)
	g.Expect(resultStr).To(ContainSubstring("Core Principles"))
	g.Expect(resultStr).To(ContainSubstring("Be good"))
	g.Expect(resultStr).To(ContainSubstring("Code Quality"))
	g.Expect(resultStr).To(ContainSubstring("Run tests"))
	g.Expect(resultStr).ToNot(ContainSubstring("learning to remove"))
}

// TEST-1113: RemoveFromClaudeMD on empty file returns nil
func TestRemoveFromClaudeMDEmptyFile(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	claudeMDPath := filepath.Join(tempDir, "CLAUDE.md")

	g.Expect(os.WriteFile(claudeMDPath, []byte(""), 0644)).To(Succeed())

	err := memory.RemoveFromClaudeMD(claudeMDPath, []string{"anything"})
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-1114: RemoveFromClaudeMD on missing file returns nil
func TestRemoveFromClaudeMDMissingFile(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	claudeMDPath := filepath.Join(tempDir, "nonexistent", "CLAUDE.md")

	err := memory.RemoveFromClaudeMD(claudeMDPath, []string{"anything"})
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-1115: RemoveFromClaudeMD removes multiple entries at once
func TestRemoveFromClaudeMDMultipleEntries(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	claudeMDPath := filepath.Join(tempDir, "CLAUDE.md")

	content := `## Promoted Learnings

- 2026-02-08 21:40: entry one
- 2026-02-08 21:41: entry two
- 2026-02-08 21:42: entry three
`
	g.Expect(os.WriteFile(claudeMDPath, []byte(content), 0644)).To(Succeed())

	err := memory.RemoveFromClaudeMD(claudeMDPath, []string{"entry one", "entry three"})
	g.Expect(err).ToNot(HaveOccurred())

	result, err := os.ReadFile(claudeMDPath)
	g.Expect(err).ToNot(HaveOccurred())

	resultStr := string(result)
	g.Expect(resultStr).ToNot(ContainSubstring("entry one"))
	g.Expect(resultStr).To(ContainSubstring("entry two"))
	g.Expect(resultStr).ToNot(ContainSubstring("entry three"))
}
