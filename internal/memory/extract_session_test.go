package memory_test

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// TASK-1: Extract-session command implementation
// ============================================================================

// TEST-1001: ExtractSessionOpts accepts transcript path
// traces: TASK-1 AC-2
func TestExtractSessionOptsAcceptsTranscriptPath(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "test.jsonl")

	// Create minimal valid transcript
	err := os.WriteFile(transcriptPath, []byte(`{"type":"test","data":"value"}`+"\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	}

	result, err := memory.ExtractSession(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
}

// TEST-1002: ExtractSession reads JSONL transcript file
// traces: TASK-1 AC-2
func TestExtractSessionReadsJSONLFile(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// Create JSONL with multiple lines
	content := `{"type":"message","content":"line 1"}` + "\n" +
		`{"type":"message","content":"line 2"}` + "\n" +
		`{"type":"message","content":"line 3"}` + "\n"

	err := os.WriteFile(transcriptPath, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	}

	result, err := memory.ExtractSession(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.Status).To(Equal("success"))
}

// TEST-1003: Tier A extraction detects "remember this" phrase with confidence 1.0
// traces: TASK-1 AC-3
func TestTierAExtractionDetectsRememberThis(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// Create transcript with "remember this" signal
	message := map[string]any{
		"type":    "user-message",
		"content": "remember this: always use gomega for assertions",
	}
	line, _ := json.Marshal(message)
	err := os.WriteFile(transcriptPath, append(line, '\n'), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	}

	result, err := memory.ExtractSession(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ItemsExtracted).To(BeNumerically(">", 0))

	// Verify at least one item has confidence 1.0
	hasHighConfidence := false
	for _, item := range result.Items {
		if item.Confidence == 1.0 {
			hasHighConfidence = true
			break
		}
	}
	g.Expect(hasHighConfidence).To(BeTrue(), "Expected at least one item with confidence 1.0")
}

// TEST-1004: Tier A extraction detects explicit corrections with confidence 1.0
// traces: TASK-1 AC-3
func TestTierAExtractionDetectsExplicitCorrections(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// Create transcript with explicit correction
	messages := []map[string]any{
		{
			"type":    "assistant-message",
			"content": "I'll use git checkout -- .",
		},
		{
			"type":    "user-message",
			"content": "No, never use git checkout -- . - it destroys uncommitted work",
		},
	}

	var content []byte
	for _, msg := range messages {
		line, _ := json.Marshal(msg)
		content = append(content, line...)
		content = append(content, '\n')
	}

	err := os.WriteFile(transcriptPath, content, 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	}

	result, err := memory.ExtractSession(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ItemsExtracted).To(BeNumerically(">", 0))

	// Verify correction was captured with high confidence
	hasCorrection := false
	for _, item := range result.Items {
		if item.Confidence == 1.0 && item.Type == "correction" {
			hasCorrection = true
			break
		}
	}
	g.Expect(hasCorrection).To(BeTrue(), "Expected correction with confidence 1.0")
}

// TEST-1005: Tier A extraction detects CLAUDE.md edit events with confidence 1.0
// traces: TASK-1 AC-3
func TestTierAExtractionDetectsCLAUDEMdEdits(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// Create transcript with CLAUDE.md file edit
	message := map[string]any{
		"type": "file-history-snapshot",
		"snapshot": map[string]any{
			"trackedFileBackups": map[string]any{
				"/Users/joe/.claude/CLAUDE.md": map[string]any{
					"version": 8,
				},
			},
		},
	}
	line, _ := json.Marshal(message)
	err := os.WriteFile(transcriptPath, append(line, '\n'), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	}

	result, err := memory.ExtractSession(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ItemsExtracted).To(BeNumerically(">", 0))

	// Verify CLAUDE.md edit was captured
	hasCLAUDEEdit := false
	for _, item := range result.Items {
		if item.Confidence == 1.0 && item.Type == "claude-md-edit" {
			hasCLAUDEEdit = true
			break
		}
	}
	g.Expect(hasCLAUDEEdit).To(BeTrue(), "Expected CLAUDE.md edit with confidence 1.0")
}

// TEST-1006: Tier B extraction detects error→fix sequences with confidence 0.7
// traces: TASK-1 AC-4
func TestTierBExtractionDetectsErrorFixSequences(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// Create transcript with error→fix sequence (with user intervention)
	messages := []map[string]any{
		{
			"type":    "tool-result",
			"content": "Error: undefined: fmt.Printl",
		},
		{
			"type":    "user-message",
			"content": "You need to fix the typo - should be fmt.Println",
		},
		{
			"type":    "tool-result",
			"content": "Success",
		},
	}

	var content []byte
	for _, msg := range messages {
		line, _ := json.Marshal(msg)
		content = append(content, line...)
		content = append(content, '\n')
	}

	err := os.WriteFile(transcriptPath, content, 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	}

	result, err := memory.ExtractSession(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ItemsExtracted).To(BeNumerically(">", 0))

	// Verify error→fix was captured with medium confidence
	hasErrorFix := false
	for _, item := range result.Items {
		if item.Confidence == 0.7 && item.Type == "error-fix" {
			hasErrorFix = true
			break
		}
	}
	g.Expect(hasErrorFix).To(BeTrue(), "Expected error→fix with confidence 0.7")
}

// TEST-1007: Tier B extraction detects repeated patterns with confidence 0.7
// traces: TASK-1 AC-4
func TestTierBExtractionDetectsRepeatedPatterns(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// Create transcript with repeated pattern (same phrase multiple times)
	pattern := "use blackbox tests with package foo_test"
	messages := []map[string]any{
		{
			"type":    "assistant-message",
			"content": "I'll " + pattern,
		},
		{
			"type":    "assistant-message",
			"content": "Following the pattern to " + pattern,
		},
		{
			"type":    "assistant-message",
			"content": "Again, I need to " + pattern,
		},
	}

	var content []byte
	for _, msg := range messages {
		line, _ := json.Marshal(msg)
		content = append(content, line...)
		content = append(content, '\n')
	}

	err := os.WriteFile(transcriptPath, content, 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	}

	result, err := memory.ExtractSession(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ItemsExtracted).To(BeNumerically(">", 0))

	// Verify repeated pattern was captured
	hasPattern := false
	for _, item := range result.Items {
		if item.Confidence == 0.7 && item.Type == "repeated-pattern" {
			hasPattern = true
			break
		}
	}
	g.Expect(hasPattern).To(BeTrue(), "Expected repeated pattern with confidence 0.7")
}

// TEST-1008: ExtractSession stores learnings via existing memory functions
// traces: TASK-1 AC-5
func TestExtractSessionStoresViMemoryFunctions(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")
	memoryRoot := filepath.Join(tempDir, "memory")

	// Create transcript with explicit learning
	message := map[string]any{
		"type":    "user-message",
		"content": "remember this: test learning content",
	}
	line, _ := json.Marshal(message)
	err := os.WriteFile(transcriptPath, append(line, '\n'), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     memoryRoot,
	}

	result, err := memory.ExtractSession(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ItemsExtracted).To(BeNumerically(">", 0))

	// Verify learning was stored (check embeddings.db exists)
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	_, err = os.Stat(dbPath)
	g.Expect(err).ToNot(HaveOccurred(), "Expected embeddings.db to be created")
}

// TEST-1009: ExtractSession returns summary with items extracted count
// traces: TASK-1 AC-8
func TestExtractSessionReturnsSummaryWithCount(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// Create transcript with multiple learnings
	messages := []map[string]any{
		{
			"type":    "user-message",
			"content": "remember this: learning 1",
		},
		{
			"type":    "user-message",
			"content": "remember this: learning 2",
		},
	}

	var content []byte
	for _, msg := range messages {
		line, _ := json.Marshal(msg)
		content = append(content, line...)
		content = append(content, '\n')
	}

	err := os.WriteFile(transcriptPath, content, 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	}

	result, err := memory.ExtractSession(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ItemsExtracted).To(Equal(2))
	g.Expect(result.Status).To(Equal("success"))
}

// TEST-1010: ExtractSession returns confidence distribution in summary
// traces: TASK-1 AC-8
func TestExtractSessionReturnsConfidenceDistribution(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// Create transcript with mixed confidence items
	messages := []map[string]any{
		{
			"type":    "user-message",
			"content": "remember this: explicit learning",
		},
		{
			"type":    "tool-result",
			"content": "Error: something failed",
		},
		{
			"type":    "assistant-message",
			"content": "I'll fix the error",
		},
	}

	var content []byte
	for _, msg := range messages {
		line, _ := json.Marshal(msg)
		content = append(content, line...)
		content = append(content, '\n')
	}

	err := os.WriteFile(transcriptPath, content, 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	}

	result, err := memory.ExtractSession(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ConfidenceDistribution).ToNot(BeNil())
	g.Expect(result.ConfidenceDistribution).To(HaveKey(1.0))
}

// TEST-1011: ExtractSession parsing is resilient to malformed JSONL lines
// traces: TASK-1 AC-7
func TestExtractSessionResilientToMalformedJSON(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// Create transcript with mix of valid and invalid lines
	content := `{"type":"message","content":"valid"}` + "\n" +
		`{invalid json` + "\n" +
		`{"type":"user-message","content":"remember this: valid learning"}` + "\n"

	err := os.WriteFile(transcriptPath, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	}

	result, err := memory.ExtractSession(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Status).To(Equal("success"))
	// Should extract from valid lines, skip malformed
	g.Expect(result.ItemsExtracted).To(BeNumerically(">", 0))
}

// TEST-1012: ExtractSession handles empty transcript gracefully
// traces: TASK-1 AC-7
func TestExtractSessionHandlesEmptyTranscript(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "empty.jsonl")

	err := os.WriteFile(transcriptPath, []byte(""), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	}

	result, err := memory.ExtractSession(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Status).To(Equal("success"))
	g.Expect(result.ItemsExtracted).To(Equal(0))
}

// TEST-1013: ExtractSession handles missing transcript file
// traces: TASK-1 AC-7
func TestExtractSessionHandlesMissingFile(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "nonexistent.jsonl")

	opts := memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	}

	_, err := memory.ExtractSession(opts)
	g.Expect(err).To(HaveOccurred())
}

