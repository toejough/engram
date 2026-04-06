// Package cli implements the engram command-line interface (ARCH-6).
package cli

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"engram/internal/anthropic"
	"engram/internal/chat"
	"engram/internal/memory"
	"engram/internal/recall"
	"engram/internal/surface"
	"engram/internal/tokenresolver"
	"engram/internal/tomlwriter"
	"engram/internal/watch"
)

// Exported variables.
var (
	AnthropicAPIURL = "https://api.anthropic.com/v1/messages" //nolint:gochecknoglobals // test-overridable endpoint
	HoldNowFunc     = time.Now                                //nolint:gochecknoglobals // test-overridable time source
)

// Run dispatches to the appropriate subcommand based on args.
// Output is written to stdout. Errors are returned (caller logs to stderr, exit 0).
func Run(
	args []string,
	stdout, _ io.Writer,
	_ io.Reader,
) error {
	if len(args) < minArgs {
		return errUsage
	}

	cmd := args[1]
	subArgs := args[minArgs:]

	switch cmd {
	case "recall":
		return runRecall(subArgs, stdout)
	case "show":
		return runShow(subArgs, stdout)
	case "chat":
		return runChatDispatch(subArgs, stdout)
	case "hold":
		return runHoldDispatch(subArgs, stdout)
	default:
		return fmt.Errorf("%w: %s", errUnknownCommand, cmd)
	}
}

// unexported constants.
const (
	anthropicMaxTokens   = 1024
	chatDirMode          = 0o700
	chatFileMode         = 0o600
	lockRetryDelay       = 5 * time.Millisecond
	maxLockRetries       = 200
	minArgs              = 2
	uuidV4VariantBitmask = 0x3f
	uuidV4VariantByte    = 0x80
	uuidV4VersionBitmask = 0x0f
	uuidV4VersionByte    = 0x40
)

// unexported variables.
var (
	defaultModifier = memory.NewModifier( //nolint:gochecknoglobals // production singleton
		memory.WithModifierWriter(tomlwriter.New()),
	)
	errAckWaitNilTimeout  = errors.New("outputAckResult: TIMEOUT result has nil Timeout field")
	errAckWaitNilWait     = errors.New("outputAckResult: WAIT result has nil Wait field")
	errAckWaitUnknown     = errors.New("outputAckResult: unexpected result type")
	errAgentRequired      = errors.New("chat ack-wait: --agent required")
	errLockTimeout        = errors.New("acquiring lock: exceeded max retries")
	errRecipientsRequired = errors.New("chat ack-wait: --recipients required")
	errUnknownCommand     = errors.New("unknown command")
	errUsage              = errors.New("usage: engram <recall|show|chat|hold> [flags]")
)

// haikuCallerAdapter adapts makeAnthropicCaller to the recall.HaikuCaller interface.
type haikuCallerAdapter struct {
	caller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)
}

func (a *haikuCallerAdapter) Call(
	ctx context.Context,
	systemPrompt, userPrompt string,
) (string, error) {
	return a.caller(ctx, anthropic.HaikuModel, systemPrompt, userPrompt)
}

// osDirLister lists .jsonl files in a directory using os.ReadDir.
type osDirLister struct{}

func (l *osDirLister) ListJSONL(dir string) ([]recall.FileEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("listing directory: %w", err)
	}

	results := make([]recall.FileEntry, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".jsonl") {
			continue
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			fmt.Fprintf(os.Stderr, "engram: listing directory: stat %s: %v\n", name, infoErr)

			continue
		}

		results = append(results, recall.FileEntry{
			Path:  filepath.Join(dir, name),
			Mtime: info.ModTime(),
		})
	}

	return results, nil
}

// I/O adapters for context package DI interfaces.

type osFileReader struct{}

func (r *osFileReader) Read(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:gosec,wrapcheck // thin I/O adapter
}

// watchResult is the JSON-serializable output of engram chat watch.
type watchResult struct {
	From   string    `json:"from"`
	To     string    `json:"to"`
	Thread string    `json:"thread"`
	Type   string    `json:"type"`
	TS     time.Time `json:"ts"`
	Text   string    `json:"text"`
	Cursor int       `json:"cursor"`
}

// applyDataDirDefault sets *dataDir to the standard engram data path when empty.
func applyDataDirDefault(dataDir *string) error {
	if *dataDir != "" {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolving home directory: %w", err)
	}

	*dataDir = DataDirFromHome(home)

	return nil
}

