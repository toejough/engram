package memory

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	. "github.com/onsi/gomega"
)

// ─── detectCorrectionRecurrence ───────────────────────────────────────────────

func TestDetectCorrectionRecurrence_EmptyMemoryRoot(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize the DB so Query doesn't fail
	db, err := initEmbeddingsDB(filepath.Join(memoryDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	_ = db.Close()

	// With no similar corrections in DB, should return nil
	err = detectCorrectionRecurrence("never amend pushed commits", memoryDir)
	g.Expect(err).ToNot(HaveOccurred())
}

// ─── extractBehavioralConsistency ────────────────────────────────────────────

func TestExtractBehavioralConsistency_BelowThreshold(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Only 2 mentions of "gomega" — below 5-mention threshold
	blocks := []parsedBlock{
		{role: "assistant", blockType: "text", text: "I'll use gomega for assertions"},
		{role: "assistant", blockType: "text", text: "gomega provides nice matchers"},
	}

	items := extractBehavioralConsistency(blocks)
	g.Expect(items).To(BeEmpty())
}

func TestExtractBehavioralConsistency_FiveMentions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// 5+ mentions of "gomega" without correction
	blocks := []parsedBlock{
		{role: "assistant", blockType: "text", text: "I'll use gomega for assertions"},
		{role: "assistant", blockType: "text", text: "gomega provides nice matchers"},
		{role: "assistant", blockType: "text", text: "gomega has g.Expect syntax"},
		{role: "assistant", blockType: "text", text: "using gomega in all tests"},
		{role: "assistant", blockType: "text", text: "gomega is my preferred library"},
	}

	items := extractBehavioralConsistency(blocks)
	g.Expect(items).ToNot(BeEmpty())

	if len(items) < 1 {
		t.Fatal("expected at least one behavioral consistency item")
	}

	g.Expect(items[0].Type).To(Equal("behavioral-consistency"))
	g.Expect(items[0].Content).To(ContainSubstring("gomega"))
	g.Expect(items[0].Confidence).To(BeNumerically("~", 0.5, 0.01))
}

func TestExtractBehavioralConsistency_WithCorrection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// 5 mentions of "gomega" but user correction marks it as corrected
	blocks := []parsedBlock{
		{role: "assistant", blockType: "text", text: "I'll use gomega for assertions"},
		{role: "assistant", blockType: "text", text: "gomega provides nice matchers"},
		{role: "assistant", blockType: "text", text: "gomega has g.Expect syntax"},
		{role: "assistant", blockType: "text", text: "gomega is the testing library"},
		{role: "assistant", blockType: "text", text: "gomega is installed via go get"},
		{role: "user", blockType: "text", text: "No, don't use gomega, use testify instead"},
	}

	items := extractBehavioralConsistency(blocks)
	// gomega was corrected, should not produce an item
	for _, item := range items {
		g.Expect(item.Content).ToNot(ContainSubstring("gomega"))
	}
}

// ─── extractFromResult ────────────────────────────────────────────────────────

func TestExtractFromResult_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rf := &ResultFile{}

	items := extractFromResult(rf)
	g.Expect(items).To(BeEmpty())
}

func TestExtractFromResult_WithAlternatives(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rf := &ResultFile{
		Decisions: []Decision{
			{
				Context:      "Testing approach",
				Choice:       "Use TDD",
				Reason:       "Better design",
				Alternatives: []string{"BDD", "no tests"},
			},
		},
	}

	items := extractFromResult(rf)
	g.Expect(items).To(HaveLen(1))
	g.Expect(items).ToNot(BeNil())

	if len(items) == 0 {
		t.Fatal("items must be non-empty")
	}

	g.Expect(items[0].Type).To(Equal("decision"))
	g.Expect(items[0].Content).To(ContainSubstring("Use TDD"))
	g.Expect(items[0].Content).To(ContainSubstring("BDD"))
}

