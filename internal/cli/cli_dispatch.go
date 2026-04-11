package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	agentpkg "engram/internal/agent"
	"engram/internal/chat"
)

// WorkerConfig holds configuration for a single dispatch worker.
type WorkerConfig struct {
	Name   string
	Prompt string
}

// unexported constants.
const (
	agentStateDead          = "DEAD"
	defaultDrainTimeoutSecs = 60
	defaultMaxConcurrent    = 4
	drainPollInterval       = 250 * time.Millisecond
	maxDeferredQueueCap     = 100
	workerChannelCap        = 16
)

// unexported variables.
var (
	errDispatchAgentRequired     = errors.New("dispatch assign: --agent is required")
	errDispatchStartNeedsAgent   = errors.New("dispatch start: at least one --agent flag is required")
	errDispatchSubcommandMissing = errors.New("dispatch: subcommand required (start|assign|drain|stop|status)")
	errDispatchTaskRequired      = errors.New("dispatch assign: --task is required")
	errDispatchUnknownSubcmd     = errors.New("dispatch: unknown subcommand")
)

// dispatchWatchResult holds the result of one Watch call in dispatchLoop.
type dispatchWatchResult struct {
	Msg    chat.Message
	Cursor int
	Err    error
}

// drainResult is the JSON output of runDispatchDrain.
type drainResult struct {
	Status      string `json:"status"`
	ActiveCount int    `json:"activeCount,omitempty"`
}

// holdCheckerFunc reports whether a named worker is currently held.
type holdCheckerFunc func(workerName string) bool

// multiStringFlag is a flag.Value that accumulates repeated string flags.
type multiStringFlag []string

func (f *multiStringFlag) Set(v string) error {
	*f = append(*f, v)

	return nil
}

func (f *multiStringFlag) String() string {
	if f == nil {
		return ""
	}

	return strings.Join(*f, ",")
}

// addFreshStartingRecords removes all existing records for each worker and adds a single
// fresh STARTING record. This ensures exactly one record per worker name, preventing
// stale SILENT/DEAD records from confusing state-update functions that match by name.
func addFreshStartingRecords(stateFile agentpkg.StateFile, workers []WorkerConfig) agentpkg.StateFile {
	for _, worker := range workers {
		stateFile = agentpkg.RemoveAgent(stateFile, worker.Name)
		stateFile = agentpkg.AddAgent(stateFile, agentpkg.AgentRecord{
			Name:      worker.Name,
			State:     "STARTING",
			SpawnedAt: time.Now().UTC(),
		})
	}

	return stateFile
}

// deferMessage adds a message to the deferred queue, or logs overflow if at cap.
func deferMessage(
	deferred map[string][]chat.Message,
	poster *chat.FilePoster,
	worker string,
	msg chat.Message,
) {
	if len(deferred[worker]) < maxDeferredQueueCap {
		deferred[worker] = append(deferred[worker], msg)
	} else {
		postQueueOverflow(poster, worker)
	}
}

// dispatchLoop watches all chat messages and routes actionable types to worker channels.
// Runs until ctx is cancelled. Uses a select loop to handle both watch results and
// silentCh notifications (ARCH-A5).
func dispatchLoop(
	ctx context.Context,
	workerChans map[string]chan chat.Message,
	stateFilePath, chatFilePath string,
	cursor int,
	silentCh <-chan string, // receives worker name when session completes (ARCH-A5)
) error {
	return dispatchLoopWith(ctx, workerChans, stateFilePath, chatFilePath, cursor, silentCh, nil, nil)
}

