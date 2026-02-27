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

func TestGenerate_CountsTestFiles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.WriteFile(filepath.Join(dir, "foo_test.go"), []byte("package main\n"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "bar_test.go"), []byte("package main\n"), 0o644)).To(Succeed())

	m, err := territory.Generate(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(m.Tests.Count).To(Equal(2))
}

func TestGenerate_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	m, err := territory.Generate(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(m.Structure.Root).To(Equal(dir))
	g.Expect(m.Structure.Languages).To(BeEmpty())
}

func TestGenerate_WithCmdDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(dir, "cmd", "mytool"), 0o755)).To(Succeed())

	m, err := territory.Generate(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(m.EntryPoints.CLI).To(ContainSubstring("mytool"))
}

func TestGenerate_WithGoMod(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n\ngo 1.21\n"), 0o644)).To(Succeed())

	m, err := territory.Generate(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(m.Structure.Languages).To(ContainElement("go"))
	g.Expect(m.Structure.BuildTool).To(Equal("go"))
	g.Expect(m.Structure.TestFramework).To(Equal("go test"))
	g.Expect(m.Tests.Pattern).To(Equal("*_test.go"))
}

func TestGenerate_WithInternalDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(dir, "internal", "pkg-a"), 0o755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(dir, "internal", "pkg-b"), 0o755)).To(Succeed())

	m, err := territory.Generate(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(m.Packages.Count).To(Equal(2))
	g.Expect(m.Packages.Internal).To(ConsistOf("pkg-a", "pkg-b"))
}

func TestGenerate_WithPackageJson(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o644)).To(Succeed())

	m, err := territory.Generate(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(m.Structure.Languages).To(ContainElement("javascript"))
	g.Expect(m.Structure.BuildTool).To(Equal("npm"))
}

func TestGenerate_WithReadme(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Project"), 0o644)).To(Succeed())

	m, err := territory.Generate(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(m.Docs.Readme).To(Equal("README.md"))
}

// TestLoadCached_CacheHitWithNonZeroFileCount covers the abs() function (negative n branch).
// Cache has FileCount=5, current dir has 3 files + 1 territory.toml = 4 files.
// diff = abs(4 - 5) = abs(-1) = 1; threshold = max(0, 1) = 1; diff <= threshold → cache hit.
func TestLoadCached_CacheHitWithNonZeroFileCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	for i := range 3 {
		g.Expect(os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%d.txt", i)), []byte("x"), 0o644)).To(Succeed())
	}

	g.Expect(os.MkdirAll(filepath.Join(dir, "context"), 0o755)).To(Succeed())

	cached := territory.CachedMap{
		Map: territory.Map{
			Structure: territory.Structure{Root: dir},
			Packages:  territory.Packages{Internal: []string{}},
			Docs:      territory.Docs{Artifacts: []string{}},
		},
		CachedAt:  time.Now(),
		FileCount: 5,
	}

	data, err := territory.MarshalCached(cached)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(os.WriteFile(filepath.Join(dir, "context", "territory.toml"), data, 0o644)).To(Succeed())

	fixedTime := time.Now()
	_, cacheHit, err := territory.LoadCached(dir, func() time.Time { return fixedTime })
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(cacheHit).To(BeTrue())
}

// TestLoadCached_CacheMissFileCountIncreased covers the abs() function (positive n branch).
// Cache has FileCount=3, current dir has 1 territory.toml + 5 extra = 6 files.
// diff = abs(6 - 3) = 3; threshold = max(0, 1) = 1; diff > threshold → cache miss.
func TestLoadCached_CacheMissFileCountIncreased(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(dir, "context"), 0o755)).To(Succeed())

	cached := territory.CachedMap{
		Map: territory.Map{
			Structure: territory.Structure{Root: dir},
			Packages:  territory.Packages{Internal: []string{}},
			Docs:      territory.Docs{Artifacts: []string{}},
		},
		CachedAt:  time.Now(),
		FileCount: 3,
	}

	data, err := territory.MarshalCached(cached)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(os.WriteFile(filepath.Join(dir, "context", "territory.toml"), data, 0o644)).To(Succeed())

	for i := range 5 {
		g.Expect(os.WriteFile(filepath.Join(dir, fmt.Sprintf("extra%d.txt", i)), []byte("x"), 0o644)).To(Succeed())
	}

	fixedTime := time.Now()
	_, cacheHit, err := territory.LoadCached(dir, func() time.Time { return fixedTime })
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(cacheHit).To(BeFalse())
}

func TestLoadCached_GeneratesWhenNoCacheExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	now := func() time.Time { return time.Now() }
	m, cacheHit, err := territory.LoadCached(dir, now)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(cacheHit).To(BeFalse())
	g.Expect(m.Structure.Root).To(Equal(dir))
}

