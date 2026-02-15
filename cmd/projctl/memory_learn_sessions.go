package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/toejough/projctl/internal/memory"
)

// memoryLearnSessionsArgs holds the command-line arguments for learn-sessions.
type memoryLearnSessionsArgs struct {
	Days       int    `targ:"flag,name=days,desc=Look back N days (default: 7),default=7"`
	Last       int    `targ:"flag,name=last,desc=Process only last N sessions (overrides --days)"`
	MinSize    string `targ:"flag,name=min-size,desc=Minimum session size (default: 8KB),default=8KB"`
	DryRun     bool   `targ:"flag,name=dry-run,desc=List sessions without processing"`
	ResetLast  int    `targ:"flag,name=reset-last,desc=Reset last N processed sessions and exit"`
	MemoryRoot string `targ:"flag,name=memory-root,desc=Memory root directory (default: ~/.claude/memory)"`
}

// memoryLearnSessions learns from unevaluated session transcripts.
func memoryLearnSessions(args memoryLearnSessionsArgs) error {
	// Set up memory root
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = filepath.Join(home, ".claude", "memory")
	}

	// Initialize database
	db, err := memory.InitEmbeddingsDB(memoryRoot)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Handle --reset-last flag
	if args.ResetLast > 0 {
		count, err := memory.ResetLastNSessions(db, args.ResetLast, memoryRoot)
		if err != nil {
			return fmt.Errorf("failed to reset sessions: %w", err)
		}
		fmt.Printf("Reset %d processed session(s)\n", count)
		return nil
	}

	// Parse MinSize
	minSize, err := parseSize(args.MinSize)
	if err != nil {
		return fmt.Errorf("invalid min-size: %w", err)
	}

	// Discover sessions from ~/.claude/projects/
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	projectsDir := filepath.Join(home, ".claude", "projects")

	discoverOpts := memory.DiscoverOpts{
		ProjectsDir: projectsDir,
		Days:        args.Days,
		Last:        args.Last,
		MinSize:     minSize,
	}

	allSessions, err := memory.DiscoverSessions(discoverOpts)
	if err != nil {
		return fmt.Errorf("failed to discover sessions: %w", err)
	}

	// Filter out already-processed sessions
	var unevaluated []memory.DiscoveredSession
	for _, session := range allSessions {
		processed, err := memory.IsSessionProcessed(db, session.SessionID)
		if err != nil {
			return fmt.Errorf("failed to check if session processed: %w", err)
		}
		if !processed {
			unevaluated = append(unevaluated, session)
		}
	}

	// Calculate total size
	var totalSize int64
	projectMap := make(map[string]bool)
	for _, session := range unevaluated {
		totalSize += session.Size
		projectMap[session.Project] = true
	}

	// Print summary
	fmt.Printf("Found %d unevaluated session(s) across %d project(s) (~%s)\n",
		len(unevaluated), len(projectMap), formatSize(totalSize))

	if len(unevaluated) == 0 {
		return nil
	}

	// If --dry-run, list sessions and return
	if args.DryRun {
		fmt.Println("\nSessions to process:")
		for _, session := range unevaluated {
			fmt.Printf("  - %s (%s) - %s\n", session.SessionID, session.Project, formatSize(session.Size))
		}
		return nil
	}

	// Set up signal handling for clean Ctrl-C exit
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Process each session sequentially
	fmt.Println()
	processed := 0
	for i, session := range unevaluated {
		// Check for cancellation before starting next session
		if ctx.Err() != nil {
			fmt.Printf("\nInterrupted. Processed %d/%d session(s).\n", processed, len(unevaluated))
			return nil
		}

		fmt.Printf("[%d/%d] Extracting %s (%s)...\n", i+1, len(unevaluated), session.SessionID, session.Project)

		// Process with 60s timeout, respecting Ctrl-C
		items, status, err := processSessionWithTimeout(ctx, session, memoryRoot)
		if err != nil {
			if ctx.Err() != nil {
				fmt.Printf("\nInterrupted. Processed %d/%d session(s).\n", processed, len(unevaluated))
				return nil
			}
			fmt.Fprintf(os.Stderr, "  -> Error: %v\n", err)
			// Record failure
			if recErr := memory.RecordProcessedSession(db, session.SessionID, session.Project, 0, "error"); recErr != nil {
				fmt.Fprintf(os.Stderr, "  -> Warning: failed to record error status: %v\n", recErr)
			}
			continue
		}

		fmt.Printf("  -> %d learning(s) extracted\n", len(items))
		for _, item := range items {
			fmt.Printf("     [%s] %s\n", item.Type, item.Content)
		}

		// Record to processed_sessions table
		if err := memory.RecordProcessedSession(db, session.SessionID, session.Project, len(items), status); err != nil {
			fmt.Fprintf(os.Stderr, "  -> Warning: failed to record session: %v\n", err)
		}
		processed++
	}

	fmt.Printf("\nProcessed %d session(s)\n", processed)
	return nil
}

