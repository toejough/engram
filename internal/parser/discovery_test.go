package parser_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/parser"
)

// MockFS implements parser.FileSystem for testing
type MockFS struct {
	Dirs  map[string]bool   // path -> exists
	Files map[string]bool   // path -> exists
}

func (m *MockFS) DirExists(path string) bool {
	return m.Dirs[path]
}

func (m *MockFS) FileExists(path string) bool {
	return m.Files[path]
}

// TEST-081 traces: TASK-011
// Test discovery finds files in docs/ directory
func TestDiscoverDocs_DocsDir(t *testing.T) {
	g := NewWithT(t)

	fs := &MockFS{
		Dirs: map[string]bool{
			"/project/docs": true,
		},
		Files: map[string]bool{
			"/project/docs/requirements.md":  true,
			"/project/docs/architecture.md":  true,
			"/project/docs/tasks.md":         true,
		},
	}

	result := parser.DiscoverDocs("/project", fs)
	g.Expect(result.Paths).To(ConsistOf(
		"/project/docs/requirements.md",
		"/project/docs/architecture.md",
		"/project/docs/tasks.md",
	))
	g.Expect(result.UsedFallback).To(BeFalse())
}

// TEST-082 traces: TASK-011
// Test discovery falls back to root when docs/ missing
func TestDiscoverDocs_RootFallback(t *testing.T) {
	g := NewWithT(t)

	fs := &MockFS{
		Dirs: map[string]bool{
			"/project/docs": false,
		},
		Files: map[string]bool{
			"/project/requirements.md": true,
			"/project/design.md":       true,
		},
	}

	result := parser.DiscoverDocs("/project", fs)
	g.Expect(result.Paths).To(ConsistOf(
		"/project/requirements.md",
		"/project/design.md",
	))
	g.Expect(result.UsedFallback).To(BeTrue())
}

// TEST-083 traces: TASK-011
// Test discovery returns empty when no files exist
func TestDiscoverDocs_NoFiles(t *testing.T) {
	g := NewWithT(t)

	fs := &MockFS{
		Dirs:  map[string]bool{},
		Files: map[string]bool{},
	}

	result := parser.DiscoverDocs("/project", fs)
	g.Expect(result.Paths).To(BeEmpty())
	g.Expect(result.UsedFallback).To(BeFalse())
}

// TEST-084 traces: TASK-011
// Test discovery prefers docs/ even if root also has files
func TestDiscoverDocs_PrefersDocsDir(t *testing.T) {
	g := NewWithT(t)

	fs := &MockFS{
		Dirs: map[string]bool{
			"/project/docs": true,
		},
		Files: map[string]bool{
			"/project/docs/requirements.md": true,
			"/project/requirements.md":      true, // Should be ignored
		},
	}

	result := parser.DiscoverDocs("/project", fs)
	g.Expect(result.Paths).To(ConsistOf("/project/docs/requirements.md"))
	g.Expect(result.UsedFallback).To(BeFalse())
}

// TEST-085 traces: TASK-011
// Test discovery finds all known file types
func TestDiscoverDocs_AllFileTypes(t *testing.T) {
	g := NewWithT(t)

	fs := &MockFS{
		Dirs: map[string]bool{
			"/project/docs": true,
		},
		Files: map[string]bool{
			"/project/docs/requirements.md":  true,
			"/project/docs/design.md":        true,
			"/project/docs/architecture.md":  true,
			"/project/docs/tasks.md":         true,
		},
	}

	result := parser.DiscoverDocs("/project", fs)
	g.Expect(result.Paths).To(HaveLen(4))
	g.Expect(result.Paths).To(ContainElement("/project/docs/requirements.md"))
	g.Expect(result.Paths).To(ContainElement("/project/docs/design.md"))
	g.Expect(result.Paths).To(ContainElement("/project/docs/architecture.md"))
	g.Expect(result.Paths).To(ContainElement("/project/docs/tasks.md"))
}

// TEST-086 traces: TASK-011
// Property test: docs/ takes priority over root
func TestDiscoverDocs_PropertyDocsHasPriority(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		knownFiles := []string{"requirements.md", "design.md", "architecture.md", "tasks.md"}

		// Generate random subset of files in docs/
		docsFileCount := rapid.IntRange(1, 4).Draw(rt, "docsFileCount")
		docsFiles := make(map[string]bool)
		dirs := map[string]bool{"/project/docs": true}

		for i := 0; i < docsFileCount; i++ {
			docsFiles["/project/docs/"+knownFiles[i]] = true
		}

		// Also add some files in root (should be ignored)
		rootFileCount := rapid.IntRange(0, 4).Draw(rt, "rootFileCount")
		for i := 0; i < rootFileCount; i++ {
			docsFiles["/project/"+knownFiles[i]] = true
		}

		fs := &MockFS{Dirs: dirs, Files: docsFiles}
		result := parser.DiscoverDocs("/project", fs)

		// Should only contain docs/ files
		g.Expect(result.UsedFallback).To(BeFalse())
		for _, path := range result.Paths {
			g.Expect(path).To(HavePrefix("/project/docs/"))
		}
	})
}
