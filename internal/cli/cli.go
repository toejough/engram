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

	"engram/internal/correct"
	"engram/internal/corpus"
	"engram/internal/enrich"
	"engram/internal/memory"
	"engram/internal/render"
	"engram/internal/tomlwriter"
)

// Run dispatches to the appropriate subcommand based on args.
// Output is written to stdout. Errors are returned (caller logs to stderr, exit 0).
func Run(args []string, stdout io.Writer) error {
	if len(args) < minArgs {
		return errUsage
	}

	cmd := args[1]
	subArgs := args[minArgs:]

	switch cmd {
	case "correct":
		return runCorrect(subArgs, stdout)
	default:
		return fmt.Errorf("%w: %s", errUnknownCommand, cmd)
	}
}

const minArgs = 2

var (
	errCorrectMissingFlags = errors.New("correct: --message and --data-dir required")
	errUnknownCommand      = errors.New("unknown command")
	errUsage               = errors.New("usage: engram correct [flags]")
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
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	enricher := enrich.New(apiKey, &http.Client{})
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
