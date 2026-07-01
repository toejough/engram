package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/toejough/engram/internal/embed"
)

// ActivateArgs holds parsed flags for `engram activate`.
type ActivateArgs struct {
	Vault string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
	Notes []string `targ:"flag,name=note,desc=note path to mark used (repeatable)"`
}

// ActivateDeps holds injected dependencies for RunActivate.
type ActivateDeps struct {
	// Lock acquires an exclusive flock on vault/.luhmann.lock and returns a release
	// func. Wired to vaultFS.Lock in newOsActivateDeps. Guards the sidecar
	// read-modify-write (bumpLastUsed) against a concurrent amend/resituate re-embed
	// that could clobber the freshly-written vectors with stale ones if it races the
	// sidecar write.
	// Acquire only at RunActivate's entry point — bumpLastUsed must NOT re-acquire
	// (RunAmend already holds the lock when it calls reEmbedAndActivate→bumpLastUsed,
	// so a helper re-acquiring would self-deadlock on a per-fd flock).
	Lock       func(vault string) (func(), error)
	Now        func() time.Time
	Read       func(string) ([]byte, error)
	Write      func(string, []byte) error
	LogWarning func(string, ...any)
}

// RunActivate bumps the LastUsed field on each note's sidecar to today's date.
// A bad path is logged and skipped (log-and-continue). Returns nil if at least
// one note was successfully activated; returns an error only when ALL fail.
func RunActivate(args ActivateArgs, deps ActivateDeps) error {
	// Acquire the vault lock before the bump loop so a concurrent amend/resituate
	// re-embed cannot clobber the freshly-written vectors with stale ones. bumpLastUsed
	// must NOT re-acquire the lock (RunAmend already holds it when it calls
	// reEmbedAndActivate→bumpLastUsed — re-acquiring would self-deadlock).
	release, lockErr := acquireOptionalLock(deps.Lock, args.Vault)
	if lockErr != nil {
		return fmt.Errorf("activate: acquiring vault lock: %w", lockErr)
	}

	defer release()

	date := deps.Now().Format(noteDateFormat)

	failures := 0

	for _, notePath := range args.Notes {
		full := notePath
		if !filepath.IsAbs(full) {
			full = filepath.Join(args.Vault, notePath)
		}

		sidecarPath := embed.SidecarPath(full)

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
// a re-embed. Idempotent for the same date. Sidecar writes go through
// atomicWriteFile (temp+rename) AND RunActivate holds the vault flock before
// calling this helper, so a concurrent amend/resituate re-embed cannot clobber
// the freshly-written vectors. This helper must NOT acquire the vault flock
// itself — RunAmend already holds it when it calls reEmbedAndActivate→bumpLastUsed,
// so a re-acquire would self-deadlock on a per-fd flock.
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
		Lock:       (&osLearnFS{}).Lock,
		Now:        time.Now,
		Read:       os.ReadFile,
		Write:      osWriteSidecar,
		LogWarning: logWarningToStderrf,
	}
}

// osWriteSidecar writes a sidecar file with 0o600 permissions using atomicWriteFile
// (temp+rename) so concurrent readers always see either the old or new file.
func osWriteSidecar(path string, data []byte) error {
	const sidecarPerm = 0o600

	return atomicWriteFile(path, data, sidecarPerm)
}
