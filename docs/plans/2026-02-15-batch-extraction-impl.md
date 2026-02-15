# Batch Extraction Pipeline Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the current per-item session extraction with a strip→chunk→Haiku→Sonnet pipeline that processes full session transcripts.

**Architecture:** Reuses existing `DirectAPIExtractor` infrastructure and `parseTranscriptMessages` parser. Adds a new `callAPIWithMessages` method for system prompt + assistant prefill support. Pipeline orchestrated as `StripSession → ChunkText → IdentifyEvents (parallel) → ExtractPrinciples → Learn`.

**Tech Stack:** Go, Anthropic API (Haiku for events, Sonnet for principles), existing SQLite embeddings DB

---

### Task 1: StripSession Function

Strip a JSONL session transcript to learning-relevant content only.

**Files:**
- Create: `internal/memory/strip_session.go`
- Create: `internal/memory/strip_session_test.go`

**Step 1: Write the failing test**

```go
// strip_session_test.go
package memory_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/toejough/projctl/internal/memory"
)

func TestStripSession_BasicBlocks(t *testing.T) {
	// Create temp JSONL file with representative blocks
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	lines := []string{
		// User text (should keep)
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"fix the bug"}]}}`,
		// System reminder (should strip)
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"<system-reminder>hook output</system-reminder>real message"}]}}`,
		// Assistant text (should keep)
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I'll fix that."}]}}`,
		// Tool use - Edit with diff (should show diff)
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Edit","input":{"file_path":"auth.go","old_string":"ExpiresAt string","new_string":"ExpiresAt any"}}]}}`,
		// Tool result success (should omit)
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"1","content":"File edited successfully"}]}}`,
		// Tool result error (should keep)
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"2","content":"Error: file not found","is_error":true}]}}`,
		// Thinking block (should skip)
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking","thinking":"let me think..."}]}}`,
		// Progress message (should skip entirely)
		`{"type":"progress","message":"loading"}`,
	}

	content := ""
	for _, l := range lines {
		content += l + "\n"
	}
	os.WriteFile(path, []byte(content), 0644)

	result, err := memory.StripSession(path)
	if err != nil {
		t.Fatalf("StripSession failed: %v", err)
	}

	// Should contain user text
	if !contains(result, "fix the bug") {
		t.Error("missing user text")
	}
	// Should strip system-reminder but keep real message
	if contains(result, "hook output") {
		t.Error("system-reminder not stripped")
	}
	if !contains(result, "real message") {
		t.Error("real message after system-reminder stripped")
	}
	// Should contain assistant text
	if !contains(result, "I'll fix that") {
		t.Error("missing assistant text")
	}
	// Should show Edit diff
	if !contains(result, "ExpiresAt string") || !contains(result, "ExpiresAt any") {
		t.Error("Edit diff not shown")
	}
	// Should omit successful tool result
	if contains(result, "File edited successfully") {
		t.Error("successful tool result not omitted")
	}
	// Should keep error tool result
	if !contains(result, "file not found") {
		t.Error("error tool result omitted")
	}
	// Should not contain thinking
	if contains(result, "let me think") {
		t.Error("thinking block not stripped")
	}
}

func TestStripSession_TeammateMessages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	content := `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"<teammate-message teammate_id=\"worker-a\" summary=\"task done\">Task 1 complete, all tests pass.</teammate-message>"}]}}` + "\n"
	os.WriteFile(path, []byte(content), 0644)

	result, err := memory.StripSession(path)
	if err != nil {
		t.Fatalf("StripSession failed: %v", err)
	}

	// Should keep teammate message content with sender
	if !contains(result, "worker-a") || !contains(result, "Task 1 complete") {
		t.Errorf("teammate message not preserved: %s", result)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
```

**Step 2: Run test to verify it fails**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run TestStripSession -v -count=1`
Expected: FAIL — `StripSession` undefined

**Step 3: Write minimal implementation**

```go
// strip_session.go
package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var systemReminderRe = regexp.MustCompile(`(?s)<system-reminder>.*?</system-reminder>`)
var teammateRe = regexp.MustCompile(`(?s)<teammate-message[^>]*teammate_id="([^"]*)"[^>]*>(.*?)</teammate-message>`)

