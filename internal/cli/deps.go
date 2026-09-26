package cli

import (
	"context"
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
	// Stdin supplies interactive prompt input (production: os.Stdin) — read
	// by skill-runbook-registration's `engram register-skills` prompt loop
	// via a bufio.Scanner internal/cli owns; a nil Stdin is safe as long as
	// no code path that reads it is exercised (e.g. dry-run/non-interactive
	// registration, or any other command).
	Stdin io.Reader
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
	// Username returns the OS username of the process running engram
	// (production: os/user.Current().Username). Used as the fallback when
	// git config user.email resolves to nothing.
	Username func() (string, error)
	// IsTerminal reports whether Stdin is an interactive terminal (production:
	// golang.org/x/term's term.IsTerminal(os.Stdin.Fd()), checked in
	// cmd/engram/main.go — skill-runbook-registration's interactive-prompt
	// gate). A nil IsTerminal is treated as non-interactive by callers that
	// check it.
	IsTerminal func() bool
	// FS is the filesystem edge (production: cmd/engram's osFS).
	FS EdgeFS
	// Lock acquires exclusive cross-process file locks (production: flockLocker).
	Lock FileLocker
	// Commander runs external commands for `engram update` (production: primCommander).
	Commander update.Commander
	// Spawner runs a binary with args + extra env, stdio inherited from the
	// parent process, for `engram update`'s post-install re-exec (production: primSpawner).
	Spawner update.Spawner
	// Embed is the production embedder backend (hugot-backed lazy embedder).
	Embed embed.Embedder
	// DebugLog is the debug-log sink; nil disables debug logging (no-op logger).
	DebugLog io.Writer
	// NewServeMux creates one opaque server-side route table (production:
	// http.NewServeMux(), composed in cmd/engram — internal/ never imports
	// net/http directly, depguard #700's internal-purity rule; RawServeMux
	// is an erased handle internal/ only ever holds and passes back,
	// mirroring embed.RawSession's erasure pattern).
	NewServeMux func() RawServeMux
	// RegisterRoute registers one method+pattern route on mux (production:
	// a single mux.HandleFunc call). Called once per route by RunServe —
	// the per-route loop lives here in internal/cli, never in cmd/engram
	// (targ check-thin-api forbids loops in cmd/engram's declarations).
	RegisterRoute func(mux RawServeMux, method, pattern string, handler ServeHandler)
	// ListenAndServe blocks serving mux on addr until ctx is canceled or an
	// unrecoverable listen error occurs (production: real http.Server).
	ListenAndServe func(ctx context.Context, mux RawServeMux, addr string) error
	// Fetch issues one HTTP request against a fully-formed URL (query
	// percent-encoding happens in internal/cli — depguard #700's
	// internal-purity rule disallows net/url under internal/, so the raw
	// primitive takes a plain string) and returns its response reduced to
	// primitive types (production: real net/http.Client, composed in
	// cmd/engram). Used by CLI targets when ENGRAM_SERVER is set.
	Fetch func(ctx context.Context, method, url string, body []byte) (FetchResponse, error)
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
	// Lstat returns FileInfo for path without following a trailing symlink
	// (D8: distinguishes symlink / real file / dir).
	Lstat(path string) (fs.FileInfo, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	Remove(path string) error
	RemoveAll(path string) error
	Rename(oldPath, newPath string) error
	// Symlink creates link as a symbolic link to target (D1/D3: materializes
	// the sync engine's engram-owned-root links).
	Symlink(target, link string) error
	// Readlink returns the destination that the symbolic link at path
	// points to (D5: dangling-link cleanup).
	Readlink(path string) (string, error)
	WalkDir(root string, fn fs.WalkDirFunc) error
}

// FileLocker acquires an exclusive advisory lock on the file at path,
// creating it if absent. unlock releases the lock and closes the handle.
// Production: flock(2) via cmd/engram's flockLocker (ADR-0013).
type FileLocker interface {
	Lock(path string) (unlock func() error, err error)
}
