package conflict_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/conflict"
)

func TestRealFS_FileExists_False(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := conflict.RealFS{}

	g.Expect(fs.FileExists("/non-existent-conflict-file-xyz")).To(BeFalse())
}

func TestRealFS_FileExists_True(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	g.Expect(os.WriteFile(path, []byte("hello"), 0o644)).To(Succeed())

	fs := conflict.RealFS{}

	g.Expect(fs.FileExists(path)).To(BeTrue())
}

func TestRealFS_OpenAppend_CreatesFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "append.txt")

	fs := conflict.RealFS{}

	w, size, err := fs.OpenAppend(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(size).To(Equal(int64(0)))
	g.Expect(w).ToNot(BeNil())

	if w != nil {
		_, writeErr := w.Write([]byte("hello"))
		g.Expect(writeErr).ToNot(HaveOccurred())
		g.Expect(w.Close()).To(Succeed())
	}

	content, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(Equal("hello"))
}

func TestRealFS_OpenAppend_ExistingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.txt")
	g.Expect(os.WriteFile(path, []byte("first"), 0o644)).To(Succeed())

	fs := conflict.RealFS{}

	w, size, err := fs.OpenAppend(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(size).To(Equal(int64(5))) // "first" is 5 bytes
	g.Expect(w).ToNot(BeNil())

	if w != nil {
		_, writeErr := w.Write([]byte("second"))
		g.Expect(writeErr).ToNot(HaveOccurred())
		g.Expect(w.Close()).To(Succeed())
	}

	content, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(Equal("firstsecond"))
}

func TestRealFS_OpenAppend_ReadOnlyFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "readonly.txt")
	g.Expect(os.WriteFile(path, []byte("existing"), 0o444)).To(Succeed())
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	fs := conflict.RealFS{}

	_, _, err := fs.OpenAppend(path)
	g.Expect(err).To(HaveOccurred())
}

func TestRealFS_ReadFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "read.txt")
	g.Expect(os.WriteFile(path, []byte("content"), 0o644)).To(Succeed())

	fs := conflict.RealFS{}

	data, err := fs.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(data)).To(Equal("content"))
}

func TestRealFS_ReadFile_NotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := conflict.RealFS{}

	_, err := fs.ReadFile("/non-existent-conflict-read-xyz")
	g.Expect(err).To(HaveOccurred())
}

func TestRealFS_WriteFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "write.txt")

	fs := conflict.RealFS{}

	g.Expect(fs.WriteFile(path, []byte("written"))).To(Succeed())

	content, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(Equal("written"))
}

func TestRunCheck_FileUnreadable(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, conflict.ConflictFile)
	g.Expect(os.WriteFile(path, []byte("# Conflicts\n"), 0o644)).To(Succeed())
	g.Expect(os.Chmod(path, 0o000)).To(Succeed())
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	err := conflict.RunCheck(conflict.CheckArgs{Dir: dir})
	g.Expect(err).To(HaveOccurred())
}

func TestRunCheck_WithRealFS(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Conflicts

### CONF-001

**Skills:** pm, architect
**Traceability:** REQ-001
**Status:** resolved
**Description:** Sync vs eventual consistency
**Resolution:** Use eventual consistency
`

	g.Expect(os.MkdirAll(dir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, conflict.ConflictFile), []byte(content), 0o644)).To(Succeed())

	err := conflict.RunCheck(conflict.CheckArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunCreate_FileReadOnly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, conflict.ConflictFile)
	// Create a readable but not writable file so loadAll succeeds but OpenAppend fails
	g.Expect(os.WriteFile(path, []byte("# Conflicts\n"), 0o444)).To(Succeed())
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	err := conflict.RunCreate(conflict.CreateArgs{
		Dir:          dir,
		Skills:       "pm",
		Traceability: "REQ-001",
		Description:  "Test conflict",
	})
	g.Expect(err).To(HaveOccurred())
}

func TestRunCreate_WithRealFS(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := conflict.RunCreate(conflict.CreateArgs{
		Dir:          dir,
		Skills:       "pm, architect",
		Traceability: "REQ-001",
		Description:  "Test conflict",
	})
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(filepath.Join(dir, conflict.ConflictFile)).To(BeAnExistingFile())
}

func TestRunList_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := conflict.RunList(conflict.ListArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunList_FileUnreadable(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, conflict.ConflictFile)
	g.Expect(os.WriteFile(path, []byte("# Conflicts\n"), 0o644)).To(Succeed())
	g.Expect(os.Chmod(path, 0o000)).To(Succeed())
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	err := conflict.RunList(conflict.ListArgs{Dir: dir})
	g.Expect(err).To(HaveOccurred())
}

func TestRunList_WithRealFS(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Conflicts

### CONF-001

**Skills:** pm, architect
**Traceability:** REQ-001
**Status:** open
**Description:** Open conflict
**Resolution:** (pending)
`
	g.Expect(os.WriteFile(filepath.Join(dir, conflict.ConflictFile), []byte(content), 0o644)).To(Succeed())

	err := conflict.RunList(conflict.ListArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}