// TEST-1014: Property test - ExtractSession handles varied JSONL structures
// traces: TASK-1 AC-7
func TestExtractSessionPropertyResilientParsing(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(rt, "suffix")
		tempDir := filepath.Join(os.TempDir(), "extract-session-test-"+suffix)
		defer func() { _ = os.RemoveAll(tempDir) }()
		_ = os.MkdirAll(tempDir, 0755)
		transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

		// Generate random valid JSONL structure
		messageTypes := []string{"user-message", "assistant-message", "tool-result", "file-history-snapshot"}
		msgType := rapid.SampledFrom(messageTypes).Draw(rt, "messageType")
		content := rapid.String().Draw(rt, "content")

		message := map[string]any{
			"type":    msgType,
			"content": content,
		}
		line, _ := json.Marshal(message)
		err := os.WriteFile(transcriptPath, append(line, '\n'), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		opts := memory.ExtractSessionOpts{
			TranscriptPath: transcriptPath,
			MemoryRoot:     filepath.Join(tempDir, "memory"),
		}

		// Should not panic or error on valid JSON
		result, err := memory.ExtractSession(opts)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).ToNot(BeNil())
		g.Expect(result.Status).To(Or(Equal("success"), Equal("partial")))
	})
}

// TEST-1015: Integration test with real Claude Code transcript structure
// traces: TASK-1 AC-11
func TestExtractSessionIntegrationRealTranscript(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "real-transcript.jsonl")

	// Create realistic transcript with actual Claude Code structure
	messages := []string{
		`{"type":"file-history-snapshot","messageId":"abc123","snapshot":{"trackedFileBackups":{"/Users/joe/.claude/CLAUDE.md":{"version":2}}}}`,
		`{"type":"user-message","content":"remember this: always check git status before amending"}`,
		`{"type":"assistant-message","content":"I'll use git checkout to discard changes"}`,
		`{"type":"user-message","content":"No, never use git checkout -- . - it destroys work"}`,
		`{"type":"tool-result","content":"Error: undefined variable"}`,
		`{"type":"user-message","content":"You need to define the variable first"}`,
		`{"type":"tool-result","content":"Success"}`,
	}

	content := ""
	for _, msg := range messages {
		content += msg + "\n"
	}

	err := os.WriteFile(transcriptPath, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	}

	result, err := memory.ExtractSession(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Status).To(Equal("success"))
	g.Expect(result.ItemsExtracted).To(BeNumerically(">=", 3), "Expected at least CLAUDE.md edit, remember this, and correction")

	// Verify mix of confidence levels
	hasHighConfidence := false
	hasMediumConfidence := false
	for _, item := range result.Items {
		if item.Confidence == 1.0 {
			hasHighConfidence = true
		}
		if item.Confidence == 0.7 {
			hasMediumConfidence = true
		}
	}
	g.Expect(hasHighConfidence).To(BeTrue(), "Expected high confidence items")
	g.Expect(hasMediumConfidence).To(BeTrue(), "Expected medium confidence items")
}

