package trace_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/trace"
)

// TEST-179 traces: TASK-29
// Test ValidateV2 passes with valid graph
func TestValidateV2_Valid(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// REQ -> ARCH -> TASK -> TEST chain
	items := []*trace.TraceItem{
		{ID: "REQ-1", Type: trace.NodeTypeREQ, Project: "test", Title: "Req", Status: "active"},
		{ID: "ARCH-1", Type: trace.NodeTypeARCH, Project: "test", Title: "Arch", Status: "active", TracesTo: []string{"REQ-1"}},
		{ID: "TASK-1", Type: trace.NodeTypeTASK, Project: "test", Title: "Task", Status: "active", TracesTo: []string{"ARCH-1"}},
		{ID: "TEST-1", Type: trace.NodeTypeTEST, Project: "test", Title: "Test", Status: "active", Location: "foo_test.go", Function: "TestFeature", TracesTo: []string{"TASK-1"}},
	}

	result, err := trace.ValidateV2(items)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Pass).To(BeTrue())
	g.Expect(result.Errors).To(BeEmpty())
}

// TEST-180 traces: TASK-29
// Test ValidateV2 fails with cycle
func TestValidateV2_WithCycle(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Create a cycle: REQ-1 -> REQ-2 -> REQ-1
	items := []*trace.TraceItem{
		{ID: "REQ-1", Type: trace.NodeTypeREQ, Project: "test", Title: "First", Status: "active", TracesTo: []string{"REQ-2"}},
		{ID: "REQ-2", Type: trace.NodeTypeREQ, Project: "test", Title: "Second", Status: "active", TracesTo: []string{"REQ-1"}},
	}

	result, err := trace.ValidateV2(items)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Pass).To(BeFalse())
	g.Expect(result.Errors).To(ContainElement(ContainSubstring("cycle")))
}

// TEST-181 traces: TASK-29
// Test ValidateV2 fails with dangling reference
func TestValidateV2_DanglingRef(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// TASK traces to non-existent ARCH
	items := []*trace.TraceItem{
		{ID: "TASK-1", Type: trace.NodeTypeTASK, Project: "test", Title: "Task", Status: "active", TracesTo: []string{"ARCH-999"}},
	}

	result, err := trace.ValidateV2(items)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Pass).To(BeFalse())
	g.Expect(result.Errors).To(ContainElement(ContainSubstring("ARCH-999")))
}

// TEST-182 traces: TASK-29
// Test ValidateV2 reports coverage warnings
func TestValidateV2_CoverageWarnings(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// REQ with no downstream ARCH
	items := []*trace.TraceItem{
		{ID: "REQ-1", Type: trace.NodeTypeREQ, Project: "test", Title: "Orphan req", Status: "active"},
	}

	result, err := trace.ValidateV2(items)
	g.Expect(err).ToNot(HaveOccurred())
	// Coverage gaps are warnings, not errors
	g.Expect(result.Pass).To(BeTrue())
	g.Expect(result.Warnings).To(ContainElement(ContainSubstring("REQ-1")))
}

// TEST-183 traces: TASK-29
// Test ValidateV2 with empty items
func TestValidateV2_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []*trace.TraceItem{}

	result, err := trace.ValidateV2(items)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Pass).To(BeTrue())
	g.Expect(result.Errors).To(BeEmpty())
}

// TEST-184 traces: TASK-29
// Test ValidateV2 returns node count
func TestValidateV2_NodeCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []*trace.TraceItem{
		{ID: "REQ-1", Type: trace.NodeTypeREQ, Project: "test", Title: "First", Status: "active"},
		{ID: "REQ-2", Type: trace.NodeTypeREQ, Project: "test", Title: "Second", Status: "active"},
	}

	result, err := trace.ValidateV2(items)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.NodeCount).To(Equal(2))
}
