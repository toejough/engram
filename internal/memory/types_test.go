package memory_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// TEST: ExtractOpts includes optional injection fields for testing
// Traces to: TASK-1 AC-3
func TestExtractOptsInjectionFields(t *testing.T) {
	t.Parallel()
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

// TEST: ExtractOpts struct exists with required fields
// Traces to: TASK-1 AC-1, AC-2, AC-3
func TestExtractOptsStructure(t *testing.T) {
	t.Parallel()
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

// TEST: ExtractResult struct includes required fields
// Traces to: TASK-1 AC-4, AC-5
func TestExtractResultStructure(t *testing.T) {
	t.Parallel()
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

// TEST: ExtractedItem JSON serialization
// Needed for potential JSON output
func TestExtractedItemJSONSerialization(t *testing.T) {
	t.Parallel()
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

// TEST: ExtractedItem type structure
// Traces to: TASK-1 AC-5
func TestExtractedItemStructure(t *testing.T) {
	t.Parallel()
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

// TEST: All types have godoc comments
// This is a documentation test - verified by reviewing types.go source
// Traces to: TASK-1 AC-10

// Property test: ResultFile struct tags handle arbitrary decisions
// Traces to: TASK-1 AC-11
func TestResultFileStructTagsProperty(t *testing.T) {
	t.Parallel()
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

// TEST: ResultFile struct matches result protocol schema
// Traces to: TASK-1 AC-6
func TestResultFileStructure(t *testing.T) {
	t.Parallel()
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

// TEST: SchemaValidationError implements error interface
// Traces to: TASK-1 AC-9
func TestSchemaValidationErrorImplementsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	schemaErr := memory.SchemaValidationError{
		Field:    "payload.decisions",
		Expected: "array",
		Actual:   "string",
		Line:     15,
	}

	// Should be able to use as error
	var err error = &schemaErr
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("payload.decisions"))
	g.Expect(err.Error()).To(ContainSubstring("array"))
	g.Expect(err.Error()).To(ContainSubstring("string"))
	g.Expect(err.Error()).To(ContainSubstring("15"))
}

// TEST: SchemaValidationError struct includes required fields
// Traces to: TASK-1 AC-8
func TestSchemaValidationErrorStructure(t *testing.T) {
	t.Parallel()
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