// applyProjectSlugDefault sets *slug to the PWD-derived slug when empty.
func applyProjectSlugDefault(slug *string, getwd func() (string, error)) error {
	if *slug != "" {
		return nil
	}

	cwd, err := getwd()
	if err != nil {
		return fmt.Errorf("resolving working directory: %w", err)
	}

	*slug = ProjectSlugFromPath(cwd)

	return nil
}

// buildRecallSurfacer creates a memory surfacer for the recall pipeline.
// Returns nil surfacer (not an error) when no memory directories exist.
func buildRecallSurfacer(_ context.Context, dataDir string) (recall.MemorySurfacer, error) {
	lister := memory.NewLister()

	_, memErr := lister.ListAllMemories(dataDir)
	if memErr != nil {
		if errors.Is(memErr, os.ErrNotExist) {
			return nil, nil //nolint:nilnil // nil surfacer is valid when no memories exist
		}

		return nil, fmt.Errorf("listing memories: %w", memErr)
	}

	surfacerOpts := []surface.SurfacerOption{
		surface.WithSurfacingRecorder(recordSurfacing),
	}

	realSurfacer := surface.New(lister, surfacerOpts...)

	return NewRecallSurfacer(
		&surfaceRunnerAdapter{surfacer: realSurfacer},
		dataDir,
	), nil
}

// deriveChatFilePath returns the chat file path, using override if non-empty.
// homeDir and getwd are injected for testability; callers pass os.UserHomeDir and os.Getwd.
func deriveChatFilePath(
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

	return filepath.Join(DataDirFromHome(home), "chat", ProjectSlugFromPath(cwd)+".toml"), nil
}

// filterHolds returns holds matching all non-empty filter criteria.
func filterHolds(holds []chat.HoldRecord, holder, target, tag string) []chat.HoldRecord {
	filtered := make([]chat.HoldRecord, 0, len(holds))

	for _, hold := range holds {
		if holder != "" && hold.Holder != holder {
			continue
		}

		if target != "" && hold.Target != target {
			continue
		}

		if tag != "" && hold.Tag != tag {
			continue
		}

		filtered = append(filtered, hold)
	}

	return filtered
}

// generateHoldID returns a UUID v4 string using crypto/rand.
func generateHoldID() (string, error) {
	var b [16]byte

	_, err := rand.Read(b[:])
	if err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}

	b[6] = (b[6] & uuidV4VersionBitmask) | uuidV4VersionByte
	b[8] = (b[8] & uuidV4VariantBitmask) | uuidV4VariantByte

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

// loadChatMessages reads and parses a TOML chat file using the provided readFile func.
// Returns nil slice (no error) when the file does not exist.
func loadChatMessages(chatFilePath string, readFile func(string) ([]byte, error)) ([]chat.Message, error) {
	data, err := readFile(chatFilePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("reading chat file: %w", err)
	}

	messages, parseErr := chat.ParseMessages(data)
	if parseErr != nil {
		return nil, fmt.Errorf("parsing chat file: %w", parseErr)
	}

	return messages, nil
}

// makeAnthropicCaller returns an LLM caller function backed by the Anthropic API.
func makeAnthropicCaller(
	token string,
) func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
	client := newAnthropicClient(token)
	return client.Caller(anthropicMaxTokens)
}

// marshalAndWriteWatchResult encodes result as JSON and writes it to stdout.
func marshalAndWriteWatchResult(stdout io.Writer, result watchResult) error {
	encoded, encErr := json.Marshal(result)
	if encErr != nil {
		return fmt.Errorf("chat watch: encoding result: %w", encErr)
	}

	_, err := fmt.Fprintln(stdout, string(encoded))
	if err != nil {
		return fmt.Errorf("chat watch: writing output: %w", err)
	}

	return nil
}

// marshalReleasePayload returns the JSON release payload for a hold ID.
// json.Marshal of a map[string]string is infallible; error is returned for interface conformance.
func marshalReleasePayload(holdID string) ([]byte, error) {
	return json.Marshal(map[string]string{"hold-id": holdID}) //nolint:wrapcheck
}

// newAnthropicClient creates a shared anthropic.Client configured with the
// current AnthropicAPIURL (supports test overrides).
func newAnthropicClient(token string) *anthropic.Client {
	client := anthropic.NewClient(token, &http.Client{})
	client.SetAPIURL(AnthropicAPIURL)

	return client
}

