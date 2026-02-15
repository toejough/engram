package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, ".claude", "memory", "extract-session.log")
	logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	dbg := func(msg string) {
		line := fmt.Sprintf("%s [extract-session] %s (+%dms)\n",
			time.Now().Format("15:04:05"), msg, time.Since(start).Milliseconds())
		if logFile != nil {
			_, _ = logFile.WriteString(line)
		}
	}
	defer func() {
		if logFile != nil {
			_ = logFile.Close()
		}
	}()
	dbg("starting")

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

	// Read stored offset for incremental extraction
	sessionID := filepath.Base(transcriptPath)
	sessionID = strings.TrimSuffix(sessionID, ".jsonl")

	// Skip if already processed by learn-sessions (batch pipeline)
	if recDB, err := memory.InitEmbeddingsDB(memoryRoot); err == nil {
		processed, checkErr := memory.IsSessionProcessed(recDB, sessionID)
		_ = recDB.Close()
		if checkErr == nil && processed {
			dbg("session already processed by learn-sessions, skipping")
			fmt.Fprintln(os.Stderr, "Session already processed by learn-sessions, skipping")
			return nil
		}
	}

	offsetDir := filepath.Join(memoryRoot, "offsets")
	offsetFile := filepath.Join(offsetDir, sessionID+".offset")
	var startOffset int64
	if data, err := os.ReadFile(offsetFile); err == nil {
		if v, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); err == nil {
			startOffset = v
		}
	}
	dbg(fmt.Sprintf("start offset: %d", startOffset))

	opts := memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     memoryRoot,
		Project:        project,
		Matcher:        matcher,
		Extractor:      extractor,
		StartOffset:    startOffset,
	}

	result, err := memory.ExtractSession(opts)
	dbg(fmt.Sprintf("extraction done: %d items, status=%s, endOffset=%d", len(result.Items), result.Status, result.EndOffset))
	if err != nil {
		fmt.Fprintf(os.Stderr, "extraction failed: %v\n", err)
		return fmt.Errorf("extraction failed: %w", err)
	}

	// Persist new offset for next incremental run
	_ = os.MkdirAll(offsetDir, 0755)
	_ = os.WriteFile(offsetFile, []byte(strconv.FormatInt(result.EndOffset, 10)), 0644)
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
