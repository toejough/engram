package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
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
	oldVocabFilePrefix   = "vocab."
	oldVocabFileSuffix   = ".md"
	vocabMigrationNotice = "old-format vocab files found — see the Upgrading section in README.md for migration steps\n"
)

// unexported variables.
var (
	_                      update.Filesystem = (*osUpdateFS)(nil)
	errSomeHarnessesFailed                   = errors.New(
		"update: one or more detected harnesses failed",
	)
)

type osCommander struct{}

func (*osCommander) Run(
	ctx context.Context, dir, name string, args ...string,
) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // name/args from internal callers
	cmd.Dir = dir
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	if err != nil {
		return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("%s %v: %w", name, args, err)
	}

	return stdout.Bytes(), stderr.Bytes(), nil
}

type osDirEntry struct{ entry fs.DirEntry }

func (o *osDirEntry) IsDir() bool { return o.entry.IsDir() }

func (o *osDirEntry) Name() string { return o.entry.Name() }

type osFileInfo struct{ info fs.FileInfo }

func (o *osFileInfo) IsDir() bool { return o.info.IsDir() }

type osUpdateEnv struct{}

func (*osUpdateEnv) Getenv(key string) string {
	return os.Getenv(key)
}

func (*osUpdateEnv) Getwd() (string, error) {
	return os.Getwd() //nolint:wrapcheck // adapter; caller wraps with context
}

func (*osUpdateEnv) UserHomeDir() (string, error) {
	return os.UserHomeDir() //nolint:wrapcheck // adapter; caller wraps with context
}

// --- production adapters --------------------------------------------------

type osUpdateFS struct{}

func (*osUpdateFS) MkdirAll(path string, perm fs.FileMode) error {
	err := os.MkdirAll(path, perm)
	if err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	return nil
}

func (*osUpdateFS) ReadDir(path string) ([]update.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		//nolint:wrapcheck // caller distinguishes fs.ErrNotExist via errors.Is
		return nil, err
	}

	out := make([]update.DirEntry, 0, len(entries))

	for _, entry := range entries {
		out = append(out, &osDirEntry{entry: entry})
	}

	return out, nil
}

func (*osUpdateFS) ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path supplied by walker
	if err != nil {
		//nolint:wrapcheck // adapter; caller adds context
		return nil, err
	}

	return data, nil
}

func (*osUpdateFS) RemoveAll(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("remove: %w", err)
	}

	return nil
}

func (*osUpdateFS) Stat(path string) (update.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		//nolint:wrapcheck // thin I/O adapter; caller distinguishes fs.ErrNotExist via errors.Is
		return nil, err
	}

	return &osFileInfo{info: info}, nil
}

func (*osUpdateFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	err := os.WriteFile(path, data, perm)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

func anyHarnessFailed(report update.Report) bool {
	return slices.ContainsFunc(report.Harnesses, harnessFailed)
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

// runUpdate wires production adapters and invokes Updater.Run.
func runUpdate(ctx context.Context, args UpdateArgs, stdout io.Writer) error {
	updater := &update.Updater{
		FS:  &osUpdateFS{},
		Cmd: &osCommander{},
		Env: &osUpdateEnv{},
	}

	report, runErr := updater.Run(ctx, update.Options{
		DryRun:       args.DryRun,
		WithGuidance: args.WithGuidance,
	})
	if runErr == nil {
		vaultPath := resolveVault("", report.Home, updater.Env.Getenv)
		report.VaultHasOldVocabFiles = oldVocabFilesPresent(vaultPath, updater.FS)
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