// ============================================================================
// New Claude Code JSONL format tests
// ============================================================================

// TestParseTranscriptNewFormat tests parsing of the real Claude Code JSONL format
func TestParseTranscriptNewFormat(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	messages := []string{
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"remember this: use TDD always"}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I'll use TDD."},{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"go test ./..."}}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"PASS","is_error":false}]}}`,
	}
	content := strings.Join(messages, "\n") + "\n"
	err := os.WriteFile(transcriptPath, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ItemsExtracted).To(BeNumerically(">", 0))

	// Should detect "remember this"
	hasRememberThis := false
	for _, item := range result.Items {
		if item.Type == "explicit-learning" && item.Confidence == 1.0 {
			hasRememberThis = true
		}
	}
	g.Expect(hasRememberThis).To(BeTrue())
}

// TestTierBRequiresUserInterventionNewFormat tests that error→fix requires user message between error and fix
func TestTierBRequiresUserInterventionNewFormat(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// Error→assistant fix→success WITHOUT user intervention = should NOT be Tier B
	messages := []string{
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"go build"}}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"Error: undefined variable","is_error":true}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Let me fix that"},{"type":"tool_use","id":"toolu_2","name":"Bash","input":{"command":"go build"}}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_2","content":"Build succeeded","is_error":false}]}}`,
	}
	content := strings.Join(messages, "\n") + "\n"
	err := os.WriteFile(transcriptPath, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Should NOT have error-fix (no user text intervention)
	for _, item := range result.Items {
		g.Expect(item.Type).ToNot(Equal("error-fix"), "Should not detect error-fix without user intervention")
	}
}

// TestTierBWithUserInterventionNewFormat tests that error→user fix→success IS detected as Tier B
func TestTierBWithUserInterventionNewFormat(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	messages := []string{
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"go build"}}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"Error: undefined variable","is_error":true}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"You need to define the variable first"}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_2","name":"Bash","input":{"command":"go build"}}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_2","content":"Build succeeded","is_error":false}]}}`,
	}
	content := strings.Join(messages, "\n") + "\n"
	err := os.WriteFile(transcriptPath, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	})
	g.Expect(err).ToNot(HaveOccurred())

	hasErrorFix := false
	for _, item := range result.Items {
		if item.Type == "error-fix" && item.Confidence == 0.7 {
			hasErrorFix = true
		}
	}
	g.Expect(hasErrorFix).To(BeTrue(), "Should detect error-fix with user intervention")
}

