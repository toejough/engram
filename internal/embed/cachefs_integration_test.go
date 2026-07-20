package embed_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

// TestCacheFS_RealOS_RenameOntoPopulatedDir keeps the exist-classification
// honest on the actual OS: on macOS the raw rename error is ENOTEMPTY, and
// the composed Rename must still satisfy the fs.ErrExist contract.
func TestCacheFS_RealOS_RenameOntoPopulatedDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	parent := t.TempDir()
	src := filepath.Join(parent, "src")
	dst := filepath.Join(parent, "dst")

	g.Expect(os.Mkdir(src, 0o755)).To(Succeed())
	g.Expect(os.Mkdir(dst, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dst, "f.txt"), []byte("hi"), 0o600)).To(Succeed())

	err := realCacheFSForTest().Rename(src, dst)
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, fs.ErrExist)).To(BeTrue(),
		"CacheFS.Rename contract: destination-exists must satisfy errors.Is(err, fs.ErrExist)")
}

// TestCacheFS_RealOS_SentinelRoundTrip proves sentinel write + probe
// against a real tempdir.
func TestCacheFS_RealOS_SentinelRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cfs := realCacheFSForTest()
	dir := t.TempDir()

	present, err := cfs.StatSentinel(dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(present).To(BeFalse())

	g.Expect(cfs.WriteSentinel(dir)).To(Succeed())

	_, statErr := os.Stat(filepath.Join(dir, ".complete"))
	g.Expect(statErr).NotTo(HaveOccurred())

	present, err = cfs.StatSentinel(dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(present).To(BeTrue())
}

// TestExtractToCache_RealOS drives the internal extraction through the
// composed CacheFS on real disk: first call extracts and stamps the
// sentinel; second call reuses without re-extracting. The injected backend
// fails so no hugot runtime is needed (extraction happens before the
// backend opens). nonEmptyTestFS is declared in cache_test.go (same
// embed_test package; its move from unpack_test.go happens in step 8).
func TestExtractToCache_RealOS(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cacheDir := filepath.Join(t.TempDir(), "models", "stub@1")

	_, err := embed.NewHugotEmbedderFromFS(
		t.Context(), failingBackend{}, realCacheFSForTest(),
		nonEmptyTestFS, "testdata", "stub@1", cacheDir)
	g.Expect(err).To(MatchError(errBackendUnused))

	_, sentinelErr := os.Stat(filepath.Join(cacheDir, ".complete"))
	g.Expect(sentinelErr).NotTo(HaveOccurred(),
		".complete sentinel must be written after first extraction")

	entries1, readErr1 := os.ReadDir(cacheDir)
	g.Expect(readErr1).NotTo(HaveOccurred())

	fileCount1 := len(entries1)
	g.Expect(fileCount1).To(BeNumerically(">", 1), "cache dir must contain model files + sentinel")

	_, err = embed.NewHugotEmbedderFromFS(
		t.Context(), failingBackend{}, realCacheFSForTest(),
		nonEmptyTestFS, "testdata", "stub@1", cacheDir)
	g.Expect(err).To(MatchError(errBackendUnused))

	entries2, readErr2 := os.ReadDir(cacheDir)
	g.Expect(readErr2).NotTo(HaveOccurred())
	g.Expect(entries2).To(HaveLen(fileCount1),
		"second call must not add/modify files — cache reused as-is")
}

// unexported variables.
var (
	errBackendUnused = errors.New("backend intentionally failing")
)

// failingBackend implements embed.Backend and always refuses to open.
type failingBackend struct{}

func (failingBackend) OpenPipeline(context.Context, string) (embed.PipelineHandle, error) {
	return nil, errBackendUnused
}

// realCacheFSForTest builds the production CacheFS composition over the
// raw os functions — the same wiring cli.NewDeps performs from
// cli.Primitives.
func realCacheFSForTest() embed.CacheFS {
	return embed.NewCacheFS(embed.CacheFSPrims{
		Stat:      os.Stat,
		MkdirAll:  os.MkdirAll,
		MkdirTemp: os.MkdirTemp,
		WriteFile: os.WriteFile,
		Rename:    os.Rename,
		RemoveAll: os.RemoveAll,
	})
}
