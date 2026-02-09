package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/toejough/projctl/internal/memory"
)

// memoryContextInjectArgs holds the command-line arguments for context-inject.
type memoryContextInjectArgs struct {
	MemoryRoot   string  `targ:"--memory-root" help:"Memory root directory (default: ~/.claude/memory)"`
	QueryText    string  `targ:"--query" help:"Query text for finding relevant memories (default: 'recent important learnings')"`
	MaxEntries   int     `targ:"--max-entries" help:"Maximum number of entries to include (default: 10)"`
	MaxTokens    int     `targ:"--max-tokens" help:"Approximate maximum token count (default: 2000)"`
	MinConfidence float64 `targ:"--min-confidence" help:"Minimum confidence threshold (default: 0.3)"`
}

// memoryContextInject queries memories and formats them as compact markdown for system prompts.
func memoryContextInject(args memoryContextInjectArgs) error {
	// Set up memory root
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = filepath.Join(home, ".claude", "memory")
	}

	// Set defaults for other parameters
	queryText := args.QueryText
	if queryText == "" {
		queryText = "recent important learnings"
	}

	maxEntries := args.MaxEntries
	if maxEntries == 0 {
		maxEntries = 10
	}

	maxTokens := args.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2000
	}

	minConfidence := args.MinConfidence
	if minConfidence == 0 {
		minConfidence = 0.3
	}

	// Call internal ContextInject function
	opts := memory.ContextInjectOpts{
		MemoryRoot:   memoryRoot,
		QueryText:    queryText,
		MaxEntries:   maxEntries,
		MaxTokens:    maxTokens,
		MinConfidence: minConfidence,
	}

	result, err := memory.ContextInject(opts)
	if err != nil {
		return fmt.Errorf("context injection failed: %w", err)
	}

	// Print the markdown to stdout
	fmt.Print(result)

	return nil
}
