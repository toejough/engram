package trace_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/trace"
)

// TEST-133 traces: TASK-20
// Test building graph from single item
func TestBuildGraph_SingleItem(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []*trace.TraceItem{
		{
			ID:      "REQ-1",
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
	g.Expect(graph.Nodes["REQ-1"]).ToNot(BeNil())
}

// TEST-134 traces: TASK-20
// Test building graph with edges
func TestBuildGraph_WithEdges(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []*trace.TraceItem{
		{
			ID:      "REQ-1",
			Type:    trace.NodeTypeREQ,
			Project: "test-project",
			Title:   "Root requirement",
			Status:  "active",
		},
		{
			ID:       "TASK-1",
			Type:     trace.NodeTypeTASK,
			Project:  "test-project",
			Title:    "Task tracing to req",
			Status:   "active",
			TracesTo: []string{"REQ-1"},
		},
	}

	graph, warnings, err := trace.BuildGraph(items)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(warnings).To(BeEmpty())
	g.Expect(graph.Nodes).To(HaveLen(2))
	g.Expect(graph.Edges["TASK-1"]).To(HaveLen(1))
}

// TEST-135 traces: TASK-20
// Test duplicate node ID returns error
func TestBuildGraph_DuplicateID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []*trace.TraceItem{
		{
			ID:      "REQ-1",
			Type:    trace.NodeTypeREQ,
			Project: "test-project",
			Title:   "First",
			Status:  "active",
		},
		{
			ID:      "REQ-1",
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

// TEST-136 traces: TASK-20
// Test dangling edge creates warning
func TestBuildGraph_DanglingEdge(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []*trace.TraceItem{
		{
			ID:       "TASK-1",
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

// TEST-137 traces: TASK-20
// Test building empty graph
func TestBuildGraph_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []*trace.TraceItem{}

	graph, warnings, err := trace.BuildGraph(items)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(warnings).To(BeEmpty())
	g.Expect(graph.Nodes).To(BeEmpty())
}

// TEST-138 traces: TASK-20
// Property test: N items creates N nodes
func TestBuildGraph_PropertyNodeCount(t *testing.T) {
	t.Parallel()
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

// TEST-161 traces: TASK-26
// Test ValidateGraph passes for valid graph
func TestValidateGraph_Valid(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []*trace.TraceItem{
		{ID: "REQ-1", Type: trace.NodeTypeREQ, Project: "test", Title: "Req", Status: "active"},
		{ID: "ARCH-1", Type: trace.NodeTypeARCH, Project: "test", Title: "Arch", Status: "active", TracesTo: []string{"REQ-1"}},
		{ID: "TASK-1", Type: trace.NodeTypeTASK, Project: "test", Title: "Task", Status: "active", TracesTo: []string{"ARCH-1"}},
		{ID: "TEST-1", Type: trace.NodeTypeTEST, Project: "test", Title: "Test", Status: "active", Location: "test.go", Function: "TestX", TracesTo: []string{"TASK-1"}},
	}
	graph, _, _ := trace.BuildGraph(items)

	result := trace.ValidateGraph(graph)
	g.Expect(result.Pass).To(BeTrue())
	g.Expect(result.Errors).To(BeEmpty())
}

// TEST-162 traces: TASK-26
// Test ValidateGraph fails with cycle
func TestValidateGraph_WithCycle(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-1", Type: trace.NodeTypeREQ})
	_ = graph.AddNode(&trace.Node{ID: "REQ-2", Type: trace.NodeTypeREQ})
	_ = graph.AddEdge(&trace.Edge{From: "REQ-1", To: "REQ-2"})
	_ = graph.AddEdge(&trace.Edge{From: "REQ-2", To: "REQ-1"})

	result := trace.ValidateGraph(graph)
	g.Expect(result.Pass).To(BeFalse())
	g.Expect(result.Errors).ToNot(BeEmpty())
}

// TEST-163 traces: TASK-26
// Test ValidateGraph reports warnings but still passes
func TestValidateGraph_WithWarnings(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []*trace.TraceItem{
		{ID: "REQ-1", Type: trace.NodeTypeREQ, Project: "test", Title: "Req", Status: "active"},
	}
	graph, _, _ := trace.BuildGraph(items)

	result := trace.ValidateGraph(graph)
	g.Expect(result.Pass).To(BeTrue()) // Warnings don't fail
	g.Expect(result.Warnings).ToNot(BeEmpty())
}

// TEST-164 traces: TASK-26
// Test ValidateGraph reports dangling as error
func TestValidateGraph_DanglingRef(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []*trace.TraceItem{
		{ID: "TASK-1", Type: trace.NodeTypeTASK, Project: "test", Title: "Task", Status: "active", TracesTo: []string{"REQ-999"}},
	}
	graph, _, _ := trace.BuildGraph(items)

	result := trace.ValidateGraph(graph)
	g.Expect(result.Pass).To(BeFalse())
	g.Expect(result.Errors).ToNot(BeEmpty())
}
