// Package corrections provides structured JSONL logging for tracking corrections in the learning loop.
package corrections

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Entry represents a single correction log entry.
type Entry struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	Context   string `json:"context"`
	SessionID string `json:"session_id,omitempty"`
}

// LogOpts holds optional fields for a correction entry.
type LogOpts struct {
	SessionID string
}

// Log appends a correction entry to the project-specific corrections.jsonl file.
func Log(dir string, message string, context string, opts LogOpts, now func() time.Time) error {
	path := filepath.Join(dir, "corrections.jsonl")
	return writeEntry(path, message, context, opts, now)
}

// LogGlobal appends a correction entry to the global ~/.claude/corrections.jsonl file.
func LogGlobal(message string, context string, opts LogOpts, homeDir string, now func() time.Time) error {
	claudeDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return err
	}
	path := filepath.Join(claudeDir, "corrections.jsonl")
	return writeEntry(path, message, context, opts, now)
}

// Read reads correction entries from the project-specific corrections.jsonl file.
func Read(dir string) ([]Entry, error) {
	path := filepath.Join(dir, "corrections.jsonl")
	return readEntries(path)
}

// ReadGlobal reads correction entries from the global ~/.claude/corrections.jsonl file.
func ReadGlobal(homeDir string) ([]Entry, error) {
	path := filepath.Join(homeDir, ".claude", "corrections.jsonl")
	return readEntries(path)
}

func writeEntry(path string, message string, context string, opts LogOpts, now func() time.Time) error {
	if now == nil {
		now = time.Now
	}
	entry := Entry{
		Timestamp: now().UTC().Format(time.RFC3339),
		Message:   message,
		Context:   context,
		SessionID: opts.SessionID,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return err
	}
	if _, err := f.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

func readEntries(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

// Pattern represents a recurring correction pattern detected by analysis.
type Pattern struct {
	Message  string   // Representative message for the pattern
	Count    int      // Number of occurrences
	Proposal string   // Proposed CLAUDE.md addition
	Examples []Entry  // Sample entries that match this pattern
}

// AnalyzeOpts holds options for analyzing correction patterns.
type AnalyzeOpts struct {
	MinOccurrences int // Minimum occurrences to report a pattern (default: 2)
}

// Analyze detects patterns in corrections using fuzzy matching.
// Returns patterns sorted by count (descending).
func Analyze(dir string, opts AnalyzeOpts) ([]Pattern, error) {
	panic("not implemented: TASK-042")
}