// StripSession reads a JSONL session transcript and returns stripped text
// containing only learning-relevant content.
func StripSession(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open session: %w", err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10MB line buffer
	for scanner.Scan() {
		var msg map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}

		msgType, _ := msg["type"].(string)
		if msgType != "user" && msgType != "assistant" {
			continue
		}

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
				text = stripNoise(text)
				if text != "" {
					lines = append(lines, fmt.Sprintf("[%s] %s", role, text))
				}

			case "tool_use":
				name, _ := block["name"].(string)
				input, _ := block["input"].(map[string]any)
				line := formatToolUse(name, input)
				if line != "" {
					lines = append(lines, fmt.Sprintf("[%s] %s", role, line))
				}

			case "tool_result":
				isError, _ := block["is_error"].(bool)
				if !isError {
					continue // omit successful results
				}
				text, _ := block["content"].(string)
				if text == "" {
					text, _ = block["text"].(string)
				}
				if len(text) > 300 {
					text = text[:300] + "..."
				}
				lines = append(lines, fmt.Sprintf("[%s] ERROR: %s", role, text))

			case "thinking":
				// skip
			}
		}
	}

	return strings.Join(lines, "\n\n"), scanner.Err()
}

// stripNoise removes system-reminders and extracts teammate messages.
func stripNoise(text string) string {
	// Extract teammate messages first (before stripping system-reminders)
	text = teammateRe.ReplaceAllStringFunc(text, func(match string) string {
		m := teammateRe.FindStringSubmatch(match)
		if len(m) >= 3 {
			return fmt.Sprintf("[teammate %s] %s", m[1], strings.TrimSpace(m[2]))
		}
		return match
	})

	// Strip system-reminders
	text = systemReminderRe.ReplaceAllString(text, "")

	// Collapse skill content
	if isSkillContent(text) {
		return "(skill loaded)"
	}

	return strings.TrimSpace(text)
}

// isSkillContent detects skill loading patterns.
func isSkillContent(text string) bool {
	return strings.Contains(text, "Base directory for this skill:") ||
		strings.Contains(text, "Launching skill:")
}

// formatToolUse formats a tool invocation for stripped output.
func formatToolUse(name string, input map[string]any) string {
	switch name {
	case "Bash":
		cmd, _ := input["command"].(string)
		// Truncate heredocs
		if idx := strings.Index(cmd, "<<"); idx > 0 {
			if nl := strings.Index(cmd[idx:], "\n"); nl > 0 {
				cmd = cmd[:idx+nl] + fmt.Sprintf("... [%d chars]", len(cmd))
			}
		}
		return fmt.Sprintf("TOOL:Bash $ %s", cmd)

	case "Edit":
		fp, _ := input["file_path"].(string)
		old, _ := input["old_string"].(string)
		new, _ := input["new_string"].(string)
		if old == "" {
			return fmt.Sprintf("TOOL:Edit %s", fp)
		}
		diff := computeEditDiff(old, new)
		return fmt.Sprintf("TOOL:Edit %s\n%s", fp, diff)

	case "Write":
		fp, _ := input["file_path"].(string)
		return fmt.Sprintf("TOOL:Write %s", fp)

	case "Read", "Glob", "Grep":
		b, _ := json.Marshal(input)
		s := string(b)
		if len(s) > 150 {
			s = s[:150] + "..."
		}
		return fmt.Sprintf("TOOL:%s %s", name, s)

	default:
		b, _ := json.Marshal(input)
		s := string(b)
		if len(s) > 150 {
			s = s[:150] + "..."
		}
		return fmt.Sprintf("TOOL:%s %s", name, s)
	}
}

// computeEditDiff shows only what changed between old and new strings.
func computeEditDiff(old, new string) string {
	// Find common prefix
	prefixLen := 0
	for i := 0; i < len(old) && i < len(new); i++ {
		if old[i] != new[i] {
			break
		}
		prefixLen = i + 1
	}

	// Find common suffix
	suffixLen := 0
	for i := 1; i <= len(old)-prefixLen && i <= len(new)-prefixLen; i++ {
		if old[len(old)-i] != new[len(new)-i] {
			break
		}
		suffixLen = i
	}

	oldChanged := old[prefixLen : len(old)-suffixLen]
	newChanged := new[prefixLen : len(new)-suffixLen]

	// Add context (up to 60 chars around the change)
	ctx := 60
	ctxBefore := ""
	if prefixLen > 0 {
		start := prefixLen - ctx
		if start < 0 {
			start = 0
		}
		ctxBefore = old[start:prefixLen]
	}

	var result strings.Builder
	if ctxBefore != "" {
		result.WriteString("  ..." + ctxBefore + "\n")
	}
	if oldChanged != "" {
		result.WriteString("  - " + oldChanged + "\n")
	}
	if newChanged != "" {
		result.WriteString("  + " + newChanged)
	}
	return result.String()
}
```

**Step 4: Run test to verify it passes**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run TestStripSession -v -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/memory/strip_session.go internal/memory/strip_session_test.go
git commit -m "feat(memory): add StripSession for transcript noise removal"
```

