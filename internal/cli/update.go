package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/toejough/engram/internal/luhmann"
	"github.com/toejough/engram/internal/update"
)

// UpdateArgs holds parsed flags for the update subcommand.
type UpdateArgs struct {
	DryRun       bool `targ:"flag,name=dry-run,desc=print planned actions without executing them"`
	WithGuidance bool `targ:"flag,name=with-guidance,desc=deploy guidance to .claude/engram/ for CLAUDE.md @import"`
	// AllowDowngrade bypasses the local-mode provable-downgrade gate
	// (update-local-install-safety): pass it to install anyway when the
	// module root's revision is not a descendant of what's currently
	// installed (e.g. testing from a feature-branch worktree that hasn't
	// merged main's tip).
	AllowDowngrade bool `targ:"flag,name=allow-downgrade,desc=install even if it would downgrade the installed binary"`
	// RegenVocab migrates a vault holding pre-tags vocab.<term>.md /
	// vocab.index.md files to the current tags-based format (#712); honors
	// --dry-run. See regenVocab (vocab_regen.go) for the mechanism.
	RegenVocab bool `targ:"flag,name=regen-vocab,desc=migrate old-format vocab files to the current tags format"`
	// ReparentLuhmann runs the derive/answer/apply batch re-evaluation of
	// existing top-level notes (see luhmann_reparent_apply.go). No --answers:
	// derive-only, prints candidates, writes nothing. --answers <file>:
	// applies the disposition judgments (or, with --dry-run, previews them).
	// Short-circuits the rest of `engram update` entirely — this is a
	// standalone one-shot command, not a step folded into a routine refresh.
	ReparentLuhmann bool `targ:"flag,name=reparent-luhmann,desc=derive/apply batch re-parenting of top-level notes"`
	// Answers names the --answers JSON file for --reparent-luhmann's apply
	// phase (design.md Decision 2). Empty means derive-only.
	Answers string `targ:"flag,name=answers,desc=JSON answer file for --reparent-luhmann (see the emitted payload)"`
	// BackfillIdentity stamps repo:/user:/vault: provenance onto notes
	// written before the note-origin-identity capability existed (detected
	// via notesMissingIdentityFields); honors --dry-run. See backfillIdentity
	// (identity.go) for the mechanism.
	BackfillIdentity bool `targ:"flag,name=backfill-identity,desc=stamp repo:/user:/vault: provenance onto notes missing it"` //nolint:lll // unbreakable struct-tag string
}

// unexported constants.
const (
	duplicateChunksNotice = "duplicate chunk-index files found — run `engram prune --duplicates` to clear them " +
		"(preview with `engram prune --duplicates --dry-run`)\n"
	emptyChunkFilesNotice = "empty chunk-index files found — run `engram prune --empty` to clear them " +
		"(preview with `engram prune --empty --dry-run`)\n"
	identityBackfillNotice = "notes missing repo:/user:/vault: provenance found — run " +
		"`engram update --backfill-identity` to stamp them " +
		"(preview with `engram update --backfill-identity --dry-run`)\n"
	luhmannBranchingNotice = "vault holds only top-level notes — run `engram update --reparent-luhmann` " +
		"to derive and apply branching\n"
	oldVocabFilePrefix = "vocab."
	oldVocabFileSuffix = ".md"
	// topLevelLuhmannDepth is the segment count of a top-level (unbranched)
	// Luhmann ID: just the leading digit run, e.g. "12" — no letter/digit
	// branch segments appended.
	topLevelLuhmannDepth = 1
	vocabMigrationNotice = "old-format vocab files found — run `engram update --regen-vocab` to migrate them " +
		"(preview with `engram update --regen-vocab --dry-run`)\n"
	vocabSelfTagNotice = "vocab definition notes missing their vocab/<term> self-tag found — " +
		"run `engram vocab tag-definitions` to backfill them\n"
)

// unexported variables.
var (
	_                      update.Env             = (*updateEnvFromDeps)(nil)
	_                      update.Filesystem      = (*updateFSFromEdge)(nil)
	_                      update.HandoffReporter = handoffReportWriter{}
	errSomeHarnessesFailed                        = errors.New(
		"update: one or more detected harnesses failed",
	)
)

// handoffReportWriter adapts writeReexecHandoffReport to
// update.HandoffReporter — the production Updater.Handoff (design D6/D8).
type handoffReportWriter struct {
	stdout io.Writer
}

// WriteHandoff writes the parent's install-result header to stdout, before
// Updater.Run spawns the re-exec child: the child inherits this same
// stdout and starts writing its own sync/check report strictly after,
// giving "install result once, first; sync report once, from the child" —
// never interleaved or reversed.
func (h handoffReportWriter) WriteHandoff(report update.Report) error {
	return writeReexecHandoffReport(h.stdout, report)
}

