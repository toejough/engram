package cli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/update"
)

// IdentityDeps holds injected dependencies for `engram update
// --backfill-identity`. Composed from cli.Deps by newIdentityDeps (pure
// composition — no direct I/O; #700).
type IdentityDeps struct {
	// Lock acquires an exclusive flock on vault/.luhmann.lock, so backfill
	// cannot race a concurrent learn/amend.
	Lock       func(vault string) (func(), error)
	ListMD     func(vault string) ([]string, error)
	ReadFile   func(path string) ([]byte, error)
	WriteFile  func(path string, data []byte) error
	DetectRepo func(ctx context.Context) string
	DetectUser func(ctx context.Context) string
	Getenv     func(string) string
	// Embedder rebuilds the sidecar of a note backfill converted from CRLF
	// to LF (#789 design D2), since conversion changes its content hash.
	// Nil skips the rebuild.
	Embedder embed.Embedder
}

// noteIdentityFields points at a typed note doc's identity fields, so one
// backfill stamper serves fact and feedback notes.
type noteIdentityFields struct {
	project string
	repo    *string
	user    *string
	vault   *string
}

// applyIdentityBackfill runs backfillIdentity over vaultPath and copies its
// result onto report (the cli-layer-only fields documented on
// update.Report). After a successful non-dry-run backfill it also re-checks
// VaultHasNotesMissingIdentity, so a subsequent `engram update` (without
// --backfill-identity) prints no notice.
func applyIdentityBackfill(
	ctx context.Context,
	vaultPath string,
	identityDeps IdentityDeps,
	dryRun bool,
	fileSystem update.Filesystem,
	report *update.Report,
) error {
	stamped, backfillErr := backfillIdentity(ctx, vaultPath, identityDeps, dryRun)
	if backfillErr != nil {
		return backfillErr
	}

	report.IdentityBackfillRan = true
	report.IdentityBackfillNotesStamped = stamped

	if !dryRun {
		report.VaultHasNotesMissingIdentity = notesMissingIdentityFields(vaultPath, fileSystem)
	}

	return nil
}

// backfillIdentity finds vault notes missing repo:/user:/vault: provenance
// and stamps them, under the vault lock so it cannot race a concurrent
// learn/amend. Idempotent: already-stamped notes are left untouched, so a
// second run with nothing newly missing modifies no files. Returns the
// number of notes stamped (or, under dryRun, that would be stamped). A
// note the node edit refuses (an anchored key it would set, or a result
// that does not decode) is left untouched; every other note is still
// stamped, and the run then fails with every refusal joined, each naming
// its note (#789 design D2).
func backfillIdentity(
	ctx context.Context,
	vaultPath string,
	deps IdentityDeps,
	dryRun bool,
) (int, error) {
	release, lockErr := deps.Lock(vaultPath)
	if lockErr != nil {
		return 0, fmt.Errorf("backfill-identity: acquiring vault lock: %w", lockErr)
	}
	defer release()

	names, listErr := deps.ListMD(vaultPath)
	if listErr != nil {
		return 0, fmt.Errorf("backfill-identity: listing vault: %w", listErr)
	}

	stamped := 0
	refusals := make([]error, 0)

	for _, name := range names {
		ok, noteErr := backfillOneNote(ctx, filepath.Join(vaultPath, name), deps, dryRun)
		if errors.Is(noteErr, errFrontmatterAnchoredKey) || errors.Is(noteErr, errFrontmatterUndecodable) {
			refusals = append(refusals, fmt.Errorf("backfill-identity: refused %s: %w", name, noteErr))

			continue
		}

		if noteErr != nil {
			return stamped, noteErr
		}

		if ok {
			stamped++
		}
	}

	return stamped, errors.Join(refusals...)
}