// processSessionWithTimeout processes a session with a 60s timeout.
// Respects parent context cancellation (e.g. Ctrl-C).
func processSessionWithTimeout(ctx context.Context, session memory.DiscoveredSession, memoryRoot string) ([]memory.SessionExtractedItem, string, error) {
	type result struct {
		items  []memory.SessionExtractedItem
		status string
		err    error
	}
	done := make(chan result, 1)

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	go func() {
		items, status, err := processSession(session, memoryRoot)
		done <- result{items, status, err}
	}()

	select {
	case r := <-done:
		return r.items, r.status, r.err
	case <-timeoutCtx.Done():
		if ctx.Err() != nil {
			return nil, "interrupted", ctx.Err()
		}
		return nil, "timeout", fmt.Errorf("processing timed out after 5 minutes")
	}
}

// processSession processes a single session using the batch extraction pipeline.
func processSession(session memory.DiscoveredSession, memoryRoot string) ([]memory.SessionExtractedItem, string, error) {
	// Wire LLM extractor
	ext := memory.NewLLMExtractor()
	if ext == nil {
		return nil, "error", fmt.Errorf("LLM extractor unavailable (keychain auth failed)")
	}

	// Cast to DirectAPIExtractor — BatchExtractSession needs the concrete type
	// for CallAPIWithMessages access
	directExt, ok := ext.(*memory.DirectAPIExtractor)
	if !ok {
		return nil, "error", fmt.Errorf("batch extraction requires DirectAPIExtractor")
	}

	// Read stored offset for incremental extraction
	sessionID := session.SessionID
	offsetDir := filepath.Join(memoryRoot, "offsets")
	offsetFile := filepath.Join(offsetDir, sessionID+".offset")
	var startOffset int64
	if data, err := os.ReadFile(offsetFile); err == nil {
		if v, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); err == nil {
			startOffset = v
		}
	}

	result, err := memory.BatchExtractSession(context.Background(), session.Path, directExt, startOffset)
	if err != nil {
		return nil, "error", err
	}

	// Persist updated offset
	_ = os.MkdirAll(offsetDir, 0755)
	_ = os.WriteFile(offsetFile, []byte(strconv.FormatInt(result.EndOffset, 10)), 0644)

	// Store each principle via Learn()
	var items []memory.SessionExtractedItem
	for _, p := range result.Principles {
		learnErr := memory.Learn(memory.LearnOpts{
			Message:    p.Principle,
			Project:    session.Project,
			Source:     "internal",
			Type:       "discovery",
			MemoryRoot: memoryRoot,
			Extractor:  ext,
			PrecomputedObservation: &memory.Observation{
				Type:      p.Category,
				Concepts:  []string{p.Category},
				Principle: p.Principle,
				Rationale: p.Evidence,
			},
		})
		if learnErr != nil {
			fmt.Fprintf(os.Stderr, "  -> Warning: failed to store principle: %v\n", learnErr)
			continue
		}
		items = append(items, memory.SessionExtractedItem{
			Type:    p.Category,
			Content: p.Principle,
		})
	}

	status := "success"
	if result.ChunkFailures > 0 {
		status = fmt.Sprintf("partial (%d/%d chunks failed)", result.ChunkFailures, result.ChunkCount)
	}

	return items, status, nil
}

// parseSize parses a human-readable size string (e.g., "8KB", "1MB") into bytes.
func parseSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" || s == "0" {
		return 0, nil
	}

	// Extract numeric part and unit
	var numStr string
	var unit string
	for i, c := range s {
		if c >= '0' && c <= '9' || c == '.' {
			numStr += string(c)
		} else {
			unit = s[i:]
			break
		}
	}

	if numStr == "" {
		return 0, fmt.Errorf("no numeric value in size: %s", s)
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric value: %s", numStr)
	}

	var multiplier int64
	switch unit {
	case "", "B":
		multiplier = 1
	case "KB":
		multiplier = 1024
	case "MB":
		multiplier = 1024 * 1024
	case "GB":
		multiplier = 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown size unit: %s", unit)
	}

	return int64(num * float64(multiplier)), nil
}

// formatSize formats a byte count into a human-readable string.
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
