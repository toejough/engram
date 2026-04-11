package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/toejough/targ"
)

// AgentKillArgs holds flags for `engram agent kill`.
type AgentKillArgs struct {
	Name      string
	ChatFile  string
	StateFile string
}

// AgentListArgs holds flags for `engram agent list`.
type AgentListArgs struct {
	StateFile string
	ChatFile  string // used for reconstruction fallback when state file is missing
}

// AgentRunArgs holds flags for `engram agent run`.
type AgentRunArgs struct {
	Name      string
	Prompt    string
	ChatFile  string
	StateFile string
}

// AgentSpawnArgs holds flags for `engram agent spawn`.
type AgentSpawnArgs struct {
	Name      string
	Prompt    string
	ChatFile  string
	StateFile string
}

// AgentWaitReadyArgs holds flags for `engram agent wait-ready`.
type AgentWaitReadyArgs struct {
	Name     string
	Cursor   int
	MaxWait  int // seconds
	ChatFile string
}

// ChatAckWaitArgs holds parsed flags for the chat ack-wait subcommand.
type ChatAckWaitArgs struct {
	Agent      string `targ:"flag,name=agent,desc=calling agent name"`
	Cursor     int    `targ:"flag,name=cursor,desc=line position to start watching from"`
	Recipients string `targ:"flag,name=recipients,desc=comma-separated recipient names"`
	MaxWait    int    `targ:"flag,name=max-wait,desc=seconds to wait for online-silent recipients (default 30)"`
	ChatFile   string `targ:"flag,name=chat-file,desc=override chat file path (testing only)"`
}

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
	MaxWait  int    `targ:"flag,name=max-wait,desc=seconds before giving up (0=block forever)"`
	ChatFile string `targ:"flag,name=chat-file,desc=override chat file path (testing only)"`
}

// DispatchAssignArgs holds parsed flags for the dispatch assign subcommand.
type DispatchAssignArgs struct {
	Agent     string `targ:"flag,name=agent,desc=agent name to assign task to"`
	Task      string `targ:"flag,name=task,desc=task description to assign"`
	ChatFile  string `targ:"flag,name=chat-file,desc=override chat file path (testing only)"`
	StateFile string `targ:"flag,name=state-file,desc=override state file path (testing only)"`
}

// DispatchDrainArgs holds parsed flags for the dispatch drain subcommand.
type DispatchDrainArgs struct {
	Timeout   int    `targ:"flag,name=timeout,desc=drain timeout in seconds (default 60)"`
	StateFile string `targ:"flag,name=state-file,desc=override state file path (testing only)"`
}

// DispatchStartArgs holds parsed flags for the dispatch start subcommand.
type DispatchStartArgs struct {
	Agent         []string `targ:"flag,name=agent,desc=agent name (use multiple times for multiple agents)"`
	MaxConcurrent int      `targ:"flag,name=max-concurrent,desc=max concurrent worker sessions (default 4)"`
	ChatFile      string   `targ:"flag,name=chat-file,desc=override chat file path (testing only)"`
	StateFile     string   `targ:"flag,name=state-file,desc=override state file path (testing only)"`
	ClaudeBinary  string   `targ:"flag,name=claude-binary,desc=override claude binary path (testing only)"`
}

// DispatchStatusArgs holds parsed flags for the dispatch status subcommand.
type DispatchStatusArgs struct {
	StateFile string `targ:"flag,name=state-file,desc=override state file path (testing only)"`
}

// DispatchStopArgs holds parsed flags for the dispatch stop subcommand.
type DispatchStopArgs struct {
	StateFile string `targ:"flag,name=state-file,desc=override state file path (testing only)"`
	ChatFile  string `targ:"flag,name=chat-file,desc=override chat file path (testing only)"`
}

// HoldAcquireArgs holds parsed flags for the hold acquire subcommand.
type HoldAcquireArgs struct {
	Holder    string `targ:"flag,name=holder,desc=agent acquiring the hold"`
	Target    string `targ:"flag,name=target,desc=agent being held"`
	Condition string `targ:"flag,name=condition,desc=auto-release condition"`
	Tag       string `targ:"flag,name=tag,desc=workflow label for bulk operations (e.g. codesign-1)"`
	ChatFile  string `targ:"flag,name=chat-file,desc=override chat file path (testing only)"`
}

