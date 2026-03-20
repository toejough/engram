package cli

import (
	"fmt"
	"io"
	"strconv"

	"github.com/toejough/targ"
)

// ApplyProposalArgs holds parsed flags for the apply-proposal subcommand.
type ApplyProposalArgs struct {
	DataDir  string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	Action   string `targ:"flag,name=action,desc=action: remove or rewrite or broaden_keywords or escalate"`
	Memory   string `targ:"flag,name=memory,desc=path to memory file"`
	Fields   string `targ:"flag,name=fields,desc=JSON object of fields to update"`
	Keywords string `targ:"flag,name=keywords,desc=comma-separated keywords to add"`
	Level    int    `targ:"flag,name=level,desc=escalation level"`
}

// --- Targ args structs ---

// ContextUpdateArgs holds parsed flags for the context-update subcommand.
type ContextUpdateArgs struct {
	TranscriptPath string `targ:"flag,name=transcript-path,desc=path to session transcript"`
	SessionID      string `targ:"flag,name=session-id,desc=session identifier"`
	DataDir        string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	ContextPath    string `targ:"flag,name=context-path,desc=path to session-context.md"`
	APIToken       string `targ:"flag,name=api-token,env=ENGRAM_API_TOKEN,desc=Anthropic API token"`
}

// CorrectArgs holds parsed flags for the correct subcommand.
type CorrectArgs struct {
	Message        string `targ:"flag,name=message,desc=user message text"`
	DataDir        string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	TranscriptPath string `targ:"flag,name=transcript-path,desc=path to session transcript"`
	APIToken       string `targ:"flag,name=api-token,env=ENGRAM_API_TOKEN,desc=Anthropic API token"`
}

// EvaluateArgs holds parsed flags for the evaluate subcommand.
type EvaluateArgs struct {
	DataDir  string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	APIToken string `targ:"flag,name=api-token,env=ENGRAM_API_TOKEN,desc=Anthropic API token"`
}

// FeedbackArgs holds parsed flags for the feedback subcommand.
type FeedbackArgs struct {
	DataDir    string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	Relevant   bool   `targ:"flag,name=relevant,desc=memory was relevant to current task"`
	Irrelevant bool   `targ:"flag,name=irrelevant,desc=memory was not relevant"`
	Used       bool   `targ:"flag,name=used,desc=memory advice was followed"`
	Notused    bool   `targ:"flag,name=notused,desc=memory advice was not followed"`
}

// FlushArgs holds parsed flags for the flush subcommand.
type FlushArgs struct {
	DataDir        string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	TranscriptPath string `targ:"flag,name=transcript-path,desc=path to session transcript"`
	SessionID      string `targ:"flag,name=session-id,desc=session identifier"`
	ContextPath    string `targ:"flag,name=context-path,desc=path to session-context.md"`
}

// InstructArgs holds parsed flags for the instruct subcommand.
type InstructArgs struct {
	DataDir    string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	ProjectDir string `targ:"flag,name=project-dir,desc=path to project directory"`
	APIToken   string `targ:"flag,name=api-token,env=ENGRAM_API_TOKEN,desc=Anthropic API token"`
}

// LearnArgs holds parsed flags for the learn subcommand.
type LearnArgs struct {
	DataDir        string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	TranscriptPath string `targ:"flag,name=transcript-path,desc=path to session transcript"`
	SessionID      string `targ:"flag,name=session-id,desc=session identifier"`
	APIToken       string `targ:"flag,name=api-token,env=ENGRAM_API_TOKEN,desc=Anthropic API token"`
}

// MaintainArgs holds parsed flags for the maintain subcommand.
type MaintainArgs struct {
	DataDir   string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	Apply     bool   `targ:"flag,name=apply,desc=apply proposals instead of generating"`
	Proposals string `targ:"flag,name=proposals,desc=path to proposals JSON file"`
	Yes       bool   `targ:"flag,name=yes,desc=auto-approve all proposals"`
	APIToken  string `targ:"flag,name=api-token,env=ENGRAM_API_TOKEN,desc=Anthropic API token"`
}