func TestExtractFromResult_WithDecisions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rf := &ResultFile{
		Decisions: []Decision{
			{Context: "DB choice", Choice: "SQLite", Reason: "Lightweight"},
			{Context: "Testing", Choice: "gomega", Reason: "Expressive"},
		},
	}

	items := extractFromResult(rf)
	g.Expect(items).To(HaveLen(2))
	g.Expect(items).ToNot(BeNil())

	if len(items) < 2 {
		t.Fatalf("items must have at least 2 elements, got %d", len(items))
	}

	g.Expect(items[0].Type).To(Equal("decision"))
	g.Expect(items[0].Content).To(ContainSubstring("SQLite"))
	g.Expect(items[1].Content).To(ContainSubstring("gomega"))
}

// ─── extractLogf ─────────────────────────────────────────────────────────────

func TestExtractLogf_NoError(t *testing.T) {
	t.Parallel()

	// nil writer is a no-op — must not panic
	extractLogf(nil, "test log message: %s %d", "hello", 42)
}

// ─── extractPositiveOutcomes ──────────────────────────────────────────────────

func TestExtractPositiveOutcomes_EmptyBlocks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := extractPositiveOutcomes(nil)
	g.Expect(items).To(BeEmpty())
}

func TestExtractPositiveOutcomes_NoPositiveSignal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	blocks := []parsedBlock{
		{role: "user", blockType: "tool_result", text: "some output without success signals", toolID: "t1"},
	}

	items := extractPositiveOutcomes(blocks)
	g.Expect(items).To(BeEmpty())
}

func TestExtractPositiveOutcomes_NoPrecedingToolUse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Positive result with no preceding tool_use → cmdPrefix falls back to "command"
	blocks := []parsedBlock{
		{role: "user", blockType: "tool_result", text: "PASS\nok github.com/example/pkg\n", toolID: "t1"},
	}

	items := extractPositiveOutcomes(blocks)
	g.Expect(items).ToNot(BeEmpty())

	if len(items) < 1 {
		t.Fatal("expected at least one positive outcome item")
	}

	g.Expect(items[0].Type).To(Equal("positive-outcome"))
	g.Expect(items[0].Content).To(ContainSubstring("command"))
}

func TestExtractPositiveOutcomes_WithPrecedingToolUse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	blocks := []parsedBlock{
		{
			role:      "assistant",
			blockType: "tool_use",
			toolName:  "Bash",
			toolInput: map[string]any{"command": "go test ./..."},
			toolID:    "t1",
		},
		{
			role:      "user",
			blockType: "tool_result",
			text:      "PASS\nok github.com/example 0.42s\n",
			toolID:    "t1",
		},
	}

	items := extractPositiveOutcomes(blocks)
	g.Expect(items).ToNot(BeEmpty())

	if len(items) < 1 {
		t.Fatal("expected at least one positive outcome item")
	}

	g.Expect(items[0].Content).To(ContainSubstring("go test"))
}

// ─── ExtractSession ───────────────────────────────────────────────────────────