// HoldCheckArgs holds parsed flags for the hold check subcommand.
type HoldCheckArgs struct {
	ChatFile string `targ:"flag,name=chat-file,desc=override chat file path (testing only)"`
}

// HoldListArgs holds parsed flags for the hold list subcommand.
type HoldListArgs struct {
	Holder   string `targ:"flag,name=holder,desc=filter by holder agent name"`
	Target   string `targ:"flag,name=target,desc=filter by target agent name"`
	Tag      string `targ:"flag,name=tag,desc=filter by workflow tag"`
	ChatFile string `targ:"flag,name=chat-file,desc=override chat file path (testing only)"`
}

// HoldReleaseArgs holds parsed flags for the hold release subcommand.
type HoldReleaseArgs struct {
	HoldID   string `targ:"flag,name=hold-id,desc=hold ID returned by engram hold acquire"`
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

// AddIntFlag appends a flag and its string value if value is non-zero.
func AddIntFlag(flags []string, name string, value int) []string {
	if value != 0 {
		flags = append(flags, name, strconv.Itoa(value))
	}

	return flags
}

// AgentKillFlags returns the CLI flag args for the agent kill subcommand.
func AgentKillFlags(a AgentKillArgs) []string {
	return BuildFlags("--name", a.Name, "--chat-file", a.ChatFile, "--state-file", a.StateFile)
}

// AgentListFlags returns the CLI flag args for the agent list subcommand.
func AgentListFlags(a AgentListArgs) []string {
	return BuildFlags("--state-file", a.StateFile, "--chat-file", a.ChatFile)
}

// AgentRunFlags builds the flag list for AgentRunArgs.
func AgentRunFlags(a AgentRunArgs) []string {
	return BuildFlags(
		"--name", a.Name,
		"--prompt", a.Prompt,
		"--chat-file", a.ChatFile,
		"--state-file", a.StateFile,
	)
}

// AgentSpawnFlags returns the CLI flag args for the agent spawn subcommand.
func AgentSpawnFlags(a AgentSpawnArgs) []string {
	return BuildFlags(
		"--name", a.Name,
		"--prompt", a.Prompt,
		"--chat-file", a.ChatFile,
		"--state-file", a.StateFile,
	)
}

// AgentWaitReadyFlags returns the CLI flag args for the agent wait-ready subcommand.
func AgentWaitReadyFlags(a AgentWaitReadyArgs) []string {
	return BuildFlags(
		"--name", a.Name,
		"--cursor", strconv.Itoa(a.Cursor),
		"--max-wait", strconv.Itoa(a.MaxWait),
		"--chat-file", a.ChatFile,
	)
}

// BuildAgentGroup builds the targ group for agent subcommands.
//
//nolint:dupl // mirrors BuildDispatchGroup structure; both use the same targ.Group pattern
func BuildAgentGroup(stdout, stderr io.Writer, stdin io.Reader) *targ.TargetGroup {
	return targ.Group("agent",
		targ.Targ(func(a AgentSpawnArgs) {
			args := append([]string{"engram", "agent", "spawn"}, AgentSpawnFlags(a)...)
			RunSafe(args, stdout, stderr, stdin)
		}).Name("spawn").Description("Spawn a new agent in a tmux pane"),
		targ.Targ(func(a AgentKillArgs) {
			args := append([]string{"engram", "agent", "kill"}, AgentKillFlags(a)...)
			RunSafe(args, stdout, stderr, stdin)
		}).Name("kill").Description("Kill a running agent and remove it from the state file"),
		targ.Targ(func(a AgentListArgs) {
			args := append([]string{"engram", "agent", "list"}, AgentListFlags(a)...)
			RunSafe(args, stdout, stderr, stdin)
		}).Name("list").Description("List running agents from the state file"),
		targ.Targ(func(a AgentWaitReadyArgs) {
			args := append([]string{"engram", "agent", "wait-ready"}, AgentWaitReadyFlags(a)...)
			RunSafe(args, stdout, stderr, stdin)
		}).Name("wait-ready").Description("Block until a named agent posts ready"),
		targ.Targ(func(a AgentRunArgs) {
			args := append([]string{"engram", "agent", "run"}, AgentRunFlags(a)...)
			RunSafe(args, stdout, stderr, stdin)
		}).Name("run").Description("Run a claude -p worker pipeline in the current pane"),
	)
}

// BuildChatGroup builds the targ group for chat subcommands.
//
//nolint:dupl // mirrors BuildHoldGroup structure; both use the same targ.Group pattern
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
		targ.Targ(func(a ChatAckWaitArgs) {
			args := append([]string{"engram", "chat", "ack-wait"}, ChatAckWaitFlags(a)...)
			RunSafe(args, stdout, stderr, stdin)
		}).Name("ack-wait").Description("Block until all recipients ACK or WAIT; returns JSON result"),
	)
}

