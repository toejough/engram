package conflict_test

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/conflict"
)

// MockFS implements conflict.FileSystem for testing
type MockFS struct {
	Files map[string][]byte
}

func (m *MockFS) FileExists(path string) bool {
	_, exists := m.Files[path]
	return exists
}

func (m *MockFS) OpenAppend(path string) (io.WriteCloser, int64, error) {
	if m.Files == nil {
		m.Files = make(map[string][]byte)
	}

	existing := m.Files[path]
	size := int64(len(existing))

	buf := bytes.NewBuffer(existing)

	return &mockWriteCloser{Buffer: buf, fs: m, path: path}, size, nil
}

func (m *MockFS) ReadFile(path string) ([]byte, error) {
	content, exists := m.Files[path]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	return content, nil
}

func (m *MockFS) WriteFile(path string, data []byte) error {
	if m.Files == nil {
		m.Files = make(map[string][]byte)
	}

	m.Files[path] = data

	return nil
}

func TestCheck(t *testing.T) {
	t.Parallel()
	t.Run("finds resolved conflicts", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{}

		// Write a conflicts file with one resolved entry
		content := `# Conflicts

### CONF-001

**Skills:** pm, architect
**Traceability:** REQ-001
**Status:** resolved
**Description:** Sync vs eventual consistency
**Resolution:** Use eventual consistency with UI optimistic updates
`
		g.Expect(fs.WriteFile(filepath.Join("testdir", conflict.ConflictFile), []byte(content))).To(Succeed())

		result, err := conflict.Check("testdir", fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Resolved).To(HaveLen(1))
		g.Expect(result.Resolved[0].ID).To(Equal("CONF-001"))
		g.Expect(result.Resolved[0].Resolution).To(ContainSubstring("eventual consistency"))
	})

	t.Run("returns empty for no resolved", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{}

		_, err := conflict.Create("testdir", "pm, design", "REQ-001", "Unresolved conflict", fs)
		g.Expect(err).ToNot(HaveOccurred())

		result, err := conflict.Check("testdir", fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Resolved).To(BeEmpty())
	})

	t.Run("returns empty for no file", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{}

		result, err := conflict.Check("testdir", fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Resolved).To(BeEmpty())
	})
}

func TestCreate(t *testing.T) {
	t.Parallel()
	t.Run("creates first conflict", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{}

		id, err := conflict.Create("testdir", "pm, architect", "REQ-001, ARCH-003", "PM requires sync but architect says eventual consistency", fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(id).To(Equal("CONF-001"))

		// File should exist
		g.Expect(fs.FileExists(filepath.Join("testdir", conflict.ConflictFile))).To(BeTrue())
	})

	t.Run("auto-increments ID", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{}

		id1, err := conflict.Create("testdir", "pm, design", "REQ-001", "First conflict", fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(id1).To(Equal("CONF-001"))

		id2, err := conflict.Create("testdir", "design, architect", "DES-002", "Second conflict", fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(id2).To(Equal("CONF-002"))
	})
}

func TestList(t *testing.T) {
	t.Parallel()
	t.Run("lists all conflicts", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{}

		content := `# Conflicts

### CONF-001

**Skills:** pm, architect
**Traceability:** REQ-001
**Status:** open
**Description:** First issue
**Resolution:** (pending)

### CONF-002

**Skills:** design, pm
**Traceability:** DES-002
**Status:** resolved
**Description:** Second issue
**Resolution:** Use design's approach
`
		g.Expect(fs.WriteFile(filepath.Join("testdir", conflict.ConflictFile), []byte(content))).To(Succeed())

		result, err := conflict.List("testdir", "", fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Conflicts).To(HaveLen(2))
		g.Expect(result.Open).To(Equal(1))
		g.Expect(result.Resolved).To(Equal(1))
	})

	t.Run("filters by status", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{}

		content := `# Conflicts

### CONF-001

**Skills:** pm, architect
**Traceability:** REQ-001
**Status:** open
**Description:** Open issue
**Resolution:** (pending)

### CONF-002

**Skills:** design, pm
**Traceability:** DES-002
**Status:** resolved
**Description:** Resolved issue
**Resolution:** Done
`
		g.Expect(fs.WriteFile(filepath.Join("testdir", conflict.ConflictFile), []byte(content))).To(Succeed())

		result, err := conflict.List("testdir", "open", fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Conflicts).To(HaveLen(1))
		g.Expect(result.Conflicts[0].ID).To(Equal("CONF-001"))
	})

	t.Run("returns empty for no file", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{}

		result, err := conflict.List("testdir", "", fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Conflicts).To(BeEmpty())
	})
}

type mockWriteCloser struct {
	*bytes.Buffer

	fs   *MockFS
	path string
}

func (m *mockWriteCloser) Close() error {
	m.fs.Files[m.path] = m.Bytes()
	return nil
}
