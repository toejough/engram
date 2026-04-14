package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"engram/internal/memory"
	"engram/internal/tomlwriter"
)

// unexported constants.
const (
	v2SchemaVersion = 2
)

// unexported variables.
var (
	errAlreadyV2 = errors.New("already v2")
)

// inferSituation derives a situation string from record content when one is missing.
func inferSituation(record *memory.MemoryRecord) string {
	if record.Type == "fact" {
		if record.Content.Subject != "" {
			return "When working with " + record.Content.Subject
		}

		return "General knowledge"
	}

	if record.Content.Behavior != "" {
		return "When " + strings.ToLower(record.Content.Behavior)
	}

	return "General development"
}

// migrateDir processes all .toml files in a single directory.
// Returns counts of migrated and skipped files.
func migrateDir(dir string, stdout io.Writer) (int, int, error) {
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return 0, 0, nil
		}

		return 0, 0, fmt.Errorf("reading %s: %w", dir, readErr)
	}

	migrated := 0
	skipped := 0

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		path := filepath.Join(dir, entry.Name())

		migErr := migrateMemoryFile(path, stdout)
		if migErr != nil {
			if errors.Is(migErr, errAlreadyV2) {
				skipped++

				continue
			}

			_, _ = fmt.Fprintf(stdout, "ERROR: %s: %v\n", path, migErr)

			continue
		}

		migrated++
	}

	return migrated, skipped, nil
}

// migrateMemoryFile reads a single TOML file, checks its version, and migrates if needed.
func migrateMemoryFile(path string, _ io.Writer) error {
	cleanPath := filepath.Clean(path)

	data, readErr := os.ReadFile(cleanPath)
	if readErr != nil {
		return fmt.Errorf("reading file: %w", readErr)
	}

	// First decode to check schema_version.
	var record memory.MemoryRecord

	_, decErr := toml.Decode(string(data), &record)
	if decErr != nil {
		return fmt.Errorf("decoding TOML: %w", decErr)
	}

	if record.SchemaVersion >= v2SchemaVersion {
		return errAlreadyV2
	}

	// Normalize source.
	record.Source = normalizeSource(record.Source)

	// Infer situation if empty.
	if record.Situation == "" {
		record.Situation = inferSituation(&record)
	}

	// Bump schema version.
	record.SchemaVersion = v2SchemaVersion

	// Write back atomically — the v2 MemoryRecord struct naturally drops legacy fields.
	writer := tomlwriter.New()

	writeErr := writer.AtomicWrite(cleanPath, &record)
	if writeErr != nil {
		return fmt.Errorf("writing migrated file: %w", writeErr)
	}

	return nil
}

// normalizeSource maps freetext source values to "human" or "agent".
func normalizeSource(source string) string {
	lower := strings.ToLower(source)
	if strings.Contains(lower, "user") || strings.Contains(lower, "human") {
		return "human"
	}

	return "agent"
}

// runMigrate migrates v1 memory files to v2 format across all memory directories.
func runMigrate(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("migrate: %w", parseErr)
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("migrate: %w", defaultErr)
	}

	dirs := []string{
		memory.FeedbackDir(*dataDir),
		memory.FactsDir(*dataDir),
		memory.MemoriesDir(*dataDir),
	}

	migrated := 0
	skipped := 0

	for _, dir := range dirs {
		dirMigrated, dirSkipped, dirErr := migrateDir(dir, stdout)
		if dirErr != nil {
			return fmt.Errorf("migrate: %w", dirErr)
		}

		migrated += dirMigrated
		skipped += dirSkipped
	}

	_, _ = fmt.Fprintf(stdout, "Migrated %d memories, skipped %d (already v2)\n", migrated, skipped)

	return nil
}
