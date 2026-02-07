package memory_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// Extract tests (TASK-4)
// ============================================================================

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

// TEST: Extract uses injected ReadFile for testing
// Traces to: TASK-4 AC-9
func TestExtract_UsesInjectedReadFile(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	readFileCalled := false
	requestedPath := ""

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

	opts := memory.ExtractOpts{
		FilePath:   "/injected/path/test.toml",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
		ReadFile: func(path string) ([]byte, error) {
			readFileCalled = true
			requestedPath = path
			return []byte(resultContent), nil
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
	filePath := filepath.Join(tempDir, "actual-result.toml")
	err := os.WriteFile(filePath, []byte(resultContent), 0644)
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