// updateDeps carries the injected surfaces Updater.Run needs. Composed
// from the CLI-wide Deps by newUpdateDeps — pure plumbing, no I/O (#700).
type updateDeps struct {
	FS    update.Filesystem
	Cmd   update.Commander
	Env   update.Env
	Spawn update.Spawner // re-exec's post-install spawn (design D1); required — never nil in production
	// Exit terminates the process with a status code (production: os.Exit,
	// via cli.Deps.Exit). runUpdate calls it with the re-execed child's
	// exit code BEFORE the vault/vocab/chunk-check block, so the parent
	// never runs those checks after handing off (design D8).
	Exit     func(int)
	Vocab    VocabDeps    // used only when args.RegenVocab is set (#712)
	Reparent ReparentDeps // used only when args.ReparentLuhmann is set
	Identity IdentityDeps // used only when args.BackfillIdentity is set
}

// updateEnvFromDeps adapts cli.Deps' env funcs to update.Env.
type updateEnvFromDeps struct {
	getenv      func(string) string
	getwd       func() (string, error)
	userHomeDir func() (string, error)
}

func (e *updateEnvFromDeps) Getenv(key string) string { return e.getenv(key) }

func (e *updateEnvFromDeps) Getwd() (string, error) { return e.getwd() }

func (e *updateEnvFromDeps) UserHomeDir() (string, error) { return e.userHomeDir() }

// updateFSFromEdge adapts the CLI-wide EdgeFS to update.Filesystem. Pure
// interface plumbing: fs.DirEntry / fs.FileInfo structurally satisfy
// update.DirEntry / update.FileInfo. Errors pass through unwrapped so
// errors.Is(err, fs.ErrNotExist) checks in the update package keep working.
type updateFSFromEdge struct {
	fs EdgeFS
}

func (a *updateFSFromEdge) Lstat(path string) (update.FileInfo, error) {
	info, err := a.fs.Lstat(path)
	if err != nil {
		// Caller distinguishes fs.ErrNotExist via errors.Is.
		return nil, err
	}

	return info, nil
}

func (a *updateFSFromEdge) MkdirAll(path string, perm fs.FileMode) error {
	return a.fs.MkdirAll(path, perm) // pass-through; update core adds context
}

func (a *updateFSFromEdge) ReadDir(path string) ([]update.DirEntry, error) {
	entries, err := a.fs.ReadDir(path)
	if err != nil {
		// Caller distinguishes fs.ErrNotExist via errors.Is.
		return nil, err
	}

	out := make([]update.DirEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry)
	}

	return out, nil
}

func (a *updateFSFromEdge) ReadFile(path string) ([]byte, error) {
	data, err := a.fs.ReadFile(path)
	if err != nil {
		// Caller distinguishes fs.ErrNotExist via errors.Is.
		return nil, err
	}

	return data, nil
}

func (a *updateFSFromEdge) ReadLink(path string) (string, error) {
	target, err := a.fs.Readlink(path)
	if err != nil {
		// Caller distinguishes fs.ErrNotExist via errors.Is.
		return "", err
	}

	return target, nil
}

func (a *updateFSFromEdge) RemoveAll(path string) error {
	return a.fs.RemoveAll(path) // pass-through; update core adds context
}

func (a *updateFSFromEdge) Stat(path string) (update.FileInfo, error) {
	info, err := a.fs.Stat(path)
	if err != nil {
		// Caller distinguishes fs.ErrNotExist via errors.Is.
		return nil, err
	}

	return info, nil
}

func (a *updateFSFromEdge) Symlink(target, link string) error {
	return a.fs.Symlink(target, link) // pass-through; update core adds context
}

func (a *updateFSFromEdge) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return a.fs.WriteFile(path, data, perm) // pass-through; update core adds context
}

func anyHarnessFailed(report update.Report) bool {
	return slices.ContainsFunc(report.Harnesses, harnessFailed)
}

// applyVocabRegen runs regenVocab over vaultPath and copies its result onto
// report (the cli-layer-only fields documented on update.Report). After a
// successful non-dry-run regen it also re-checks VaultHasOldVocabFiles, so a
// subsequent `engram update` (without --regen-vocab) prints no notice.
func applyVocabRegen(
	ctx context.Context,
	vaultPath string,
	vocabDeps VocabDeps,
	dryRun bool,
	fileSystem update.Filesystem,
	report *update.Report,
) error {
	regenResult, regenErr := regenVocab(ctx, vaultPath, vocabDeps, dryRun)
	if regenErr != nil {
		return regenErr
	}

	report.VocabRegenRan = true
	report.VocabRegenOldFilesRemoved = regenResult.OldFilesRemoved
	report.VocabRegenMembersCleaned = regenResult.MembersCleaned
	report.VocabRegenTermsSeeded = regenResult.TermsSeeded
	report.VocabRegenNotesAssigned = regenResult.NotesAssigned

	if !dryRun {
		report.VaultHasOldVocabFiles = oldVocabFilesPresent(vaultPath, fileSystem)
	}

	return nil
}

