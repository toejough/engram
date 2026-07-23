package cli

import (
	"context"
	"io"
	"io/fs"
	"time"

	"github.com/toejough/engram/internal/embed"
)

// ExecPrims groups the raw external-command capabilities (doctrine flag
// C-1): one erased run closure + the platform not-found sentinel value;
// collection, wrapping, and not-found translation live internal in
// primCommander.
type ExecPrims struct {
	RunCommand func(
		ctx context.Context, dir, name string, args []string, stdout, stderr io.Writer,
	) error // closure: exec.CommandContext; Dir/Stdout/Stderr assignment; Run
	NotFoundErr error // exec.ErrNotFound
}

// FSPrims groups the raw filesystem capabilities: direct os/filepath
// references plus the exclusive-create eraser (doctrine survivor S-1 —
// a single-call primitive: os.OpenFile with O_CREATE|O_EXCL erases *os.File
// to io.WriteCloser; the write+close error-merge lives in primFS.WriteFileExcl,
// unit-testable with fakes).
type FSPrims struct {
	ReadFile     func(path string) ([]byte, error)                           // os.ReadFile
	WriteFile    func(path string, data []byte, perm fs.FileMode) error      // os.WriteFile
	MkdirAll     func(path string, perm fs.FileMode) error                   // os.MkdirAll
	MkdirTemp    func(dir, pattern string) (string, error)                   // os.MkdirTemp
	Stat         func(path string) (fs.FileInfo, error)                      // os.Stat
	ReadDir      func(path string) ([]fs.DirEntry, error)                    // os.ReadDir
	Remove       func(path string) error                                     // os.Remove
	RemoveAll    func(path string) error                                     // os.RemoveAll
	Rename       func(oldPath, newPath string) error                         // os.Rename
	WalkDir      func(root string, fn fs.WalkDirFunc) error                  // filepath.WalkDir
	Chmod        func(path string, mode fs.FileMode) error                   // os.Chmod
	OpenFileExcl func(path string, perm fs.FileMode) (io.WriteCloser, error) // S-1 eraser
}

// LockPrims groups the raw advisory file-locking capabilities:
// single-syscall closures over a raw fd; the lock lifecycle lives internal
// in primLocker (design flags P-2/P-3: semantic per-op funcs over a raw
// uintptr fd, via syscall.Open, never os.OpenFile().Fd()).
type LockPrims struct {
	OpenLockFile   func(path string, perm fs.FileMode) (uintptr, error) // syscall.Open O_CREAT|O_RDWR
	FlockExclusive func(fd uintptr) error                               // syscall.Flock LOCK_EX
	FlockUnlock    func(fd uintptr) error                               // syscall.Flock LOCK_UN
	CloseFD        func(fd uintptr) error                               // syscall.Close
}

// Primitives carries raw impure capabilities as func values, grouped into
// cohesive capability sub-structs (FS, Lock, Exec, Proc). cmd/engram
// populates each group with direct references to os/syscall/filepath/time
// functions, or single-call erasers where a signature must be hidden (fd
// instead of *os.File, WriteSyncer instead of *os.File, io.WriteCloser
// instead of *os.File, pulses instead of os.Signal, or an enumerated
// stdlib-equivalent primitive (doctrine survivors: S-1 OpenFileExcl eraser,
// C-1 RunCommand, SIG-1 StartSignalPulses). ALL composition, error
// wrapping, and lifecycle logic lives in internal/cli; targ check-thin-api
// enforces that the cmd side stays wiring-only (#700).
type Primitives struct {
	// Filesystem capabilities (consumed by primFS and the embed cache).
	FS FSPrims

	// Advisory file locking (consumed by primLocker).
	Lock LockPrims

	// External command execution (consumed by primCommander).
	Exec ExecPrims

	// Process-scoped capabilities (consumed by NewDeps directly, the
	// debug sink, and startForceExit).
	Proc ProcPrims

	// Embedding runtime (cmd wires an EMPTY struct with single-call
	// methods; all lifecycle/config/cache policy is internal — doctrine
	// flags D-1/E-1/E-2).
	EmbedRuntime embed.Runtime
}