// RegistryMergeArgs holds parsed flags for the registry merge subcommand.
type RegistryMergeArgs struct {
	DataDir  string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	SourceID string `targ:"flag,name=source,desc=source instruction ID"`
	TargetID string `targ:"flag,name=target,desc=target instruction ID"`
}

// ReviewArgs holds parsed flags for the review subcommand.
type ReviewArgs struct {
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	Format  string `targ:"flag,name=format,default=table,desc=output format: json or table"`
}

// ShowArgs holds parsed flags for the show subcommand.
type ShowArgs struct {
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
}

// SurfaceArgs holds parsed flags for the surface subcommand.
type SurfaceArgs struct {
	Mode        string `targ:"flag,name=mode,desc=surface mode: session-start or prompt or tool"`
	DataDir     string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	Message     string `targ:"flag,name=message,desc=user message (prompt mode)"`
	ToolName    string `targ:"flag,name=tool-name,desc=tool name (tool mode)"`
	ToolInput   string `targ:"flag,name=tool-input,desc=tool input JSON (tool mode)"`
	ToolOutput  string `targ:"flag,name=tool-output,desc=tool output or error text (tool mode)"`
	ToolErrored bool   `targ:"flag,name=tool-errored,desc=true if tool call failed (tool mode)"`
	Format      string `targ:"flag,name=format,desc=output format: json"`
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
	flags := BuildFlags(
		"--data-dir", a.DataDir,
		"--action", a.Action,
		"--memory", a.Memory,
		"--fields", a.Fields,
		"--keywords", a.Keywords,
	)

	if a.Level > 0 {
		flags = append(flags, "--level", strconv.Itoa(a.Level))
	}

	return flags
}

// --- Flag construction helpers ---

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
		targ.Targ(func(a EvaluateArgs) { run("evaluate", EvaluateFlags(a)) }).
			Name("evaluate").Description("Evaluate memory effectiveness"),
		targ.Targ(func(a ReviewArgs) { run("review", ReviewFlags(a)) }).
			Name("review").Description("Review instruction registry"),
		targ.Targ(func(a MaintainArgs) { run("maintain", MaintainFlags(a)) }).
			Name("maintain").Description("Generate or apply maintenance proposals"),
		targ.Targ(func(a SurfaceArgs) { run("surface", SurfaceFlags(a)) }).
			Name("surface").Description("Surface relevant memories"),
		targ.Targ(func(a LearnArgs) { run("learn", LearnFlags(a)) }).
			Name("learn").Description("Extract learnings from session"),
		targ.Targ(func(a InstructArgs) { run("instruct", InstructFlags(a)) }).
			Name("instruct").Description("Audit instruction quality"),
		targ.Targ(func(a ContextUpdateArgs) { run("context-update", ContextUpdateFlags(a)) }).
			Name("context-update").Description("Update session context"),
		targ.Targ(func(a FeedbackArgs) { run("feedback", FeedbackFlags(a)) }).
			Name("feedback").Description("Record memory relevance feedback"),
		targ.Targ(func(a FlushArgs) { run("flush", FlushFlags(a)) }).
			Name("flush").Description("Run end-of-turn flush pipeline"),
		targ.Targ(func(a ShowArgs) { run("show", ShowFlags(a)) }).
			Name("show").Description("Display full memory details"),
		targ.Targ(func(a ApplyProposalArgs) { run("apply-proposal", ApplyProposalFlags(a)) }).
			Name("apply-proposal").Description("Apply a maintenance proposal"),
		targ.Group("registry",
			targ.Targ(func(a RegistryMergeArgs) {
				run("registry", append([]string{"merge"}, RegistryMergeFlags(a)...))
			}).Name("merge").Description("Merge two registry entries"),
		),
	}
}

