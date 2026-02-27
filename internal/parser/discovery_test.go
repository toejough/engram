package parser_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/parser"
)

// MockFS implements parser.FileSystem for testing
type MockFS struct {
	Dirs  map[string]bool // path -> exists
	Files map[string]bool // path -> exists
}

func (m *MockFS) DirExists(path string) bool {
	return m.Dirs[path]
}

func (m *MockFS) FileExists(path string) bool {
	return m.Files[path]
}

// MockWalkFS implements parser.WalkableFS for testing test file discovery
type MockWalkFS struct {
	Files []string // All file paths in the tree
}

func (m *MockWalkFS) Walk(root string, fn func(path string, isDir bool) error) error {
	for _, f := range m.Files {
		// Determine if it's a directory by checking if anything has it as prefix
		isDir := false

		for _, other := range m.Files {
			if other != f && len(other) > len(f) && other[:len(f)] == f && other[len(f)] == '/' {
				isDir = true
				break
			}
		}

		if err := fn(f, isDir); err != nil {
			if err.Error() == "skip" {
				continue
			}

			return err
		}
	}

	return nil
}

// TEST-200 traces: TASK-005
// Test DiscoverDocsWithConfig discovers issues.md and glossary.md
func TestDiscoverDocsWithConfig_AllArtifacts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &MockFS{
		Dirs: map[string]bool{
			"/project/docs": true,
		},
		Files: map[string]bool{
			"/project/docs/requirements.md": true,
			"/project/docs/design.md":       true,
			"/project/docs/architecture.md": true,
			"/project/docs/tasks.md":        true,
			"/project/docs/issues.md":       true,
			"/project/docs/glossary.md":     true,
		},
	}

	cfg := &parser.DiscoveryConfig{
		DocsDir:      "docs",
		Requirements: "requirements.md",
		Design:       "design.md",
		Architecture: "architecture.md",
		Tasks:        "tasks.md",
		Issues:       "issues.md",
		Glossary:     "glossary.md",
	}

	result := parser.DiscoverDocsWithConfig("/project", cfg, fs)
	g.Expect(result.Paths).To(HaveLen(6))
	g.Expect(result.Paths).To(ContainElement("/project/docs/issues.md"))
	g.Expect(result.Paths).To(ContainElement("/project/docs/glossary.md"))
}

// TEST-199 traces: TASK-005
// Test DiscoverDocsWithConfig uses custom docs_dir from config
func TestDiscoverDocsWithConfig_CustomDocsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &MockFS{
		Dirs: map[string]bool{
			"/project/documentation": true,
		},
		Files: map[string]bool{
			"/project/documentation/requirements.md": true,
			"/project/documentation/design.md":       true,
		},
	}

	cfg := &parser.DiscoveryConfig{
		DocsDir:      "documentation",
		Requirements: "requirements.md",
		Design:       "design.md",
		Architecture: "architecture.md",
		Tasks:        "tasks.md",
		Issues:       "issues.md",
		Glossary:     "glossary.md",
	}

	result := parser.DiscoverDocsWithConfig("/project", cfg, fs)
	g.Expect(result.Paths).To(ConsistOf(
		"/project/documentation/requirements.md",
		"/project/documentation/design.md",
	))
	g.Expect(result.UsedFallback).To(BeFalse())
}

// TEST-201 traces: TASK-005
// Test DiscoverDocsWithConfig falls back to root with custom paths
func TestDiscoverDocsWithConfig_RootFallback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &MockFS{
		Dirs: map[string]bool{
			"/project/documentation": false,
		},
		Files: map[string]bool{
			"/project/reqs.md": true,
		},
	}

	cfg := &parser.DiscoveryConfig{
		DocsDir:      "documentation",
		Requirements: "reqs.md",
	}

	result := parser.DiscoverDocsWithConfig("/project", cfg, fs)
	g.Expect(result.Paths).To(ConsistOf("/project/reqs.md"))
	g.Expect(result.UsedFallback).To(BeTrue())
}

