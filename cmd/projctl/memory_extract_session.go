package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/toejough/projctl/internal/memory"
)

// memoryExtractSessionArgs holds the command-line arguments for extract-session.
type memoryExtractSessionArgs struct {
	TranscriptPath string `targ:"--transcript" help:"Path to JSONL transcript file"`
	MemoryRoot     string `targ:"--memory-root" help:"Memory root directory (default: ~/.claude/memory)"`
}

// memoryExtractSession extracts learnings from a Claude Code session transcript.
func memoryExtractSession(args memoryExtractSessionArgs) error {
	// Validate that transcript file is provided
	if args.TranscriptPath == "" {
		fmt.Fprintln(os.Stderr, "Error: --transcript must be provided")
		os.Exit(1)
	}

	transcriptPath := args.TranscriptPath

	// Set up memory root
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = filepath.Join(home, ".claude", "memory")
	}

	// Call internal ExtractSession function
	opts := memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     memoryRoot,
	}

	result, err := memory.ExtractSession(opts)
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	// Print summary to stdout
	fmt.Printf("Session extraction complete\n")
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Items extracted: %d\n", result.ItemsExtracted)

	if len(result.ConfidenceDistribution) > 0 {
		fmt.Printf("Confidence distribution:\n")
		for confidence, count := range result.ConfidenceDistribution {
			fmt.Printf("  %.1f: %d items\n", confidence, count)
		}
	}

	fmt.Printf("Memory root: %s\n", memoryRoot)

	return nil
}
