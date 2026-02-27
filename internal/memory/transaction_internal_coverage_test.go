//go:build sqlite_fts5

package memory

import (
	"os"
	"testing"

	. "github.com/onsi/gomega"
)

// TestCopyDir_NonExistentSource verifies copyDir returns error when source does not exist.
func TestCopyDir_NonExistentSource(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := copyDir(osDirOps{}, "/nonexistent/source/path", t.TempDir())

	g.Expect(err).To(HaveOccurred())
}

// TestCopyDir_UnwritableDestination verifies copyDir returns error when destination cannot be created.
func TestCopyDir_UnwritableDestination(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	err := copyDir(osDirOps{}, tmpDir, "/dev/null/cannot/create/here")

	g.Expect(err).To(HaveOccurred())
}

// TestCopyFile_NonExistentSource verifies copyFile returns error when source does not exist.
func TestCopyFile_NonExistentSource(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := copyFile("/nonexistent/source/file.txt", t.TempDir()+"/dest.txt")

	g.Expect(err).To(HaveOccurred())
}

// TestCopyFile_UnwritableDestination verifies copyFile returns error when destination cannot be created.
func TestCopyFile_UnwritableDestination(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Create a valid source file
	tmpDir := t.TempDir()
	srcPath := tmpDir + "/source.txt"

	err := os.WriteFile(srcPath, []byte("content"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	err = copyFile(srcPath, "/dev/null/cannot/write/here.txt")

	g.Expect(err).To(HaveOccurred())
}
