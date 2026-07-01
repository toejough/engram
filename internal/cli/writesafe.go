package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// atomicWriteFile writes data to path atomically: it creates a unique temp
// file in filepath.Dir(path), sets perms, writes, closes, then renames into
// place. A same-directory rename is atomic on POSIX — a concurrent reader
// sees either the old or the new file, never a partial one. On any error the
// temp file is removed and the original (if any) is left untouched.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	return doAtomicWrite(path, data, perm, os.Rename)
}

// doAtomicWrite is the testable core of atomicWriteFile. The rename parameter
// is injected so tests can trigger the rename-failure path and verify that the
// defer cleanup removes the temp file and the original is left untouched.
func doAtomicWrite(
	path string,
	data []byte,
	perm os.FileMode,
	rename func(oldpath, newpath string) error,
) error {
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

	renameErr := rename(tmpName, path)
	if renameErr != nil {
		return fmt.Errorf("atomic write %s: rename: %w", path, renameErr)
	}

	success = true

	return nil
}
