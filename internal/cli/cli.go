// Package cli implements the engram command-line interface (ARCH-6).
package cli

import (
	"bytes"
	"context"
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
	case "agent":
		return runAgentDispatch(subArgs, stdout, osTmuxSpawn)
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
	errAckWaitNilTimeout     = errors.New("outputAckResult: TIMEOUT result has nil Timeout field")
	errAckWaitNilWait        = errors.New("outputAckResult: WAIT result has nil Wait field")
	errAckWaitUnknown        = errors.New("outputAckResult: unexpected result type")
	errAgentRequired         = errors.New("chat ack-wait: --agent required")
	errHoldIDRequired        = errors.New("hold release: --hold-id is required")
	errHolderRequired        = errors.New("hold acquire: --holder is required")
	errLockTimeout           = errors.New("acquiring lock: exceeded max retries")
	errRecipientsRequired    = errors.New("chat ack-wait: --recipients required")
	errTargetRequired        = errors.New("hold acquire: --target is required")
	errUnknownCommand        = errors.New("unknown command")
	errUsage                 = errors.New("usage: engram <recall|show|chat|hold> [flags]")
	errWaitReadyNameRequired = errors.New("agent wait-ready: --name is required")
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

// applyDataDirDefault sets *dataDir to the standard engram data path when empty.
func applyDataDirDefault(dataDir *string) error {
	if *dataDir != "" {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolving home directory: %w", err)
	}

	*dataDir = DataDirFromHome(home, os.Getenv)

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

	return filepath.Join(DataDirFromHome(home, os.Getenv), "chat", ProjectSlugFromPath(cwd)+".toml"), nil
}

// loadChatMessages reads and parses a TOML chat file using the provided readFile func.
// Uses ParseMessagesSafe to tolerate per-message corruption (same attack surface as #515).
// Returns nil slice (no error) when the file does not exist.
func loadChatMessages(chatFilePath string, readFile func(string) ([]byte, error)) ([]chat.Message, error) {
	data, err := readFile(chatFilePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("reading chat file: %w", err)
	}

	return chat.ParseMessagesSafe(data), nil
}

// makeAnthropicCaller returns an LLM caller function backed by the Anthropic API.
func makeAnthropicCaller(
	token string,
) func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
	client := newAnthropicClient(token)
	return client.Caller(anthropicMaxTokens)
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
