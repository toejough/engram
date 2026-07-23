// Package main provides the engram CLI binary entry point (ARCH-6). It is
// wiring-only: raw impure capabilities enter as func references and
// sanctioned closures, grouped into cli.Primitives capability sub-structs
// by the checker-thin per-group functions below (each a single return of
// an external carrier call — zero composition, zero branching), and ALL
// composition, error wrapping, and lifecycle logic lives in internal/cli
// (targ check-thin-api enforces this shape; #700).
package main

import (
	"context"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/cli"
)

func main() {
	targ.Main(cli.Targets(cli.NewDeps(cli.Primitives{
		FS:           fsPrimitives(),
		Lock:         lockPrimitives(),
		Exec:         execPrimitives(),
		Proc:         procPrimitives(),
		EmbedRuntime: hugotRuntime{},
	}, os.Stdout, os.Stderr, os.Exit))...)
}

// execPrimitives groups the raw external-command capabilities: the C-1
// run-survivor closure plus the platform not-found sentinel value.
func execPrimitives() cli.ExecPrims {
	return cli.ExecPrims{
		RunCommand: func(ctx context.Context, dir, name string, args []string, stdout, stderr io.Writer) error {
			// Doctrine survivor C-1: construction + field assignments + one
			// invocation, zero branching. Behavior changes (timeout, env,
			// output policy, retry) extend the Primitives SIGNATURE, never
			// this body. Raw error out; primCommander wraps + translates.
			cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // name/args from internal callers
			cmd.Dir, cmd.Stdout, cmd.Stderr = dir, stdout, stderr

			return cmd.Run()
		},
		NotFoundErr: exec.ErrNotFound,
	}
}

// fsPrimitives groups the raw filesystem capabilities: direct os/filepath
// references plus the S-1 exclusive-create eraser.
func fsPrimitives() cli.FSPrims {
	return cli.FSPrims{
		ReadFile:  os.ReadFile,
		WriteFile: os.WriteFile,
		MkdirAll:  os.MkdirAll,
		MkdirTemp: os.MkdirTemp,
		Stat:      os.Stat,
		ReadDir:   os.ReadDir,
		Remove:    os.Remove,
		RemoveAll: os.RemoveAll,
		Rename:    os.Rename,
		WalkDir:   filepath.WalkDir,
		Chmod:     os.Chmod,
		OpenFileExcl: func(path string, perm fs.FileMode) (io.WriteCloser, error) {
			return os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm) //nolint:gosec // operator-controlled path
		},
	}
}

// lockPrimitives groups the raw advisory-flock capabilities: semantic
// single-syscall closures over a raw fd (design flags P-2/P-3 —
// syscall.Open so no finalizer can release the lock mid-hold; the lock
// lifecycle lives internal in primLocker).
func lockPrimitives() cli.LockPrims {
	return cli.LockPrims{
		OpenLockFile: func(path string, perm fs.FileMode) (uintptr, error) {
			fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_RDWR, uint32(perm))
			return uintptr(fd), err
		},
		FlockExclusive: func(fd uintptr) error {
			return syscall.Flock(int(fd), syscall.LOCK_EX)
		},
		FlockUnlock: func(fd uintptr) error {
			return syscall.Flock(int(fd), syscall.LOCK_UN)
		},
		CloseFD: func(fd uintptr) error {
			return syscall.Close(int(fd))
		},
	}
}

// procPrimitives groups the raw process-scoped capabilities: env, clock,
// working dir, home dir, the debug-sink opener, and the SIG-1
// signal-pulse starter closure.
func procPrimitives() cli.ProcPrims {
	return cli.ProcPrims{
		Getenv:      os.Getenv,
		Now:         time.Now,
		Getwd:       os.Getwd,
		UserHomeDir: os.UserHomeDir,
		OpenDebugFile: func(path string, perm fs.FileMode) (cli.WriteSyncer, error) {
			// Path comes from operator-set ENGRAM_DEBUG_LOG, not user input.
			//nolint:gosec // operator-controlled path
			return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perm)
		},
		StartSignalPulses: func(pulses chan<- struct{}, buffer int) {
			// Doctrine flag SIG-1: single-purpose starter closure — three
			// single-call statements (make, Notify, go forward), zero
			// branching. Behavior changes (signal set, buffer policy,
			// forwarding) extend the Primitives SIGNATURE, never this body;
			// pulse-channel, buffer-size, and force-exit policy live
			// internal in startForceExit.
			sigCh := make(chan os.Signal, buffer)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			go cli.ForwardAsPulses(sigCh, pulses)
		},
	}
}