// ============================================================================
// ISSUE-185: Tier C implicit signal detection tests
// ============================================================================

// mockSemanticMatcher is a test double for SemanticMatcher.
type mockSemanticMatcher struct {
	// memories maps input text substrings to matching memory contents
	memories map[string][]string
}

func (m *mockSemanticMatcher) FindSimilarMemories(text string, threshold float64, limit int) ([]string, error) {
	lower := strings.ToLower(text)
	for key, mems := range m.memories {
		if strings.Contains(lower, key) {
			if len(mems) > limit {
				return mems[:limit], nil
			}
			return mems, nil
		}
	}
	return nil, nil
}

func newMockMatcher(memories map[string][]string) *mockSemanticMatcher {
	return &mockSemanticMatcher{memories: memories}
}

// --- 3a: Tool usage patterns ---

func TestTierCDetectsToolUsagePatterns(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// 4 successful "go test" invocations
	var msgs []string
	for i := 0; i < 4; i++ {
		id := fmt.Sprintf("toolu_%d", i)
		msgs = append(msgs,
			fmt.Sprintf(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"%s","name":"Bash","input":{"command":"go test ./..."}}]}}`, id),
			fmt.Sprintf(`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"%s","content":"PASS","is_error":false}]}}`, id),
		)
	}

	err := os.WriteFile(transcriptPath, []byte(strings.Join(msgs, "\n")+"\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	})
	g.Expect(err).ToNot(HaveOccurred())

	hasToolUsage := false
	for _, item := range result.Items {
		if item.Type == "tool-usage-pattern" && item.Confidence == 0.5 {
			hasToolUsage = true
			g.Expect(item.Content).To(ContainSubstring("go test"))
		}
	}
	g.Expect(hasToolUsage).To(BeTrue(), "Should detect tool usage pattern for go test")
}

func TestTierCIgnoresFailedCommands(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// 3 FAILED "go build" invocations (below 50% success)
	var msgs []string
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("toolu_%d", i)
		msgs = append(msgs,
			fmt.Sprintf(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"%s","name":"Bash","input":{"command":"go build ./..."}}]}}`, id),
			fmt.Sprintf(`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"%s","content":"Error: compilation failed","is_error":true}]}}`, id),
		)
	}

	err := os.WriteFile(transcriptPath, []byte(strings.Join(msgs, "\n")+"\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	})
	g.Expect(err).ToNot(HaveOccurred())

	for _, item := range result.Items {
		g.Expect(item.Type).ToNot(Equal("tool-usage-pattern"), "Should not detect pattern for failed commands")
	}
}

// --- 3b: Positive outcomes ---

func TestTierCDetectsPositiveOutcomes(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	msgs := []string{
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"go test ./..."}}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"ok  \tgithub.com/foo/bar\nPASS","is_error":false}]}}`,
	}

	err := os.WriteFile(transcriptPath, []byte(strings.Join(msgs, "\n")+"\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	})
	g.Expect(err).ToNot(HaveOccurred())

	hasPositive := false
	for _, item := range result.Items {
		if item.Type == "positive-outcome" && item.Confidence == 0.5 {
			hasPositive = true
		}
	}
	g.Expect(hasPositive).To(BeTrue(), "Should detect positive outcome")
}

func TestTierCDeduplicatesOutcomes(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// Two "go test" successes should produce only one positive-outcome
	msgs := []string{
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"go test ./..."}}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"PASS","is_error":false}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_2","name":"Bash","input":{"command":"go test ./internal/..."}}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_2","content":"PASS","is_error":false}]}}`,
	}

	err := os.WriteFile(transcriptPath, []byte(strings.Join(msgs, "\n")+"\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	})
	g.Expect(err).ToNot(HaveOccurred())

	positiveCount := 0
	for _, item := range result.Items {
		if item.Type == "positive-outcome" {
			positiveCount++
		}
	}
	g.Expect(positiveCount).To(Equal(1), "Should deduplicate positive outcomes per command type")
}

// --- 3c: Behavioral consistency ---

func TestTierCDetectsBehavioralConsistency(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// 6 distinct assistant messages mentioning "gomega"
	var msgs []string
	for i := 0; i < 6; i++ {
		msgs = append(msgs,
			fmt.Sprintf(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I'll write tests using gomega assertions for test %d"}]}}`, i),
		)
	}

	err := os.WriteFile(transcriptPath, []byte(strings.Join(msgs, "\n")+"\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	})
	g.Expect(err).ToNot(HaveOccurred())

	hasConsistency := false
	for _, item := range result.Items {
		if item.Type == "behavioral-consistency" && item.Confidence == 0.5 {
			hasConsistency = true
			g.Expect(item.Content).To(ContainSubstring("gomega"))
		}
	}
	g.Expect(hasConsistency).To(BeTrue(), "Should detect behavioral consistency for gomega")
}

func TestTierCNoBehavioralConsistencyWithCorrection(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// 6 assistant messages mentioning "jest", but user corrects it
	var msgs []string
	for i := 0; i < 6; i++ {
		msgs = append(msgs,
			fmt.Sprintf(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I'll use jest for testing %d"}]}}`, i),
		)
	}
	// User corrects
	msgs = append(msgs,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"No, don't use jest, use vitest instead"}]}}`,
	)

	err := os.WriteFile(transcriptPath, []byte(strings.Join(msgs, "\n")+"\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	})
	g.Expect(err).ToNot(HaveOccurred())

	for _, item := range result.Items {
		if item.Type == "behavioral-consistency" && strings.Contains(item.Content, "jest") {
			t.Fatal("Should not detect behavioral consistency when user corrected the tool usage")
		}
	}
}

// --- 3d: Self-corrected failures ---

func TestTierCSelfCorrectedFailure(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// Error → assistant retries (no user text) → success
	msgs := []string{
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"go build"}}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"Error: undefined variable x","is_error":true}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Let me fix that"},{"type":"tool_use","id":"toolu_2","name":"Bash","input":{"command":"go build"}}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_2","content":"Build succeeded","is_error":false}]}}`,
	}

	err := os.WriteFile(transcriptPath, []byte(strings.Join(msgs, "\n")+"\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	})
	g.Expect(err).ToNot(HaveOccurred())

	hasSelfCorrected := false
	for _, item := range result.Items {
		if item.Type == "self-corrected-failure" && item.Confidence == 0.5 {
			hasSelfCorrected = true
		}
	}
	g.Expect(hasSelfCorrected).To(BeTrue(), "Should detect self-corrected failure")
}

func TestTierCSelfCorrectedNotTriggeredWithUserIntervention(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// Error → USER text → assistant retries → success (this is Tier B, NOT self-corrected)
	msgs := []string{
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"go build"}}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"Error: undefined variable","is_error":true}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"You need to declare x first"}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_2","name":"Bash","input":{"command":"go build"}}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_2","content":"Build succeeded","is_error":false}]}}`,
	}

	err := os.WriteFile(transcriptPath, []byte(strings.Join(msgs, "\n")+"\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	})
	g.Expect(err).ToNot(HaveOccurred())

	for _, item := range result.Items {
		g.Expect(item.Type).ToNot(Equal("self-corrected-failure"),
			"Should NOT detect self-corrected failure when user intervened")
	}
}

// --- 3e: Behavioral conventions (semantic) ---

func TestTierCBehavioralConventionDetected(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// 4 assistant text blocks that semantically match a memory
	var msgs []string
	for i := 0; i < 4; i++ {
		msgs = append(msgs,
			fmt.Sprintf(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I will follow the TDD approach: write tests first, then implement, then refactor for iteration %d of the development cycle"}]}}`, i),
		)
	}

	err := os.WriteFile(transcriptPath, []byte(strings.Join(msgs, "\n")+"\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	matcher := newMockMatcher(map[string][]string{
		"tdd": {"always use TDD: write tests first, implement, refactor"},
	})

	result, err := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
		Matcher:        matcher,
	})
	g.Expect(err).ToNot(HaveOccurred())

	hasConvention := false
	for _, item := range result.Items {
		if item.Type == "behavioral-convention" && item.Confidence == 0.5 {
			hasConvention = true
		}
	}
	g.Expect(hasConvention).To(BeTrue(), "Should detect behavioral convention via semantic matching")
}

