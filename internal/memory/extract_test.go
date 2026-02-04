package memory_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// Extract tests (TASK-4)
// ============================================================================

// TEST: Extract parses yield file and returns extracted items
// Traces to: TASK-4 AC-1, AC-2, AC-5, AC-7
func TestExtract_ParsesYieldFile(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	yieldContent := `
[yield]
type = "complete"
timestamp = "2026-02-04T10:30:00Z"

[payload]
summary = "Implemented the Extract function for memory extraction"
findings = ["Found existing embedding infrastructure", "Need to reuse ONNX model"]

[context]
phase = "tdd-green"
subphase = "implementation"
task = "TASK-4"
`

	opts := memory.ExtractOpts{
		FilePath:   "/test/yield.toml",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
		ReadFile: func(path string) ([]byte, error) {
			return []byte(yieldContent), nil
		},
	}

	result, err := opts.Extract()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.Status).To(Equal("success"))
	g.Expect(result.FilePath).To(Equal("/test/yield.toml"))
	g.Expect(result.ItemsExtracted).To(BeNumerically(">", 0))
}

// TEST: Extract parses result file with decisions
// Traces to: TASK-4 AC-1, AC-3, AC-7
func TestExtract_ParsesResultFileWithDecisions(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	resultContent := `
[status]
result = "success"
timestamp = "2026-02-04T10:45:00Z"

[[decisions]]
context = "Error handling strategy"
choice = "Use wrapped errors with context"
reason = "Provides clear error traces"
alternatives = ["Sentinel errors", "Error codes"]

[[decisions]]
context = "Database selection"
choice = "SQLite-vec"
reason = "Local, no server, supports vector search"
alternatives = ["PostgreSQL with pgvector", "Milvus"]

[context]
phase = "design"
subphase = "review"
task = "TASK-10"
`

	opts := memory.ExtractOpts{
		FilePath:   "/test/result.toml",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
		ReadFile: func(path string) ([]byte, error) {
			return []byte(resultContent), nil
		},
	}

	result, err := opts.Extract()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.Status).To(Equal("success"))
	g.Expect(result.ItemsExtracted).To(Equal(2)) // Two decisions

	// Verify decisions are extracted
	decisionItems := filterByType(result.Items, "decision")
	g.Expect(decisionItems).To(HaveLen(2))
	g.Expect(decisionItems[0].Context).To(Equal("Error handling strategy"))
	g.Expect(decisionItems[0].Content).To(ContainSubstring("wrapped errors"))
}

// TEST: Extract extracts learnings from yield payload
// Traces to: TASK-4 AC-4, AC-5
func TestExtract_ExtractsLearningsFromYieldPayload(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	yieldContent := `
[yield]
type = "complete"
timestamp = "2026-02-04T10:30:00Z"

[payload]
summary = "Completed the ONNX integration"
findings = ["Model downloads automatically", "384 dimensions work well"]
learnings = ["SQLite-vec requires vec0 extension", "Mean pooling gives good results"]

[context]
phase = "tdd-green"
subphase = "complete"
task = "TASK-52"
`

	opts := memory.ExtractOpts{
		FilePath:   "/test/yield-with-learnings.toml",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
		ReadFile: func(path string) ([]byte, error) {
			return []byte(yieldContent), nil
		},
	}

	result, err := opts.Extract()
	g.Expect(err).ToNot(HaveOccurred())

	// Should extract summary, findings, and learnings
	summaryItems := filterByType(result.Items, "summary")
	g.Expect(summaryItems).To(HaveLen(1))
	g.Expect(summaryItems[0].Content).To(ContainSubstring("ONNX integration"))

	findingItems := filterByType(result.Items, "finding")
	g.Expect(findingItems).To(HaveLen(2))

	learningItems := filterByType(result.Items, "learning")
	g.Expect(learningItems).To(HaveLen(2))
}

// TEST: Extract sets source field correctly for yield files
// Traces to: TASK-4 AC-8
func TestExtract_SetsSourceFieldForYieldFile(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	yieldContent := `
[yield]
type = "complete"
timestamp = "2026-02-04T10:30:00Z"

[payload]
summary = "Test summary"

[context]
phase = "tdd-green"
task = "TASK-4"
`

	mockWriteDBCalled := false
	var capturedItems []memory.ExtractedItem

	opts := memory.ExtractOpts{
		FilePath:   "/path/to/my-yield.toml",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
		ReadFile: func(path string) ([]byte, error) {
			return []byte(yieldContent), nil
		},
		WriteDB: func(items []memory.ExtractedItem) error {
			mockWriteDBCalled = true
			capturedItems = items
			return nil
		},
	}

	result, err := opts.Extract()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(mockWriteDBCalled).To(BeTrue())

	// All items should have source = "yield:my-yield.toml"
	for _, item := range capturedItems {
		g.Expect(item.Source).To(Equal("yield:my-yield.toml"))
	}
}

