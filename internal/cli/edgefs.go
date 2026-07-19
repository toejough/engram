package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

// unexported constants.
const (
	// maxTempAttempts bounds the fs.ErrExist retry when deriving a unique
	// temp name for the atomic-write dance (doctrine flag P-4).
	maxTempAttempts = 10
)

// unexported variables.
var (
	_                    EdgeFS = primFS{}
	errTempNameExhausted        = errors.New("no unique temp name available")
)

// primFS is the production EdgeFS: it composes the injected raw primitives
// with contextual error wrapping (%w preserves errors.Is chains such as
// fs.ErrNotExist) and the ADR-0013 atomic-write dance. All orchestration
// lives here in internal/; cmd/engram contributes only raw os/filepath
// references (#700).
type primFS struct {
	prims Primitives
}

// MkdirAll creates path with any missing parents; no-op when path exists.
func (f primFS) MkdirAll(path string, perm fs.FileMode) error {
	err := f.prims.MkdirAll(path, perm)
	if err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}

	return nil
}

// MkdirTemp creates a fresh unique directory in dir matching pattern.
func (f primFS) MkdirTemp(dir, pattern string) (string, error) {
	made, err := f.prims.MkdirTemp(dir, pattern)
	if err != nil {
		return "", fmt.Errorf("mkdir temp in %s: %w", dir, err)
	}

	return made, nil
}

// ReadDir returns the directory entries of path.
func (f primFS) ReadDir(path string) ([]fs.DirEntry, error) {
	entries, err := f.prims.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", path, err)
	}

	return entries, nil
}

// ReadFile reads the file at path.
func (f primFS) ReadFile(path string) ([]byte, error) {
	data, err := f.prims.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return data, nil
}

// Remove deletes the file or empty directory at path.
func (f primFS) Remove(path string) error {
	err := f.prims.Remove(path)
	if err != nil {
		return fmt.Errorf("remove %s: %w", path, err)
	}

	return nil
}

// RemoveAll deletes path and any children; no-op when path is absent.
func (f primFS) RemoveAll(path string) error {
	err := f.prims.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("remove all %s: %w", path, err)
	}

	return nil
}

// Rename atomically renames oldPath to newPath (same-directory renames are
// atomic on POSIX — the ADR-0013 primitive).
func (f primFS) Rename(oldPath, newPath string) error {
	err := f.prims.Rename(oldPath, newPath)
	if err != nil {
		return fmt.Errorf("rename %s -> %s: %w", oldPath, newPath, err)
	}

	return nil
}

// Stat returns the fs.FileInfo for path.
func (f primFS) Stat(path string) (fs.FileInfo, error) {
	info, err := f.prims.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	return info, nil
}

// WalkDir walks the file tree rooted at root, calling fn for each entry.
func (f primFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	err := f.prims.WalkDir(root, fn)
	if err != nil {
		return fmt.Errorf("walk %s: %w", root, err)
	}

	return nil
}

// WriteFile writes data to path with perm.
func (f primFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	err := f.prims.WriteFile(path, data, perm)
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

// WriteFileAtomic writes data to path atomically: it derives a unique temp
// name in filepath.Dir(path) from the target base + the injected clock's
// nanos + an attempt counter, creates it exclusively (data written at perm)
// via the WriteFileExcl primitive — retrying fresh candidates on
// fs.ErrExist, bounded by maxTempAttempts — then chmods the temp to the
// exact target perm (umask-independent) and renames into place. A
// same-directory rename is atomic on POSIX — a concurrent reader sees
// either the old or the new file, never a partial one. On any failure
// after creation the temp file is removed and the original (if any) is
// left untouched (ADR-0013; design flag P-4: the unique-temp-name policy
// is INTERNAL — cmd contributes only the stdlib-equivalent WriteFileExcl
// primitive, doctrine survivor S-1, plus the restored direct Chmod
// primitive for umask-independent perms).
func (f primFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	tmpName, err := f.createUniqueTemp(path, data, perm)
	if err != nil {
		return fmt.Errorf("atomic write %s: create temp: %w", path, err)
	}

	// chmod after write (temp is never wider than final); explicit chmod
	// keeps atomic-write perms umask-independent — parity with the
	// pre-#700 dance. Do NOT reorder chmod before the data write.
	chmodErr := f.prims.Chmod(tmpName, perm)
	if chmodErr != nil {
		_ = f.prims.Remove(tmpName)

		return fmt.Errorf("atomic write %s: chmod temp: %w", path, chmodErr)
	}

	renameErr := f.prims.Rename(tmpName, path)
	if renameErr != nil {
		// Cleanup on any failure after creation (P-4).
		_ = f.prims.Remove(tmpName)

		return fmt.Errorf("atomic write %s: rename: %w", path, renameErr)
	}

	return nil
}

// createUniqueTemp writes data exclusively to a fresh candidate temp name
// beside path (".<base>.tmp-<nanos>-<attempt>"). A candidate that already
// exists (fs.ErrExist) is retried with the next attempt counter, bounded
// by maxTempAttempts; any other error aborts immediately — nothing was
// created, so there is nothing to clean.
func (f primFS) createUniqueTemp(path string, data []byte, perm fs.FileMode) (string, error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	nanos := f.prims.Now().UnixNano()

	var lastErr error

	for attempt := range maxTempAttempts {
		candidate := filepath.Join(dir, fmt.Sprintf(".%s.tmp-%d-%d", base, nanos, attempt))

		lastErr = f.prims.WriteFileExcl(candidate, data, perm)
		if lastErr == nil {
			return candidate, nil
		}

		if !errors.Is(lastErr, fs.ErrExist) {
			return "", lastErr
		}
	}

	return "", fmt.Errorf("%w after %d attempts: %w", errTempNameExhausted, maxTempAttempts, lastErr)
}
