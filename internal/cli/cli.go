// Package cli implements the engram command-line interface (ARCH-6).
package cli

import (
	"context"
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

// unexported constants.
const (
	anthropicMaxTokens = 1024
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

func newSummarizer(token string) recall.SummarizerI {
	if token == "" {
		return recall.NoopSummarizer{}
	}

	return recall.NewSummarizer(&haikuCallerAdapter{
		caller: makeAnthropicCaller(token),
	})
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

func runRecall(ctx context.Context, args RecallArgs, stdout io.Writer) error {
	dataDir := args.DataDir

	defaultErr := applyDataDirDefault(&dataDir)
	if defaultErr != nil {
		return fmt.Errorf("recall: %w", defaultErr)
	}

	token := resolveToken(ctx)
	summarizer := newSummarizer(token)
	memLister := memory.NewLister()

	if args.MemoriesOnly {
		limit := args.Limit
		if limit == 0 {
			limit = recall.DefaultMemoryLimit
		}

		return runRecallMemoriesOnly(ctx, stdout, summarizer, memLister, dataDir, args.Query, limit)
	}

	projectSlug := args.ProjectSlug

	return runRecallSessions(
		ctx, stdout, &projectSlug, summarizer, memLister,
		dataDir, args.Query, os.Getwd, os.UserHomeDir,
	)
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
	getwd func() (string, error),
	userHomeDir func() (string, error),
) error {
	slugErr := applyProjectSlugDefault(projectSlug, getwd)
	if slugErr != nil {
		return fmt.Errorf("recall: %w", slugErr)
	}

	home, homeErr := userHomeDir()
	if homeErr != nil {
		return fmt.Errorf("recall: %w", homeErr)
	}

	projectDir := filepath.Join(home, ".claude", "projects", *projectSlug)

	finder := recall.NewSessionFinder(&osDirLister{})
	reader := recall.NewTranscriptReader(&osFileReader{})

	externalFiles, externalCache := discoverExternalSources(ctx, home)

	orch := recall.NewOrchestrator(finder, reader, summarizer, memLister, dataDir,
		recall.WithStatusWriter(os.Stderr),
		recall.WithExternalSources(externalFiles, externalCache),
	)

	result, err := orch.Recall(ctx, projectDir, query)
	if err != nil {
		return fmt.Errorf("recall: %w", err)
	}

	return recall.FormatResult(stdout, result)
}
