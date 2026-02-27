package territory_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/territory"
)

// TestLoadCached_CorruptedToml ensures corrupted cache falls through to regeneration.
func TestLoadCached_CorruptedToml(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(dir, "context"), 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "context", "territory.toml"), []byte("not valid toml :::"), 0o644)).To(Succeed())

	result, hit, err := territory.LoadCached(dir, time.Now)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(hit).To(BeFalse())
	g.Expect(result.Structure.Root).To(Equal(dir))
}

// TestLoadCached_FileCountPositive_CacheHit exercises abs(n >= 0).
// Cache has FileCount=1, dir has exactly 1 file → abs(0)=0 ≤ threshold → cache hit.
func TestLoadCached_FileCountPositive_CacheHit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(dir, "context"), 0o755)).To(Succeed())

	cached := territory.CachedMap{
		Map:       territory.Map{Structure: territory.Structure{Root: "cached-root"}},
		CachedAt:  time.Now(),
		FileCount: 1,
	}

	data, err := territory.MarshalCached(cached)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(os.WriteFile(filepath.Join(dir, "context", "territory.toml"), data, 0o644)).To(Succeed())

	// currentCount=1 (the cache file), FileCount=1 → diff=abs(0)=0 ≤ threshold=1 → hit
	result, hit, err := territory.LoadCached(dir, time.Now)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(hit).To(BeTrue())
	g.Expect(result.Structure.Root).To(Equal("cached-root"))
}

// TestLoadCached_FileCountPositive_Regenerates exercises abs(n < 0).
// Cache has FileCount=5, dir has 1 file → abs(-4)=4 > threshold=1 → regenerate.
func TestLoadCached_FileCountPositive_Regenerates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(dir, "context"), 0o755)).To(Succeed())

	cached := territory.CachedMap{
		Map:       territory.Map{Structure: territory.Structure{Root: "cached-root"}},
		CachedAt:  time.Now(),
		FileCount: 5,
	}

	data, err := territory.MarshalCached(cached)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(os.WriteFile(filepath.Join(dir, "context", "territory.toml"), data, 0o644)).To(Succeed())

	// currentCount=1 (the cache file), FileCount=5 → diff=abs(1-5)=4 > threshold=1 → regenerate
	result, hit, err := territory.LoadCached(dir, time.Now)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(hit).To(BeFalse())
	g.Expect(result.Structure.Root).To(Equal(dir))
}