// TEST-085 traces: TASK-011
// Test discovery finds all known file types
func TestDiscoverDocs_AllFileTypes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &MockFS{
		Dirs: map[string]bool{
			"/project/docs": true,
		},
		Files: map[string]bool{
			"/project/docs/requirements.md": true,
			"/project/docs/design.md":       true,
			"/project/docs/architecture.md": true,
			"/project/docs/tasks.md":        true,
		},
	}

	result := parser.DiscoverDocs("/project", fs)
	g.Expect(result.Paths).To(HaveLen(4))
	g.Expect(result.Paths).To(ContainElement("/project/docs/requirements.md"))
	g.Expect(result.Paths).To(ContainElement("/project/docs/design.md"))
	g.Expect(result.Paths).To(ContainElement("/project/docs/architecture.md"))
	g.Expect(result.Paths).To(ContainElement("/project/docs/tasks.md"))
}

// TEST-081 traces: TASK-011
// Test discovery finds files in docs/ directory
func TestDiscoverDocs_DocsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &MockFS{
		Dirs: map[string]bool{
			"/project/docs": true,
		},
		Files: map[string]bool{
			"/project/docs/requirements.md": true,
			"/project/docs/architecture.md": true,
			"/project/docs/tasks.md":        true,
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

// TEST-083 traces: TASK-011
// Test discovery returns empty when no files exist
func TestDiscoverDocs_NoFiles(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

// TEST-086 traces: TASK-011
// Property test: docs/ takes priority over root
func TestDiscoverDocs_PropertyDocsHasPriority(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		knownFiles := []string{"requirements.md", "design.md", "architecture.md", "tasks.md"}

		// Generate random subset of files in docs/
		docsFileCount := rapid.IntRange(1, 4).Draw(rt, "docsFileCount")
		docsFiles := make(map[string]bool)
		dirs := map[string]bool{"/project/docs": true}

		for i := range docsFileCount {
			docsFiles["/project/docs/"+knownFiles[i]] = true
		}

		// Also add some files in root (should be ignored)
		rootFileCount := rapid.IntRange(0, 4).Draw(rt, "rootFileCount")
		for i := range rootFileCount {
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

// TEST-082 traces: TASK-011
// Test discovery falls back to root when docs/ missing
func TestDiscoverDocs_RootFallback(t *testing.T) {
	t.Parallel()
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

// TEST-114 traces: TASK-016
// Test excluding .git directory
func TestDiscoverTestFiles_ExcludesGit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &MockWalkFS{
		Files: []string{
			"/project/main_test.go",
			"/project/.git/hooks/pre-commit_test.go",
		},
	}

	paths := parser.DiscoverTestFiles("/project", fs)
	g.Expect(paths).To(ConsistOf("/project/main_test.go"))
}

// TEST-113 traces: TASK-016
// Test excluding vendor directory
func TestDiscoverTestFiles_ExcludesVendor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &MockWalkFS{
		Files: []string{
			"/project/main_test.go",
			"/project/vendor/dep_test.go",
		},
	}

	paths := parser.DiscoverTestFiles("/project", fs)
	g.Expect(paths).To(ConsistOf("/project/main_test.go"))
}

// TEST-112 traces: TASK-016
// Test discovering test files
func TestDiscoverTestFiles_FindsTests(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &MockWalkFS{
		Files: []string{
			"/project/main.go",
			"/project/main_test.go",
			"/project/internal/foo.go",
			"/project/internal/foo_test.go",
		},
	}

	paths := parser.DiscoverTestFiles("/project", fs)
	g.Expect(paths).To(ConsistOf(
		"/project/main_test.go",
		"/project/internal/foo_test.go",
	))
}

// TEST-115 traces: TASK-016
// Test ignoring non-test Go files
func TestDiscoverTestFiles_IgnoresNonTest(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &MockWalkFS{
		Files: []string{
			"/project/main.go",
			"/project/helper.go",
			"/project/main_test.go",
		},
	}

	paths := parser.DiscoverTestFiles("/project", fs)
	g.Expect(paths).To(ConsistOf("/project/main_test.go"))
}
