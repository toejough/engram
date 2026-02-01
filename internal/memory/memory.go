// Package memory provides memory management operations for storing learnings.
package memory

import (
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
	return nil, fmt.Errorf("not implemented")
}