// newFilePoster creates a FilePoster wired to the OS I/O adapters.
func newFilePoster(chatFilePath string) *chat.FilePoster {
	return &chat.FilePoster{
		FilePath:   chatFilePath,
		Lock:       osLockFile,
		AppendFile: osAppendFile,
		LineCount:  osLineCount,
	}
}

// newFileWatcher creates a FileWatcher wired to the OS I/O adapters.
func newFileWatcher(chatFilePath string) *chat.FileWatcher {
	return &chat.FileWatcher{
		FilePath:  chatFilePath,
		FSWatcher: &watch.FSNotifyWatcher{},
		ReadFile:  os.ReadFile,
	}
}

// newFlagSet creates a flag set with error output discarded (flags handle their own usage).
func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	return fs
}

func newTokenResolver() *tokenresolver.Resolver {
	return tokenresolver.New(
		os.Getenv,
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // platform-internal cmd, not user input
			return cmd.Output()
		},
		runtime.GOOS,
	)
}

// osAppendFile appends data to path, creating it if needed.
func osAppendFile(path string, data []byte) error {
	mkdirErr := os.MkdirAll(filepath.Dir(path), chatDirMode)
	if mkdirErr != nil {
		return fmt.Errorf("creating directories: %w", mkdirErr)
	}

	f, openErr := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, chatFileMode) //nolint:gosec
	if openErr != nil {
		return fmt.Errorf("opening file: %w", openErr)
	}

	defer f.Close() //nolint:errcheck

	_, writeErr := f.Write(data)
	if writeErr != nil {
		return fmt.Errorf("writing file: %w", writeErr)
	}

	return nil
}

// osLineCount counts newlines in path. Returns 0 if file does not exist.
func osLineCount(path string) (int, error) {
	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}

		return 0, fmt.Errorf("reading file: %w", err)
	}

	return bytes.Count(data, []byte("\n")), nil
}

// osLockFile implements chat.LockFile using O_CREATE|O_EXCL with retry.
func osLockFile(name string) (func() error, error) {
	for range maxLockRetries {
		f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL, chatFileMode) //nolint:gosec
		if err == nil {
			return func() error {
				_ = f.Close()

				return os.Remove(name)
			}, nil
		}

		if !os.IsExist(err) {
			return nil, fmt.Errorf("creating lock: %w", err)
		}

		time.Sleep(lockRetryDelay)
	}

	return nil, errLockTimeout
}

// outputAckResult writes the flat JSON format expected by skills.
func outputAckResult(w io.Writer, result chat.AckResult) error {
	var out map[string]any

	switch result.Result {
	case "ACK":
		out = map[string]any{"result": "ACK", "cursor": result.NewCursor}
	case "WAIT":
		if result.Wait == nil {
			return errAckWaitNilWait
		}

		out = map[string]any{
			"result": "WAIT",
			"from":   result.Wait.From,
			"cursor": result.NewCursor,
			"text":   result.Wait.Text,
		}
	case "TIMEOUT":
		if result.Timeout == nil {
			return errAckWaitNilTimeout
		}

		out = map[string]any{"result": "TIMEOUT", "recipient": result.Timeout.Recipient, "cursor": result.NewCursor}
	default:
		return fmt.Errorf("%w: %q", errAckWaitUnknown, result.Result)
	}

	data, err := json.Marshal(out)
	if err != nil {
		return fmt.Errorf("outputAckResult: encoding: %w", err)
	}

	_, err = fmt.Fprintln(w, string(data))
	if err != nil {
		return fmt.Errorf("outputAckResult: writing: %w", err)
	}

	return nil
}

// recordSurfacing increments the surfaced count for a memory file.
func recordSurfacing(path string) error {
	return defaultModifier.ReadModifyWrite(path, func(record *memory.MemoryRecord) {
		record.SurfacedCount++
	})
}

// resolveChatFile derives the chat file path and wraps any error with the subcommand name.
// homeDir and getwd default to os.UserHomeDir and os.Getwd; callers may inject alternatives for testing.
func resolveChatFile(
	override, cmd string, homeDir func() (string, error), getwd func() (string, error),
) (string, error) {
	path, err := deriveChatFilePath(override, homeDir, getwd)
	if err != nil {
		return "", fmt.Errorf("%s: %w", cmd, err)
	}

	return path, nil
}

// resolveToken returns the API token from the environment or macOS Keychain.
// tokenresolver.Resolve is documented to never return a non-nil error.
func resolveToken(ctx context.Context) string {
	token, _ := newTokenResolver().Resolve(ctx)
	return token
}

