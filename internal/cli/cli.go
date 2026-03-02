// Package cli implements the engram command-line interface (ARCH-2).
package cli

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"engram/internal/audit"
	"engram/internal/corpus"
	"engram/internal/correct"
	"engram/internal/reconcile"
	"engram/internal/store"
	"engram/internal/surface"
)

// Run dispatches to the appropriate subcommand based on args.
// Returns an error on failure; callers should log to stderr and exit 0 (ARCH-2 output contract).
func Run(args []string) error {
	if len(args) < minArgs {
		return errUsage
	}

	cmd := args[1]
	subArgs := args[minArgs:]

	switch cmd {
	case "extract":
		return runExtract(subArgs)
	case "correct":
		return runCorrect(subArgs)
	case "catchup":
		return runCatchup(subArgs)
	case "surface":
		return runSurface(subArgs)
	default:
		return fmt.Errorf("%w: %s", errUnknownCommand, cmd)
	}
}

// unexported constants.
const (
	defaultCandidateCount     = 3
	defaultPreToolUseBudget   = 1
	defaultSessionStartBudget = 5
	defaultUserPromptBudget   = 3
	dirPermissions            = 0o750
	filePermissions           = 0o640
	minArgs                   = 2
)

// unexported variables.
var (
	errCatchupMissingFlags = errors.New("catchup: --session and --data-dir required")
	errCatchupNoLLM        = errors.New("catchup: LLM client not yet implemented (needs Evaluator)")
	errCorrectMissingFlags = errors.New("correct: --message and --data-dir required")
	errExtractMissingFlags = errors.New("extract: --session and --data-dir required")
	errExtractNoLLM        = errors.New(
		"extract: LLM client not yet implemented (needs Enricher, Classifier, OverlapGate)",
	)
	errSurfaceMissingFlags = errors.New("surface: --hook and --data-dir required")
	errSurfaceMissingQuery = errors.New(
		"surface: one of --query, --message, --tool-input, --project-dir required",
	)
	errUnknownCommand = errors.New("unknown command")
	errUsage          = errors.New("usage: engram <extract|correct|catchup|surface> [flags]")
)

// deps bundles shared dependencies opened from the data directory.
type deps struct {
	db      *sql.DB
	store   *store.SQLiteStore
	audit   *audit.Logger
	logFile *os.File
}

func (d *deps) close() {
	if d.logFile != nil {
		_ = d.logFile.Close()
	}

	if d.db != nil {
		_ = d.db.Close()
	}
}

// formatAdapter wraps a standalone format function as a surface.Formatter.
type formatAdapter func([]store.ScoredMemory, string) string

func (f formatAdapter) FormatSurfacing(memories []store.ScoredMemory, hookType string) string {
	return f(memories, hookType)
}

// noOpGate always reports no overlap. Used until the LLM OverlapGate is implemented.
type noOpGate struct{}

func (noOpGate) Check(context.Context, reconcile.Learning, store.Memory) (bool, string, error) {
	return false, "", nil
}

// stubReconciler wraps the reconcile package for correct.DetectCorrection.
// Uses a noOpGate (always returns no overlap) since the LLM OverlapGate is not yet implemented.
// This means every correction creates a new memory rather than enriching existing ones.
type stubReconciler struct {
	store *store.SQLiteStore
}

func (r *stubReconciler) Reconcile(
	ctx context.Context,
	learning correct.Learning,
) (correct.ReconcileResult, error) {
	result, err := reconcile.Run(
		ctx,
		r.store,
		noOpGate{},
		defaultCandidateCount,
		reconcile.Learning{
			Content:  learning.Content,
			Keywords: learning.Keywords,
			Title:    learning.Title,
		},
	)

	// On error, result is zero-valued — field access is safe, err propagates.
	return correct.ReconcileResult{
		Action:   result.Action,
		MemoryID: result.MemoryID,
		Title:    result.Title,
	}, err
}

func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}

	return ""
}

func openDeps(dataDir string) (*deps, error) {
	err := os.MkdirAll(dataDir, dirPermissions)
	if err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "engram.db")

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	memoryStore, err := store.New(db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init store: %w", err)
	}

	logPath := filepath.Join(dataDir, "audit.log")

	logFile, err := os.OpenFile( //nolint:gosec // path built from CLI flag, not web input
		logPath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		filePermissions,
	)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open audit log: %w", err)
	}

	return &deps{
		db:      db,
		store:   memoryStore,
		audit:   audit.NewLogger(logFile),
		logFile: logFile,
	}, nil
}

