package memory

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExtractSessionOpts contains the options for session-based extraction.
type ExtractSessionOpts struct {
	// TranscriptPath is the path to the JSONL transcript file
	TranscriptPath string

	// MemoryRoot is the root directory for memory storage
	MemoryRoot string

	// Project is the project name for tagging extracted learnings
	Project string

	// Matcher is an optional semantic matcher for behavioral convention detection (Tier C.3e).
	// If nil, behavioral convention detection is skipped.
	Matcher SemanticMatcher

	// Extractor is an optional LLM extractor for enriching stored learnings.
	// If nil, learnings are stored without LLM enrichment.
	Extractor LLMExtractor

	// StartOffset is the byte offset to start reading from in the transcript file.
	// Used for incremental extraction — skip already-processed content.
	StartOffset int64

	// LogWriter is an optional writer for debug logging. If nil, logs are written
	// to ~/.claude/memory/extract-session.log (best-effort; errors are silently ignored).
	LogWriter io.Writer
}

// ExtractSessionResult contains the results of a session extraction operation.
type ExtractSessionResult struct {
	// Status indicates the result of the extraction. Values: "success", "error", "partial"
	Status string

	// ItemsExtracted is the count of items successfully extracted
	ItemsExtracted int

	// Items contains the individual extracted items
	Items []SessionExtractedItem

	// ConfidenceDistribution maps confidence levels to counts
	ConfidenceDistribution map[float64]int

	// EndOffset is the byte offset after the last line read.
	// Pass this as StartOffset on the next call for incremental extraction.
	EndOffset int64
}

// SemanticMatcher finds memories semantically similar to a given text.
// Used by Tier C behavioral convention detection.
type SemanticMatcher interface {
	FindSimilarMemories(text string, threshold float64, limit int) ([]string, error)
	// FindSimilarMemoriesBatch queries multiple texts in a single batch, sharing DB/model setup.
	// Returns a slice parallel to texts where each entry is the matching memories (or nil).
	FindSimilarMemoriesBatch(texts []string, threshold float64, limit int) ([][]string, error)
}

// SessionExtractedItem represents a single item extracted from a session transcript.
type SessionExtractedItem struct {
	// Type indicates the kind of item (e.g., "correction", "repeated-pattern", "error-fix", "claude-md-edit")
	Type string

	// Content is the actual content of the extracted item
	Content string

	// Confidence is the confidence score (1.0 for Tier A, 0.7 for Tier B)
	Confidence float64
}