// chunkIndexHasEmptyFiles reports whether the chunk index holds any 0-byte
// .jsonl file (the backlog older versions accreted before the rebuildIndex
// guard — #694). It scans through the injected filesystem seam and returns
// false for a missing/unreadable dir, so fresh or already-pruned indexes stay
// silent. Empties are detected by len==0 (the seam exposes no file size),
// early-returning on the first one found.
func chunkIndexHasEmptyFiles(chunksDir string, fileSystem update.Filesystem) bool {
	entries, readErr := fileSystem.ReadDir(chunksDir)
	if readErr != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), jsonlExt) {
			continue
		}

		data, fileErr := fileSystem.ReadFile(filepath.Join(chunksDir, entry.Name()))
		if fileErr == nil && len(data) == 0 {
			return true
		}
	}

	return false
}

// chunkIndexHasPrunableDuplicates reports whether `engram prune
// --duplicates` would actually remove anything right now: it re-runs
// prune's own reconciliation in dry-run mode (reconcileDuplicateGroups —
// the same (FileHash, chunkingClass) grouping, canonical selection, and
// record-level coverage gate), true only when at least one duplicate
// would be removed. A backlog whose every group would be REFUSED stays
// silent (#713) — refusals are pending manual review by design; prune
// and ingest explain them when run. Detection stays stateless and
// notify-only: update never removes anything itself (Unit 5's
// detect-and-notify contract holds).
//
// Cost: one manifest read + decode always; the coverage gate's
// per-source .jsonl reads happen only for hash groups with >= 2 live
// members, so an index with no duplicate backlog does no per-source
// I/O. A missing/unreadable/malformed manifest is false (self-silencing,
// same convention as chunkIndexHasEmptyFiles) — a detection failure must
// never fail `engram update`'s primary job.
func chunkIndexHasPrunableDuplicates(chunksDir string, fileSystem update.Filesystem) bool {
	manifest, readErr := readChunkManifest(chunksDir, fileSystem.ReadFile)
	if readErr != nil {
		return false // missing/unreadable/malformed manifest self-silences; must never fail update
	}

	// DryRun guarantees reconcileDuplicateGroups never removes or writes:
	// pruneOneDuplicate short-circuits before removeDuplicateIndex, so
	// PruneDeps.Remove is never invoked (left nil deliberately — a call
	// would be a bug, and failing loud beats silently deleting). Refusal
	// detail lines go to io.Discard; update only needs the counts.
	counts := reconcileDuplicateGroups(
		PruneArgs{ChunksDir: chunksDir, DryRun: true},
		manifest,
		PruneDeps{
			ReadFile: fileSystem.ReadFile,
			Exists:   func(path string) bool { return statExists(fileSystem.Stat, path) },
		},
		io.Discard,
	)

	return counts.removed > 0
}

func describeBinary(report update.Report) string {
	if report.DryRun {
		return report.GoInstall
	}

	suffix := "engram"
	if report.BinaryVersion != "" {
		suffix = "engram " + report.BinaryVersion
	}

	return fmt.Sprintf("%s ... ok (%s → %s)",
		report.GoInstall, suffix, tildify(report.BinaryPath, report.Home))
}

func describeSource(report update.Report, home string) string {
	switch report.Source.Mode {
	case update.SourceLocal:
		desc := "local clone at " + tildify(report.Source.Root, home)
		if report.Source.Version != "" {
			desc += " (rev " + report.Source.Version + ")"
		}

		return desc
	case update.SourceRemote:
		return "remote module " + update.ModulePath + " " + report.Source.Version
	default:
		return "unknown"
	}
}

// finishUpdate is the pure-decision tail of runUpdate, broken out for tests.
func finishUpdate(stdout io.Writer, report update.Report, runErr error) error {
	if runErr != nil {
		return fmt.Errorf("update: %w", runErr)
	}

	writeErr := writeUpdateReport(stdout, report)
	if writeErr != nil {
		return fmt.Errorf("update: writing report: %w", writeErr)
	}

	if anyHarnessFailed(report) {
		return errSomeHarnessesFailed
	}

	return nil
}

func harnessFailed(harness update.HarnessReport) bool { return harness.Err != nil }

// newRenameRewriteDeps composes RenameRewriteDeps from the injected edge Deps
// (pure composition — no direct I/O; #700).
func newRenameRewriteDeps(d Deps) RenameRewriteDeps {
	vfs := newVaultFS(d.FS)

	return RenameRewriteDeps{
		ListMD:   vfs.ListMD,
		ReadFile: vfs.ReadFile,
		WriteFile: func(path string, data []byte) error {
			return d.FS.WriteFileAtomic(path, data, vocabNotePerm)
		},
		Rename: d.FS.Rename,
	}
}

