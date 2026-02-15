package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
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

// ExtractSession extracts learnings from a Claude Code session transcript.
// It implements a multi-tier extraction approach:
// - Tier A (confidence 1.0): explicit signals like "remember this", corrections, CLAUDE.md edits
// - Tier B (confidence 0.7): inferred patterns like error→fix sequences, repeated patterns
func ExtractSession(opts ExtractSessionOpts) (*ExtractSessionResult, error) {
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

	// Read JSONL line by line — use 1MB buffer since transcript lines
	// can exceed the default 64KB (large tool results, file reads, etc.)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	var messages []map[string]any

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// Skip malformed lines
			continue
		}

		messages = append(messages, msg)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read transcript: %w", err)
	}

	// Parse transcript into normalized blocks
	blocks := parseTranscriptMessages(messages)

	// Extract items using multi-tier approach
	items := extractTierA(blocks, messages) // pass raw messages for file-history-snapshot
	items = append(items, extractTierB(blocks)...)
	items = append(items, extractTierC(blocks, opts.Matcher)...)

	// Store extracted items using existing memory functions
	for _, item := range items {
		if err := Learn(LearnOpts{
			Message:    item.Content,
			MemoryRoot: opts.MemoryRoot,
			Project:    opts.Project,
			Extractor:  opts.Extractor,
		}); err != nil {
			// Continue on error but mark as partial
			result.Status = "partial"
		}

		// Detect recurrence for corrections (Task 4: self-reinforcing learning)
		if item.Type == "correction" {
			if err := detectCorrectionRecurrence(item.Content, opts.MemoryRoot); err != nil {
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

// extractTierA extracts high-confidence items (explicit signals).
// It uses parsed blocks for text-based detection and raw messages for file-history-snapshot.
func extractTierA(blocks []parsedBlock, rawMessages []map[string]any) []SessionExtractedItem {
	var items []SessionExtractedItem

	// Detect "remember this" phrase in user text blocks
	for _, block := range blocks {
		if block.role == "user" && block.blockType == "text" &&
			strings.Contains(strings.ToLower(block.text), "remember this") {
			items = append(items, SessionExtractedItem{
				Type:       "explicit-learning",
				Content:    block.text,
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

	// Detect CLAUDE.md edits from raw messages (file-history-snapshot is top-level, not in blocks)
	for _, msg := range rawMessages {
		msgType, _ := msg["type"].(string)
		if msgType == "file-history-snapshot" {
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
						Content:    "CLAUDE.md was edited",
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
	for i := 0; i < len(blocks); i++ {
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

	// Detect repeated patterns in assistant text blocks
	patternCounts := make(map[string]int)
	for _, block := range blocks {
		if block.role != "assistant" || block.blockType != "text" {
			continue
		}
		words := strings.Fields(block.text)
		for i := 0; i < len(words)-3; i++ {
			phrase := strings.Join(words[i:i+4], " ")
			if len(phrase) > 20 {
				patternCounts[phrase]++
			}
		}
	}

	for pattern, count := range patternCounts {
		if count >= 3 {
			items = append(items, SessionExtractedItem{
				Type:       "repeated-pattern",
				Content:    pattern,
				Confidence: 0.7,
			})
		}
	}

	return items
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

// SemanticMatcher finds memories semantically similar to a given text.
// Used by Tier C behavioral convention detection.
type SemanticMatcher interface {
	FindSimilarMemories(text string, threshold float64, limit int) ([]string, error)
}

// extractTierC extracts low-confidence implicit signal items (confidence 0.5).
// It detects 5 types of implicit signals:
// - 3a: Tool usage patterns (command used 3+ times with >50% success)
// - 3b: Positive outcomes (PASS, "ok ", "0 errors", etc.)
// - 3c: Behavioral consistency (tool/library mentioned 5+ times without correction)
// - 3d: Self-corrected failures (error → fix with NO user intervention)
// - 3e: Behavioral conventions (assistant text semantically matches existing memories)
func extractTierC(blocks []parsedBlock, matcher SemanticMatcher) []SessionExtractedItem {
	var items []SessionExtractedItem

	items = append(items, extractToolUsagePatterns(blocks)...)
	items = append(items, extractPositiveOutcomes(blocks)...)
	items = append(items, extractBehavioralConsistency(blocks)...)
	items = append(items, extractSelfCorrectedFailures(blocks)...)
	if matcher != nil {
		items = append(items, extractBehavioralConventions(blocks, matcher)...)
	} else {
		fmt.Fprintf(os.Stderr, "SemanticMatcher not configured, skipping behavioral convention detection\n")
	}

	return items
}

// extractToolUsagePatterns (3a) detects Bash commands used 3+ times with >50% success.
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
			items = append(items, SessionExtractedItem{
				Type:       "tool-usage-pattern",
				Content:    fmt.Sprintf("used %s successfully in session", cmd),
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

// extractPositiveOutcomes (3b) detects strong success signals in tool results.
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
		cmd := ""
		for j := i - 1; j >= 0; j-- {
			if blocks[j].blockType == "tool_use" {
				if blocks[j].toolID == block.toolID || block.toolID == "" {
					cmd = extractCommandPrefix(blocks[j].toolInput)
					if cmd == "" {
						cmd = blocks[j].toolName
					}
				}
				break
			}
		}
		if cmd == "" {
			cmd = "command"
		}

		if seen[cmd] {
			continue
		}
		seen[cmd] = true

		items = append(items, SessionExtractedItem{
			Type:       "positive-outcome",
			Content:    fmt.Sprintf("tests passed using %s", cmd),
			Confidence: 0.5,
		})
	}
	return items
}

// containsPositiveSignal checks if text contains strong success indicators.
func containsPositiveSignal(text string) bool {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
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

// extractBehavioralConsistency (3c) detects tool/library names mentioned
// in 5+ distinct assistant messages without user correction.
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
				Content:    fmt.Sprintf("consistently used %s throughout session", name),
				Confidence: 0.5,
			})
		}
	}
	return items
}

// extractSelfCorrectedFailures (3d) detects error→fix sequences with NO user intervention.
// Based on SCoRe's operational definition: no user text block between error and fix.
func extractSelfCorrectedFailures(blocks []parsedBlock) []SessionExtractedItem {
	var items []SessionExtractedItem

	for i := 0; i < len(blocks); i++ {
		// Find error tool_result
		if blocks[i].blockType != "tool_result" {
			continue
		}
		if !blocks[i].isError && !containsErrorSignal(blocks[i].text) {
			continue
		}

		errorToolName := ""
		// Find the tool_use that produced this error
		for j := i - 1; j >= 0; j-- {
			if blocks[j].blockType == "tool_use" {
				if blocks[j].toolID == blocks[i].toolID || blocks[i].toolID == "" {
					errorToolName = blocks[j].toolName
				}
				break
			}
		}

		// Look for a subsequent successful tool_result WITHOUT user text in between
		hasUserText := false
		for j := i + 1; j < len(blocks); j++ {
			if blocks[j].role == "user" && blocks[j].blockType == "text" {
				hasUserText = true
				break // User intervened — this is Tier B, not self-corrected
			}
			if blocks[j].blockType == "tool_result" && !blocks[j].isError && !containsErrorSignal(blocks[j].text) {
				// Check it's the same tool type if we know it
				if errorToolName != "" {
					// Find the tool_use for this result
					for k := j - 1; k > i; k-- {
						if blocks[k].blockType == "tool_use" && blocks[k].toolName == errorToolName {
							if !hasUserText {
								desc := fmt.Sprintf("autonomously fixed %s error", errorToolName)
								items = append(items, SessionExtractedItem{
									Type:       "self-corrected-failure",
									Content:    desc,
									Confidence: 0.5,
								})
							}
							break
						}
					}
				} else if !hasUserText {
					items = append(items, SessionExtractedItem{
						Type:       "self-corrected-failure",
						Content:    "autonomously fixed error",
						Confidence: 0.5,
					})
				}
				break
			}
		}
	}
	return items
}

// extractBehavioralConventions (3e) detects assistant text that semantically matches
// existing memories. Uses the memory corpus as the convention library.
func extractBehavioralConventions(blocks []parsedBlock, matcher SemanticMatcher) []SessionExtractedItem {
	// Track which memories are matched by distinct assistant text blocks
	type memoryMatch struct {
		blockIndices []int
	}
	matches := make(map[string]*memoryMatch)

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

		// Find similar memories
		memories, err := matcher.FindSimilarMemories(block.text, 0.7, 3)
		if err != nil || len(memories) == 0 {
			continue
		}

		for _, mem := range memories {
			if matches[mem] == nil {
				matches[mem] = &memoryMatch{}
			}
			matches[mem].blockIndices = append(matches[mem].blockIndices, i)
		}
	}

	// Emit for memories matched by 3+ distinct blocks
	var items []SessionExtractedItem
	seen := make(map[string]bool) // deduplicate
	for mem, m := range matches {
		if len(m.blockIndices) >= 3 && !seen[mem] {
			seen[mem] = true
			items = append(items, SessionExtractedItem{
				Type:       "behavioral-convention",
				Content:    fmt.Sprintf("session behavior aligns with: %s", truncateForItem(mem, 100)),
				Confidence: 0.5,
			})
		}
	}
	return items
}

// truncateForItem truncates text for inclusion in an extracted item.
func truncateForItem(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}

// detectCorrectionRecurrence checks if a similar correction already exists in the embeddings database.
// If so, it logs a recurrence event to the changelog.
func detectCorrectionRecurrence(correctionContent, memoryRoot string) error {
	// Query for similar prior corrections using hybrid search
	results, err := Query(QueryOpts{
		Text:       correctionContent,
		Limit:      5,
		MemoryRoot: memoryRoot,
	})
	if err != nil {
		return fmt.Errorf("failed to query for similar corrections: %w", err)
	}

	// Check if any prior correction is similar (cosine similarity > 0.8)
	for _, result := range results.Results {
		if result.Score > 0.8 {
			// Found a recurrent correction
			// Log to changelog
			entry := ChangelogEntry{
				Action:         "correction_recurrence",
				SourceTier:     "embeddings",
				DestinationTier: "embeddings",
				ContentSummary: truncateForItem(correctionContent, 100),
				Reason:         "same correction detected (prior similarity: " + fmt.Sprintf("%.2f", result.Score) + ")",
			}
			if err := WriteChangelogEntry(memoryRoot, entry); err != nil {
				return fmt.Errorf("failed to write changelog entry: %w", err)
			}

			// Only log once per correction (first match is sufficient)
			break
		}
	}

	return nil
}
