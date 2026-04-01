package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"engram/internal/correct"
	"engram/internal/memory"
	"engram/internal/policy"
	"engram/internal/recall"
	"engram/internal/tomlwriter"
)

// CallerFunc is a function that calls an LLM model.
type CallerFunc = func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)

// unexported constants.
const (
	maxTranscriptMatchWindow = 24 * time.Hour
)

// findAllTranscripts walks ~/.claude/projects/*/*.jsonl and returns all paths found.
func findAllTranscripts(projectsDir string) ([]string, error) {
	projectDirs, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("reading projects dir: %w", err)
	}

	var transcripts []string

	for _, projectDir := range projectDirs {
		if !projectDir.IsDir() {
			continue
		}

		dir := filepath.Join(projectsDir, projectDir.Name())

		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			if filepath.Ext(entry.Name()) != ".jsonl" {
				continue
			}

			transcripts = append(transcripts, filepath.Join(dir, entry.Name()))
		}
	}

	return transcripts, nil
}

// findTranscriptForMemory returns the path of the transcript whose mtime is
// closest in time to the memory's created_at, within maxTranscriptMatchWindow.
// Returns "" if no transcript falls within the window.
func findTranscriptForMemory(record memory.MemoryRecord, transcripts []string) string {
	createdAt, parseErr := time.Parse(time.RFC3339, record.CreatedAt)
	if parseErr != nil {
		return ""
	}

	bestPath := ""

	var bestDiff time.Duration

	found := false

	for _, path := range transcripts {
		info, statErr := os.Stat(path)
		if statErr != nil {
			continue
		}

		diff := info.ModTime().Sub(createdAt)
		if diff < 0 {
			diff = -diff
		}

		if diff > maxTranscriptMatchWindow {
			continue
		}

		if !found || diff < bestDiff {
			bestDiff = diff
			bestPath = path
			found = true
		}
	}

	return bestPath
}

func runRefine(args []string, stdout io.Writer) error {
	return runRefineWith(args, stdout, nil)
}

//nolint:cyclop,funlen,gocognit // CLI command with sequential setup steps
func runRefineWith(args []string, stdout io.Writer, callerOverride CallerFunc) error {
	fs := flag.NewFlagSet("refine", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	apiToken := fs.String("api-token", "", "Anthropic API token")
	dryRun := fs.Bool("dry-run", false, "show what would be refined without changing files")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("refine: %w", parseErr)
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("refine: %w", defaultErr)
	}

	ctx, cancel := signalContext()
	defer cancel()

	// Resolve API token from env if not provided via flag.
	if *apiToken == "" {
		*apiToken = resolveToken(ctx)
	}

	memoriesDir := memory.MemoriesDir(*dataDir)

	records, listErr := memory.ListAll(memoriesDir)
	if listErr != nil {
		if errors.Is(listErr, os.ErrNotExist) {
			_, _ = fmt.Fprintln(stdout, "[engram] refine: no memories found")

			return nil
		}

		return fmt.Errorf("refine: listing memories: %w", listErr)
	}

	if len(records) == 0 {
		_, _ = fmt.Fprintln(stdout, "[engram] refine: 0 refined, 0 skipped")

		return nil
	}

	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return fmt.Errorf("refine: resolving home: %w", homeErr)
	}

	projectsDir := filepath.Join(home, ".claude", "projects")

	transcripts, transcriptErr := findAllTranscripts(projectsDir)
	if transcriptErr != nil {
		return fmt.Errorf("refine: finding transcripts: %w", transcriptErr)
	}

	policyPath := filepath.Join(*dataDir, "policy.toml")

	pol, polErr := policy.LoadFromPath(policyPath)
	if polErr != nil {
		return fmt.Errorf("refine: loading policy: %w", polErr)
	}

	reader := recall.NewTranscriptReader(&osFileReader{})

	var caller CallerFunc
	if callerOverride != nil {
		caller = callerOverride
	} else {
		caller = makeCLICaller(*apiToken)
	}

	modifier := memory.NewModifier(
		memory.WithModifierWriter(tomlwriter.New()),
	)

	refinedCount := 0
	skippedCount := 0

	total := len(records)

	for idx, stored := range records {
		if ctx.Err() != nil {
			_, _ = fmt.Fprintf(stdout, "[engram] refine: interrupted\n")

			break
		}

		name := memory.NameFromPath(stored.Path)

		transcriptPath := findTranscriptForMemory(stored.Record, transcripts)
		if transcriptPath == "" {
			_, _ = fmt.Fprintf(stdout, "[%d/%d] skip %s: no transcript\n", idx+1, total, name)
			skippedCount++

			continue
		}

		transcriptContext, _, readErr := reader.Read(transcriptPath, pol.ContextByteBudget)
		if readErr != nil {
			_, _ = fmt.Fprintf(stdout, "[%d/%d] skip %s: read error: %v\n", idx+1, total, name, readErr)
			skippedCount++

			continue
		}

		// Skip already-refined memories (no Keywords: blob in situation).
		if !strings.Contains(stored.Record.Situation, "Keywords:") {
			_, _ = fmt.Fprintf(stdout, "[%d/%d] skip %s: already refined\n", idx+1, total, name)
			skippedCount++

			continue
		}

		if *dryRun {
			_, _ = fmt.Fprintf(stdout, "[%d/%d] would refine %s\n", idx+1, total, name)
			refinedCount++

			continue
		}

		extraction, extractErr := correct.Refine(
			ctx,
			caller,
			&stored.Record,
			transcriptContext,
			pol.RefineSonnetPrompt,
		)
		if extractErr != nil {
			if errors.Is(extractErr, correct.ErrEmptyResponse) {
				_, _ = fmt.Fprintf(stdout, "[%d/%d] skip %s: no extraction from transcript\n",
					idx+1, total, name)
			} else {
				_, _ = fmt.Fprintf(stdout, "[%d/%d] FAIL %s: %v\n", idx+1, total, name, extractErr)
			}

			skippedCount++

			continue
		}

		modifyErr := modifier.ReadModifyWrite(stored.Path, func(record *memory.MemoryRecord) {
			if extraction.Situation != "" {
				record.Situation = extraction.Situation
			}

			if extraction.Behavior != "" {
				record.Behavior = extraction.Behavior
			}

			if extraction.Impact != "" {
				record.Impact = extraction.Impact
			}

			if extraction.Action != "" {
				record.Action = extraction.Action
			}
		})
		if modifyErr != nil {
			_, _ = fmt.Fprintf(stdout, "[%d/%d] FAIL %s: write error: %v\n", idx+1, total, name, modifyErr)
			skippedCount++

			continue
		}

		_, _ = fmt.Fprintf(stdout, "[%d/%d] refined %s\n", idx+1, total, name)
		refinedCount++
	}

	_, _ = fmt.Fprintf(stdout, "[engram] refine: %d refined, %d skipped (of %d)\n",
		refinedCount, skippedCount, total)

	return nil
}
