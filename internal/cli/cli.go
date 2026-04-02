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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"net/http"

	"engram/internal/anthropic"
	sessionctx "engram/internal/context"
	"engram/internal/correct"
	"engram/internal/maintain"
	"engram/internal/memory"
	"engram/internal/policy"
	"engram/internal/recall"
	"engram/internal/retrieve"
	"engram/internal/surface"
	"engram/internal/tokenresolver"
	"engram/internal/tomlwriter"
	"engram/internal/track"
)

// Exported variables.
var (
	AnthropicAPIURL = "https://api.anthropic.com/v1/messages" //nolint:gochecknoglobals // test-overridable endpoint
)

// ReadFileFunc reads a file by path, injected for testability.
type ReadFileFunc func(path string) ([]byte, error)

// Run dispatches to the appropriate subcommand based on args.
// Output is written to stdout. Errors are returned (caller logs to stderr, exit 0).
//
//nolint:cyclop // CLI dispatch switch grows with each new subcommand
func Run(
	args []string,
	stdout, stderr io.Writer,
	_ io.Reader,
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
	case "show":
		return runShow(subArgs, stdout)
	case "recall":
		return runRecall(subArgs, stdout)
	case "migrate-slugs":
		return runMigrateSlugs(subArgs, stdout)
	case "maintain":
		return runMaintain(subArgs, stdout)
	case "apply-proposal":
		return runApplyProposal(subArgs, stdout)
	case "reject-proposal":
		return runRejectProposal(subArgs, stdout)
	case "migrate-scores":
		return runMigrateScores(subArgs, stdout, stderr)
	case "migrate-sbia":
		return runMigrateSBIA(subArgs, stdout)
	case "refine":
		return runRefine(subArgs, stdout)
	case "evaluate":
		return runEvaluate(subArgs, stdout)
	default:
		return fmt.Errorf("%w: %s", errUnknownCommand, cmd)
	}
}

// unexported constants.
const (
	anthropicMaxTokens = 1024
	dirPerms           = 0o755
	filePerms          = 0o644
	formatJSON         = "json"
	minArgs            = 2
)

// unexported variables.
var (
	defaultModifier = memory.NewModifier( //nolint:gochecknoglobals // production singleton
		memory.WithModifierWriter(tomlwriter.New()),
	)
	errNoToken                 = errors.New("no API token available for memory filtering")
	errSkillExists             = errors.New("skill already exists")
	errSurfaceMissingFlags     = errors.New("surface: --mode required")
	errSurfaceStopNoTranscript = errors.New("surface: --transcript-path required for stop mode")
	errUnknownCommand          = errors.New("unknown command")
	errUsage                   = errors.New(
		"usage: engram <correct|surface|show|recall|maintain" +
			"|apply-proposal|reject-proposal|evaluate|refine|migrate-slugs> [flags]",
	)
)

// cliConfirmer prompts the user for confirmation via stdin/stdout.
type cliConfirmer struct {
	stdout      io.Writer
	stdin       io.Reader
	autoConfirm bool
}

func (c *cliConfirmer) Confirm(preview string) (bool, error) {
	_, _ = fmt.Fprintf(c.stdout, "\n--- Skill Preview ---\n%s\n--- End Preview ---\n", preview)

	if c.autoConfirm {
		_, _ = fmt.Fprintln(c.stdout, "Auto-confirmed (--yes).")

		return true, nil
	}

	_, _ = fmt.Fprint(c.stdout, "Apply this change? [y/N] ")

	var response string

	_, err := fmt.Fscan(c.stdin, &response)
	if err != nil {
		return false, fmt.Errorf("reading confirmation: %w", err)
	}

	return strings.EqualFold(response, "y") || strings.EqualFold(response, "yes"), nil
}

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

// osClaudeMDStore reads and writes CLAUDE.md files on disk.
type osClaudeMDStore struct {
	path string
}

func (s *osClaudeMDStore) Read() (string, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("reading CLAUDE.md: %w", err)
	}

	return string(data), nil
}