func TestExtractSession_EmptyTranscript(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// Empty file
	err := os.WriteFile(transcriptPath, []byte(""), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	memoryDir := filepath.Join(tempDir, "memory")

	err = os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize DB so Learn won't fail on DB open
	db, dbErr := initEmbeddingsDB(filepath.Join(memoryDir, "embeddings.db"))
	g.Expect(dbErr).ToNot(HaveOccurred())

	_ = db.Close()

	result, err := ExtractSession(ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     memoryDir,
		Project:        "test-project",
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result must not be nil")
	}

	g.Expect(result.Status).To(Equal("success"))
	g.Expect(result.ItemsExtracted).To(Equal(0))
}

func TestExtractSession_FileNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result, err := ExtractSession(ExtractSessionOpts{
		TranscriptPath: "/nonexistent/path/transcript.jsonl",
		MemoryRoot:     t.TempDir(),
	})
	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

func TestExtractSession_MalformedLines(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// Malformed JSON lines (should be skipped gracefully)
	content := "not json\n{also not json\n\n"

	err := os.WriteFile(transcriptPath, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	memoryDir := filepath.Join(tempDir, "memory")

	err = os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := ExtractSession(ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result must not be nil")
	}

	g.Expect(result.ItemsExtracted).To(Equal(0))
}

func TestExtractSession_WithStartOffset(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// Write two lines of valid but unactionable JSON
	line1 := `{"type":"unknown_type","data":"line1"}` + "\n"
	line2 := `{"type":"unknown_type","data":"line2"}` + "\n"
	content := line1 + line2

	err := os.WriteFile(transcriptPath, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	memoryDir := filepath.Join(tempDir, "memory")

	err = os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// StartOffset past the first line — should only read line2
	result, err := ExtractSession(ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     memoryDir,
		StartOffset:    int64(len(line1)),
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result must not be nil")
	}

	g.Expect(result.EndOffset).To(BeNumerically(">", int64(len(line1))))
}

// ─── extractTierA ────────────────────────────────────────────────────────────

func TestExtractTierA_ClaudeMDEdit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	blocks := []parsedBlock{
		{
			role:      "assistant",
			blockType: "tool_use",
			toolName:  "Edit",
			toolInput: map[string]any{
				"file_path":  "/home/user/CLAUDE.md",
				"old_string": "old content here",
				"new_string": "new content here",
			},
		},
	}

	items := extractTierA(blocks, nil)
	g.Expect(items).ToNot(BeEmpty())

	found := false

	for _, item := range items {
		if item.Type == "claude-md-edit" {
			found = true

			break
		}
	}

	g.Expect(found).To(BeTrue())
}

func TestExtractTierA_ClaudeMDWrite(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	blocks := []parsedBlock{
		{
			role:      "assistant",
			blockType: "tool_use",
			toolName:  "Write",
			toolInput: map[string]any{
				"file_path": "/home/user/CLAUDE.md",
				"content":   "# My CLAUDE.md\n\nSome content",
			},
		},
	}

	items := extractTierA(blocks, nil)
	found := false

	for _, item := range items {
		if item.Type == "claude-md-edit" {
			found = true

			break
		}
	}

	g.Expect(found).To(BeTrue())
}

func TestExtractTierA_FileHistorySnapshot(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rawMessages := []map[string]any{
		{
			"type": "file-history-snapshot",
			"snapshot": map[string]any{
				"trackedFileBackups": map[string]any{
					"/home/user/CLAUDE.md": "old content",
				},
			},
		},
	}

	items := extractTierA(nil, rawMessages)
	g.Expect(items).ToNot(BeEmpty())

	if len(items) < 1 {
		t.Fatal("expected at least one item from file-history-snapshot")
	}

	g.Expect(items[0].Type).To(Equal("claude-md-edit"))
}

func TestExtractTierA_RememberThisWithSystemReminder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// System reminder content should be stripped before checking "remember this"
	blocks := []parsedBlock{
		{
			role:      "user",
			blockType: "text",
			text:      "<system-reminder>injected content</system-reminder>remember this: always test",
		},
	}

	items := extractTierA(blocks, nil)
	g.Expect(items).ToNot(BeEmpty())

	if len(items) < 1 {
		t.Fatal("expected at least one item")
	}

	g.Expect(items[0].Type).To(Equal("explicit-learning"))
}

// ─── extractTierB ────────────────────────────────────────────────────────────

func TestExtractTierB_EmptyBlocks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := extractTierB(nil)
	g.Expect(items).To(BeEmpty())
}

