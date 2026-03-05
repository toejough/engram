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
	"path/filepath"

	"engram/internal/corpus"
	"engram/internal/correct"
	"engram/internal/dedup"
	"engram/internal/enforce"
	"engram/internal/enrich"
	"engram/internal/extract"
	"engram/internal/learn"
	"engram/internal/memory"
	"engram/internal/render"
	"engram/internal/retrieve"
	"engram/internal/surface"
	"engram/internal/tomlwriter"
)

// RenderLearnResult writes DES-10 feedback for a learn result to w.
func RenderLearnResult(w io.Writer, result *learn.Result) {
	if len(result.CreatedPaths) == 0 {
		_, _ = fmt.Fprintln(w, "[engram] No new learnings extracted.")
		return
	}

	_, _ = fmt.Fprintf(
		w,
		"[engram] Extracted %d learnings from session.\n",
		len(result.CreatedPaths),
	)

	for _, path := range result.CreatedPaths {
		base := filepath.Base(path)
		_, _ = fmt.Fprintf(w, "  - %q (%s)\n", base, base)
	}

	if result.SkippedCount > 0 {
		_, _ = fmt.Fprintf(w, "[engram] Skipped %d duplicates.\n", result.SkippedCount)
	}
}

// Run dispatches to the appropriate subcommand based on args.
// Output is written to stdout. Errors are returned (caller logs to stderr, exit 0).
func Run(
	args []string,
	stdout, stderr io.Writer,
	stdin io.Reader,
	blockStore surface.BlockHashStore,
) error {
	if len(args) < minArgs {
		return errUsage
	}

	cmd := args[1]
	subArgs := args[minArgs:]

	switch cmd {
	case "correct":
		return runCorrect(subArgs, stdout)
	case "surface":
		return runSurface(subArgs, stdout, blockStore)
	case "learn":
		return runLearn(subArgs, stderr, stdin)
	default:
		return fmt.Errorf("%w: %s", errUnknownCommand, cmd)
	}
}

// unexported constants.
const (
	minArgs = 2
)

// unexported variables.
var (
	errCorrectMissingFlags = errors.New("correct: --message and --data-dir required")
	errLearnMissingFlags   = errors.New("learn: --data-dir required")
	errSurfaceMissingFlags = errors.New("surface: --mode and --data-dir required")
	errUnknownCommand      = errors.New("unknown command")
	errUsage               = errors.New("usage: engram <correct|surface|learn> [flags]")
)

// corpusAdapter adapts *corpus.Corpus to satisfy correct.PatternMatcher.
type corpusAdapter struct {
	corpus *corpus.Corpus
}

func (a *corpusAdapter) Match(message string) *memory.PatternMatch {
	m := a.corpus.Match(message)
	if m == nil {
		return nil
	}

	return &memory.PatternMatch{
		Pattern:    m.Pattern.Regex.String(),
		Label:      m.Pattern.Label,
		Confidence: m.Confidence,
	}
}

func runCorrect(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("correct", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	message := fs.String("message", "", "user message text")
	dataDir := fs.String("data-dir", "", "path to data directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("correct: %w", parseErr)
	}

	if *message == "" || *dataDir == "" {
		return errCorrectMissingFlags
	}

	matcher := &corpusAdapter{corpus: corpus.New(corpus.DefaultPatterns())}
	token := os.Getenv("ENGRAM_API_TOKEN")
	enricher := enrich.New(token, &http.Client{})
	writer := tomlwriter.New()
	renderer := render.New()

	corrector := correct.New(matcher, enricher, writer, renderer, *dataDir)
	ctx := context.Background()

	output, err := corrector.Run(ctx, *message)
	if err != nil {
		return fmt.Errorf("correct: %w", err)
	}

	if output != "" {
		_, _ = fmt.Fprint(stdout, output)
	}

	return nil
}

func runLearn(args []string, stderr io.Writer, stdin io.Reader) error {
	fs := flag.NewFlagSet("learn", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("learn: %w", parseErr)
	}

	if *dataDir == "" {
		return errLearnMissingFlags
	}

	token := os.Getenv("ENGRAM_API_TOKEN")
	if token == "" {
		_, _ = fmt.Fprintln(
			stderr,
			"[engram] Error: session learning skipped — no API token configured",
		)

		return nil
	}

	transcriptBytes, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("learn: reading stdin: %w", err)
	}

	transcript := string(transcriptBytes)

	extractor := extract.New(token, &http.Client{})
	retriever := retrieve.New()
	deduplicator := dedup.New()
	writer := tomlwriter.New()

	learner := learn.New(extractor, retriever, deduplicator, writer, *dataDir)
	ctx := context.Background()

	result, err := learner.Run(ctx, transcript)
	if err != nil {
		return fmt.Errorf("learn: %w", err)
	}

	RenderLearnResult(stderr, result)

	return nil
}

func runSurface(args []string, stdout io.Writer, blockStore surface.BlockHashStore) error {
	fs := flag.NewFlagSet("surface", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	mode := fs.String("mode", "", "surface mode: session-start, prompt, tool")
	dataDir := fs.String("data-dir", "", "path to data directory")
	message := fs.String("message", "", "user message (prompt mode)")
	toolName := fs.String("tool-name", "", "tool name (tool mode)")
	toolInput := fs.String("tool-input", "", "tool input JSON (tool mode)")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("surface: %w", parseErr)
	}

	if *mode == "" || *dataDir == "" {
		return errSurfaceMissingFlags
	}

	token := os.Getenv("ENGRAM_API_TOKEN")
	retriever := retrieve.New()
	enforcer := enforce.New(&http.Client{})
	surfacer := surface.New(retriever, enforcer, blockStore, os.Stderr)
	ctx := context.Background()

	return surfacer.Run(ctx, stdout, surface.Options{
		Mode:      *mode,
		DataDir:   *dataDir,
		Message:   *message,
		ToolName:  *toolName,
		ToolInput: *toolInput,
		Token:     token,
	})
}
