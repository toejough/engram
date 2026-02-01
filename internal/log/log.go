// Package log provides structured JSONL logging for project orchestration.
package log

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	Timestamp       string `json:"timestamp"`
	Level           string `json:"level"`
	Subject         string `json:"subject"`
	Task            string `json:"task,omitempty"`
	Phase           string `json:"phase,omitempty"`
	Model           string `json:"model,omitempty"`
	Session         string `json:"session,omitempty"`
	Message         string `json:"message"`
	Detail          any    `json:"detail,omitempty"`
	TokensEstimate  int    `json:"tokens_estimate,omitempty"`
	ContextEstimate int    `json:"context_estimate,omitempty"`
}

// WriteOpts holds optional fields for a log entry.
type WriteOpts struct {
	Task            string
	Phase           string
	Model           string
	Session         string
	Detail          any
	Tokens          int // Override token estimate (0 = calculate from message)
	ContextEstimate int // Current context usage estimate (tokens)
}

// ReadOpts holds options for reading log entries.
type ReadOpts struct {
	Model   string // Filter by model (empty = all)
	Session string // Filter by session (empty = all)
}

// Write appends a structured JSONL entry to the log file.
func Write(dir string, level string, subject string, message string, opts WriteOpts, now func() time.Time) error {
	if !ValidLevels[level] {
		return fmt.Errorf("invalid level %q (valid: verbose, status, phase)", level)
	}

	if !ValidSubjects[subject] {
		return fmt.Errorf("invalid subject %q (valid: %v)", subject, subjectKeys())
	}

	// Calculate token estimate: chars/4, round up
	tokens := opts.Tokens
	if tokens == 0 && len(message) > 0 {
		tokens = (len(message) + 3) / 4 // Round up
	}

	entry := Entry{
		Timestamp:       now().UTC().Format(time.RFC3339),
		Level:           level,
		Subject:         subject,
		Message:         message,
		Task:            opts.Task,
		Phase:           opts.Phase,
		Model:           opts.Model,
		Session:         opts.Session,
		Detail:          opts.Detail,
		TokensEstimate:  tokens,
		ContextEstimate: opts.ContextEstimate,
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

// Read reads log entries from the log file with optional filtering.
func Read(dir string, opts ReadOpts) ([]Entry, error) {
	logPath := filepath.Join(dir, LogFile)

	content, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}

		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []Entry{}, nil
	}

	entries := make([]Entry, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("failed to parse log entry: %w", err)
		}

		// Apply model filter
		if opts.Model != "" && entry.Model != opts.Model {
			continue
		}

		// Apply session filter
		if opts.Session != "" && entry.Session != opts.Session {
			continue
		}

		entries = append(entries, entry)
	}

	return entries, nil
}
