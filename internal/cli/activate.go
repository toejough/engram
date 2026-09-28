package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/luhmann"
)

// ActivateArgs holds parsed flags for `engram activate`.
type ActivateArgs struct {
	Vault string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
	Notes []string `targ:"flag,name=note,desc=note path to mark used (repeatable)"`
	// Parent resolves every ref against ENGRAM_PARENT, pulling it down,
	// even when a local file of that name exists (design D8).
	Parent bool `targ:"flag,name=parent,desc=resolve each ref against ENGRAM_PARENT (pulling it down) instead of the local vault"` //nolint:lll // single struct-tag string
	// VaultName is the vault: identity stamped on a pulled-down copy.
	VaultName string `targ:"flag,name=vault-name,env=ENGRAM_VAULT_NAME,desc=vault name stamped on a pulled-down note's vault: field (default \"personal\")"` //nolint:lll // single struct-tag string
}

// ActivateDeps holds injected dependencies for RunActivate.
type ActivateDeps struct {
	// Lock acquires an exclusive flock on vault/.luhmann.lock and returns a release
	// func. Wired via vaultLockFromLocker in newActivateDeps. Guards the sidecar
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
	// NoteExists reports whether a note's .md file exists: that, not its
	// sidecar, makes a ref a local hit (design D8, M8).
	NoteExists func(path string) bool
	// Resolve, when set, replaces the path-based local lookup: it maps a
	// ref to its note file and reports whether one exists. A served
	// activate sets it to resolve only against the vault's listed note
	// names (final review F2); the local CLI leaves it nil and keeps
	// resolving absolute and vault-relative paths.
	Resolve func(ref string) (string, bool, error)
	// Pull pulls a ref down from the configured parent (design D8); nil
	// when no parent is configured, and for a served activate, which never
	// reaches past its own vault. It runs with no vault lock held.
	Pull func(ctx context.Context, ref string) error
	// Finish runs once after every pull, with no lock held (the drain
	// after a parent contact, design D6); nil when there is no parent.
	Finish func(ctx context.Context)
}

// RunActivate marks each ref used (design D8). A ref whose .md exists
// locally is a hit: its sidecar's LastUsed is bumped (a note without a
// sidecar still counts). A miss — or every ref with --parent — is pulled
// down from the parent when the ref is a basename or <basename>.md; a bare
// Luhmann ID never goes to the parent. Each failed ref is reported, and
// the command fails when any ref failed.
func RunActivate(ctx context.Context, args ActivateArgs, deps ActivateDeps) error {
	result, err := activateRefs(ctx, args, deps)
	if err != nil {
		return err
	}

	if len(result.Failed) > 0 {
		return fmt.Errorf("%w: %d of %d", errActivateFailed, len(result.Failed), len(args.Notes))
	}

	return nil
}

// unexported variables.
var (
	errActivateBadRef = errors.New("activate: a served ref must be a vault note name " +
		"(no absolute path, path separator or \"..\")")
	errActivateFailed   = errors.New("activate: note ref(s) failed")
	errActivateNotFound = errors.New("note not found")
	errActivateNotSent  = errors.New("not sent to the parent: only a basename or <basename>.md is " +
		"(a bare Luhmann ID or a path never is)")
)

// activateFailure is one ref that could not be activated, and why.
type activateFailure struct {
	Ref   string `json:"ref"`
	Error string `json:"error"`
	// notFound marks a ref with no note to activate (a client error), as
	// opposed to a failure while activating one.
	notFound bool
}

// activateResult is the per-ref outcome of an activate: the refs that
// were activated and those that failed. A served activate returns it as
// its body when some refs failed.
type activateResult struct {
	Activated []string          `json:"activated"`
	Failed    []activateFailure `json:"failed"`
}

// fail records a failed ref and reports it.
func (r *activateResult) fail(deps ActivateDeps, ref string, err error) {
	deps.LogWarning("activate: %s: %v", ref, err)

	r.Failed = append(r.Failed, activateFailure{
		Ref: ref, Error: err.Error(), notFound: errors.Is(err, errActivateNotFound),
	})
}

// activateLocally is the locked pass: it bumps every local hit, records
// each ref's outcome, and returns the refs to pull from the parent.
func activateLocally(args ActivateArgs, deps ActivateDeps, result *activateResult) ([]string, error) {
	// Acquire the vault lock before the bump loop so a concurrent amend/resituate
	// re-embed cannot clobber the freshly-written vectors with stale ones. bumpLastUsed
	// must NOT re-acquire the lock (RunAmend already holds it when it calls
	// reEmbedAndActivate→bumpLastUsed — re-acquiring would self-deadlock).
	release, lockErr := acquireOptionalLock(deps.Lock, args.Vault)
	if lockErr != nil {
		return nil, fmt.Errorf("activate: acquiring vault lock: %w", lockErr)
	}

	defer release()

	date := deps.Now().Format(noteDateFormat)
	remote := make([]string, 0, len(args.Notes))

	for _, ref := range args.Notes {
		pull, refErr := activateRef(args, deps, ref, date)

		switch {
		case refErr != nil:
			result.fail(deps, ref, refErr)
		case pull:
			remote = append(remote, ref)
		default:
			result.Activated = append(result.Activated, ref)
		}
	}

	return remote, nil
}

