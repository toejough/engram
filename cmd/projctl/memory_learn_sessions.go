package main

import (
	"context"
	"fmt"
	"os"
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
		count, err := memory.ResetLastNSessions(db, args.ResetLast)
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

	// Process each session sequentially
	fmt.Println()
	for i, session := range unevaluated {
		fmt.Printf("[%d/%d] Extracting %s (%s)...\n", i+1, len(unevaluated), session.SessionID, session.Project)

		// Process with 60s timeout
		itemsFound, status, err := processSessionWithTimeout(session, memoryRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  -> Error: %v\n", err)
			// Record failure
			if recErr := memory.RecordProcessedSession(db, session.SessionID, session.Project, 0, "error"); recErr != nil {
				fmt.Fprintf(os.Stderr, "  -> Warning: failed to record error status: %v\n", recErr)
			}
			continue
		}

		fmt.Printf("  -> %d learning(s) extracted\n", itemsFound)

		// Record to processed_sessions table
		if err := memory.RecordProcessedSession(db, session.SessionID, session.Project, itemsFound, status); err != nil {
			fmt.Fprintf(os.Stderr, "  -> Warning: failed to record session: %v\n", err)
		}
	}

	fmt.Printf("\nProcessed %d session(s)\n", len(unevaluated))
	return nil
}

// processSessionWithTimeout processes a session with a 60s timeout.
// Returns (itemsFound, status, error).
func processSessionWithTimeout(session memory.DiscoveredSession, memoryRoot string) (int, string, error) {
	type result struct {
		itemsFound int
		status     string
		err        error
	}
	done := make(chan result, 1)

	go func() {
		itemsFound, status, err := processSession(session, memoryRoot)
		done <- result{itemsFound, status, err}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	select {
	case r := <-done:
		return r.itemsFound, r.status, r.err
	case <-ctx.Done():
		return 0, "timeout", fmt.Errorf("processing timed out after 60s")
	}
}

// processSession processes a single session and returns (itemsFound, status, error).
func processSession(session memory.DiscoveredSession, memoryRoot string) (int, string, error) {
	// Wire SemanticMatcher
	matcher := memory.NewMemoryStoreSemanticMatcher(memoryRoot)

	// Wire LLM extractor
	extractor := memory.NewLLMExtractor()
	if extractor == nil {
		return 0, "error", fmt.Errorf("LLM extractor unavailable (keychain auth failed)")
	}

	opts := memory.ExtractSessionOpts{
		TranscriptPath: session.Path,
		MemoryRoot:     memoryRoot,
		Project:        session.Project,
		Matcher:        matcher,
		Extractor:      extractor,
	}

	result, err := memory.ExtractSession(opts)
	if err != nil {
		return 0, "error", err
	}

	return len(result.Items), "success", nil
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
