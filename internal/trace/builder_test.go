package trace_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/trace"
)

// TEST-133 traces: TASK-020
// Test building graph from single item
func TestBuildGraph_SingleItem(t *testing.T) {
	g := NewWithT(t)

	items := []*trace.TraceItem{
		{
			ID:      "REQ-001",
			Type:    trace.NodeTypeREQ,
			Project: "test-project",
			Title:   "A requirement",
			Status:  "active",
		},
	}

	graph, warnings, err := trace.BuildGraph(items)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(warnings).To(BeEmpty())
	g.Expect(graph.Nodes).To(HaveLen(1))
	g.Expect(graph.Nodes["REQ-001"]).ToNot(BeNil())
}

// TEST-134 traces: TASK-020
// Test building graph with edges
func TestBuildGraph_WithEdges(t *testing.T) {
	g := NewWithT(t)

	items := []*trace.TraceItem{
		{
			ID:      "REQ-001",
			Type:    trace.NodeTypeREQ,
			Project: "test-project",
			Title:   "Root requirement",
			Status:  "active",
		},
		{
			ID:       "TASK-001",
			Type:     trace.NodeTypeTASK,
			Project:  "test-project",
			Title:    "Task tracing to req",
			Status:   "active",
			TracesTo: []string{"REQ-001"},
		},
	}

	graph, warnings, err := trace.BuildGraph(items)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(warnings).To(BeEmpty())
	g.Expect(graph.Nodes).To(HaveLen(2))
	g.Expect(graph.Edges["TASK-001"]).To(HaveLen(1))
}

// TEST-135 traces: TASK-020
// Test duplicate node ID returns error
func TestBuildGraph_DuplicateID(t *testing.T) {
	g := NewWithT(t)

	items := []*trace.TraceItem{
		{
			ID:      "REQ-001",
			Type:    trace.NodeTypeREQ,
			Project: "test-project",
			Title:   "First",
			Status:  "active",
		},
		{
			ID:      "REQ-001",
			Type:    trace.NodeTypeREQ,
			Project: "test-project",
			Title:   "Duplicate",
			Status:  "active",
		},
	}

	_, _, err := trace.BuildGraph(items)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("duplicate"))
}

// TEST-136 traces: TASK-020
// Test dangling edge creates warning
func TestBuildGraph_DanglingEdge(t *testing.T) {
	g := NewWithT(t)

	items := []*trace.TraceItem{
		{
			ID:       "TASK-001",
			Type:     trace.NodeTypeTASK,
			Project:  "test-project",
			Title:    "Task with missing target",
			Status:   "active",
			TracesTo: []string{"REQ-999"}, // Doesn't exist
		},
	}

	graph, warnings, err := trace.BuildGraph(items)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(warnings).To(HaveLen(1))
	g.Expect(warnings[0]).To(ContainSubstring("REQ-999"))
	g.Expect(graph.Nodes).To(HaveLen(1))
}

// TEST-137 traces: TASK-020
// Test building empty graph
func TestBuildGraph_Empty(t *testing.T) {
	g := NewWithT(t)

	items := []*trace.TraceItem{}

	graph, warnings, err := trace.BuildGraph(items)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(warnings).To(BeEmpty())
	g.Expect(graph.Nodes).To(BeEmpty())
}

// TEST-138 traces: TASK-020
// Property test: N items creates N nodes
func TestBuildGraph_PropertyNodeCount(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		count := rapid.IntRange(1, 20).Draw(rt, "count")
		items := make([]*trace.TraceItem, count)

		for i := 0; i < count; i++ {
			items[i] = &trace.TraceItem{
				ID:      "REQ-" + zeroPad(i+1),
				Type:    trace.NodeTypeREQ,
				Project: "test-project",
				Title:   "Requirement " + zeroPad(i+1),
				Status:  "active",
			}
		}

		graph, _, err := trace.BuildGraph(items)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(graph.Nodes).To(HaveLen(count))
	})
}

func zeroPad(n int) string {
	if n < 10 {
		return "00" + numStr(n)
	}
	if n < 100 {
		return "0" + numStr(n)
	}
	return numStr(n)
}

func numStr(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
