//go:build sqlite_fts5

package memory_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/memory"
)

// MockDirEntry implements os.DirEntry for testing
type MockDirEntry struct {
	name  string
	isDir bool
}

func (m MockDirEntry) Name() string               { return m.name }
func (m MockDirEntry) IsDir() bool                { return m.isDir }
func (m MockDirEntry) Type() os.FileMode          { return 0 }
func (m MockDirEntry) Info() (os.FileInfo, error) { return nil, fmt.Errorf("not implemented") }

// MockFileInfo implements os.FileInfo for testing
type MockFileInfo struct {
	name  string
	isDir bool
}

func (m MockFileInfo) Name() string       { return m.name }
func (m MockFileInfo) Size() int64        { return 0 }
func (m MockFileInfo) Mode() os.FileMode  { return 0 }
func (m MockFileInfo) ModTime() time.Time { return time.Time{} }
func (m MockFileInfo) IsDir() bool        { return m.isDir }
func (m MockFileInfo) Sys() interface{}   { return nil }

// MigrateMockFS implements memory.FileSystem for migrate tests
type MigrateMockFS struct {
	Files       map[string][]byte
	Directories map[string][]MockDirEntry
	Removed     []string
	Renamed     map[string]string // oldPath -> newPath
}

func (m *MigrateMockFS) ReadFile(path string) ([]byte, error) {
	content, exists := m.Files[path]
	if !exists {
		return nil, os.ErrNotExist
	}
	return content, nil
}

func (m *MigrateMockFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	m.Files[path] = data
	return nil
}

func (m *MigrateMockFS) ReadDir(path string) ([]os.DirEntry, error) {
	entries, exists := m.Directories[path]
	if !exists {
		return nil, os.ErrNotExist
	}
	result := make([]os.DirEntry, len(entries))
	for i, e := range entries {
		result[i] = e
	}
	return result, nil
}

func (m *MigrateMockFS) Stat(path string) (os.FileInfo, error) {
	// Check if path exists in Files
	if _, exists := m.Files[path]; exists {
		return MockFileInfo{name: path, isDir: false}, nil
	}
	// Check if path exists in Directories
	if _, exists := m.Directories[path]; exists {
		return MockFileInfo{name: path, isDir: true}, nil
	}
	// Check if path has been renamed (destination exists)
	for _, newPath := range m.Renamed {
		if newPath == path {
			return MockFileInfo{name: path, isDir: true}, nil
		}
	}
	return nil, os.ErrNotExist
}

func (m *MigrateMockFS) Rename(oldPath, newPath string) error {
	if m.Renamed == nil {
		m.Renamed = make(map[string]string)
	}
	m.Renamed[oldPath] = newPath
	// Move directory entry if it exists
	if entries, exists := m.Directories[oldPath]; exists {
		delete(m.Directories, oldPath)
		m.Directories[newPath] = entries
	}
	return nil
}

func (m *MigrateMockFS) Remove(path string) error {
	if m.Removed == nil {
		m.Removed = []string{}
	}
	m.Removed = append(m.Removed, path)
	delete(m.Directories, path)
	delete(m.Files, path)
	return nil
}

func (m *MigrateMockFS) MkdirAll(path string, perm os.FileMode) error {
	if m.Directories == nil {
		m.Directories = make(map[string][]MockDirEntry)
	}
	m.Directories[path] = []MockDirEntry{}
	return nil
}

// TestMigrateMemoryGenSkills verifies that migrateMemoryGenSkills moves
// skills from memory-gen/{slug}/ to mem-{slug}/ and removes memory-gen/.
func TestMigrateMemoryGenSkills(t *testing.T) {
	g := NewWithT(t)

	fs := &MigrateMockFS{
		Files: map[string][]byte{
			"/test/skills/memory-gen/foo/SKILL.md": []byte("# Foo Skill\n\nFoo content here."),
			"/test/skills/memory-gen/bar/SKILL.md": []byte("# Bar Skill\n\nBar content here."),
		},
		Directories: map[string][]MockDirEntry{
			"/test/skills/memory-gen": {
				{name: "foo", isDir: true},
				{name: "bar", isDir: true},
			},
		},
	}

	err := memory.MigrateMemoryGenSkillsForTest(fs, "/test/skills")
	g.Expect(err).ToNot(HaveOccurred())

	// Verify renames happened
	g.Expect(fs.Renamed).To(HaveKeyWithValue("/test/skills/memory-gen/foo", "/test/skills/mem-foo"))
	g.Expect(fs.Renamed).To(HaveKeyWithValue("/test/skills/memory-gen/bar", "/test/skills/mem-bar"))

	// Verify memory-gen directory was removed
	g.Expect(fs.Removed).To(ContainElement("/test/skills/memory-gen"))
}

// TestMigrateMemoryGenSkillsIdempotent verifies that calling migration twice
// is safe and doesn't cause errors.
func TestMigrateMemoryGenSkillsIdempotent(t *testing.T) {
	g := NewWithT(t)

	fs := &MigrateMockFS{
		Files: map[string][]byte{
			"/test/skills/memory-gen/foo/SKILL.md": []byte("# Foo Skill\n\nFoo content here."),
		},
		Directories: map[string][]MockDirEntry{
			"/test/skills/memory-gen": {
				{name: "foo", isDir: true},
			},
		},
	}

	// First migration
	err := memory.MigrateMemoryGenSkillsForTest(fs, "/test/skills")
	g.Expect(err).ToNot(HaveOccurred())

	// Second migration - memory-gen should not exist anymore
	fs.Directories = map[string][]MockDirEntry{} // Simulate directory gone
	err = memory.MigrateMemoryGenSkillsForTest(fs, "/test/skills")
	g.Expect(err).ToNot(HaveOccurred())
}

// TestMigrateMemoryGenSkillsNoOp verifies that migration is a no-op when
// there is no memory-gen/ directory to migrate.
func TestMigrateMemoryGenSkillsNoOp(t *testing.T) {
	g := NewWithT(t)

	fs := &MigrateMockFS{
		Files:       map[string][]byte{},
		Directories: map[string][]MockDirEntry{},
	}

	err := memory.MigrateMemoryGenSkillsForTest(fs, "/test/skills")
	g.Expect(err).ToNot(HaveOccurred())

	// Verify nothing was renamed or removed
	g.Expect(fs.Renamed).To(BeEmpty())
	g.Expect(fs.Removed).To(BeEmpty())
}
