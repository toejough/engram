package memory

import "fmt"

// ExtractOpts contains the options for memory extraction operations.
// It provides file paths for input and output locations, along with optional
// injection points for testing.
type ExtractOpts struct {
	// Required fields

	// FilePath is the path to the TOML file to extract from
	FilePath string

	// MemoryRoot is the root directory for memory storage
	MemoryRoot string

	// ModelDir is the directory containing the ONNX model files
	ModelDir string

	// Optional injection fields for testing

	// ReadFile is an optional injected function for reading files (for testing)
	ReadFile func(path string) ([]byte, error)

	// WriteDB is an optional injected function for writing to the database (for testing)
	WriteDB func(items []ExtractedItem) error
}

// ExtractResult contains the results of a memory extraction operation.
// It includes status information and the extracted items.
type ExtractResult struct {
	// Status indicates the result of the extraction. Values: "success", "error", "partial"
	Status string

	// FilePath is the absolute path to the file that was processed
	FilePath string

	// ItemsExtracted is the count of items successfully extracted from the file
	ItemsExtracted int

	// Items contains the individual extracted items (decisions, learnings, etc.)
	Items []ExtractedItem
}

// ExtractedItem represents a single extracted item from a result file.
// Each item has a type, situational context, content, and source information.
type ExtractedItem struct {
	// Type indicates the kind of item. Common values: "decision", "learning", "finding", "summary"
	Type string `json:"type"`

	// Context provides the situational context for the item (e.g., "API design", "performance")
	Context string `json:"context"`

	// Content is the actual content of the extracted item
	Content string `json:"content"`

	// Source identifies where the item was extracted from. Format: "result:{filename}"
	Source string `json:"source"`
}

// ResultFile represents the structure of a result protocol TOML file.
// It matches the result protocol schema with sections for status, decisions, and context.
type ResultFile struct {
	// Status contains the result status information
	Status StatusSection `toml:"status"`

	// Decisions contains the decisions made during execution
	Decisions []Decision `toml:"decisions"`

	// Context contains the execution context information
	Context ContextSection `toml:"context"`
}

// StatusSection contains status information for a result file.
type StatusSection struct {
	// Result indicates the outcome. Typical values: "success", "failure", "error"
	Result string `toml:"result"`

	// Timestamp indicates when the result was created (RFC3339 format)
	Timestamp string `toml:"timestamp"`
}

// Decision represents a decision made during execution.
type Decision struct {
	// Context provides the situational context for the decision
	Context string `toml:"context"`

	// Choice is the decision that was made
	Choice string `toml:"choice"`

	// Reason explains why the choice was made
	Reason string `toml:"reason"`

	// Alternatives lists other options that were considered
	Alternatives []string `toml:"alternatives"`
}

// ContextSection contains execution context information for result files.
type ContextSection struct {
	// Phase indicates the current execution phase (e.g., "tdd-red", "design")
	Phase string `toml:"phase"`

	// Subphase provides more granular phase information
	Subphase string `toml:"subphase"`

	// Task identifies the task being worked on (e.g., "TASK-5")
	Task string `toml:"task"`
}

// SchemaValidationError represents an error that occurred during schema validation.
// It provides detailed information about what was expected versus what was found.
type SchemaValidationError struct {
	// Field is the name of the field that failed validation
	Field string

	// Expected describes what type or value was expected
	Expected string

	// Actual describes what type or value was actually found
	Actual string

	// Line is the line number in the source file where the error occurred
	Line int
}

// Error implements the error interface for SchemaValidationError.
// It formats the validation error with all relevant details.
func (e *SchemaValidationError) Error() string {
	return fmt.Sprintf("schema validation error: field '%s' expected %s but got %s at line %d",
		e.Field, e.Expected, e.Actual, e.Line)
}