// ExtractSession extracts learnings from a Claude Code session transcript.
// It implements a multi-tier extraction approach:
// - Tier A (confidence 1.0): explicit signals like "remember this", corrections, CLAUDE.md edits
// - Tier B (confidence 0.7): inferred patterns like error→fix sequences, repeated patterns
func ExtractSession(opts ExtractSessionOpts) (*ExtractSessionResult, error) {
	esStart := time.Now()

	// Resolve log writer: use injected writer or open the default log file.
	logW := opts.LogWriter
	if logW == nil {
		if f, err := openExtractLog(); err == nil {
			defer func() { _ = f.Close() }()

			logW = f
		}
	}

	extractLogf(logW, "start: %s", filepath.Base(opts.TranscriptPath))

	// Open and read the transcript file
	file, err := os.Open(opts.TranscriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open transcript: %w", err)
	}

	defer func() { _ = file.Close() }()

	result := &ExtractSessionResult{
		Status:                 "success",
		Items:                  []SessionExtractedItem{},
		ConfidenceDistribution: make(map[float64]int),
	}

	// Seek to start offset for incremental extraction
	if opts.StartOffset > 0 {
		if _, err := file.Seek(opts.StartOffset, 0); err != nil {
			return nil, fmt.Errorf("failed to seek to offset %d: %w", opts.StartOffset, err)
		}

		extractLogf(logW, "seeking to offset %d", opts.StartOffset)
	}

	// Read JSONL line by line — use 1MB buffer since transcript lines
	// can exceed the default 64KB (large tool results, file reads, etc.)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	var messages []map[string]any

	bytesRead := opts.StartOffset

	for scanner.Scan() {
		line := scanner.Text()
		bytesRead += int64(len(scanner.Bytes())) + 1 // +1 for newline

		if line == "" {
			continue
		}

		var msg map[string]any

		err := json.Unmarshal([]byte(line), &msg)
		if err != nil {
			// Skip malformed lines
			continue
		}

		messages = append(messages, msg)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read transcript: %w", err)
	}

	result.EndOffset = bytesRead
	extractLogf(logW, "parsed %d new messages from offset %d to %d (+%dms)", len(messages), opts.StartOffset, bytesRead, time.Since(esStart).Milliseconds())

	// Nothing new to process
	if len(messages) == 0 {
		return result, nil
	}

	// Parse transcript into normalized blocks
	blocks := parseTranscriptMessages(messages)
	extractLogf(logW, "normalized to %d blocks (+%dms)", len(blocks), time.Since(esStart).Milliseconds())

	// Extract items using multi-tier approach
	items := extractTierA(blocks, messages) // pass raw messages for file-history-snapshot
	extractLogf(logW, "tierA: %d items (+%dms)", len(items), time.Since(esStart).Milliseconds())
	items = append(items, extractTierB(blocks)...)
	extractLogf(logW, "tierB: %d items total (+%dms)", len(items), time.Since(esStart).Milliseconds())
	items = append(items, extractTierC(blocks, opts.Matcher, logW)...)
	extractLogf(logW, "tierC: %d items total (+%dms)", len(items), time.Since(esStart).Milliseconds())

	// Cap items to avoid blowing the Stop hook timeout.
	// Items are already ordered by tier (A=1.0, B=0.7, C=0.5), so truncating
	// keeps the highest-confidence items.
	const maxItems = 10
	if len(items) > maxItems {
		items = items[:maxItems]
	}

	// Batch Extract: get all observations in a single API call (~2s instead of N×1.5s)
	var observations []*Observation

	if opts.Extractor != nil {
		if batcher, ok := opts.Extractor.(interface {
			ExtractBatch(ctx context.Context, contents []string) ([]*Observation, error)
		}); ok {
			contents := make([]string, len(items))
			for i, item := range items {
				contents[i] = item.Content
			}

			batchStart := time.Now()
			observations, err = batcher.ExtractBatch(context.Background(), contents)

			extractLogf(logW, "ExtractBatch: %d items (%dms)", len(items), time.Since(batchStart).Milliseconds())

			if err != nil {
				extractLogf(logW, "ExtractBatch FAIL: %v", err)

				observations = nil // fall back to per-item Extract via Learn()
			}
		}
	}

	// Store extracted items using existing memory functions
	for i, item := range items {
		learnStart := time.Now()
		learnOpts := LearnOpts{
			Message:    item.Content,
			MemoryRoot: opts.MemoryRoot,
			Project:    opts.Project,
			Extractor:  opts.Extractor,
		}
		// Use precomputed observation if batch succeeded
		if observations != nil && i < len(observations) && observations[i] != nil {
			learnOpts.PrecomputedObservation = observations[i]
		}

		err := Learn(learnOpts)
		if err != nil {
			// Continue on error but mark as partial
			result.Status = "partial"

			extractLogf(logW, "Learn[%d/%d] FAIL (%dms): %v", i+1, len(items), time.Since(learnStart).Milliseconds(), err)
		} else {
			extractLogf(logW, "Learn[%d/%d] ok (%dms)", i+1, len(items), time.Since(learnStart).Milliseconds())
		}

		// Detect recurrence for corrections (Task 4: self-reinforcing learning)
		if item.Type == "correction" {
			err := detectCorrectionRecurrence(item.Content, opts.MemoryRoot)
			if err != nil {
				// Log error but don't fail the extraction
				fmt.Fprintf(os.Stderr, "failed to detect correction recurrence: %v\n", err)
			}
		}
	}

	// Build result
	result.Items = items
	result.ItemsExtracted = len(items)

	for _, item := range items {
		result.ConfidenceDistribution[item.Confidence]++
	}

	return result, nil
}

