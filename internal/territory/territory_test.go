package territory_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/territory"
)

// TEST-570 traces: TASK-034
// Test Generate creates a territory map with structure section.
func TestGenerate_StructureSection(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a minimal Go project structure
	os.MkdirAll(filepath.Join(dir, "cmd", "myapp"), 0o755)
	os.MkdirAll(filepath.Join(dir, "internal", "pkg1"), 0o755)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "cmd", "myapp", "main.go"), []byte("package main\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "internal", "pkg1", "pkg.go"), []byte("package pkg1\n"), 0o644)

	m, err := territory.Generate(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(m.Structure.Root).To(Equal(dir))
	g.Expect(m.Structure.Languages).To(ContainElement("go"))
}

// TEST-571 traces: TASK-034
// Test Generate identifies entry points.
func TestGenerate_EntryPoints(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create CLI entry point
	os.MkdirAll(filepath.Join(dir, "cmd", "mycli"), 0o755)
	os.WriteFile(filepath.Join(dir, "cmd", "mycli", "main.go"), []byte("package main\n"), 0o644)

	m, err := territory.Generate(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(m.EntryPoints.CLI).To(ContainSubstring("cmd/mycli"))
}

// TEST-572 traces: TASK-034
// Test Generate counts packages.
func TestGenerate_PackagesSection(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create internal packages
	os.MkdirAll(filepath.Join(dir, "internal", "pkg1"), 0o755)
	os.MkdirAll(filepath.Join(dir, "internal", "pkg2"), 0o755)
	os.WriteFile(filepath.Join(dir, "internal", "pkg1", "a.go"), []byte("package pkg1\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "internal", "pkg2", "b.go"), []byte("package pkg2\n"), 0o644)

	m, err := territory.Generate(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(m.Packages.Count).To(BeNumerically(">=", 2))
	g.Expect(m.Packages.Internal).To(ContainElements("pkg1", "pkg2"))
}

// TEST-573 traces: TASK-034
// Test Generate detects test patterns.
func TestGenerate_TestsSection(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a Go project with test files
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0o644)
	os.MkdirAll(filepath.Join(dir, "internal", "pkg1"), 0o755)
	os.WriteFile(filepath.Join(dir, "internal", "pkg1", "pkg.go"), []byte("package pkg1\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "internal", "pkg1", "pkg_test.go"), []byte("package pkg1_test\n"), 0o644)

	m, err := territory.Generate(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(m.Tests.Pattern).To(Equal("*_test.go"))
	g.Expect(m.Tests.Count).To(BeNumerically(">=", 1))
}

// TEST-574 traces: TASK-034
// Test Generate identifies docs.
func TestGenerate_DocsSection(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create docs
	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "docs", "design.md"), []byte("# Design\n"), 0o644)

	m, err := territory.Generate(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(m.Docs.Readme).To(Equal("README.md"))
	g.Expect(m.Docs.Artifacts).To(ContainElement("docs/design.md"))
}

// TEST-575 traces: TASK-034
// Test Marshal produces output under token budget.
func TestMarshal_UnderTokenBudget(t *testing.T) {
	g := NewWithT(t)

	m := territory.Map{
		Structure: territory.Structure{
			Root:          "/path/to/project",
			Languages:     []string{"go"},
			BuildTool:     "go",
			TestFramework: "go test",
		},
		EntryPoints: territory.EntryPoints{
			CLI:       "cmd/app",
			PublicAPI: "project.go",
		},
		Packages: territory.Packages{
			Count:    10,
			Internal: []string{"pkg1", "pkg2", "pkg3"},
		},
		Tests: territory.Tests{
			Pattern: "*_test.go",
			Count:   50,
		},
		Docs: territory.Docs{
			Readme:    "README.md",
			Artifacts: []string{"docs/design.md", "docs/arch.md"},
		},
	}

	data, err := territory.Marshal(m)
	g.Expect(err).ToNot(HaveOccurred())
	// Must be under 4000 chars (1000 tokens)
	g.Expect(len(data)).To(BeNumerically("<", 4000))
}

// TEST-580 traces: TASK-035
// Test LoadCached returns cached map if recent.
func TestLoadCached_ReturnsCachedIfRecent(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a cached map with recent timestamp
	os.MkdirAll(filepath.Join(dir, "context"), 0o755)
	cached := territory.CachedMap{
		Map: territory.Map{
			Structure: territory.Structure{Root: dir},
			Packages:  territory.Packages{Count: 5},
		},
		CachedAt:  time.Now(),
		FileCount: 10,
	}
	data, _ := territory.MarshalCached(cached)
	os.WriteFile(filepath.Join(dir, "context", "territory.toml"), data, 0o644)

	// Create directory structure matching the cached count
	for i := 0; i < 10; i++ {
		os.WriteFile(filepath.Join(dir, fmt.Sprintf("file%d.go", i)), []byte("package main\n"), 0o644)
	}

	result, hit, err := territory.LoadCached(dir, time.Now)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(hit).To(BeTrue())
	g.Expect(result.Structure.Root).To(Equal(dir))
}

// TEST-581 traces: TASK-035
// Test LoadCached regenerates if cache is old.
func TestLoadCached_RegeneratesIfOld(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a cached map with old timestamp
	os.MkdirAll(filepath.Join(dir, "context"), 0o755)
	cached := territory.CachedMap{
		Map: territory.Map{
			Structure: territory.Structure{Root: "/old/path"},
		},
		CachedAt:  time.Now().Add(-2 * time.Hour), // 2 hours ago
		FileCount: 10,
	}
	data, _ := territory.MarshalCached(cached)
	os.WriteFile(filepath.Join(dir, "context", "territory.toml"), data, 0o644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644)

	result, hit, err := territory.LoadCached(dir, time.Now)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(hit).To(BeFalse())
	g.Expect(result.Structure.Root).To(Equal(dir)) // Fresh generation
}

// TEST-582 traces: TASK-035
// Test LoadCached regenerates if file count changed significantly.
func TestLoadCached_RegeneratesIfFileCountChanged(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a cached map with file count of 100
	os.MkdirAll(filepath.Join(dir, "context"), 0o755)
	cached := territory.CachedMap{
		Map: territory.Map{
			Structure: territory.Structure{Root: "/old/path"},
		},
		CachedAt:  time.Now(),
		FileCount: 100,
	}
	data, _ := territory.MarshalCached(cached)
	os.WriteFile(filepath.Join(dir, "context", "territory.toml"), data, 0o644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644)

	// Only create 5 files (> 10% difference from 100)
	for i := 0; i < 5; i++ {
		os.WriteFile(filepath.Join(dir, fmt.Sprintf("file%d.go", i)), []byte("package main\n"), 0o644)
	}

	result, hit, err := territory.LoadCached(dir, time.Now)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(hit).To(BeFalse())
	g.Expect(result.Structure.Root).To(Equal(dir))
}
