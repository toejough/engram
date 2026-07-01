package cli_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestAtomicWriteFile_FailureDoesNotTouchOriginal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	// Create a subdirectory to hold the original file.
	subdir := filepath.Join(dir, "sub")
	mkErr := os.Mkdir(subdir, 0o700)
	g.Expect(mkErr).NotTo(HaveOccurred())

	if mkErr != nil {
		return
	}

	target := filepath.Join(subdir, "original.txt")
	original := []byte("original untouched content")

	writeErr := os.WriteFile(target, original, 0o600)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	// Make the directory read-only so CreateTemp fails.
	chmodErr := os.Chmod(subdir, 0o500)
	g.Expect(chmodErr).NotTo(HaveOccurred())

	if chmodErr != nil {
		return
	}

	// Restore permissions so TempDir cleanup can succeed.
	t.Cleanup(func() { _ = os.Chmod(subdir, 0o700) })

	// Make the directory readable again for the final assertions.
	defer func() { _ = os.Chmod(subdir, 0o700) }()

	err := cli.ExportAtomicWriteFile(target, []byte("new content"), 0o600)
	g.Expect(err).To(HaveOccurred(), "write to read-only dir must fail")

	// Restore permissions to check the original.
	_ = os.Chmod(subdir, 0o700)

	got, readErr := os.ReadFile(target)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(got).To(Equal(original), "original file must be untouched after failure")

	// No leftover temp files.
	tmpFiles, globErr := filepath.Glob(filepath.Join(subdir, ".original.txt.tmp-*"))
	g.Expect(globErr).NotTo(HaveOccurred())
	g.Expect(tmpFiles).To(BeEmpty(), "no leftover .tmp-* files must remain after failure")
}

func TestAtomicWriteFile_NoLeftoverTempFiles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	target := filepath.Join(dir, "out.txt")

	err := cli.ExportAtomicWriteFile(target, []byte("data"), 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tmpFiles, globErr := filepath.Glob(filepath.Join(dir, ".out.txt.tmp-*"))
	g.Expect(globErr).NotTo(HaveOccurred())
	g.Expect(tmpFiles).To(BeEmpty(), "no leftover .tmp-* files must remain after success")
}

func TestAtomicWriteFile_OverwritesExistingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	target := filepath.Join(dir, "out.txt")
	original := []byte("original content")
	updated := []byte("updated content")

	writeErr := os.WriteFile(target, original, 0o600)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	err := cli.ExportAtomicWriteFile(target, updated, 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	got, readErr := os.ReadFile(target)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(got).To(Equal(updated), "file must contain the updated bytes")
}

func TestAtomicWriteFile_RenameFailure_CleansTempAndLeavesOriginalUntouched(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	target := filepath.Join(dir, "file.txt")
	original := []byte("original content")

	// Write the original file before the failing atomic write.
	writeErr := os.WriteFile(target, original, 0o600)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	// Track the temp path so we can verify the defer cleans it up.
	var tmpSeen string

	err := cli.ExportDoAtomicWrite(target, []byte("new content"), 0o600,
		func(src, _ string) error {
			tmpSeen = src

			return errors.New("injected rename failure")
		},
	)

	g.Expect(err).To(MatchError(ContainSubstring("rename")), "error must mention rename")

	// Original file must be untouched.
	got, readErr := os.ReadFile(target)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(got).To(Equal(original), "original must be untouched after rename failure")

	// Temp file must be removed by the defer cleanup.
	if tmpSeen != "" {
		_, statErr := os.Stat(tmpSeen)
		g.Expect(os.IsNotExist(statErr)).To(BeTrue(), "temp file must be cleaned up by defer")
	}
}

func TestAtomicWriteFile_WritesNewFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	target := filepath.Join(dir, "out.txt")
	content := []byte("hello atomic world")

	err := cli.ExportAtomicWriteFile(target, content, 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	got, readErr := os.ReadFile(target)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(got).To(Equal(content), "file must contain exactly the written bytes")
}
