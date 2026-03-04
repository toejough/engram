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
)

// EnrichedMemory holds the structured data to persist as a TOML memory file.
type EnrichedMemory struct {
	Title           string
	Content         string
	ObservationType string
	Concepts        []string
	Keywords        []string
	Principle       string
	AntiPattern     string
	Rationale       string
	FilenameSummary string // 3-5 words used to generate the slug
	Confidence      string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// MemoryWriter writes enriched memories to TOML files.
type MemoryWriter interface {
	Write(memory *EnrichedMemory, dataDir string) (string, error)
}

// Writer implements MemoryWriter.
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

// Write writes memory as a TOML file under <dataDir>/memories/<slug>.toml.
// If the slug path is already taken, it appends -2, -3, etc. until a free name is found.
// The file is written atomically via a temp file and rename.
// Returns the absolute path of the written file.
func (w *Writer) Write(memory *EnrichedMemory, dataDir string) (string, error) {
	memoriesDir := filepath.Join(dataDir, "memories")
	if err := os.MkdirAll(memoriesDir, 0o755); err != nil {
		return "", fmt.Errorf("tomlwriter: create memories dir: %w", err)
	}

	slug := slugify(memory.FilenameSummary)

	finalPath, err := availablePath(memoriesDir, slug)
	if err != nil {
		return "", err
	}

	concepts := memory.Concepts
	if concepts == nil {
		concepts = make([]string, 0)
	}

	keywords := memory.Keywords
	if keywords == nil {
		keywords = make([]string, 0)
	}

	record := tomlRecord{
		Title:           memory.Title,
		Content:         memory.Content,
		ObservationType: memory.ObservationType,
		Concepts:        concepts,
		Keywords:        keywords,
		Principle:       memory.Principle,
		AntiPattern:     memory.AntiPattern,
		Rationale:       memory.Rationale,
		Confidence:      memory.Confidence,
		CreatedAt:       memory.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       memory.UpdatedAt.UTC().Format(time.RFC3339),
	}

	if err := writeAtomic(memoriesDir, finalPath, record); err != nil {
		return "", err
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
func writeAtomic(memoriesDir, finalPath string, record tomlRecord) error {
	tempFile, err := os.CreateTemp(memoriesDir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("tomlwriter: create temp file: %w", err)
	}

	tempPath := tempFile.Name()

	if encodeErr := toml.NewEncoder(tempFile).Encode(record); encodeErr != nil {
		tempFile.Close()
		os.Remove(tempPath)

		return fmt.Errorf("tomlwriter: encode TOML: %w", encodeErr)
	}

	if closeErr := tempFile.Close(); closeErr != nil {
		os.Remove(tempPath)

		return fmt.Errorf("tomlwriter: close temp file: %w", closeErr)
	}

	if renameErr := os.Rename(tempPath, finalPath); renameErr != nil {
		os.Remove(tempPath)

		return fmt.Errorf("tomlwriter: rename to final path: %w", renameErr)
	}

	return nil
}
