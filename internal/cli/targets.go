package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/toejough/targ"
)

// CommonLearnArgs holds shared flags for learn subcommands.
type CommonLearnArgs struct {
	Situation  string `targ:"flag,name=situation,desc=context when this applies"`
	Source     string `targ:"flag,name=source,desc=human or agent"`
	DataDir    string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	NoDupCheck bool   `targ:"flag,name=no-dup-check,desc=skip duplicate/contradiction detection"`
}

// LearnFactArgs holds parsed flags for the learn fact subcommand.
type LearnFactArgs struct {
	CommonLearnArgs

	Subject   string `targ:"flag,name=subject,desc=subject of the fact"`
	Predicate string `targ:"flag,name=predicate,desc=relationship or verb"`
	Object    string `targ:"flag,name=object,desc=object of the fact"`
}

// LearnFeedbackArgs holds parsed flags for the learn feedback subcommand.
type LearnFeedbackArgs struct {
	CommonLearnArgs

	Behavior string `targ:"flag,name=behavior,desc=observed behavior"`
	Impact   string `targ:"flag,name=impact,desc=impact of the behavior"`
	Action   string `targ:"flag,name=action,desc=recommended action"`
}

// ListArgs holds parsed flags for the list subcommand.
type ListArgs struct {
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
}

// RecallArgs holds parsed flags for the recall subcommand.
type RecallArgs struct {
	DataDir      string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	ProjectSlug  string `targ:"flag,name=project-slug,desc=project directory slug"`
	Query        string `targ:"flag,name=query,desc=search query (omit for summary mode)"`
	MemoriesOnly bool   `targ:"flag,name=memories-only,desc=search only memory files"`
	Limit        int    `targ:"flag,name=limit,desc=max memories to return (default 10)"`
}

// ShowArgs holds parsed flags for the show subcommand.
type ShowArgs struct {
	Name    string `targ:"flag,name=name,desc=memory slug to display"`
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
}

// UpdateArgs holds parsed flags for the update subcommand.
type UpdateArgs struct {
	Name      string `targ:"flag,name=name,required,desc=memory slug (required)"`
	DataDir   string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	Situation string `targ:"flag,name=situation,desc=context when this applies"`
	Behavior  string `targ:"flag,name=behavior,desc=observed behavior"`
	Impact    string `targ:"flag,name=impact,desc=impact of the behavior"`
	Action    string `targ:"flag,name=action,desc=recommended action"`
	Subject   string `targ:"flag,name=subject,desc=subject of the fact"`
	Predicate string `targ:"flag,name=predicate,desc=relationship or verb"`
	Object    string `targ:"flag,name=object,desc=object of the fact"`
	Source    string `targ:"flag,name=source,desc=human or agent"`
}

// AddBoolFlag appends a flag if the bool is true.
func AddBoolFlag(flags []string, name string, value bool) []string {
	if value {
		flags = append(flags, name)
	}

	return flags
}

// AddIntFlag appends a flag with the int value converted to string, if non-zero.
func AddIntFlag(flags []string, name string, value int) []string {
	if value != 0 {
		flags = append(flags, name, strconv.Itoa(value))
	}

	return flags
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
		targ.Targ(func(a RecallArgs) { run("recall", RecallFlags(a)) }).
			Name("recall").Description("Recall recent session context"),
		targ.Targ(func(a ShowArgs) { run("show", ShowFlags(a)) }).
			Name("show").Description("Display full memory details"),
		targ.Targ(func(a ListArgs) { run("list", ListFlags(a)) }).
			Name("list").Description("List all memories with type, name, and situation"),
		targ.Group("learn",
			targ.Targ(func(a LearnFeedbackArgs) { run("learn feedback", LearnFeedbackFlags(a)) }).
				Name("feedback").Description("Learn from behavioral feedback"),
			targ.Targ(func(a LearnFactArgs) { run("learn fact", LearnFactFlags(a)) }).
				Name("fact").Description("Learn a factual statement"),
		),
		targ.Targ(func(a UpdateArgs) { run("update", UpdateFlags(a)) }).
			Name("update").Description("Update an existing memory"),
	}
}

// DataDirFromHome returns the standard engram data directory for a given home path.
// It respects $XDG_DATA_HOME if set, otherwise defaults to $HOME/.local/share/engram.
// getenv is injected so callers control environment access (pass os.Getenv in production).
func DataDirFromHome(home string, getenv func(string) string) string {
	if xdg := getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "engram")
	}

	return filepath.Join(home, ".local", "share", "engram")
}

// LearnFactFlags returns the CLI flag args for the learn fact subcommand.
func LearnFactFlags(a LearnFactArgs) []string {
	flags := BuildFlags(
		"--situation", a.Situation,
		"--source", a.Source,
		"--data-dir", a.DataDir,
		"--subject", a.Subject,
		"--predicate", a.Predicate,
		"--object", a.Object,
	)
	flags = AddBoolFlag(flags, "--no-dup-check", a.NoDupCheck)

	return flags
}

// LearnFeedbackFlags returns the CLI flag args for the learn feedback subcommand.
func LearnFeedbackFlags(a LearnFeedbackArgs) []string {
	flags := BuildFlags(
		"--situation", a.Situation,
		"--source", a.Source,
		"--data-dir", a.DataDir,
		"--behavior", a.Behavior,
		"--impact", a.Impact,
		"--action", a.Action,
	)
	flags = AddBoolFlag(flags, "--no-dup-check", a.NoDupCheck)

	return flags
}

// ListFlags returns the CLI flag args for the list subcommand.
func ListFlags(a ListArgs) []string {
	return BuildFlags("--data-dir", a.DataDir)
}

// ProjectSlugFromPath converts a filesystem path to a project slug by replacing
// path separators with dashes, matching the shell convention: echo "$PWD" | tr '/' '-'.
func ProjectSlugFromPath(path string) string {
	return strings.ReplaceAll(path, string(filepath.Separator), "-")
}

// RecallFlags returns the CLI flag args for the recall subcommand.
func RecallFlags(a RecallArgs) []string {
	flags := BuildFlags(
		"--data-dir", a.DataDir,
		"--project-slug", a.ProjectSlug,
		"--query", a.Query,
	)
	flags = AddBoolFlag(flags, "--memories-only", a.MemoriesOnly)
	flags = AddIntFlag(flags, "--limit", a.Limit)

	return flags
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

// Targets returns all targ targets for the engram CLI.
func Targets(stdout, stderr io.Writer, stdin io.Reader) []any {
	run := func(subcmd string, flags []string) {
		parts := strings.Fields(subcmd)
		args := append([]string{"engram"}, parts...)
		args = append(args, flags...)
		RunSafe(args, stdout, stderr, stdin)
	}

	return BuildTargets(run)
}

// UpdateFlags returns the CLI flag args for the update subcommand.
func UpdateFlags(a UpdateArgs) []string {
	return BuildFlags(
		"--name", a.Name,
		"--data-dir", a.DataDir,
		"--situation", a.Situation,
		"--behavior", a.Behavior,
		"--impact", a.Impact,
		"--action", a.Action,
		"--subject", a.Subject,
		"--predicate", a.Predicate,
		"--object", a.Object,
		"--source", a.Source,
	)
}
