package learn

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"engram/internal/keyword"
	"engram/internal/memory"
)

// JSONMergeWriter writes merged memories as JSON (for testing).
type JSONMergeWriter struct{}

// UpdateMerged writes the merged memory fields to disk as JSON.
func (w *JSONMergeWriter) UpdateMerged(
	existing *memory.Stored,
	principle string,
	keywords, concepts []string,
	now time.Time,
) error {
	data := map[string]any{
		"title":      existing.Title,
		"principle":  principle,
		"keywords":   keywords,
		"concepts":   concepts,
		"updated_at": now,
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling merged memory: %w", err)
	}

	err = os.WriteFile(existing.FilePath, jsonBytes, mergedFileMode)
	if err != nil {
		return fmt.Errorf("writing merged memory file: %w", err)
	}

	return nil
}

// TOMLMergeWriter writes merged memories to TOML files (UC-33).
type TOMLMergeWriter struct{}

// UpdateMerged reads an existing TOML memory file, updates it with merged fields, and writes it back.
func (w *TOMLMergeWriter) UpdateMerged(
	existing *memory.Stored,
	principle string,
	keywords, concepts []string,
	now time.Time,
) error {
	// Read the existing TOML file (just to verify it exists)
	_, err := os.ReadFile(existing.FilePath)
	if err != nil {
		return fmt.Errorf("reading existing memory: %w", err)
	}

	// Build new TOML content
	var content strings.Builder

	fmt.Fprintf(&content, "principle = %q\n", principle)
	fmt.Fprintf(&content, "updated_at = %q\n", now.Format(time.RFC3339))

	keywords = keyword.NormalizeAll(keywords)

	content.WriteString("keywords = [")

	for i, k := range keywords {
		if i > 0 {
			content.WriteString(", ")
		}

		fmt.Fprintf(&content, "%q", k)
	}

	content.WriteString("]\n")
	content.WriteString("concepts = [")

	for i, c := range concepts {
		if i > 0 {
			content.WriteString(", ")
		}

		fmt.Fprintf(&content, "%q", c)
	}

	content.WriteString("]\n")

	writeErr := os.WriteFile(existing.FilePath, []byte(content.String()), mergedFileMode)
	if writeErr != nil {
		return fmt.Errorf("writing merged memory: %w", writeErr)
	}

	return nil
}

// unexported constants.
const (
	mergedFileMode = 0o600
)
