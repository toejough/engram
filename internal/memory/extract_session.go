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

	// Read JSONL line by line
	scanner := bufio.NewScanner(file)
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

	// Extract items using multi-tier approach
	items := extractTierA(messages)
	items = append(items, extractTierB(messages)...)

	// Store extracted items using existing memory functions
	for _, item := range items {
		if err := Learn(LearnOpts{
			Message:    item.Content,
			MemoryRoot: opts.MemoryRoot,
		}); err != nil {
			// Continue on error but mark as partial
			result.Status = "partial"
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
func extractTierA(messages []map[string]any) []SessionExtractedItem {
	var items []SessionExtractedItem

	for i, msg := range messages {
		msgType, _ := msg["type"].(string)
		content, _ := msg["content"].(string)

		// Detect "remember this" phrase
		if msgType == "user-message" && strings.Contains(strings.ToLower(content), "remember this") {
			items = append(items, SessionExtractedItem{
				Type:       "explicit-learning",
				Content:    content,
				Confidence: 1.0,
			})
		}

		// Detect explicit corrections (user negating assistant statement)
		if msgType == "user-message" && i > 0 {
			prevMsg := messages[i-1]
			prevType, _ := prevMsg["type"].(string)

			if prevType == "assistant-message" {
				// Look for correction patterns: "No", "Never", etc.
				lower := strings.ToLower(content)
				if strings.HasPrefix(lower, "no,") ||
					strings.Contains(lower, "never use") ||
					strings.Contains(lower, "don't use") {
					items = append(items, SessionExtractedItem{
						Type:       "correction",
						Content:    content,
						Confidence: 1.0,
					})
				}
			}
		}

		// Detect CLAUDE.md edits
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
func extractTierB(messages []map[string]any) []SessionExtractedItem {
	var items []SessionExtractedItem

	// Detect error→fix sequences
	for i := 0; i < len(messages)-2; i++ {
		msg1 := messages[i]
		msg2 := messages[i+1]
		msg3 := messages[i+2]

		type1, _ := msg1["type"].(string)
		type2, _ := msg2["type"].(string)
		type3, _ := msg3["type"].(string)

		content1, _ := msg1["content"].(string)
		content2, _ := msg2["content"].(string)
		content3, _ := msg3["content"].(string)

		// Pattern: tool-result error → assistant response → tool-result success
		if type1 == "tool-result" && strings.Contains(strings.ToLower(content1), "error") &&
			type2 == "assistant-message" &&
			type3 == "tool-result" && strings.Contains(strings.ToLower(content3), "success") {
			items = append(items, SessionExtractedItem{
				Type:       "error-fix",
				Content:    content2,
				Confidence: 0.7,
			})
		}
	}

	// Detect repeated patterns
	patternCounts := make(map[string]int)
	for _, msg := range messages {
		msgType, _ := msg["type"].(string)
		if msgType != "assistant-message" {
			continue
		}

		content, _ := msg["content"].(string)
		// Look for phrases longer than 20 chars
		words := strings.Fields(content)
		for i := 0; i < len(words)-3; i++ {
			phrase := strings.Join(words[i:i+4], " ")
			if len(phrase) > 20 {
				patternCounts[phrase]++
			}
		}
	}

	// Add patterns that appear 3+ times
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