// newUpdateDeps composes update's dependency surface from cli.Deps.
func newUpdateDeps(d Deps) updateDeps {
	return updateDeps{
		FS:    &updateFSFromEdge{fs: d.FS},
		Cmd:   d.Commander,
		Spawn: d.Spawner,
		Env: &updateEnvFromDeps{
			getenv:      d.Getenv,
			getwd:       d.Getwd,
			userHomeDir: d.UserHomeDir,
		},
		Exit:  d.Exit,
		Vocab: newVocabDeps(d),
		Reparent: ReparentDeps{
			Rename: newRenameRewriteDeps(d),
			Ingest: newIngestDeps(d),
			Prune:  newPruneDeps(d),
		},
		Identity: newIdentityDeps(d),
	}
}

// oldVocabFilesPresent reports whether vaultPath still holds pre-tags vocab
// files (vocab.<term>.md term notes, vocab.index.md) — the signal that a
// vault predates the 2026-07-10 vocab→tags migration (#678). A missing or
// unreadable vault directory is treated as false (self-silencing for fresh
// installs); the underlying ReadDir error is never surfaced.
// The ".md" suffix guard is load-bearing: vocab.centroids.json is a current,
// always-present vault file sharing the "vocab." prefix — it must NOT match.
func oldVocabFilesPresent(vaultPath string, fileSystem update.Filesystem) bool {
	entries, readErr := fileSystem.ReadDir(vaultPath)
	if readErr != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasPrefix(name, oldVocabFilePrefix) && strings.HasSuffix(name, oldVocabFileSuffix) {
			return true
		}
	}

	return false
}

func pluralFile(n int) string {
	if n == 1 {
		return "file"
	}

	return "files"
}

// reexecArgsFrom reconstructs the flags to pass to a re-execed `engram
// update` invocation from the parsed args of THIS invocation (design D2:
// "preserve the original update args"). --dry-run is deliberately excluded:
// Updater.Run never re-execs on a dry run in the first place (resolveSource
// skips install), so the flag would never reach here regardless.
func reexecArgsFrom(args UpdateArgs) []string {
	//nolint:mnd // WithGuidance + RegenVocab + BackfillIdentity, the only re-execable flags
	reexecArgs := make([]string, 0, 3)

	if args.WithGuidance {
		reexecArgs = append(reexecArgs, "--with-guidance")
	}

	if args.RegenVocab {
		reexecArgs = append(reexecArgs, "--regen-vocab")
	}

	if args.BackfillIdentity {
		reexecArgs = append(reexecArgs, "--backfill-identity")
	}

	return reexecArgs
}

// runPostUpdateChecks runs the vault/vocab/chunk-check detector battery and
// the opt-in --regen-vocab/--backfill-identity actions, mutating report in
// place. Split out of runUpdate to keep the "belongs to whichever process
// actually ran the sync phase" block (see runUpdate's D8 comment) within the
// house per-function length/nesting budget.
func runPostUpdateChecks(ctx context.Context, args UpdateArgs, deps updateDeps, report *update.Report) error {
	vaultPath := resolveVault("", report.Home, deps.Env.Getenv)
	report.VaultHasOldVocabFiles = oldVocabFilesPresent(vaultPath, deps.FS)
	report.VaultHasUntaggedVocabDefinitions = vocabDefinitionsMissingSelfTags(vaultPath, deps.FS)
	report.VaultHasOnlyTopLevelNotes = vaultHasOnlyTopLevelNotes(vaultPath, deps.FS)
	chunksDir := ResolveChunksDir("", report.Home, deps.Env.Getenv)
	report.ChunkIndexHasEmptyFiles = chunkIndexHasEmptyFiles(chunksDir, deps.FS)
	report.ChunkIndexHasPrunableDuplicates = chunkIndexHasPrunableDuplicates(chunksDir, deps.FS)
	report.VaultHasNotesMissingIdentity = notesMissingIdentityFields(vaultPath, deps.FS)
	report.VaultHasPendingOffers = vaultHasPendingOffers(vaultPath, deps.FS)

	if args.RegenVocab {
		regenErr := applyVocabRegen(ctx, vaultPath, deps.Vocab, args.DryRun, deps.FS, report)
		if regenErr != nil {
			return fmt.Errorf("update: %w", regenErr)
		}
	}

	if args.BackfillIdentity {
		backfillErr := applyIdentityBackfill(ctx, vaultPath, deps.Identity, args.DryRun, deps.FS, report)
		if backfillErr != nil {
			return fmt.Errorf("update: %w", backfillErr)
		}
	}

	return nil
}