// ProcPrims groups the raw process-scoped capabilities: env, clock,
// working dir, home dir, the debug-sink opener (empty-path branch + sync
// policy internal), and the signal-pulse starter (doctrine flag SIG-1:
// single-purpose starter closure; pulse forwarding is internal via
// ForwardAsPulses; buffer/pulse-channel/force-exit policy internal).
type ProcPrims struct {
	Getenv            func(key string) string                                  // os.Getenv
	Now               func() time.Time                                         // time.Now
	Getwd             func() (string, error)                                   // os.Getwd
	UserHomeDir       func() (string, error)                                   // os.UserHomeDir
	OpenDebugFile     func(path string, perm fs.FileMode) (WriteSyncer, error) // os.OpenFile O_APPEND|O_CREATE|O_WRONLY
	StartSignalPulses func(pulses chan<- struct{}, buffer int)                 // SIG-1 closure
}

// WriteSyncer is the debug-sink capability surface (*os.File satisfies it).
type WriteSyncer interface {
	io.Writer
	Sync() error
}

// NewDeps composes the production Deps carrier from raw primitives: the
// EdgeFS implementation (contextual %w wrapping + the ADR-0013 atomic-write
// dance), the flock lifecycle, the debug sink (ENGRAM_DEBUG_LOG; empty path
// or failed open → nil → no-op logger), and the repeated-signal force-exit
// watcher. cmd/engram calls this exactly once from main(); tests call it
// with fake primitives to unit-test the composition (#700).
func NewDeps(prims Primitives, stdout, stderr io.Writer, exit func(int)) Deps {
	startForceExit(prims, exit)

	deps := Deps{
		Stdout:      stdout,
		Stderr:      stderr,
		Exit:        exit,
		Getenv:      prims.Proc.Getenv,
		Now:         prims.Proc.Now,
		Getwd:       prims.Proc.Getwd,
		UserHomeDir: prims.Proc.UserHomeDir,
		FS:          primFS{prims: prims},
		Lock:        primLocker{prims: prims},
		Commander:   primCommander{prims: prims},
		DebugLog:    openDebugSink(envOrEmpty(prims.Proc.Getenv, debugLogEnvVar), prims.Proc.OpenDebugFile),
	}

	// The lazy embedder is constructed exactly once, here: NewDeps is the
	// sole composition point for Deps.Embed, so the model unpacks at most
	// once per process (guarded: minimal fake Primitives without Getenv
	// skip it). R6/D-1:
	// backend composed from the raw EmbedRuntime, cache FS from the raw
	// filesystem primitives — no embed wiring in cmd. A nil EmbedRuntime
	// surfaces as embed.ErrRuntimeMissing on first use (fail-loud lazy),
	// never a panic.
	if prims.Proc.Getenv != nil {
		deps.Embed = embed.NewLazyEmbedder(
			embed.NewRuntimeBackend(prims.EmbedRuntime),
			embed.NewCacheFS(embed.CacheFSPrims{
				Stat:      prims.FS.Stat,
				MkdirAll:  prims.FS.MkdirAll,
				MkdirTemp: prims.FS.MkdirTemp,
				WriteFile: prims.FS.WriteFile,
				Rename:    prims.FS.Rename,
				RemoveAll: prims.FS.RemoveAll,
			}),
			CacheDirFromHome(homeOrEmpty(deps), embed.BundledModelID, prims.Proc.Getenv))
	}

	return deps
}

// envOrEmpty reads key via getenv, tolerating a nil (unwired) capability.
func envOrEmpty(getenv func(string) string, key string) string {
	if getenv == nil {
		return ""
	}

	return getenv(key)
}
