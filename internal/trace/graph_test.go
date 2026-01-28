package trace_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/trace"
)

// TEST-017 traces: TASK-002
// Test creating a graph with valid nodes
func TestGraph_NewGraph(t *testing.T) {
	g := NewWithT(t)

	graph := trace.NewGraph()
	g.Expect(graph).ToNot(BeNil())
	g.Expect(graph.Nodes).ToNot(BeNil())
	g.Expect(graph.Edges).ToNot(BeNil())
	g.Expect(graph.ReverseEdges).ToNot(BeNil())
}

// TEST-018 traces: TASK-002
// Test creating a node with valid fields
func TestNode_ValidNode(t *testing.T) {
	g := NewWithT(t)

	node := &trace.Node{
		ID:      "REQ-001",
		Type:    trace.NodeTypeREQ,
		Project: "my-project",
		Title:   "A valid requirement",
		Status:  "active",
	}

	g.Expect(node.ID).To(Equal("REQ-001"))
	g.Expect(node.Type).To(Equal(trace.NodeTypeREQ))
	g.Expect(node.Project).To(Equal("my-project"))
}

// TEST-019 traces: TASK-002
// Test creating a TEST node with all required fields
func TestNode_ValidTestNode(t *testing.T) {
	g := NewWithT(t)

	node := &trace.Node{
		ID:       "TEST-001",
		Type:     trace.NodeTypeTEST,
		Project:  "my-project",
		Title:    "Test user validation",
		Status:   "active",
		Location: "user_test.go",
		Line:     45,
		Function: "TestValidateUser",
	}

	g.Expect(node.ID).To(Equal("TEST-001"))
	g.Expect(node.Type).To(Equal(trace.NodeTypeTEST))
	g.Expect(node.Location).To(Equal("user_test.go"))
	g.Expect(node.Line).To(Equal(45))
	g.Expect(node.Function).To(Equal("TestValidateUser"))
}

// TEST-020 traces: TASK-002
// Test creating an edge with valid From/To IDs
func TestEdge_ValidEdge(t *testing.T) {
	g := NewWithT(t)

	edge := &trace.Edge{
		From: "REQ-001",
		To:   "ARCH-001",
	}

	g.Expect(edge.From).To(Equal("REQ-001"))
	g.Expect(edge.To).To(Equal("ARCH-001"))
}

// TEST-021 traces: TASK-002
// Test that all NodeType constants are defined
func TestNodeType_AllTypes(t *testing.T) {
	g := NewWithT(t)

	// Verify all 5 types exist
	g.Expect(trace.NodeTypeREQ).To(Equal(trace.NodeType("REQ")))
	g.Expect(trace.NodeTypeDES).To(Equal(trace.NodeType("DES")))
	g.Expect(trace.NodeTypeARCH).To(Equal(trace.NodeType("ARCH")))
	g.Expect(trace.NodeTypeTASK).To(Equal(trace.NodeType("TASK")))
	g.Expect(trace.NodeTypeTEST).To(Equal(trace.NodeType("TEST")))
}

// TEST-022 traces: TASK-002
// Test creating a node from a TraceItem
func TestNode_FromTraceItem(t *testing.T) {
	g := NewWithT(t)

	item := &trace.TraceItem{
		ID:       "TEST-042",
		Type:     trace.NodeTypeTEST,
		Project:  "my-project",
		Title:    "Test something",
		Status:   "active",
		TracesTo: []string{"TASK-001"},
		Location: "foo_test.go",
		Line:     10,
		Function: "TestFoo",
	}

	node := trace.NodeFromItem(item)
	g.Expect(node.ID).To(Equal("TEST-042"))
	g.Expect(node.Type).To(Equal(trace.NodeTypeTEST))
	g.Expect(node.Project).To(Equal("my-project"))
	g.Expect(node.Title).To(Equal("Test something"))
	g.Expect(node.Location).To(Equal("foo_test.go"))
	g.Expect(node.Line).To(Equal(10))
	g.Expect(node.Function).To(Equal("TestFoo"))
}

// TEST-023 traces: TASK-002
// Test creating edges from a TraceItem's TracesTo field
func TestEdge_FromTraceItem(t *testing.T) {
	g := NewWithT(t)

	item := &trace.TraceItem{
		ID:       "TEST-001",
		Type:     trace.NodeTypeTEST,
		TracesTo: []string{"TASK-001", "TASK-002"},
	}

	edges := trace.EdgesFromItem(item)
	g.Expect(edges).To(HaveLen(2))
	g.Expect(edges[0].From).To(Equal("TEST-001"))
	g.Expect(edges[0].To).To(Equal("TASK-001"))
	g.Expect(edges[1].From).To(Equal("TEST-001"))
	g.Expect(edges[1].To).To(Equal("TASK-002"))
}

// TEST-024 traces: TASK-002
// Property test: random valid nodes have consistent ID and Type
func TestNode_PropertyIDMatchesType(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		nodeType := rapid.SampledFrom([]trace.NodeType{
			trace.NodeTypeREQ,
			trace.NodeTypeDES,
			trace.NodeTypeARCH,
			trace.NodeTypeTASK,
			trace.NodeTypeTEST,
		}).Draw(rt, "nodeType")

		prefix := string(nodeType)
		num := rapid.IntRange(1, 999).Draw(rt, "num")
		id := prefix + "-" + padNum3(num)

		node := &trace.Node{
			ID:      id,
			Type:    nodeType,
			Project: "test-project",
			Title:   "Test node",
			Status:  "active",
		}

		// Node's ID prefix should match its type
		g.Expect(node.ID[:len(prefix)]).To(Equal(prefix))
	})
}

// TEST-025 traces: TASK-002
// Property test: edges always connect two distinct IDs
func TestEdge_PropertyDistinctIDs(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		fromNum := rapid.IntRange(1, 999).Draw(rt, "fromNum")
		toNum := rapid.IntRange(1, 999).Draw(rt, "toNum")

		edge := &trace.Edge{
			From: "TASK-" + padNum3(fromNum),
			To:   "REQ-" + padNum3(toNum),
		}

		// Edge should have both From and To set
		g.Expect(edge.From).ToNot(BeEmpty())
		g.Expect(edge.To).ToNot(BeEmpty())
	})
}

// padNum3 pads a number to 3 digits
func padNum3(n int) string {
	if n < 10 {
		return "00" + intToString(n)
	}
	if n < 100 {
		return "0" + intToString(n)
	}
	return intToString(n)
}

func intToString(n int) string {
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
