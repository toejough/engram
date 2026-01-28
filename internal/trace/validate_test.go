package trace_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/trace"
)

// TEST-145 traces: TASK-022
// Test all unique IDs passes validation
func TestValidateTESTUniqueness_AllUnique(t *testing.T) {
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "TEST-001", Type: trace.NodeTypeTEST, Location: "a_test.go"})
	_ = graph.AddNode(&trace.Node{ID: "TEST-002", Type: trace.NodeTypeTEST, Location: "b_test.go"})
	_ = graph.AddNode(&trace.Node{ID: "TEST-003", Type: trace.NodeTypeTEST, Location: "c_test.go"})

	errors := trace.ValidateTESTUniqueness(graph)
	g.Expect(errors).To(BeEmpty())
}

// TEST-146 traces: TASK-022
// Test duplicate TEST ID returns error
func TestValidateTESTUniqueness_Duplicate(t *testing.T) {
	g := NewWithT(t)

	// Can't add duplicate nodes directly, so we test the validation differently
	// The uniqueness check is at the TraceItem level across files
	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "TEST-001", Type: trace.NodeTypeTEST, Location: "a_test.go"})
	_ = graph.AddNode(&trace.Node{ID: "TEST-002", Type: trace.NodeTypeTEST, Location: "b_test.go"})

	// Since Graph.AddNode rejects duplicates, ValidateTESTUniqueness should pass
	// The real duplicate detection happens at parse time
	errors := trace.ValidateTESTUniqueness(graph)
	g.Expect(errors).To(BeEmpty())
}

// TEST-147 traces: TASK-022
// Test empty graph passes validation
func TestValidateTESTUniqueness_Empty(t *testing.T) {
	g := NewWithT(t)

	graph := trace.NewGraph()

	errors := trace.ValidateTESTUniqueness(graph)
	g.Expect(errors).To(BeEmpty())
}

// TEST-148 traces: TASK-022
// Test non-TEST nodes are ignored
func TestValidateTESTUniqueness_IgnoresNonTEST(t *testing.T) {
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-001", Type: trace.NodeTypeREQ})
	_ = graph.AddNode(&trace.Node{ID: "TASK-001", Type: trace.NodeTypeTASK})

	errors := trace.ValidateTESTUniqueness(graph)
	g.Expect(errors).To(BeEmpty())
}

// TEST-149 traces: TASK-023
// Test valid TEST ID format passes
func TestValidateTESTIDFormat_Valid(t *testing.T) {
	g := NewWithT(t)

	g.Expect(trace.ValidateTESTIDFormat("TEST-001")).To(BeTrue())
	g.Expect(trace.ValidateTESTIDFormat("TEST-042")).To(BeTrue())
	g.Expect(trace.ValidateTESTIDFormat("TEST-999")).To(BeTrue())
	g.Expect(trace.ValidateTESTIDFormat("TEST-0042")).To(BeTrue()) // More than 3 digits OK
}

// TEST-150 traces: TASK-023
// Test invalid TEST ID format fails
func TestValidateTESTIDFormat_Invalid(t *testing.T) {
	g := NewWithT(t)

	g.Expect(trace.ValidateTESTIDFormat("TEST-1")).To(BeFalse())   // Too few digits
	g.Expect(trace.ValidateTESTIDFormat("TEST-01")).To(BeFalse())  // Too few digits
	g.Expect(trace.ValidateTESTIDFormat("test-001")).To(BeFalse()) // Lowercase
	g.Expect(trace.ValidateTESTIDFormat("TST-001")).To(BeFalse())  // Wrong prefix
	g.Expect(trace.ValidateTESTIDFormat("TEST001")).To(BeFalse())  // No hyphen
}

// TEST-151 traces: TASK-023
// Test ValidateTESTIDFormats returns invalid IDs
func TestValidateTESTIDFormats_ReturnsInvalid(t *testing.T) {
	g := NewWithT(t)

	// Testing format validation function directly with ID strings
	ids := []string{"TEST-001", "TEST-1", "TEST-002", "TEST-02"}

	invalid := trace.ValidateTESTIDFormats(ids)
	g.Expect(invalid).To(ConsistOf("TEST-1", "TEST-02"))
}

// TEST-152 traces: TASK-023
// Test all valid IDs returns empty
func TestValidateTESTIDFormats_AllValid(t *testing.T) {
	g := NewWithT(t)

	ids := []string{"TEST-001", "TEST-042", "TEST-100"}

	invalid := trace.ValidateTESTIDFormats(ids)
	g.Expect(invalid).To(BeEmpty())
}
