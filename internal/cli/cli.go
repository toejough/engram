// Package cli implements the engram command-line interface (ARCH-6).
package cli

import (
	"context"
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

	"engram/internal/anthropic"
	"engram/internal/memory"
	"engram/internal/recall"
	"engram/internal/tokenresolver"
)

// Exported variables.
var (
	AnthropicAPIURL = "https://api.anthropic.com/v1/messages" //nolint:gochecknoglobals // test-overridable endpoint
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
	case "list":
		return runList(subArgs, stdout)
	case "learn":
		return runLearn(subArgs, stdout)
	case "update":
		return runUpdate(subArgs, stdout)
	default:
		return fmt.Errorf("%w: %s", errUnknownCommand, cmd)
	}
}

// unexported constants.
const (
	anthropicMaxTokens = 1024
	minArgs            = 2
)

// unexported variables.
var (
	errUnknownCommand = errors.New("unknown command")
	errUsage          = errors.New("usage: engram <recall|show|list|learn|update> [flags]")
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

// newFlagSet creates a flag set with error output discarded (flags handle their own usage).
func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	return fs
}

func newSummarizer(token string) recall.SummarizerI {
	if token != "" {
		return recall.NewSummarizer(&haikuCallerAdapter{
			caller: makeAnthropicCaller(token),
		})
	}

	return nil
}

func newTokenResolver() *tokenresolver.Resolver {
	return tokenresolver.New(
		os.Getenv,
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			cmd := exec.CommandContext( //nolint:gosec // platform-internal cmd, not user input
				ctx,
				name,
				args...)

			return cmd.Output()
		},
		runtime.GOOS,
	)
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
	memoriesOnly := fs.Bool("memories-only", false, "search only memory files")
	limit := fs.Int("limit", recall.DefaultMemoryLimit, "max memories to return")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("recall: %w", parseErr)
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("recall: %w", defaultErr)
	}

	ctx, cancel := signalContext()
	defer cancel()

	token := resolveToken(ctx)
	summarizer := newSummarizer(token)
	memLister := memory.NewLister()

	if *memoriesOnly {
		return runRecallMemoriesOnly(ctx, stdout, summarizer, memLister, *dataDir, *query, *limit)
	}

	return runRecallSessions(ctx, stdout, projectSlug, summarizer, memLister, *dataDir, *query)
}

func runRecallMemoriesOnly(
	ctx context.Context,
	stdout io.Writer,
	summarizer recall.SummarizerI,
	memLister recall.MemoryLister,
	dataDir, query string,
	limit int,
) error {
	orch := recall.NewOrchestrator(nil, nil, summarizer, memLister, dataDir)

	result, err := orch.RecallMemoriesOnly(ctx, query, limit)
	if err != nil {
		return fmt.Errorf("recall: %w", err)
	}

	return recall.FormatResult(stdout, result)
}

func runRecallSessions(
	ctx context.Context,
	stdout io.Writer,
	projectSlug *string,
	summarizer recall.SummarizerI,
	memLister recall.MemoryLister,
	dataDir, query string,
) error {
	slugErr := applyProjectSlugDefault(projectSlug, os.Getwd)
	if slugErr != nil {
		return fmt.Errorf("recall: %w", slugErr)
	}

	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return fmt.Errorf("recall: %w", homeErr)
	}

	projectDir := filepath.Join(home, ".claude", "projects", *projectSlug)

	finder := recall.NewSessionFinder(&osDirLister{})
	reader := recall.NewTranscriptReader(&osFileReader{})

	orch := recall.NewOrchestrator(finder, reader, summarizer, memLister, dataDir)

	result, err := orch.Recall(ctx, projectDir, query)
	if err != nil {
		return fmt.Errorf("recall: %w", err)
	}

	return recall.FormatResult(stdout, result)
}
