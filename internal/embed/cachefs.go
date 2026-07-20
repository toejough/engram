package embed

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// CacheFSPrims carries the raw filesystem capabilities the production
// CacheFS composition needs. Field signatures are identical to the
// matching cli.Primitives fields — cli.NewDeps forwards its Primitives
// fields into this struct verbatim (doctrine flag E-2: no new cache
// fields on cli.Primitives). The funcs return RAW os errors; all wrapping
// and exist-classification happen here, internally.
type CacheFSPrims struct {
	Stat      func(path string) (fs.FileInfo, error)
	MkdirAll  func(path string, perm fs.FileMode) error
	MkdirTemp func(dir, pattern string) (string, error)
	WriteFile func(path string, data []byte, perm fs.FileMode) error
	Rename    func(oldPath, newPath string) error
	RemoveAll func(path string) error
}

// NewCacheFS composes the production CacheFS from raw filesystem
// primitives: sentinel policy, permission policy, error wrapping, and the
// fs.ErrExist rename contract all live here (#700).
func NewCacheFS(prims CacheFSPrims) CacheFS {
	return primCacheFS{prims: prims}
}

// unexported constants.
const (
	cacheDirPerm  fs.FileMode = 0o755
	cacheFilePerm fs.FileMode = 0o600
	// sentinelFileName marks a fully extracted model cache dir.
	sentinelFileName = ".complete"
)

// primCacheFS is the CacheFS composition over raw primitives.
type primCacheFS struct {
	prims CacheFSPrims
}

// MkdirAll ensures the parent directory of the cache dir exists.
func (c primCacheFS) MkdirAll(path string) error {
	err := c.prims.MkdirAll(path, cacheDirPerm)
	if err != nil {
		return fmt.Errorf("mkdir all: %w", err)
	}

	return nil
}

// MkdirTemp creates a temporary sibling dir for atomic extraction.
func (c primCacheFS) MkdirTemp(parent, pattern string) (string, error) {
	tmp, err := c.prims.MkdirTemp(parent, pattern)
	if err != nil {
		return "", fmt.Errorf("mkdir temp: %w", err)
	}

	return tmp, nil
}

// RemoveAll deletes path. The raw primitive's contract (os.RemoveAll: nil
// on missing paths) is already caller-friendly; the error passes through
// unwrapped.
func (c primCacheFS) RemoveAll(path string) error {
	return c.prims.RemoveAll(path)
}

// Rename renames src to dst atomically. When the destination already
// exists (including macOS ENOTEMPTY for dir-over-dir renames), the
// returned error satisfies errors.Is(err, fs.ErrExist) per the CacheFS
// contract.
func (c primCacheFS) Rename(src, dst string) error {
	err := c.prims.Rename(src, dst)
	if err != nil {
		if renameIsExist(err) {
			return fmt.Errorf("%w: %w", fs.ErrExist, err)
		}

		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

// StatSentinel reports whether cacheDir already has a .complete sentinel.
func (c primCacheFS) StatSentinel(cacheDir string) (bool, error) {
	_, err := c.prims.Stat(filepath.Join(cacheDir, sentinelFileName))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("stat sentinel: %w", err)
	}

	return true, nil
}

// WriteFile writes data to path (copies model files into the temp dir).
func (c primCacheFS) WriteFile(path string, data []byte) error {
	err := c.prims.WriteFile(path, data, cacheFilePerm)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// WriteSentinel writes the .complete sentinel into tmpDir.
func (c primCacheFS) WriteSentinel(tmpDir string) error {
	err := c.prims.WriteFile(filepath.Join(tmpDir, sentinelFileName), []byte{}, cacheFilePerm)
	if err != nil {
		return fmt.Errorf("write sentinel: %w", err)
	}

	return nil
}

// renameIsExist reports whether err (raw from the rename primitive) is a
// destination-exists error. errors.Is(err, fs.ErrExist) covers EEXIST
// and — via syscall.Errno's Is mapping — ENOTEMPTY through
// *os.LinkError's Unwrap; the string fallback preserves the previous
// defensive platform sniffing for chains that don't unwrap to a mapped
// errno, without importing os (doctrine flag E-3).
func renameIsExist(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, fs.ErrExist) {
		return true
	}

	message := err.Error()

	return strings.Contains(message, "file exists") ||
		strings.Contains(message, "directory not empty")
}
