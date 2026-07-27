package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/toejough/engram/internal/update"
)

// UpdateArgs holds parsed flags for the update subcommand.
type UpdateArgs struct {
	DryRun       bool `targ:"flag,name=dry-run,desc=print planned actions without executing them"`
	WithGuidance bool `targ:"flag,name=with-guidance,desc=deploy guidance to .claude/engram/ for CLAUDE.md @import"`
	// RegenVocab migrates a vault holding pre-tags vocab.<term>.md /
	// vocab.index.md files to the current tags-based format (#712); honors
	// --dry-run. See regenVocab (vocab_regen.go) for the mechanism.
	RegenVocab bool `targ:"flag,name=regen-vocab,desc=migrate old-format vocab files to the current tags format"`
}

// unexported constants.
const (
	duplicateChunksNotice = "duplicate chunk-index files found — run `engram prune --duplicates` to clear them " +
		"(preview with `engram prune --duplicates --dry-run`)\n"
	emptyChunkFilesNotice = "empty chunk-index files found — run `engram prune --empty` to clear them " +
		"(preview with `engram prune --empty --dry-run`)\n"
	oldVocabFilePrefix   = "vocab."
	oldVocabFileSuffix   = ".md"
	vocabMigrationNotice = "old-format vocab files found — run `engram update --regen-vocab` to migrate them " +
		"(preview with `engram update --regen-vocab --dry-run`)\n"
)

// unexported variables.
var (
	_                      update.Env        = (*updateEnvFromDeps)(nil)
	_                      update.Filesystem = (*updateFSFromEdge)(nil)
	errSomeHarnessesFailed                   = errors.New(
		"update: one or more detected harnesses failed",
	)
)

// updateDeps carries the injected surfaces Updater.Run needs. Composed
// from the CLI-wide Deps by newUpdateDeps — pure plumbing, no I/O (#700).
type updateDeps struct {
	FS    update.Filesystem
	Cmd   update.Commander
	Env   update.Env
	Vocab VocabDeps // used only when args.RegenVocab is set (#712)
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
	data, readErr := fileSystem.ReadFile(filepath.Join(chunksDir, manifestName))
	if readErr != nil {
		return false
	}

	manifest := ingestManifest{}

	if json.Unmarshal(data, &manifest) != nil {
		return false // a malformed manifest self-silences; must never fail update
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
			Exists: func(path string) bool {
				_, statErr := fileSystem.Stat(path)

				return statErr == nil
			},
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
		return "local clone at " + tildify(report.Source.Root, home)
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

// newUpdateDeps composes update's dependency surface from cli.Deps.
func newUpdateDeps(d Deps) updateDeps {
	return updateDeps{
		FS:  &updateFSFromEdge{fs: d.FS},
		Cmd: d.Commander,
		Env: &updateEnvFromDeps{
			getenv:      d.Getenv,
			getwd:       d.Getwd,
			userHomeDir: d.UserHomeDir,
		},
		Vocab: newVocabDeps(d),
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

// runUpdate invokes Updater.Run over the injected dependency surface.
func runUpdate(ctx context.Context, args UpdateArgs, deps updateDeps, stdout io.Writer) error {
	updater := &update.Updater{
		FS:  deps.FS,
		Cmd: deps.Cmd,
		Env: deps.Env,
	}

	report, runErr := updater.Run(ctx, update.Options{
		DryRun:       args.DryRun,
		WithGuidance: args.WithGuidance,
	})
	if runErr == nil {
		vaultPath := resolveVault("", report.Home, deps.Env.Getenv)
		report.VaultHasOldVocabFiles = oldVocabFilesPresent(vaultPath, deps.FS)
		chunksDir := ResolveChunksDir("", report.Home, deps.Env.Getenv)
		report.ChunkIndexHasEmptyFiles = chunkIndexHasEmptyFiles(chunksDir, deps.FS)
		report.ChunkIndexHasPrunableDuplicates = chunkIndexHasPrunableDuplicates(chunksDir, deps.FS)

		if args.RegenVocab {
			regenErr := applyVocabRegen(ctx, vaultPath, deps.Vocab, args.DryRun, deps.FS, &report)
			if regenErr != nil {
				return fmt.Errorf("update: %w", regenErr)
			}
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

func writeCommandRows(buffer *bytes.Buffer, harness update.HarnessReport, home string) {
	if harness.CommandsRoot == "" {
		return
	}

	for _, name := range harness.CommandFiles {
		dst := filepath.Join(harness.CommandsRoot, name)
		fmt.Fprintf(buffer, "    agent-instructions/commands/%s → %s\n", name, tildify(dst, home))
	}
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
// Harnesses whose config cannot @import guidance (empty ImportsFileRel,
// e.g. OpenCode) render nothing. Paths use forward slashes always: import
// syntax is OS-independent.
func writeHarnessGuidanceHints(buffer *bytes.Buffer, report update.Report, harness update.HarnessReport) bool {
	if harness.ImportsFileRel == "" {
		return false
	}

	guidanceDir := "~/" + filepath.ToSlash(harness.GuidanceTargetRel)
	importsFile := "~/" + filepath.ToSlash(harness.ImportsFileRel)
	imported := report.GuidanceImports[harness.Name]

	for _, name := range harness.GuidanceFiles {
		if imported[name] {
			fmt.Fprintf(buffer, "guidance refreshed: %s/%s\n", guidanceDir, name)

			continue
		}

		fmt.Fprintf(buffer,
			"guidance deployed to %s/%s — add '@%s/%s' to %s to activate it"+
				" (%s will ask you to approve the import once)\n",
			guidanceDir, name, guidanceDir, name, importsFile, harness.Name,
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
		writeCommandRows(buffer, harness, report.Home)
		successes = append(successes, string(harness.Name))
	}

	return successes
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

func writeUpdateReport(out io.Writer, report update.Report) error {
	var buffer bytes.Buffer

	prefix := ""
	if report.DryRun {
		prefix = dryRunLinePrefix
	}

	fmt.Fprintf(&buffer, "%sengram update\n", prefix)
	fmt.Fprintf(&buffer, "  source: %s\n", describeSource(report, report.Home))
	fmt.Fprintf(&buffer, "  binary: %s\n", describeBinary(report))

	successes := writeHarnessSections(&buffer, report)

	if len(successes) > 0 {
		fmt.Fprintf(&buffer, "%sinstalled: %s\n", prefix, strings.Join(successes, ", "))
	}

	writeGuidanceHints(&buffer, report)
	writeVocabMigrationHint(&buffer, report)
	writeEmptyChunkHint(&buffer, report)
	writeDuplicatesHint(&buffer, report)

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

	prefix := ""
	verb := "removed"

	if report.DryRun {
		prefix = dryRunLinePrefix
		verb = "would remove"
	}

	fmt.Fprintf(buffer,
		"%supdate --regen-vocab: %s %d old-format file(s), cleaned %d member note(s), "+
			"seeded %d term(s), reassigned %d note(s)\n",
		prefix, verb, report.VocabRegenOldFilesRemoved, report.VocabRegenMembersCleaned,
		report.VocabRegenTermsSeeded, report.VocabRegenNotesAssigned)
}
