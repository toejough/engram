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
	"engram/internal/surface"
	"engram/internal/tokenresolver"
	"engram/internal/tomlwriter"
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
	case "serve":
		return runServe(subArgs, stdout)
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
	defaultModifier = memory.NewModifier( //nolint:gochecknoglobals // production singleton
		memory.WithModifierWriter(tomlwriter.New()),
	)
	errUnknownCommand = errors.New("unknown command")
	errUsage          = errors.New("usage: engram <recall|show|serve> [flags]")
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

// recordSurfacing increments the surfaced count for a memory file.
func recordSurfacing(path string) error {
	return defaultModifier.ReadModifyWrite(path, func(record *memory.MemoryRecord) {
		record.SurfacedCount++
	})
}

// resolveToken returns the API token from the environment or macOS Keychain.
// tokenresolver.Resolve is documented to never return a non-nil error.
func resolveToken(ctx context.Context) string {
	token, _ := newTokenResolver().Resolve(ctx)
	return token
}

func runRecall(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("recall", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

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
