package memory_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// TEST: ExtractOpts struct exists with required fields
// Traces to: TASK-1 AC-1, AC-2, AC-3
func TestExtractOptsStructure(t *testing.T) {
	g := NewWithT(t)

	opts := memory.ExtractOpts{
		FilePath:   "/path/to/file.toml",
		MemoryRoot: "/path/to/memory",
		ModelDir:   "/path/to/models",
	}

	g.Expect(opts.FilePath).To(Equal("/path/to/file.toml"))
	g.Expect(opts.MemoryRoot).To(Equal("/path/to/memory"))
	g.Expect(opts.ModelDir).To(Equal("/path/to/models"))
}

// TEST: ExtractOpts includes optional injection fields for testing
// Traces to: TASK-1 AC-3
func TestExtractOptsInjectionFields(t *testing.T) {
	g := NewWithT(t)

	mockReadFile := func(path string) ([]byte, error) {
		return []byte("test data"), nil
	}

	opts := memory.ExtractOpts{
		FilePath: "/path/to/file.toml",
		ReadFile: mockReadFile,
	}

	g.Expect(opts.ReadFile).ToNot(BeNil())

	// Verify the injected function can be called
	data, err := opts.ReadFile("/test/path")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(data).To(Equal([]byte("test data")))
}

// TEST: ExtractResult struct includes required fields
// Traces to: TASK-1 AC-4, AC-5
func TestExtractResultStructure(t *testing.T) {
	g := NewWithT(t)

	result := memory.ExtractResult{
		Status:         "success",
		FilePath:       "/path/to/result.toml",
		ItemsExtracted: 5,
		Items: []memory.ExtractedItem{
			{
				Type:    "decision",
				Context: "API design",
				Content: "Use REST over GraphQL",
			},
			{
				Type:    "learning",
				Context: "performance",
				Content: "Caching reduces database load",
			},
		},
	}

	g.Expect(result.Status).To(Equal("success"))
	g.Expect(result.FilePath).To(Equal("/path/to/result.toml"))
	g.Expect(result.ItemsExtracted).To(Equal(5))
	g.Expect(result.Items).To(HaveLen(2))
	g.Expect(result.Items[0].Type).To(Equal("decision"))
	g.Expect(result.Items[1].Type).To(Equal("learning"))
}

// TEST: YieldFile struct matches yield protocol schema
// Traces to: TASK-1 AC-6
func TestYieldFileStructure(t *testing.T) {
	g := NewWithT(t)

	yieldData := `
[yield]
type = "complete"
timestamp = "2026-02-04T10:30:00Z"

[payload]
artifact = "internal/foo/foo.go"

[context]
phase = "tdd-red"
subphase = "complete"
task = "TASK-5"
`

	var yieldFile memory.YieldFile
	err := toml.Unmarshal([]byte(yieldData), &yieldFile)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(yieldFile.Yield.Type).To(Equal("complete"))
	g.Expect(yieldFile.Yield.Timestamp).ToNot(BeEmpty())
	g.Expect(yieldFile.Context.Phase).To(Equal("tdd-red"))
	g.Expect(yieldFile.Context.Task).To(Equal("TASK-5"))
}

// TEST: ResultFile struct matches result protocol schema
// Traces to: TASK-1 AC-6
func TestResultFileStructure(t *testing.T) {
	g := NewWithT(t)

	resultData := `
[status]
result = "success"
timestamp = "2026-02-04T10:45:00Z"

[[decisions]]
context = "Error handling strategy"
choice = "Use wrapped errors with context"
reason = "Provides clear error traces"
alternatives = ["Sentinel errors", "Error codes"]

[context]
phase = "design"
task = "TASK-10"
`

	var resultFile memory.ResultFile
	err := toml.Unmarshal([]byte(resultData), &resultFile)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(resultFile.Status.Result).To(Equal("success"))
	g.Expect(resultFile.Decisions).To(HaveLen(1))
	g.Expect(resultFile.Decisions[0].Context).To(Equal("Error handling strategy"))
	g.Expect(resultFile.Decisions[0].Alternatives).To(ContainElement("Sentinel errors"))
	g.Expect(resultFile.Context.Phase).To(Equal("design"))
}

// TEST: Structs use TOML struct tags for validation
// Traces to: TASK-1 AC-7
func TestTOMLStructTagsPresent(t *testing.T) {
	g := NewWithT(t)

	// Test that TOML unmarshaling works with struct tags
	yieldData := `
[yield]
type = "complete"
timestamp = "2026-02-04T10:30:00Z"

[context]
phase = "implementation"
`

	var yieldFile memory.YieldFile
	err := toml.Unmarshal([]byte(yieldData), &yieldFile)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify fields mapped correctly via struct tags
	g.Expect(yieldFile.Yield.Type).To(Equal("complete"))
	g.Expect(yieldFile.Context.Phase).To(Equal("implementation"))
}

// TEST: Structs reject invalid TOML due to struct tags
// Traces to: TASK-1 AC-7
func TestTOMLStructTagsValidation(t *testing.T) {
	g := NewWithT(t)

	// Invalid TOML with wrong field type
	invalidData := `
[yield]
type = 123
timestamp = "2026-02-04T10:30:00Z"
`

	var yieldFile memory.YieldFile
	err := toml.Unmarshal([]byte(invalidData), &yieldFile)

	// Should fail because type should be string, not int
	g.Expect(err).To(HaveOccurred())
}

// TEST: SchemaValidationError struct includes required fields
// Traces to: TASK-1 AC-8
func TestSchemaValidationErrorStructure(t *testing.T) {
	g := NewWithT(t)

	schemaErr := memory.SchemaValidationError{
		Field:    "payload.decisions",
		Expected: "array of objects",
		Actual:   "string",
		Line:     15,
	}

	g.Expect(schemaErr.Field).To(Equal("payload.decisions"))
	g.Expect(schemaErr.Expected).To(Equal("array of objects"))
	g.Expect(schemaErr.Actual).To(Equal("string"))
	g.Expect(schemaErr.Line).To(Equal(15))
}

