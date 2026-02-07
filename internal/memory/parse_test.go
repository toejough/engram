package memory_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// TEST: ParseResultFile parses valid TOML
// Traces to: TASK-3 AC-2, AC-3, AC-7
func TestParseResultFile_ValidTOML(t *testing.T) {
	g := NewWithT(t)

	validResult := `
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
subphase = "review"
task = "TASK-10"
`

	result, err := memory.ParseResultFile([]byte(validResult))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.Status.Result).To(Equal("success"))
	g.Expect(result.Status.Timestamp).To(Equal("2026-02-04T10:45:00Z"))
	g.Expect(result.Context.Phase).To(Equal("design"))
	g.Expect(result.Context.Task).To(Equal("TASK-10"))
	g.Expect(result.Decisions).To(HaveLen(1))
	g.Expect(result.Decisions[0].Context).To(Equal("Error handling strategy"))
	g.Expect(result.Decisions[0].Choice).To(Equal("Use wrapped errors with context"))
	g.Expect(result.Decisions[0].Alternatives).To(ContainElements("Sentinel errors", "Error codes"))
}

// TEST: ParseResultFile returns error on invalid TOML
// Traces to: TASK-3 AC-5, AC-8
func TestParseResultFile_InvalidTOML(t *testing.T) {
	g := NewWithT(t)

	invalidTOML := `
[status
result = "success"
`

	result, err := memory.ParseResultFile([]byte(invalidTOML))
	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
	g.Expect(err.Error()).To(ContainSubstring("parse"))
}

// TEST: ParseResultFile returns SchemaValidationError on missing status.result
// Traces to: TASK-3 AC-6, AC-9, AC-11
func TestParseResultFile_MissingStatusResult(t *testing.T) {
	g := NewWithT(t)

	missingResult := `
[status]
timestamp = "2026-02-04T10:45:00Z"

[context]
phase = "design"
`

	result, err := memory.ParseResultFile([]byte(missingResult))
	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())

	var schemaErr *memory.SchemaValidationError
	g.Expect(err).To(BeAssignableToTypeOf(schemaErr))

	schemaErr = err.(*memory.SchemaValidationError)
	g.Expect(schemaErr.Field).To(Equal("status.result"))
	g.Expect(schemaErr.Expected).To(ContainSubstring("non-empty string"))
}

// TEST: ParseResultFile returns SchemaValidationError on missing status.timestamp
// Traces to: TASK-3 AC-6, AC-9
func TestParseResultFile_MissingStatusTimestamp(t *testing.T) {
	g := NewWithT(t)

	missingTimestamp := `
[status]
result = "success"

[context]
phase = "design"
`

	result, err := memory.ParseResultFile([]byte(missingTimestamp))
	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())

	var schemaErr *memory.SchemaValidationError
	g.Expect(err).To(BeAssignableToTypeOf(schemaErr))

	schemaErr = err.(*memory.SchemaValidationError)
	g.Expect(schemaErr.Field).To(Equal("status.timestamp"))
}

// TEST: ParseResultFile fails fast on first error
// Traces to: TASK-3 AC-9
func TestParseResultFile_FailsFastOnFirstError(t *testing.T) {
	g := NewWithT(t)

	// Multiple missing fields - should fail on first one (status.result)
	multipleErrors := `
[context]
phase = "design"
`

	result, err := memory.ParseResultFile([]byte(multipleErrors))
	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())

	var schemaErr *memory.SchemaValidationError
	g.Expect(err).To(BeAssignableToTypeOf(schemaErr))

	schemaErr = err.(*memory.SchemaValidationError)
	g.Expect(schemaErr.Field).To(Equal("status.result"))
}

// TEST: ParseResultFile handles multiple decisions
// Traces to: TASK-3 AC-3, AC-7
func TestParseResultFile_MultipleDecisions(t *testing.T) {
	g := NewWithT(t)

	multipleDecisions := `
[status]
result = "success"
timestamp = "2026-02-04T10:45:00Z"

[[decisions]]
context = "First decision"
choice = "Choice A"
reason = "Reason A"
alternatives = ["Alt A1"]

[[decisions]]
context = "Second decision"
choice = "Choice B"
reason = "Reason B"
alternatives = ["Alt B1", "Alt B2"]

[context]
phase = "design"
task = "TASK-10"
`

	result, err := memory.ParseResultFile([]byte(multipleDecisions))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Decisions).To(HaveLen(2))
	g.Expect(result.Decisions[0].Context).To(Equal("First decision"))
	g.Expect(result.Decisions[1].Context).To(Equal("Second decision"))
}

// TEST: ParseResultFile allows empty decisions array
// Traces to: TASK-3 AC-3
func TestParseResultFile_EmptyDecisions(t *testing.T) {
	g := NewWithT(t)

	emptyDecisions := `
[status]
result = "success"
timestamp = "2026-02-04T10:45:00Z"

[context]
phase = "design"
task = "TASK-10"
`

	result, err := memory.ParseResultFile([]byte(emptyDecisions))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Decisions).To(BeEmpty())
}

// Property test: valid result TOML always parses successfully
// Traces to: TASK-3 AC-12
func TestParseResultFile_PropertyValidTOMLParses(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		resultStatus := rapid.SampledFrom([]string{
			"success", "failure", "error",
		}).Draw(rt, "resultStatus")

		phase := rapid.SampledFrom([]string{
			"tdd-red", "tdd-green", "design", "implementation",
		}).Draw(rt, "phase")

		task := rapid.StringMatching(`TASK-\d+`).Draw(rt, "task")
		timestamp := rapid.StringMatching(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`).Draw(rt, "timestamp")

		tomlData := `
[status]
result = "` + resultStatus + `"
timestamp = "` + timestamp + `"

[context]
phase = "` + phase + `"
task = "` + task + `"
`

		result, err := memory.ParseResultFile([]byte(tomlData))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Status.Result).To(Equal(resultStatus))
		g.Expect(result.Context.Phase).To(Equal(phase))
	})
}
