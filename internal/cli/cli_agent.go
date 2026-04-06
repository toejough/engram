package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	agentpkg "engram/internal/agent"
	"engram/internal/chat"
)

// unexported constants.
const (
	spawnAckMaxWait      = 30 * time.Second
	stateFileLockDelay   = 25 * time.Millisecond
	stateFileLockRetries = 200
)

// unexported variables.
var (
	errNotImplemented       = errors.New("not implemented")
	errSpawnNameRequired    = errors.New("agent spawn: --name is required")
	errSpawnPromptRequired  = errors.New("agent spawn: --prompt is required")
	errStateFileLockTimeout = errors.New("state file lock timeout after 5s")
	errUnexpectedTmuxOutput = errors.New("unexpected tmux output")
)

// spawnFlagsResult holds parsed and validated flags for agent spawn.
type spawnFlagsResult struct {
	name, prompt, intentMsg, chatFile, stateFile string
}

// spawnFunc is the type for both the OS spawner and the test-injectable spawner.
type spawnFunc = func(ctx context.Context, name, prompt string) (paneID, sessionID string, err error)

// deriveStateFilePath mirrors deriveChatFilePath but uses the "state" subdirectory.
func deriveStateFilePath(
	override string,
	homeDir func() (string, error),
	getwd func() (string, error),
) (string, error) {
	if override != "" {
		return override, nil
	}

	home, err := homeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}

	cwd, cwdErr := getwd()
	if cwdErr != nil {
		return "", fmt.Errorf("resolving working directory: %w", cwdErr)
	}

	return filepath.Join(DataDirFromHome(home, os.Getenv), "state", ProjectSlugFromPath(cwd)+".toml"), nil
}

// osStateFileLock acquires a lockfile with a 5s timeout for the wider R-M-W critical section.
func osStateFileLock(name string) (func() error, error) {
	for range stateFileLockRetries {
		f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL, chatFileMode) //nolint:gosec
		if err == nil {
			return func() error {
				_ = f.Close()

				return os.Remove(name)
			}, nil
		}

		if !os.IsExist(err) {
			return nil, fmt.Errorf("creating state lock: %w", err)
		}

		time.Sleep(stateFileLockDelay)
	}

	return nil, errStateFileLockTimeout
}

// osTmuxSpawn creates a new tmux window for the agent and returns pane-id and session-id.
func osTmuxSpawn(ctx context.Context, name, prompt string) (paneID, sessionID string, err error) {
	return osTmuxSpawnWith(ctx, "tmux", name, prompt)
}

// osTmuxSpawnWith creates a tmux window using the given binary path and returns pane-id and session-id.
// Extracted so tests can supply a fake binary path without modifying global state.
func osTmuxSpawnWith(ctx context.Context, tmuxBin, name, prompt string) (paneID, sessionID string, err error) {
	out, cmdErr := exec.CommandContext(ctx, tmuxBin, //nolint:gosec
		"new-window",
		"-d",
		"-n", name,
		"-P", "-F", "#{pane_id} #{session_id}",
		"--", "sh", "-c", prompt,
	).Output()
	if cmdErr != nil {
		return "", "", fmt.Errorf("tmux new-window: %w", cmdErr)
	}

	return parseTmuxOutput(out)
}

// parseSpawnFlags parses and validates agent spawn flags.
// Returns flag.ErrHelp if --help was passed (caller should return nil).
func parseSpawnFlags(args []string) (spawnFlagsResult, error) {
	fs := newFlagSet("agent spawn")
	name := fs.String("name", "", "agent name (required)")
	prompt := fs.String("prompt", "", "initial prompt for the agent (required)")
	intentMsg := fs.String("intent-text", "", "task description in spawn intent (optional)")
	chatFile := fs.String("chat-file", "", "override chat file path (testing only)")
	stateFile := fs.String("state-file", "", "override state file path (testing only)")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return spawnFlagsResult{}, fmt.Errorf("agent spawn: %w", parseErr)
	}

	if *name == "" {
		return spawnFlagsResult{}, errSpawnNameRequired
	}

	if *prompt == "" {
		return spawnFlagsResult{}, errSpawnPromptRequired
	}

	return spawnFlagsResult{
		name:      *name,
		prompt:    *prompt,
		intentMsg: *intentMsg,
		chatFile:  *chatFile,
		stateFile: *stateFile,
	}, nil
}

// parseTmuxOutput parses the "tmux new-window -P -F ..." output into pane-id and session-id.
func parseTmuxOutput(out []byte) (paneID, sessionID string, err error) {
	parts := strings.Fields(strings.TrimSpace(string(out)))

	const expectedParts = 2
	if len(parts) != expectedParts {
		return "", "", fmt.Errorf("%w: %q", errUnexpectedTmuxOutput, string(out))
	}

	return parts[0], parts[1], nil
}

// postSpawnIntentAndWait posts a spawn intent to the chat file and waits for engram-agent ACK.
// Fixes #503: binary auto-posts spawn intent + waits for ACK before returning.
func postSpawnIntentAndWait(ctx context.Context, chatFilePath, name, paneID, intentMsg string) error {
	intentText := fmt.Sprintf(
		"Situation: About to spawn agent %q in pane %s.\nBehavior: Agent will post ready when initialized.",
		name, paneID,
	)

	if intentMsg != "" {
		intentText = fmt.Sprintf(
			"Situation: About to spawn agent %q in pane %s. Task: %s\nBehavior: Agent will post ready when initialized.",
			name, paneID, intentMsg,
		)
	}

	poster := newFilePoster(chatFilePath)

	cursor, postErr := poster.Post(chat.Message{
		From:   "system",
		To:     "engram-agent",
		Thread: "lifecycle",
		Type:   "intent",
		Text:   intentText,
	})
	if postErr != nil {
		return fmt.Errorf("posting spawn intent: %w", postErr)
	}

	waiter := &chat.FileAckWaiter{
		FilePath: chatFilePath,
		Watcher:  newFileWatcher(chatFilePath),
		ReadFile: os.ReadFile,
		NowFunc:  time.Now,
		MaxWait:  spawnAckMaxWait,
	}

	_, ackErr := waiter.AckWait(ctx, "system", cursor, []string{"engram-agent"})
	if ackErr != nil {
		return fmt.Errorf("waiting for engram-agent ACK: %w", ackErr)
	}

	return nil
}