// dispatchLoopWith is dispatchLoop with injectable hold checker and poster for testing.
func dispatchLoopWith(
	ctx context.Context,
	workerChans map[string]chan chat.Message,
	stateFilePath, chatFilePath string,
	cursor int,
	silentCh <-chan string,
	holdChecker holdCheckerFunc,
	poster *chat.FilePoster,
) error {
	deferred := make(map[string][]chat.Message, len(workerChans))
	for name := range workerChans {
		deferred[name] = make([]chat.Message, 0, maxDeferredQueueCap)
	}

	watcher := newFileWatcher(chatFilePath)

	// Actionable message types + hold-release for drain trigger (agent-e2e A4).
	routeTypes := []string{"intent", "wait", "shutdown", "hold-release"}

	// Crash recovery: batch-scan any messages already in the file from cursor.
	// Watch only returns one message per call and advances cursor to end-of-file,
	// so messages between the crash cursor and end-of-file would be lost without
	// this initial scan.
	data, scanErr := os.ReadFile(chatFilePath) //nolint:gosec
	if scanErr == nil {
		suffix := suffixAtLine(data, cursor)

		for _, msg := range chat.ParseMessagesSafe(suffix) {
			routeMessageWithPoster(workerChans, deferred, holdChecker, stateFilePath, chatFilePath, poster, msg, cursor)
		}

		cursor = bytes.Count(data, []byte("\n"))
	}

	msgCh := make(chan dispatchWatchResult, 1)

	startWatch := func() {
		go func() {
			msg, newCursor, err := watcher.Watch(ctx, "", cursor, routeTypes)
			msgCh <- dispatchWatchResult{Msg: msg, Cursor: newCursor, Err: err}
		}()
	}

	startWatch()

	for {
		select {
		case <-ctx.Done():
			return nil

		case name := <-silentCh:
			handleWorkerSilent(name, workerChans, deferred, holdChecker, stateFilePath, cursor, poster)

		case res := <-msgCh:
			if res.Err != nil {
				if ctx.Err() != nil {
					return nil //nolint:nilerr // ctx cancel during watch = clean exit
				}

				return res.Err
			}

			cursor = res.Cursor
			routeMessageWithPoster(workerChans, deferred, holdChecker, stateFilePath, chatFilePath, poster, res.Msg, cursor)

			startWatch()
		}
	}
}

// drainDeferredQueue sends all deferred messages for a worker to its channel
// and advances last_delivered_cursor for each.
// Called on hold release or ACTIVE→SILENT transition.
func drainDeferredQueue(
	worker string,
	workerChans map[string]chan chat.Message,
	deferred map[string][]chat.Message,
	stateFilePath string,
	cursor int,
	poster *chat.FilePoster,
) {
	ch, ok := workerChans[worker]
	if !ok {
		return
	}

	for _, msg := range deferred[worker] {
		ch <- msg

		updateLastDeliveredCursor(stateFilePath, worker, cursor)
		postRoutingInfo(poster, worker, msg, cursor)
	}

	deferred[worker] = deferred[worker][:0]
}

// handleHoldRelease processes a hold-release message: resolves the hold target and drains
// the deferred queue for that worker.
func handleHoldRelease(
	workerChans map[string]chan chat.Message,
	deferred map[string][]chat.Message,
	stateFilePath, chatFilePath string,
	cursor int,
	poster *chat.FilePoster,
	msgText string,
) {
	var payload struct {
		HoldID string `json:"hold-id"` //nolint:tagliatelle // protocol field matches binary wire format
	}

	unmarshalErr := json.Unmarshal([]byte(msgText), &payload)
	if unmarshalErr != nil || payload.HoldID == "" {
		return
	}

	target := resolveHoldTarget(payload.HoldID, chatFilePath)
	if target != "" {
		drainDeferredQueue(target, workerChans, deferred, stateFilePath, cursor, poster)
	}
}

// handleWorkerSilent is called when a worker transitions ACTIVE→SILENT.
// It drains the deferred queue only if the worker is not currently on hold.
// If held, deferred messages remain until the hold is released.
func handleWorkerSilent(
	name string,
	workerChans map[string]chan chat.Message,
	deferred map[string][]chat.Message,
	holdChecker holdCheckerFunc,
	stateFilePath string,
	cursor int,
	poster *chat.FilePoster,
) {
	if holdChecker == nil || !holdChecker(name) {
		drainDeferredQueue(name, workerChans, deferred, stateFilePath, cursor, poster)
	}
}

// hasStartingRecord reports whether agents contains a STARTING record for name.
func hasStartingRecord(agents []agentpkg.AgentRecord, name string) bool {
	for _, rec := range agents {
		if rec.Name == name && rec.State == "STARTING" {
			return true
		}
	}

	return false
}