// runUpdate invokes Updater.Run over the injected dependency surface.
func runUpdate(ctx context.Context, args UpdateArgs, deps updateDeps, stdout io.Writer) error {
	// --reparent-luhmann is a standalone one-shot command (design.md): it
	// short-circuits the rest of `engram update` (binary install, harness
	// sync, vocab/chunk checks) entirely rather than folding in as an extra
	// step the way --regen-vocab does.
	if args.ReparentLuhmann {
		home, homeErr := deps.Env.UserHomeDir()
		if homeErr != nil {
			return fmt.Errorf("update --reparent-luhmann: resolving home: %w", homeErr)
		}

		vaultPath := resolveVault("", home, deps.Env.Getenv)
		chunksDir := ResolveChunksDir("", home, deps.Env.Getenv)

		reparentErr := RunReparentLuhmann(ctx, vaultPath, chunksDir, args.Answers, args.DryRun, deps.Reparent, stdout)
		if reparentErr != nil {
			return fmt.Errorf("update: %w", reparentErr)
		}

		return nil
	}

	updater := &update.Updater{
		FS:      deps.FS,
		Cmd:     deps.Cmd,
		Env:     deps.Env,
		Spawn:   deps.Spawn,
		Handoff: handoffReportWriter{stdout: stdout},
	}

	report, runErr := updater.Run(ctx, update.Options{
		DryRun:         args.DryRun,
		WithGuidance:   args.WithGuidance,
		AllowDowngrade: args.AllowDowngrade,
		ReexecArgs:     reexecArgsFrom(args),
	})

	// D8: a non-nil ReexecExitCode means Run handed off to a re-execed
	// child that already ran sync/checks — this parent process must exit
	// with the child's code WITHOUT running the vault/vocab/chunk-check
	// block below (that block belongs to whichever process actually ran
	// the sync phase, and here that was the child, not us). The parent's
	// own report was already written by Handoff.WriteHandoff above, before
	// the child ever started — nothing left to print here.
	if report.ReexecExitCode != nil {
		deps.Exit(*report.ReexecExitCode)

		return nil
	}

	if runErr == nil {
		checksErr := runPostUpdateChecks(ctx, args, deps, &report)
		if checksErr != nil {
			return checksErr
		}
	}

	return finishUpdate(stdout, report, runErr)
}

// tildify replaces a leading home path with "~" for spec-style output.
func tildify(path, home string) string {
	if home == "" || !strings.HasPrefix(path, home) {
		return path
	}

	return "~" + strings.TrimPrefix(path, home)
}

// vaultHasOnlyTopLevelNotes reports whether vaultPath holds at least one
// note AND every note's Luhmann ID is top-level (depth 1 — no letter/digit
// branch segments) — the signal that `engram update --reparent-luhmann`
// (derive/answer/apply) has genuine work to consider. A missing or
// unreadable vault directory, or an empty vault, is treated as false
// (self-silencing, same convention as oldVocabFilesPresent): a detection
// failure must never fail `engram update`'s primary job. Entries that
// idAndDateFromNoteFilename rejects (non-note files, e.g. .vec.json
// sidecars) are skipped. An unparseable Luhmann ID is treated as
// non-top-level (conservative: never claim "flat" on a malformed vault).
func vaultHasOnlyTopLevelNotes(vaultPath string, fileSystem update.Filesystem) bool {
	entries, readErr := fileSystem.ReadDir(vaultPath)
	if readErr != nil {
		return false
	}

	foundNote := false

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		id, _, ok := idAndDateFromNoteFilename(entry.Name())
		if !ok {
			continue
		}

		foundNote = true

		segments, parseErr := luhmann.ParseID(id)
		if parseErr != nil || len(segments) != topLevelLuhmannDepth {
			return false
		}
	}

	return foundNote
}

// vocabDefinitionsMissingSelfTags reports whether vaultPath holds at least
// one vocab definition note missing its vocab/<term> self-tag — the signal
// that the vault predates the vocab-definition-self-tags change (4f68fada),
// for which `engram vocab tag-definitions` is the idempotent backfill. A
// definition note is one whose slug parses to a term
// (termFromDefinitionSlug — the family note's "vocab-definition" slug
// deliberately does NOT, keeping it exempt: it stays bare-vocab-only by
// design) AND whose tags carry the bare vocab marker (isVocabDefinitionNote
// — the same both-sides gate tag-definitions itself uses). A missing or
// unreadable vault directory, unreadable note, or unparseable frontmatter is
// treated as no-signal (self-silencing, same convention as
// oldVocabFilesPresent): a detection failure must never fail `engram
// update`'s primary job. Fully tagged vaults return false, so
// idempotent-clean vaults print nothing.
func vocabDefinitionsMissingSelfTags(vaultPath string, fileSystem update.Filesystem) bool {
	entries, readErr := fileSystem.ReadDir(vaultPath)
	if readErr != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		term, ok := termFromDefinitionSlug(slugFromNoteFilename(entry.Name()))
		if !ok {
			continue // family note, member notes, non-.md files
		}

		raw, fileErr := fileSystem.ReadFile(filepath.Join(vaultPath, entry.Name()))
		if fileErr != nil || len(raw) == 0 || !isVocabDefinitionNote(string(raw)) {
			continue
		}

		frontmatter, _, splitOK := splitFrontmatterAndBody(string(raw))
		if !splitOK {
			continue
		}

		if !slices.Contains(parseTagsFromFrontmatter(frontmatter), vocabTagPrefix+term) {
			return true
		}
	}

	return false
}