// ContextUpdateFlags returns the CLI flag args for the context-update subcommand.
func ContextUpdateFlags(a ContextUpdateArgs) []string {
	return BuildFlags(
		"--transcript-path", a.TranscriptPath,
		"--session-id", a.SessionID,
		"--data-dir", a.DataDir,
		"--context-path", a.ContextPath,
	)
}

// CorrectFlags returns the CLI flag args for the correct subcommand.
func CorrectFlags(a CorrectArgs) []string {
	return BuildFlags(
		"--message", a.Message,
		"--data-dir", a.DataDir,
		"--transcript-path", a.TranscriptPath,
	)
}

// EvaluateFlags returns the CLI flag args for the evaluate subcommand.
func EvaluateFlags(a EvaluateArgs) []string {
	return BuildFlags("--data-dir", a.DataDir)
}

// FeedbackFlags returns the CLI flag args for the feedback subcommand.
func FeedbackFlags(a FeedbackArgs) []string {
	flags := BuildFlags("--data-dir", a.DataDir)
	flags = AddBoolFlag(flags, "--relevant", a.Relevant)
	flags = AddBoolFlag(flags, "--irrelevant", a.Irrelevant)
	flags = AddBoolFlag(flags, "--used", a.Used)
	flags = AddBoolFlag(flags, "--notused", a.Notused)

	return flags
}

// FlushFlags returns the CLI flag args for the flush subcommand.
func FlushFlags(a FlushArgs) []string {
	return BuildFlags(
		"--data-dir", a.DataDir,
		"--transcript-path", a.TranscriptPath,
		"--session-id", a.SessionID,
		"--context-path", a.ContextPath,
	)
}

// InstructFlags returns the CLI flag args for the instruct subcommand.
func InstructFlags(a InstructArgs) []string {
	return BuildFlags("--data-dir", a.DataDir, "--project-dir", a.ProjectDir)
}

// LearnFlags returns the CLI flag args for the learn subcommand.
func LearnFlags(a LearnArgs) []string {
	return BuildFlags(
		"--data-dir", a.DataDir,
		"--transcript-path", a.TranscriptPath,
		"--session-id", a.SessionID,
	)
}

// MaintainFlags returns the CLI flag args for the maintain subcommand.
func MaintainFlags(a MaintainArgs) []string {
	flags := BuildFlags("--data-dir", a.DataDir, "--proposals", a.Proposals)
	flags = AddBoolFlag(flags, "--apply", a.Apply)
	flags = AddBoolFlag(flags, "--yes", a.Yes)

	return flags
}

// RegistryMergeFlags returns the CLI flag args for the registry merge subcommand.
func RegistryMergeFlags(a RegistryMergeArgs) []string {
	return BuildFlags("--data-dir", a.DataDir, "--source", a.SourceID, "--target", a.TargetID)
}

// ReviewFlags returns the CLI flag args for the review subcommand.
func ReviewFlags(a ReviewArgs) []string {
	return BuildFlags("--data-dir", a.DataDir, "--format", a.Format)
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
	return BuildFlags("--data-dir", a.DataDir)
}

// SurfaceFlags returns the CLI flag args for the surface subcommand.
func SurfaceFlags(a SurfaceArgs) []string {
	flags := BuildFlags(
		"--mode", a.Mode,
		"--data-dir", a.DataDir,
		"--message", a.Message,
		"--tool-name", a.ToolName,
		"--tool-input", a.ToolInput,
		"--tool-output", a.ToolOutput,
		"--format", a.Format,
	)

	return AddBoolFlag(flags, "--tool-errored", a.ToolErrored)
}

// Targets returns all targ targets for the engram CLI.
func Targets(stdout, stderr io.Writer, stdin io.Reader) []any {
	return BuildTargets(func(subcmd string, flags []string) {
		args := append([]string{"engram", subcmd}, flags...)
		RunSafe(args, stdout, stderr, stdin)
	})
}