// initWorkerStateRecords writes STARTING records for all workers before goroutines start.
// Any existing STARTING or ACTIVE record for a worker is marked DEAD first (stale from a
// prior crashed dispatch). An info message is posted to chat for each stale record.
func initWorkerStateRecords(stateFilePath, chatFilePath string, workers []WorkerConfig) error {
	workerSet := make(map[string]bool, len(workers))
	for _, w := range workers {
		workerSet[w.Name] = true
	}

	var staleNames []string

	rmwErr := readModifyWriteStateFile(stateFilePath, func(sf agentpkg.StateFile) agentpkg.StateFile {
		staleNames = nil
		sf, staleNames = markStaleWorkersDeadAndCollect(sf, workerSet)

		return addFreshStartingRecords(sf, workers)
	})
	if rmwErr != nil {
		return rmwErr
	}

	if len(staleNames) == 0 {
		return nil
	}

	poster := newFilePoster(chatFilePath)

	for _, name := range staleNames {
		_, _ = poster.Post(chat.Message{
			From:   "dispatch",
			To:     "all",
			Thread: "lifecycle",
			Type:   "info",
			Text:   fmt.Sprintf("Stale worker %s found in STARTING/ACTIVE state — marking DEAD.", name),
		})
	}

	return nil
}

// isWorkerActive reports whether the named worker has state=ACTIVE in the state file.
func isWorkerActive(stateFilePath, workerName string) bool {
	data, err := os.ReadFile(stateFilePath) //nolint:gosec
	if err != nil {
		return false
	}

	state, err := agentpkg.ParseStateFile(data)
	if err != nil {
		return false
	}

	for _, rec := range state.Agents {
		if rec.Name == workerName {
			return rec.State == "ACTIVE"
		}
	}

	return false
}

// makeHoldChecker returns a holdCheckerFunc that reads the chat file on each call
// and returns true if the named worker has any unreleased holds.
func makeHoldChecker(chatFilePath string) holdCheckerFunc {
	return func(workerName string) bool {
		data, err := os.ReadFile(chatFilePath) //nolint:gosec
		if err != nil {
			return false
		}

		messages := chat.ParseMessagesSafe(data)
		holds := chat.ScanActiveHolds(messages)

		for _, hold := range holds {
			if hold.Target == workerName {
				return true
			}
		}

		return false
	}
}

// markStaleWorkersDeadAndCollect marks STARTING or ACTIVE records for known workers as DEAD.
// Returns the updated state file and the names of any records that were marked.
func markStaleWorkersDeadAndCollect(
	stateFile agentpkg.StateFile,
	workerSet map[string]bool,
) (agentpkg.StateFile, []string) {
	var stale []string

	for i, rec := range stateFile.Agents {
		if workerSet[rec.Name] && (rec.State == "STARTING" || rec.State == "ACTIVE") {
			stateFile.Agents[i].State = agentStateDead

			stale = append(stale, rec.Name)
		}
	}

	return stateFile, stale
}

// parseIntentMarkerTO extracts the TO: subfield from an INTENT: marker text.
// Returns the trimmed TO value, or "engram-agent" if no TO: subfield is present.
// Format: "TO: recipient1, recipient2. Situation: ..."
func parseIntentMarkerTO(markerText string) string {
	const toPrefix = "TO: "

	if !strings.HasPrefix(markerText, toPrefix) {
		return "engram-agent"
	}

	rest := markerText[len(toPrefix):]

	before, _, found := strings.Cut(rest, ". ")
	if !found {
		return strings.TrimSpace(rest)
	}

	return strings.TrimSpace(before)
}

// postQueueOverflow posts a queue-overflow warning to chat.
func postQueueOverflow(poster *chat.FilePoster, worker string) {
	if poster == nil {
		return
	}

	_, _ = poster.Post(chat.Message{
		From:   "dispatch",
		To:     "all",
		Thread: "dispatch",
		Type:   "info",
		Text:   fmt.Sprintf("dispatch: deferredQueue overflow for worker=%s; oldest message dropped", worker),
	})
}

// postRoutingInfo posts a dispatch observability info message to chat.
func postRoutingInfo(poster *chat.FilePoster, recipient string, msg chat.Message, cursor int) {
	if poster == nil {
		return
	}

	text := fmt.Sprintf(
		"dispatch: routed type=%s from=%s to=%s cursor=%d",
		msg.Type, msg.From, recipient, cursor,
	)

	_, _ = poster.Post(chat.Message{
		From:   "dispatch",
		To:     "all",
		Thread: "dispatch",
		Type:   "info",
		Text:   text,
	})
}

