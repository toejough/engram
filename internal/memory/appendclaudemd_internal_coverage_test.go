//go:build sqlite_fts5

package memory

import (
	"errors"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// TestAppendToClaudeMDWithFS_ContentWithSectionBeforeNextSection inserts before next section.
func TestAppendToClaudeMDWithFS_ContentWithSectionBeforeNextSection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := newAppendMockFS()
	existing := "# Config\n\n## Promoted Learnings\n\n- Old learning\n\n## Other Section\n\nOther content\n"
	fs.files["/path/CLAUDE.md"] = []byte(existing)

	err := appendToClaudeMDWithFS(fs, "/path/CLAUDE.md", []string{"Inserted learning"})

	g.Expect(err).ToNot(HaveOccurred())
	result := string(fs.written["/path/CLAUDE.md"])
	g.Expect(result).To(ContainSubstring("- Inserted learning"))
	g.Expect(result).To(ContainSubstring("## Other Section"))
	// The inserted learning should come before "## Other Section"
	insertedIdx := strings.Index(result, "- Inserted learning")
	otherIdx := strings.Index(result, "## Other Section")
	g.Expect(insertedIdx).To(BeNumerically("<", otherIdx))
}

// TestAppendToClaudeMDWithFS_ContentWithSectionNoNextSection appends to existing section at end.
func TestAppendToClaudeMDWithFS_ContentWithSectionNoNextSection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := newAppendMockFS()
	existing := "# Config\n\n## Promoted Learnings\n\n- Old learning\n"
	fs.files["/path/CLAUDE.md"] = []byte(existing)

	err := appendToClaudeMDWithFS(fs, "/path/CLAUDE.md", []string{"New learning"})

	g.Expect(err).ToNot(HaveOccurred())
	result := string(fs.written["/path/CLAUDE.md"])
	g.Expect(result).To(ContainSubstring("- Old learning"))
	g.Expect(result).To(ContainSubstring("- New learning"))
	g.Expect(result).To(ContainSubstring("## Promoted Learnings"))
}

// TestAppendToClaudeMDWithFS_ContentWithoutSection adds section to non-empty content.
func TestAppendToClaudeMDWithFS_ContentWithoutSection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := newAppendMockFS()
	fs.files["/path/CLAUDE.md"] = []byte("# My Config\n\nSome existing content\n")

	err := appendToClaudeMDWithFS(fs, "/path/CLAUDE.md", []string{"Learn from tests"})

	g.Expect(err).ToNot(HaveOccurred())
	result := string(fs.written["/path/CLAUDE.md"])
	g.Expect(result).To(ContainSubstring("## Promoted Learnings"))
	g.Expect(result).To(ContainSubstring("- Learn from tests"))
	g.Expect(result).To(ContainSubstring("# My Config"))
}

// TestAppendToClaudeMDWithFS_EmptyFile verifies adding learnings to an empty file creates section.
func TestAppendToClaudeMDWithFS_EmptyFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := newAppendMockFS()
	// File doesn't exist (os.ErrNotExist), so empty content is used

	err := appendToClaudeMDWithFS(fs, "/path/CLAUDE.md", []string{"Use targ for builds"})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(fs.written["/path/CLAUDE.md"])).To(ContainSubstring("## Promoted Learnings"))
	g.Expect(string(fs.written["/path/CLAUDE.md"])).To(ContainSubstring("- Use targ for builds"))
}

// TestAppendToClaudeMDWithFS_MultipleItems writes multiple learnings.
func TestAppendToClaudeMDWithFS_MultipleItems(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := newAppendMockFS()

	err := appendToClaudeMDWithFS(fs, "/path/CLAUDE.md", []string{"First", "Second", "Third"})

	g.Expect(err).ToNot(HaveOccurred())
	result := string(fs.written["/path/CLAUDE.md"])
	g.Expect(result).To(ContainSubstring("- First"))
	g.Expect(result).To(ContainSubstring("- Second"))
	g.Expect(result).To(ContainSubstring("- Third"))
}

// TestAppendToClaudeMDWithFS_ReadError returns error when read fails with non-NotExist error.
func TestAppendToClaudeMDWithFS_ReadError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := newAppendMockFS()
	fs.readErr = errors.New("permission denied")

	err := appendToClaudeMDWithFS(fs, "/path/CLAUDE.md", []string{"Some learning"})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to read CLAUDE.md"))
}

// appendMockFS is a simple FileSystem implementation for testing appendToClaudeMDWithFS.
type appendMockFS struct {
	files    map[string][]byte
	readErr  error
	writeErr error
	written  map[string][]byte
}

func (m *appendMockFS) MkdirAll(path string, perm os.FileMode) error {
	return nil
}

func (m *appendMockFS) ReadDir(path string) ([]os.DirEntry, error) {
	return nil, errors.New("not implemented")
}

func (m *appendMockFS) ReadFile(path string) ([]byte, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}

	if data, ok := m.files[path]; ok {
		return data, nil
	}

	return nil, os.ErrNotExist
}

func (m *appendMockFS) Remove(path string) error {
	return errors.New("not implemented")
}

func (m *appendMockFS) Rename(oldPath, newPath string) error {
	return errors.New("not implemented")
}

func (m *appendMockFS) Stat(path string) (os.FileInfo, error) {
	return nil, errors.New("not implemented")
}

func (m *appendMockFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	if m.writeErr != nil {
		return m.writeErr
	}

	m.written[path] = append([]byte(nil), data...)

	return nil
}

func newAppendMockFS() *appendMockFS {
	return &appendMockFS{
		files:   make(map[string][]byte),
		written: make(map[string][]byte),
	}
}