func (s *osClaudeMDStore) Write(content string) error {
	writeErr := os.WriteFile(s.path, []byte(content), filePerms)
	if writeErr != nil {
		return fmt.Errorf("writing CLAUDE.md: %w", writeErr)
	}

	return nil
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

// osMemoryRemover deletes a memory TOML file from disk.
type osMemoryRemover struct{}

func (r *osMemoryRemover) Remove(path string) error {
	rmErr := os.Remove(path)
	if rmErr != nil {
		return fmt.Errorf("removing memory: %w", rmErr)
	}

	return nil
}

// osSkillWriter writes skill files to a directory on disk.
type osSkillWriter struct {
	dir string
}

func (w *osSkillWriter) Write(name, content string) (string, error) {
	mkErr := os.MkdirAll(w.dir, dirPerms)
	if mkErr != nil {
		return "", fmt.Errorf("creating skills dir: %w", mkErr)
	}

	path := filepath.Join(w.dir, name+".md")

	_, err := os.Stat(path)
	if err == nil {
		return "", fmt.Errorf("%w at %s: %q", errSkillExists, path, name)
	}

	writeErr := os.WriteFile(path, []byte(content), filePerms)
	if writeErr != nil {
		return "", fmt.Errorf("writing skill: %w", writeErr)
	}

	return path, nil
}

// stdinConfirmer implements maintain.Confirmer with stdin/stdout interaction.
type stdinConfirmer struct {
	stdout io.Writer
	stdin  io.Reader
}

func (sc *stdinConfirmer) Confirm(preview string) (bool, error) {
	_, _ = fmt.Fprintf(sc.stdout, "\n%s\n\nApply? [a]pply / [s]kip / [q]uit: ", preview)

	const confirmBufSize = 16

	buf := make([]byte, confirmBufSize)

	n, err := sc.stdin.Read(buf)
	if err != nil {
		return false, fmt.Errorf("reading confirmation: %w", err)
	}

	input := string(buf[:n])
	if len(input) > 0 {
		switch input[0] {
		case 'a', 'A', 'y', 'Y':
			return true, nil
		case 'q', 'Q':
			return false, maintain.ErrUserQuit
		}
	}

	return false, nil
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
// Returns nil surfacer (not an error) when the memories directory does not exist.
func buildRecallSurfacer(ctx context.Context, dataDir string) (recall.MemorySurfacer, error) {
	retriever := retrieve.New()

	_, memErr := retriever.ListMemories(ctx, dataDir)
	if memErr != nil {
		if errors.Is(memErr, os.ErrNotExist) {
			return nil, nil //nolint:nilnil // nil surfacer is valid when no memories exist
		}

		return nil, fmt.Errorf("listing memories: %w", memErr)
	}

	surfacerOpts := []surface.SurfacerOption{
		surface.WithSurfacingRecorder(recordSurfacing),
	}

	realSurfacer := surface.New(retriever, surfacerOpts...)

	return NewRecallSurfacer(
		&surfaceRunnerAdapter{surfacer: realSurfacer},
		dataDir,
	), nil
}

// extractAssistantDelta reads new transcript lines since the last stop offset,
// strips JSONL to text, and returns only assistant content joined by newlines.
func extractAssistantDelta(dataDir, transcriptPath, sessionID string) (string, error) {
	offsetPath := filepath.Join(dataDir, "stop-surface-offset.json")

	//nolint:tagliatelle // on-disk JSON format uses snake_case
	type offsetData struct {
		Offset    int64  `json:"offset"`
		SessionID string `json:"session_id"`
	}

	var stored offsetData

	data, readErr := os.ReadFile(offsetPath) //nolint:gosec // internal path
	if readErr == nil {
		unmarshalErr := json.Unmarshal(data, &stored)
		if unmarshalErr != nil {
			fmt.Fprintf(os.Stderr, "engram: surface: parsing offset file %s: %v\n", offsetPath, unmarshalErr)
		}
	}

	offset := stored.Offset
	if sessionID != stored.SessionID {
		offset = 0
	}

	reader := &osFileReader{}
	delta := sessionctx.NewDeltaReader(reader)

	lines, newOffset, deltaErr := delta.Read(transcriptPath, offset)
	if deltaErr != nil {
		return "", fmt.Errorf("reading transcript delta: %w", deltaErr)
	}

	// Always update offset so next call resumes from here.
	newStored := offsetData{Offset: newOffset, SessionID: sessionID}

	//nolint:errchkjson // offsetData has only int64/string fields; cannot fail.
	storedBytes, _ := json.Marshal(newStored)

	writeOffsetErr := os.WriteFile(offsetPath, storedBytes, filePerms)
	if writeOffsetErr != nil {
		fmt.Fprintf(os.Stderr, "engram: surface: writing offset file %s: %v\n", offsetPath, writeOffsetErr)
	}

	if len(lines) == 0 {
		return "", nil
	}

	stripped := sessionctx.Strip(lines)

	var assistantLines []string

	for _, line := range stripped {
		if text, ok := strings.CutPrefix(line, "ASSISTANT: "); ok {
			assistantLines = append(assistantLines, text)
		}
	}

	return strings.Join(assistantLines, "\n"), nil
}

// makeAnthropicCaller returns an LLM caller function backed by the Anthropic API.
func makeAnthropicCaller(
	token string,
) func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
	client := newAnthropicClient(token)
	return client.Caller(anthropicMaxTokens)
}

// makeCLICaller returns a CallerFunc that routes through `claude -p --bare`
// instead of the direct Anthropic API. This uses Claude Code's internal routing,
// avoiding 429s that occur when the OAuth token's direct API quota is exhausted
// by the active session. The --bare flag skips hooks (preventing recursion) and
// CLAUDE.md loading (preventing context pollution).
func makeCLICaller(token string) anthropic.CallerFunc {
	return func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
		args := []string{
			"-p", "--bare", "--model", model, "--max-turns", "1",
			"--tools", "",
			"--system-prompt", systemPrompt,
		}

		//nolint:gosec // args are constructed from trusted internal values
		cmd := exec.CommandContext(ctx, "claude", args...)
		cmd.Stdin = strings.NewReader(userPrompt)

		cmd.Env = append(os.Environ(), "ANTHROPIC_API_KEY="+token)

		var stdout, stderr bytes.Buffer

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			return "", fmt.Errorf(
				"claude -p: %w\nstdout: %s\nstderr: %s",
				err, stdout.String(), stderr.String(),
			)
		}

		return strings.TrimSpace(stdout.String()), nil
	}
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
			return exec.CommandContext(ctx, name, args...).Output() //nolint:gosec // args are hardcoded in caller
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