func runChatAckWait(args []string, stdout io.Writer) error {
	fs := newFlagSet("chat ack-wait")

	agent := fs.String("agent", "", "calling agent name")
	cursor := fs.Int("cursor", 0, "line position to start watching from")
	recips := fs.String("recipients", "", "comma-separated recipient names")
	maxWaitS := fs.Int("max-wait", 0, "seconds to wait for online-silent recipients (default 30)")
	chatFile := fs.String("chat-file", "", "override chat file path (testing only)")

	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}

	if parseErr != nil {
		return fmt.Errorf("chat ack-wait: %w", parseErr)
	}

	if *agent == "" {
		return errAgentRequired
	}

	if *recips == "" {
		return errRecipientsRequired
	}

	chatFilePath, pathErr := resolveChatFile(*chatFile, "chat ack-wait", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	waiter := &chat.FileAckWaiter{
		FilePath: chatFilePath,
		Watcher:  newFileWatcher(chatFilePath),
		ReadFile: os.ReadFile,
		NowFunc:  time.Now,
		MaxWait:  time.Duration(*maxWaitS) * time.Second,
	}

	ctx, cancel := signalContext()
	defer cancel()

	result, ackErr := waiter.AckWait(ctx, *agent, *cursor, strings.Split(*recips, ","))
	if ackErr != nil {
		return fmt.Errorf("chat ack-wait: %w", ackErr)
	}

	return outputAckResult(stdout, result)
}

func runChatCursor(args []string, stdout io.Writer) error {
	fs := newFlagSet("chat cursor")

	chatFile := fs.String("chat-file", "", "override chat file path (testing only)")

	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}

	if parseErr != nil {
		return fmt.Errorf("chat cursor: %w", parseErr)
	}

	chatFilePath, pathErr := resolveChatFile(*chatFile, "chat cursor", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	count, countErr := osLineCount(chatFilePath)
	if countErr != nil {
		return fmt.Errorf("chat cursor: %w", countErr)
	}

	_, err := fmt.Fprintln(stdout, count)
	if err != nil {
		return fmt.Errorf("chat cursor: writing output: %w", err)
	}

	return nil
}

// runChatDispatch routes chat subcommands (post|watch|cursor).
func runChatDispatch(subArgs []string, stdout io.Writer) error {
	if len(subArgs) < 1 {
		return fmt.Errorf("%w: chat requires a subcommand (post|watch|cursor)", errUsage)
	}

	switch subArgs[0] {
	case "post":
		return runChatPost(subArgs[1:], stdout)
	case "watch":
		return runChatWatch(subArgs[1:], stdout)
	case "cursor":
		return runChatCursor(subArgs[1:], stdout)
	case "ack-wait":
		return runChatAckWait(subArgs[1:], stdout)
	default:
		return fmt.Errorf("%w: chat %s", errUnknownCommand, subArgs[0])
	}
}

func runChatPost(args []string, stdout io.Writer) error {
	fs := newFlagSet("chat post")

	from := fs.String("from", "", "sender agent name")
	toField := fs.String("to", "", "recipient names or all")
	thread := fs.String("thread", "", "conversation thread name")
	msgType := fs.String("type", "", "message type")
	text := fs.String("text", "", "message content")
	chatFile := fs.String("chat-file", "", "override chat file path (testing only)")

	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}

	if parseErr != nil {
		return fmt.Errorf("chat post: %w", parseErr)
	}

	chatFilePath, pathErr := resolveChatFile(*chatFile, "chat post", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	poster := newFilePoster(chatFilePath)

	cursor, postErr := poster.Post(chat.Message{
		From:   *from,
		To:     *toField,
		Thread: *thread,
		Type:   *msgType,
		Text:   *text,
	})
	if postErr != nil {
		return fmt.Errorf("chat post: %w", postErr)
	}

	_, err := fmt.Fprintln(stdout, cursor)
	if err != nil {
		return fmt.Errorf("chat post: writing output: %w", err)
	}

	return nil
}

