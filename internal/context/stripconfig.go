package context

import (
	"encoding/json"
	"fmt"
	"strings"
)

// StripConfig controls behavior of StripWithConfig.
type StripConfig struct {
	// KeepToolCalls, when true, preserves tool_use and tool_result blocks (SBIA mode).
	// When false (default), behaves exactly like Strip() and drops all tool blocks.
	KeepToolCalls bool

	// ToolArgsTruncate is the max length for serialized tool arguments.
	// Only applies when KeepToolCalls is true. Zero means no truncation.
	ToolArgsTruncate int

	// ToolResultTruncate is the max length for tool result content.
	// Only applies when KeepToolCalls is true. Zero means no truncation.
	ToolResultTruncate int
}

// StripWithConfig parses JSONL transcript lines like Strip(), with optional tool call preservation.
//
// When cfg.KeepToolCalls is false, output is identical to Strip().
// When cfg.KeepToolCalls is true (SBIA mode), tool_use blocks are formatted as:
//
//	TOOL_USE [ToolName]: {args}
//
// and tool_result blocks are formatted as:
//
//	TOOL_RESULT [ok|error]: content
//
// A single JSONL line may produce multiple output lines (text + tool calls).
func StripWithConfig(lines []string, cfg StripConfig) []string {
	if !cfg.KeepToolCalls {
		return Strip(lines)
	}

	result := make([]string, 0, len(lines))

	for _, line := range lines {
		if !isKeptType(line) {
			continue
		}

		cleaned := replaceBase64(line)

		extracted := extractTextWithTools(cleaned, cfg)
		result = append(result, extracted...)
	}

	return result
}

// rawBlockType is a minimal block used to detect block type before full unmarshaling.
type rawBlockType struct {
	Type string `json:"type"`
}

// toolResultBlock is the full representation of a tool_result content block.
// JSON tags match Claude API snake_case field names.
//
//nolint:tagliatelle // Claude API uses snake_case: tool_use_id, is_error
type toolResultBlock struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error"`
}

// toolUseBlock is the full representation of a tool_use content block.
type toolUseBlock struct {
	Type  string          `json:"type"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// extractBlock extracts output lines from a single raw content block.
// Always returns a non-nil slice.
func extractBlock(raw json.RawMessage, rolePrefix string, cfg StripConfig) []string {
	var blockType rawBlockType

	err := json.Unmarshal(raw, &blockType)
	if err != nil {
		return make([]string, 0)
	}

	switch blockType.Type {
	case "text":
		return extractTextBlock(raw, rolePrefix)
	case "tool_use":
		return extractToolUseBlock(raw, cfg)
	case "tool_result":
		return extractToolResultBlock(raw, cfg)
	default:
		return make([]string, 0)
	}
}

// extractTextBlock extracts a text content block into a "ROLE: text" line.
// Always returns a non-nil slice.
func extractTextBlock(raw json.RawMessage, rolePrefix string) []string {
	var block contentBlock

	err := json.Unmarshal(raw, &block)
	if err != nil {
		return make([]string, 0)
	}

	if isSystemReminder(block.Text) {
		return make([]string, 0)
	}

	trimmed := strings.TrimSpace(block.Text)
	if trimmed == "" {
		return make([]string, 0)
	}

	return []string{truncateContent(rolePrefix + trimmed)}
}

// extractTextWithTools parses a JSONL line and returns one or more output lines.
// Text blocks become "ROLE: text", tool_use becomes "TOOL_USE [Name]: args",
// tool_result becomes "TOOL_RESULT [ok|error]: content".
// Always returns a non-nil slice.
func extractTextWithTools(line string, cfg StripConfig) []string {
	result := make([]string, 0)

	var entry jsonlLine

	err := json.Unmarshal([]byte(line), &entry)
	if err != nil {
		return result
	}

	role := normalizeRole(entry)
	if role == "" {
		return result
	}

	rolePrefix := "USER: "
	if role == roleAssistant {
		rolePrefix = "ASSISTANT: "
	}

	raw := entry.Message.Content
	if len(raw) == 0 {
		return result
	}

	// Try plain string content first.
	var str string

	strErr := json.Unmarshal(raw, &str)
	if strErr == nil {
		if isSystemReminder(str) {
			return result
		}

		return []string{truncateContent(rolePrefix + str)}
	}

	// Try array of blocks.
	var rawBlocks []json.RawMessage

	blocksErr := json.Unmarshal(raw, &rawBlocks)
	if blocksErr != nil {
		return result
	}

	result = make([]string, 0, len(rawBlocks))

	for _, block := range rawBlocks {
		lines := extractBlock(block, rolePrefix, cfg)
		result = append(result, lines...)
	}

	return result
}

// extractToolResultBlock extracts a tool_result block into a "TOOL_RESULT [ok|error]: content" line.
// Always returns a non-nil slice.
func extractToolResultBlock(raw json.RawMessage, cfg StripConfig) []string {
	var block toolResultBlock

	err := json.Unmarshal(raw, &block)
	if err != nil {
		return make([]string, 0)
	}

	status := "ok"
	if block.IsError {
		status = "error"
	}

	content := block.Content
	if cfg.ToolResultTruncate > 0 && len(content) > cfg.ToolResultTruncate {
		content = content[:cfg.ToolResultTruncate] + truncatedPlaceholder
	}

	return []string{fmt.Sprintf("TOOL_RESULT [%s]: %s", status, content)}
}

// extractToolUseBlock extracts a tool_use block into a "TOOL_USE [Name]: args" line.
// Always returns a non-nil slice.
func extractToolUseBlock(raw json.RawMessage, cfg StripConfig) []string {
	var block toolUseBlock

	err := json.Unmarshal(raw, &block)
	if err != nil {
		return make([]string, 0)
	}

	args := string(block.Input)
	if cfg.ToolArgsTruncate > 0 && len(args) > cfg.ToolArgsTruncate {
		args = args[:cfg.ToolArgsTruncate] + truncatedPlaceholder
	}

	return []string{fmt.Sprintf("TOOL_USE [%s]: %s", block.Name, args)}
}