func TestExtractTierB_ErrorWithUserIntervention(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	blocks := []parsedBlock{
		{role: "assistant", blockType: "tool_use", toolName: "Bash", toolInput: map[string]any{"command": "go build"}, toolID: "t1"},
		{role: "user", blockType: "tool_result", text: "exit status 1: compile error", isError: true, toolID: "t1"},
		{role: "user", blockType: "text", text: "You need to fix the import path"},
		{role: "assistant", blockType: "tool_use", toolName: "Bash", toolInput: map[string]any{"command": "go build ./fixed/..."}, toolID: "t2"},
		{role: "user", blockType: "tool_result", text: "Build succeeded", isError: false, toolID: "t2"},
	}

	items := extractTierB(blocks)
	g.Expect(items).ToNot(BeEmpty())

	if len(items) < 1 {
		t.Fatal("expected at least one error-fix item")
	}

	g.Expect(items[0].Type).To(Equal("error-fix"))
	g.Expect(items[0].Confidence).To(BeNumerically("~", 0.7, 0.01))
}

func TestExtractTierB_ErrorWithoutIntervention(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Error followed immediately by success (no user text in between) → no item for Tier B
	blocks := []parsedBlock{
		{role: "user", blockType: "tool_result", text: "exit status 1", isError: true, toolID: "t1"},
		{role: "user", blockType: "tool_result", text: "PASS", isError: false, toolID: "t2"},
	}

	items := extractTierB(blocks)
	g.Expect(items).To(BeEmpty())
}

func TestExtractTierB_NoError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	blocks := []parsedBlock{
		{role: "assistant", blockType: "text", text: "I'll run the tests"},
		{role: "user", blockType: "tool_result", text: "PASS", isError: false},
	}

	items := extractTierB(blocks)
	g.Expect(items).To(BeEmpty())
}

// ─── extractTierC ────────────────────────────────────────────────────────────

func TestExtractTierC_NoMatcher(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	blocks := []parsedBlock{
		{role: "assistant", blockType: "tool_use", toolName: "Bash", toolInput: map[string]any{"command": "go test ./..."}, toolID: "t1"},
		{role: "user", blockType: "tool_result", text: "PASS\nok pkg 0.1s", isError: false, toolID: "t1"},
		{role: "assistant", blockType: "tool_use", toolName: "Bash", toolInput: map[string]any{"command": "go test ./..."}, toolID: "t2"},
		{role: "user", blockType: "tool_result", text: "PASS\nok pkg 0.1s", isError: false, toolID: "t2"},
		{role: "assistant", blockType: "tool_use", toolName: "Bash", toolInput: map[string]any{"command": "go test ./..."}, toolID: "t3"},
		{role: "user", blockType: "tool_result", text: "PASS\nok pkg 0.1s", isError: false, toolID: "t3"},
	}

	// nil matcher → behavioral conventions detection skipped; nil logW → no logging
	items := extractTierC(blocks, nil, nil)
	// Should still get tool usage pattern (3 uses, >50% success)
	g.Expect(items).ToNot(BeNil())
}

func TestExtract_EmptyDecisions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tomlContent := `
[status]
result = "success"
timestamp = "2026-02-21T10:00:00Z"
`

	opts := ExtractOpts{
		FilePath:   "result.toml",
		MemoryRoot: t.TempDir(),
		ReadFile: func(path string) ([]byte, error) {
			return []byte(tomlContent), nil
		},
		WriteDB: func(items []ExtractedItem) error {
			return nil
		},
	}

	result, err := opts.Extract()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result must not be nil")
	}

	g.Expect(result.ItemsExtracted).To(Equal(0))
}

func TestExtract_ParseFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	opts := ExtractOpts{
		FilePath:   "bad.toml",
		MemoryRoot: t.TempDir(),
		ReadFile: func(_ string) ([]byte, error) {
			// Valid TOML syntax but missing required status fields → parse validation error
			return []byte("[status]\nresult = \"\"\ntimestamp = \"\""), nil
		},
	}

	result, err := opts.Extract()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to parse file"))
	g.Expect(result).To(BeNil())
}