// writeDuplicatesHint prints a one-line notice naming `engram prune
// --duplicates` when the chunk index holds a duplicate backlog that
// command would actually remove (refusal-only backlogs stay silent —
// #713). Deliberately just a notice: update never removes anything on
// the user's behalf here.
func writeDuplicatesHint(buffer *bytes.Buffer, report update.Report) {
	if report.ChunkIndexHasPrunableDuplicates {
		buffer.WriteString(duplicateChunksNotice)
	}
}

// writeEmptyChunkHint prints a one-line notice naming `engram prune --empty`
// when the chunk index still holds 0-byte .jsonl files. Silent otherwise — a
// vault whose index was already pruned never sees it.
func writeEmptyChunkHint(buffer *bytes.Buffer, report update.Report) {
	if report.ChunkIndexHasEmptyFiles {
		buffer.WriteString(emptyChunkFilesNotice)
	}
}

// writeEngramRootNotice renders one harness's engram-root sync findings
// (D2/D4/D5/D6): every path the sync engine deleted (dry-run-prefixed, since
// nothing was actually removed under --dry-run — an action line), a
// refusal notice when the root exists without its `.engram-owned` marker,
// any unattributable root-level files found there, every surface path
// adopted (task 5.1, dry-run-prefixed — a preview under --dry-run), every
// surface stray found outside the intended set (task 5.2 — never deleted,
// listed here for manual review per the spec), and every dangling engram
// symlink cleaned up (task 4.2/D5, dry-run-prefixed — a preview under
// --dry-run). Silent when the harness has nothing to report.
func writeEngramRootNotice(buffer *bytes.Buffer, report update.Report, harness update.HarnessReport) {
	prefix := dryRunPrefix(report.DryRun)

	for _, rel := range harness.EngramSyncDeleted {
		fmt.Fprintf(buffer, "    %sengram root: deleted %s\n", prefix, filepath.ToSlash(rel))
	}

	if harness.EngramDeletionRefused {
		fmt.Fprintf(buffer,
			"    engram root %s exists without the .engram-owned marker — sync-deletion refused there\n",
			tildify(harness.EngramRoot, report.Home),
		)
	}

	for _, name := range harness.EngramUnattributable {
		fmt.Fprintf(buffer, "    unattributable: %s\n", name)
	}

	for _, path := range harness.EngramAdopted {
		fmt.Fprintf(buffer, "    %sadopted: %s\n", prefix, tildify(path, report.Home))
	}

	for _, path := range harness.SurfaceUnmanaged {
		fmt.Fprintf(buffer, "    unmanaged (left alone): %s\n", tildify(path, report.Home))
	}

	for _, path := range harness.DanglingLinksRemoved {
		fmt.Fprintf(buffer, "    %scleanup: removed dangling link %s\n", prefix, tildify(path, report.Home))
	}
}

// writeGuidanceHints renders guidance deploy/wiring status per harness from
// its spec-derived paths, then — when nothing was deployed anywhere and the
// user has neither opted in nor imported — a one-line nudge.
func writeGuidanceHints(buffer *bytes.Buffer, report update.Report) {
	anyDeployed := false

	for _, harness := range report.Harnesses {
		if writeHarnessGuidanceHints(buffer, report, harness) {
			anyDeployed = true
		}
	}

	if anyDeployed || report.WithGuidance || report.GuidanceImported {
		return
	}

	buffer.WriteString(
		"engram ships recall- and delegation-firing guidance; run 'engram update --with-guidance' to deploy it\n",
	)
}