// backfillOneNote reads one vault file, dispatches to the fact/feedback
// stamper by its type: key, and self-silences (stamped=false, err=nil) for
// anything that isn't a parseable fact/feedback note (e.g. a vocab
// definition note) — a backfill run must never fail on unrelated vault
// content.
func backfillOneNote(
	ctx context.Context,
	notePath string,
	deps IdentityDeps,
	dryRun bool,
) (bool, error) {
	raw, readErr := deps.ReadFile(notePath)
	if readErr != nil {
		return false, fmt.Errorf("backfill-identity: read %s: %w", notePath, readErr)
	}

	// Read as LF (#789 design D2): a CRLF note is converted only when
	// backfill writes it, in that same write, and its sidecar is rebuilt.
	lfRaw := toLF(raw)

	frontmatter, ok := splitFrontmatter(lfRaw)
	if !ok {
		return false, nil
	}

	var (
		rendered string
		stamped  bool
		editErr  error
	)

	switch peekNoteType(frontmatter) {
	case typeFact:
		rendered, stamped, editErr = backfillTypedNote(ctx, lfRaw, frontmatter, deps,
			func(doc *factFrontmatterDoc) noteIdentityFields {
				return noteIdentityFields{project: doc.Project, repo: &doc.Repo, user: &doc.User, vault: &doc.Vault}
			})
	case typeFeedback:
		rendered, stamped, editErr = backfillTypedNote(ctx, lfRaw, frontmatter, deps,
			func(doc *feedbackFrontmatterDoc) noteIdentityFields {
				return noteIdentityFields{project: doc.Project, repo: &doc.Repo, user: &doc.User, vault: &doc.Vault}
			})
	default:
		return false, nil
	}

	if editErr != nil || !stamped || dryRun {
		return stamped, editErr
	}

	return true, writeBackfilledNote(ctx, deps, notePath, rendered, len(lfRaw) != len(raw))
}

// backfillTypedNote computes the stamp of repo:/user:/vault: on a fact or
// feedback note whose identity is missing, as YAML-node edits on its parsed
// frontmatter (nodeEditFrontmatter, #789 design D2): every key it does not
// set — including keys the typed doc T does not define, and anchors on keys
// it does not edit — keeps its value. created: is re-emitted in its quoted
// form, as the typed re-marshal did, so a note without unknown keys is
// stamped byte-for-byte as before. Returns stamped=false, err=nil for an
// unparseable or already-stamped note (self-silencing — a detection/parse
// failure must never fail the backfill); an anchored identity key or an
// undecodable result is returned as the node edit's refusal.
func backfillTypedNote[T any](
	ctx context.Context, raw, frontmatter []byte, deps IdentityDeps, identityOf func(doc *T) noteIdentityFields,
) (string, bool, error) {
	mapping, parseErr := parseFrontmatterMapping(frontmatter)
	if parseErr != nil {
		return "", false, nil //nolint:nilerr // unparseable self-silences, same as oldVocabFilesPresent
	}

	var doc T

	decodeErr := mapping.Decode(&doc)
	if decodeErr != nil {
		return "", false, nil //nolint:nilerr // unparseable self-silences, same as oldVocabFilesPresent
	}

	fields := identityOf(&doc)
	if !identityMissing(*fields.user) {
		return "", false, nil
	}

	before := encodeNode(doc)
	identity := resolvedBackfillIdentity(ctx, fields.project, deps)

	switch {
	case *fields.vault == "":
		// Predates identity: stamp all three, as always.
		*fields.repo, *fields.user, *fields.vault = identity.Repo, identity.User, identity.Vault
	case identity.User != "":
		// First written without a detectable user: fill in only user:.
		*fields.user = identity.User
	default:
		return "", false, nil // still nothing to fill in
	}

	rendered, editErr := nodeEditFrontmatter(mapping, before, doc, string(embed.ExtractBody(raw)))
	if editErr != nil {
		return "", false, editErr
	}

	return rendered, true, nil
}

// identityMissing reports whether a note lacks user: — either it predates
// the note-origin-identity capability (user:/vault: both empty), or it was
// first written where user detection resolved empty, which omits user:
// rather than writing user: "" (#789 design D6). repo: is never checked: it
// is legitimately omitted for a non-git working directory, so checking it
// would false-positive forever on git-repo-less notes.
func identityMissing(user string) bool {
	return user == ""
}