// resolveHoldTarget looks up the target worker for a hold-id by scanning
// hold-acquire messages in the chat file. Returns "" if the hold-id is not found.
func resolveHoldTarget(holdID, chatFilePath string) string {
	data, err := os.ReadFile(chatFilePath) //nolint:gosec
	if err != nil {
		return ""
	}

	for _, msg := range chat.ParseMessagesSafe(data) {
		if msg.Type != "hold-acquire" {
			continue
		}

		var record chat.HoldRecord
		if json.Unmarshal([]byte(msg.Text), &record) == nil && record.HoldID == holdID {
			return record.Target
		}
	}

	return ""
}

// resolveRecipients expands "all" to all registered worker names,
// or parses a comma-separated recipient list.
func resolveRecipients(toField string, workers map[string]chan chat.Message) []string {
	if strings.EqualFold(strings.TrimSpace(toField), "all") {
		names := make([]string, 0, len(workers))
		for name := range workers {
			names = append(names, name)
		}

		return names
	}

	parts := strings.Split(toField, ",")
	result := make([]string, 0, len(parts))

	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// routeMessage processes a single incoming message and delivers it to the
// appropriate worker channel(s) or deferred queue(s).
// This is the core routing logic of dispatchLoop, extracted for testability.
func routeMessage(
	workerChans map[string]chan chat.Message,
	deferred map[string][]chat.Message,
	holdChecker holdCheckerFunc,
	stateFilePath, chatFilePath string,
	msg chat.Message,
	cursor int,
) {
	routeMessageWithPoster(workerChans, deferred, holdChecker, stateFilePath, chatFilePath, nil, msg, cursor)
}

// routeMessageWithPoster is routeMessage with an optional chat.FilePoster for observability.
func routeMessageWithPoster(
	workerChans map[string]chan chat.Message,
	deferred map[string][]chat.Message,
	holdChecker holdCheckerFunc,
	stateFilePath, chatFilePath string,
	poster *chat.FilePoster,
	msg chat.Message,
	cursor int,
) {
	// Hold-release: resolve target from chat file and drain deferred queue.
	if msg.Type == "hold-release" {
		handleHoldRelease(workerChans, deferred, stateFilePath, chatFilePath, cursor, poster, msg.Text)

		return
	}

	// Route only actionable types that require the recipient to take action.
	if msg.Type != "intent" && msg.Type != "wait" && msg.Type != "shutdown" {
		return
	}

	for _, recipient := range resolveRecipients(msg.To, workerChans) {
		routeToRecipient(workerChans, deferred, holdChecker, stateFilePath, poster, msg, cursor, recipient)
	}
}

// routeToRecipient delivers a single message to one worker channel or deferred queue.
func routeToRecipient(
	workerChans map[string]chan chat.Message,
	deferred map[string][]chat.Message,
	holdChecker holdCheckerFunc,
	stateFilePath string,
	poster *chat.FilePoster,
	msg chat.Message,
	cursor int,
	recipient string,
) {
	ch, ok := workerChans[recipient]
	if !ok {
		return
	}

	// FROM-filter: do not route messages back to sender (prevents self-resume loop).
	if msg.From == recipient {
		return
	}

	// Hold check: defer if held; do NOT drop.
	if holdChecker != nil && holdChecker(recipient) {
		deferMessage(deferred, poster, recipient, msg)

		return
	}

	// ACTIVE check: defer if worker is currently in a session.
	// STARTING workers are NOT deferred — channel buffering handles them.
	if isWorkerActive(stateFilePath, recipient) {
		deferMessage(deferred, poster, recipient, msg)

		return
	}

	// Non-blocking send (ARCH-A6): if channel is full, defer to deferredQueue.
	// This prevents dispatchLoop from blocking on a stalled worker.
	select {
	case ch <- msg:
		updateLastDeliveredCursor(stateFilePath, recipient, cursor)
		postRoutingInfo(poster, recipient, msg, cursor)
	default:
		deferMessage(deferred, poster, recipient, msg)
	}
}

// runDispatch sets up workers and runs the dispatch loop until ctx is cancelled.
func runDispatch(
	ctx context.Context,
	workers []WorkerConfig,
	maxConcurrent int,
	chatFilePath, stateFilePath, claudeBinary string,
	stdout io.Writer,
) error {
	sem := make(chan struct{}, maxConcurrent)

	intentChans := make(map[string]chan chat.Message, len(workers))
	for _, w := range workers {
		intentChans[w.Name] = make(chan chat.Message, workerChannelCap)
	}

	cursor, cursorErr := chatFileCursor(chatFilePath, os.ReadFile)
	if cursorErr != nil {
		return fmt.Errorf("dispatch: reading chat cursor: %w", cursorErr)
	}

	// Initialize state file records for all workers as STARTING before spawning goroutines.
	// Stale STARTING/ACTIVE records from a prior crashed dispatch are marked DEAD first.
	initErr := initWorkerStateRecords(stateFilePath, chatFilePath, workers)
	if initErr != nil {
		return fmt.Errorf("dispatch: initializing state file: %w", initErr)
	}

	// silentCh notifies dispatchLoop when a worker session completes (ARCH-A5).
	silentCh := make(chan string, len(workers))

	var wg sync.WaitGroup

	for _, w := range workers {
		wg.Add(1)

		go func(cfg WorkerConfig) {
			defer wg.Done()

			intentCh := intentChans[cfg.Name]
			runWorkerUnderSemaphore(ctx, cfg, sem, chatFilePath, stateFilePath, claudeBinary, intentCh, silentCh, stdout)
		}(w)
	}

	// Run dispatchLoopWith until ctx is cancelled.
	// Wire real hold checker and poster for production observability (criteria 6 and 9).
	poster := newFilePoster(chatFilePath)
	loopErr := dispatchLoopWith(ctx, intentChans, stateFilePath, chatFilePath, cursor, silentCh,
		makeHoldChecker(chatFilePath), poster)

	wg.Wait()

	return loopErr
}

// runDispatchAssign implements the "engram dispatch assign" subcommand.
// Posts an intent message to the chat file addressed to the named worker.
func runDispatchAssign(args []string, stdout io.Writer) error {
	var (
		agentName string
		task      string
		chatFile  string
	)

	fs := newFlagSet("dispatch assign")
	fs.StringVar(&agentName, "agent", "", "target agent name")
	fs.StringVar(&task, "task", "", "task description to assign")
	fs.StringVar(&chatFile, "chat-file", "", "override chat file path (testing only)")

	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("dispatch assign: %w", err)
	}

	if agentName == "" {
		return errDispatchAgentRequired
	}

	if task == "" {
		return errDispatchTaskRequired
	}

	chatFilePath, chatErr := deriveChatFilePath(chatFile, os.UserHomeDir, os.Getwd)
	if chatErr != nil {
		return fmt.Errorf("dispatch assign: %w", chatErr)
	}

	poster := newFilePoster(chatFilePath)

	cursor, _ := poster.Post(chat.Message{
		From:   "dispatch",
		To:     agentName,
		Thread: "dispatch",
		Type:   "intent",
		Text:   task,
	})

	_, _ = fmt.Fprintf(stdout, "Assigned to %s at cursor %d\n", agentName, cursor)

	return nil
}