// TEST: Extract sets source field correctly for result files
// Traces to: TASK-4 AC-8
func TestExtract_SetsSourceFieldForResultFile(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	resultContent := `
[status]
result = "success"
timestamp = "2026-02-04T10:45:00Z"

[[decisions]]
context = "Test decision"
choice = "Test choice"
reason = "Test reason"
alternatives = []

[context]
phase = "design"
task = "TASK-10"
`

	mockWriteDBCalled := false
	var capturedItems []memory.ExtractedItem

	opts := memory.ExtractOpts{
		FilePath:   "/path/to/my-result.toml",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
		ReadFile: func(path string) ([]byte, error) {
			return []byte(resultContent), nil
		},
		WriteDB: func(items []memory.ExtractedItem) error {
			mockWriteDBCalled = true
			capturedItems = items
			return nil
		},
	}

	result, err := opts.Extract()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(mockWriteDBCalled).To(BeTrue())

	// All items should have source = "result:my-result.toml"
	for _, item := range capturedItems {
		g.Expect(item.Source).To(Equal("result:my-result.toml"))
	}
}

// TEST: Extract returns wrapped error on read failure
// Traces to: TASK-4 AC-10
func TestExtract_ReturnsWrappedErrorOnReadFailure(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	opts := memory.ExtractOpts{
		FilePath:   "/nonexistent/file.toml",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
		ReadFile: func(path string) ([]byte, error) {
			return nil, errors.New("file not found")
		},
	}

	result, err := opts.Extract()
	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
	g.Expect(err.Error()).To(ContainSubstring("/nonexistent/file.toml"))
	g.Expect(err.Error()).To(ContainSubstring("file not found"))
}

// TEST: Extract returns wrapped error on parse failure
// Traces to: TASK-4 AC-10
func TestExtract_ReturnsWrappedErrorOnParseFailure(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	invalidTOML := `
[yield
type = "complete"
`

	opts := memory.ExtractOpts{
		FilePath:   "/test/invalid.toml",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
		ReadFile: func(path string) ([]byte, error) {
			return []byte(invalidTOML), nil
		},
	}

	result, err := opts.Extract()
	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
	g.Expect(err.Error()).To(ContainSubstring("/test/invalid.toml"))
}

// TEST: Extract extracts context (phase, subphase, status)
// Traces to: TASK-4 AC-5
func TestExtract_ExtractsContext(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	yieldContent := `
[yield]
type = "complete"
timestamp = "2026-02-04T10:30:00Z"

[payload]
summary = "Test extraction"

[context]
phase = "tdd-green"
subphase = "refactor"
task = "TASK-4"
`

	opts := memory.ExtractOpts{
		FilePath:   "/test/yield.toml",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
		ReadFile: func(path string) ([]byte, error) {
			return []byte(yieldContent), nil
		},
	}

	result, err := opts.Extract()
	g.Expect(err).ToNot(HaveOccurred())

	// Items should have context from the file
	for _, item := range result.Items {
		g.Expect(item.Context).ToNot(BeEmpty())
	}
}

// TEST: Extract uses injected ReadFile for testing
// Traces to: TASK-4 AC-9
func TestExtract_UsesInjectedReadFile(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	readFileCalled := false
	requestedPath := ""

	yieldContent := `
[yield]
type = "complete"
timestamp = "2026-02-04T10:30:00Z"

[payload]
summary = "Test"

[context]
phase = "tdd-green"
task = "TASK-4"
`

	opts := memory.ExtractOpts{
		FilePath:   "/injected/path/test.toml",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
		ReadFile: func(path string) ([]byte, error) {
			readFileCalled = true
			requestedPath = path
			return []byte(yieldContent), nil
		},
	}

	_, err := opts.Extract()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(readFileCalled).To(BeTrue())
	g.Expect(requestedPath).To(Equal("/injected/path/test.toml"))
}

// TEST: Extract reads from file system when ReadFile is nil
// Traces to: TASK-4 AC-9
func TestExtract_ReadsFromFileSystemWhenReadFileNil(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	// Create an actual file
	yieldContent := `
[yield]
type = "complete"
timestamp = "2026-02-04T10:30:00Z"

[payload]
summary = "Read from actual file"

[context]
phase = "tdd-green"
task = "TASK-4"
`
	filePath := filepath.Join(tempDir, "actual-yield.toml")
	err := os.WriteFile(filePath, []byte(yieldContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ExtractOpts{
		FilePath:   filePath,
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
		// ReadFile is nil, should use os.ReadFile
	}

	result, err := opts.Extract()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.Status).To(Equal("success"))
}

// TEST: Extract returns ExtractResult with items list
// Traces to: TASK-4 AC-7
func TestExtract_ReturnsExtractResultWithItemsList(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	resultContent := `
[status]
result = "success"
timestamp = "2026-02-04T10:45:00Z"

[[decisions]]
context = "First decision"
choice = "Choice A"
reason = "Reason A"
alternatives = ["Alt1"]

[[decisions]]
context = "Second decision"
choice = "Choice B"
reason = "Reason B"
alternatives = ["Alt2"]

[context]
phase = "design"
task = "TASK-10"
`

	opts := memory.ExtractOpts{
		FilePath:   "/test/result.toml",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
		ReadFile: func(path string) ([]byte, error) {
			return []byte(resultContent), nil
		},
	}

	result, err := opts.Extract()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Items).To(HaveLen(2))
	g.Expect(result.ItemsExtracted).To(Equal(2))

	// Verify item structure
	g.Expect(result.Items[0].Type).To(Equal("decision"))
	g.Expect(result.Items[0].Context).ToNot(BeEmpty())
	g.Expect(result.Items[0].Content).ToNot(BeEmpty())
}

