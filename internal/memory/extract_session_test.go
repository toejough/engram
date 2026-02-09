package memory_test

import (
	"encoding/json"
	"os"
	"path/filepath"
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

	// Create transcript with error→fix sequence
	messages := []map[string]any{
		{
			"type":    "tool-result",
			"content": "Error: undefined: fmt.Printl",
		},
		{
			"type":    "assistant-message",
			"content": "I need to fix the typo - should be fmt.Println",
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

	// Verify learning was stored (check index.md exists)
	indexPath := filepath.Join(memoryRoot, "index.md")
	_, err = os.Stat(indexPath)
	g.Expect(err).ToNot(HaveOccurred(), "Expected index.md to be created")
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
		`{"type":"assistant-message","content":"I need to define the variable first"}`,
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
