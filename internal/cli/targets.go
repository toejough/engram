package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/toejough/targ"
)

// ApplyProposalArgs holds parsed flags for the apply-proposal subcommand.
type ApplyProposalArgs struct {
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	ID      string `targ:"flag,name=id,desc=proposal ID to apply"`
}

// --- Targ args structs ---

// CorrectArgs holds parsed flags for the correct subcommand.
type CorrectArgs struct {
	Message        string `targ:"flag,name=message,desc=user message text"`
	DataDir        string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	TranscriptPath string `targ:"flag,name=transcript-path,desc=path to session transcript"`
	ProjectSlug    string `targ:"flag,name=project-slug,desc=originating project slug"`
	APIToken       string `targ:"flag,name=api-token,env=ENGRAM_API_TOKEN,desc=Anthropic API token"`
}

// EvaluateArgs holds parsed flags for the evaluate subcommand.
type EvaluateArgs struct {
	TranscriptPath string `targ:"flag,name=transcript-path,desc=path to session transcript"`
	SessionID      string `targ:"flag,name=session-id,desc=session ID to evaluate"`
	DataDir        string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
}

// MaintainArgs holds parsed flags for the maintain subcommand.
type MaintainArgs struct {
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
}

// MigrateSBIAArgs holds parsed flags for the migrate-sbia subcommand.
type MigrateSBIAArgs struct {
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
}

// MigrateScoresArgs holds parsed flags for the migrate-scores subcommand.
type MigrateScoresArgs struct {
	DataDir  string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	Apply    bool   `targ:"flag,name=apply,desc=apply consolidations instead of dry-run"`
	APIToken string `targ:"flag,name=api-token,env=ENGRAM_API_TOKEN,desc=Anthropic API token"`
}

// MigrateSlugsArgs holds parsed flags for the migrate-slugs subcommand.
type MigrateSlugsArgs struct {
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	Slug    string `targ:"flag,name=slug,desc=project slug to backfill (defaults to PWD-derived slug)"`
	Apply   bool   `targ:"flag,name=apply,desc=apply changes instead of dry-run"`
}

// RecallArgs holds parsed flags for the recall subcommand.
type RecallArgs struct {
	DataDir     string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	ProjectSlug string `targ:"flag,name=project-slug,desc=project directory slug"`
	Query       string `targ:"flag,name=query,desc=search query (omit for summary mode)"`
}

// RefineArgs holds parsed flags for the refine subcommand.
type RefineArgs struct {
	DataDir  string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	APIToken string `targ:"flag,name=api-token,env=ENGRAM_API_TOKEN,desc=Anthropic API token"`
	DryRun   bool   `targ:"flag,name=dry-run,desc=show what would be refined without changing files"`
}

// RejectProposalArgs holds parsed flags for the reject-proposal subcommand.
type RejectProposalArgs struct {
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	ID      string `targ:"flag,name=id,desc=proposal ID to reject"`
}

// ShowArgs holds parsed flags for the show subcommand.
type ShowArgs struct {
	Name    string `targ:"flag,name=name,desc=memory slug to display"`
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
}

// SurfaceArgs holds parsed flags for the surface subcommand.
type SurfaceArgs struct {
	Mode           string `targ:"flag,name=mode,desc=surface mode: prompt or stop"`
	DataDir        string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	Message        string `targ:"flag,name=message,desc=user message (prompt mode)"`
	Format         string `targ:"flag,name=format,desc=output format: json"`
	TranscriptPath string `targ:"flag,name=transcript-path,desc=transcript JSONL path (stop mode)"`
	SessionID      string `targ:"flag,name=session-id,desc=session ID (stop mode)"`
}

// AddBoolFlag appends a flag if the bool is true.
func AddBoolFlag(flags []string, name string, value bool) []string {
	if value {
		flags = append(flags, name)
	}

	return flags
}

// --- Per-subcommand flag builders ---

// ApplyProposalFlags returns the CLI flag args for the apply-proposal subcommand.
func ApplyProposalFlags(a ApplyProposalArgs) []string {
	return BuildFlags("--data-dir", a.DataDir, "--id", a.ID)
}

// BuildFlags constructs a []string flag list from key-value pairs, skipping empty values.
func BuildFlags(pairs ...string) []string {
	flags := make([]string, 0, len(pairs))

	for i := 0; i+1 < len(pairs); i += 2 {
		if pairs[i+1] != "" {
			flags = append(flags, pairs[i], pairs[i+1])
		}
	}

	return flags
}

