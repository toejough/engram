package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/toejough/projctl/internal/memory"
)

// memoryExtractSessionArgs holds the command-line arguments for extract-session.
type memoryExtractSessionArgs struct {
	TranscriptPath string `targ:"flag,name=transcript,desc=Path to JSONL transcript file"`
	MemoryRoot     string `targ:"flag,name=memory-root,desc=Memory root directory (default: ~/.claude/memory)"`
	Project        string `targ:"flag,name=project,desc=Project name for tagging extracted learnings (default: derived from stdin cwd)"`
}

// memoryExtractSession extracts learnings from a Claude Code session transcript.
func memoryExtractSession(args memoryExtractSessionArgs) error {
	// Read hook input from stdin for project and transcript derivation
	project := args.Project
	transcriptPath := args.TranscriptPath

	hookInput, _ := memory.ParseHookInput(os.Stdin)
	if hookInput != nil {
		if project == "" {
			project = memory.DeriveProjectName(hookInput.Cwd)
		}
		if transcriptPath == "" && hookInput.TranscriptPath != "" {
			transcriptPath = hookInput.TranscriptPath
		}
	}

	// Validate that transcript file is provided
	if transcriptPath == "" {
		fmt.Fprintln(os.Stderr, "Error: --transcript must be provided (or pass transcript_path via stdin JSON)")
		os.Exit(1)
	}

	// Set up memory root
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = filepath.Join(home, ".claude", "memory")
	}

	// Wire SemanticMatcher (uses local ONNX, no LLM needed)
	matcher := memory.NewMemoryStoreSemanticMatcher(memoryRoot)

	// Call internal ExtractSession function
	opts := memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     memoryRoot,
		Project:        project,
		Matcher:        matcher,
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

	if project != "" {
		fmt.Printf("Project: %s\n", project)
	}
	fmt.Printf("Memory root: %s\n", memoryRoot)

	return nil
}
