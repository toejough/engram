package cli

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/toejough/engram/internal/embed"
)

// ActivateArgs holds parsed flags for `engram activate`.
type ActivateArgs struct {
	Notes []string `targ:"flag,name=note,desc=note path to mark used (repeatable)"`
}

// ActivateDeps holds injected dependencies for RunActivate.
type ActivateDeps struct {
	Now        func() time.Time
	Read       func(string) ([]byte, error)
	Write      func(string, []byte) error
	LogWarning func(string, ...any)
}

// RunActivate bumps the LastUsed field on each note's sidecar to today's date.
// A bad path is logged and skipped (log-and-continue). Returns nil if at least
// one note was successfully activated; returns an error only when ALL fail.
func RunActivate(args ActivateArgs, deps ActivateDeps) error {
	date := deps.Now().Format(noteDateFormat)

	failures := 0

	for _, notePath := range args.Notes {
		sidecarPath := embed.SidecarPath(notePath)

		bumpErr := bumpLastUsed(sidecarPath, date, deps.Read, deps.Write)
		if bumpErr != nil {
			deps.LogWarning("activate: skipping %s: %v", notePath, bumpErr)

			failures++
		}
	}

	if failures == len(args.Notes) && len(args.Notes) > 0 {
		return errActivateAllFailed
	}

	return nil
}

// unexported variables.
var (
	errActivateAllFailed = errors.New("activate: all note paths failed")
)

// bumpLastUsed reads a note's sidecar, sets LastUsed=date, and rewrites it.
// Vectors/ContentHash are preserved (LastUsed is metadata) so it never triggers
// a re-embed. Idempotent for the same date. No lock: sidecar writes are atomic
// per-file and the vault flock guards only Luhmann ID sequencing.
func bumpLastUsed(
	sidecarPath, date string,
	read func(string) ([]byte, error),
	write func(string, []byte) error,
) error {
	data, readErr := read(sidecarPath)
	if readErr != nil {
		return fmt.Errorf("activate: reading sidecar %s: %w", sidecarPath, readErr)
	}

	sidecar, parseErr := embed.UnmarshalSidecar(data)
	if parseErr != nil {
		return fmt.Errorf("activate: parsing sidecar %s: %w", sidecarPath, parseErr)
	}

	if sidecar.LastUsed == date {
		return nil
	}

	sidecar.LastUsed = date

	writeErr := write(sidecarPath, embed.MarshalSidecar(sidecar))
	if writeErr != nil {
		return fmt.Errorf("activate: writing sidecar %s: %w", sidecarPath, writeErr)
	}

	return nil
}

// newOsActivateDeps wires RunActivate to the real filesystem and clock.
func newOsActivateDeps() ActivateDeps {
	return ActivateDeps{
		Now:        time.Now,
		Read:       os.ReadFile,
		Write:      osWriteSidecar,
		LogWarning: logWarningToStderrf,
	}
}

// osWriteSidecar writes a sidecar file with 0o600 permissions.
func osWriteSidecar(path string, data []byte) error {
	const sidecarPerm = 0o600

	return os.WriteFile(path, data, sidecarPerm) //nolint:wrapcheck // thin I/O adapter
}
