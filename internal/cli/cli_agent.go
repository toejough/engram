package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
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
	errAgentKillNameRequired = errors.New("agent kill: --name is required")
	errNotImplemented        = errors.New("not implemented")
	errSpawnNameRequired     = errors.New("agent spawn: --name is required")
	errSpawnPromptRequired   = errors.New("agent spawn: --prompt is required")
	errStateFileLockTimeout  = errors.New("state file lock timeout after 5s")
	errUnexpectedTmuxOutput  = errors.New("unexpected tmux output")
	errUnmetHoldCondition    = errors.New("condition not satisfied; release it first")
	testPaneKiller           func(paneID string) error //nolint:gochecknoglobals // test-overridable pane killer
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

// evaluateAndRelease checks one hold: returns error if unmet, posts release message if met.
func evaluateAndRelease(hold chat.HoldRecord, messages []chat.Message, poster *chat.FilePoster) error {
	met, _ := chat.EvaluateCondition(hold, messages)
	if !met {
		return fmt.Errorf("agent kill: active hold %w", newUnmetHoldError(hold.HoldID, hold.Condition))
	}

	releaseText, marshalErr := marshalReleasePayload(hold.HoldID)
	if marshalErr != nil {
		return fmt.Errorf("agent kill: marshaling release: %w", marshalErr)
	}

	_, postErr := poster.Post(chat.Message{
		From:   "system",
		To:     "all",
		Thread: "hold",
		Type:   "hold-release",
		Text:   string(releaseText),
	})
	if postErr != nil {
		return fmt.Errorf("agent kill: posting release for %s: %w", hold.HoldID, postErr)
	}

	return nil
}

// killAgentPane kills the tmux pane for the agent if a pane ID was found.
// Uses testPaneKiller in tests; falls back to osTmuxKillPane in production.
// Silently succeeds if the pane is already gone.
func killAgentPane(paneID string) error {
	if paneID == "" {
		return nil
	}

	killFn := testPaneKiller
	if killFn == nil {
		killFn = osTmuxKillPane
	}

	killErr := killFn(paneID)
	if killErr != nil && !strings.Contains(killErr.Error(), "no such pane") {
		return fmt.Errorf("agent kill: killing pane %s: %w", paneID, killErr)
	}

	return nil
}