---

### Task 2: ChunkText Function

Split stripped text into ~25KB chunks on line boundaries.

**Files:**
- Create: `internal/memory/chunk.go`
- Create: `internal/memory/chunk_test.go`

**Step 1: Write the failing test**

```go
// chunk_test.go
package memory_test

import (
	"strings"
	"testing"

	"github.com/toejough/projctl/internal/memory"
)

func TestChunkText_SmallInput(t *testing.T) {
	// Input smaller than chunk size → single chunk
	text := "line 1\nline 2\nline 3"
	chunks := memory.ChunkText(text, 1000)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Text != text {
		t.Errorf("chunk text mismatch")
	}
	if chunks[0].StartLine != 1 || chunks[0].EndLine != 3 {
		t.Errorf("chunk lines: want 1-3, got %d-%d", chunks[0].StartLine, chunks[0].EndLine)
	}
}

func TestChunkText_SplitsOnLineBoundary(t *testing.T) {
	// Build text that requires splitting
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, strings.Repeat("x", 100)) // 100 bytes per line
	}
	text := strings.Join(lines, "\n") // ~10KB total

	chunks := memory.ChunkText(text, 2500) // ~25 lines per chunk
	if len(chunks) < 3 {
		t.Fatalf("expected at least 3 chunks, got %d", len(chunks))
	}

	// Verify no content lost
	var reassembled []string
	for _, c := range chunks {
		reassembled = append(reassembled, c.Text)
	}
	if strings.Join(reassembled, "\n") != text {
		t.Error("reassembled chunks don't match original")
	}

	// Verify line numbers are contiguous
	for i := 1; i < len(chunks); i++ {
		if chunks[i].StartLine != chunks[i-1].EndLine+1 {
			t.Errorf("gap between chunk %d (end %d) and %d (start %d)",
				i-1, chunks[i-1].EndLine, i, chunks[i].StartLine)
		}
	}
}

func TestChunkText_EmptyInput(t *testing.T) {
	chunks := memory.ChunkText("", 1000)
	if len(chunks) != 0 {
		t.Fatalf("expected 0 chunks for empty input, got %d", len(chunks))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run TestChunkText -v -count=1`
Expected: FAIL — `ChunkText` undefined

**Step 3: Write minimal implementation**

```go
// chunk.go
package memory

import "strings"

// TextChunk represents a portion of stripped session text.
type TextChunk struct {
	Text      string
	StartLine int
	EndLine   int
	Index     int
}

// ChunkText splits text into chunks of approximately maxBytes each,
// splitting on line boundaries. Returns empty slice for empty input.
func ChunkText(text string, maxBytes int) []TextChunk {
	if text == "" {
		return nil
	}

	lines := strings.Split(text, "\n")
	totalLines := len(lines)

	// Estimate lines per chunk based on average line length
	avgLineLen := len(text) / totalLines
	if avgLineLen == 0 {
		avgLineLen = 1
	}
	linesPerChunk := maxBytes / avgLineLen
	if linesPerChunk < 1 {
		linesPerChunk = 1
	}

	var chunks []TextChunk
	for i := 0; i < totalLines; i += linesPerChunk {
		end := i + linesPerChunk
		if end > totalLines {
			end = totalLines
		}
		chunkLines := lines[i:end]
		chunks = append(chunks, TextChunk{
			Text:      strings.Join(chunkLines, "\n"),
			StartLine: i + 1,
			EndLine:   end,
			Index:     len(chunks),
		})
	}

	return chunks
}
```