// readModifyWriteStateFile performs a locked read-modify-write on the state file.
// Creates the file and its parent directory if they do not exist.
func readModifyWriteStateFile(stateFilePath string, modify func(agentpkg.StateFile) agentpkg.StateFile) error {
	lockPath := stateFilePath + ".lock"

	unlock, lockErr := osStateFileLock(lockPath)
	if lockErr != nil {
		return fmt.Errorf("acquiring state file lock: %w", lockErr)
	}

	defer func() { _ = unlock() }()

	data, readErr := os.ReadFile(stateFilePath) //nolint:gosec
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return fmt.Errorf("reading state file: %w", readErr)
	}

	currentState, parseErr := agentpkg.ParseStateFile(data)
	if parseErr != nil {
		return fmt.Errorf("parsing state file: %w", parseErr)
	}

	currentState = modify(currentState)

	newData, marshalErr := agentpkg.MarshalStateFile(currentState)
	if marshalErr != nil {
		return fmt.Errorf("marshaling state file: %w", marshalErr)
	}

	dir := filepath.Dir(stateFilePath)

	mkdirErr := os.MkdirAll(dir, chatDirMode)
	if mkdirErr != nil {
		return fmt.Errorf("creating state directory: %w", mkdirErr)
	}

	writeErr := os.WriteFile(stateFilePath, newData, chatFileMode)
	if writeErr != nil {
		return fmt.Errorf("writing state file: %w", writeErr)
	}

	return nil
}

// resolveStateFile derives the state file path, wrapping errors with the subcommand name.
func resolveStateFile(
	override, cmd string,
	homeDir func() (string, error),
	getwd func() (string, error),
) (string, error) {
	path, err := deriveStateFilePath(override, homeDir, getwd)
	if err != nil {
		return "", fmt.Errorf("%s: %w", cmd, err)
	}

	return path, nil
}

// runAgentDispatch routes agent subcommands (spawn|kill|list|wait-ready).
func runAgentDispatch(subArgs []string, stdout io.Writer, spawner spawnFunc) error {
	if len(subArgs) < 1 {
		return fmt.Errorf("%w: agent requires a subcommand (spawn|kill|list|wait-ready)", errUsage)
	}

	switch subArgs[0] {
	case "spawn":
		return runAgentSpawn(subArgs[1:], stdout, spawner)
	case "kill":
		return runAgentKill(subArgs[1:], stdout)
	case "list":
		return runAgentList(subArgs[1:], stdout)
	case "wait-ready":
		return runAgentWaitReady(subArgs[1:], stdout)
	default:
		return fmt.Errorf("%w: agent %s", errUnknownCommand, subArgs[0])
	}
}

func runAgentKill(_ []string, _ io.Writer) error { return errNotImplemented }

func runAgentList(_ []string, _ io.Writer) error { return errNotImplemented }

// runAgentSpawn spawns a new agent in a tmux pane, writes the AgentRecord to the state file,
// posts a spawn intent and waits for engram-agent ACK (fixes #503), then prints pane-id|session-id.
func runAgentSpawn(args []string, stdout io.Writer, spawner spawnFunc) error {
	flags, parseErr := parseSpawnFlags(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}

	if parseErr != nil {
		return parseErr
	}

	chatFilePath, pathErr := resolveChatFile(flags.chatFile, "agent spawn", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	stateFilePath, statePathErr := resolveStateFile(flags.stateFile, "agent spawn", os.UserHomeDir, os.Getwd)
	if statePathErr != nil {
		return statePathErr
	}

	ctx, cancel := signalContext()
	defer cancel()

	paneID, sessionID, spawnErr := spawner(ctx, flags.name, flags.prompt)
	if spawnErr != nil {
		return fmt.Errorf("agent spawn: launching pane: %w", spawnErr)
	}

	rmwErr := readModifyWriteStateFile(stateFilePath, func(sf agentpkg.StateFile) agentpkg.StateFile {
		return agentpkg.AddAgent(sf, agentpkg.AgentRecord{
			Name:      flags.name,
			PaneID:    paneID,
			SessionID: sessionID,
			State:     "STARTING",
			SpawnedAt: time.Now().UTC(),
		})
	})
	if rmwErr != nil {
		return fmt.Errorf("agent spawn: updating state file: %w", rmwErr)
	}

	intentErr := postSpawnIntentAndWait(ctx, chatFilePath, flags.name, paneID, flags.intentMsg)
	if intentErr != nil {
		return fmt.Errorf("agent spawn: %w", intentErr)
	}

	_, writeErr := fmt.Fprintf(stdout, "%s|%s\n", paneID, sessionID)
	if writeErr != nil {
		return fmt.Errorf("agent spawn: writing output: %w", writeErr)
	}

	return nil
}

func runAgentWaitReady(_ []string, _ io.Writer) error { return errNotImplemented }
