package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
)

// unexported constants.
const (
	notePerm    fs.FileMode = 0o600
	sidecarPerm fs.FileMode = 0o600
)

// edgeVaultInitFS adapts EdgeFS to the VaultInitFS bootstrap surface.
// WriteFileIfMissing swallows fs.ErrExist so re-initialization is idempotent
// and never clobbers user-edited starter files.
type edgeVaultInitFS struct{ fsys EdgeFS }

func (e edgeVaultInitFS) MkdirAll(path string, perm fs.FileMode) error {
	err := e.fsys.MkdirAll(path, perm)
	if err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	return nil
}

func (e edgeVaultInitFS) WriteFileIfMissing(path string, data []byte, perm fs.FileMode) error {
	err := e.fsys.WriteFileExcl(path, data, perm)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return nil
		}

		return fmt.Errorf("write if missing: %w", err)
	}

	return nil
}

// initVaultFromFS returns an InitVault func composed over the injected EdgeFS.
func initVaultFromFS(fsys EdgeFS) func(string) error {
	return func(path string) error { return initializeVault(edgeVaultInitFS{fsys: fsys}, path) }
}

// listBasenamesFromFS returns a ListBasenames func: note basenames (filename
// minus .md) for luhmann-id notes at the flat vault root (D1).
func listBasenamesFromFS(fsys EdgeFS) func(string) ([]string, error) {
	return func(vault string) ([]string, error) {
		return listRootNotes(fsys.ReadDir, vault, func(name string) (string, bool) {
			if _, ok := extractLuhmannFromFilename(name); !ok {
				return "", false
			}

			return strings.TrimSuffix(name, ".md"), true
		})
	}
}

// listIDsFromFS returns a ListIDs func: Luhmann IDs from .md filenames at the
// flat vault root.
func listIDsFromFS(fsys EdgeFS) func(string) ([]string, error) {
	return func(vault string) ([]string, error) {
		return listRootNotes(fsys.ReadDir, vault, extractLuhmannFromFilename)
	}
}

// listMDFromFS returns a ListMD func with osVaultFS.ListMD semantics: the .md
// filenames directly inside dir; a missing dir yields (nil, nil).
func listMDFromFS(fsys EdgeFS) func(string) ([]string, error) {
	return func(dir string) ([]string, error) {
		entries, err := fsys.ReadDir(dir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, nil
			}

			return nil, fmt.Errorf("reading dir %s: %w", dir, err)
		}

		out := make([]string, 0, len(entries))

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			out = append(out, entry.Name())
		}

		return out, nil
	}
}

// logWarningTo returns the production LogWarning hook writing to w — the
// Deps-threaded replacement for the old os.Stderr-bound logWarningToStderrf.
func logWarningTo(w io.Writer) func(string, ...any) {
	return func(format string, args ...any) {
		_, _ = fmt.Fprintf(w, "warning: "+format+"\n", args...)
	}
}

// statDirFromFS returns a StatDir func: fs.ErrNotExist when the directory is
// missing, errNotADirectory when the path is a file, wrapped error otherwise.
func statDirFromFS(fsys EdgeFS) func(string) error {
	return func(path string) error {
		info, err := fsys.Stat(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return fs.ErrNotExist
			}

			return fmt.Errorf("stat: %w", err)
		}

		if !info.IsDir() {
			return fmt.Errorf("%w: %s", errNotADirectory, path)
		}

		return nil
	}
}

// vaultLockFromLocker returns a vault-lock func over the injected FileLocker:
// an exclusive flock on vault/.luhmann.lock (ADR-0013). The locker's
// unlock-with-error is adapted to the deps structs' plain release func.
func vaultLockFromLocker(locker FileLocker) func(string) (func(), error) {
	return func(vault string) (func(), error) {
		unlock, err := locker.Lock(filepath.Join(vault, luhmannLockFile))
		if err != nil {
			return nil, fmt.Errorf("lock vault: %w", err)
		}

		return func() { _ = unlock() }, nil
	}
}

// writeNewFromFS returns a WriteNew func: exclusive create, preserving
// errors.Is(err, fs.ErrExist) — the K1 collision backstop under the vault lock.
func writeNewFromFS(fsys EdgeFS) func(string, []byte) error {
	return func(path string, data []byte) error {
		err := fsys.WriteFileExcl(path, data, notePerm)
		if err != nil {
			return fmt.Errorf("write new: %w", err)
		}

		return nil
	}
}

// writeNoteAtomicFromFS returns an atomic note-rewrite func at the given perm
// (temp+rename via EdgeFS.WriteFileAtomic — ADR-0013's atomic-rename edge).
func writeNoteAtomicFromFS(fsys EdgeFS, perm fs.FileMode) func(string, []byte) error {
	return func(path string, data []byte) error {
		err := fsys.WriteFileAtomic(path, data, perm)
		if err != nil {
			return fmt.Errorf("write note: %w", err)
		}

		return nil
	}
}

// writeSidecarFromFS returns a WriteSidecar func: atomic .vec.json write.
func writeSidecarFromFS(fsys EdgeFS) func(string, []byte) error {
	return func(path string, data []byte) error {
		err := fsys.WriteFileAtomic(path, data, sidecarPerm)
		if err != nil {
			return fmt.Errorf("write sidecar: %w", err)
		}

		return nil
	}
}