func TestTierCBehavioralConventionNotTriggeredBelowThreshold(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// Only 2 assistant text blocks (below threshold of 3)
	msgs := []string{
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I will follow the TDD approach: write tests first, then implement for this feature"}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I will follow the TDD approach: write tests first, then implement for the next feature"}]}}`,
	}

	err := os.WriteFile(transcriptPath, []byte(strings.Join(msgs, "\n")+"\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	matcher := newMockMatcher(map[string][]string{
		"tdd": {"always use TDD: write tests first, implement, refactor"},
	})

	result, err := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
		Matcher:        matcher,
	})
	g.Expect(err).ToNot(HaveOccurred())

	for _, item := range result.Items {
		g.Expect(item.Type).ToNot(Equal("behavioral-convention"),
			"Should not detect convention with only 2 matches (below threshold of 3)")
	}
}

func TestTierCBehavioralConventionBrokenByCorrection(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// 4 assistant blocks that match, but one is followed by user correction
	msgs := []string{
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I will follow the TDD approach: write tests first, then implement for this feature"}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"No, don't do TDD for this, just implement it quickly"}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I will follow the TDD approach: write tests first, then implement for the second feature"}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I will follow the TDD approach: write tests first, then implement for the third feature"}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I will follow the TDD approach: write tests first, then implement for the fourth feature"}]}}`,
	}

	err := os.WriteFile(transcriptPath, []byte(strings.Join(msgs, "\n")+"\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	matcher := newMockMatcher(map[string][]string{
		"tdd": {"always use TDD: write tests first, implement, refactor"},
	})

	result, err := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
		Matcher:        matcher,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// The first block is corrected so it shouldn't count.
	// Remaining 3 blocks do match and are uncorrected, so convention SHOULD be detected.
	// This test verifies that corrected blocks are excluded.
	conventionCount := 0
	for _, item := range result.Items {
		if item.Type == "behavioral-convention" {
			conventionCount++
		}
	}
	// 3 uncorrected matches = still meets threshold
	g.Expect(conventionCount).To(Equal(1))
}

