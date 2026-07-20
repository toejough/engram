// Package cli implements the engram command-line interface (ARCH-6).
package cli

import (
	"errors"
	"fmt"
	"io/fs"
)

// unexported constants.
const (
	luhmannLockFile  = ".luhmann.lock"
	manifestLockFile = ".manifest.lock"
)

// unexported variables.
var (
	errNotADirectory = errors.New("not a directory")
)

// acquireOptionalLock calls lock(arg) if lock is non-nil and returns (release, nil).
// When lock is nil it returns a no-op release and nil so callers can always defer
// the release unconditionally without a nil guard. Used at all Run* entry points
// to handle an injected-vs-nil lock without adding complexity branches to those
// already-complex functions.
func acquireOptionalLock(lock func(string) (func(), error), arg string) (func(), error) {
	if lock == nil {
		return func() {}, nil
	}

	return lock(arg)
}

// listRootNotes reads the flat vault root via the injected readDir and
// collects one string per non-dir entry for which extract returns ok. A
// missing vault is treated as empty. Shared by listIDsFromFS and
// listBasenamesFromFS so the flat-root traversal lives in exactly one place.
func listRootNotes(
	readDir func(string) ([]fs.DirEntry, error),
	vault string,
	extract func(name string) (string, bool),
) ([]string, error) {
	out := []string{}

	entries, err := readDir(vault)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
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