func TestLoadCached_UsesCacheWhenFresh(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	fixedTime := time.Now()
	now := func() time.Time { return fixedTime }

	// First call: generate and cache
	_, _, err := territory.LoadCached(dir, now)
	g.Expect(err).ToNot(HaveOccurred())

	// Second call with same time: should use cache
	_, cacheHit, err := territory.LoadCached(dir, now)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(cacheHit).To(BeTrue())
}

func TestMarshalCached_RoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Now()
	cached := territory.CachedMap{
		Map: territory.Map{
			Structure: territory.Structure{Root: "/test"},
			Packages:  territory.Packages{Internal: []string{}},
			Docs:      territory.Docs{Artifacts: []string{}},
		},
		CachedAt:  now,
		FileCount: 42,
	}

	data, err := territory.MarshalCached(cached)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(data).ToNot(BeEmpty())
	g.Expect(string(data)).To(ContainSubstring("file_count"))
}

func TestMarshal_RoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	m := territory.Map{
		Structure: territory.Structure{
			Root:      "/test",
			Languages: []string{"go"},
			BuildTool: "go",
		},
		Packages: territory.Packages{
			Count:    1,
			Internal: []string{"mypkg"},
		},
		Docs: territory.Docs{
			Artifacts: []string{},
		},
	}

	data, err := territory.Marshal(m)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(data).ToNot(BeEmpty())
	g.Expect(string(data)).To(ContainSubstring("go"))
}

// TestRunMap_CacheHit covers the cacheHit=true branch (prints "Using cached territory map").
// First call generates and saves cache with FileCount=0; second call finds it fresh → cacheHit=true.
func TestRunMap_CacheHit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := territory.RunMap(territory.MapArgs{Dir: dir, Cached: true})
	g.Expect(err).ToNot(HaveOccurred())

	err = territory.RunMap(territory.MapArgs{Dir: dir, Cached: true})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunMap_DefaultDir(t *testing.T) {
	// No t.Parallel: uses t.Chdir
	g := NewWithT(t)
	dir := t.TempDir()
	t.Chdir(dir)

	err := territory.RunMap(territory.MapArgs{})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunMap_WithCachedFlag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := territory.RunMap(territory.MapArgs{Dir: dir, Cached: true})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunMap_WithExplicitDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := territory.RunMap(territory.MapArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunMap_WithOutputFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	outFile := filepath.Join(dir, "output.toml")

	err := territory.RunMap(territory.MapArgs{Dir: dir, Output: outFile})
	g.Expect(err).ToNot(HaveOccurred())

	_, statErr := os.Stat(outFile)
	g.Expect(statErr).ToNot(HaveOccurred())
}

func TestRunShow_DefaultDir(t *testing.T) {
	// No t.Parallel: uses t.Chdir
	g := NewWithT(t)
	dir := t.TempDir()
	t.Chdir(dir)

	err := territory.RunShow(territory.ShowArgs{})
	g.Expect(err).To(HaveOccurred())
}

func TestRunShow_NoCacheFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := territory.RunShow(territory.ShowArgs{Dir: dir})
	g.Expect(err).To(HaveOccurred())
}

func TestRunShow_WithCacheFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a cache file
	m, err := territory.Generate(dir)
	g.Expect(err).ToNot(HaveOccurred())

	cached := territory.CachedMap{
		Map:       m,
		CachedAt:  time.Now(),
		FileCount: 5,
	}

	data, err := territory.MarshalCached(cached)
	g.Expect(err).ToNot(HaveOccurred())

	cacheDir := filepath.Join(dir, "context")
	g.Expect(os.MkdirAll(cacheDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(cacheDir, "territory.toml"), data, 0o644)).To(Succeed())

	err = territory.RunShow(territory.ShowArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestShow_NoCacheFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	_, err := territory.Show(dir)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("no cached territory map"))
}

func TestShow_WithCacheFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Generate a map and save it as cache
	m, err := territory.Generate(dir)
	g.Expect(err).ToNot(HaveOccurred())

	cached := territory.CachedMap{
		Map:       m,
		CachedAt:  time.Now(),
		FileCount: 1,
	}

	data, err := territory.MarshalCached(cached)
	g.Expect(err).ToNot(HaveOccurred())

	cacheDir := filepath.Join(dir, "context")
	g.Expect(os.MkdirAll(cacheDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(cacheDir, "territory.toml"), data, 0o644)).To(Succeed())

	result, err := territory.Show(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.FileCount).To(Equal(1))
}
