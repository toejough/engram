package parser_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/parser"
)

func TestDiscoverTestFiles_SkipsVendor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	vendorDir := filepath.Join(dir, "vendor")
	g.Expect(os.Mkdir(vendorDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vendorDir, "vendor_test.go"), []byte("package p"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "real_test.go"), []byte("package p"), 0o644)).To(Succeed())

	fs := parser.NewRealFS()
	files := parser.DiscoverTestFiles(dir, fs)
	g.Expect(files).To(HaveLen(1))
	g.Expect(files).ToNot(BeNil())

	if len(files) > 0 {
		g.Expect(files[0]).To(ContainSubstring("real_test.go"))
	}
}

func TestNewRealFS(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := parser.NewRealFS()
	g.Expect(fs).ToNot(BeNil())
}

func TestRealFS_DirExists_False(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := parser.NewRealFS()
	g.Expect(fs.DirExists("/non-existent-dir-parser-xyz-12345")).To(BeFalse())
}

func TestRealFS_DirExists_File(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	g.Expect(os.WriteFile(path, []byte("x"), 0o644)).To(Succeed())

	fs := parser.NewRealFS()
	// A file path should return false for DirExists
	g.Expect(fs.DirExists(path)).To(BeFalse())
}

func TestRealFS_DirExists_True(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	fs := parser.NewRealFS()
	g.Expect(fs.DirExists(dir)).To(BeTrue())
}

func TestRealFS_FileExists_Dir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	fs := parser.NewRealFS()
	// A directory path should return false for FileExists
	g.Expect(fs.FileExists(dir)).To(BeFalse())
}

func TestRealFS_FileExists_False(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := parser.NewRealFS()
	g.Expect(fs.FileExists("/non-existent-file-parser-xyz-12345")).To(BeFalse())
}

func TestRealFS_FileExists_True(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	g.Expect(os.WriteFile(path, []byte("content"), 0o644)).To(Succeed())

	fs := parser.NewRealFS()
	g.Expect(fs.FileExists(path)).To(BeTrue())
}

func TestRealFS_ReadFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "read.txt")
	g.Expect(os.WriteFile(path, []byte("hello world"), 0o644)).To(Succeed())

	fs := parser.NewRealFS()
	content, err := fs.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).To(Equal("hello world"))
}

func TestRealFS_ReadFile_NotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := parser.NewRealFS()
	_, err := fs.ReadFile("/non-existent-file-parser-xyz")
	g.Expect(err).To(HaveOccurred())
}

func TestRealFS_Walk(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	subdir := filepath.Join(dir, "sub")
	g.Expect(os.Mkdir(subdir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(subdir, "b.txt"), []byte("b"), 0o644)).To(Succeed())

	fs := parser.NewRealFS()

	var paths []string

	err := fs.Walk(dir, func(path string, isDir bool) error {
		paths = append(paths, path)
		return nil
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(paths).To(ContainElement(dir))
	g.Expect(paths).To(ContainElement(filepath.Join(dir, "a.txt")))
	g.Expect(paths).To(ContainElement(subdir))
	g.Expect(paths).To(ContainElement(filepath.Join(subdir, "b.txt")))
}

func TestRealFS_Walk_CallbackError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0o644)).To(Succeed())

	myErr := errors.New("custom walk error")
	fs := parser.NewRealFS()

	err := fs.Walk(dir, func(path string, isDir bool) error {
		if !isDir {
			return myErr
		}

		return nil
	})
	g.Expect(err).To(MatchError(myErr))
}

func TestRealFS_Walk_DirError(t *testing.T) {
	t.Parallel()

	if os.Getuid() == 0 {
		t.Skip("skipping chmod test when running as root")
	}

	g := NewWithT(t)
	dir := t.TempDir()

	secret := filepath.Join(dir, "secret")
	g.Expect(os.Mkdir(secret, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(secret, "hidden.txt"), []byte("x"), 0o644)).To(Succeed())
	g.Expect(os.Chmod(secret, 0o000)).To(Succeed())
	t.Cleanup(func() { _ = os.Chmod(secret, 0o755) })

	fs := parser.NewRealFS()

	err := fs.Walk(dir, func(_ string, _ bool) error {
		return nil
	})
	g.Expect(err).To(HaveOccurred())
}

func TestRealFS_Walk_MultipleFiles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.WriteFile(filepath.Join(dir, "c.txt"), []byte("c"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "d.txt"), []byte("d"), 0o644)).To(Succeed())

	fs := parser.NewRealFS()

	var fileCount int

	err := fs.Walk(dir, func(_ string, isDir bool) error {
		if !isDir {
			fileCount++
		}

		return nil
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(fileCount).To(Equal(2))
}
