package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"github.com/toejough/engram/internal/cli"
)

// unexported constants.
const (
	lockFilePerm = 0o600
)

// unexported variables.
var (
	_ cli.EdgeFS     = osFS{}
	_ cli.FileLocker = flockLocker{}
)

// flockLocker is the production cli.FileLocker: opens path (O_CREATE|O_RDWR)
// and acquires an exclusive flock(2). unlock releases the lock and closes the
// handle. ADR-0013: advisory flock + atomic rename are the two safety
// primitives for concurrent vault writers (port of internal/cli's flockPath).
type flockLocker struct{}

// Lock acquires an exclusive flock on path, creating the file if absent.
func (flockLocker) Lock(path string) (func() error, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, lockFilePerm) //nolint:gosec // path from caller
	if err != nil {
		return nil, fmt.Errorf("open lock %s: %w", path, err)
	}

	fileDescriptor := int(f.Fd())

	flockErr := syscall.Flock(fileDescriptor, syscall.LOCK_EX)
	if flockErr != nil {
		_ = f.Close()

		return nil, fmt.Errorf("flock %s: %w", path, flockErr)
	}

	unlock := func() error {
		unlockErr := syscall.Flock(fileDescriptor, syscall.LOCK_UN)
		closeErr := f.Close()

		if unlockErr != nil {
			return fmt.Errorf("funlock %s: %w", path, unlockErr)
		}

		if closeErr != nil {
			return fmt.Errorf("close lock %s: %w", path, closeErr)
		}

		return nil
	}

	return unlock, nil
}

// osFS is the production cli.EdgeFS: thin wrappers over os.* with
// contextual error wrapping (%w preserves errors.Is chains such as
// fs.ErrNotExist). All production filesystem I/O lives here, not in
// internal/ (#700).
type osFS struct{}

// MkdirAll creates path with any missing parents; no-op when path exists.
func (osFS) MkdirAll(path string, perm fs.FileMode) error {
	err := os.MkdirAll(path, perm)
	if err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}

	return nil
}

// MkdirTemp creates a fresh unique directory in dir matching pattern.
func (osFS) MkdirTemp(dir, pattern string) (string, error) {
	made, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		return "", fmt.Errorf("mkdir temp in %s: %w", dir, err)
	}

	return made, nil
}

// ReadDir returns the directory entries of path.
func (osFS) ReadDir(path string) ([]fs.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", path, err)
	}

	return entries, nil
}

// ReadFile reads the file at path.
func (osFS) ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path from caller
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return data, nil
}

// Remove deletes the file or empty directory at path.
func (osFS) Remove(path string) error {
	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("remove %s: %w", path, err)
	}

	return nil
}

// RemoveAll deletes path and any children; no-op when path is absent.
func (osFS) RemoveAll(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("remove all %s: %w", path, err)
	}

	return nil
}

// Rename atomically renames oldPath to newPath (same-directory renames are
// atomic on POSIX — the ADR-0013 primitive).
func (osFS) Rename(oldPath, newPath string) error {
	err := os.Rename(oldPath, newPath)
	if err != nil {
		return fmt.Errorf("rename %s -> %s: %w", oldPath, newPath, err)
	}

	return nil
}

// Stat returns the fs.FileInfo for path.
func (osFS) Stat(path string) (fs.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	return info, nil
}

// WalkDir walks the file tree rooted at root, calling fn for each entry.
func (osFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	err := filepath.WalkDir(root, fn)
	if err != nil {
		return fmt.Errorf("walk %s: %w", root, err)
	}

	return nil
}

// WriteFile writes data to path with perm.
func (osFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	err := os.WriteFile(path, data, perm)
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

// WriteFileAtomic writes data to path atomically: it creates a unique temp
// file in filepath.Dir(path), sets perms, writes, closes, then renames into
// place. A same-directory rename is atomic on POSIX — a concurrent reader
// sees either the old or the new file, never a partial one. On any error the
// temp file is removed and the original (if any) is left untouched
// (ADR-0013; port of internal/cli's doAtomicWrite).
func (osFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	tmp, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return fmt.Errorf("atomic write %s: create temp: %w", path, err)
	}

	tmpName := tmp.Name()

	// Best-effort cleanup on any error path.
	success := false

	defer func() {
		if !success {
			_ = os.Remove(tmpName)
		}
	}()

	chmodErr := os.Chmod(tmpName, perm)
	if chmodErr != nil {
		_ = tmp.Close()

		return fmt.Errorf("atomic write %s: chmod temp: %w", path, chmodErr)
	}

	_, writeErr := tmp.Write(data)
	if writeErr != nil {
		_ = tmp.Close()

		return fmt.Errorf("atomic write %s: write temp: %w", path, writeErr)
	}

	closeErr := tmp.Close()
	if closeErr != nil {
		return fmt.Errorf("atomic write %s: close temp: %w", path, closeErr)
	}

	renameErr := os.Rename(tmpName, path)
	if renameErr != nil {
		return fmt.Errorf("atomic write %s: rename: %w", path, renameErr)
	}

	success = true

	return nil
}
