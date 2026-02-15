package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ChangelogEntry represents a single mutation event in the memory system.
type ChangelogEntry struct {
	Timestamp       time.Time         `json:"timestamp"`
	Action          string            `json:"action"`
	SourceTier      string            `json:"source_tier,omitempty"`
	DestinationTier string            `json:"destination_tier,omitempty"`
	ContentID       string            `json:"content_id,omitempty"`
	ContentSummary  string            `json:"content_summary,omitempty"`
	Reason          string            `json:"reason,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	SessionID       string            `json:"session_id,omitempty"`
}

// ChangelogFilter specifies criteria for filtering changelog entries.
type ChangelogFilter struct {
	Since           time.Time // Only entries after this time
	Action          string    // Filter by action type
	SourceTier      string    // Filter by source tier
	DestinationTier string    // Filter by destination tier
}

// WriteChangelogEntry appends a changelog entry as a JSON line to changelog.jsonl.
// The memoryRoot directory is created if it doesn't exist.
// ContentSummary is truncated to 100 characters.
func WriteChangelogEntry(memoryRoot string, entry ChangelogEntry) error {
	if err := os.MkdirAll(memoryRoot, 0755); err != nil {
		return fmt.Errorf("failed to create memory directory: %w", err)
	}

	// Set timestamp if not already set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Truncate content summary to 100 chars
	if len(entry.ContentSummary) > 100 {
		entry.ContentSummary = entry.ContentSummary[:100]
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal changelog entry: %w", err)
	}
	data = append(data, '\n')

	logPath := filepath.Join(memoryRoot, "changelog.jsonl")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open changelog: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write changelog entry: %w", err)
	}

	return nil
}

// ReadChangelogEntries reads changelog entries from changelog.jsonl, applying optional filters.
// Returns an empty slice (not an error) if the file doesn't exist.
func ReadChangelogEntries(memoryRoot string, filter ChangelogFilter) ([]ChangelogEntry, error) {
	logPath := filepath.Join(memoryRoot, "changelog.jsonl")

	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open changelog: %w", err)
	}
	defer f.Close()

	var entries []ChangelogEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry ChangelogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue // skip malformed lines
		}

		if matchesFilter(entry, filter) {
			entries = append(entries, entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read changelog: %w", err)
	}

	return entries, nil
}

// logChangelogMutation is an internal helper for logging optimize mutations.
// It silently ignores errors since changelog logging is best-effort.
func logChangelogMutation(memoryRoot, action, sourceTier, destTier, reason string) {
	_ = WriteChangelogEntry(memoryRoot, ChangelogEntry{
		Action:          action,
		SourceTier:      sourceTier,
		DestinationTier: destTier,
		Reason:          reason,
	})
}

func matchesFilter(entry ChangelogEntry, filter ChangelogFilter) bool {
	if !filter.Since.IsZero() && entry.Timestamp.Before(filter.Since) {
		return false
	}
	if filter.Action != "" && entry.Action != filter.Action {
		return false
	}
	if filter.SourceTier != "" && entry.SourceTier != filter.SourceTier {
		return false
	}
	if filter.DestinationTier != "" && entry.DestinationTier != filter.DestinationTier {
		return false
	}
	return true
}