// TEST: Extract handles empty payload gracefully
// Traces to: TASK-4 AC-7
func TestExtract_HandlesEmptyPayloadGracefully(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	yieldContent := `
[yield]
type = "complete"
timestamp = "2026-02-04T10:30:00Z"

[context]
phase = "tdd-green"
task = "TASK-4"
`

	opts := memory.ExtractOpts{
		FilePath:   "/test/empty-payload.toml",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
		ReadFile: func(path string) ([]byte, error) {
			return []byte(yieldContent), nil
		},
	}

	result, err := opts.Extract()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Status).To(Equal("success"))
	g.Expect(result.ItemsExtracted).To(Equal(0))
	g.Expect(result.Items).To(BeEmpty())
}

// TEST: Extract handles empty decisions array gracefully
// Traces to: TASK-4 AC-7
func TestExtract_HandlesEmptyDecisionsGracefully(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	resultContent := `
[status]
result = "success"
timestamp = "2026-02-04T10:45:00Z"

[context]
phase = "design"
task = "TASK-10"
`

	opts := memory.ExtractOpts{
		FilePath:   "/test/empty-decisions.toml",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
		ReadFile: func(path string) ([]byte, error) {
			return []byte(resultContent), nil
		},
	}

	result, err := opts.Extract()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Status).To(Equal("success"))
	g.Expect(result.ItemsExtracted).To(Equal(0))
	g.Expect(result.Items).To(BeEmpty())
}

// TEST: Extract detects file type from content (yield vs result)
// Traces to: TASK-4 AC-1, AC-2
func TestExtract_DetectsFileTypeFromContent(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	// Yield file content
	yieldContent := `
[yield]
type = "complete"
timestamp = "2026-02-04T10:30:00Z"

[payload]
summary = "Yield file test"

[context]
phase = "tdd-green"
task = "TASK-4"
`

	opts := memory.ExtractOpts{
		FilePath:   "/test/unknown-extension.toml",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
		ReadFile: func(path string) ([]byte, error) {
			return []byte(yieldContent), nil
		},
	}

	result, err := opts.Extract()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Status).To(Equal("success"))

	// Result file content
	resultContent := `
[status]
result = "success"
timestamp = "2026-02-04T10:45:00Z"

[[decisions]]
context = "Test"
choice = "A"
reason = "B"
alternatives = []

[context]
phase = "design"
task = "TASK-10"
`

	opts2 := memory.ExtractOpts{
		FilePath:   "/test/another-file.toml",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
		ReadFile: func(path string) ([]byte, error) {
			return []byte(resultContent), nil
		},
	}

	result2, err := opts2.Extract()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result2.Status).To(Equal("success"))
}

// Property test: Extract always succeeds with valid yield content
// Traces to: TASK-4 AC-11
func TestExtract_PropertyValidYieldSucceeds(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		tempDir := t.TempDir()
		memoryDir := filepath.Join(tempDir, "memory")
		modelDir := filepath.Join(tempDir, "models")

		yieldType := rapid.SampledFrom([]string{
			"complete", "need-context", "blocked", "error",
		}).Draw(rt, "yieldType")

		phase := rapid.SampledFrom([]string{
			"tdd-red", "tdd-green", "design", "implementation",
		}).Draw(rt, "phase")

		summary := rapid.StringMatching(`[a-zA-Z0-9 ]{10,50}`).Draw(rt, "summary")
		timestamp := rapid.StringMatching(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`).Draw(rt, "timestamp")

		yieldContent := `
[yield]
type = "` + yieldType + `"
timestamp = "` + timestamp + `"

[payload]
summary = "` + summary + `"

[context]
phase = "` + phase + `"
task = "TASK-4"
`

		opts := memory.ExtractOpts{
			FilePath:   "/test/property-yield.toml",
			MemoryRoot: memoryDir,
			ModelDir:   modelDir,
			ReadFile: func(path string) ([]byte, error) {
				return []byte(yieldContent), nil
			},
		}

		result, err := opts.Extract()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Status).To(Equal("success"))
	})
}

// Helper function to filter items by type
func filterByType(items []memory.ExtractedItem, itemType string) []memory.ExtractedItem {
	var filtered []memory.ExtractedItem
	for _, item := range items {
		if item.Type == itemType {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
