package memory_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestWriteChangelogEntry_MkdirError verifies error is returned when memoryRoot cannot be created.
func TestWriteChangelogEntry_MkdirError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// /dev/null is a file, so creating a subdirectory under it fails
	err := memory.WriteChangelogEntry("/dev/null/cannot/create", memory.ChangelogEntry{
		Action: "test",
	})

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("failed to create memory directory"))
	}
}

// TestWriteChangelogEntry_OpenFileError verifies error when changelog.jsonl cannot be opened.
func TestWriteChangelogEntry_OpenFileError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	// Create changelog.jsonl as a directory to make os.OpenFile fail
	err := os.Mkdir(filepath.Join(tmpDir, "changelog.jsonl"), 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.WriteChangelogEntry(tmpDir, memory.ChangelogEntry{
		Action: "test",
	})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("failed to open changelog"))
	}
}