// BuildTargets constructs targ targets using the given run function.
func BuildTargets(run func(subcmd string, flags []string)) []any {
	return []any{
		targ.Targ(func(a CorrectArgs) { run("correct", CorrectFlags(a)) }).
			Name("correct").Description("Correct from user feedback"),
		targ.Targ(func(a MaintainArgs) { run("maintain", MaintainFlags(a)) }).
			Name("maintain").Description("Generate maintenance proposals"),
		targ.Targ(func(a SurfaceArgs) { run("surface", SurfaceFlags(a)) }).
			Name("surface").Description("Surface relevant memories"),
		targ.Targ(func(a EvaluateArgs) { run("evaluate", EvaluateFlags(a)) }).
			Name("evaluate").Description("Evaluate pending memory assessments via Haiku"),
		targ.Targ(func(a RefineArgs) { run("refine", RefineFlags(a)) }).
			Name("refine").Description("Re-extract SBIA fields from original transcripts"),
		targ.Targ(func(a ShowArgs) { run("show", ShowFlags(a)) }).
			Name("show").Description("Display full memory details"),
		targ.Targ(func(a ApplyProposalArgs) { run("apply-proposal", ApplyProposalFlags(a)) }).
			Name("apply-proposal").Description("Apply a maintenance proposal"),
		targ.Targ(func(a RejectProposalArgs) { run("reject-proposal", RejectProposalFlags(a)) }).
			Name("reject-proposal").Description("Reject a maintenance proposal"),
		targ.Targ(func(a RecallArgs) { run("recall", RecallFlags(a)) }).
			Name("recall").Description("Recall recent session context"),
		targ.Targ(func(a MigrateScoresArgs) { run("migrate-scores", MigrateScoresFlags(a)) }).
			Name("migrate-scores").Description("Score and consolidate existing memories"),
		targ.Targ(func(a MigrateSlugsArgs) { run("migrate-slugs", MigrateSlugsFlags(a)) }).
			Name("migrate-slugs").Description("Backfill project_slug on existing memories"),
		targ.Targ(func(a MigrateSBIAArgs) { run("migrate-sbia", MigrateSBIAFlags(a)) }).
			Name("migrate-sbia").Description("One-time migration to SBIA memory format"),
	}
}

// CorrectFlags returns the CLI flag args for the correct subcommand.
func CorrectFlags(a CorrectArgs) []string {
	return BuildFlags(
		"--message", a.Message,
		"--data-dir", a.DataDir,
		"--transcript-path", a.TranscriptPath,
		"--project-slug", a.ProjectSlug,
	)
}

// DataDirFromHome returns the standard engram data directory for a given home path.
func DataDirFromHome(home string) string {
	return filepath.Join(home, ".claude", "engram", "data")
}

// --- Flag construction helpers ---

// EvaluateFlags returns the CLI flag args for the evaluate subcommand.
func EvaluateFlags(a EvaluateArgs) []string {
	return BuildFlags(
		"--transcript-path", a.TranscriptPath,
		"--session-id", a.SessionID,
		"--data-dir", a.DataDir,
	)
}

// MaintainFlags returns the CLI flag args for the maintain subcommand.
func MaintainFlags(a MaintainArgs) []string {
	return BuildFlags("--data-dir", a.DataDir)
}

// MigrateSBIAFlags returns the CLI flag args for the migrate-sbia subcommand.
func MigrateSBIAFlags(a MigrateSBIAArgs) []string {
	return BuildFlags("--data-dir", a.DataDir)
}

// MigrateScoresFlags returns the CLI flag args for the migrate-scores subcommand.
func MigrateScoresFlags(a MigrateScoresArgs) []string {
	flags := BuildFlags("--data-dir", a.DataDir)
	flags = AddBoolFlag(flags, "--apply", a.Apply)

	return flags
}

// MigrateSlugsFlags returns the CLI flag args for the migrate-slugs subcommand.
func MigrateSlugsFlags(a MigrateSlugsArgs) []string {
	flags := BuildFlags("--data-dir", a.DataDir, "--slug", a.Slug)
	flags = AddBoolFlag(flags, "--apply", a.Apply)

	return flags
}

// ProjectSlugFromPath converts a filesystem path to a project slug by replacing
// path separators with dashes, matching the shell convention: echo "$PWD" | tr '/' '-'.
func ProjectSlugFromPath(path string) string {
	return strings.ReplaceAll(path, string(filepath.Separator), "-")
}

// RecallFlags returns the CLI flag args for the recall subcommand.
func RecallFlags(a RecallArgs) []string {
	return BuildFlags(
		"--data-dir", a.DataDir,
		"--project-slug", a.ProjectSlug,
		"--query", a.Query,
	)
}

// RefineFlags returns the CLI flag args for the refine subcommand.
func RefineFlags(a RefineArgs) []string {
	flags := BuildFlags("--data-dir", a.DataDir, "--api-token", a.APIToken)
	flags = AddBoolFlag(flags, "--dry-run", a.DryRun)

	return flags
}

// RejectProposalFlags returns the CLI flag args for the reject-proposal subcommand.
func RejectProposalFlags(a RejectProposalArgs) []string {
	return BuildFlags("--data-dir", a.DataDir, "--id", a.ID)
}

// RunSafe runs the CLI and prints errors to the given writer (ARCH-6: always exit 0).
func RunSafe(args []string, stdout, stderr io.Writer, stdin io.Reader) {
	err := Run(args, stdout, stderr, stdin)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
	}
}

// ShowFlags returns the CLI flag args for the show subcommand.
func ShowFlags(a ShowArgs) []string {
	return BuildFlags("--data-dir", a.DataDir, "--name", a.Name)
}

// SurfaceFlags returns the CLI flag args for the surface subcommand.
func SurfaceFlags(a SurfaceArgs) []string {
	return BuildFlags(
		"--mode", a.Mode,
		"--data-dir", a.DataDir,
		"--message", a.Message,
		"--format", a.Format,
		"--transcript-path", a.TranscriptPath,
		"--session-id", a.SessionID,
	)
}

// Targets returns all targ targets for the engram CLI.
func Targets(stdout, stderr io.Writer, stdin io.Reader) []any {
	return BuildTargets(func(subcmd string, flags []string) {
		args := append([]string{"engram", subcmd}, flags...)
		RunSafe(args, stdout, stderr, stdin)
	})
}