// runDispatchDispatch routes dispatch subcommands.
// ctx is passed to runDispatchStart so signal handling can cancel the dispatch loop.
func runDispatchDispatch(ctx context.Context, subArgs []string, stdout io.Writer) error {
	if len(subArgs) == 0 {
		return errDispatchSubcommandMissing
	}

	switch subArgs[0] {
	case "start":
		return runDispatchStart(ctx, subArgs[1:], stdout)
	case "assign":
		return runDispatchAssign(subArgs[1:], stdout)
	case "drain":
		return runDispatchDrain(subArgs[1:], stdout)
	case "stop":
		return runDispatchStop(subArgs[1:], stdout)
	case "status":
		return runDispatchStatus(subArgs[1:], stdout)
	default:
		return fmt.Errorf("%w %q (want: start|assign|drain|stop|status)", errDispatchUnknownSubcmd, subArgs[0])
	}
}

// runDispatchDrain implements the "engram dispatch drain" subcommand.
// Waits for all workers to reach SILENT state, then outputs a JSON summary.
func runDispatchDrain(args []string, stdout io.Writer) error {
	var (
		timeoutSecs int
		stateFile   string
	)

	fs := newFlagSet("dispatch drain")
	fs.IntVar(&timeoutSecs, "secs", defaultDrainTimeoutSecs, "drain timeout in seconds")
	fs.StringVar(&stateFile, "state-file", "", "override state file path (testing only)")

	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("dispatch drain: %w", err)
	}

	stateFilePath, stateErr := resolveStateFile(stateFile, "dispatch drain", os.UserHomeDir, os.Getwd)
	if stateErr != nil {
		return fmt.Errorf("dispatch drain: %w", stateErr)
	}

	deadline := time.Now().Add(time.Duration(timeoutSecs) * time.Second)

	for time.Now().Before(deadline) {
		data, readErr := os.ReadFile(stateFilePath) //nolint:gosec
		if readErr != nil {
			break
		}

		parsedState, parseErr := agentpkg.ParseStateFile(data)
		if parseErr != nil {
			break
		}

		activeCount := agentpkg.ActiveWorkerCount(parsedState)
		if activeCount == 0 {
			drainedSummary, marshalErr := json.Marshal(drainResult{Status: "drained"})
			if marshalErr != nil {
				return fmt.Errorf("dispatch drain: marshaling summary: %w", marshalErr)
			}

			_, _ = fmt.Fprintln(stdout, string(drainedSummary))

			return nil
		}

		time.Sleep(drainPollInterval)
	}

	timeoutSummary, marshalErr := json.Marshal(drainResult{Status: "timeout"})
	if marshalErr != nil {
		return fmt.Errorf("dispatch drain: marshaling timeout: %w", marshalErr)
	}

	_, _ = fmt.Fprintln(stdout, string(timeoutSummary))

	return nil
}

