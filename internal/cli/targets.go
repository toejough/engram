package cli

import (
	"fmt"
	"io"
	"strconv"

	"github.com/toejough/targ"
)

// --- Targ args structs ---

// AuditArgs holds parsed flags for the audit subcommand.
type AuditArgs struct {
	DataDir   string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	Timestamp string `targ:"flag,name=timestamp,desc=audit timestamp (ISO 8601)"`
	APIToken  string `targ:"flag,name=api-token,env=ENGRAM_API_TOKEN,desc=Anthropic API token"`
}

// AutomateArgs holds parsed flags for the automate subcommand.
type AutomateArgs struct {
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
}

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

// DemoteArgs holds parsed flags for the demote subcommand.
type DemoteArgs struct {
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	ToSkill bool   `targ:"flag,name=to-skill,desc=demote CLAUDE.md entry to skill"`
	Yes     bool   `targ:"flag,name=yes,desc=skip confirmation prompt"`
}

// EvaluateArgs holds parsed flags for the evaluate subcommand.
type EvaluateArgs struct {
	DataDir  string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	APIToken string `targ:"flag,name=api-token,env=ENGRAM_API_TOKEN,desc=Anthropic API token"`
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

// PromoteArgs holds parsed flags for the promote subcommand.
type PromoteArgs struct {
	DataDir    string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	ToSkill    bool   `targ:"flag,name=to-skill,desc=promote memory to skill"`
	ToClaudeMD bool   `targ:"flag,name=to-claude-md,desc=promote skill to CLAUDE.md"`
	Threshold  int    `targ:"flag,name=threshold,default=50,desc=minimum surfaced_count"`
	Yes        bool   `targ:"flag,name=yes,desc=skip confirmation prompt"`
}

// RegistryInitArgs holds parsed flags for the registry init subcommand.
type RegistryInitArgs struct {
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	DryRun  bool   `targ:"flag,name=dry-run,desc=print entries without writing"`
}

// RegistryMergeArgs holds parsed flags for the registry merge subcommand.
type RegistryMergeArgs struct {
	DataDir  string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	SourceID string `targ:"flag,name=source,desc=source instruction ID"`
	TargetID string `targ:"flag,name=target,desc=target instruction ID"`
}

// RegistryRegisterSourceArgs holds parsed flags for the registry register-source subcommand.
type RegistryRegisterSourceArgs struct {
	DataDir    string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	SourceType string `targ:"flag,name=type,desc=source type (claude-md or memory-md or rule or skill)"`
	Path       string `targ:"flag,name=path,desc=path to source file or name"`
}

// RemindArgs holds parsed flags for the remind subcommand.
type RemindArgs struct {
	DataDir  string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	FilePath string `targ:"flag,name=file-path,desc=file path from tool call"`
}

// ReviewArgs holds parsed flags for the review subcommand.
type ReviewArgs struct {
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	Format  string `targ:"flag,name=format,default=table,desc=output format: json or table"`
}

// SurfaceArgs holds parsed flags for the surface subcommand.
type SurfaceArgs struct {
	Mode      string `targ:"flag,name=mode,desc=surface mode: session-start or prompt or tool"`
	DataDir   string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	Message   string `targ:"flag,name=message,desc=user message (prompt mode)"`
	ToolName  string `targ:"flag,name=tool-name,desc=tool name (tool mode)"`
	ToolInput string `targ:"flag,name=tool-input,desc=tool input JSON (tool mode)"`
	Format    string `targ:"flag,name=format,desc=output format: json"`
}

// AddBoolFlag appends a flag if the bool is true.
func AddBoolFlag(flags []string, name string, value bool) []string {
	if value {
		flags = append(flags, name)
	}

	return flags
}

// --- Per-subcommand flag builders ---

// AuditFlags returns the CLI flag args for the audit subcommand.
func AuditFlags(a AuditArgs) []string {
	return BuildFlags("--data-dir", a.DataDir, "--timestamp", a.Timestamp)
}

// AutomateFlags returns the CLI flag args for the automate subcommand.
func AutomateFlags(a AutomateArgs) []string {
	return BuildFlags("--data-dir", a.DataDir)
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

// DemoteFlags returns the CLI flag args for the demote subcommand.
func DemoteFlags(a DemoteArgs) []string {
	flags := BuildFlags("--data-dir", a.DataDir)
	flags = AddBoolFlag(flags, "--to-skill", a.ToSkill)
	flags = AddBoolFlag(flags, "--yes", a.Yes)

	return flags
}

// EvaluateFlags returns the CLI flag args for the evaluate subcommand.
func EvaluateFlags(a EvaluateArgs) []string {
	return BuildFlags("--data-dir", a.DataDir)
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

// PromoteFlags returns the CLI flag args for the promote subcommand.
func PromoteFlags(a PromoteArgs) []string {
	flags := BuildFlags("--data-dir", a.DataDir)
	flags = AddBoolFlag(flags, "--to-skill", a.ToSkill)
	flags = AddBoolFlag(flags, "--to-claude-md", a.ToClaudeMD)
	flags = AddBoolFlag(flags, "--yes", a.Yes)

	if a.Threshold > 0 {
		flags = append(flags, "--threshold", strconv.Itoa(a.Threshold))
	}

	return flags
}

// RegistryInitFlags returns the CLI flag args for the registry init subcommand.
func RegistryInitFlags(a RegistryInitArgs) []string {
	flags := BuildFlags("--data-dir", a.DataDir)
	flags = AddBoolFlag(flags, "--dry-run", a.DryRun)

	return flags
}

// RegistryMergeFlags returns the CLI flag args for the registry merge subcommand.
func RegistryMergeFlags(a RegistryMergeArgs) []string {
	return BuildFlags("--data-dir", a.DataDir, "--source", a.SourceID, "--target", a.TargetID)
}

// RegistryRegisterSourceFlags returns the CLI flag args for the registry register-source subcommand.
func RegistryRegisterSourceFlags(a RegistryRegisterSourceArgs) []string {
	return BuildFlags("--data-dir", a.DataDir, "--type", a.SourceType, "--path", a.Path)
}

// RemindFlags returns the CLI flag args for the remind subcommand.
func RemindFlags(a RemindArgs) []string {
	return BuildFlags("--data-dir", a.DataDir, "--file-path", a.FilePath)
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

// SurfaceFlags returns the CLI flag args for the surface subcommand.
func SurfaceFlags(a SurfaceArgs) []string {
	return BuildFlags(
		"--mode", a.Mode,
		"--data-dir", a.DataDir,
		"--message", a.Message,
		"--tool-name", a.ToolName,
		"--tool-input", a.ToolInput,
		"--format", a.Format,
	)
}

// Targets returns all targ targets for the engram CLI.
//
func Targets(stdout io.Writer, stderr io.Writer, stdin io.Reader) []any {
	run := func(subcmd string, flags []string) {
		args := append([]string{"engram", subcmd}, flags...)
		RunSafe(args, stdout, stderr, stdin)
	}

	return []any{
		targ.Targ(func(a AuditArgs) { run("audit", AuditFlags(a)) }).
			Name("audit").Description("Run compliance audit"),
		targ.Targ(func(a AutomateArgs) { run("automate", AutomateFlags(a)) }).
			Name("automate").Description("Generate automation proposals"),
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
		targ.Targ(func(a RemindArgs) { run("remind", RemindFlags(a)) }).
			Name("remind").Description("Proactive reminders for tool calls"),
		targ.Targ(func(a InstructArgs) { run("instruct", InstructFlags(a)) }).
			Name("instruct").Description("Audit instruction quality"),
		targ.Targ(func(a ContextUpdateArgs) { run("context-update", ContextUpdateFlags(a)) }).
			Name("context-update").Description("Update session context"),
		targ.Targ(func(a PromoteArgs) { run("promote", PromoteFlags(a)) }).
			Name("promote").Description("Promote memories to skills or CLAUDE.md"),
		targ.Targ(func(a DemoteArgs) { run("demote", DemoteFlags(a)) }).
			Name("demote").Description("Demote CLAUDE.md entries to skills"),
		targ.Group("registry",
			targ.Targ(func(a RegistryInitArgs) {
				run("registry", append([]string{"init"}, RegistryInitFlags(a)...))
			}).Name("init").Description("Backfill registry from memory files"),
			targ.Targ(func(a RegistryRegisterSourceArgs) {
				run("registry", append([]string{"register-source"}, RegistryRegisterSourceFlags(a)...))
			}).Name("register-source").Description("Register a single source"),
			targ.Targ(func(a RegistryMergeArgs) {
				run("registry", append([]string{"merge"}, RegistryMergeFlags(a)...))
			}).Name("merge").Description("Merge two registry entries"),
		),
	}
}
