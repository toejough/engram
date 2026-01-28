// Package log provides structured JSONL logging for project orchestration.
package log

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LogFile is the filename for the updates log.
const LogFile = "updates.jsonl"

// Valid levels.
var ValidLevels = map[string]bool{
	"verbose": true,
	"status":  true,
	"phase":   true,
}

// Valid subjects.
var ValidSubjects = map[string]bool{
	"thinking":     true,
	"skill-result": true,
	"skill-change": true,
	"task-status":  true,
	"alignment":    true,
	"conflict":     true,
	"lesson":       true,
	"phase-change": true,
	"phase-result": true,
}

// Entry is a single log entry.
type Entry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Subject   string `json:"subject"`
	Task      string `json:"task,omitempty"`
	Phase     string `json:"phase,omitempty"`
	Message   string `json:"message"`
	Detail    any    `json:"detail,omitempty"`
}

// WriteOpts holds optional fields for a log entry.
type WriteOpts struct {
	Task   string
	Phase  string
	Detail any
}

// Write appends a structured JSONL entry to the log file.
func Write(dir string, level string, subject string, message string, opts WriteOpts, now func() time.Time) error {
	if !ValidLevels[level] {
		return fmt.Errorf("invalid level %q (valid: verbose, status, phase)", level)
	}

	if !ValidSubjects[subject] {
		return fmt.Errorf("invalid subject %q (valid: %v)", subject, subjectKeys())
	}

	entry := Entry{
		Timestamp: now().UTC().Format(time.RFC3339),
		Level:     level,
		Subject:   subject,
		Message:   message,
		Task:      opts.Task,
		Phase:     opts.Phase,
		Detail:    opts.Detail,
	}

	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	logPath := filepath.Join(dir, LogFile)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}

	return nil
}

func subjectKeys() []string {
	keys := make([]string, 0, len(ValidSubjects))
	for k := range ValidSubjects {
		keys = append(keys, k)
	}

	return keys
}
