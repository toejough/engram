package embed_test

import (
	stdembed "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

// TestClose_DoesNotDeleteCacheDir verifies that HugotEmbedder.Close does
// NOT delete any directory. The model cache is a shared persistent cache.
func TestClose_DoesNotDeleteCacheDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	closeCalled := false
	embedder := embed.NewHugotEmbedderWithPipelineForTest(
		"test@1", 4,
		func(_ string) ([][]float32, error) { return nil, nil },
		func() error { closeCalled = true; return nil },
	)

	// SetCacheDirForTest is intentionally a no-op because HugotEmbedder
	// no longer holds a tmpDir field; Close simply closes the Hugot session.
	embed.SetCacheDirForTest(embedder, dir)

	closeErr := embedder.Close()
	g.Expect(closeErr).NotTo(HaveOccurred())
	g.Expect(closeCalled).To(BeTrue(), "Hugot session must be closed")

	// The externally created dir must be untouched by Close.
	_, statErr := os.Stat(dir)
	g.Expect(statErr).NotTo(HaveOccurred(),
		"Close must NOT delete the model cache dir")
}

// TestExtractToCache_AtomicRenameRace verifies that when the cache dir is
// populated by a concurrent process between our sentinel check and our rename
// (rename fails with an exist-style error AND the sentinel is now present),
// extractToCache discards the temp and uses the complete pre-existing cache.
func TestExtractToCache_AtomicRenameRace(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// renameErr satisfies the CacheFS.Rename contract for a lost race: the
	// composed production Rename translates macOS ENOTEMPTY into fs.ErrExist.
	// After the rename fails, the sentinel IS present (concurrent winner wrote it).
	cfs := &fakeCacheFS{
		renameErr:           fmt.Errorf("%w: directory not empty", fs.ErrExist),
		sentinelAfterRename: true,
	}
	cacheDir := "/cache/engram/models/minilm-l6-v2@384"

	resultDir, err := embed.ExportExtractToCache(cfs, nonEmptyTestFS, "testdata", cacheDir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resultDir).To(Equal(cacheDir))
	g.Expect(cfs.removeCalls).To(BeNumerically(">", 0),
		"must discard the temp dir that lost the race")
}

// TestExtractToCache_FirstInitExtractsToCache verifies that when the cache
// dir does not exist, extractToCache unpacks the model into it and writes
// the .complete sentinel, then returns the cache dir path.
func TestExtractToCache_FirstInitExtractsToCache(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cfs := &fakeCacheFS{}
	cacheDir := "/cache/engram/models/minilm-l6-v2@384"

	resultDir, err := embed.ExportExtractToCache(cfs, nonEmptyTestFS, "testdata", cacheDir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resultDir).To(Equal(cacheDir))
	g.Expect(cfs.mkdirTempCalls).To(BeNumerically(">", 0), "must create sibling temp dir")
	g.Expect(cfs.sentinelWritten).To(BeTrue(), "must write .complete sentinel after extraction")
	g.Expect(cfs.writeFileCalls).To(BeNumerically(">", 0), "must write model files")
}

// TestExtractToCache_MkdirTempFailurePropagates verifies the MkdirTemp error path
// in populateCache: failure must return a wrapped error without leaving temp dirs.
func TestExtractToCache_MkdirTempFailurePropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cfs := &fakeCacheFS{mkdirTempErr: errors.New("no space")}
	cacheDir := "/cache/engram/models/minilm-l6-v2@384"

	_, err := embed.ExportExtractToCache(cfs, nonEmptyTestFS, "testdata", cacheDir)
	g.Expect(err).To(MatchError(ContainSubstring("cache temp dir")))
	g.Expect(err).To(MatchError(ContainSubstring("no space")))
}

// TestExtractToCache_RaceWithoutSentinelFails verifies that if the rename fails
// with an exist-style error but the cache dir has no sentinel, the error propagates.
func TestExtractToCache_RaceWithoutSentinelFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// renameErr looks like an exist-err, but sentinelAfterRename is false
	// (the pre-existing dir is not yet complete).
	cfs := &fakeCacheFS{
		renameErr: fmt.Errorf("%w: directory not empty", fs.ErrExist),
	}
	cacheDir := "/cache/engram/models/minilm-l6-v2@384"

	_, err := embed.ExportExtractToCache(cfs, nonEmptyTestFS, "testdata", cacheDir)
	g.Expect(err).To(HaveOccurred())
	g.Expect(cfs.removeCalls).To(BeNumerically(">", 0), "temp dir must be cleaned up")
}

