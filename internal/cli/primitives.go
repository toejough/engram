package cli

import (
	"io"
	"io/fs"
	"time"

	"github.com/toejough/engram/internal/embed"
)

// Primitives carries raw impure capabilities as func values. cmd/engram
// populates it with direct references to os/syscall/filepath/time functions,
// single-call closures where a signature must be erased (fd instead of
// *os.File, WriteSyncer instead of *os.File, pulses instead of os.Signal),
// or an enumerated stdlib-equivalent survivor closure (doctrine survivors:
// S-1 WriteFileExcl here; C-1 RunCommand lands in T17).
// ALL composition, error wrapping, and lifecycle logic lives in internal/cli;
// targ check-thin-api enforces that the cmd side stays declaration-free (#700).
type Primitives struct {
	// Filesystem (direct os/filepath references).
	ReadFile  func(path string) ([]byte, error)                      // os.ReadFile
	WriteFile func(path string, data []byte, perm fs.FileMode) error // os.WriteFile
	MkdirAll  func(path string, perm fs.FileMode) error              // os.MkdirAll
	MkdirTemp func(dir, pattern string) (string, error)              // os.MkdirTemp
	Stat      func(path string) (fs.FileInfo, error)                 // os.Stat
	ReadDir   func(path string) ([]fs.DirEntry, error)               // os.ReadDir
	Remove    func(path string) error                                // os.Remove
	RemoveAll func(path string) error                                // os.RemoveAll
	Rename    func(oldPath, newPath string) error                    // os.Rename
	WalkDir   func(root string, fn fs.WalkDirFunc) error             // filepath.WalkDir
	Chmod     func(path string, mode fs.FileMode) error              // os.Chmod

	// Exclusive create (doctrine survivor S-1 — a stdlib-equivalent
	// primitive closure: os.WriteFile's own body with O_CREATE|O_EXCL;
	// behavior changes extend this SIGNATURE, never the cmd body).
	WriteFileExcl func(path string, data []byte, perm fs.FileMode) error

	// Process, env, clock (direct references).
	Getenv      func(key string) string // os.Getenv
	Now         func() time.Time        // time.Now
	Getwd       func() (string, error)  // os.Getwd
	UserHomeDir func() (string, error)  // os.UserHomeDir

	// Advisory file locking (single-syscall closures; lifecycle internal —
	// design flags P-2/P-3: semantic per-op funcs over a raw uintptr fd,
	// via syscall.Open, never os.OpenFile().Fd()).
	OpenLockFile   func(path string, perm fs.FileMode) (uintptr, error) // syscall.Open O_CREAT|O_RDWR
	FlockExclusive func(fd uintptr) error                               // syscall.Flock LOCK_EX
	FlockUnlock    func(fd uintptr) error                               // syscall.Flock LOCK_UN
	CloseFD        func(fd uintptr) error                               // syscall.Close

	// Debug sink (single-call closure; empty-path branch + sync policy internal).
	OpenDebugFile func(path string, perm fs.FileMode) (WriteSyncer, error) // os.OpenFile O_APPEND|O_CREATE|O_WRONLY

	// Signal (single-purpose starter closure; pulse forwarding is internal
	// via ForwardAsPulses; buffer/pulse-channel/force-exit policy internal).
	StartSignalPulses func(pulses chan<- struct{}, buffer int)
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
		Getenv:      prims.Getenv,
		Now:         prims.Now,
		Getwd:       prims.Getwd,
		UserHomeDir: prims.UserHomeDir,
		FS:          primFS{prims: prims},
		Lock:        primLocker{prims: prims},
		DebugLog:    openDebugSink(envOrEmpty(prims.Getenv, debugLogEnvVar), prims.OpenDebugFile),
	}

	// The lazy embedder is constructed once here, preserving the
	// one-unpack-per-process property of the old sharedEmbedder singleton
	// (guarded: minimal fake Primitives without Getenv skip it). R6: T14
	// swaps this line to the 3-arg constructor over cmd-injected backend
	// and cache capabilities.
	if prims.Getenv != nil {
		deps.Embed = embed.NewLazyEmbedder(
			CacheDirFromHome(homeOrEmpty(deps), embed.BundledModelID, prims.Getenv))
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
