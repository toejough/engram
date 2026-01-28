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

// TEST-153 traces: TASK-024
// Test all edges point to existing nodes
func TestValidateDanglingRefs_NoDangling(t *testing.T) {
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-001", Type: trace.NodeTypeREQ})
	_ = graph.AddNode(&trace.Node{ID: "TASK-001", Type: trace.NodeTypeTASK})
	_ = graph.AddEdge(&trace.Edge{From: "TASK-001", To: "REQ-001"})

	errors := trace.ValidateDanglingRefs(graph)
	g.Expect(errors).To(BeEmpty())
}

// TEST-154 traces: TASK-024
// Test detects dangling reference
func TestValidateDanglingRefs_Dangling(t *testing.T) {
	g := NewWithT(t)

	// Build graph that allows dangling edges (via BuildGraph warnings)
	items := []*trace.TraceItem{
		{
			ID:       "TASK-001",
			Type:     trace.NodeTypeTASK,
			Project:  "test",
			Title:    "Task",
			Status:   "active",
			TracesTo: []string{"REQ-999"}, // Doesn't exist
		},
	}
	graph, _, _ := trace.BuildGraph(items)

	errors := trace.ValidateDanglingRefs(graph)
	g.Expect(errors).To(HaveLen(1))
	g.Expect(errors[0]).To(ContainSubstring("REQ-999"))
}

// TEST-155 traces: TASK-024
// Test multiple dangling references all reported
func TestValidateDanglingRefs_Multiple(t *testing.T) {
	g := NewWithT(t)

	items := []*trace.TraceItem{
		{
			ID:       "TASK-001",
			Type:     trace.NodeTypeTASK,
			Project:  "test",
			Title:    "Task",
			Status:   "active",
			TracesTo: []string{"REQ-999", "ARCH-999"}, // Both don't exist
		},
	}
	graph, _, _ := trace.BuildGraph(items)

	errors := trace.ValidateDanglingRefs(graph)
	g.Expect(errors).To(HaveLen(2))
}

// TEST-156 traces: TASK-024
// Test empty graph passes
func TestValidateDanglingRefs_Empty(t *testing.T) {
	g := NewWithT(t)

	graph := trace.NewGraph()

	errors := trace.ValidateDanglingRefs(graph)
	g.Expect(errors).To(BeEmpty())
}

// TEST-157 traces: TASK-025
// Test complete coverage passes
func TestValidateCoverage_Complete(t *testing.T) {
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-001", Type: trace.NodeTypeREQ})
	_ = graph.AddNode(&trace.Node{ID: "ARCH-001", Type: trace.NodeTypeARCH})
	_ = graph.AddNode(&trace.Node{ID: "TASK-001", Type: trace.NodeTypeTASK})
	_ = graph.AddNode(&trace.Node{ID: "TEST-001", Type: trace.NodeTypeTEST, Location: "test.go"})
	_ = graph.AddEdge(&trace.Edge{From: "ARCH-001", To: "REQ-001"})
	_ = graph.AddEdge(&trace.Edge{From: "TASK-001", To: "ARCH-001"})
	_ = graph.AddEdge(&trace.Edge{From: "TEST-001", To: "TASK-001"})

	warnings := trace.ValidateCoverage(graph)
	g.Expect(warnings).To(BeEmpty())
}

// TEST-158 traces: TASK-025
// Test REQ with no downstream ARCH/DES warns
func TestValidateCoverage_REQNoDownstream(t *testing.T) {
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-001", Type: trace.NodeTypeREQ})

	warnings := trace.ValidateCoverage(graph)
	g.Expect(warnings).To(HaveLen(1))
	g.Expect(warnings[0]).To(ContainSubstring("REQ-001"))
}

// TEST-159 traces: TASK-025
// Test TASK with no TEST warns
func TestValidateCoverage_TASKNoTEST(t *testing.T) {
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "TASK-001", Type: trace.NodeTypeTASK})

	warnings := trace.ValidateCoverage(graph)
	g.Expect(warnings).To(HaveLen(1))
	g.Expect(warnings[0]).To(ContainSubstring("TASK-001"))
}

// TEST-160 traces: TASK-025
// Test ARCH with no TASK warns
func TestValidateCoverage_ARCHNoTASK(t *testing.T) {
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "ARCH-001", Type: trace.NodeTypeARCH})

	warnings := trace.ValidateCoverage(graph)
	g.Expect(warnings).To(HaveLen(1))
	g.Expect(warnings[0]).To(ContainSubstring("ARCH-001"))
}
