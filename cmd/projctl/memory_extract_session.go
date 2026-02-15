package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
// The entire function is wrapped in a 60s timeout because any step can block:
// stdin read (waiting for EOF), ONNX model load, keychain auth, API calls, etc.
func memoryExtractSession(args memoryExtractSessionArgs) error {
	type funcResult struct {
		err error
	}
	done := make(chan funcResult, 1)
	go func() {
		done <- funcResult{doExtractSession(args)}
	}()

	select {
	case r := <-done:
		return r.err
	case <-time.After(60 * time.Second):
		fmt.Fprintln(os.Stderr, "Warning: extract-session timed out after 60s, skipping")
		return nil
	}
}

func doExtractSession(args memoryExtractSessionArgs) error {
	start := time.Now()
	dbg := func(msg string) {
		fmt.Fprintf(os.Stderr, "[extract-session] %s (+%dms)\n", msg, time.Since(start).Milliseconds())
	}

	// Read hook input from stdin for project and transcript derivation
	project := args.Project
	transcriptPath := args.TranscriptPath

	hookInput, _ := memory.ParseHookInput(os.Stdin)
	dbg("stdin parsed")
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
	dbg("semantic matcher ready")

	// Wire LLM extractor for enrichment (uses Haiku via direct API)
	extractor := memory.NewLLMExtractor()
	dbg("LLM extractor ready")
	if extractor == nil {
		return fmt.Errorf("LLM extractor unavailable (keychain auth failed); cannot extract session without enrichment")
	}

	opts := memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     memoryRoot,
		Project:        project,
		Matcher:        matcher,
		Extractor:      extractor,
	}

	result, err := memory.ExtractSession(opts)
	dbg(fmt.Sprintf("extraction done: %d items, status=%s", len(result.Items), result.Status))
	if err != nil {
		fmt.Fprintf(os.Stderr, "extraction failed: %v\n", err)
		return fmt.Errorf("extraction failed: %w", err)
	}

	// Record processed session (best-effort, don't fail extraction on error)
	sessionID := filepath.Base(transcriptPath)
	sessionID = strings.TrimSuffix(sessionID, ".jsonl")
	if recDB, err := memory.InitEmbeddingsDB(memoryRoot); err == nil {
		_ = memory.RecordProcessedSession(recDB, sessionID, project, len(result.Items), "success")
		_ = recDB.Close()
	}
	dbg("session recorded")

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
	}

	// Print formatted summary
	memory.PrintSessionSummary(summary, os.Stdout)

	return nil
}
