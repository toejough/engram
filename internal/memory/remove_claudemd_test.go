package memory_test

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// MockFS implements memory.FileSystem for testing
type MockFS struct {
	Files map[string][]byte
}

func (m *MockFS) ReadFile(path string) ([]byte, error) {
	content, exists := m.Files[path]
	if !exists {
		return nil, os.ErrNotExist
	}
	return content, nil
}

func (m *MockFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	m.Files[path] = data
	return nil
}

func (m *MockFS) ReadDir(path string) ([]os.DirEntry, error) {
	return nil, fmt.Errorf("ReadDir not implemented in MockFS")
}

func (m *MockFS) Stat(path string) (os.FileInfo, error) {
	return nil, fmt.Errorf("Stat not implemented in MockFS")
}

func (m *MockFS) Rename(oldPath, newPath string) error {
	return fmt.Errorf("Rename not implemented in MockFS")
}

func (m *MockFS) Remove(path string) error {
	return fmt.Errorf("Remove not implemented in MockFS")
}

func (m *MockFS) MkdirAll(path string, perm os.FileMode) error {
	return nil
}

// ============================================================================
// Unit tests for removeFromClaudeMD
// traces: ISSUE-184
// ============================================================================

// TEST-1110: RemoveFromClaudeMD removes matching entry
func TestRemoveFromClaudeMDRemovesEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `# Working With Joe

## Promoted Learnings

- 2026-02-08 21:40: important pattern for review
- 2026-02-08 21:40: learning number A
- 2026-02-08 21:41: learning number B

## Other Section

Some content here.
`
	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte(content),
		},
	}

	err := memory.RemoveFromClaudeMD(fs, "/test/CLAUDE.md", []string{"learning number A"})
	g.Expect(err).ToNot(HaveOccurred())

	result := string(fs.Files["/test/CLAUDE.md"])
	g.Expect(result).To(ContainSubstring("important pattern for review"))
	g.Expect(result).ToNot(ContainSubstring("learning number A"))
	g.Expect(result).To(ContainSubstring("learning number B"))
	g.Expect(result).To(ContainSubstring("Other Section"))
	g.Expect(result).To(ContainSubstring("Some content here"))
}

// TEST-1111: RemoveFromClaudeMD with nonexistent entry is a no-op
func TestRemoveFromClaudeMDNonexistentEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `## Promoted Learnings

- 2026-02-08 21:40: important pattern for review
`
	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte(content),
		},
	}

	err := memory.RemoveFromClaudeMD(fs, "/test/CLAUDE.md", []string{"nonexistent entry"})
	g.Expect(err).ToNot(HaveOccurred())

	result := string(fs.Files["/test/CLAUDE.md"])
	g.Expect(result).To(ContainSubstring("important pattern for review"))
}

// TEST-1112: RemoveFromClaudeMD leaves other sections untouched
func TestRemoveFromClaudeMDOtherSectionsUntouched(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `# Main Title

## Core Principles

1. Be good

## Promoted Learnings

- 2026-02-08 21:40: learning to remove

## Code Quality

- Run tests
`
	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte(content),
		},
	}

	err := memory.RemoveFromClaudeMD(fs, "/test/CLAUDE.md", []string{"learning to remove"})
	g.Expect(err).ToNot(HaveOccurred())

	result := string(fs.Files["/test/CLAUDE.md"])
	g.Expect(result).To(ContainSubstring("Core Principles"))
	g.Expect(result).To(ContainSubstring("Be good"))
	g.Expect(result).To(ContainSubstring("Code Quality"))
	g.Expect(result).To(ContainSubstring("Run tests"))
	g.Expect(result).ToNot(ContainSubstring("learning to remove"))
}

// TEST-1113: RemoveFromClaudeMD on empty file returns nil
func TestRemoveFromClaudeMDEmptyFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte(""),
		},
	}

	err := memory.RemoveFromClaudeMD(fs, "/test/CLAUDE.md", []string{"anything"})
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-1114: RemoveFromClaudeMD on missing file returns nil
func TestRemoveFromClaudeMDMissingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &MockFS{
		Files: map[string][]byte{},
	}

	err := memory.RemoveFromClaudeMD(fs, "/nonexistent/CLAUDE.md", []string{"anything"})
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-1115: RemoveFromClaudeMD removes multiple entries at once
func TestRemoveFromClaudeMDMultipleEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `## Promoted Learnings

- 2026-02-08 21:40: entry one
- 2026-02-08 21:41: entry two
- 2026-02-08 21:42: entry three
`
	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte(content),
		},
	}

	err := memory.RemoveFromClaudeMD(fs, "/test/CLAUDE.md", []string{"entry one", "entry three"})
	g.Expect(err).ToNot(HaveOccurred())

	result := string(fs.Files["/test/CLAUDE.md"])
	g.Expect(result).ToNot(ContainSubstring("entry one"))
	g.Expect(result).To(ContainSubstring("entry two"))
	g.Expect(result).ToNot(ContainSubstring("entry three"))
}
