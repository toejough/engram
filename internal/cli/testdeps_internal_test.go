package cli

// Test-only composition of production Deps over real OS primitives, so
// wiring-integration tests drive the exact primFS/primLocker/debug-sink
// implementations the binary ships. Composition doctrine (#700): NewDeps is
// the single composition root — no hand-rolled adapter mirrors anywhere.

import (
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// ExportNewTestOsDeps returns production-composed Deps for wiring tests:
// one NewDeps call over an inline real-OS Primitives literal (mirrors
// cmd/engram/main.go's — doctrine flag DRIFT — minus StartSignalPulses,
// nil so startForceExit skips per SIG-1). Closures return raw os errors by
// primitive contract; primFS/primLocker add the single %w wrap.
func ExportNewTestOsDeps() Deps {
	return NewDeps(Primitives{
		ReadFile:    os.ReadFile,
		WriteFile:   os.WriteFile,
		MkdirAll:    os.MkdirAll,
		MkdirTemp:   os.MkdirTemp,
		Stat:        os.Stat,
		ReadDir:     os.ReadDir,
		Remove:      os.Remove,
		RemoveAll:   os.RemoveAll,
		Rename:      os.Rename,
		WalkDir:     filepath.WalkDir,
		Chmod:       os.Chmod,
		Getenv:      os.Getenv,
		Now:         time.Now,
		Getwd:       os.Getwd,
		UserHomeDir: os.UserHomeDir,
		WriteFileExcl: func(path string, data []byte, perm fs.FileMode) error {
			// Test-helper path; gosec is path-excluded for _test files.
			file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
			if err != nil {
				return err
			}

			_, err = file.Write(data)
			if closeErr := file.Close(); closeErr != nil && err == nil {
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
		OpenDebugFile: func(path string, perm fs.FileMode) (WriteSyncer, error) {
			// Operator-controlled path; gosec is path-excluded for _test files.
			return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perm)
		},
	}, os.Stdout, os.Stderr, func(int) {})
}
