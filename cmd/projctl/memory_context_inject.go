package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/toejough/projctl/internal/memory"
)

// memoryContextInjectArgs holds the command-line arguments for context-inject.
type memoryContextInjectArgs struct {
	MemoryRoot    string  `targ:"flag,name=memory-root,desc=Memory root directory (default: ~/.claude/memory)"`
	QueryText     string  `targ:"flag,name=query,desc=Query text for finding relevant memories (default: 'recent important learnings')"`
	MaxEntries    int     `targ:"flag,name=max-entries,desc=Maximum number of entries to include (default: 10)"`
	MaxTokens     int     `targ:"flag,name=max-tokens,desc=Approximate maximum token count (default: 2000)"`
	MinConfidence float64 `targ:"flag,name=min-confidence,desc=Minimum confidence threshold (default: 0.3)"`
	Project       string  `targ:"flag,name=project,desc=Project name for project-aware retrieval (default: derived from stdin cwd)"`
}

// memoryContextInject queries memories and formats them as compact markdown for system prompts.
func memoryContextInject(args memoryContextInjectArgs) error {
	// Read hook input from stdin for project derivation
	project := args.Project
	if project == "" {
		hookInput, _ := memory.ParseHookInput(os.Stdin)
		if hookInput != nil {
			project = memory.DeriveProjectName(hookInput.Cwd)
		}
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
		MemoryRoot:    memoryRoot,
		QueryText:     queryText,
		MaxEntries:    maxEntries,
		MaxTokens:     maxTokens,
		MinConfidence: minConfidence,
		Project:       project,
	}

	result, err := memory.ContextInject(opts)
	if err != nil {
		return fmt.Errorf("context injection failed: %w", err)
	}

	// Print the markdown to stdout
	fmt.Print(result)

	return nil
}