// writeHarnessGuidanceHints renders one harness's guidance lines — already
// wired files as "refreshed", the rest with the @import line to add to the
// harness's own config file — and reports whether anything was deployed.
// Both lines are action lines (they describe a guidance file that was, or
// under --dry-run WOULD BE, written/relinked — #709) and so carry the
// dry-run prefix like every other action line in this report. Harnesses
// whose config cannot @import guidance (empty ImportsFileRel) render
// nothing. Paths use forward slashes always: import syntax
// is OS-independent. When the harness's guidance surface IS its engram root
// (Claude today — D1's guidance caveat), each line also states the
// canonical guidance/ path (task 5.3): the flat @import path is now a
// compat symlink, not where the real content lives, and the report must say
// so.
func writeHarnessGuidanceHints(buffer *bytes.Buffer, report update.Report, harness update.HarnessReport) bool {
	if harness.ImportsFileRel == "" {
		return false
	}

	prefix := dryRunPrefix(report.DryRun)
	guidanceDir := "~/" + filepath.ToSlash(harness.GuidanceTargetRel)
	importsFile := "~/" + filepath.ToSlash(harness.ImportsFileRel)
	imported := report.GuidanceImports[harness.Name]
	canonicalRoot := filepath.Join(report.Home, harness.GuidanceTargetRel) == harness.EngramRoot

	for _, name := range harness.GuidanceFiles {
		canonical := ""
		if canonicalRoot {
			canonical = fmt.Sprintf(" (canonical: %s/guidance/%s)", guidanceDir, name)
		}

		if imported[name] {
			fmt.Fprintf(buffer, "%sguidance refreshed: %s/%s%s\n", prefix, guidanceDir, name, canonical)

			continue
		}

		fmt.Fprintf(buffer,
			"%sguidance deployed to %s/%s — add '@%s/%s' to %s to activate it"+
				" (%s will ask you to approve the import once)%s\n",
			prefix, guidanceDir, name, guidanceDir, name, importsFile, harness.Name, canonical,
		)
	}

	return len(harness.GuidanceFiles) > 0
}

func writeHarnessSections(buffer *bytes.Buffer, report update.Report) []string {
	successes := make([]string, 0, len(report.Harnesses))

	for _, harness := range report.Harnesses {
		fmt.Fprintf(
			buffer,
			"  %s (%s):\n",
			harness.Name,
			tildify(
				filepath.Join(report.Home, harness.ProbeRoot)+string(filepath.Separator),
				report.Home,
			),
		)

		if harness.Err != nil {
			fmt.Fprintf(buffer, "    error: %v\n", harness.Err)

			continue
		}

		writeSkillRows(buffer, harness, report.Home)
		writeEngramRootNotice(buffer, report, harness)
		successes = append(successes, string(harness.Name))
	}

	return successes
}

// writeIdentityBackfillHint prints the `engram update --backfill-identity`
// notice when the vault holds notes missing repo:/user:/vault: provenance,
// or — when this run just backfilled — the summary instead (never both, so
// a run that just acted never repeats the notice it acted on).
func writeIdentityBackfillHint(buffer *bytes.Buffer, report update.Report) {
	if report.IdentityBackfillRan {
		writeIdentityBackfillReport(buffer, report)

		return
	}

	if report.VaultHasNotesMissingIdentity {
		buffer.WriteString(identityBackfillNotice)
	}
}

// writeIdentityBackfillReport renders the --backfill-identity summary: a
// one-line no-op notice when there was nothing to backfill, else a
// stamped/would-stamp line (prefixed "[dry-run]" and phrased "would stamp"
// for a dry run) with the note count.
func writeIdentityBackfillReport(buffer *bytes.Buffer, report update.Report) {
	if report.IdentityBackfillNotesStamped == 0 {
		buffer.WriteString("update --backfill-identity: nothing to backfill\n")

		return
	}

	prefix := dryRunPrefix(report.DryRun)
	verb := "stamped"

	if report.DryRun {
		verb = "would stamp"
	}

	fmt.Fprintf(buffer, "%supdate --backfill-identity: %s %d note(s) with repo:/user:/vault: provenance\n",
		prefix, verb, report.IdentityBackfillNotesStamped)
}

// writeLuhmannBranchingNotice prints a one-line notice naming `engram update
// --reparent-luhmann` when the vault holds only top-level (unbranched)
// notes. Silent otherwise — a vault that already has branched notes never
// sees it. Deliberately just a notice: update never renames or rewrites
// vault notes on the user's behalf here.
func writeLuhmannBranchingNotice(buffer *bytes.Buffer, report update.Report) {
	if report.VaultHasOnlyTopLevelNotes {
		buffer.WriteString(luhmannBranchingNotice)
	}
}

// writePendingOfferHint prints a one-line notice naming the pending_offers
// query flag when the vault holds at least one pending-offer note awaiting
// curation. Silent otherwise. Deliberately just a notice: update never
// curates offers itself — that's the offer-curation skill's job, off this
// process entirely.
func writePendingOfferHint(buffer *bytes.Buffer, report update.Report) {
	if report.VaultHasPendingOffers {
		buffer.WriteString(pendingOfferUpdateNotice)
	}
}

