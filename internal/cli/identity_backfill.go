package cli

import (
	"context"
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

// backfillFactNote stamps repo:/user:/vault: on a fact note when its
// identity is missing, leaving every other field untouched. Returns
// stamped=false, err=nil for an unparseable or already-stamped note
// (self-silencing — a detection/parse failure must never fail the backfill).
func backfillFactNote(
	ctx context.Context, notePath string, raw, frontmatter []byte, deps IdentityDeps, dryRun bool,
) (bool, error) {
	var doc factFrontmatterDoc

	unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
	if unmarshalErr != nil || !identityMissing(doc.User, doc.Vault) {
		return false, nil //nolint:nilerr // unparseable/already-stamped self-silences, same as oldVocabFilesPresent
	}

	identity := resolvedBackfillIdentity(ctx, doc.Project, deps)
	doc.Repo, doc.User, doc.Vault = identity.Repo, identity.User, identity.Vault

	if dryRun {
		return true, nil
	}

	writeErr := deps.WriteFile(
		notePath,
		[]byte(marshalFrontmatter(doc)+string(embed.ExtractBody(raw))),
	)
	if writeErr != nil {
		return false, fmt.Errorf("backfill-identity: write %s: %w", notePath, writeErr)
	}

	return true, nil
}

// backfillFeedbackNote mirrors backfillFactNote for feedback notes.
func backfillFeedbackNote(
	ctx context.Context, notePath string, raw, frontmatter []byte, deps IdentityDeps, dryRun bool,
) (bool, error) {
	var doc feedbackFrontmatterDoc

	unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
	if unmarshalErr != nil || !identityMissing(doc.User, doc.Vault) {
		return false, nil //nolint:nilerr // unparseable/already-stamped self-silences, same as oldVocabFilesPresent
	}

	identity := resolvedBackfillIdentity(ctx, doc.Project, deps)
	doc.Repo, doc.User, doc.Vault = identity.Repo, identity.User, identity.Vault

	if dryRun {
		return true, nil
	}

	writeErr := deps.WriteFile(
		notePath,
		[]byte(marshalFrontmatter(doc)+string(embed.ExtractBody(raw))),
	)
	if writeErr != nil {
		return false, fmt.Errorf("backfill-identity: write %s: %w", notePath, writeErr)
	}

	return true, nil
}

// backfillIdentity finds vault notes missing repo:/user:/vault: provenance
// and stamps them, under the vault lock so it cannot race a concurrent
// learn/amend. Idempotent: already-stamped notes are left untouched, so a
// second run with nothing newly missing modifies no files. Returns the
// number of notes stamped (or, under dryRun, that would be stamped).
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

	for _, name := range names {
		ok, noteErr := backfillOneNote(ctx, filepath.Join(vaultPath, name), deps, dryRun)
		if noteErr != nil {
			return stamped, noteErr
		}

		if ok {
			stamped++
		}
	}

	return stamped, nil
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

	frontmatter, ok := splitFrontmatter(raw)
	if !ok {
		return false, nil
	}

	switch peekNoteType(frontmatter) {
	case typeFact:
		return backfillFactNote(ctx, notePath, raw, frontmatter, deps, dryRun)
	case typeFeedback:
		return backfillFeedbackNote(ctx, notePath, raw, frontmatter, deps, dryRun)
	default:
		return false, nil
	}
}

// identityMissing reports whether a note's user:/vault: are both empty —
// the unambiguous signal that it predates the note-origin-identity
// capability. Every note written since always gets a non-empty user: and
// vault:, even when repo: is legitimately omitted for a non-git working
// directory, so checking repo: alone would false-positive forever on
// git-repo-less notes; checking user:/vault: together never does.
func identityMissing(user, vault string) bool {
	return user == "" && vault == ""
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
		Getenv: d.Getenv,
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

		frontmatter, ok := splitFrontmatter(raw)
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

		if yaml.Unmarshal(frontmatter, &probe) == nil && identityMissing(probe.User, probe.Vault) {
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
