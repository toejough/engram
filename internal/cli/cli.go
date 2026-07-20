// Package cli implements the engram command-line interface (ARCH-6).
package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
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

// osLearnFS is the production filesystem adapter for the learn subcommand.
// Shrunk to Lock-only (#700 T3): the other methods moved to deps_compose.go
// compositions over EdgeFS. Lock stays here until Task L2 — it has four
// consumers outside the learn cluster (activate.go, amend.go, resituate.go,
// vocab_commands.go).
type osLearnFS struct{}

// Lock acquires an exclusive flock on vault/.luhmann.lock; returns a release func.
func (*osLearnFS) Lock(vault string) (func(), error) {
	return flockPath(filepath.Join(vault, luhmannLockFile))
}

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

// flockPath opens lockPath (O_CREATE|O_RDWR) and acquires an exclusive flock.
// Returns a release func that unlocks and closes the file. Used by osLearnFS.Lock
// (vault/.luhmann.lock) and the manifest lock wired into IngestDeps/PruneDeps
// (chunksDir/.manifest.lock) so all cross-process locking goes through one helper.
func flockPath(lockPath string) (func(), error) {
	const perm = 0o600

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, perm) //nolint:gosec // path from caller
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

// logWarningToStderrf is the transitional os.Stderr-bound LogWarning hook.
// Deps-migrated constructors use logWarningTo(d.Stderr) instead; this stays
// only for the not-yet-migrated constructors and dies in the cli.go purge task.
func logWarningToStderrf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
}

// pathOf returns the vault-relative path for a note, e.g. "foo.md". The vault
// is flat — notes live at the root (Permanent/ and MOCs/ are retired).
func pathOf(basename string) string {
	return basename + ".md"
}
