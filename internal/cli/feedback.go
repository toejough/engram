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
)

// unexported variables.
var (
	errFeedbackMissingDataDir   = errors.New("feedback: --data-dir required")
	errFeedbackMissingRelevance = errors.New(
		"feedback: --relevant or --irrelevant required",
	)
	errFeedbackMissingSlug = errors.New("feedback: slug argument required")
)

// applyFeedbackCounters updates the appropriate counter in record based on flags.
// Returns the label describing the feedback type.
func applyFeedbackCounters(
	record map[string]any, relevant, used, notused bool,
) string {
	if !relevant {
		current, _ := record["irrelevant_count"].(int64)
		record["irrelevant_count"] = current + 1

		return "irrelevant"
	}

	if used {
		current, _ := record["followed_count"].(int64)
		record["followed_count"] = current + 1

		return "relevant, used"
	}

	if notused {
		current, _ := record["ignored_count"].(int64)
		record["ignored_count"] = current + 1

		return "relevant, not used"
	}

	return "relevant"
}

// readFeedbackTOML reads and decodes a memory TOML into a map.
func readFeedbackTOML(
	memPath, slug string,
) (map[string]any, error) {
	data, err := os.ReadFile(memPath) //nolint:gosec // user-provided path at CLI boundary
	if err != nil {
		return nil, fmt.Errorf("feedback: reading %s: %w", slug, err)
	}

	var record map[string]any

	_, decErr := toml.Decode(string(data), &record)
	if decErr != nil {
		return nil, fmt.Errorf("feedback: decoding %s: %w", slug, decErr)
	}

	if record == nil {
		record = make(map[string]any)
	}

	return record, nil
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

	parseErr := fs.Parse(flagArgs)
	if parseErr != nil {
		return fmt.Errorf("feedback: %w", parseErr)
	}

	slug = resolveSlug(slug, *nameFlag, fs)

	if *dataDir == "" {
		return errFeedbackMissingDataDir
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

	writeErr := writeFeedbackTOML(memPath, record, slug)
	if writeErr != nil {
		return writeErr
	}

	_, _ = fmt.Fprintf(stdout,
		"[engram] Feedback recorded: %s (%s)\n", slug, label)

	return nil
}

// writeFeedbackTOML atomically writes a memory TOML map back to disk.
func writeFeedbackTOML(
	memPath string, record map[string]any, slug string,
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