// writeReexecHandoffReport renders the parent's contribution to the
// combined report after a successful re-exec handoff (design D6): source
// and binary lines only — never the harness/guidance/vocab/chunk sections,
// which belong to the re-execed child's own writeUpdateReport call. This
// keeps "install output in the parent, one sync/check report in the child"
// true by construction rather than by suppressing fields.
func writeReexecHandoffReport(out io.Writer, report update.Report) error {
	var buffer bytes.Buffer

	writeUpdateHeader(&buffer, report, "")

	_, err := out.Write(buffer.Bytes())
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

func writeSkillRows(buffer *bytes.Buffer, harness update.HarnessReport, home string) {
	for _, dirCount := range harness.SkillDirs {
		dst := filepath.Join(harness.SkillsRoot, dirCount.Name) + string(filepath.Separator)
		fmt.Fprintf(buffer, "    agent-instructions/skills/%s/ → %s  (%d %s)\n",
			dirCount.Name,
			tildify(dst, home),
			dirCount.Files,
			pluralFile(dirCount.Files),
		)
	}
}

// writeUpdateHeader writes the three-line header common to both full and
// re-exec handoff reports: title, source, and binary lines. prefix is
// prepended to the title line (empty for handoff, "[dry-run] " for full
// reports under --dry-run).
func writeUpdateHeader(buffer *bytes.Buffer, report update.Report, prefix string) {
	fmt.Fprintf(buffer, "%sengram update\n", prefix)
	fmt.Fprintf(buffer, "  source: %s\n", describeSource(report, report.Home))
	fmt.Fprintf(buffer, "  binary: %s\n", describeBinary(report))
}

func writeUpdateReport(out io.Writer, report update.Report) error {
	var buffer bytes.Buffer

	prefix := dryRunPrefix(report.DryRun)

	// ReexecChild: this run's report is the re-execed CHILD's — the parent
	// already wrote the header (source/binary lines) via Handoff.WriteHandoff
	// before spawning us, and this run never performed the install those
	// lines describe. Printing it again here would both duplicate the
	// header (defect: two "engram update" lines) and misattribute an
	// install this run never ran (defect: child claiming "binary: go
	// install ... ok").
	if !report.ReexecChild {
		writeUpdateHeader(&buffer, report, prefix)
	}

	if report.ReexecFallbackErr != "" {
		fmt.Fprintf(&buffer, "  %s\n", report.ReexecFallbackErr)
	}

	successes := writeHarnessSections(&buffer, report)

	if len(successes) > 0 {
		fmt.Fprintf(&buffer, "%sinstalled: %s\n", prefix, strings.Join(successes, ", "))
	}

	writeGuidanceHints(&buffer, report)
	writeVocabMigrationHint(&buffer, report)
	writeVocabSelfTagHint(&buffer, report)
	writeLuhmannBranchingNotice(&buffer, report)
	writeEmptyChunkHint(&buffer, report)
	writeDuplicatesHint(&buffer, report)
	writeIdentityBackfillHint(&buffer, report)
	writePendingOfferHint(&buffer, report)

	_, err := out.Write(buffer.Bytes())
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

// writeVocabMigrationHint prints a one-line notice naming `engram update
// --regen-vocab` when the vault still holds pre-tags vocab files. Silent
// otherwise — a vault that never had the old format never sees it.
func writeVocabMigrationHint(buffer *bytes.Buffer, report update.Report) {
	if report.VocabRegenRan {
		writeVocabRegenReport(buffer, report)

		return
	}

	if report.VaultHasOldVocabFiles {
		buffer.WriteString(vocabMigrationNotice)
	}
}

// writeVocabRegenReport renders the --regen-vocab summary: a one-line
// no-op notice when there was nothing to regenerate, else a removed/would-
// remove line (prefixed "[dry-run]" and phrased "would remove" for a dry
// run) with the file/member/note counts.
func writeVocabRegenReport(buffer *bytes.Buffer, report update.Report) {
	if report.VocabRegenOldFilesRemoved == 0 && report.VocabRegenMembersCleaned == 0 {
		buffer.WriteString("update --regen-vocab: nothing to regenerate\n")

		return
	}

	prefix := dryRunPrefix(report.DryRun)
	verb := "removed"

	if report.DryRun {
		verb = "would remove"
	}

	fmt.Fprintf(buffer,
		"%supdate --regen-vocab: %s %d old-format file(s), cleaned %d member note(s), "+
			"seeded %d term(s), reassigned %d note(s)\n",
		prefix, verb, report.VocabRegenOldFilesRemoved, report.VocabRegenMembersCleaned,
		report.VocabRegenTermsSeeded, report.VocabRegenNotesAssigned)
}

// writeVocabSelfTagHint prints a one-line notice naming `engram vocab
// tag-definitions` when the vault holds definition notes missing their
// vocab/<term> self-tag (pre-4f68fada vaults). Silent otherwise — a fully
// tagged vault never sees it (the backfill is idempotent-clean). Deliberately
// just a notice: update never rewrites vault notes on the user's behalf here.
func writeVocabSelfTagHint(buffer *bytes.Buffer, report update.Report) {
	if report.VaultHasUntaggedVocabDefinitions {
		buffer.WriteString(vocabSelfTagNotice)
	}
}
