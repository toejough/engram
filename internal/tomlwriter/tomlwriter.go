// Package tomlwriter writes enriched memories to TOML files atomically.
package tomlwriter

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"engram/internal/memory"
)

// Writer writes enriched memories to TOML files.
type Writer struct{}

// New creates a Writer.
func New() *Writer {
	return &Writer{}
}

// tomlRecord is the on-disk TOML representation with snake_case field names.
type tomlRecord struct {
	Title           string   `toml:"title"`
	Content         string   `toml:"content"`
	ObservationType string   `toml:"observation_type"`
	Concepts        []string `toml:"concepts"`
	Keywords        []string `toml:"keywords"`
	Principle       string   `toml:"principle"`
	AntiPattern     string   `toml:"anti_pattern"`
	Rationale       string   `toml:"rationale"`
	Confidence      string   `toml:"confidence"`
	CreatedAt       string   `toml:"created_at"`
	UpdatedAt       string   `toml:"updated_at"`
}

const memoriesDirPerm = 0o750

var nonAlphanumericRun = regexp.MustCompile(`[^a-z0-9]+`)

// slugify converts a filename summary to a lowercase hyphen-separated slug.
// Returns "memory" if the result would otherwise be empty.
func slugify(summary string) string {
	lower := strings.ToLower(summary)
	slug := nonAlphanumericRun.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")

	if slug == "" {
		return "memory"
	}

	return slug
}

// Write writes mem as a TOML file under <dataDir>/memories/<slug>.toml.
// If the slug path is already taken, it appends -2, -3, etc. until a free name is found.
// The file is written atomically via a temp file and rename.
// Returns the absolute path of the written file.
func (w *Writer) Write(mem *memory.Enriched, dataDir string) (string, error) {
	memoriesDir := filepath.Join(dataDir, "memories")

	mkdirErr := os.MkdirAll(memoriesDir, memoriesDirPerm)
	if mkdirErr != nil {
		return "", fmt.Errorf("tomlwriter: create memories dir: %w", mkdirErr)
	}

	slug := slugify(mem.FilenameSummary)

	finalPath, err := availablePath(memoriesDir, slug)
	if err != nil {
		return "", err
	}

	concepts := mem.Concepts
	if concepts == nil {
		concepts = make([]string, 0)
	}

	keywords := mem.Keywords
	if keywords == nil {
		keywords = make([]string, 0)
	}

	record := tomlRecord{
		Title:           mem.Title,
		Content:         mem.Content,
		ObservationType: mem.ObservationType,
		Concepts:        concepts,
		Keywords:        keywords,
		Principle:       mem.Principle,
		AntiPattern:     mem.AntiPattern,
		Rationale:       mem.Rationale,
		Confidence:      mem.Confidence,
		CreatedAt:       mem.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       mem.UpdatedAt.UTC().Format(time.RFC3339),
	}

	writeErr := writeAtomic(memoriesDir, finalPath, record)
	if writeErr != nil {
		return "", writeErr
	}

	return finalPath, nil
}

// availablePath returns the first available file path for slug in memoriesDir.
// Tries <slug>.toml, then <slug>-2.toml, <slug>-3.toml, etc.
func availablePath(memoriesDir, slug string) (string, error) {
	candidate := filepath.Join(memoriesDir, slug+".toml")

	for suffix := 2; ; suffix++ {
		_, statErr := os.Stat(candidate)
		if os.IsNotExist(statErr) {
			return candidate, nil
		}

		if statErr != nil {
			return "", fmt.Errorf("tomlwriter: stat %s: %w", candidate, statErr)
		}

		candidate = filepath.Join(memoriesDir, fmt.Sprintf("%s-%d.toml", slug, suffix))
	}
}

// writeAtomic writes record as TOML to finalPath using a temp file and rename.
// Paths are constructed internally via filepath.Join — no user-controlled input.
func writeAtomic(memoriesDir, finalPath string, record tomlRecord) error {
	tempFile, err := os.CreateTemp(memoriesDir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("tomlwriter: create temp file: %w", err)
	}

	tempPath := filepath.Clean(tempFile.Name())
	cleanFinal := filepath.Clean(finalPath)

	encodeErr := toml.NewEncoder(tempFile).Encode(record)
	if encodeErr != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempPath) //nolint:gosec // safe: tempPath from CreateTemp

		return fmt.Errorf("tomlwriter: encode TOML: %w", encodeErr)
	}

	closeErr := tempFile.Close()
	if closeErr != nil {
		_ = os.Remove(tempPath) //nolint:gosec // safe: tempPath from CreateTemp
		return fmt.Errorf("tomlwriter: close temp file: %w", closeErr)
	}

	renameErr := os.Rename(tempPath, cleanFinal) //nolint:gosec // safe: internal paths
	if renameErr != nil {
		_ = os.Remove(tempPath) //nolint:gosec // safe: tempPath from CreateTemp
		return fmt.Errorf("tomlwriter: rename to final path: %w", renameErr)
	}

	return nil
}