// resolveSkillsDir returns the plugin's skills directory if available.
func resolveSkillsDir() string {
	pluginRoot := os.Getenv("CLAUDE_PLUGIN_ROOT")
	if pluginRoot == "" {
		return ""
	}

	return filepath.Join(pluginRoot, "skills")
}

// resolveToken returns the API token from the environment or macOS Keychain.
func resolveToken(ctx context.Context) string {
	token, resolveErr := newTokenResolver().Resolve(ctx)
	if resolveErr != nil {
		fmt.Fprintf(os.Stderr, "engram: resolving API token: %v\n", resolveErr)
	}

	return token
}

//nolint:funlen // CLI wiring: sequential flag parsing + dependency setup
func runCorrect(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("correct", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	message := fs.String("message", "", "user message text")
	dataDir := fs.String("data-dir", "", "path to data directory")
	transcriptPath := fs.String("transcript-path", "", "path to session transcript")
	projectSlug := fs.String("project-slug", "", "originating project slug")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("correct: %w", parseErr)
	}

	if *message == "" {
		return nil
	}

	if *dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("correct: %w", err)
		}

		defaultDir := DataDirFromHome(home)
		dataDir = &defaultDir
	}

	ctx, cancel := signalContext()
	defer cancel()

	token := resolveToken(ctx)

	policyPath := filepath.Join(*dataDir, "policy.toml")

	pol, polErr := policy.LoadFromPath(policyPath)
	if polErr != nil {
		return fmt.Errorf("correct: %w", polErr)
	}

	// Route through claude -p --bare to use Claude Code's internal routing,
	// avoiding 429s from the shared OAuth direct API quota.
	caller := makeCLICaller(token)

	reader := recall.NewTranscriptReader(&osFileReader{})
	retriever := retrieve.New()

	corrector := correct.New(
		correct.WithCaller(caller),
		correct.WithTranscriptReader(reader.Read),
		correct.WithMemoryRetriever(retriever.ListMemories),
		correct.WithWriter(tomlwriter.New()),
		correct.WithModifier(defaultModifier),
		correct.WithPolicy(pol),
	)

	result, err := corrector.Run(
		ctx,
		*message, *transcriptPath, *dataDir, *projectSlug,
	)
	if err != nil {
		return fmt.Errorf("correct: %w", err)
	}

	if result != "" {
		_, _ = fmt.Fprintln(stdout, result)
	}

	return nil
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