**Step 4: Run test to verify it passes**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run TestChunkText -v -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/memory/chunk.go internal/memory/chunk_test.go
git commit -m "feat(memory): add ChunkText for line-boundary splitting"
```

---

### Task 3: callAPIWithMessages — Extended API Method

Extend `DirectAPIExtractor` to support system prompts, multiple messages, and assistant prefill.

**Files:**
- Modify: `internal/memory/llm_api.go` (add `callAPIWithMessages`)
- Create: `internal/memory/llm_api_messages_test.go`

**Step 1: Write the failing test**

```go
// llm_api_messages_test.go
package memory_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/toejough/projctl/internal/memory"
)

func TestCallAPIWithMessages_SendsSystemAndPrefill(t *testing.T) {
	var receivedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"text": `{"result": "ok"}`},
			},
		})
	}))
	defer server.Close()

	ext := memory.NewDirectAPIExtractor("test-token",
		memory.WithBaseURL(server.URL),
	)

	params := memory.APIMessageParams{
		System:   "You are a test analyzer.",
		Messages: []memory.APIMessage{
			{Role: "user", Content: "Analyze this."},
			{Role: "assistant", Content: "["},
		},
		MaxTokens: 1024,
	}

	_, err := ext.CallAPIWithMessages(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify system prompt was sent
	sys, _ := receivedBody["system"].(string)
	if sys != "You are a test analyzer." {
		t.Errorf("system prompt not sent: got %q", sys)
	}

	// Verify messages include both user and assistant
	msgs, _ := receivedBody["messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}

	msg1, _ := msgs[1].(map[string]any)
	if role, _ := msg1["role"].(string); role != "assistant" {
		t.Errorf("second message role: want assistant, got %s", role)
	}
	if content, _ := msg1["content"].(string); content != "[" {
		t.Errorf("assistant prefill: want [, got %s", content)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run TestCallAPIWithMessages -v -count=1`
Expected: FAIL — `APIMessageParams`, `CallAPIWithMessages` undefined

**Step 3: Write minimal implementation**

Add to `internal/memory/llm_api.go`:

```go
// APIMessage represents a single message in a conversation.
type APIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// APIMessageParams holds parameters for multi-message API calls.
type APIMessageParams struct {
	System    string
	Messages  []APIMessage
	MaxTokens int
	Model     string // override default model if set
}

// CallAPIWithMessages sends a multi-message request with optional system prompt
// and assistant prefill. Returns the raw text response.
func (d *DirectAPIExtractor) CallAPIWithMessages(ctx context.Context, params APIMessageParams) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	model := d.model
	if params.Model != "" {
		model = params.Model
	}

	body := map[string]any{
		"model":      model,
		"max_tokens": params.MaxTokens,
	}
	if params.System != "" {
		body["system"] = params.System
	}

	var msgs []map[string]any
	for _, m := range params.Messages {
		msgs = append(msgs, map[string]any{"role": m.Role, "content": m.Content})
	}
	body["messages"] = msgs

	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", d.baseURL+"/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLLMUnavailable, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.token)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLLMUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w: API returned %d", ErrLLMUnavailable, resp.StatusCode)
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%w: failed to decode API response: %v", ErrLLMUnavailable, err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("%w: API error: %s", ErrLLMUnavailable, result.Error.Message)
	}
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("%w: empty response content", ErrLLMUnavailable)
	}

	return []byte(result.Content[0].Text), nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run TestCallAPIWithMessages -v -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/memory/llm_api.go internal/memory/llm_api_messages_test.go
git commit -m "feat(memory): add CallAPIWithMessages for system prompt + prefill support"
```

---

### Task 4: IdentifyEvents — Haiku Event Extraction

Add `IdentifyEvents` method that sends a text chunk to Haiku and returns structured events.

**Files:**
- Create: `internal/memory/batch_extract.go` (types + IdentifyEvents)
- Create: `internal/memory/batch_extract_test.go`

**Step 1: Write the failing test**

```go
// batch_extract_test.go
package memory_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/toejough/projctl/internal/memory"
)