// runDispatchStart implements the "engram dispatch start" subcommand.
// Spawns workers, prints startup info block, and blocks running the dispatch loop.
func runDispatchStart(ctx context.Context, args []string, stdout io.Writer) error {
	var (
		agentFlags    multiStringFlag
		maxConcurrent int
		chatFile      string
		stateFile     string
		claudeBinary  string
	)

	fs := newFlagSet("dispatch start")
	fs.Var(&agentFlags, "agent", "worker agent name (repeatable)")
	fs.IntVar(&maxConcurrent, "max-concurrent", defaultMaxConcurrent, "maximum concurrent worker sessions")
	fs.StringVar(&chatFile, "chat-file", "", "override chat file path (testing only)")
	fs.StringVar(&stateFile, "state-file", "", "override state file path (testing only)")
	fs.StringVar(&claudeBinary, "claude-binary", "claude", "override claude binary path (testing only)")

	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("dispatch start: %w", err)
	}

	if len(agentFlags) == 0 {
		return errDispatchStartNeedsAgent
	}

	chatFilePath, chatErr := deriveChatFilePath(chatFile, os.UserHomeDir, os.Getwd)
	if chatErr != nil {
		return fmt.Errorf("dispatch start: %w", chatErr)
	}

	stateFilePath, stateErr := resolveStateFile(stateFile, "dispatch start", os.UserHomeDir, os.Getwd)
	if stateErr != nil {
		return fmt.Errorf("dispatch start: %w", stateErr)
	}

	// Print startup info block before blocking.
	agentNames := strings.Join(agentFlags, ", ")
	_, _ = fmt.Fprintf(stdout, "Dispatch started. Workers: %s\n", agentNames)
	_, _ = fmt.Fprintf(stdout, "Chat file: %s\n", chatFilePath)
	_, _ = fmt.Fprintf(stdout, "Max concurrent: %d\n", maxConcurrent)
	_, _ = fmt.Fprintf(stdout, "Status:  engram dispatch status\n")
	_, _ = fmt.Fprintf(stdout, "Assign:  engram dispatch assign --agent <name> --task '<task>'\n")
	_, _ = fmt.Fprintf(stdout, "Stop:    engram dispatch stop (or Ctrl-C)\n")

	workers := make([]WorkerConfig, 0, len(agentFlags))
	for _, name := range agentFlags {
		workers = append(workers, WorkerConfig{Name: name, Prompt: ""})
	}

	return runDispatch(ctx, workers, maxConcurrent, chatFilePath, stateFilePath, claudeBinary, stdout)
}