func runChatWatch(args []string, stdout io.Writer) error {
	fs := newFlagSet("chat watch")

	agent := fs.String("agent", "", "agent name to filter messages for")
	cursor := fs.Int("cursor", 0, "line number to start watching from")
	typesStr := fs.String("type", "", "comma-separated message types to filter")
	timeoutSec := fs.Int("max-wait", 0, "seconds before giving up (0=block forever)")
	chatFile := fs.String("chat-file", "", "override chat file path (testing only)")

	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}

	if parseErr != nil {
		return fmt.Errorf("chat watch: %w", parseErr)
	}

	chatFilePath, pathErr := resolveChatFile(*chatFile, "chat watch", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	var msgTypes []string

	if *typesStr != "" {
		msgTypes = strings.Split(*typesStr, ",")
	}

	ctx, cancel := signalContext()
	defer cancel()

	if *timeoutSec > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*timeoutSec)*time.Second)
		defer cancel()
	}

	watcher := newFileWatcher(chatFilePath)

	msg, newCursor, watchErr := watcher.Watch(ctx, *agent, *cursor, msgTypes)
	if watchErr != nil {
		return fmt.Errorf("chat watch: %w", watchErr)
	}

	result := watchResult{
		From:   msg.From,
		To:     msg.To,
		Thread: msg.Thread,
		Type:   msg.Type,
		TS:     msg.TS,
		Text:   msg.Text,
		Cursor: newCursor,
	}

	return marshalAndWriteWatchResult(stdout, result)
}

func runHoldAcquire(args []string, stdout io.Writer) error {
	fs := newFlagSet("hold acquire")

	holder := fs.String("holder", "", "agent acquiring the hold")
	target := fs.String("target", "", "agent being held")
	condition := fs.String("condition", "", "auto-release condition")
	tag := fs.String("tag", "", "workflow label for bulk operations (e.g. codesign-1)")
	chatFile := fs.String("chat-file", "", "override chat file path (testing only)")

	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}

	if parseErr != nil {
		return fmt.Errorf("hold acquire: %w", parseErr)
	}

	chatFilePath, pathErr := resolveChatFile(*chatFile, "hold acquire", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	holdID, genErr := generateHoldID()
	if genErr != nil {
		return fmt.Errorf("generating hold id: %w", genErr)
	}

	record := chat.HoldRecord{
		HoldID:     holdID,
		Holder:     *holder,
		Target:     *target,
		Condition:  *condition,
		Tag:        *tag,
		AcquiredTS: HoldNowFunc().UTC(),
	}

	text, marshalErr := json.Marshal(record)
	if marshalErr != nil {
		return fmt.Errorf("marshaling hold record: %w", marshalErr)
	}

	_, postErr := newFilePoster(chatFilePath).Post(chat.Message{
		From:   *holder,
		To:     *target,
		Thread: "hold",
		Type:   "hold-acquire",
		Text:   string(text),
	})
	if postErr != nil {
		return fmt.Errorf("hold acquire: posting: %w", postErr)
	}

	_, err := fmt.Fprintln(stdout, holdID)
	if err != nil {
		return fmt.Errorf("hold acquire: writing output: %w", err)
	}

	return nil
}

func runHoldCheck(args []string, stdout io.Writer) error {
	fs := newFlagSet("hold check")

	chatFile := fs.String("chat-file", "", "override chat file path (testing only)")

	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}

	if parseErr != nil {
		return fmt.Errorf("hold check: %w", parseErr)
	}

	chatFilePath, pathErr := resolveChatFile(*chatFile, "hold check", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	messages, loadErr := loadChatMessages(chatFilePath, os.ReadFile)
	if loadErr != nil {
		return fmt.Errorf("hold check: %w", loadErr)
	}

	activeHolds := chat.ScanActiveHolds(messages)
	poster := newFilePoster(chatFilePath)

	for _, hold := range activeHolds {
		met, _ := chat.EvaluateCondition(hold, messages)
		if !met {
			continue
		}

		releaseText, marshalErr := marshalReleasePayload(hold.HoldID)
		if marshalErr != nil {
			return fmt.Errorf("hold check: marshaling release: %w", marshalErr)
		}

		_, postErr := poster.Post(chat.Message{
			From:   "system",
			To:     "all",
			Thread: "hold",
			Type:   "hold-release",
			Text:   string(releaseText),
		})
		if postErr != nil {
			return fmt.Errorf("hold check: posting release for %s: %w", hold.HoldID, postErr)
		}

		_, writeErr := fmt.Fprintln(stdout, hold.HoldID)
		if writeErr != nil {
			return fmt.Errorf("hold check: writing output: %w", writeErr)
		}
	}

	return nil
}

