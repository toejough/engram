package cli

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"engram/internal/memory"
)

// unexported variables.
var (
	errFeedbackMissingRelevance = errors.New(
		"feedback: --relevant or --irrelevant required",
	)
	errFeedbackMissingSlug = errors.New("feedback: slug argument required")
)

// appendIrrelevantQuery appends the surfacing query to the memory's
// IrrelevantQueries list (capped at 20, dropping oldest on overflow).
func appendIrrelevantQuery(
	record *memory.MemoryRecord, irrelevant bool, query string,
) {
	const maxIrrelevantQueries = 20

	if !irrelevant || query == "" {
		return
	}

	record.IrrelevantQueries = append(record.IrrelevantQueries, query)
	if len(record.IrrelevantQueries) > maxIrrelevantQueries {
		record.IrrelevantQueries = record.IrrelevantQueries[len(record.IrrelevantQueries)-maxIrrelevantQueries:]
	}
}

// applyFeedbackCounters updates the appropriate counter in record based on flags.
// Returns the label describing the feedback type.
func applyFeedbackCounters(
	record *memory.MemoryRecord, relevant, used, notused bool,
) string {
	if !relevant {
		record.IrrelevantCount++

		return "irrelevant"
	}

	if used {
		record.FollowedCount++

		return "relevant, used"
	}

	if notused {
		record.IgnoredCount++

		return "relevant, not used"
	}

	return "relevant"
}

// readFeedbackTOML reads and decodes a memory TOML into a MemoryRecord.
func readFeedbackTOML(
	memPath, slug string,
) (*memory.MemoryRecord, error) {
	data, err := os.ReadFile(memPath) //nolint:gosec // user-provided path at CLI boundary
	if err != nil {
		return nil, fmt.Errorf("feedback: reading %s: %w", slug, err)
	}

	var record memory.MemoryRecord

	_, decErr := toml.Decode(string(data), &record)
	if decErr != nil {
		return nil, fmt.Errorf("feedback: decoding %s: %w", slug, decErr)
	}

	return &record, nil
}

// runFeedback implements the feedback subcommand: records relevance/usage
// feedback for a memory by updating its TOML counters.
func runFeedback(args []string, stdout io.Writer) error {
	slug, flagArgs := extractSlug(args)

	fs := flag.NewFlagSet("feedback", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	nameFlag := fs.String("name", "", "memory slug (alternative to positional arg)")
	relevant := fs.Bool("relevant", false, "memory was relevant")
	irrelevant := fs.Bool("irrelevant", false, "memory was not relevant")
	used := fs.Bool("used", false, "memory was used (with --relevant)")
	notused := fs.Bool("notused", false, "memory was not used (with --relevant)")
	surfacingQuery := fs.String("surfacing-query", "", "query that caused memory to surface")

	parseErr := fs.Parse(flagArgs)
	if parseErr != nil {
		return fmt.Errorf("feedback: %w", parseErr)
	}

	slug = resolveSlug(slug, *nameFlag, fs)

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("feedback: %w", defaultErr)
	}

	if slug == "" {
		return errFeedbackMissingSlug
	}

	if !*relevant && !*irrelevant {
		return errFeedbackMissingRelevance
	}

	memPath := filepath.Join(*dataDir, "memories", slug+".toml")

	record, err := readFeedbackTOML(memPath, slug)
	if err != nil {
		return err
	}

	label := applyFeedbackCounters(record, *relevant, *used, *notused)
	appendIrrelevantQuery(record, *irrelevant, *surfacingQuery)

	writeErr := writeFeedbackTOML(memPath, record, slug)
	if writeErr != nil {
		return writeErr
	}

	_, _ = fmt.Fprintf(stdout,
		"[engram] Feedback recorded: %s (%s)\n", slug, label)

	if *irrelevant && *surfacingQuery != "" {
		_, _ = fmt.Fprintf(stdout,
			"[engram] Surfacing context recorded for refinement\n")
	}

	return nil
}

// writeFeedbackTOML atomically writes a MemoryRecord back to disk.
func writeFeedbackTOML(
	memPath string, record *memory.MemoryRecord, slug string,
) error {
	var buf bytes.Buffer

	encErr := toml.NewEncoder(&buf).Encode(record)
	if encErr != nil {
		return fmt.Errorf("feedback: encoding %s: %w", slug, encErr)
	}

	dir := filepath.Dir(memPath)
	tmpPath := filepath.Join(dir, ".tmp-feedback")

	const filePerm = 0o644

	writeErr := os.WriteFile(tmpPath, buf.Bytes(), filePerm)
	if writeErr != nil {
		return fmt.Errorf("feedback: writing temp: %w", writeErr)
	}

	renameErr := os.Rename(tmpPath, memPath)
	if renameErr != nil {
		return fmt.Errorf("feedback: renaming temp: %w", renameErr)
	}

	return nil
}
