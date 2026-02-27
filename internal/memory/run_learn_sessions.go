package memory

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// RunLearnSessions processes unprocessed sessions and extracts learnings.
func RunLearnSessions(args LearnSessionsArgs, homeDir string) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		memoryRoot = filepath.Join(homeDir, ".claude", "memory")
	}

	db, err := InitEmbeddingsDB(memoryRoot)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	defer func() { _ = db.Close() }()

	if args.ResetLast > 0 {
		count, err := ResetLastNSessions(db, args.ResetLast, memoryRoot)
		if err != nil {
			return fmt.Errorf("failed to reset sessions: %w", err)
		}

		fmt.Printf("Reset %d processed session(s)\n", count)

		return nil
	}

	minSize, err := learnParseSize(args.MinSize)
	if err != nil {
		return fmt.Errorf("invalid min-size: %w", err)
	}

	projectsDir := filepath.Join(homeDir, ".claude", "projects")

	discoverOpts := DiscoverOpts{
		ProjectsDir: projectsDir,
		Days:        args.Days,
		Last:        args.Last,
		MinSize:     minSize,
	}

	allSessions, err := DiscoverSessions(discoverOpts)
	if err != nil {
		return fmt.Errorf("failed to discover sessions: %w", err)
	}

	var unevaluated []DiscoveredSession

	for _, session := range allSessions {
		processed, err := IsSessionProcessed(db, session.SessionID)
		if err != nil {
			return fmt.Errorf("failed to check if session processed: %w", err)
		}

		if !processed {
			unevaluated = append(unevaluated, session)
		}
	}

	var totalSize int64

	projectMap := make(map[string]bool)

	for _, session := range unevaluated {
		totalSize += session.Size
		projectMap[session.Project] = true
	}

	fmt.Printf("Found %d unevaluated session(s) across %d project(s) (~%s)\n",
		len(unevaluated), len(projectMap), learnFormatSize(totalSize))

	if len(unevaluated) == 0 {
		return nil
	}

	if args.DryRun {
		fmt.Println("\nSessions to process:")

		for _, session := range unevaluated {
			fmt.Printf("  - %s (%s) - %s\n", session.SessionID, session.Project, learnFormatSize(session.Size))
		}

		return nil
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	fmt.Println()

	processed := 0

	for i, session := range unevaluated {
		if ctx.Err() != nil {
			fmt.Printf("\nInterrupted. Processed %d/%d session(s).\n", processed, len(unevaluated))
			return nil
		}

		fmt.Printf("[%d/%d] Extracting %s (%s)...\n", i+1, len(unevaluated), session.SessionID, session.Project)

		items, status, err := learnProcessWithTimeout(ctx, session, memoryRoot)
		if err != nil {
			if ctx.Err() != nil {
				fmt.Printf("\nInterrupted. Processed %d/%d session(s).\n", processed, len(unevaluated))
				return nil
			}

			fmt.Fprintf(os.Stderr, "  -> Error: %v (will retry on next run)\n", err)

			continue
		}

		fmt.Printf("  -> %d learning(s) extracted\n", len(items))

		for _, item := range items {
			fmt.Printf("     [%s] %s\n", item.Type, item.Content)
		}

		if err := RecordProcessedSession(db, session.SessionID, session.Project, len(items), status); err != nil {
			fmt.Fprintf(os.Stderr, "  -> Warning: failed to record session: %v\n", err)
		}

		processed++
	}

	fmt.Printf("\nProcessed %d session(s)\n", processed)

	return nil
}

// learnApplyResult writes the offset and stores principles from a batch extract result.
func learnApplyResult(result BatchExtractResult, session DiscoveredSession, memoryRoot, offsetDir, offsetFile string) ([]SessionExtractedItem, string) {
	_ = os.MkdirAll(offsetDir, 0755)
	_ = os.WriteFile(offsetFile, []byte(strconv.FormatInt(result.EndOffset, 10)), 0644)

	var items []SessionExtractedItem

	for _, p := range result.Principles {
		learnErr := Learn(LearnOpts{
			Message:    p.Principle,
			Project:    session.Project,
			Source:     "internal",
			Type:       "discovery",
			MemoryRoot: memoryRoot,
			PrecomputedObservation: &Observation{
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

		items = append(items, SessionExtractedItem{
			Type:    p.Category,
			Content: p.Principle,
		})
	}

	status := "success"
	if result.ChunkFailures > 0 {
		status = fmt.Sprintf("partial (%d/%d chunks failed)", result.ChunkFailures, result.ChunkCount)
	}

	return items, status
}

func learnFormatSize(bytes int64) string {
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

func learnParseSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" || s == "0" {
		return 0, nil
	}

	var (
		numStr string
		unit   string
		sb     strings.Builder
	)

	for i, c := range s {
		if c >= '0' && c <= '9' || c == '.' {
			sb.WriteRune(c)
		} else {
			unit = s[i:]
			break
		}
	}

	numStr = sb.String()

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

func learnProcessSession(session DiscoveredSession, memoryRoot string) ([]SessionExtractedItem, string, error) {
	ext := NewLLMExtractor()
	if ext == nil {
		return nil, "error", errors.New("LLM extractor unavailable (keychain auth failed)")
	}

	directExt, ok := ext.(*DirectAPIExtractor)
	if !ok {
		return nil, "error", errors.New("batch extraction requires DirectAPIExtractor")
	}

	sessionID := session.SessionID
	offsetDir := filepath.Join(memoryRoot, "offsets")
	offsetFile := filepath.Join(offsetDir, sessionID+".offset")

	var startOffset int64

	if data, err := os.ReadFile(offsetFile); err == nil {
		if v, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); err == nil {
			startOffset = v
		}
	}

	result, err := BatchExtractSession(context.Background(), session.Path, directExt, startOffset, os.Stdout)
	if err != nil {
		return nil, "error", err
	}

	items, status := learnApplyResult(*result, session, memoryRoot, offsetDir, offsetFile)

	return items, status, nil
}

func learnProcessWithTimeout(ctx context.Context, session DiscoveredSession, memoryRoot string) ([]SessionExtractedItem, string, error) {
	type result struct {
		items  []SessionExtractedItem
		status string
		err    error
	}

	done := make(chan result, 1)

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	go func() {
		items, status, err := learnProcessSession(session, memoryRoot)
		done <- result{items, status, err}
	}()

	select {
	case r := <-done:
		return r.items, r.status, r.err
	case <-timeoutCtx.Done():
		if ctx.Err() != nil {
			return nil, "interrupted", ctx.Err()
		}

		return nil, "timeout", errors.New("processing timed out after 5 minutes")
	}
}