// parsedBlock is a normalized representation of a transcript block.
// It flattens the nested Claude Code JSONL format into a uniform structure.
type parsedBlock struct {
	role      string         // "user" or "assistant"
	blockType string         // "text", "tool_use", "tool_result"
	text      string         // for text blocks
	toolName  string         // for tool_use (e.g. "Bash", "Read")
	toolInput map[string]any // for tool_use (e.g. {"command": "go test"})
	toolID    string         // links tool_use ↔ tool_result
	isError   bool           // for tool_result
}

// applyRecurrenceCheck logs a changelog entry if any result has Score > 0.8.
// Extracted to enable unit-testing without requiring the ONNX embedding model.
func applyRecurrenceCheck(results []QueryResult, correctionContent, memoryRoot string) error {
	for _, result := range results {
		if result.Score > 0.8 {
			entry := ChangelogEntry{
				Action:          "correction_recurrence",
				SourceTier:      "embeddings",
				DestinationTier: "embeddings",
				ContentSummary:  truncateForItem(correctionContent, 100),
				Reason:          "same correction detected (prior similarity: " + fmt.Sprintf("%.2f", result.Score) + ")",
			}

			if err := WriteChangelogEntry(memoryRoot, entry); err != nil {
				return fmt.Errorf("failed to write changelog entry: %w", err)
			}

			break
		}
	}

	return nil
}

// containsErrorSignal checks if text contains common error indicators.
func containsErrorSignal(text string) bool {
	lower := strings.ToLower(text)

	return strings.Contains(lower, "error") ||
		strings.Contains(lower, "fail") ||
		strings.Contains(lower, "exception") ||
		strings.Contains(lower, "panic") ||
		strings.Contains(lower, "exit status")
}

// containsPositiveSignal checks if text contains strong success indicators.
func containsPositiveSignal(text string) bool {
	lines := strings.SplitSeq(text, "\n")
	for line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "PASS") ||
			strings.HasPrefix(trimmed, "ok ") ||
			strings.Contains(trimmed, "0 errors") ||
			strings.Contains(trimmed, "Build succeeded") {
			return true
		}
	}

	return false
}

// detectCorrectionRecurrence checks if a similar correction already exists in the embeddings database.
// If so, it logs a recurrence event to the changelog.
func detectCorrectionRecurrence(correctionContent, memoryRoot string) error {
	results, err := Query(QueryOpts{
		Text:       correctionContent,
		Limit:      5,
		MemoryRoot: memoryRoot,
	})
	if err != nil {
		return fmt.Errorf("failed to query for similar corrections: %w", err)
	}

	return applyRecurrenceCheck(results.Results, correctionContent, memoryRoot)
}

// extractBehavioralConsistency (3c) detects tool/library names mentioned
// in 5+ distinct assistant messages without user correction.
// Collects example usage text for LLM reflection.
func extractBehavioralConsistency(blocks []parsedBlock) []SessionExtractedItem {
	// Track mentions per distinct assistant text block index
	type mentionTracker struct {
		blockIndices []int
		corrected    bool
	}

	mentions := make(map[string]*mentionTracker)

	// Known tool/library names to look for
	toolNames := []string{
		"gomega", "rapid", "targ", "cobra", "viper",
		"react", "vue", "angular", "express", "fastapi",
		"pytest", "jest", "vitest", "mocha",
		"docker", "kubernetes", "terraform",
		"sqlite", "postgres", "redis", "mongodb",
		"onnx", "pytorch", "tensorflow",
		"gin", "echo", "fiber", "chi",
		"tailwind", "bootstrap",
	}

	for i, block := range blocks {
		if block.role == "assistant" && block.blockType == "text" {
			lower := strings.ToLower(block.text)
			for _, name := range toolNames {
				if strings.Contains(lower, name) {
					if mentions[name] == nil {
						mentions[name] = &mentionTracker{}
					}

					mentions[name].blockIndices = append(mentions[name].blockIndices, i)
				}
			}
		}

		// Check for user corrections that would break consistency
		if block.role == "user" && block.blockType == "text" {
			lower := strings.ToLower(block.text)
			if strings.HasPrefix(lower, "no,") ||
				strings.Contains(lower, "don't use") ||
				strings.Contains(lower, "never use") ||
				strings.Contains(lower, "stop using") ||
				strings.Contains(lower, "instead of") {
				// Mark any tool mentioned in this correction as corrected
				for _, name := range toolNames {
					if strings.Contains(lower, name) && mentions[name] != nil {
						mentions[name].corrected = true
					}
				}
			}
		}
	}

	var items []SessionExtractedItem

	for name, tracker := range mentions {
		if len(tracker.blockIndices) >= 5 && !tracker.corrected {
			items = append(items, SessionExtractedItem{
				Type:       "behavioral-consistency",
				Content:    fmt.Sprintf("Prefer '%s' — used consistently (%d times) without correction", name, len(tracker.blockIndices)),
				Confidence: 0.5,
			})
		}
	}

	return items
}

