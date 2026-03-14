// Package migrate implements the JSONL→TOML migration for engram (UC-23, ARCH-58).
package migrate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"engram/internal/registry"
)

// Migrator reads instruction-registry.jsonl and merges metrics into TOML memory files.
type Migrator struct {
	readFile   func(name string) ([]byte, error)
	writeFile  func(name string, data []byte, perm os.FileMode) error
	renameFile func(oldpath, newpath string) error
	remove     func(name string) error
	stat       func(name string) (os.FileInfo, error)
	stdout     io.Writer
}

// New creates a Migrator with real filesystem operations by default.
func New(opts ...MigratorOption) *Migrator {
	m := &Migrator{
		readFile:   os.ReadFile,
		writeFile:  os.WriteFile,
		renameFile: os.Rename,
		remove:     os.Remove,
		stat:   os.Stat,
		stdout: os.Stdout,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Run executes the migration from jsonlPath into memoriesDir.
// If dryRun is true, no writes or deletes are performed.
func (m *Migrator) Run(jsonlPath, memoriesDir string, dryRun bool) error {
	_, statErr := m.stat(jsonlPath)
	if statErr != nil {
		_, _ = fmt.Fprintln(m.stdout, "[engram] nothing to migrate: instruction-registry.jsonl not found")

		return nil //nolint:nilerr // intentional: file not found is success
	}

	data, readErr := m.readFile(jsonlPath)
	if readErr != nil {
		return fmt.Errorf("reading %s: %w", jsonlPath, readErr)
	}

	entries := parseJSONL(data)

	migrated, skipped, unmatched, processErr := m.processEntries(entries, memoriesDir, dryRun)
	if processErr != nil {
		return processErr
	}

	if dryRun {
		_, _ = fmt.Fprintf(m.stdout,
			"[engram] dry-run: %d would be migrated, %d skipped (non-memory), %d unmatched\n",
			migrated, skipped, unmatched)

		return nil
	}

	_, _ = fmt.Fprintf(m.stdout,
		"[engram] Migrated %d entries. Skipped %d non-memory. Unmatched %d.\n",
		migrated, skipped, unmatched)

	removeErr := m.remove(jsonlPath)
	if removeErr != nil {
		return fmt.Errorf("removing %s: %w", jsonlPath, removeErr)
	}

	return nil
}

func (m *Migrator) processEntries(
	entries []registry.InstructionEntry,
	memoriesDir string,
	dryRun bool,
) (migrated, skipped, unmatched int, err error) {
	for idx := range entries {
		entry := &entries[idx]

		if entry.SourceType != registry.SourceTypeMemory {
			skipped++

			continue
		}

		tomlPath := filepath.Join(memoriesDir, filepath.Base(entry.SourcePath))

		_, statErr := m.stat(tomlPath)
		if statErr != nil {
			unmatched++
			_, _ = fmt.Fprintf(m.stdout, "[engram] unmatched: %s (TOML not found)\n", entry.SourcePath)

			continue
		}

		if dryRun {
			_, _ = fmt.Fprintf(m.stdout, "[engram] dry-run: would migrate %s\n", tomlPath)
			migrated++

			continue
		}

		mergeErr := m.mergeIntoTOML(tomlPath, entry)
		if mergeErr != nil {
			return migrated, skipped, unmatched, fmt.Errorf("merging %s: %w", entry.SourcePath, mergeErr)
		}

		migrated++
	}

	return migrated, skipped, unmatched, nil
}

func (m *Migrator) mergeIntoTOML(path string, entry *registry.InstructionEntry) error {
	data, readErr := m.readFile(path)
	if readErr != nil {
		return fmt.Errorf("reading TOML: %w", readErr)
	}

	var existing map[string]any

	_, decodeErr := toml.Decode(string(data), &existing)
	if decodeErr != nil {
		return fmt.Errorf("decoding TOML: %w", decodeErr)
	}

	maps.Copy(existing, buildUpdates(entry))

	var buf bytes.Buffer

	encodeErr := toml.NewEncoder(&buf).Encode(existing)
	if encodeErr != nil {
		return fmt.Errorf("encoding TOML: %w", encodeErr)
	}

	dir := filepath.Dir(path)
	tmpPath := filepath.Join(dir, ".tmp-migrate-"+filepath.Base(path))

	writeErr := m.writeFile(tmpPath, buf.Bytes(), tomlFilePerm)
	if writeErr != nil {
		return fmt.Errorf("writing temp file: %w", writeErr)
	}

	renameErr := m.renameFile(tmpPath, path)
	if renameErr != nil {
		return fmt.Errorf("renaming temp to final: %w", renameErr)
	}

	return nil
}

// MigratorOption configures a Migrator.
type MigratorOption func(*Migrator)

// WithReadFile injects a file reader.
func WithReadFile(fn func(name string) ([]byte, error)) MigratorOption {
	return func(m *Migrator) { m.readFile = fn }
}

// WithWriteFile injects a file writer.
func WithWriteFile(fn func(name string, data []byte, perm os.FileMode) error) MigratorOption {
	return func(m *Migrator) { m.writeFile = fn }
}

// WithRenameFile injects a rename function (used for atomic writes).
func WithRenameFile(fn func(oldpath, newpath string) error) MigratorOption {
	return func(m *Migrator) { m.renameFile = fn }
}

// WithRemove injects a file removal function.
func WithRemove(fn func(name string) error) MigratorOption {
	return func(m *Migrator) { m.remove = fn }
}

// WithStat injects a stat function.
func WithStat(fn func(name string) (os.FileInfo, error)) MigratorOption {
	return func(m *Migrator) { m.stat = fn }
}

// WithStdout injects a writer for progress output.
func WithStdout(w io.Writer) MigratorOption {
	return func(m *Migrator) { m.stdout = w }
}

// unexported constants.
const tomlFilePerm = 0o644

func buildUpdates(entry *registry.InstructionEntry) map[string]any {
	updates := map[string]any{
		"surfaced_count":     entry.SurfacedCount,
		"followed_count":     entry.Evaluations.Followed,
		"contradicted_count": entry.Evaluations.Contradicted,
		"ignored_count":      entry.Evaluations.Ignored,
		"enforcement_level":  string(entry.EnforcementLevel),
		"content_hash":       entry.ContentHash,
	}

	if entry.LastSurfaced != nil {
		updates["last_surfaced_at"] = entry.LastSurfaced.Format(time.RFC3339)
	}

	if len(entry.Links) > 0 {
		links := make([]map[string]any, 0, len(entry.Links))

		for _, link := range entry.Links {
			links = append(links, map[string]any{
				"target": link.Target,
				"weight": link.Weight,
				"basis":  link.Basis,
			})
		}

		updates["links"] = links
	}

	if len(entry.Absorbed) > 0 {
		absorbed := make([]map[string]any, 0, len(entry.Absorbed))

		for _, rec := range entry.Absorbed {
			absorbed = append(absorbed, map[string]any{
				"from":               rec.From,
				"surfaced_count":     rec.SurfacedCount,
				"followed_count":     rec.Evaluations.Followed,
				"contradicted_count": rec.Evaluations.Contradicted,
				"ignored_count":      rec.Evaluations.Ignored,
				"content_hash":       rec.ContentHash,
				"merged_at":          rec.MergedAt.Format(time.RFC3339),
			})
		}

		updates["absorbed"] = absorbed
	}

	return updates
}

func parseJSONL(data []byte) []registry.InstructionEntry {
	var entries []registry.InstructionEntry

	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry registry.InstructionEntry

		unmarshalErr := json.Unmarshal([]byte(line), &entry)
		if unmarshalErr != nil {
			continue // skip malformed lines
		}

		entries = append(entries, entry)
	}

	return entries
}