// runDispatchStatus implements the "engram dispatch status" subcommand.
// Outputs a JSON object with current worker state.
func runDispatchStatus(args []string, stdout io.Writer) error {
	var stateFile string

	fs := newFlagSet("dispatch status")
	fs.StringVar(&stateFile, "state-file", "", "override state file path (testing only)")

	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("dispatch status: %w", err)
	}

	stateFilePath, stateErr := resolveStateFile(stateFile, "dispatch status", os.UserHomeDir, os.Getwd)
	if stateErr != nil {
		return fmt.Errorf("dispatch status: %w", stateErr)
	}

	data, err := os.ReadFile(stateFilePath) //nolint:gosec
	if err != nil {
		return fmt.Errorf("dispatch status: reading state: %w", err)
	}

	state, err := agentpkg.ParseStateFile(data)
	if err != nil {
		return fmt.Errorf("dispatch status: parsing state: %w", err)
	}

	type agentStatus struct {
		Name  string `json:"name"`
		State string `json:"state"`
	}

	agents := make([]agentStatus, 0, len(state.Agents))
	for _, a := range state.Agents {
		agents = append(agents, agentStatus{Name: a.Name, State: a.State})
	}

	out, marshalErr := json.Marshal(map[string]any{
		"agents":       agents,
		"active_count": agentpkg.ActiveWorkerCount(state),
		"hold_count":   len(state.Holds),
	})
	if marshalErr != nil {
		return fmt.Errorf("dispatch status: marshaling: %w", marshalErr)
	}

	_, _ = fmt.Fprintln(stdout, string(out))

	return nil
}

// runDispatchStop implements the "engram dispatch stop" subcommand.
// Posts shutdown messages to all registered workers.
func runDispatchStop(args []string, stdout io.Writer) error {
	var (
		chatFile  string
		stateFile string
	)

	fs := newFlagSet("dispatch stop")
	fs.StringVar(&chatFile, "chat-file", "", "override chat file path (testing only)")
	fs.StringVar(&stateFile, "state-file", "", "override state file path (testing only)")

	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("dispatch stop: %w", err)
	}

	chatFilePath, chatErr := deriveChatFilePath(chatFile, os.UserHomeDir, os.Getwd)
	if chatErr != nil {
		return fmt.Errorf("dispatch stop: %w", chatErr)
	}

	stateFilePath, stateErr := resolveStateFile(stateFile, "dispatch stop", os.UserHomeDir, os.Getwd)
	if stateErr != nil {
		return fmt.Errorf("dispatch stop: %w", stateErr)
	}

	data, err := os.ReadFile(stateFilePath) //nolint:gosec
	if err != nil {
		return fmt.Errorf("dispatch stop: reading state: %w", err)
	}

	state, err := agentpkg.ParseStateFile(data)
	if err != nil {
		return fmt.Errorf("dispatch stop: parsing state: %w", err)
	}

	poster := newFilePoster(chatFilePath)

	for _, agent := range state.Agents {
		if agent.State == agentStateDead {
			continue
		}

		_, _ = poster.Post(chat.Message{
			From:   "dispatch",
			To:     agent.Name,
			Thread: "dispatch",
			Type:   "shutdown",
			Text:   "dispatch stop: shutting down",
		})

		_, _ = fmt.Fprintf(stdout, "Sent shutdown to %s\n", agent.Name)
	}

	return nil
}

// runWorkerUnderSemaphore acquires the concurrency semaphore, runs one worker's conversation loop,
// and releases the semaphore on exit. Called as a goroutine by runDispatch.
func runWorkerUnderSemaphore(
	ctx context.Context,
	cfg WorkerConfig,
	sem chan struct{},
	chatFilePath, stateFilePath, claudeBinary string,
	intentCh <-chan chat.Message,
	silentCh chan<- string,
	stdout io.Writer,
) {
	sem <- struct{}{}

	defer func() { <-sem }()

	flags := agentRunFlags{name: cfg.Name, prompt: cfg.Prompt, chatFile: chatFilePath, stateFile: stateFilePath}
	runner := buildAgentRunner(flags, stateFilePath, chatFilePath, stdout)

	_ = runConversationLoopWith(
		ctx, runner,
		cfg.Name, cfg.Prompt,
		chatFilePath, stateFilePath,
		claudeBinary, stdout,
		waitAndBuildPrompt,
		defaultWatchForIntent,
		intentCh,
		silentCh,
		defaultMemFileSelector,
	)
}

// updateLastDeliveredCursor atomically updates the LastDeliveredCursor for the named worker.
func updateLastDeliveredCursor(stateFilePath, worker string, cursor int) {
	_ = readModifyWriteStateFile(stateFilePath, func(state agentpkg.StateFile) agentpkg.StateFile {
		for i, rec := range state.Agents {
			if rec.Name == worker {
				state.Agents[i].LastDeliveredCursor = cursor

				return state
			}
		}

		return state
	})
}