//nolint:funlen,cyclop // wired with flags, policy loading, transcript window, and stop-mode delegation
func runSurface(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("surface", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	mode := fs.String("mode", "", "surface mode: prompt")
	dataDir := fs.String("data-dir", "", "path to data directory")
	message := fs.String("message", "", "user message (prompt mode)")
	format := fs.String("format", "", "output format: json")
	transcriptWindow := fs.String("transcript-window", "", "recent transcript text for suppression")
	transcriptPath := fs.String("transcript-path", "", "transcript JSONL path (stop mode)")
	sessionID := fs.String("session-id", "", "session ID (stop mode)")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("surface: %w", parseErr)
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("surface: %w", defaultErr)
	}

	if *mode == "" {
		return errSurfaceMissingFlags
	}

	policyPath := filepath.Join(*dataDir, "policy.toml")

	pol, polErr := policy.LoadFromPath(policyPath)
	if polErr != nil {
		pol = policy.Defaults()
	}

	cfg := surface.ConfigFromPolicy(pol)

	// Capture original user prompt before stop-mode rewriting overwrites *message.
	userPrompt := *message

	if *mode == "stop" {
		if *transcriptPath == "" {
			return errSurfaceStopNoTranscript
		}

		assistantText, deltaErr := extractAssistantDelta(*dataDir, *transcriptPath, *sessionID)
		if deltaErr != nil {
			return fmt.Errorf("surface: %w", deltaErr)
		}

		if assistantText == "" {
			return nil
		}

		*mode = surface.ModePrompt
		*message = assistantText
		userPrompt = "" // stop mode has no original user prompt
	}

	currentProjectSlug := filepath.Base(*dataDir)

	opts := surface.Options{
		Mode:               *mode,
		DataDir:            *dataDir,
		Message:            *message,
		Format:             *format,
		TranscriptWindow:   *transcriptWindow,
		CurrentProjectSlug: currentProjectSlug,
		SessionID:          *sessionID,
		UserPrompt:         userPrompt,
	}

	retriever := retrieve.New()
	recorder := track.NewRecorder()

	surfacerOpts := []surface.SurfacerOption{
		surface.WithTracker(recorder),
		surface.WithSurfaceConfig(cfg),
		surface.WithPendingEvalModifier(func(path string, mutate func(*memory.MemoryRecord)) error {
			return defaultModifier.ReadModifyWrite(path, func(record *memory.MemoryRecord) {
				record.SurfacedCount++ // combine surfaced_count increment with pending eval write
				mutate(record)
			})
		}),
	}

	ctx, cancel := signalContext()
	defer cancel()

	token := resolveToken(ctx)

	if token != "" {
		caller := makeAnthropicCaller(token)
		surfacerOpts = append(surfacerOpts, surface.WithHaikuGate(caller))
	} else {
		return fmt.Errorf("surface: %w", errNoToken)
	}

	surfacer := surface.New(retriever, surfacerOpts...)

	runErr := surfacer.Run(ctx, stdout, opts)
	if runErr != nil {
		return runErr
	}

	return nil
}