// activateRef resolves one ref in the locked pass: a local hit is bumped,
// a parent candidate is reported for pulling, anything else fails.
func activateRef(args ActivateArgs, deps ActivateDeps, ref, date string) (bool, error) {
	if args.Parent {
		// --parent never looks locally, so a non-candidate ref is only "not
		// sent", never "not found".
		if !isParentCandidate(ref) {
			return false, errActivateNotSent
		}

		return true, nil
	}

	full, found, resolveErr := resolveLocalRef(args, deps, ref)
	if resolveErr != nil {
		return false, resolveErr
	}

	if found {
		return false, bumpLocalHit(deps, full, date)
	}

	if deps.Pull == nil {
		return false, errActivateNotFound
	}

	if !isParentCandidate(ref) {
		return false, fmt.Errorf("%w; %w", errActivateNotFound, errActivateNotSent)
	}

	return true, nil
}

// activateRefs runs activate and returns each ref's outcome; the error is
// only for a failure of the whole command (no parent for --parent, or the
// vault lock).
func activateRefs(ctx context.Context, args ActivateArgs, deps ActivateDeps) (activateResult, error) {
	if args.Parent && deps.Pull == nil {
		return activateResult{}, errParentNotConfigured
	}

	var result activateResult

	remote, lockErr := activateLocally(args, deps, &result)
	if lockErr != nil {
		return activateResult{}, lockErr
	}

	// Pulls run after the local pass released the vault lock: each takes
	// the lock only for its own write (no fetch under the lock).
	for _, ref := range remote {
		pullErr := deps.Pull(ctx, ref)
		if pullErr != nil {
			result.fail(deps, ref, pullErr)

			continue
		}

		result.Activated = append(result.Activated, ref)
	}

	if len(remote) > 0 && deps.Finish != nil {
		deps.Finish(ctx)
	}

	return result, nil
}

// bumpLastUsed reads a note's sidecar, sets LastUsed=date, and rewrites it.
// Vectors/ContentHash are preserved (LastUsed is metadata) so it never triggers
// a re-embed. Idempotent for the same date. Sidecar writes go through the
// injected atomic write (WriteFileAtomic, temp+rename) AND RunActivate holds
// the vault flock before calling this helper, so a concurrent amend/resituate
// re-embed cannot clobber the freshly-written vectors. This helper must NOT
// acquire the vault flock itself — RunAmend already holds it when it calls
// reEmbedAndActivate→bumpLastUsed, so a re-acquire would self-deadlock on a
// per-fd flock.
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

// bumpLocalHit bumps a local hit's sidecar; a note without a sidecar is
// still a hit (M8).
func bumpLocalHit(deps ActivateDeps, full, date string) error {
	bumpErr := bumpLastUsed(embed.SidecarPath(full), date, deps.Read, deps.Write)
	if bumpErr != nil && !errors.Is(bumpErr, fs.ErrNotExist) {
		return bumpErr
	}

	return nil
}

// isParentCandidate reports whether ref may be resolved against the
// parent: a basename or <basename>.md — never a bare Luhmann ID, which is
// minted per vault (M8), and never a path.
func isParentCandidate(ref string) bool {
	name := normalizeNoteRef(ref)
	if strings.ContainsAny(name, `/\`) {
		return false
	}

	_, isBasename := luhmann.FromBasename(name)

	return isBasename
}

// listedNoteResolver resolves a served ref only against the vault's listed
// note names (final review F2), as the raw show route does: the ref is
// matched, never joined into a path, so it cannot reach outside the vault.
func listedNoteResolver(deps Deps, vault string) func(ref string) (string, bool, error) {
	listMD := listMDFromFS(deps.FS)

	return func(ref string) (string, bool, error) {
		names, listErr := listMD(vault)
		if listErr != nil {
			return "", false, fmt.Errorf("activate: listing notes: %w", listErr)
		}

		fileName := normalizeNoteRef(ref) + mdExt
		if !slices.Contains(names, fileName) {
			return "", false, nil
		}

		return filepath.Join(vault, fileName), true, nil
	}
}

// localNotePath is ref's note file: an absolute path as given, otherwise
// joined to the vault, with .md added when the ref omits it.
func localNotePath(vault, ref string) string {
	full := ref
	if !filepath.IsAbs(full) {
		full = filepath.Join(vault, ref)
	}

	if !strings.HasSuffix(full, mdExt) {
		full += mdExt
	}

	return full
}

// newActivateDeps composes RunActivate's dependencies from the injected edge
// Deps (pure composition — no direct I/O; #700). Sidecar writes go through
// WriteFileAtomic (temp+rename) so concurrent readers always see either the
// old or new file.
func newActivateDeps(d Deps) ActivateDeps {
	const sidecarPerm = 0o600

	return ActivateDeps{
		Lock: vaultLockFromLocker(d.Lock),
		Now:  d.Now,
		Read: d.FS.ReadFile,
		Write: func(path string, data []byte) error {
			return d.FS.WriteFileAtomic(path, data, sidecarPerm)
		},
		LogWarning: logWarningTo(d.Stderr),
		NoteExists: func(path string) bool {
			_, statErr := d.FS.Stat(path)

			return statErr == nil
		},
	}
}

// resolveLocalRef maps ref to its local note file and reports whether the
// note exists: through deps.Resolve when set, otherwise as a path
// (localNotePath) whose .md must exist.
func resolveLocalRef(args ActivateArgs, deps ActivateDeps, ref string) (string, bool, error) {
	if deps.Resolve != nil {
		return deps.Resolve(ref)
	}

	full := localNotePath(args.Vault, ref)

	return full, deps.NoteExists(full), nil
}

// validateServedActivateRefs rejects a served activate whose refs are not
// plain note names: an absolute path, a path separator or ".." anywhere
// fails the whole request before anything is touched (final review F2).
func validateServedActivateRefs(refs []string) error {
	for _, ref := range refs {
		if filepath.IsAbs(ref) || strings.ContainsAny(ref, `/\`) || strings.Contains(ref, "..") {
			return fmt.Errorf("%w: %q", errActivateBadRef, ref)
		}
	}

	return nil
}
