package trace

import "time"

// NodeType represents the type of a traceability node.
type NodeType string

const (
	NodeTypeREQ  NodeType = "REQ"
	NodeTypeDES  NodeType = "DES"
	NodeTypeARCH NodeType = "ARCH"
	NodeTypeTASK NodeType = "TASK"
	NodeTypeTEST NodeType = "TEST"
)

// TraceItem represents a parsed traceability item before graph construction.
// It supports both documentation nodes (REQ/DES/ARCH/TASK) and test nodes (TEST).
type TraceItem struct {
	ID        string    // REQ-001, TEST-042, etc.
	Type      NodeType  // REQ, DES, ARCH, TASK, TEST
	Project   string    // Project identifier
	Title     string    // One-line summary
	Status    string    // draft|active|completed|deprecated
	TracesTo  []string  // Upstream IDs
	Tags      []string  // Optional metadata

	// Metadata
	Created time.Time
	Updated time.Time

	// TEST-specific fields
	Location string // File path (tests only)
	Line     int    // Line number (tests only)
	Function string // Test function name (tests only)

	// Source tracking
	SourceFile   string // Original file path
	SourceFormat string // "yaml", "toml", or "go-ast"
}

// Validate checks that the TraceItem has all required fields and valid values.
// Returns nil if valid, or an error describing the validation failure.
func (item *TraceItem) Validate() error {
	// TODO: Implement validation
	return nil
}