// BuildDispatchGroup builds the targ group for dispatch subcommands.
//
//nolint:dupl // mirrors BuildAgentGroup structure; both use the same targ.Group pattern
func BuildDispatchGroup(stdout, stderr io.Writer, stdin io.Reader) *targ.TargetGroup {
	return targ.Group("dispatch",
		targ.Targ(func(a DispatchStartArgs) {
			args := append([]string{"engram", "dispatch", "start"}, DispatchStartFlags(a)...)
			RunSafe(args, stdout, stderr, stdin)
		}).Name("start").Description("Start the dispatch loop managing one or more workers"),
		targ.Targ(func(a DispatchAssignArgs) {
			args := append([]string{"engram", "dispatch", "assign"}, DispatchAssignFlags(a)...)
			RunSafe(args, stdout, stderr, stdin)
		}).Name("assign").Description("Assign a task to a named worker"),
		targ.Targ(func(a DispatchDrainArgs) {
			args := append([]string{"engram", "dispatch", "drain"}, DispatchDrainFlags(a)...)
			RunSafe(args, stdout, stderr, stdin)
		}).Name("drain").Description("Wait for all in-flight workers to complete"),
		targ.Targ(func(a DispatchStatusArgs) {
			args := append([]string{"engram", "dispatch", "status"}, DispatchStatusFlags(a)...)
			RunSafe(args, stdout, stderr, stdin)
		}).Name("status").Description("Print worker states and queue depth"),
		targ.Targ(func(a DispatchStopArgs) {
			args := append([]string{"engram", "dispatch", "stop"}, DispatchStopFlags(a)...)
			RunSafe(args, stdout, stderr, stdin)
		}).Name("stop").Description("Send shutdown to all workers and exit"),
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

// BuildHoldGroup builds the targ group for hold subcommands.
//
//nolint:dupl // mirrors BuildChatGroup structure; both use the same targ.Group pattern
func BuildHoldGroup(stdout, stderr io.Writer, stdin io.Reader) *targ.TargetGroup {
	return targ.Group("hold",
		targ.Targ(func(a HoldAcquireArgs) {
			args := append([]string{"engram", "hold", "acquire"}, HoldAcquireFlags(a)...)
			RunSafe(args, stdout, stderr, stdin)
		}).Name("acquire").Description("Place a hold on an agent (outputs UUID hold-id)"),
		targ.Targ(func(a HoldReleaseArgs) {
			args := append([]string{"engram", "hold", "release"}, HoldReleaseFlags(a)...)
			RunSafe(args, stdout, stderr, stdin)
		}).Name("release").Description("Release a hold by hold-id"),
		targ.Targ(func(a HoldListArgs) {
			args := append([]string{"engram", "hold", "list"}, HoldListFlags(a)...)
			RunSafe(args, stdout, stderr, stdin)
		}).Name("list").Description("List active (unreleased) holds"),
		targ.Targ(func(a HoldCheckArgs) {
			args := append([]string{"engram", "hold", "check"}, HoldCheckFlags(a)...)
			RunSafe(args, stdout, stderr, stdin)
		}).Name("check").Description("Evaluate auto-conditions and release met holds"),
	)
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

// ChatAckWaitFlags returns the CLI flag args for the chat ack-wait subcommand.
func ChatAckWaitFlags(a ChatAckWaitArgs) []string {
	flags := BuildFlags(
		"--agent", a.Agent,
		"--recipients", a.Recipients,
		"--chat-file", a.ChatFile,
	)

	flags = AddIntFlag(flags, "--cursor", a.Cursor)
	flags = AddIntFlag(flags, "--max-wait", a.MaxWait)

	return flags
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

	flags = AddIntFlag(flags, "--cursor", a.Cursor)
	flags = AddIntFlag(flags, "--max-wait", a.MaxWait)

	return flags
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

// DispatchAssignFlags returns the CLI flag args for the dispatch assign subcommand.
func DispatchAssignFlags(a DispatchAssignArgs) []string {
	return BuildFlags(
		"--agent", a.Agent,
		"--task", a.Task,
		"--chat-file", a.ChatFile,
		"--state-file", a.StateFile,
	)
}

// DispatchDrainFlags returns the CLI flag args for the dispatch drain subcommand.
func DispatchDrainFlags(a DispatchDrainArgs) []string {
	return append(
		BuildFlags("--state-file", a.StateFile),
		AddIntFlag(nil, "--timeout", a.Timeout)...,
	)
}

// DispatchStartFlags returns the CLI flag args for the dispatch start subcommand.
func DispatchStartFlags(a DispatchStartArgs) []string {
	flags := make([]string, 0, len(a.Agent)*2+dispatchStartFlagsNonAgentCap)

	for _, agent := range a.Agent {
		if agent != "" {
			flags = append(flags, "--agent", agent)
		}
	}

	flags = append(flags, BuildFlags(
		"--chat-file", a.ChatFile,
		"--state-file", a.StateFile,
		"--claude-binary", a.ClaudeBinary,
	)...)
	flags = append(flags, AddIntFlag(nil, "--max-concurrent", a.MaxConcurrent)...)

	return flags
}

// DispatchStatusFlags returns the CLI flag args for the dispatch status subcommand.
func DispatchStatusFlags(a DispatchStatusArgs) []string {
	return BuildFlags("--state-file", a.StateFile)
}

// DispatchStopFlags returns the CLI flag args for the dispatch stop subcommand.
func DispatchStopFlags(a DispatchStopArgs) []string {
	return BuildFlags("--state-file", a.StateFile, "--chat-file", a.ChatFile)
}

// HoldAcquireFlags returns the CLI flag args for the hold acquire subcommand.
func HoldAcquireFlags(a HoldAcquireArgs) []string {
	return BuildFlags(
		"--holder", a.Holder,
		"--target", a.Target,
		"--condition", a.Condition,
		"--tag", a.Tag,
		"--chat-file", a.ChatFile,
	)
}

// HoldCheckFlags returns the CLI flag args for the hold check subcommand.
func HoldCheckFlags(a HoldCheckArgs) []string {
	return BuildFlags("--chat-file", a.ChatFile)
}

// HoldListFlags returns the CLI flag args for the hold list subcommand.
func HoldListFlags(a HoldListArgs) []string {
	return BuildFlags(
		"--holder", a.Holder,
		"--target", a.Target,
		"--tag", a.Tag,
		"--chat-file", a.ChatFile,
	)
}

// HoldReleaseFlags returns the CLI flag args for the hold release subcommand.
func HoldReleaseFlags(a HoldReleaseArgs) []string {
	return BuildFlags("--hold-id", a.HoldID, "--chat-file", a.ChatFile)
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

	return append(
		BuildTargets(run),
		BuildChatGroup(stdout, stderr, stdin),
		BuildDispatchGroup(stdout, stderr, stdin),
		BuildHoldGroup(stdout, stderr, stdin),
		BuildAgentGroup(stdout, stderr, stdin),
	)
}

// unexported constants.
const (
	dispatchStartFlagsNonAgentCap = 8
)