func TestIdentifyEvents_ParsesHaikuResponse(t *testing.T) {
	// Mock Haiku returning events (note: response is WITHOUT the leading [
	// because the prefill provides it)
	responseEvents := `
		{"line_range": "1-20", "event_type": "root-cause-discovery", "what_happened": "Found type mismatch.", "why_it_matters": "Check types."},
		{"line_range": "30-40", "event_type": "user-correction", "what_happened": "User said no.", "why_it_matters": "Listen."}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"text": responseEvents},
			},
		})
	}))
	defer server.Close()

	ext := memory.NewDirectAPIExtractor("test-token",
		memory.WithBaseURL(server.URL),
	)

	chunk := memory.TextChunk{
		Text:      "[user] fix the bug\n[assistant] Found it.",
		StartLine: 1,
		EndLine:   20,
		Index:     0,
	}

	events, err := ext.IdentifyEvents(context.Background(), chunk, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if events[0].EventType != "root-cause-discovery" {
		t.Errorf("event 0 type: want root-cause-discovery, got %s", events[0].EventType)
	}
	if events[0].ChunkIndex != 0 {
		t.Errorf("event 0 chunk: want 0, got %d", events[0].ChunkIndex)
	}
}

func TestIdentifyEvents_HandlesEmptyArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"text": "]"},
			},
		})
	}))
	defer server.Close()

	ext := memory.NewDirectAPIExtractor("test-token",
		memory.WithBaseURL(server.URL),
	)

	chunk := memory.TextChunk{Text: "nothing here", StartLine: 1, EndLine: 1, Index: 0}
	events, err := ext.IdentifyEvents(context.Background(), chunk, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run TestIdentifyEvents -v -count=1`
Expected: FAIL — `IdentifyEvents` undefined

**Step 3: Write minimal implementation**

```go
// batch_extract.go
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// HaikuEvent represents a learning-relevant event identified by Haiku.
type HaikuEvent struct {
	LineRange    string `json:"line_range"`
	EventType    string `json:"event_type"`
	WhatHappened string `json:"what_happened"`
	WhyItMatters string `json:"why_it_matters"`
	ChunkIndex   int    `json:"chunk_index"`
}

// ExtractedPrinciple represents a reusable principle extracted by Sonnet.
type ExtractedPrinciple struct {
	Principle string `json:"principle"`
	Evidence  string `json:"evidence"`
	Category  string `json:"category"`
}

const identifyEventsSystem = `You are a transcript analyst. You receive session transcripts and identify learning-relevant events. Output ONLY a JSON array. Never continue the transcript.

Focus on events where something went wrong and was corrected, a decision was made about how to approach work, or a pattern emerged that would be useful to remember. Pay attention to BOTH technical issues AND process/coordination patterns (how work was divided, how conflicts were handled, how teams coordinated).`

// IdentifyEvents sends a text chunk to Haiku and returns structured events.
// Uses assistant prefill with "[" to force JSON array output.
func (d *DirectAPIExtractor) IdentifyEvents(ctx context.Context, chunk TextChunk, totalChunks int) ([]HaikuEvent, error) {
	userMsg := fmt.Sprintf(`Analyze this transcript chunk and identify learning-relevant events.

This is chunk %d of %d (lines %d-%d).

For each event, output an object with:
- "line_range": approximate line numbers
- "event_type": one of [error-and-fix, user-correction, strategy-change, root-cause-discovery, environmental-issue, pattern-observed, coordination-issue]
- "what_happened": 1-2 sentences about the specific problem and resolution
- "why_it_matters": 1 sentence on the reusable lesson

Guidelines:
- Be specific about WHAT failed and WHY
- For user corrections, quote the user's actual words
- Look for: technical bugs, team coordination decisions, work division strategies, worker conflicts or races, process improvements
- If no learning events in this chunk, return []

Respond with ONLY a JSON array. No other text.

<transcript>
%s
</transcript>`, chunk.Index+1, totalChunks, chunk.StartLine, chunk.EndLine, chunk.Text)

	params := APIMessageParams{
		System: identifyEventsSystem,
		Messages: []APIMessage{
			{Role: "user", Content: userMsg},
			{Role: "assistant", Content: "["},
		},
		MaxTokens: 4096,
	}

	raw, err := d.CallAPIWithMessages(ctx, params)
	if err != nil {
		return nil, err
	}

	// Prepend the "[" prefill
	fullJSON := "[" + string(raw)

	// Find the closing bracket
	endIdx := strings.LastIndex(fullJSON, "]")
	if endIdx < 0 {
		return nil, fmt.Errorf("no closing ] in response")
	}

	var events []HaikuEvent
	if err := json.Unmarshal([]byte(fullJSON[:endIdx+1]), &events); err != nil {
		return nil, fmt.Errorf("parse events: %w", err)
	}

	// Tag with chunk index
	for i := range events {
		events[i].ChunkIndex = chunk.Index
	}

	return events, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run TestIdentifyEvents -v -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/memory/batch_extract.go internal/memory/batch_extract_test.go
git commit -m "feat(memory): add IdentifyEvents for Haiku event extraction per chunk"
```

---

### Task 5: ExtractPrinciples — Sonnet Principle Extraction

Add `ExtractPrinciples` method that sends all events to Sonnet and returns principles.

**Files:**
- Modify: `internal/memory/batch_extract.go` (add `ExtractPrinciples`)
- Modify: `internal/memory/batch_extract_test.go` (add tests)

**Step 1: Write the failing test**

```go
// Add to batch_extract_test.go
func TestExtractPrinciples_ParsesSonnetResponse(t *testing.T) {
	responsePrinciples := `
		{"principle": "When X, do Y.", "evidence": "Session showed X failing.", "category": "debugging"}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Sonnet model is used
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		model, _ := body["model"].(string)
		if model != "claude-sonnet-4-5-20250929" {
			t.Errorf("expected sonnet model, got %s", model)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"text": responsePrinciples},
			},
		})
	}))
	defer server.Close()

	ext := memory.NewDirectAPIExtractor("test-token",
		memory.WithBaseURL(server.URL),
	)

	events := []memory.HaikuEvent{
		{EventType: "root-cause-discovery", WhatHappened: "Found bug.", WhyItMatters: "Check types."},
	}

	principles, err := ext.ExtractPrinciples(context.Background(), events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(principles) != 1 {
		t.Fatalf("expected 1 principle, got %d", len(principles))
	}
	if principles[0].Category != "debugging" {
		t.Errorf("category: want debugging, got %s", principles[0].Category)
	}
}