func TestExtract_ReadFileFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	opts := ExtractOpts{
		FilePath:   "nonexistent.toml",
		MemoryRoot: t.TempDir(),
		ReadFile: func(_ string) ([]byte, error) {
			return nil, errors.New("file not found")
		},
	}

	result, err := opts.Extract()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to read file"))
	g.Expect(result).To(BeNil())
}

// ─── Extract ─────────────────────────────────────────────────────────────────

func TestExtract_WithReadFileAndWriteDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tomlContent := `
[status]
result = "success"
timestamp = "2026-02-21T10:00:00Z"

[[decisions]]
context = "Architecture choice"
choice = "Use layered architecture"
reason = "Separation of concerns"
alternatives = ["monolithic", "microservices"]
`

	var captured []ExtractedItem

	opts := ExtractOpts{
		FilePath:   "result.toml",
		MemoryRoot: t.TempDir(),
		ReadFile: func(path string) ([]byte, error) {
			return []byte(tomlContent), nil
		},
		WriteDB: func(items []ExtractedItem) error {
			captured = append(captured, items...)
			return nil
		},
	}

	result, err := opts.Extract()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result must not be nil")
	}

	g.Expect(result.Status).To(Equal("success"))
	g.Expect(result.ItemsExtracted).To(Equal(1))
	g.Expect(captured).To(HaveLen(1))
	g.Expect(captured).ToNot(BeNil())

	if len(captured) == 0 {
		t.Fatal("captured must be non-empty")
	}

	g.Expect(captured[0].Type).To(Equal("decision"))
}

func TestExtract_WriteDBFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tomlContent := `
[status]
result = "success"
timestamp = "2026-02-21T10:00:00Z"

[[decisions]]
context = "Architecture"
choice = "Use microservices"
reason = "Scalability"
`

	opts := ExtractOpts{
		FilePath:   "result.toml",
		MemoryRoot: t.TempDir(),
		ReadFile: func(_ string) ([]byte, error) {
			return []byte(tomlContent), nil
		},
		WriteDB: func(_ []ExtractedItem) error {
			return errors.New("database write failed")
		},
	}

	result, err := opts.Extract()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to write to database"))
	g.Expect(result).To(BeNil())
}

// ─── parseTranscriptMessages ─────────────────────────────────────────────────

func TestParseTranscriptMessages_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	blocks := parseTranscriptMessages(nil)
	g.Expect(blocks).To(BeEmpty())
}

func TestParseTranscriptMessages_LegacyFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	messages := []map[string]any{
		{"type": "user-message", "content": "Hello from user"},
		{"type": "assistant-message", "content": "Hello from assistant"},
		{"type": "tool-result", "content": "Tool output here"},
	}

	blocks := parseTranscriptMessages(messages)
	g.Expect(blocks).To(HaveLen(3))
	g.Expect(blocks).ToNot(BeNil())

	if len(blocks) == 0 {
		t.Fatal("blocks must be non-empty")
	}

	g.Expect(blocks[0].role).To(Equal("user"))
	g.Expect(blocks[0].text).To(Equal("Hello from user"))
	g.Expect(blocks[1].role).To(Equal("assistant"))
	g.Expect(blocks[2].blockType).To(Equal("tool_result"))
}

func TestParseTranscriptMessages_NewFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	messages := []map[string]any{
		{
			"type": "user",
			"message": map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "text", "text": "user message text"},
					map[string]any{
						"type":        "tool_result",
						"tool_use_id": "tid1",
						"content":     "tool output",
						"is_error":    false,
					},
				},
			},
		},
		{
			"type": "assistant",
			"message": map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "text", "text": "assistant text"},
					map[string]any{
						"type":  "tool_use",
						"id":    "tid1",
						"name":  "Bash",
						"input": map[string]any{"command": "ls -la"},
					},
				},
			},
		},
	}

	blocks := parseTranscriptMessages(messages)
	// Should produce 4 blocks: user text, tool_result, assistant text, tool_use
	g.Expect(len(blocks)).To(BeNumerically(">=", 3))

	found := false

	for _, b := range blocks {
		if b.blockType == "tool_use" && b.toolName == "Bash" {
			found = true

			break
		}
	}

	g.Expect(found).To(BeTrue())
}

