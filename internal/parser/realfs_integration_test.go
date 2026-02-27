//go:build integration

package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/parser"
)

// TEST-172 traces: TASK-028
// Test RealFS DirExists returns false for non-existent directory
func TestRealFS_DirExists_False(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := parser.NewRealFS()
	g.Expect(fs.DirExists("/non-existent-dir-12345")).To(BeFalse())
}

// TEST-171 traces: TASK-028
// Test RealFS DirExists returns true for existing directory
func TestRealFS_DirExists_True(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	fs := parser.NewRealFS()
	g.Expect(fs.DirExists(dir)).To(BeTrue())
}

// TEST-174 traces: TASK-028
// Test RealFS FileExists returns false for non-existent file
func TestRealFS_FileExists_False(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := parser.NewRealFS()
	g.Expect(fs.FileExists("/non-existent-file-12345")).To(BeFalse())
}

// TEST-173 traces: TASK-028
// Test RealFS FileExists returns true for existing file
func TestRealFS_FileExists_True(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	g.Expect(os.WriteFile(path, []byte("content"), 0644)).To(Succeed())

	fs := parser.NewRealFS()
	g.Expect(fs.FileExists(path)).To(BeTrue())
}

// TEST-178 traces: TASK-028
// Test RealFS implements CollectableFS
func TestRealFS_ImplementsCollectableFS(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := parser.NewRealFS()
	var _ parser.CollectableFS = fs
	g.Expect(fs).ToNot(BeNil())
}

// TEST-175 traces: TASK-028
// Test RealFS ReadFile returns file content
func TestRealFS_ReadFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	g.Expect(os.WriteFile(path, []byte("hello world"), 0644)).To(Succeed())

	fs := parser.NewRealFS()
	content, err := fs.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).To(Equal("hello world"))
}

// TEST-176 traces: TASK-028
// Test RealFS ReadFile returns error for non-existent file
func TestRealFS_ReadFile_NotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := parser.NewRealFS()
	_, err := fs.ReadFile("/non-existent-file-12345")
	g.Expect(err).To(HaveOccurred())
}

// TEST-177 traces: TASK-028
// Test RealFS Walk traverses directory tree
func TestRealFS_Walk(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Create structure
	subdir := filepath.Join(dir, "sub")
	g.Expect(os.Mkdir(subdir, 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(subdir, "b.txt"), []byte("b"), 0644)).To(Succeed())

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
