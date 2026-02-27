package trace

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Exported constants.
const (
	NodeTypeARCH  NodeType = "ARCH"
	NodeTypeDES   NodeType = "DES"
	NodeTypeISSUE NodeType = "ISSUE"
	NodeTypeREQ   NodeType = "REQ"
	NodeTypeTASK  NodeType = "TASK"
	NodeTypeTEST  NodeType = "TEST"
)

// NodeType represents the type of a traceability node.
type NodeType string

// TraceItem represents a parsed traceability item before graph construction.
// It supports both documentation nodes (REQ/DES/ARCH/TASK) and test nodes (TEST).
type TraceItem struct {
	ID       string   // REQ-1, TEST-42, etc.
	Type     NodeType // REQ, DES, ARCH, TASK, TEST
	Project  string   // Project identifier
	Title    string   // One-line summary
	Status   string   // draft|active|completed|deprecated
	TracesTo []string // Upstream IDs
	Tags     []string // Optional metadata

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
	// Check required fields
	if item.ID == "" {
		return errors.New("field ID required")
	}

	if item.Type == "" {
		return errors.New("field Type required")
	}

	if !validNodeTypes[item.Type] {
		return fmt.Errorf("field Type must be one of ISSUE, REQ, DES, ARCH, TASK, TEST; got %q", item.Type)
	}

	if item.Project == "" {
		return errors.New("field Project required")
	}

	if item.Title == "" {
		return errors.New("field Title required")
	}

	if item.Status == "" {
		return errors.New("field Status required")
	}

	if !validStatuses[item.Status] {
		return fmt.Errorf("field Status must be one of draft, active, completed, deprecated; got %q", item.Status)
	}

	// Validate ID format
	if !itemIDPattern.MatchString(item.ID) {
		return fmt.Errorf("field ID must match format PREFIX-N (e.g., REQ-1); got %q", item.ID)
	}

	// Validate ID prefix matches Type
	expectedPrefix := string(item.Type) + "-"
	if !strings.HasPrefix(item.ID, expectedPrefix) {
		return fmt.Errorf("id prefix mismatch: ID %q does not match Type %q", item.ID, item.Type)
	}

	// TEST-specific validations
	if item.Type == NodeTypeTEST {
		if item.Location == "" {
			return errors.New("field Location required for TEST items")
		}

		if item.Function == "" {
			return errors.New("field Function required for TEST items")
		}
	}

	return nil
}

// unexported variables.
var (
	itemIDPattern  = regexp.MustCompile(`^(ISSUE|REQ|DES|ARCH|TASK|TEST)-\d+$`)
	validNodeTypes = map[NodeType]bool{
		NodeTypeISSUE: true,
		NodeTypeREQ:   true,
		NodeTypeDES:   true,
		NodeTypeARCH:  true,
		NodeTypeTASK:  true,
		NodeTypeTEST:  true,
	}
	validStatuses = map[string]bool{
		"draft":      true,
		"active":     true,
		"completed":  true,
		"deprecated": true,
	}
)