func runCatchup(args []string) error {
	fs := flag.NewFlagSet("catchup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	session := fs.String("session", "", "path to session transcript")
	dataDir := fs.String("data-dir", "", "path to data directory")

	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("catchup: %w", err)
	}

	if *session == "" || *dataDir == "" {
		return errCatchupMissingFlags
	}

	_, err = os.ReadFile(*session)
	if err != nil {
		return fmt.Errorf("catchup: read session: %w", err)
	}

	openedDeps, err := openDeps(*dataDir)
	if err != nil {
		return err
	}

	defer openedDeps.close()

	return errCatchupNoLLM
}

func runCorrect(args []string) error {
	fs := flag.NewFlagSet("correct", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	message := fs.String("message", "", "user message text")
	dataDir := fs.String("data-dir", "", "path to data directory")

	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("correct: %w", err)
	}

	if *message == "" || *dataDir == "" {
		return errCorrectMissingFlags
	}

	openedDeps, err := openDeps(*dataDir)
	if err != nil {
		return err
	}

	defer openedDeps.close()

	ctx := context.Background()
	patterns := corpus.New(corpus.DefaultPatterns())
	recon := &stubReconciler{store: openedDeps.store}

	reminder, recordings, auditStr, err := correct.DetectCorrection(
		ctx,
		recon,
		patterns,
		nil,
		*message,
	)
	if err != nil {
		return fmt.Errorf("correct: %w", err)
	}

	if auditStr != "" {
		_ = openedDeps.audit.Log(audit.Entry{Operation: "correct", Action: auditStr})
	}

	_ = recordings // session log not yet implemented

	if reminder != "" {
		fmt.Print(reminder)
	}

	return nil
}

func runExtract(args []string) error {
	fs := flag.NewFlagSet("extract", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	session := fs.String("session", "", "path to session transcript")
	dataDir := fs.String("data-dir", "", "path to data directory")

	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	if *session == "" || *dataDir == "" {
		return errExtractMissingFlags
	}

	_, err = os.ReadFile(*session)
	if err != nil {
		return fmt.Errorf("extract: read session: %w", err)
	}

	openedDeps, err := openDeps(*dataDir)
	if err != nil {
		return err
	}

	defer openedDeps.close()

	return errExtractNoLLM
}

func runSurface(args []string) error {
	fs := flag.NewFlagSet("surface", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	hookType := fs.String("hook", "", "hook type: session-start, user-prompt, pre-tool-use")
	queryFlag := fs.String("query", "", "search query text")
	message := fs.String("message", "", "user message (alias for --query)")
	toolInput := fs.String("tool-input", "", "tool input (alias for --query)")
	projectDir := fs.String("project-dir", "", "project directory (alias for --query)")
	dataDir := fs.String("data-dir", "", "path to data directory")

	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("surface: %w", err)
	}

	if *hookType == "" || *dataDir == "" {
		return errSurfaceMissingFlags
	}

	searchQuery := coalesce(*queryFlag, *message, *toolInput, *projectDir)
	if searchQuery == "" {
		return errSurfaceMissingQuery
	}

	openedDeps, err := openDeps(*dataDir)
	if err != nil {
		return err
	}

	defer openedDeps.close()

	budget := surfaceBudget(*hookType)
	ctx := context.Background()
	formatter := formatAdapter(surface.FormatSurfacing)

	output, err := surface.Run(
		ctx,
		openedDeps.store,
		formatter,
		openedDeps.audit,
		*hookType,
		searchQuery,
		budget,
	)
	if err != nil {
		return fmt.Errorf("surface: %w", err)
	}

	if output != "" {
		fmt.Print(output)
	}

	return nil
}

// surfaceBudget returns the top-K budget for each hook type (REQ-7/8/9).
func surfaceBudget(hookType string) int {
	switch hookType {
	case "session-start":
		return defaultSessionStartBudget
	case "user-prompt":
		return defaultUserPromptBudget
	case "pre-tool-use":
		return defaultPreToolUseBudget
	default:
		return defaultUserPromptBudget
	}
}
