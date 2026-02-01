// Package corrections provides structured JSONL logging for tracking corrections in the learning loop.
package corrections

import (
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
	panic("not implemented")
}

// LogGlobal appends a correction entry to the global ~/.claude/corrections.jsonl file.
func LogGlobal(message string, context string, opts LogOpts, homeDir string, now func() time.Time) error {
	panic("not implemented")
}

// Read reads correction entries from the project-specific corrections.jsonl file.
func Read(dir string) ([]Entry, error) {
	panic("not implemented")
}

// ReadGlobal reads correction entries from the global ~/.claude/corrections.jsonl file.
func ReadGlobal(homeDir string) ([]Entry, error) {
	panic("not implemented")
}