// TestExtractToCache_SecondInitReusesExistingCache verifies that when the
// cache dir already has a .complete sentinel, extractToCache returns immediately
// without re-extracting: zero mkdir/write calls.
func TestExtractToCache_SecondInitReusesExistingCache(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cfs := &fakeCacheFS{sentinelExists: true}
	cacheDir := "/cache/engram/models/minilm-l6-v2@384"

	resultDir, err := embed.ExportExtractToCache(cfs, nonEmptyTestFS, "testdata", cacheDir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resultDir).To(Equal(cacheDir))
	// Fast path: no extraction work performed.
	g.Expect(cfs.mkdirTempCalls).To(Equal(0), "must NOT create temp dir when cache is complete")
	g.Expect(cfs.writeFileCalls).To(Equal(0), "must NOT write files when cache is complete")
	g.Expect(cfs.sentinelWritten).To(BeFalse(), "must NOT write sentinel when cache is complete")
}

// TestExtractToCache_SentinelWriteFailureCleansUpTemp verifies that a WriteSentinel
// error triggers RemoveAll on the temp dir.
func TestExtractToCache_SentinelWriteFailureCleansUpTemp(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cfs := &fakeCacheFS{sentinelWriteErr: errors.New("sentinel write failed")}
	cacheDir := "/cache/engram/models/minilm-l6-v2@384"

	_, err := embed.ExportExtractToCache(cfs, nonEmptyTestFS, "testdata", cacheDir)
	g.Expect(err).To(MatchError(ContainSubstring("cache sentinel")))
	g.Expect(cfs.removeCalls).To(BeNumerically(">", 0), "temp dir must be cleaned up on sentinel failure")
}

// TestExtractToCache_TrueRenameFailurePropagates verifies that a non-existence rename
// error (true failure) propagates and the temp dir is cleaned up.
func TestExtractToCache_TrueRenameFailurePropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cfs := &fakeCacheFS{renameErr: errors.New("rename: permission denied")}
	cacheDir := "/cache/engram/models/minilm-l6-v2@384"

	_, err := embed.ExportExtractToCache(cfs, nonEmptyTestFS, "testdata", cacheDir)
	g.Expect(err).To(MatchError(ContainSubstring("cache rename")))
	g.Expect(cfs.removeCalls).To(BeNumerically(">", 0), "temp dir must be cleaned up on rename failure")
}

// TestExtractToCache_WriteFileFailureCleansUpTemp verifies that a WriteFile error
// while copying model files triggers RemoveAll on the temp dir.
func TestExtractToCache_WriteFileFailureCleansUpTemp(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cfs := &fakeCacheFS{writeErr: errors.New("disk full")}
	cacheDir := "/cache/engram/models/minilm-l6-v2@384"

	_, err := embed.ExportExtractToCache(cfs, nonEmptyTestFS, "testdata", cacheDir)
	g.Expect(err).To(HaveOccurred())
	g.Expect(cfs.removeCalls).To(BeNumerically(">", 0), "temp dir must be cleaned up on write failure")
}

//go:embed testdata/gen-reference.py
var nonEmptyTestFS stdembed.FS

// unexported variables.
var (
	_ embed.CacheFS = (*fakeCacheFS)(nil)
)

// fakeCacheFS records cacheFS method calls for assertion in unit tests.
type fakeCacheFS struct {
	// Configuration fields.
	sentinelExists      bool  // StatSentinel returns true on first call
	sentinelAfterRename bool  // StatSentinel returns true after first Rename
	renameErr           error // Rename returns this error
	removeErr           error
	writeErr            error // WriteFile returns this error
	mkdirTempErr        error // MkdirTemp returns this error
	sentinelWriteErr    error // WriteSentinel returns this error

	// Observation fields.
	mkdirTempCalls  int
	writeFileCalls  int
	removeCalls     int
	sentinelWritten bool
	renameCalls     int
}

func (f *fakeCacheFS) MkdirAll(_ string) error {
	return nil
}

func (f *fakeCacheFS) MkdirTemp(parent, _ string) (string, error) {
	f.mkdirTempCalls++

	if f.mkdirTempErr != nil {
		return "", f.mkdirTempErr
	}

	return filepath.Join(parent, ".tmp-fake"), nil
}

func (f *fakeCacheFS) RemoveAll(_ string) error {
	f.removeCalls++

	return f.removeErr
}

func (f *fakeCacheFS) Rename(_, _ string) error {
	f.renameCalls++

	return f.renameErr
}

func (f *fakeCacheFS) StatSentinel(_ string) (bool, error) {
	if f.renameCalls > 0 && f.sentinelAfterRename {
		return true, nil
	}

	return f.sentinelExists, nil
}

func (f *fakeCacheFS) WriteFile(_ string, _ []byte) error {
	f.writeFileCalls++

	return f.writeErr
}

func (f *fakeCacheFS) WriteSentinel(_ string) error {
	f.sentinelWritten = true

	return f.sentinelWriteErr
}
