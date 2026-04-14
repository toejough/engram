package context

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// unexported constants.
const (
	toolSummaryArgsCap   = 120
	toolSummaryOutputCap = 120
)

// toolSummaryPair holds a pending tool_use waiting for its matching tool_result.
type toolSummaryPair struct {
	name string
	args string
}

// extractSummaryBlocks parses a single JSONL line and returns output lines.
// Text blocks become role-prefixed lines. tool_use blocks are stored as pending.
// tool_result blocks are paired with pending tool_use to emit summary lines.
func extractSummaryBlocks(line string, pending map[string]toolSummaryPair) []string {
	var entry jsonlLine

	err := json.Unmarshal([]byte(line), &entry)
	if err != nil {
		return make([]string, 0)
	}

	role := normalizeRole(entry)
	if role == "" {
		return make([]string, 0)
	}

	rolePrefix := userPrefix
	if role == roleAssistant {
		rolePrefix = assistantPrefix
	}

	raw := entry.Message.Content
	if len(raw) == 0 {
		return make([]string, 0)
	}

	// Try plain string content first.
	var str string

	strErr := json.Unmarshal(raw, &str)
	if strErr == nil {
		trimmed := strings.TrimSpace(str)
		if trimmed == "" || isSystemReminder(trimmed) {
			return make([]string, 0)
		}

		return []string{truncateContent(rolePrefix + trimmed)}
	}

	// Try array of blocks.
	var rawBlocks []json.RawMessage

	blocksErr := json.Unmarshal(raw, &rawBlocks)
	if blocksErr != nil {
		return make([]string, 0)
	}

	return processSummaryContentBlocks(rawBlocks, rolePrefix, pending)
}

// firstNonEmptyLine returns the first non-blank line from content.
func firstNonEmptyLine(content string) string {
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}

// formatToolSummaryArgs formats a tool input JSON object as key=value pairs.
// String values are quoted, others are rendered as-is.
// The entire arg string is truncated at toolSummaryArgsCap characters.
func formatToolSummaryArgs(input json.RawMessage) string {
	var rawMap map[string]json.RawMessage

	err := json.Unmarshal(input, &rawMap)
	if err != nil {
		return ""
	}

	// Sort keys for deterministic output.
	keys := make([]string, 0, len(rawMap))
	for key := range rawMap {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	parts := make([]string, 0, len(keys))

	for _, key := range keys {
		val := rawMap[key]

		var strVal string

		strErr := json.Unmarshal(val, &strVal)
		if strErr == nil {
			parts = append(parts, fmt.Sprintf("%s=%q", key, strVal))
		} else {
			parts = append(parts, fmt.Sprintf("%s=%s", key, string(val)))
		}
	}

	joined := strings.Join(parts, ", ")
	if len(joined) > toolSummaryArgsCap {
		return joined[:toolSummaryArgsCap] + truncatedPlaceholder
	}

	return joined
}

// formatToolSummaryLine produces a compact tool summary line:
//
//	[tool] Name(args) → exit 0 | first line of output
func formatToolSummaryLine(name, args string, exitCode int, firstLine string) string {
	if len(firstLine) > toolSummaryOutputCap {
		firstLine = firstLine[:toolSummaryOutputCap] + truncatedPlaceholder
	}

	if firstLine == "" {
		return fmt.Sprintf("[tool] %s(%s) → exit %d", name, args, exitCode)
	}

	return fmt.Sprintf("[tool] %s(%s) → exit %d | %s", name, args, exitCode, firstLine)
}

// matchToolResult parses a tool_result block, pairs it with a pending tool_use,
// and returns the formatted summary line. Returns empty string if no match found.
func matchToolResult(block json.RawMessage, pending map[string]toolSummaryPair) string {
	var toolResult toolResultBlock

	err := json.Unmarshal(block, &toolResult)
	if err != nil {
		return ""
	}

	pair, found := pending[toolResult.ToolUseID]
	if !found {
		return ""
	}

	delete(pending, toolResult.ToolUseID)

	exitCode := 0
	if toolResult.IsError {
		exitCode = 1
	}

	firstLine := firstNonEmptyLine(toolResult.Content)

	return formatToolSummaryLine(pair.name, pair.args, exitCode, firstLine)
}

// processSummaryContentBlocks processes an array of content blocks for tool summary mode.
func processSummaryContentBlocks(
	blocks []json.RawMessage,
	rolePrefix string,
	pending map[string]toolSummaryPair,
) []string {
	result := make([]string, 0, len(blocks))

	for _, block := range blocks {
		var blockInfo rawBlockType

		unmarshalErr := json.Unmarshal(block, &blockInfo)
		if unmarshalErr != nil {
			continue
		}

		switch blockInfo.Type {
		case blockTypeText:
			result = append(result, extractTextBlock(block, rolePrefix)...)
		case blockTypeToolUse:
			storePendingToolUse(block, pending)
		case blockTypeToolResult:
			if summary := matchToolResult(block, pending); summary != "" {
				result = append(result, summary)
			}
		}
	}

	return result
}

// storePendingToolUse parses a tool_use block and stores it in the pending map.
func storePendingToolUse(block json.RawMessage, pending map[string]toolSummaryPair) {
	var toolUse toolUseBlock

	err := json.Unmarshal(block, &toolUse)
	if err != nil {
		return
	}

	pending[toolUse.ID] = toolSummaryPair{
		name: toolUse.Name,
		args: formatToolSummaryArgs(toolUse.Input),
	}
}

// stripWithToolSummary processes JSONL lines producing compact tool summaries.
// Text blocks are emitted as "USER: ..." / "ASSISTANT: ..." lines.
// Each tool_use + tool_result pair becomes a single "[tool]" summary line.
// Orphaned tool_use blocks (no matching result) are silently dropped.
func stripWithToolSummary(lines []string) []string {
	result := make([]string, 0, len(lines))
	pending := make(map[string]toolSummaryPair)

	for _, line := range lines {
		if !isKeptType(line) {
			continue
		}

		cleaned := replaceBase64(line)
		extracted := extractSummaryBlocks(cleaned, pending)
		result = append(result, extracted...)
	}

	return result
}
