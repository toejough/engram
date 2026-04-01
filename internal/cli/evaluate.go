package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"engram/internal/anthropic"
	sessionctx "engram/internal/context"
	"engram/internal/evaluate"
	"engram/internal/memory"
	"engram/internal/policy"
	"engram/internal/tomlwriter"
)

// unexported variables.
var (
	errEvaluateMissingSessionID      = errors.New("evaluate: --session-id required")
	errEvaluateMissingTranscriptPath = errors.New("evaluate: --transcript-path required")
)

// evaluateResult is the JSON-serialisable outcome of a single memory evaluation.
//
//nolint:tagliatelle // snake_case output matches CLI convention for other commands
type evaluateResult struct {
	MemoryPath string `json:"memory_path"`
	MemoryName string `json:"memory_name"`
	Verdict    string `json:"verdict"`
	Error      string `json:"error,omitempty"`
}

// readAndStripTranscript reads a JSONL transcript file and strips it to clean text.
func readAndStripTranscript(transcriptPath string) (string, error) {
	transcriptBytes, readErr := os.ReadFile(transcriptPath) //nolint:gosec // caller-controlled path
	if readErr != nil {
		return "", fmt.Errorf("reading transcript: %w", readErr)
	}

	lines := strings.Split(strings.TrimRight(string(transcriptBytes), "\n"), "\n")

	nonEmptyLines := make([]string, 0, len(lines))

	for _, line := range lines {
		if line != "" {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}

	strippedLines := sessionctx.Strip(nonEmptyLines)

	return strings.Join(strippedLines, "\n"), nil
}

// resolveEvaluateCaller returns the injected caller or builds a real Anthropic caller.
func resolveEvaluateCaller(override CallerFunc) CallerFunc {
	if override != nil {
		return override
	}

	ctx, cancel := signalContext()
	defer cancel()

	token := resolveToken(ctx)

	return makeAnthropicCaller(token)
}

// runEvaluate is the public entry point for the evaluate command.
func runEvaluate(args []string, stdout io.Writer) error {
	return runEvaluateWith(args, stdout, nil)
}

// runEvaluateWith evaluates pending memories for a session against a transcript.
// callerOverride injects a mock LLM caller for testing; pass nil to use the real Anthropic client.
//
//nolint:funlen // CLI wiring: sequential flag parsing + dependency setup
func runEvaluateWith(args []string, stdout io.Writer, callerOverride CallerFunc) error {
	fs := flag.NewFlagSet("evaluate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	sessionID := fs.String("session-id", "", "session ID to evaluate")
	transcriptPath := fs.String("transcript-path", "", "path to transcript JSONL")
	dataDir := fs.String("data-dir", "", "path to data directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("evaluate: %w", parseErr)
	}

	if *sessionID == "" {
		return errEvaluateMissingSessionID
	}

	if *transcriptPath == "" {
		return errEvaluateMissingTranscriptPath
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("evaluate: %w", defaultErr)
	}

	policyPath := filepath.Join(*dataDir, "policy.toml")

	pol, polErr := policy.LoadFromPath(policyPath)
	if polErr != nil {
		pol = policy.Defaults()
	}

	transcript, transcriptErr := readAndStripTranscript(*transcriptPath)
	if transcriptErr != nil {
		return fmt.Errorf("evaluate: %w", transcriptErr)
	}

	if transcript == "" {
		return writeEvaluateResults(stdout, nil)
	}

	caller := resolveEvaluateCaller(callerOverride)

	// Scan for pending memories matching this session.
	scanner := evaluate.NewFileScanner(*dataDir, os.ReadFile, os.ReadDir)

	pendingMemories, scanErr := scanner(*sessionID)
	if scanErr != nil {
		return fmt.Errorf("evaluate: scanning memories: %w", scanErr)
	}

	// Build modifier and evaluator.
	modifier := memory.NewModifier(
		memory.WithModifierWriter(tomlwriter.New()),
	)

	evaluator := evaluate.New(
		caller,
		modifier.ReadModifyWrite,
		pol.EvaluateHaikuPrompt,
		anthropic.HaikuModel,
	)

	ctx, cancel := signalContext()
	defer cancel()

	results := evaluator.Run(ctx, pendingMemories, transcript)

	return writeEvaluateResults(stdout, results)
}

// writeEvaluateResults serialises results to stdout as a JSON array.
func writeEvaluateResults(stdout io.Writer, results []evaluate.Result) error {
	jsonResults := make([]evaluateResult, 0, len(results))

	for _, result := range results {
		entry := evaluateResult{
			MemoryPath: result.MemoryPath,
			MemoryName: result.MemoryName,
			Verdict:    string(result.Verdict),
		}

		if result.Err != nil {
			entry.Error = result.Err.Error()
		}

		jsonResults = append(jsonResults, entry)
	}

	encoded, encErr := json.Marshal(jsonResults)
	if encErr != nil {
		return fmt.Errorf("evaluate: encoding results: %w", encErr)
	}

	_, _ = fmt.Fprintf(stdout, "%s\n", encoded)

	return nil
}
