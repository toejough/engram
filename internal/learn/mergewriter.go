package learn

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"engram/internal/memory"
)

// JSONMergeWriter writes merged memories to JSON files (UC-33, placeholder).
type JSONMergeWriter struct{}

// UpdateMerged implements MergeWriter by serializing merged fields as JSON.
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

	err = os.WriteFile(existing.FilePath, jsonBytes, mergeFileMode)
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

	err = os.WriteFile(existing.FilePath, []byte(content.String()), mergeFileMode)
	if err != nil {
		return fmt.Errorf("writing merged memory: %w", err)
	}

	return nil
}

// unexported constants.
const (
	mergeFileMode = 0o600
)
