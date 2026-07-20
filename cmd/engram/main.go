// Package main provides the engram CLI binary entry point (ARCH-6). It is
// deliberately declaration-free: raw impure capabilities enter as func
// references and sanctioned closures in the cli.Primitives literal, and
// ALL composition, error wrapping, and lifecycle logic lives in
// internal/cli (targ check-thin-api enforces this shape; #700).
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
		ReadFile:     os.ReadFile,
		WriteFile:    os.WriteFile,
		MkdirAll:     os.MkdirAll,
		MkdirTemp:    os.MkdirTemp,
		Stat:         os.Stat,
		ReadDir:      os.ReadDir,
		Remove:       os.Remove,
		RemoveAll:    os.RemoveAll,
		Rename:       os.Rename,
		WalkDir:      filepath.WalkDir,
		EmbedRuntime: hugotRuntime{},
		Chmod:        os.Chmod,
		Getenv:       os.Getenv,
		Now:          time.Now,
		Getwd:        os.Getwd,
		UserHomeDir:  os.UserHomeDir,
		WriteFileExcl: func(path string, data []byte, perm fs.FileMode) error {
			// Doctrine survivor S-1: os.WriteFile's own body with
			// O_CREATE|O_EXCL — mechanical error propagation only; behavior
			// changes extend the Primitives SIGNATURE, never this body.
			// Errors return RAW (unwrapped): the *fs.PathError must keep
			// errors.Is(err, fs.ErrExist) alive; internal/cli wraps once.
			//nolint:gosec // operator-controlled path
			file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
			if err != nil {
				return err //nolint:wrapcheck // S-1 contract: raw error, internal wraps once
			}

			_, err = file.Write(data)

			closeErr := file.Close()
			if closeErr != nil && err == nil {
				err = closeErr
			}

			return err
		},
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
		OpenDebugFile: func(path string, perm fs.FileMode) (cli.WriteSyncer, error) {
			// Path comes from operator-set ENGRAM_DEBUG_LOG, not user input.
			//nolint:gosec // operator-controlled path
			return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perm)
		},
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
		StartSignalPulses: func(pulses chan<- struct{}, buffer int) {
			sigCh := make(chan os.Signal, buffer)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			go cli.ForwardAsPulses(sigCh, pulses)
		},
	}, os.Stdout, os.Stderr, os.Exit))...)
}
