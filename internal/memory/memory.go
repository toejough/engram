// Package memory provides memory management operations for storing learnings.
package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LearnOpts holds options for learning storage.
type LearnOpts struct {
	Message    string
	Project    string
	MemoryRoot string
}

// Learn stores a learning in the memory index.
func Learn(opts LearnOpts) error {
	if opts.Message == "" {
		return fmt.Errorf("message is required")
	}

	// Ensure memory directory exists
	if err := os.MkdirAll(opts.MemoryRoot, 0755); err != nil {
		return fmt.Errorf("failed to create memory directory: %w", err)
	}

	indexPath := filepath.Join(opts.MemoryRoot, "index.md")

	// Open file for appending (create if doesn't exist)
	f, err := os.OpenFile(indexPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open index file: %w", err)
	}
	defer f.Close()

	// Format entry: - YYYY-MM-DD HH:MM: [project] message
	timestamp := time.Now().Format("2006-01-02 15:04")
	var entry string
	if opts.Project != "" {
		entry = fmt.Sprintf("- %s: [%s] %s\n", timestamp, opts.Project, opts.Message)
	} else {
		entry = fmt.Sprintf("- %s: %s\n", timestamp, opts.Message)
	}

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("failed to write entry: %w", err)
	}

	return nil
}

// DecideOpts holds options for decision logging.
type DecideOpts struct {
	Context      string
	Choice       string
	Reason       string
	Alternatives []string
	Project      string
	MemoryRoot   string
}

// DecideResult contains the result of logging a decision.
type DecideResult struct {
	FilePath string
}

// Decide logs a decision with reasoning and alternatives.
func Decide(opts DecideOpts) (*DecideResult, error) {
	if opts.Context == "" {
		return nil, fmt.Errorf("context is required")
	}
	if opts.Choice == "" {
		return nil, fmt.Errorf("choice is required")
	}
	if opts.Reason == "" {
		return nil, fmt.Errorf("reason is required")
	}

	// Ensure decisions directory exists
	decisionsDir := filepath.Join(opts.MemoryRoot, "decisions")
	if err := os.MkdirAll(decisionsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create decisions directory: %w", err)
	}

	// Build filename: {DATE}-{PROJECT}.jsonl
	today := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%s-%s.jsonl", today, opts.Project)
	filePath := filepath.Join(decisionsDir, filename)

	// Open file for appending (create if doesn't exist)
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open decisions file: %w", err)
	}
	defer f.Close()

	// Build JSON entry
	entry := map[string]interface{}{
		"timestamp":    time.Now().Format(time.RFC3339),
		"context":      opts.Context,
		"choice":       opts.Choice,
		"reason":       opts.Reason,
		"alternatives": opts.Alternatives,
	}

	// Marshal and write
	data, err := json.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entry: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write entry: %w", err)
	}
	if _, err := f.WriteString("\n"); err != nil {
		return nil, fmt.Errorf("failed to write newline: %w", err)
	}

	return &DecideResult{
		FilePath: filePath,
	}, nil
}
