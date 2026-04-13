package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/toejough/targ"
)

// IntentArgs holds flags for `engram intent`.
type IntentArgs struct {
	From          string `targ:"flag,name=from,desc=sender agent name"`
	To            string `targ:"flag,name=to,desc=recipient agent name"`
	Situation     string `targ:"flag,name=situation,desc=current situation description"`
	PlannedAction string `targ:"flag,name=planned-action,desc=planned action description"`
	Addr          string `targ:"flag,name=addr,desc=API server address"`
}

// LearnArgs holds flags for `engram learn`.
type LearnArgs struct {
	From      string `targ:"flag,name=from,desc=sender agent name"`
	Type      string `targ:"flag,name=type,desc=learn type: feedback or fact"`
	Situation string `targ:"flag,name=situation,desc=situation description"`
	Behavior  string `targ:"flag,name=behavior,desc=observed behavior (feedback)"`
	Impact    string `targ:"flag,name=impact,desc=impact description (feedback)"`
	Action    string `targ:"flag,name=action,desc=corrective action (feedback)"`
	Subject   string `targ:"flag,name=subject,desc=fact subject"`
	Predicate string `targ:"flag,name=predicate,desc=fact predicate"`
	Object    string `targ:"flag,name=object,desc=fact object"`
	Addr      string `targ:"flag,name=addr,desc=API server address"`
}

// PostArgs holds flags for `engram post`.
type PostArgs struct {
	From string `targ:"flag,name=from,desc=sender agent name"`
	To   string `targ:"flag,name=to,desc=recipient agent name"`
	Text string `targ:"flag,name=text,desc=message content"`
	Addr string `targ:"flag,name=addr,desc=API server address"`
}

// RecallArgs holds parsed flags for the recall subcommand.
type RecallArgs struct {
	DataDir     string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	ProjectSlug string `targ:"flag,name=project-slug,desc=project directory slug"`
	Query       string `targ:"flag,name=query,desc=search query (omit for summary mode)"`
}

// ServerUpArgs holds flags for `engram server up`.
type ServerUpArgs struct {
	ChatFile string `targ:"flag,name=chat-file,desc=chat file path"`
	LogFile  string `targ:"flag,name=log-file,desc=log file path (optional)"`
	Addr     string `targ:"flag,name=addr,desc=listen address"`
}

// ShowArgs holds parsed flags for the show subcommand.
type ShowArgs struct {
	Name    string `targ:"flag,name=name,desc=memory slug to display"`
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
}

// StatusArgs holds flags for `engram status`.
type StatusArgs struct {
	Addr string `targ:"flag,name=addr,desc=API server address"`
}

// SubscribeArgs holds flags for `engram subscribe`.
type SubscribeArgs struct {
	Agent       string `targ:"flag,name=agent,desc=agent name to subscribe as"`
	AfterCursor int    `targ:"flag,name=after-cursor,desc=cursor position to start from"`
	Addr        string `targ:"flag,name=addr,desc=API server address"`
}

// AddBoolFlag appends a flag if the bool is true.
func AddBoolFlag(flags []string, name string, value bool) []string {
	if value {
		flags = append(flags, name)
	}

	return flags
}

// AddIntFlag appends a flag and its string value if value is non-zero.
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

// BuildServerGroup builds the targ group for server subcommands.
func BuildServerGroup(stdout, stderr io.Writer, stdin io.Reader) *targ.TargetGroup {
	return targ.Group("server",
		targ.Targ(func(a ServerUpArgs) {
			args := append([]string{"engram", "server", "up"}, ServerUpFlags(a)...)
			RunSafe(args, stdout, stderr, stdin)
		}).Name("up").Description("Start the engram API server"),
	)
}

// BuildTargets constructs targ targets using the given run function.
func BuildTargets(run func(subcmd string, flags []string)) []any {
	return []any{
		targ.Targ(func(a RecallArgs) { run("recall", RecallFlags(a)) }).
			Name("recall").Description("Recall recent session context"),
		targ.Targ(func(a ShowArgs) { run("show", ShowFlags(a)) }).
			Name("show").Description("Display full memory details"),
		targ.Targ(func(a PostArgs) { run(postCmd, PostFlags(a)) }).
			Name(postCmd).Description("Post a message to the engram chat"),
		targ.Targ(func(a IntentArgs) { run(intentCmd, IntentFlags(a)) }).
			Name(intentCmd).Description("Post an intent and wait for engram response"),
		targ.Targ(func(a LearnArgs) { run(learnCmd, LearnFlags(a)) }).
			Name(learnCmd).Description("Submit feedback or fact to the engram agent"),
		targ.Targ(func(a StatusArgs) { run(statusCmd, StatusFlags(a)) }).
			Name(statusCmd).Description("Check API server status and connected agents"),
		targ.Targ(func(a SubscribeArgs) {
			run(subscribeCmd, SubscribeFlags(a))
		}).
			Name(subscribeCmd).Description("Subscribe to messages for an agent"),
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

// IntentFlags returns the CLI flag args for the intent subcommand.
func IntentFlags(a IntentArgs) []string {
	return BuildFlags(
		"--from", a.From,
		"--to", a.To,
		"--situation", a.Situation,
		"--planned-action", a.PlannedAction,
		"--addr", a.Addr,
	)
}

// LearnFlags returns the CLI flag args for the learn subcommand.
func LearnFlags(a LearnArgs) []string {
	return BuildFlags(
		"--from", a.From,
		"--type", a.Type,
		"--situation", a.Situation,
		"--behavior", a.Behavior,
		"--impact", a.Impact,
		"--action", a.Action,
		"--subject", a.Subject,
		"--predicate", a.Predicate,
		"--object", a.Object,
		"--addr", a.Addr,
	)
}

// PostFlags returns the CLI flag args for the post subcommand.
func PostFlags(a PostArgs) []string {
	return BuildFlags("--from", a.From, "--to", a.To, "--text", a.Text, "--addr", a.Addr)
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

// RunSafe runs the CLI and prints errors to the given writer (ARCH-6: always exit 0).
func RunSafe(args []string, stdout, stderr io.Writer, stdin io.Reader) {
	err := Run(args, stdout, stderr, stdin)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
	}
}

// ServerUpFlags returns the CLI flag args for the server up subcommand.
func ServerUpFlags(a ServerUpArgs) []string {
	return BuildFlags("--chat-file", a.ChatFile, "--log-file", a.LogFile, "--addr", a.Addr)
}

// ShowFlags returns the CLI flag args for the show subcommand.
func ShowFlags(a ShowArgs) []string {
	return BuildFlags("--data-dir", a.DataDir, "--name", a.Name)
}

// StatusFlags returns the CLI flag args for the status subcommand.
func StatusFlags(a StatusArgs) []string {
	return BuildFlags("--addr", a.Addr)
}

// SubscribeFlags returns the CLI flag args for the subscribe subcommand.
func SubscribeFlags(a SubscribeArgs) []string {
	flags := BuildFlags("--agent", a.Agent, "--addr", a.Addr)
	flags = AddIntFlag(flags, "--after-cursor", a.AfterCursor)

	return flags
}

// Targets returns all targ targets for the engram CLI.
func Targets(stdout, stderr io.Writer, stdin io.Reader) []any {
	run := func(subcmd string, flags []string) {
		args := append([]string{"engram", subcmd}, flags...)
		RunSafe(args, stdout, stderr, stdin)
	}

	return append(
		BuildTargets(run),
		BuildServerGroup(stdout, stderr, stdin),
	)
}
