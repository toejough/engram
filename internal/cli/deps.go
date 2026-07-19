package cli

import (
	"io"
	"io/fs"
	"time"

	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/update"
)

// Deps carries every impure capability the CLI needs, wired by cmd/engram.
// internal/ code never calls os.*, exec, syscall, or time.Now directly —
// production I/O enters exclusively through this struct (#700, ADR-0001).
type Deps struct {
	// Stdout receives command output (production: os.Stdout).
	Stdout io.Writer
	// Stderr receives error output (production: os.Stderr).
	Stderr io.Writer
	// Exit terminates the process with a status code (production: os.Exit).
	Exit func(int)
	// Getenv reads an environment variable (production: os.Getenv).
	Getenv func(string) string
	// Now returns the current wall-clock time (production: time.Now).
	Now func() time.Time
	// Getwd returns the process working directory (production: os.Getwd).
	Getwd func() (string, error)
	// UserHomeDir returns the user's home directory (production: os.UserHomeDir).
	UserHomeDir func() (string, error)
	// FS is the filesystem edge (production: cmd/engram's osFS).
	FS EdgeFS
	// Lock acquires exclusive cross-process file locks (production: flockLocker).
	Lock FileLocker
	// Commander runs external commands for `engram update` (production: osCommander).
	Commander update.Commander
	// Embed is the production embedder backend (hugot-backed lazy embedder).
	Embed embed.Embedder
	// DebugLog is the debug-log sink; nil disables debug logging (no-op logger).
	DebugLog io.Writer
}

// EdgeFS is the filesystem capability surface for production wiring. All
// mode/info/entry types come from io/fs — never os — so internal/ stays
// free of I/O-capable imports (#700). It is a single injected FS capability
// surface by design; splitting it would be an architectural redesign, not a
// lint fix, and later #700 tasks add further methods (e.g. WriteFileExcl)
// per constraints-and-resolutions.md.
//
//nolint:interfacebloat // see doc comment above
type EdgeFS interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	// WriteFileAtomic writes via temp file + same-directory rename so a
	// concurrent reader sees the old or the new content, never a torn
	// write (ADR-0013).
	WriteFileAtomic(path string, data []byte, perm fs.FileMode) error
	// WriteFileExcl creates path exclusively (O_CREATE|O_EXCL semantics): it
	// errors with an error satisfying errors.Is(err, fs.ErrExist) when path
	// already exists. The learn family's ID-collision backstop (ADR-0013 K1)
	// and idempotent vault bootstrap both require exclusive create.
	WriteFileExcl(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	MkdirTemp(dir, pattern string) (string, error)
	Stat(path string) (fs.FileInfo, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	Remove(path string) error
	RemoveAll(path string) error
	Rename(oldPath, newPath string) error
	WalkDir(root string, fn fs.WalkDirFunc) error
}

// FileLocker acquires an exclusive advisory lock on the file at path,
// creating it if absent. unlock releases the lock and closes the handle.
// Production: flock(2) via cmd/engram's flockLocker (ADR-0013).
type FileLocker interface {
	Lock(path string) (unlock func() error, err error)
}
