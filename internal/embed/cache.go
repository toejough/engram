package embed

import (
	stdembed "embed"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

// CacheFS is the I/O surface extractToCache depends on. The production
// implementation is composed internally by NewCacheFS (cachefs.go) over
// raw filesystem primitives; tests inject fakes to exercise every branch
// without touching the real disk.
type CacheFS interface {
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
	// Rename renames src to dst atomically. When dst already exists
	// (concurrent-race scenario), the returned error MUST satisfy
	// errors.Is(err, fs.ErrExist) — implementations translate platform
	// quirks (e.g. macOS ENOTEMPTY on dir-over-dir renames) before returning.
	Rename(src, dst string) error
	// RemoveAll deletes path recursively (used to clean up temp on rename race).
	RemoveAll(path string) error
}

// commitCache atomically renames tmp into cacheDir. If the rename fails with
// a destination-exists error (concurrent-process race), it re-checks the
// sentinel: if the winner completed the cache, discards tmp and returns
// cacheDir. Otherwise returns the rename error.
func commitCache(cfs CacheFS, tmp, cacheDir string) (string, error) {
	renameErr := cfs.Rename(tmp, cacheDir)
	if renameErr == nil {
		return cacheDir, nil
	}

	// If the rename failed because the destination exists, check whether
	// another process just won the race and completed the cache. If so,
	// discard our temp.
	if errors.Is(renameErr, fs.ErrExist) {
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
func copyModelFiles(cfs CacheFS, modelFS stdembed.FS, modelDir, tmpDir string) error {
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
	cfs CacheFS,
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

// populateCache handles the slow path of extractToCache: verifying the model
// FS, creating a sibling temp dir, copying model files, and atomically renaming
// the temp dir into place.
func populateCache(
	cfs CacheFS,
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
