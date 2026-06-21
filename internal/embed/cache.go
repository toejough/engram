package embed

import (
	stdembed "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// unexported constants.
const (
	sentinel = ".complete"
)

// cacheFS is the I/O surface extractToCache depends on. Production wires
// productionCacheFS (thin wrappers around os.*); tests inject fakes to
// exercise every branch without touching the real disk.
type cacheFS interface {
	// StatSentinel reports whether the cache dir already has a .complete sentinel.
	StatSentinel(cacheDir string) (bool, error)
	// MkdirAll ensures the parent directory of the cache dir exists.
	MkdirAll(path string) error
	// MkdirTemp creates a temporary directory sibling of cacheDir for atomic extraction.
	MkdirTemp(parent, pattern string) (string, error)
	// WriteFile writes data to path (used to copy model files into the temp dir).
	WriteFile(path string, data []byte) error
	// WriteSentinel writes the .complete sentinel into tmpDir.
	WriteSentinel(tmpDir string) error
	// Rename renames src to dst atomically (os.Rename). Returns an error wrapping
	// os.ErrExist when dst already exists (concurrent-race scenario).
	Rename(src, dst string) error
	// RemoveAll deletes path recursively (used to clean up temp on rename race).
	RemoveAll(path string) error
}

// productionCacheFS is the canonical os.*-backed cacheFS. Every method is
// a thin passthrough — coverage is provided through integration via
// ExportExtractToCacheProduction.
type productionCacheFS struct{}

func (productionCacheFS) MkdirAll(path string) error {
	const perm = 0o755

	err := os.MkdirAll(path, perm)
	if err != nil {
		return fmt.Errorf("mkdir all: %w", err)
	}

	return nil
}

func (productionCacheFS) MkdirTemp(parent, pattern string) (string, error) {
	tmp, err := os.MkdirTemp(parent, pattern)
	if err != nil {
		return "", fmt.Errorf("mkdir temp: %w", err)
	}

	return tmp, nil
}

func (productionCacheFS) RemoveAll(path string) error {
	return os.RemoveAll(path) //nolint:wrapcheck // thin adapter; nil on missing paths
}

func (productionCacheFS) Rename(src, dst string) error {
	err := os.Rename(src, dst)
	if err != nil {
		// Wrap with os.ErrExist when the destination already exists so callers
		// can distinguish a lost race from a true rename failure.
		if isExistErr(err) {
			return fmt.Errorf("%w: %w", os.ErrExist, err)
		}

		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

func (productionCacheFS) StatSentinel(cacheDir string) (bool, error) {
	_, err := os.Stat(filepath.Join(cacheDir, sentinel))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("stat sentinel: %w", err)
	}

	return true, nil
}

func (productionCacheFS) WriteFile(path string, data []byte) error {
	const perm = 0o600

	err := os.WriteFile(path, data, perm)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func (productionCacheFS) WriteSentinel(tmpDir string) error {
	const perm = 0o600

	err := os.WriteFile(filepath.Join(tmpDir, sentinel), []byte{}, perm)
	if err != nil {
		return fmt.Errorf("write sentinel: %w", err)
	}

	return nil
}

// commitCache atomically renames tmp into cacheDir. If the rename fails with an
// existence error (concurrent-process race), it re-checks the sentinel: if the
// winner completed the cache, discards tmp and returns cacheDir. Otherwise returns
// the rename error.
func commitCache(cfs cacheFS, tmp, cacheDir string) (string, error) {
	renameErr := cfs.Rename(tmp, cacheDir)
	if renameErr == nil {
		return cacheDir, nil
	}

	// If the rename failed with an existence-style error, check whether another
	// process just won the race and completed the cache. If so, discard our temp.
	if isExistErr(renameErr) {
		complete, statErr := cfs.StatSentinel(cacheDir)
		if statErr == nil && complete {
			_ = cfs.RemoveAll(tmp)

			return cacheDir, nil
		}
	}

	// True rename failure (or sentinel absent after race check).
	_ = cfs.RemoveAll(tmp)

	return "", fmt.Errorf("cache rename: %w", renameErr)
}

// copyModelFiles copies every non-directory entry from modelFS/modelDir into tmpDir.
func copyModelFiles(cfs cacheFS, modelFS stdembed.FS, modelDir, tmpDir string) error {
	entries, _ := modelFS.ReadDir(modelDir) // already validated by caller

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, readErr := modelFS.ReadFile(filepath.Join(modelDir, entry.Name()))
		if readErr != nil {
			return fmt.Errorf("read embedded %s: %w", entry.Name(), readErr)
		}

		writeErr := cfs.WriteFile(filepath.Join(tmpDir, entry.Name()), data)
		if writeErr != nil {
			return fmt.Errorf("unpack %s: %w", entry.Name(), writeErr)
		}
	}

	return nil
}

// extractToCache ensures that <cacheDir> contains the fully extracted model
// and the .complete sentinel. On first call it extracts into a sibling temp
// dir and atomically renames it into place. On subsequent calls (sentinel
// present) it returns immediately without any I/O. A concurrent-process race
// (rename fails because another process just won) is handled by discarding
// the temp dir and using the pre-existing complete cache.
func extractToCache(
	cfs cacheFS,
	modelFS stdembed.FS,
	modelDir string,
	cacheDir string,
) (string, error) {
	// Fast path: already extracted.
	ok, statErr := cfs.StatSentinel(cacheDir)
	if statErr != nil {
		return "", statErr
	}

	if ok {
		return cacheDir, nil
	}

	return populateCache(cfs, modelFS, modelDir, cacheDir)
}

// isExistErr reports whether err (from os.Rename) is a "destination exists"
// error. On macOS/Linux, os.Rename replaces the destination atomically when
// both are dirs ONLY on the same filesystem; on macOS it returns ENOTEMPTY
// when the destination dir exists, which maps to syscall.ENOTEMPTY — handled
// by os.IsExist on some platforms. We also check the string for robustness.
func isExistErr(err error) bool {
	if errors.Is(err, os.ErrExist) {
		return true
	}
	// macOS returns "file exists" or "directory not empty" for rename-over-existing-dir.
	// unwrap to syscall layer and check.
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		if errors.Is(linkErr.Err, os.ErrExist) {
			return true
		}
		// ENOTEMPTY is returned on macOS when renaming over an existing dir.
		// We treat it the same as ErrExist for the cache-race case.
		errStr := linkErr.Err.Error()
		if errStr == "file exists" || errStr == "directory not empty" {
			return true
		}
	}

	return false
}

// populateCache handles the slow path of extractToCache: verifying the model
// FS, creating a sibling temp dir, copying model files, and atomically renaming
// the temp dir into place.
func populateCache(
	cfs cacheFS,
	modelFS stdembed.FS,
	modelDir string,
	cacheDir string,
) (string, error) {
	// Verify the model FS has files before creating any directories.
	entries, dirErr := modelFS.ReadDir(modelDir)
	if dirErr != nil || len(entries) == 0 {
		return "", fmt.Errorf("%w: dir %s (underlying: %w)",
			ErrBundledModelUnavailable, modelDir, dirErr,
		)
	}

	// Ensure the parent directory exists.
	parent := filepath.Dir(cacheDir)

	mkdirErr := cfs.MkdirAll(parent)
	if mkdirErr != nil {
		return "", fmt.Errorf("cache parent dir: %w", mkdirErr)
	}

	// Extract into a sibling temp dir so the rename is atomic.
	tmp, tmpErr := cfs.MkdirTemp(parent, ".tmp-engram-model-*")
	if tmpErr != nil {
		return "", fmt.Errorf("cache temp dir: %w", tmpErr)
	}

	copyErr := copyModelFiles(cfs, modelFS, modelDir, tmp)
	if copyErr != nil {
		_ = cfs.RemoveAll(tmp)

		return "", copyErr
	}

	sentinelErr := cfs.WriteSentinel(tmp)
	if sentinelErr != nil {
		_ = cfs.RemoveAll(tmp)

		return "", fmt.Errorf("cache sentinel: %w", sentinelErr)
	}

	return commitCache(cfs, tmp, cacheDir)
}