// newIdentityDeps composes engram update --backfill-identity's dependencies
// from the injected edge Deps (pure composition — no direct I/O; #700).
func newIdentityDeps(d Deps) IdentityDeps {
	vfs := newVaultFS(d.FS)

	return IdentityDeps{
		Lock:     vaultLockFromLocker(d.Lock),
		ListMD:   vfs.ListMD,
		ReadFile: vfs.ReadFile,
		WriteFile: func(path string, data []byte) error {
			return d.FS.WriteFileAtomic(path, data, atomicFilePerm)
		},
		DetectRepo: func(ctx context.Context) string {
			return detectRepo(ctx, d.Getwd, d.Commander)
		},
		DetectUser: func(ctx context.Context) string {
			return detectUser(ctx, d.Commander, d.Username)
		},
		Getenv:   d.Getenv,
		Embedder: d.Embed,
	}
}

// notesMissingIdentityFields reports whether vaultPath holds at least one
// fact/feedback note with no repo:/user:/vault: provenance (identityMissing)
// — the signal that `engram update --backfill-identity` has genuine work to
// do. A missing/unreadable vault directory, or a note this can't parse, is
// treated as no-signal (self-silencing, same convention as
// oldVocabFilesPresent): a detection failure must never fail `engram
// update`'s primary job.
func notesMissingIdentityFields(vaultPath string, fileSystem update.Filesystem) bool {
	entries, readErr := fileSystem.ReadDir(vaultPath)
	if readErr != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		raw, fileErr := fileSystem.ReadFile(filepath.Join(vaultPath, entry.Name()))
		if fileErr != nil {
			continue
		}

		frontmatter, ok := splitFrontmatter(toLF(raw)) // a CRLF note is backfilled too (#789)
		if !ok {
			continue
		}

		var probe struct {
			User  string `yaml:"user"`
			Vault string `yaml:"vault"`
		}

		noteType := peekNoteType(frontmatter)
		if noteType != typeFact && noteType != typeFeedback {
			continue
		}

		if yaml.Unmarshal(frontmatter, &probe) == nil && identityMissing(probe.User) {
			return true
		}
	}

	return false
}

// resolvedBackfillIdentity computes the repo:/user:/vault: values a flagged
// note should be stamped with: user:/vault: from the current environment
// (safe to blind-stamp — vault-wide constants), repo: from the note's own
// project: field when non-empty, else freshly detected (design.md: a single
// backfill invocation runs from one working directory but may touch notes
// that originated in different repos, so project: is a better per-note
// signal than the invocation's cwd — unlike amend, which always uses fresh
// detection directly).
func resolvedBackfillIdentity(
	ctx context.Context,
	project string,
	deps IdentityDeps,
) identityStamp {
	return identityStamp{
		Repo:  repoWithProjectFallback(project, deps.DetectRepo(ctx)),
		User:  deps.DetectUser(ctx),
		Vault: resolveVaultName("", deps.Getenv),
	}
}

// writeBackfilledNote writes a stamped note in one atomic write and, when
// backfill converted it from CRLF, rebuilds its sidecar: conversion changes
// its content hash (#789 design D2). A nil Embedder skips the rebuild.
func writeBackfilledNote(ctx context.Context, deps IdentityDeps, notePath, rendered string, converted bool) error {
	writeErr := deps.WriteFile(notePath, []byte(rendered))
	if writeErr != nil {
		return fmt.Errorf("backfill-identity: write %s: %w", notePath, writeErr)
	}

	if !converted {
		return nil
	}

	rebuildErr := rebuildConvertedSidecar(ctx, deps.Embedder, deps.WriteFile, notePath, rendered)
	if rebuildErr != nil {
		return fmt.Errorf("backfill-identity: %w", rebuildErr)
	}

	return nil
}
