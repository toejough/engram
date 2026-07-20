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

	"github.com/toejough/engram/internal/update"
)

// UpdateArgs holds parsed flags for the update subcommand.
type UpdateArgs struct {
	DryRun       bool `targ:"flag,name=dry-run,desc=print planned actions without executing them"`
	WithGuidance bool `targ:"flag,name=with-guidance,desc=deploy guidance to .claude/engram/ for CLAUDE.md @import"`
}

// unexported constants.
const (
	emptyChunkFilesNotice = "empty chunk-index files found — run `engram prune --empty` to clear them; " +
		"see the Upgrading section in README.md\n"
	oldVocabFilePrefix   = "vocab."
	oldVocabFileSuffix   = ".md"
	vocabMigrationNotice = "old-format vocab files found — see the Upgrading section in README.md for migration steps\n"
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
	FS  update.Filesystem
	Cmd update.Commander
	Env update.Env
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

// claudeGuidanceFiles returns the guidance basenames deployed to Claude Code
// this run (empty if none / harness absent).
func claudeGuidanceFiles(report update.Report) []string {
	for _, harness := range report.Harnesses {
		if harness.Name == update.HarnessClaude {
			return harness.GuidanceFiles
		}
	}

	return nil
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
		fmt.Fprintf(buffer, "    commands/%s → %s\n", name, tildify(dst, home))
	}
}

// writeEmptyChunkHint prints a one-line pointer to the README "Upgrading"
// section when the chunk index still holds 0-byte .jsonl files. Silent
// otherwise — a vault whose index was already pruned never sees it.
func writeEmptyChunkHint(buffer *bytes.Buffer, report update.Report) {
	if report.ChunkIndexHasEmptyFiles {
		buffer.WriteString(emptyChunkFilesNotice)
	}
}

func writeGuidanceHints(buffer *bytes.Buffer, report update.Report) {
	deployed := claudeGuidanceFiles(report)

	if len(deployed) > 0 {
		for _, name := range deployed {
			if report.GuidanceImports[name] {
				fmt.Fprintf(buffer, "guidance refreshed: ~/.claude/engram/%s\n", name)

				continue
			}

			fmt.Fprintf(buffer,
				"guidance deployed to ~/.claude/engram/%s — add '@~/.claude/engram/%s'"+
					" to ~/.claude/CLAUDE.md to activate it (Claude Code will ask you to"+
					" approve the import once)\n", name, name,
			)
		}

		return
	}

	if !report.WithGuidance && !report.GuidanceImported {
		fmt.Fprintf(buffer,
			"engram ships recall- and delegation-firing guidance; run 'engram update --with-guidance' to deploy it\n",
		)
	}
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
		fmt.Fprintf(buffer, "    skills/%s/ → %s  (%d %s)\n",
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
		prefix = "[dry-run] "
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

	_, err := out.Write(buffer.Bytes())
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

// writeVocabMigrationHint prints a one-line pointer to the README
// "Upgrading" section when the vault still holds pre-tags vocab files.
// Silent otherwise — a vault that never had the old format never sees it.
func writeVocabMigrationHint(buffer *bytes.Buffer, report update.Report) {
	if report.VaultHasOldVocabFiles {
		buffer.WriteString(vocabMigrationNotice)
	}
}