func TestExtractPrinciples_NoEvents(t *testing.T) {
	ext := memory.NewDirectAPIExtractor("test-token")
	principles, err := ext.ExtractPrinciples(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(principles) != 0 {
		t.Errorf("expected 0 principles for nil events, got %d", len(principles))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run TestExtractPrinciples -v -count=1`
Expected: FAIL — `ExtractPrinciples` undefined

**Step 3: Write minimal implementation**

Add to `internal/memory/batch_extract.go`:

```go
const sonnetModel = "claude-sonnet-4-5-20250929"

const extractPrinciplesSystem = `You are a learning extraction system. You receive events identified from coding session transcripts and synthesize them into reusable, actionable principles.

Your output is ONLY a JSON array of principle objects. Never output anything else.`

// ExtractPrinciples sends all events to Sonnet and returns actionable principles.
func (d *DirectAPIExtractor) ExtractPrinciples(ctx context.Context, events []HaikuEvent) ([]ExtractedPrinciple, error) {
	if len(events) == 0 {
		return nil, nil
	}

	eventsJSON, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal events: %w", err)
	}

	userMsg := fmt.Sprintf(`Given these events from a coding session, extract reusable principles that an AI coding assistant should remember for future sessions.

Rules:
- Merge events about the same underlying issue into one principle
- Each principle must be specific and actionable — not generic advice
- Frame principles as "When X, do Y" or "Before X, check Y" patterns
- Include the concrete example from the session that demonstrates the principle
- If an event is just routine work (no lesson), skip it
- Aim for 3-8 principles per session — fewer is better than padding
- Process and coordination lessons are EQUALLY important as technical lessons. If events describe how work was structured, how agents were assigned roles, what quality gates were used, or how conflicts between workers were resolved — these MUST be extracted as separate principles. Do not merge them into technical principles or drop them.

Output each principle as:
- "principle": The actionable rule (1-2 sentences)
- "evidence": What happened in the session that demonstrates this (1-2 sentences)
- "category": one of [debugging, git-workflow, api-design, team-coordination, testing, code-quality, cli-design]

Events:
%s`, string(eventsJSON))

	params := APIMessageParams{
		System: extractPrinciplesSystem,
		Messages: []APIMessage{
			{Role: "user", Content: userMsg},
			{Role: "assistant", Content: "["},
		},
		MaxTokens: 4096,
		Model:     sonnetModel,
	}

	raw, err := d.CallAPIWithMessages(ctx, params)
	if err != nil {
		return nil, err
	}

	fullJSON := "[" + string(raw)
	endIdx := strings.LastIndex(fullJSON, "]")
	if endIdx < 0 {
		return nil, fmt.Errorf("no closing ] in response")
	}

	var principles []ExtractedPrinciple
	if err := json.Unmarshal([]byte(fullJSON[:endIdx+1]), &principles); err != nil {
		return nil, fmt.Errorf("parse principles: %w", err)
	}

	return principles, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run TestExtractPrinciples -v -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/memory/batch_extract.go internal/memory/batch_extract_test.go
git commit -m "feat(memory): add ExtractPrinciples for Sonnet-based principle extraction"
```

---

### Task 6: BatchExtractSession — Pipeline Orchestration

Wire the full pipeline: strip → chunk → parallel Haiku → Sonnet → store.

**Files:**
- Modify: `internal/memory/batch_extract.go` (add `BatchExtractSession`)
- Modify: `internal/memory/batch_extract_test.go` (add integration test)

**Step 1: Write the failing test**

```go
// Add to batch_extract_test.go
func TestBatchExtractSession_EndToEnd(t *testing.T) {
	// Create a minimal session JSONL
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.jsonl")
	content := strings.Join([]string{
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"fix the auth bug"}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Found it: ExpiresAt was string, should be any."}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Edit","input":{"file_path":"auth.go","old_string":"ExpiresAt string","new_string":"ExpiresAt any"}}]}}`,
	}, "\n")
	os.WriteFile(sessionPath, []byte(content), 0644)

	// Mock server that responds to both Haiku and Sonnet calls
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		model, _ := body["model"].(string)

		callCount++
		if strings.Contains(model, "haiku") {
			// Return a single event
			json.NewEncoder(w).Encode(map[string]any{
				"content": []map[string]any{
					{"text": `{"event_type":"root-cause-discovery","what_happened":"Type mismatch found.","why_it_matters":"Check types.","line_range":"1-3"}]`},
				},
			})
		} else {
			// Sonnet: return a principle
			json.NewEncoder(w).Encode(map[string]any{
				"content": []map[string]any{
					{"text": `{"principle":"When integrating with external data, use flexible types.","evidence":"ExpiresAt was string but should be any.","category":"debugging"}]`},
				},
			})
		}
	}))
	defer server.Close()

	ext := memory.NewDirectAPIExtractor("test-token",
		memory.WithBaseURL(server.URL),
	)

	result, err := memory.BatchExtractSession(context.Background(), sessionPath, ext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.StrippedSize == 0 {
		t.Error("stripped size should be > 0")
	}
	if result.ChunkCount == 0 {
		t.Error("chunk count should be > 0")
	}
	if len(result.Events) == 0 {
		t.Error("should have events")
	}
	if len(result.Principles) == 0 {
		t.Error("should have principles")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run TestBatchExtractSession -v -count=1`
