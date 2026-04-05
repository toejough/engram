package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/toejough/targ"
)

// --- Targ args structs ---

// ChatCursorArgs holds parsed flags for the chat cursor subcommand.
type ChatCursorArgs struct {
	ChatFile string `targ:"flag,name=chat-file,desc=override chat file path (testing only)"`
}

// ChatPostArgs holds parsed flags for the chat post subcommand.
type ChatPostArgs struct {
	From     string `targ:"flag,name=from,desc=sender agent name"`
	To       string `targ:"flag,name=to,desc=comma-separated recipient names or all"`
	Thread   string `targ:"flag,name=thread,desc=conversation thread name"`
	MsgType  string `targ:"flag,name=type,desc=message type (intent|ack|wait|info|done|learned|ready|shutdown|escalate)"`
	Text     string `targ:"flag,name=text,desc=message content"`
	ChatFile string `targ:"flag,name=chat-file,desc=override chat file path (testing only)"`
}

// ChatWatchArgs holds parsed flags for the chat watch subcommand.
type ChatWatchArgs struct {
	Agent    string `targ:"flag,name=agent,desc=agent name to filter messages for"`
	Cursor   int    `targ:"flag,name=cursor,desc=line number to start watching from"`
	Types    string `targ:"flag,name=type,desc=comma-separated message types to filter (empty=all)"`
	Timeout  int    `targ:"flag,name=timeout,desc=seconds before giving up (0=block forever)"`
	ChatFile string `targ:"flag,name=chat-file,desc=override chat file path (testing only)"`
}

// RecallArgs holds parsed flags for the recall subcommand.
type RecallArgs struct {
	DataDir     string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	ProjectSlug string `targ:"flag,name=project-slug,desc=project directory slug"`
	Query       string `targ:"flag,name=query,desc=search query (omit for summary mode)"`
}

// ShowArgs holds parsed flags for the show subcommand.
type ShowArgs struct {
	Name    string `targ:"flag,name=name,desc=memory slug to display"`
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
}

// AddBoolFlag appends a flag if the bool is true.
func AddBoolFlag(flags []string, name string, value bool) []string {
	if value {
		flags = append(flags, name)
	}

	return flags
}

// BuildChatGroup builds the targ group for chat subcommands.
func BuildChatGroup(stdout, stderr io.Writer, stdin io.Reader) *targ.TargetGroup {
	return targ.Group("chat",
		targ.Targ(func(a ChatPostArgs) {
			args := append([]string{"engram", "chat", "post"}, ChatPostFlags(a)...)
			RunSafe(args, stdout, stderr, stdin)
		}).Name("post").Description("Post a message to the engram chat file"),
		targ.Targ(func(a ChatWatchArgs) {
			args := append([]string{"engram", "chat", "watch"}, ChatWatchFlags(a)...)
			RunSafe(args, stdout, stderr, stdin)
		}).Name("watch").Description("Block until a matching message arrives in the chat file"),
		targ.Targ(func(a ChatCursorArgs) {
			args := append([]string{"engram", "chat", "cursor"}, ChatCursorFlags(a)...)
			RunSafe(args, stdout, stderr, stdin)
		}).Name("cursor").Description("Output the current chat file line count (cursor position)"),
	)
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
	}
}

// ChatCursorFlags returns the CLI flag args for the chat cursor subcommand.
func ChatCursorFlags(a ChatCursorArgs) []string {
	return BuildFlags("--chat-file", a.ChatFile)
}

// ChatPostFlags returns the CLI flag args for the chat post subcommand.
func ChatPostFlags(a ChatPostArgs) []string {
	return BuildFlags(
		"--from", a.From,
		"--to", a.To,
		"--thread", a.Thread,
		"--type", a.MsgType,
		"--text", a.Text,
		"--chat-file", a.ChatFile,
	)
}

// ChatWatchFlags returns the CLI flag args for the chat watch subcommand.
func ChatWatchFlags(a ChatWatchArgs) []string {
	flags := BuildFlags(
		"--agent", a.Agent,
		"--type", a.Types,
		"--chat-file", a.ChatFile,
	)

	if a.Cursor != 0 {
		flags = append(flags, "--cursor", strconv.Itoa(a.Cursor))
	}

	if a.Timeout != 0 {
		flags = append(flags, "--timeout", strconv.Itoa(a.Timeout))
	}

	return flags
}

// DataDirFromHome returns the standard engram data directory for a given home path.
// It respects $XDG_DATA_HOME if set, otherwise defaults to $HOME/.local/share/engram.
func DataDirFromHome(home string) string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "engram")
	}

	return filepath.Join(home, ".local", "share", "engram")
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

// ShowFlags returns the CLI flag args for the show subcommand.
func ShowFlags(a ShowArgs) []string {
	return BuildFlags("--data-dir", a.DataDir, "--name", a.Name)
}

// Targets returns all targ targets for the engram CLI.
func Targets(stdout, stderr io.Writer, stdin io.Reader) []any {
	run := func(subcmd string, flags []string) {
		args := append([]string{"engram", subcmd}, flags...)
		RunSafe(args, stdout, stderr, stdin)
	}

	return append(BuildTargets(run), BuildChatGroup(stdout, stderr, stdin))
}