// extractBehavioralConventions (3e) detects assistant text that semantically matches
// existing memories. Uses the memory corpus as the convention library.
// Uses batch FindSimilarMemories to avoid per-call DB/model overhead.
func extractBehavioralConventions(blocks []parsedBlock, matcher SemanticMatcher) []SessionExtractedItem {
	// Collect qualifying texts and their block indices
	type candidate struct {
		text     string
		blockIdx int
	}

	var candidates []candidate

	for i, block := range blocks {
		if block.role != "assistant" || block.blockType != "text" {
			continue
		}

		if len(block.text) <= 50 {
			continue
		}

		// Check for user correction following this block
		correctionFollows := false

		if i+1 < len(blocks) && blocks[i+1].role == "user" && blocks[i+1].blockType == "text" {
			lower := strings.ToLower(blocks[i+1].text)
			if strings.HasPrefix(lower, "no,") ||
				strings.Contains(lower, "don't") ||
				strings.Contains(lower, "never") ||
				strings.Contains(lower, "wrong") ||
				strings.Contains(lower, "stop") {
				correctionFollows = true
			}
		}

		if correctionFollows {
			continue
		}

		candidates = append(candidates, candidate{text: block.text, blockIdx: i})
	}

	if len(candidates) == 0 {
		return nil
	}

	// Batch query all candidate texts at once (single DB open/close)
	texts := make([]string, len(candidates))
	for i, c := range candidates {
		texts[i] = c.text
	}

	batchResults, err := matcher.FindSimilarMemoriesBatch(texts, 0.7, 3)
	if err != nil {
		return nil
	}

	// Track which memories are matched by distinct assistant text blocks
	type memoryMatch struct {
		blockIndices []int
	}

	matches := make(map[string]*memoryMatch)

	for i, memories := range batchResults {
		for _, mem := range memories {
			if matches[mem] == nil {
				matches[mem] = &memoryMatch{}
			}

			matches[mem].blockIndices = append(matches[mem].blockIndices, candidates[i].blockIdx)
		}
	}

	// Emit for memories matched by 3+ distinct blocks
	var items []SessionExtractedItem

	seen := make(map[string]bool) // deduplicate
	for mem, m := range matches {
		if len(m.blockIndices) >= 3 && !seen[mem] {
			seen[mem] = true
			// Collect example behavior snippets
			var examples []string
			for _, idx := range m.blockIndices {
				if idx < len(blocks) && len(examples) < 3 {
					examples = append(examples, truncateForItem(blocks[idx].text, 100))
				}
			}

			content := fmt.Sprintf(
				"Session behavior aligns with existing memory: %q\nMatching behavior in %d assistant messages:\n  - %s",
				truncateForItem(mem, 100), len(m.blockIndices),
				strings.Join(examples, "\n  - "))
			items = append(items, SessionExtractedItem{
				Type:       "behavioral-convention",
				Content:    content,
				Confidence: 0.5,
			})
		}
	}

	return items
}

