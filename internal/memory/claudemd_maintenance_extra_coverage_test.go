//go:build sqlite_fts5

package memory_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestApplyClaudeMDProposal_ConsolidateInvalidTarget verifies error when consolidate target has no pipe separator.
func TestApplyClaudeMDProposal_ConsolidateInvalidTarget(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte("## Promoted Learnings\n\n- entry one\n- entry two\n"),
		},
	}

	proposal := memory.MaintenanceProposal{
		Action:  "consolidate",
		Tier:    "claude-md",
		Target:  "entry one without pipe",
		Preview: "merged entry",
	}

	err := memory.ApplyClaudeMDProposal(fs, "/test/CLAUDE.md", proposal)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("consolidate target must be"))
}

// TestApplyClaudeMDProposal_SplitInvalidPreview verifies error when split preview has no pipe separator.
func TestApplyClaudeMDProposal_SplitInvalidPreview(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte("## Promoted Learnings\n\n- long entry without split\n"),
		},
	}

	proposal := memory.MaintenanceProposal{
		Action:  "split",
		Tier:    "claude-md",
		Target:  "long entry without split",
		Preview: "single part only",
	}

	err := memory.ApplyClaudeMDProposal(fs, "/test/CLAUDE.md", proposal)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("split preview must contain at least 2 parts"))
}

// TestScanClaudeMDFeedback_DBError verifies ScanClaudeMDFeedback returns error when embeddings table is corrupt.
func TestScanClaudeMDFeedback_DBError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	// Write a CLAUDE.md with a Promoted Learnings section and entries
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
	content := "## Promoted Learnings\n\n- always use targ for builds\n"

	err := os.WriteFile(claudeMDPath, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize DB, insert a promoted+flagged embedding, then corrupt schema
	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec("INSERT INTO embeddings (content, source, promoted, flagged_for_review) VALUES (?, ?, 1, 1)",
		"always use targ for builds", "memory")
	g.Expect(err).ToNot(HaveOccurred())

	// Corrupt embeddings schema so the flagged query fails
	_, err = db.Exec("DROP TABLE embeddings")
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec("CREATE TABLE embeddings (id INTEGER PRIMARY KEY)")
	g.Expect(err).ToNot(HaveOccurred())

	proposals, err := memory.ScanClaudeMDFeedback(db, claudeMDPath)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to query flagged embeddings"))
	g.Expect(proposals).To(BeNil())
}

// TestScanClaudeMDFeedback_EmptyContent verifies ScanClaudeMDFeedback returns nil for empty file.
func TestScanClaudeMDFeedback_EmptyContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Write an empty CLAUDE.md
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")

	err = os.WriteFile(claudeMDPath, []byte(""), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	proposals, err := memory.ScanClaudeMDFeedback(db, claudeMDPath)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposals).To(BeNil())
}

// TestScanClaudeMDFeedback_NoPromotedLearnings verifies nil returned when no Promoted Learnings section.
func TestScanClaudeMDFeedback_NoPromotedLearnings(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")

	err = os.WriteFile(claudeMDPath, []byte("# Some other section\n\n- item\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	proposals, err := memory.ScanClaudeMDFeedback(db, claudeMDPath)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposals).To(BeNil())
}

// TestScanClaudeMDFeedback_NonExistentFile verifies ScanClaudeMDFeedback returns nil for non-existent CLAUDE.md.
func TestScanClaudeMDFeedback_NonExistentFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	claudeMDPath := filepath.Join(tmpDir, "nonexistent-CLAUDE.md")

	proposals, err := memory.ScanClaudeMDFeedback(db, claudeMDPath)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposals).To(BeNil())
}