// runHoldDispatch routes hold subcommands (acquire|release|list|check).
func runHoldDispatch(subArgs []string, stdout io.Writer) error {
	if len(subArgs) < 1 {
		return fmt.Errorf("%w: hold requires a subcommand (acquire|release|list|check)", errUsage)
	}

	switch subArgs[0] {
	case "acquire":
		return runHoldAcquire(subArgs[1:], stdout)
	case "release":
		return runHoldRelease(subArgs[1:], stdout)
	case "list":
		return runHoldList(subArgs[1:], stdout)
	case "check":
		return runHoldCheck(subArgs[1:], stdout)
	default:
		return fmt.Errorf("%w: hold %s", errUnknownCommand, subArgs[0])
	}
}

func runHoldList(args []string, stdout io.Writer) error {
	fs := newFlagSet("hold list")

	holder := fs.String("holder", "", "filter by holder agent name")
	target := fs.String("target", "", "filter by target agent name")
	tag := fs.String("tag", "", "filter by workflow tag")
	chatFile := fs.String("chat-file", "", "override chat file path (testing only)")

	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}

	if parseErr != nil {
		return fmt.Errorf("hold list: %w", parseErr)
	}

	chatFilePath, pathErr := resolveChatFile(*chatFile, "hold list", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	messages, loadErr := loadChatMessages(chatFilePath, os.ReadFile)
	if loadErr != nil {
		return fmt.Errorf("hold list: %w", loadErr)
	}

	for _, hold := range filterHolds(chat.ScanActiveHolds(messages), *holder, *target, *tag) {
		_, writeErr := fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\t%s\n",
			hold.HoldID, hold.Holder, hold.Target, hold.Condition, hold.Tag,
		)
		if writeErr != nil {
			return fmt.Errorf("hold list: writing output: %w", writeErr)
		}
	}

	return nil
}

func runHoldRelease(args []string, stdout io.Writer) error {
	fs := newFlagSet("hold release")

	holdID := fs.String("hold-id", "", "hold ID returned by engram hold acquire")
	chatFile := fs.String("chat-file", "", "override chat file path (testing only)")

	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}

	if parseErr != nil {
		return fmt.Errorf("hold release: %w", parseErr)
	}

	chatFilePath, pathErr := resolveChatFile(*chatFile, "hold release", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	releasePayload, marshalErr := marshalReleasePayload(*holdID)
	if marshalErr != nil {
		return fmt.Errorf("hold release: marshaling: %w", marshalErr)
	}

	_, postErr := newFilePoster(chatFilePath).Post(chat.Message{
		From:   "system",
		To:     "all",
		Thread: "hold",
		Type:   "hold-release",
		Text:   string(releasePayload),
	})
	if postErr != nil {
		return fmt.Errorf("hold release: posting: %w", postErr)
	}

	_, err := fmt.Fprintln(stdout, "OK")
	if err != nil {
		return fmt.Errorf("hold release: writing output: %w", err)
	}

	return nil
}

func runRecall(args []string, stdout io.Writer) error {
	fs := newFlagSet("recall")

	dataDir := fs.String("data-dir", "", "path to data directory")
	projectSlug := fs.String("project-slug", "", "project directory slug")
	query := fs.String("query", "", "search query (omit for summary mode)")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("recall: %w", parseErr)
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("recall: %w", defaultErr)
	}

	slugErr := applyProjectSlugDefault(projectSlug, os.Getwd)
	if slugErr != nil {
		return fmt.Errorf("recall: %w", slugErr)
	}

	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return fmt.Errorf("recall: %w", homeErr)
	}

	projectDir := filepath.Join(home, ".claude", "projects", *projectSlug)

	ctx, cancel := signalContext()
	defer cancel()

	token := resolveToken(ctx)

	finder := recall.NewSessionFinder(&osDirLister{})
	reader := recall.NewTranscriptReader(&osFileReader{})

	var summarizer recall.SummarizerI
	if token != "" {
		summarizer = recall.NewSummarizer(&haikuCallerAdapter{
			caller: makeAnthropicCaller(token),
		})
	}

	memorySurfacer, surfErr := buildRecallSurfacer(ctx, *dataDir)
	if surfErr != nil {
		return fmt.Errorf("recall: %w", surfErr)
	}

	orch := recall.NewOrchestrator(finder, reader, summarizer, memorySurfacer)

	result, err := orch.Recall(ctx, projectDir, *query)
	if err != nil {
		return fmt.Errorf("recall: %w", err)
	}

	return recall.FormatResult(stdout, result)
}