func newUnmetHoldError(holdID, condition string) error {
	return fmt.Errorf("%s (condition: %s): %w", holdID, condition, errUnmetHoldCondition)
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

// osTmuxKillPane kills the tmux pane with the given pane-id.
// Returns nil if the pane is already gone ("can't find pane") — graceful shutdown
// may have auto-closed the pane before kill is called.
func osTmuxKillPane(paneID string) error {
	cmd := exec.CommandContext(context.Background(), "tmux", "kill-pane", "-t", paneID) //nolint:gosec

	out, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(out), "can't find pane") {
		return fmt.Errorf("tmux kill-pane: %w", err)
	}

	return nil
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

// parseAgentKillFlags parses the agent-kill flag set. Returns (name, chatFile, stateFile, err).
// Returns ("", "", "", nil) when --help was requested.
func parseAgentKillFlags(args []string) (string, string, string, error) {
	flagSet := newFlagSet("agent kill")
	name := flagSet.String("name", "", "agent name to kill (required)")
	chatFileFlag := flagSet.String("chat-file", "", "override chat file path (testing only)")
	stateFileFlag := flagSet.String("state-file", "", "override state file path (testing only)")

	parseErr := flagSet.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return "", "", "", nil
	}

	if parseErr != nil {
		return "", "", "", fmt.Errorf("agent kill: %w", parseErr)
	}

	if *name == "" {
		return "", "", "", errAgentKillNameRequired
	}

	return *name, *chatFileFlag, *stateFileFlag, nil
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

// reconstructStateFileFromChat builds a best-effort StateFile from the chat history.
// Reads all lifecycle messages to extract agent names from `ready` messages.
// Reads all hold-acquire/release pairs via chat.ScanActiveHolds.
// Result is in-memory only — NOT written to disk. Agent list may be partial
// (agents that posted ready but whose done/shutdown was missed are included;
// agents that never posted ready are absent).
func reconstructStateFileFromChat(
	chatFilePath string,
	readFile func(string) ([]byte, error),
) (agentpkg.StateFile, error) {
	messages, loadErr := loadChatMessages(chatFilePath, readFile)
	if loadErr != nil {
		return agentpkg.StateFile{}, fmt.Errorf("loading chat for reconstruction: %w", loadErr)
	}

	state := agentpkg.StateFile{}

	// Reconstruct holds from chat log — ScanActiveHolds already handles acquire/release pairs.
	activeHolds := chat.ScanActiveHolds(messages)
	for _, hold := range activeHolds {
		state = agentpkg.AddHold(state, agentpkg.HoldEntry{
			HoldID:     hold.HoldID,
			Holder:     hold.Holder,
			Target:     hold.Target,
			Condition:  hold.Condition,
			Tag:        hold.Tag,
			AcquiredTS: hold.AcquiredTS,
		})
	}

	// Reconstruct agents from ready messages. Track which agents posted done/shutdown
	// so we can exclude them (they exited cleanly).
	doneAgents := make(map[string]bool)

	for _, msg := range messages {
		if msg.Type == "done" || msg.Type == "shutdown" {
			doneAgents[msg.From] = true
		}
	}

	seen := make(map[string]bool)

	for _, msg := range messages {
		if msg.Type != "ready" || seen[msg.From] || doneAgents[msg.From] {
			continue
		}

		seen[msg.From] = true
		state = agentpkg.AddAgent(state, agentpkg.AgentRecord{
			Name:  msg.From,
			State: "UNKNOWN", // reconstruction cannot verify live state without tmux
		})
	}

	return state, nil
}

// releaseMetHoldsForAgent checks holds targeting the named agent. Returns an error if any unmet hold is found.
// Releases holds whose conditions are already met.
func releaseMetHoldsForAgent(chatFilePath, agentName string, messages []chat.Message) error {
	activeHolds := chat.ScanActiveHolds(messages)
	poster := newFilePoster(chatFilePath)

	for _, hold := range activeHolds {
		if hold.Target != agentName {
			continue
		}

		releaseErr := evaluateAndRelease(hold, messages, poster)
		if releaseErr != nil {
			return releaseErr
		}
	}

	return nil
}

// removeAgentFromStateFile removes the named agent from the state file and returns its pane ID.
func removeAgentFromStateFile(stateFilePath, agentName string) (string, error) {
	var paneID string

	rmwErr := readModifyWriteStateFile(stateFilePath, func(stateFile agentpkg.StateFile) agentpkg.StateFile {
		for _, record := range stateFile.Agents {
			if record.Name == agentName {
				paneID = record.PaneID

				break
			}
		}

		return agentpkg.RemoveAgent(stateFile, agentName)
	})

	return paneID, rmwErr
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

func runAgentKill(args []string, stdout io.Writer) error {
	agentName, chatFileFlag, stateFileFlag, flagErr := parseAgentKillFlags(args)
	if flagErr != nil {
		return flagErr
	}

	if agentName == "" {
		return nil // --help was requested
	}

	chatFilePath, pathErr := resolveChatFile(chatFileFlag, "agent kill", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	stateFilePath, statePathErr := resolveStateFile(stateFileFlag, "agent kill", os.UserHomeDir, os.Getwd)
	if statePathErr != nil {
		return statePathErr
	}

	// Evaluate hold conditions using domain functions directly (no self-invocation).
	messages, loadErr := loadChatMessages(chatFilePath, os.ReadFile)
	if loadErr != nil {
		return fmt.Errorf("agent kill: %w", loadErr)
	}

	releaseErr := releaseMetHoldsForAgent(chatFilePath, agentName, messages)
	if releaseErr != nil {
		return releaseErr
	}

	paneID, rmwErr := removeAgentFromStateFile(stateFilePath, agentName)
	if rmwErr != nil {
		return fmt.Errorf("agent kill: updating state file: %w", rmwErr)
	}

	killErr := killAgentPane(paneID)
	if killErr != nil {
		return killErr
	}

	writeErr := writeKilledLine(stdout, agentName)
	if writeErr != nil {
		return fmt.Errorf("agent kill: %w", writeErr)
	}

	return nil
}

func runAgentList(args []string, stdout io.Writer) error {
	fs := newFlagSet("agent list")
	stateFile := fs.String("state-file", "", "override state file path (testing only)")
	chatFile := fs.String("chat-file", "", "override chat file path (used for reconstruction fallback)")

	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}

	if parseErr != nil {
		return fmt.Errorf("agent list: %w", parseErr)
	}

	stateFilePath, statePathErr := resolveStateFile(*stateFile, "agent list", os.UserHomeDir, os.Getwd)
	if statePathErr != nil {
		return statePathErr
	}

	data, readErr := os.ReadFile(stateFilePath) //nolint:gosec
	if errors.Is(readErr, os.ErrNotExist) {
		// Spec §6.3: detect missing state file and attempt reconstruction from chat history.
		// This is cheap in Phase 3; expensive after Phase 5 when full-file parse paths may be removed.
		chatFilePath, chatPathErr := resolveChatFile(*chatFile, "agent list", os.UserHomeDir, os.Getwd)
		if chatPathErr != nil {
			slog.Warn("agent list: state file missing, reconstruction skipped (could not resolve chat file)", "err", chatPathErr)
			return nil
		}

		return runAgentListFromChat(chatFilePath, stdout)
	}

	if readErr != nil {
		return fmt.Errorf("agent list: reading state file: %w", readErr)
	}

	parsed, parseStateErr := agentpkg.ParseStateFile(data)
	if parseStateErr != nil {
		return fmt.Errorf("agent list: %w", parseStateErr)
	}

	enc := json.NewEncoder(stdout)

	for _, rec := range parsed.Agents {
		encErr := enc.Encode(rec)
		if encErr != nil {
			return fmt.Errorf("agent list: encoding record: %w", encErr)
		}
	}

	return nil
}

// runAgentListFromChat is the reconstruction fallback for runAgentList.
// Called when the state file is missing. Emits a warning, reconstructs an
// in-memory StateFile from chat history, and lists agents from that.
func runAgentListFromChat(chatFilePath string, stdout io.Writer) error {
	reconstructed, reconErr := reconstructStateFileFromChat(chatFilePath, os.ReadFile)
	if reconErr != nil {
		slog.Warn("agent list: state file missing, reconstruction failed", "err", reconErr)
		return nil
	}

	slog.Warn("agent list: state file missing, using reconstructed state from chat history (agent list may be incomplete)")

	enc := json.NewEncoder(stdout)

	for _, rec := range reconstructed.Agents {
		encErr := enc.Encode(rec)
		if encErr != nil {
			return fmt.Errorf("agent list: encoding reconstructed record: %w", encErr)
		}
	}

	return nil
}

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

// writeKilledLine writes the "killed <name>" confirmation line to stdout.
func writeKilledLine(stdout io.Writer, agentName string) error {
	_, err := fmt.Fprintf(stdout, "killed %s\n", agentName)
	if err != nil {
		return fmt.Errorf("writing killed line: %w", err)
	}

	return nil
}