// TEST: SchemaValidationError implements error interface
// Traces to: TASK-1 AC-9
func TestSchemaValidationErrorImplementsError(t *testing.T) {
	g := NewWithT(t)

	schemaErr := memory.SchemaValidationError{
		Field:    "payload.decisions",
		Expected: "array",
		Actual:   "string",
		Line:     15,
	}

	// Should be able to use as error
	var err error = &schemaErr
	g.Expect(err).ToNot(BeNil())
	g.Expect(err.Error()).To(ContainSubstring("payload.decisions"))
	g.Expect(err.Error()).To(ContainSubstring("array"))
	g.Expect(err.Error()).To(ContainSubstring("string"))
	g.Expect(err.Error()).To(ContainSubstring("15"))
}

// TEST: All types have godoc comments
// This is a documentation test - verified by reviewing types.go source
// Traces to: TASK-1 AC-10

// TEST: Unit tests verify struct tags map to TOML fields correctly
// Traces to: TASK-1 AC-11
func TestStructTagsMappingToTOML(t *testing.T) {
	g := NewWithT(t)

	// Create a YieldFile programmatically
	yieldFile := memory.YieldFile{
		Yield: memory.YieldSection{
			Type:      "complete",
			Timestamp: "2026-02-04T10:30:00Z",
		},
		Payload: map[string]interface{}{
			"artifact": "internal/foo/foo.go",
		},
		Context: memory.ContextSection{
			Phase:    "tdd-red",
			Subphase: "complete",
			Task:     "TASK-5",
		},
	}

	// Marshal to TOML
	var buf strings.Builder
	encoder := toml.NewEncoder(&buf)
	err := encoder.Encode(yieldFile)
	g.Expect(err).ToNot(HaveOccurred())

	tomlOutput := buf.String()

	// Verify expected TOML field names appear
	g.Expect(tomlOutput).To(ContainSubstring("[yield]"))
	g.Expect(tomlOutput).To(ContainSubstring("type = \"complete\""))
	g.Expect(tomlOutput).To(ContainSubstring("[context]"))
	g.Expect(tomlOutput).To(ContainSubstring("phase = \"tdd-red\""))

	// Unmarshal back and verify round-trip
	var decoded memory.YieldFile
	err = toml.Unmarshal([]byte(tomlOutput), &decoded)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(decoded.Yield.Type).To(Equal("complete"))
	g.Expect(decoded.Context.Phase).To(Equal("tdd-red"))
}

// Property test: YieldFile struct tags handle arbitrary valid TOML
// Traces to: TASK-1 AC-11
func TestYieldFileStructTagsProperty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		// Generate random valid yield types
		yieldType := rapid.SampledFrom([]string{
			"complete", "need-context", "blocked", "error",
		}).Draw(rt, "yieldType")

		phase := rapid.SampledFrom([]string{
			"tdd-red", "tdd-green", "design", "implementation",
		}).Draw(rt, "phase")

		// Create TOML with random valid values
		tomlData := fmt.Sprintf(`
[yield]
type = "%s"
timestamp = "2026-02-04T10:30:00Z"

[context]
phase = "%s"
`, yieldType, phase)

		var yieldFile memory.YieldFile
		err := toml.Unmarshal([]byte(tomlData), &yieldFile)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(yieldFile.Yield.Type).To(Equal(yieldType))
		g.Expect(yieldFile.Context.Phase).To(Equal(phase))
	})
}

// Property test: ResultFile struct tags handle arbitrary decisions
// Traces to: TASK-1 AC-11
func TestResultFileStructTagsProperty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		// Generate random decision context
		context := rapid.StringMatching(`[A-Za-z ]+`).Draw(rt, "context")
		choice := rapid.StringMatching(`[A-Za-z ]+`).Draw(rt, "choice")

		// Create TOML with random valid values
		tomlData := fmt.Sprintf(`
[status]
result = "success"

[[decisions]]
context = "%s"
choice = "%s"
reason = "Testing"
alternatives = ["alt1", "alt2"]
`, context, choice)

		var resultFile memory.ResultFile
		err := toml.Unmarshal([]byte(tomlData), &resultFile)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(resultFile.Decisions).To(HaveLen(1))
		g.Expect(resultFile.Decisions[0].Context).To(Equal(context))
		g.Expect(resultFile.Decisions[0].Choice).To(Equal(choice))
	})
}

// TEST: ExtractedItem type structure
// Traces to: TASK-1 AC-5
func TestExtractedItemStructure(t *testing.T) {
	g := NewWithT(t)

	item := memory.ExtractedItem{
		Type:    "decision",
		Context: "Database choice",
		Content: "Use PostgreSQL for relational data",
	}

	g.Expect(item.Type).To(Equal("decision"))
	g.Expect(item.Context).To(Equal("Database choice"))
	g.Expect(item.Content).To(Equal("Use PostgreSQL for relational data"))
}

// TEST: ExtractedItem JSON serialization
// Needed for potential JSON output
func TestExtractedItemJSONSerialization(t *testing.T) {
	g := NewWithT(t)

	item := memory.ExtractedItem{
		Type:    "learning",
		Context: "performance",
		Content: "Index foreign keys for faster joins",
	}

	jsonData, err := json.Marshal(item)
	g.Expect(err).ToNot(HaveOccurred())

	var decoded memory.ExtractedItem
	err = json.Unmarshal(jsonData, &decoded)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(decoded).To(Equal(item))
}