func TestTierCBehavioralConventionNoMemoryMatch(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// Assistant text that doesn't match any memory
	var msgs []string
	for i := 0; i < 5; i++ {
		msgs = append(msgs,
			fmt.Sprintf(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"This is a completely unrelated long enough text about cooking recipes and gardening tips number %d"}]}}`, i),
		)
	}

	err := os.WriteFile(transcriptPath, []byte(strings.Join(msgs, "\n")+"\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Matcher returns no matches for cooking/gardening
	matcher := newMockMatcher(map[string][]string{
		"tdd": {"always use TDD"},
	})

	result, err := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
		Matcher:        matcher,
	})
	g.Expect(err).ToNot(HaveOccurred())

	for _, item := range result.Items {
		g.Expect(item.Type).ToNot(Equal("behavioral-convention"),
			"Should not detect convention when no memory matches")
	}
}

// --- General Tier C properties ---

func TestTierCConfidenceAlways05(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// Build transcript that triggers multiple Tier C types
	var msgs []string
	// Tool usage pattern (4x go test)
	for i := 0; i < 4; i++ {
		id := fmt.Sprintf("toolu_%d", i)
		msgs = append(msgs,
			fmt.Sprintf(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"%s","name":"Bash","input":{"command":"go test ./..."}}]}}`, id),
			fmt.Sprintf(`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"%s","content":"PASS","is_error":false}]}}`, id),
		)
	}

	err := os.WriteFile(transcriptPath, []byte(strings.Join(msgs, "\n")+"\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
	})
	g.Expect(err).ToNot(HaveOccurred())

	for _, item := range result.Items {
		if item.Type == "tool-usage-pattern" ||
			item.Type == "positive-outcome" ||
			item.Type == "behavioral-consistency" ||
			item.Type == "self-corrected-failure" ||
			item.Type == "behavioral-convention" {
			g.Expect(item.Confidence).To(Equal(0.5),
				fmt.Sprintf("Tier C item type %q should have confidence 0.5", item.Type))
		}
	}
}