Expected: FAIL — `BatchExtractSession` undefined

**Step 3: Write minimal implementation**

Add to `internal/memory/batch_extract.go`:

```go
import "sync"

const defaultChunkSize = 25000 // 25KB
const maxParallelChunks = 4

// BatchExtractResult holds the full pipeline output.
type BatchExtractResult struct {
	StrippedSize  int
	ChunkCount    int
	ChunkFailures int
	Events        []HaikuEvent
	Principles    []ExtractedPrinciple
}

// BatchExtractSession runs the full extraction pipeline on a session transcript.
func BatchExtractSession(ctx context.Context, sessionPath string, ext *DirectAPIExtractor) (*BatchExtractResult, error) {
	// Stage 1: Strip
	stripped, err := StripSession(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("strip session: %w", err)
	}

	if len(stripped) == 0 {
		return &BatchExtractResult{}, nil
	}

	// Stage 2: Chunk
	chunks := ChunkText(stripped, defaultChunkSize)

	// Stage 3: Haiku event identification (parallel)
	type chunkResult struct {
		events []HaikuEvent
		err    error
		index  int
	}

	results := make(chan chunkResult, len(chunks))
	sem := make(chan struct{}, maxParallelChunks)
	var wg sync.WaitGroup

	for _, chunk := range chunks {
		wg.Add(1)
		go func(c TextChunk) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			events, err := ext.IdentifyEvents(ctx, c, len(chunks))
			results <- chunkResult{events: events, err: err, index: c.Index}
		}(chunk)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var allEvents []HaikuEvent
	failures := 0
	for r := range results {
		if r.err != nil {
			failures++
			continue
		}
		allEvents = append(allEvents, r.events...)
	}

	// Sort events by chunk index then line range for stable ordering
	sortEvents(allEvents)

	// Stage 4: Sonnet principle extraction
	principles, err := ext.ExtractPrinciples(ctx, allEvents)
	if err != nil {
		return nil, fmt.Errorf("extract principles: %w", err)
	}

	return &BatchExtractResult{
		StrippedSize:  len(stripped),
		ChunkCount:    len(chunks),
		ChunkFailures: failures,
		Events:        allEvents,
		Principles:    principles,
	}, nil
}

func sortEvents(events []HaikuEvent) {
	// Simple sort by chunk index, then line range string
	for i := 1; i < len(events); i++ {
		for j := i; j > 0; j-- {
			if events[j].ChunkIndex < events[j-1].ChunkIndex ||
				(events[j].ChunkIndex == events[j-1].ChunkIndex && events[j].LineRange < events[j-1].LineRange) {
				events[j], events[j-1] = events[j-1], events[j]
			} else {
				break
			}
		}
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run TestBatchExtractSession -v -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/memory/batch_extract.go internal/memory/batch_extract_test.go
git commit -m "feat(memory): add BatchExtractSession pipeline orchestration"
```

