// Package cli implements the engram command-line interface (ARCH-6).
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"syscall"

	"engram/internal/llmcmd"
	"engram/internal/memory"
	"engram/internal/recall"
)

// unexported constants.
const (
	envLLMCmd          = "ENGRAM_LLM_CMD"
	luhmannLockFile    = ".luhmann.lock"
	recallOptsCapacity = 2 // WithStatusWriter + WithExternalSources
)

// unexported variables.
var (
	errLLMCmdRequired = errors.New(
		"llm-cmd is required: set --llm-cmd flag or ENGRAM_LLM_CMD environment variable",
	)
	errNotADirectory = errors.New("not a directory")
)

// osDirLister lists .jsonl files in a directory using os.ReadDir.
type osDirLister struct{}

func (l *osDirLister) ListJSONL(dir string) ([]recall.FileEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

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

// osPromoteFS is the production filesystem adapter for the promote subcommand.
type osPromoteFS struct{}

// DeleteFleeting removes the fleeting file at path.
func (*osPromoteFS) DeleteFleeting(path string) error {
	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("remove: %w", err)
	}

	return nil
}

// ListIDs returns Luhmann IDs from filenames in vault/Permanent and vault/MOCs.
func (*osPromoteFS) ListIDs(vault string) ([]string, error) {
	out := []string{}

	for _, sub := range []string{"Permanent", "MOCs"} {
		entries, err := os.ReadDir(filepath.Join(vault, sub))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, fmt.Errorf("read %s: %w", sub, err)
		}

		for _, e := range entries {
			if e.IsDir() {
				continue
			}

			id, ok := extractLuhmannFromFilename(e.Name())
			if !ok {
				continue
			}

			out = append(out, id)
		}
	}

	return out, nil
}

// Lock acquires an exclusive flock on vault/.luhmann.lock; returns a release func.
func (*osPromoteFS) Lock(vault string) (func(), error) {
	path := filepath.Join(vault, luhmannLockFile)

	const perm = 0o600

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, perm) //nolint:gosec // path from caller
	if err != nil {
		return nil, fmt.Errorf("open lock: %w", err)
	}

	fileDescriptor := int(f.Fd()) //nolint:gosec // fd fits in int on supported platforms

	flockErr := syscall.Flock(fileDescriptor, syscall.LOCK_EX)
	if flockErr != nil {
		_ = f.Close()

		return nil, fmt.Errorf("flock: %w", flockErr)
	}

	release := func() {
		_ = syscall.Flock(fileDescriptor, syscall.LOCK_UN)
		_ = f.Close()
	}

	return release, nil
}

// StatDir returns an error if the directory does not exist or isn't accessible.
func (*osPromoteFS) StatDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%w: %s", errNotADirectory, path)
	}

	return nil
}

// WriteNew creates the file with O_EXCL — errors with fs.ErrExist if it already exists.
func (*osPromoteFS) WriteNew(path string, data []byte) error {
	const perm = 0o600

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm) //nolint:gosec // path from caller
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}

	defer func() { _ = f.Close() }()

	_, writeErr := f.Write(data)
	if writeErr != nil {
		return fmt.Errorf("write: %w", writeErr)
	}

	return nil
}

// osQuickFS is the production filesystem adapter for the quick subcommand.
type osQuickFS struct{}

// StatDir returns an error if the directory does not exist or isn't accessible.
func (*osQuickFS) StatDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%w: %s", errNotADirectory, path)
	}

	return nil
}

// WriteNew creates the file with O_EXCL — errors with fs.ErrExist if it already exists.
func (*osQuickFS) WriteNew(path string, data []byte) error {
	const perm = 0o600

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm) //nolint:gosec // path from caller
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}

	defer func() { _ = f.Close() }()

	_, writeErr := f.Write(data)
	if writeErr != nil {
		return fmt.Errorf("write: %w", writeErr)
	}

	return nil
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

func requireLLMCmd(flagValue string) error {
	if resolveLLMCmd(flagValue) == "" {
		return errLLMCmdRequired
	}

	return nil
}

// resolveLLMCmd returns the explicit flag value if set, otherwise the
// ENGRAM_LLM_CMD env var, otherwise the empty string.
func resolveLLMCmd(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}

	return os.Getenv(envLLMCmd)
}

func runRecall(ctx context.Context, args RecallArgs, stdout io.Writer) error {
	dataDir := args.DataDir

	defaultErr := applyDataDirDefault(&dataDir)
	if defaultErr != nil {
		return fmt.Errorf("recall: %w", defaultErr)
	}

	runner := llmcmd.New(resolveLLMCmd(args.LLMCmd))
	summarizer := llmcmd.NewExtractor(runner)
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
		args.TranscriptDir,
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
	transcriptDir string,
) error {
	slugErr := applyProjectSlugDefault(projectSlug, getwd)
	if slugErr != nil {
		return fmt.Errorf("recall: %w", slugErr)
	}

	home, homeErr := userHomeDir()
	if homeErr != nil {
		return fmt.Errorf("recall: %w", homeErr)
	}

	cwd, cwdErr := getwd()
	if cwdErr != nil {
		return fmt.Errorf("recall: %w", cwdErr)
	}

	var dirs []string
	if transcriptDir != "" {
		dirs = []string{transcriptDir}
	} else {
		dirs = []string{
			filepath.Join(home, ".claude", "projects", *projectSlug),
		}
	}

	finder := recall.NewCompositeSessionFinder(
		recall.NewSessionFinder(&osDirLister{}),
		recall.NewOpencodeSessionFinder(recall.DefaultOpencodeDBPath(), cwd),
	)
	reader := recall.NewCompositeTranscriptReader(
		recall.NewTranscriptReader(&osFileReader{}),
		recall.NewOpencodeTranscriptReader(recall.DefaultOpencodeDBPath()),
	)

	opts := make([]recall.OrchestratorOption, 0, recallOptsCapacity)
	opts = append(opts, recall.WithStatusWriter(os.Stderr))

	externalFiles, externalCache := discoverExternalSources(ctx, home)
	opts = append(opts, recall.WithExternalSources(externalFiles, externalCache))

	orch := recall.NewOrchestrator(finder, reader, summarizer, memLister, dataDir, opts...)

	result, err := orch.Recall(ctx, query, dirs...)
	if err != nil {
		return fmt.Errorf("recall: %w", err)
	}

	return recall.FormatResult(stdout, result)
}
