// Package cli implements the engram command-line interface (ARCH-6).
package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/toejough/engram/internal/transcript"
)

// unexported constants.
const (
	luhmannLockFile = ".luhmann.lock"
)

// unexported variables.
var (
	errNotADirectory = errors.New("not a directory")
)

// osDirLister lists .jsonl files in a directory using os.ReadDir.
type osDirLister struct{}

func (l *osDirLister) ListJSONL(dir string) ([]transcript.FileEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("listing directory: %w", err)
	}

	results := make([]transcript.FileEntry, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".jsonl") {
			continue
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}

		results = append(results, transcript.FileEntry{
			Path:  filepath.Join(dir, name),
			Mtime: info.ModTime(),
		})
	}

	return results, nil
}

// I/O adapters for context package DI interfaces.

type osFileReader struct{}

func (r *osFileReader) Read(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:gosec,wrapcheck // thin I/O adapter
}

// osLearnFS is the production filesystem adapter for the learn subcommand.
type osLearnFS struct{}

// ListBasenames returns note basenames (filename minus .md) for luhmann-id
// notes at the vault root (flat layout) — used to resolve a relation's bare
// Luhmann id to its full basename (D1).
func (*osLearnFS) ListBasenames(vault string) ([]string, error) {
	return listRootNotes(vault, func(name string) (string, bool) {
		if _, ok := extractLuhmannFromFilename(name); !ok {
			return "", false
		}

		return strings.TrimSuffix(name, ".md"), true
	})
}

// ListIDs returns Luhmann IDs from .md filenames at the vault root (flat layout).
func (*osLearnFS) ListIDs(vault string) ([]string, error) {
	return listRootNotes(vault, extractLuhmannFromFilename)
}

// Lock acquires an exclusive flock on vault/.luhmann.lock; returns a release func.
func (*osLearnFS) Lock(vault string) (func(), error) {
	path := filepath.Join(vault, luhmannLockFile)

	const perm = 0o600

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, perm) //nolint:gosec // path from caller
	if err != nil {
		return nil, fmt.Errorf("open lock: %w", err)
	}

	fileDescriptor := int(f.Fd())

	flockErr := syscall.Flock(fileDescriptor, syscall.LOCK_EX)
	if flockErr != nil {
		_ = f.Close()

		return nil, fmt.Errorf("flock: %w", flockErr)
	}

	release := func() {
		_ = syscall.Flock(fileDescriptor, syscall.LOCK_UN)
		_ = f.Close()
	}

	return release, nil
}

// MkdirAll creates path with any missing parents; no-op when path exists.
func (*osLearnFS) MkdirAll(path string, perm fs.FileMode) error {
	err := os.MkdirAll(path, perm)
	if err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	return nil
}

// StatDir returns fs.ErrNotExist if the directory is missing, errNotADirectory
// if the path exists but is a file, or a wrapped error otherwise.
func (*osLearnFS) StatDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fs.ErrNotExist
		}

		return fmt.Errorf("stat: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%w: %s", errNotADirectory, path)
	}

	return nil
}

// WriteFileIfMissing writes data with O_EXCL so existing files are left
// untouched; ErrExist is swallowed so initializeVault is idempotent.
func (*osLearnFS) WriteFileIfMissing(path string, data []byte, perm fs.FileMode) error {
	f, err := os.OpenFile( //nolint:gosec // path from caller
		path,
		os.O_CREATE|os.O_EXCL|os.O_WRONLY,
		perm,
	)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return nil
		}

		return fmt.Errorf("open: %w", err)
	}

	defer func() { _ = f.Close() }()

	_, writeErr := f.Write(data)
	if writeErr != nil {
		return fmt.Errorf("write: %w", writeErr)
	}

	return nil
}

// WriteNew creates the file with O_EXCL — errors with fs.ErrExist if it already exists.
func (*osLearnFS) WriteNew(path string, data []byte) error {
	const perm = 0o600

	f, err := os.OpenFile( //nolint:gosec // path from caller
		path,
		os.O_CREATE|os.O_EXCL|os.O_WRONLY,
		perm,
	)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}

	defer func() { _ = f.Close() }()

	_, writeErr := f.Write(data)
	if writeErr != nil {
		return fmt.Errorf("write: %w", writeErr)
	}

	return nil
}

// WriteSidecar writes a .vec.json sidecar to path with 0o600 perms. Used
// by autoEmbedNote after a successful note write; lives on osLearnFS so
// the production wiring uses a named method (visible to coverage) instead
// of an anonymous closure.
func (*osLearnFS) WriteSidecar(path string, data []byte) error {
	const perm = 0o600

	err := os.WriteFile(path, data, perm)
	if err != nil {
		return fmt.Errorf("write sidecar: %w", err)
	}

	return nil
}

// listRootNotes reads the flat vault root and collects one string per non-dir
// entry for which extract returns ok; a ("", false) result skips the entry. A
// missing vault is treated as empty. Shared by ListBasenames and ListIDs so the
// flat-root traversal lives in exactly one place.
func listRootNotes(vault string, extract func(name string) (string, bool)) ([]string, error) {
	out := []string{}

	entries, err := os.ReadDir(vault)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}

		return nil, fmt.Errorf("read vault root: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		if s, ok := extract(e.Name()); ok {
			out = append(out, s)
		}
	}

	return out, nil
}

// pathOf returns the vault-relative path for a note, e.g. "foo.md". The vault
// is flat — notes live at the root (Permanent/ and MOCs/ are retired).
func pathOf(basename string) string {
	return basename + ".md"
}