// extractCommandPrefix extracts a normalized command prefix from tool input.
// Returns first word, or first two words for compound commands like "go test", "git commit".
func extractCommandPrefix(input map[string]any) string {
	if input == nil {
		return ""
	}

	command, _ := input["command"].(string)
	if command == "" {
		return ""
	}

	words := strings.Fields(command)
	if len(words) == 0 {
		return ""
	}

	// Compound command prefixes
	compoundPrefixes := map[string]bool{
		"go": true, "git": true, "npm": true, "yarn": true,
		"docker": true, "kubectl": true, "make": true, "cargo": true,
	}

	if len(words) >= 2 && compoundPrefixes[words[0]] {
		return words[0] + " " + words[1]
	}

	return words[0]
}

// extractFullCommand returns the complete command string from tool input.
func extractFullCommand(input map[string]any) string {
	if input == nil {
		return ""
	}

	command, _ := input["command"].(string)

	return command
}

// extractLogf writes a timestamped debug line to w.
// If w is nil the call is a no-op.
func extractLogf(w io.Writer, format string, args ...any) {
	if w == nil {
		return
	}

	_, _ = fmt.Fprintf(w, "%s [ExtractSession] %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
}

// extractPositiveOutcomes (3b) detects strong success signals in tool results.
// Collects actual command and output for LLM reflection.
func extractPositiveOutcomes(blocks []parsedBlock) []SessionExtractedItem {
	seen := make(map[string]bool) // deduplicate by command type

	var items []SessionExtractedItem

	for i, block := range blocks {
		if block.blockType != "tool_result" {
			continue
		}

		if block.isError || containsErrorSignal(block.text) {
			continue
		}

		if !containsPositiveSignal(block.text) {
			continue
		}

		// Find the preceding tool_use to get the command
		cmdPrefix := ""

		for j := i - 1; j >= 0; j-- {
			if blocks[j].blockType == "tool_use" {
				if blocks[j].toolID == block.toolID || block.toolID == "" {
					cmdPrefix = extractCommandPrefix(blocks[j].toolInput)
					if cmdPrefix == "" {
						cmdPrefix = blocks[j].toolName
					}
				}

				break
			}
		}

		if cmdPrefix == "" {
			cmdPrefix = "command"
		}

		if seen[cmdPrefix] {
			continue
		}

		seen[cmdPrefix] = true

		items = append(items, SessionExtractedItem{
			Type:       "positive-outcome",
			Content:    fmt.Sprintf("Use '%s' — produces successful results", cmdPrefix),
			Confidence: 0.5,
		})
	}

	return items
}

// extractSelfCorrectedFailures (3d) detects error→fix sequences with NO user intervention.
// Based on SCoRe's operational definition: no user text block between error and fix.
// Collects full error/fix context for LLM reflection (Reflexion pattern).
func extractSelfCorrectedFailures(blocks []parsedBlock) []SessionExtractedItem {
	var items []SessionExtractedItem

	for i := range blocks {
		// Find error tool_result
		if blocks[i].blockType != "tool_result" {
			continue
		}

		if !blocks[i].isError && !containsErrorSignal(blocks[i].text) {
			continue
		}

		errorToolName := ""
		failedCommand := ""

		// Find the tool_use that produced this error
		for j := i - 1; j >= 0; j-- {
			if blocks[j].blockType == "tool_use" {
				if blocks[j].toolID == blocks[i].toolID || blocks[i].toolID == "" {
					errorToolName = blocks[j].toolName
					failedCommand = extractFullCommand(blocks[j].toolInput)
				}

				break
			}
		}

		// Look for a subsequent successful tool_result WITHOUT user text in between
		for j := i + 1; j < len(blocks); j++ {
			if blocks[j].role == "user" && blocks[j].blockType == "text" {
				break // User intervened — this is Tier B, not self-corrected
			}

			if blocks[j].blockType == "tool_result" && !blocks[j].isError && !containsErrorSignal(blocks[j].text) {
				// Check it's the same tool type if we know it
				if errorToolName != "" {
					// Find the tool_use for this result
					for k := j - 1; k > i; k-- {
						if blocks[k].blockType == "tool_use" && blocks[k].toolName == errorToolName {
							fixCommand := extractFullCommand(blocks[k].toolInput)
							items = append(items, SessionExtractedItem{
								Type:       "self-corrected-failure",
								Content:    fmt.Sprintf("When '%s' fails, fix: '%s'", failedCommand, fixCommand),
								Confidence: 0.5,
							})

							break
						}
					}
				} else if failedCommand != "" {
					items = append(items, SessionExtractedItem{
						Type:       "self-corrected-failure",
						Content:    fmt.Sprintf("'%s' may fail — self-corrected without user help", failedCommand),
						Confidence: 0.5,
					})
				}

				break
			}
		}
	}

	return items
}

// extractTierA extracts high-confidence items (explicit signals).
// It uses parsed blocks for text-based detection and raw messages for file-history-snapshot.
func extractTierA(blocks []parsedBlock, rawMessages []map[string]any) []SessionExtractedItem {
	var items []SessionExtractedItem

	// Detect "remember this" phrase in user text blocks.
	// Skip system-reminder content injected into user messages (skill files, hook output, etc.)
	for _, block := range blocks {
		if block.role != "user" || block.blockType != "text" {
			continue
		}

		text := block.text
		// Strip system-reminder tags and their content — these are injected by hooks/skills,
		// not typed by the user
		text = stripSystemReminders(text)

		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		if strings.Contains(strings.ToLower(text), "remember this") {
			items = append(items, SessionExtractedItem{
				Type:       "explicit-learning",
				Content:    text,
				Confidence: 1.0,
			})
		}
	}

	// Detect explicit corrections (user text block preceded by assistant text block)
	for i, block := range blocks {
		if block.role == "user" && block.blockType == "text" && i > 0 {
			prev := blocks[i-1]
			if prev.role == "assistant" && prev.blockType == "text" {
				lower := strings.ToLower(block.text)
				if strings.HasPrefix(lower, "no,") ||
					strings.Contains(lower, "never use") ||
					strings.Contains(lower, "don't use") {
					items = append(items, SessionExtractedItem{
						Type:       "correction",
						Content:    block.text,
						Confidence: 1.0,
					})
				}
			}
		}
	}

	// Detect CLAUDE.md edits: first scan blocks for actual Edit/Write operations,
	// then fall back to file-history-snapshot detection.
	var claudeEdits []string

	for _, block := range blocks {
		if block.blockType != "tool_use" {
			continue
		}

		if block.toolName != "Edit" && block.toolName != "Write" {
			continue
		}

		filePath, _ := block.toolInput["file_path"].(string)
		if !strings.Contains(filePath, "CLAUDE.md") {
			continue
		}

		if block.toolName == "Edit" {
			oldStr, _ := block.toolInput["old_string"].(string)
			newStr, _ := block.toolInput["new_string"].(string)
			claudeEdits = append(claudeEdits,
				fmt.Sprintf("Changed: %q → %q",
					truncateForItem(oldStr, 100), truncateForItem(newStr, 100)))
		} else {
			content, _ := block.toolInput["content"].(string)
			claudeEdits = append(claudeEdits,
				"Wrote: "+truncateForItem(content, 200))
		}
	}

	if len(claudeEdits) > 0 {
		items = append(items, SessionExtractedItem{
			Type:       "claude-md-edit",
			Content:    "CLAUDE.md modifications:\n" + strings.Join(claudeEdits, "\n"),
			Confidence: 1.0,
		})
	} else {
		// Fall back to file-history-snapshot detection
		for _, msg := range rawMessages {
			msgType, _ := msg["type"].(string)
			if msgType != "file-history-snapshot" {
				continue
			}

			snapshot, ok := msg["snapshot"].(map[string]any)
			if !ok {
				continue
			}

			trackedFiles, ok := snapshot["trackedFileBackups"].(map[string]any)
			if !ok {
				continue
			}

			for path := range trackedFiles {
				if strings.Contains(path, "CLAUDE.md") {
					items = append(items, SessionExtractedItem{
						Type:       "claude-md-edit",
						Content:    "CLAUDE.md was modified (changes not captured in transcript)",
						Confidence: 1.0,
					})

					break
				}
			}
		}
	}

	return items
}

// extractTierB extracts medium-confidence items (inferred patterns).
// Error→fix detection requires user intervention between error and fix.
func extractTierB(blocks []parsedBlock) []SessionExtractedItem {
	var items []SessionExtractedItem

	// Detect error→fix sequences WITH user intervention
	for i := range blocks {
		// Find error tool_result
		if blocks[i].blockType != "tool_result" || !blocks[i].isError {
			continue
		}

		if !containsErrorSignal(blocks[i].text) && !blocks[i].isError {
			continue
		}

		// Look for user text block followed by successful tool_result
		hasUserIntervention := false

		var userContent string

		for j := i + 1; j < len(blocks); j++ {
			if blocks[j].role == "user" && blocks[j].blockType == "text" {
				hasUserIntervention = true
				userContent = blocks[j].text
			}

			if blocks[j].blockType == "tool_result" && !blocks[j].isError && !containsErrorSignal(blocks[j].text) {
				if hasUserIntervention {
					items = append(items, SessionExtractedItem{
						Type:       "error-fix",
						Content:    userContent,
						Confidence: 0.7,
					})
				}

				break
			}
		}
	}

	return items
}

// extractTierC extracts low-confidence implicit signal items (confidence 0.5).
// It detects 5 types of implicit signals:
// - 3a: Tool usage patterns (command used 3+ times with >50% success)
// - 3b: Positive outcomes (PASS, "ok ", "0 errors", etc.)
// - 3c: Behavioral consistency (tool/library mentioned 5+ times without correction)
// - 3d: Self-corrected failures (error → fix with NO user intervention)
// - 3e: Behavioral conventions (assistant text semantically matches existing memories)
func extractTierC(blocks []parsedBlock, matcher SemanticMatcher, logW io.Writer) []SessionExtractedItem {
	var items []SessionExtractedItem

	tcStart := time.Now()

	items = append(items, extractToolUsagePatterns(blocks)...)
	extractLogf(logW, "  tierC.3a toolUsage: %d items (%dms)", len(items), time.Since(tcStart).Milliseconds())

	prevCount := len(items)
	items = append(items, extractPositiveOutcomes(blocks)...)
	extractLogf(logW, "  tierC.3b positiveOutcomes: %d items (%dms)", len(items)-prevCount, time.Since(tcStart).Milliseconds())

	prevCount = len(items)
	items = append(items, extractBehavioralConsistency(blocks)...)
	extractLogf(logW, "  tierC.3c consistency: %d items (%dms)", len(items)-prevCount, time.Since(tcStart).Milliseconds())

	prevCount = len(items)
	items = append(items, extractSelfCorrectedFailures(blocks)...)
	extractLogf(logW, "  tierC.3d selfCorrected: %d items (%dms)", len(items)-prevCount, time.Since(tcStart).Milliseconds())

	if matcher != nil {
		prevCount = len(items)
		items = append(items, extractBehavioralConventions(blocks, matcher)...)
		extractLogf(logW, "  tierC.3e conventions: %d items (%dms)", len(items)-prevCount, time.Since(tcStart).Milliseconds())
	} else {
		fmt.Fprintf(os.Stderr, "SemanticMatcher not configured, skipping behavioral convention detection\n")
	}

	return items
}

// extractToolUsagePatterns (3a) detects Bash commands used 3+ times with >50% success.
// Collects actual command strings for LLM reflection.
func extractToolUsagePatterns(blocks []parsedBlock) []SessionExtractedItem {
	type cmdStats struct {
		success int
		total   int
	}

	stats := make(map[string]*cmdStats)

	for i, block := range blocks {
		if block.blockType != "tool_use" || block.toolName != "Bash" {
			continue
		}

		cmd := extractCommandPrefix(block.toolInput)
		if cmd == "" {
			continue
		}

		if stats[cmd] == nil {
			stats[cmd] = &cmdStats{}
		}

		stats[cmd].total++

		// Check if the next tool_result is successful
		for j := i + 1; j < len(blocks); j++ {
			if blocks[j].blockType == "tool_result" && blocks[j].toolID == block.toolID {
				if !blocks[j].isError && !containsErrorSignal(blocks[j].text) {
					stats[cmd].success++
				}

				break
			}
			// Also match tool_results without toolID (legacy format) — take the next one
			if blocks[j].blockType == "tool_result" && blocks[j].toolID == "" {
				if !blocks[j].isError && !containsErrorSignal(blocks[j].text) {
					stats[cmd].success++
				}

				break
			}
		}
	}

	var items []SessionExtractedItem

	for cmd, s := range stats {
		if s.total >= 3 && float64(s.success)/float64(s.total) > 0.5 {
			// Produce a concise observation, not a raw dump
			items = append(items, SessionExtractedItem{
				Type:       "tool-usage-pattern",
				Content:    fmt.Sprintf("Prefer '%s' for this task (used %d times, %d%% success rate)", cmd, s.total, s.success*100/s.total),
				Confidence: 0.5,
			})
		}
	}

	return items
}

// openExtractLog opens the debug log file at ~/.claude/memory/extract-session.log.
func openExtractLog() (io.WriteCloser, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return os.OpenFile(filepath.Join(home, ".claude", "memory", "extract-session.log"),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
}

// parseTranscriptMessages converts raw JSONL messages into flat typed blocks.
// Handles both the new Claude Code format (type:"user"/"assistant" with nested content arrays)
// and the legacy format (type:"user-message"/"assistant-message"/"tool-result" with flat content).
func parseTranscriptMessages(messages []map[string]any) []parsedBlock {
	var blocks []parsedBlock

	for _, msg := range messages {
		msgType, _ := msg["type"].(string)

		switch msgType {
		case "user", "assistant":
			// New Claude Code format: {"type":"user","message":{"role":"...","content":[...]}}
			message, ok := msg["message"].(map[string]any)
			if !ok {
				continue
			}

			role, _ := message["role"].(string)

			contentArr, ok := message["content"].([]any)
			if !ok {
				continue
			}

			for _, item := range contentArr {
				block, ok := item.(map[string]any)
				if !ok {
					continue
				}

				blockType, _ := block["type"].(string)
				switch blockType {
				case "text":
					text, _ := block["text"].(string)
					blocks = append(blocks, parsedBlock{
						role:      role,
						blockType: "text",
						text:      text,
					})
				case "tool_use":
					id, _ := block["id"].(string)
					name, _ := block["name"].(string)
					input, _ := block["input"].(map[string]any)
					blocks = append(blocks, parsedBlock{
						role:      role,
						blockType: "tool_use",
						toolName:  name,
						toolInput: input,
						toolID:    id,
					})
				case "tool_result":
					toolUseID, _ := block["tool_use_id"].(string)
					content, _ := block["content"].(string)
					isError, _ := block["is_error"].(bool)
					blocks = append(blocks, parsedBlock{
						role:      role,
						blockType: "tool_result",
						text:      content,
						toolID:    toolUseID,
						isError:   isError,
					})
				}
			}

		// Legacy format support
		case "user-message":
			content, _ := msg["content"].(string)
			blocks = append(blocks, parsedBlock{
				role:      "user",
				blockType: "text",
				text:      content,
			})
		case "assistant-message":
			content, _ := msg["content"].(string)
			blocks = append(blocks, parsedBlock{
				role:      "assistant",
				blockType: "text",
				text:      content,
			})
		case "tool-result":
			content, _ := msg["content"].(string)
			isError := strings.Contains(strings.ToLower(content), "error")
			blocks = append(blocks, parsedBlock{
				role:      "user", // tool results come in user turn
				blockType: "tool_result",
				text:      content,
				isError:   isError,
			})
		}
	}

	return blocks
}

// truncateForItem truncates text for inclusion in an extracted item.
// stripSystemReminders removes <system-reminder>...</system-reminder> blocks from text.
// These are injected by hooks and skills into user messages and are not user-authored content.
func stripSystemReminders(text string) string {
	result := text
	for {
		start := strings.Index(result, "<system-reminder>")
		if start == -1 {
			break
		}

		end := strings.Index(result[start:], "</system-reminder>")
		if end == -1 {
			// Unclosed tag — strip from start to end of string
			result = result[:start]
			break
		}

		result = result[:start] + result[start+end+len("</system-reminder>"):]
	}

	return result
}

func truncateForItem(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	return text[:maxLen-3] + "..."
}
