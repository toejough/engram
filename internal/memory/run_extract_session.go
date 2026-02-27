package memory

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// RunExtractSession runs extract-session with a timeout.
func RunExtractSession(args ExtractSessionArgs, homeDir string, stdin io.Reader) error {
	timeout := args.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	type funcResult struct {
		err error
	}

	done := make(chan funcResult, 1)

	go func() {
		done <- funcResult{doExtractSession(args, homeDir, stdin)}
	}()

	select {
	case r := <-done:
		return r.err
	case <-time.After(timeout):
		fmt.Fprintln(os.Stderr, "Warning: extract-session timed out after 60s, skipping")
		return nil
	}
}

func doExtractSession(args ExtractSessionArgs, homeDir string, stdin io.Reader) error {
	start := time.Now()
	logPath := filepath.Join(homeDir, ".claude", "memory", "extract-session.log")
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

	project := args.Project
	transcriptPath := args.TranscriptPath

	hookInput, _ := ParseHookInput(stdin)

	dbg("stdin parsed")

	if hookInput != nil {
		if project == "" {
			project = DeriveProjectName(hookInput.Cwd)
		}

		if transcriptPath == "" && hookInput.TranscriptPath != "" {
			transcriptPath = hookInput.TranscriptPath
		}
	}

	if transcriptPath == "" {
		fmt.Fprintln(os.Stderr, "Warning: no transcript path provided, skipping extraction")
		return nil
	}

	if _, err := os.Stat(transcriptPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Warning: transcript file not found (%s), skipping extraction\n", transcriptPath)
		return nil
	}

	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		memoryRoot = filepath.Join(homeDir, ".claude", "memory")
	}

	ext := NewLLMExtractor()

	dbg("LLM extractor ready")

	if ext == nil {
		return errors.New("LLM extractor unavailable (keychain auth failed); cannot extract session without enrichment")
	}

	directExt, ok := ext.(*DirectAPIExtractor)
	if !ok {
		return errors.New("batch extraction requires DirectAPIExtractor")
	}

	sessionID := filepath.Base(transcriptPath)
	sessionID = strings.TrimSuffix(sessionID, ".jsonl")

	offsetDir := filepath.Join(memoryRoot, "offsets")
	offsetFile := filepath.Join(offsetDir, sessionID+".offset")

	var startOffset int64

	if data, err := os.ReadFile(offsetFile); err == nil {
		if v, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); err == nil {
			startOffset = v
		}
	}

	dbg(fmt.Sprintf("start offset: %d", startOffset))

	result, err := BatchExtractSession(context.Background(), transcriptPath, directExt, startOffset, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "extraction failed: %v\n", err)
		return fmt.Errorf("extraction failed: %w", err)
	}

	dbg(fmt.Sprintf("extraction done: %d principles, endOffset=%d", len(result.Principles), result.EndOffset))

	_ = os.MkdirAll(offsetDir, 0755)
	_ = os.WriteFile(offsetFile, []byte(strconv.FormatInt(result.EndOffset, 10)), 0644)

	learnings := make([]LearningItem, 0, len(result.Principles))

	for _, p := range result.Principles {
		learnErr := Learn(LearnOpts{
			Message:    p.Principle,
			Project:    project,
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
			fmt.Fprintf(os.Stderr, "Warning: failed to store principle: %v\n", learnErr)
			continue
		}

		learnings = append(learnings, LearningItem{
			Type:    p.Category,
			Content: p.Principle,
		})
	}

	maintenanceDB, dbErr := InitEmbeddingsDB(memoryRoot)
	if dbErr == nil {
		pruned, decayed, maintErr := AutoMaintenance(maintenanceDB)
		_ = maintenanceDB.Close()

		if maintErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: auto-maintenance failed: %v\n", maintErr)
		} else if pruned > 0 || decayed > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "Memory maintenance: pruned %d, decayed %d\n", pruned, decayed)
		}
	}

	summary := SessionSummary{
		SessionID:   filepath.Base(transcriptPath),
		ExtractedAt: time.Now(),
		Learnings:   learnings,
	}
	PrintSessionSummary(summary, os.Stdout)

	return nil
}