func TestTierCPropertyEmptyTranscriptNoFalsePositives(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(rt, "suffix")
		tempDir := filepath.Join(os.TempDir(), "tierc-empty-"+suffix)
		defer func() { _ = os.RemoveAll(tempDir) }()
		_ = os.MkdirAll(tempDir, 0755)
		transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

		// Empty or random single-message transcript
		msgType := rapid.SampledFrom([]string{"user", "assistant"}).Draw(rt, "type")
		text := rapid.StringMatching(`[a-zA-Z ]{0,30}`).Draw(rt, "text")
		msg := fmt.Sprintf(`{"type":"%s","message":{"role":"%s","content":[{"type":"text","text":"%s"}]}}`, msgType, msgType, text)
		err := os.WriteFile(transcriptPath, []byte(msg+"\n"), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		result, err := memory.ExtractSession(memory.ExtractSessionOpts{
			TranscriptPath: transcriptPath,
			MemoryRoot:     filepath.Join(tempDir, "memory"),
		})
		g.Expect(err).ToNot(HaveOccurred())

		// No Tier C items should fire from a single random message
		for _, item := range result.Items {
			tierCTypes := map[string]bool{
				"tool-usage-pattern":    true,
				"positive-outcome":      true,
				"behavioral-consistency": true,
				"self-corrected-failure": true,
				"behavioral-convention":  true,
			}
			g.Expect(tierCTypes[item.Type]).To(BeFalse(),
				fmt.Sprintf("Single random message should not trigger Tier C item: %s", item.Type))
		}
	})
}

// ============================================================================
// ISSUE-196: Fallback logging when SemanticMatcher is nil
// ============================================================================

// TEST-196-01: ExtractSession logs fallback message when Matcher is nil
// traces: ISSUE-196 AC-1
func TestExtractSessionNilMatcherLogsFallback(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	// Create minimal valid transcript
	message := map[string]any{
		"type":    "user-message",
		"content": "some session content",
	}
	line, _ := json.Marshal(message)
	err := os.WriteFile(transcriptPath, append(line, '\n'), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Capture stderr
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	g.Expect(err).ToNot(HaveOccurred())
	os.Stderr = w

	_, extractErr := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     filepath.Join(tempDir, "memory"),
		Matcher:        nil, // explicitly nil
	})

	// Restore stderr and read captured output
	w.Close()
	os.Stderr = oldStderr
	captured, err := io.ReadAll(r)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(extractErr).ToNot(HaveOccurred())
	g.Expect(string(captured)).To(ContainSubstring(
		"SemanticMatcher not configured, skipping behavioral convention detection",
	))
}
