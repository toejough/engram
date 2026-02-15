package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

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

	// Validate that transcript file is provided and exists
	if transcriptPath == "" {
		fmt.Fprintln(os.Stderr, "Warning: no transcript path provided, skipping extraction")
		return nil
	}
	if _, err := os.Stat(transcriptPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Warning: transcript file not found (%s), skipping extraction\n", transcriptPath)
		return nil
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

	// Wire LLM extractor for enrichment (uses Haiku via direct API)
	extractor := memory.NewLLMExtractor()

	// Call internal ExtractSession function
	opts := memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     memoryRoot,
		Project:        project,
		Matcher:        matcher,
		Extractor:      extractor,
	}

	result, err := memory.ExtractSession(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "extraction failed: %v\n", err)
		return fmt.Errorf("extraction failed: %w", err)
	}

	// Build session summary from extraction result
	learnings := make([]memory.LearningItem, 0, len(result.Items))
	for _, item := range result.Items {
		learnings = append(learnings, memory.LearningItem{
			Type:       item.Type,
			Content:    item.Content,
			Confidence: item.Confidence,
		})
	}

	summary := memory.SessionSummary{
		SessionID:   filepath.Base(transcriptPath),
		ExtractedAt: time.Now(),
		Learnings:   learnings,
		// Note: RetrievalsCount, SkillCandidates, etc. will be populated by future tasks
	}

	// Print formatted summary
	memory.PrintSessionSummary(summary, os.Stdout)

	return nil
}