---

### Task 7: Update CLI — Wire Pipeline Into learn-sessions

Replace the existing `processSession` in the CLI with the new batch pipeline, storing extracted principles via `Learn()`.

**Files:**
- Modify: `cmd/projctl/memory_learn_sessions.go` (replace `processSession`)

**Step 1: Run existing tests to establish baseline**

Run: `go build -tags sqlite_fts5 ./cmd/projctl/ 2>&1`
Expected: PASS (builds clean)

**Step 2: Update processSession to use BatchExtractSession**

Replace the `processSession` function (lines 194-218) and update `processSessionWithTimeout` to use the new pipeline:

```go
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

	result, err := memory.BatchExtractSession(context.Background(), session.Path, directExt)
	if err != nil {
		return nil, "error", err
	}

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
```

Also update the progress output in `memoryLearnSessions` (line 131) to show pipeline stages:

```go
fmt.Printf("[%d/%d] Processing %s (%s, %s)...\n",
	i+1, len(unevaluated), session.SessionID, session.Project, formatSize(session.Size))
```

**Step 3: Increase timeout for batch processing**

The 60s timeout is too short for large sessions with many chunks. Update `processSessionWithTimeout` (line 174):

```go
timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
```

**Step 4: Build and verify**

Run: `go build -tags sqlite_fts5 ./cmd/projctl/ 2>&1`
Expected: Builds clean

**Step 5: Commit**

```bash
git add cmd/projctl/memory_learn_sessions.go
git commit -m "feat(memory): wire batch extraction pipeline into learn-sessions CLI"
```

---

### Task 8: Export DirectAPIExtractor Type

The CLI needs to cast `LLMClient` to `*DirectAPIExtractor`. Ensure `NewLLMExtractor` returns the concrete type or make `BatchExtractSession` accept the interface.

**Files:**
- Modify: `internal/memory/llm_api.go` (check NewLLMExtractor return type)

**Step 1: Check current return type**

Read `internal/memory/llm_api.go` around line 397. `NewLLMExtractor` currently returns `LLMClient` interface. The CLI does a type assertion `ext.(*memory.DirectAPIExtractor)`.

This works because `DirectAPIExtractor` is already an exported type. The type assertion is fine as long as `NewLLMExtractor` actually returns a `*DirectAPIExtractor` underneath.

Verify this works by checking the existing code compiles with the Task 7 changes:

Run: `go build -tags sqlite_fts5 ./cmd/projctl/ 2>&1`
Expected: Builds clean. If not, adjust BatchExtractSession to accept LLMClient and add IdentifyEvents/ExtractPrinciples to the interface.

**Step 2: If type assertion fails, add a BatchExtractor interface**

Only needed if Step 1 fails:

```go
// BatchExtractor extends LLMExtractor with batch pipeline methods.
type BatchExtractor interface {
	IdentifyEvents(ctx context.Context, chunk TextChunk, totalChunks int) ([]HaikuEvent, error)
	ExtractPrinciples(ctx context.Context, events []HaikuEvent) ([]ExtractedPrinciple, error)
}
```

**Step 3: Commit if any changes needed**

```bash
git add internal/memory/llm_api.go
git commit -m "fix(memory): ensure BatchExtractSession works with LLMClient interface"
```
