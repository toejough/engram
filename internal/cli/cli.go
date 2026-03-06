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
	"strings"

	"engram/internal/classify"
	"engram/internal/correct"
	"engram/internal/dedup"
	"engram/internal/extract"
	"engram/internal/learn"
	"engram/internal/render"
	"engram/internal/retrieve"
	"engram/internal/surface"
	"engram/internal/tomlwriter"
	"engram/internal/track"
	"engram/internal/transcript"
)

// RenderLearnResult writes DES-10 feedback for a learn result to w.
func RenderLearnResult(w io.Writer, result *learn.Result) {
	if len(result.CreatedPaths) == 0 {
		_, _ = fmt.Fprintln(w, "[engram] No new learnings extracted.")
		return
	}

	tierBreakdown := formatTierBreakdown(result.TierCounts)

	_, _ = fmt.Fprintf(
		w,
		"[engram] Extracted %d learnings from session. %s\n",
		len(result.CreatedPaths),
		tierBreakdown,
	)

	for _, path := range result.CreatedPaths {
		base := filepath.Base(path)
		_, _ = fmt.Fprintf(w, "  - %q (%s)\n", base, base)
	}

	if result.SkippedCount > 0 {
		_, _ = fmt.Fprintf(
			w,
			"[engram] Skipped %d duplicates.\n",
			result.SkippedCount,
		)
	}
}

// Run dispatches to the appropriate subcommand based on args.
// Output is written to stdout. Errors are returned (caller logs to stderr, exit 0).
func Run(
	args []string,
	stdout, stderr io.Writer,
	stdin io.Reader,
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
		return runSurface(subArgs, stdout)
	case "learn":
		return runLearn(subArgs, stderr, stdin)
	default:
		return fmt.Errorf("%w: %s", errUnknownCommand, cmd)
	}
}

// unexported constants.
const (
	maxTranscriptTok = 2000
	minArgs          = 2
)

// unexported variables.
var (
	errCorrectMissingFlags = errors.New(
		"correct: --message and --data-dir required",
	)
	errLearnMissingFlags   = errors.New("learn: --data-dir required")
	errSurfaceMissingFlags = errors.New(
		"surface: --mode and --data-dir required",
	)
	errUnknownCommand = errors.New("unknown command")
	errUsage          = errors.New(
		"usage: engram <correct|surface|learn> [flags]",
	)
)

// formatTierBreakdown returns a string like "(A: 2, B: 1, C: 3)" from tier counts.
func formatTierBreakdown(counts map[string]int) string {
	if len(counts) == 0 {
		return ""
	}

	parts := make([]string, 0, len(counts))

	for _, tier := range []string{"A", "B", "C"} {
		if count, ok := counts[tier]; ok && count > 0 {
			parts = append(
				parts,
				fmt.Sprintf("%s: %d", tier, count),
			)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return "(" + strings.Join(parts, ", ") + ")"
}

func runCorrect(
	args []string,
	stdout io.Writer,
) error {
	fs := flag.NewFlagSet("correct", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	message := fs.String("message", "", "user message text")
	dataDir := fs.String("data-dir", "", "path to data directory")
	transcriptPath := fs.String(
		"transcript-path", "", "path to session transcript",
	)

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("correct: %w", parseErr)
	}

	if *message == "" || *dataDir == "" {
		return errCorrectMissingFlags
	}

	// Read transcript context if available (os.ReadFile wired at the edge)
	reader := transcript.New(os.ReadFile)

	transcriptCtx, _ := reader.ReadRecent(
		*transcriptPath, maxTranscriptTok,
	)

	token := os.Getenv("ENGRAM_API_TOKEN")
	classifier := classify.New(token, &http.Client{})
	writer := tomlwriter.New()
	renderer := render.New()

	corrector := correct.New(classifier, writer, renderer, *dataDir)
	ctx := context.Background()

	output, err := corrector.Run(ctx, *message, transcriptCtx)
	if err != nil {
		return fmt.Errorf("correct: %w", err)
	}

	if output != "" {
		_, _ = fmt.Fprint(stdout, output)
	}

	return nil
}

func runLearn(
	args []string,
	stderr io.Writer,
	stdin io.Reader,
) error {
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
			"[engram] Error: session learning skipped"+
				" — no API token configured",
		)

		return nil
	}

	transcriptBytes, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("learn: reading stdin: %w", err)
	}

	transcriptText := string(transcriptBytes)

	extractor := extract.New(token, &http.Client{})
	retriever := retrieve.New()
	deduplicator := dedup.New()
	writer := tomlwriter.New()

	learner := learn.New(
		extractor, retriever, deduplicator, writer, *dataDir,
	)
	ctx := context.Background()

	result, err := learner.Run(ctx, transcriptText)
	if err != nil {
		return fmt.Errorf("learn: %w", err)
	}

	RenderLearnResult(stderr, result)

	return nil
}

func runSurface(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("surface", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	mode := fs.String(
		"mode", "", "surface mode: session-start, prompt, tool",
	)
	dataDir := fs.String("data-dir", "", "path to data directory")
	message := fs.String("message", "", "user message (prompt mode)")
	toolName := fs.String("tool-name", "", "tool name (tool mode)")
	toolInput := fs.String(
		"tool-input", "", "tool input JSON (tool mode)",
	)
	format := fs.String("format", "", "output format: json")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("surface: %w", parseErr)
	}

	if *mode == "" || *dataDir == "" {
		return errSurfaceMissingFlags
	}

	retriever := retrieve.New()
	recorder := track.NewRecorder()
	surfacer := surface.New(retriever, surface.WithTracker(recorder))
	ctx := context.Background()

	return surfacer.Run(ctx, stdout, surface.Options{
		Mode:      *mode,
		DataDir:   *dataDir,
		Message:   *message,
		ToolName:  *toolName,
		ToolInput: *toolInput,
		Format:    *format,
	})
}