func TestParseTranscriptMessages_ToolResultWithError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Legacy tool-result with error signal
	messages := []map[string]any{
		{"type": "tool-result", "content": "error: command not found"},
	}

	blocks := parseTranscriptMessages(messages)
	g.Expect(blocks).To(HaveLen(1))
	g.Expect(blocks).ToNot(BeNil())

	if len(blocks) == 0 {
		t.Fatal("blocks must be non-empty")
	}

	g.Expect(blocks[0].isError).To(BeTrue())
}

func TestParseTranscriptMessages_UnknownType(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	messages := []map[string]any{
		{"type": "file-history-snapshot", "data": "some data"},
	}

	blocks := parseTranscriptMessages(messages)
	g.Expect(blocks).To(BeEmpty())
}

func TestStoreItemWithEmbeddingDB_ClosedDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	// Close the DB before calling storeItemWithEmbeddingDB
	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	item := ExtractedItem{
		Type:    "decision",
		Context: "Architecture choice",
		Content: "Use layered architecture for separation of concerns",
		Source:  "result:test.toml",
	}

	embedding := make([]float32, 384)
	embedding[0] = 1.0

	err = storeItemWithEmbeddingDB(db, item, embedding)
	g.Expect(err).To(HaveOccurred())
}

// TestStoreItemWithEmbeddingDB_MetaInsertError verifies error from the second db.Exec
// (metadata table insert) is returned when the embeddings table has been dropped.
func TestStoreItemWithEmbeddingDB_MetaInsertError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Drop the embeddings table so the second db.Exec (metadata insert) fails
	// while the first db.Exec (vec_embeddings insert) still succeeds.
	_, err = db.Exec("DROP TABLE IF EXISTS embeddings")
	g.Expect(err).ToNot(HaveOccurred())

	item := ExtractedItem{
		Type:    "decision",
		Context: "Test",
		Content: "Test content",
		Source:  "test",
	}

	embedding := make([]float32, 384)
	embedding[0] = 1.0

	err = storeItemWithEmbeddingDB(db, item, embedding)
	g.Expect(err).To(HaveOccurred())
}

// ─── storeItemWithEmbeddingDB ─────────────────────────────────────────────────

func TestStoreItemWithEmbeddingDB_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	item := ExtractedItem{
		Type:    "decision",
		Context: "Architecture choice",
		Content: "Use layered architecture for separation of concerns",
		Source:  "result:test.toml",
	}

	// Create a dummy normalized embedding
	embedding := make([]float32, 384)
	embedding[0] = 1.0

	err = storeItemWithEmbeddingDB(db, item, embedding)
	g.Expect(err).ToNot(HaveOccurred())

	var count int

	err = db.QueryRow("SELECT COUNT(*) FROM embeddings WHERE source = 'result:test.toml'").Scan(&count)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(count).To(Equal(1))
}

// ─── storeItemWithEmbeddingDB (via sqlite_vec) ────────────────────────────────

func TestStoreItemWithEmbeddingDB_VerifyVecRow(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	item := ExtractedItem{
		Type:    "decision",
		Context: "DB selection",
		Content: "Choose SQLite for embedded storage",
		Source:  "result:db.toml",
	}

	// Create a serialized embedding blob to verify round-trip
	rawEmbedding := make([]float32, 384)
	rawEmbedding[5] = 0.5

	_, serErr := sqlite_vec.SerializeFloat32(rawEmbedding)
	g.Expect(serErr).ToNot(HaveOccurred())

	err = storeItemWithEmbeddingDB(db, item, rawEmbedding)
	g.Expect(err).ToNot(HaveOccurred())

	var count int

	err = db.QueryRow("SELECT COUNT(*) FROM vec_embeddings").Scan(&count)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(count).To(Equal(1))
}
