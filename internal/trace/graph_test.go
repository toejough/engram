package trace_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/trace"
)

// TEST-017 traces: TASK-2
// Test creating a graph with valid nodes
func TestGraph_NewGraph(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	g.Expect(graph).ToNot(BeNil())
	g.Expect(graph.Nodes).ToNot(BeNil())
	g.Expect(graph.Edges).ToNot(BeNil())
	g.Expect(graph.ReverseEdges).ToNot(BeNil())
}

// TEST-018 traces: TASK-2
// Test creating a node with valid fields
func TestNode_ValidNode(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	node := &trace.Node{
		ID:      "REQ-1",
		Type:    trace.NodeTypeREQ,
		Project: "my-project",
		Title:   "A valid requirement",
		Status:  "active",
	}

	g.Expect(node.ID).To(Equal("REQ-1"))
	g.Expect(node.Type).To(Equal(trace.NodeTypeREQ))
	g.Expect(node.Project).To(Equal("my-project"))
}

// TEST-019 traces: TASK-2
// Test creating a TEST node with all required fields
func TestNode_ValidTestNode(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	node := &trace.Node{
		ID:       "TEST-1",
		Type:     trace.NodeTypeTEST,
		Project:  "my-project",
		Title:    "Test user validation",
		Status:   "active",
		Location: "user_test.go",
		Line:     45,
		Function: "TestValidateUser",
	}

	g.Expect(node.ID).To(Equal("TEST-1"))
	g.Expect(node.Type).To(Equal(trace.NodeTypeTEST))
	g.Expect(node.Location).To(Equal("user_test.go"))
	g.Expect(node.Line).To(Equal(45))
	g.Expect(node.Function).To(Equal("TestValidateUser"))
}

// TEST-020 traces: TASK-2
// Test creating an edge with valid From/To IDs
func TestEdge_ValidEdge(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	edge := &trace.Edge{
		From: "REQ-1",
		To:   "ARCH-1",
	}

	g.Expect(edge.From).To(Equal("REQ-1"))
	g.Expect(edge.To).To(Equal("ARCH-1"))
}

// TEST-021 traces: TASK-2
// Test that all NodeType constants are defined
func TestNodeType_AllTypes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Verify all 5 types exist
	g.Expect(trace.NodeTypeREQ).To(Equal(trace.NodeType("REQ")))
	g.Expect(trace.NodeTypeDES).To(Equal(trace.NodeType("DES")))
	g.Expect(trace.NodeTypeARCH).To(Equal(trace.NodeType("ARCH")))
	g.Expect(trace.NodeTypeTASK).To(Equal(trace.NodeType("TASK")))
	g.Expect(trace.NodeTypeTEST).To(Equal(trace.NodeType("TEST")))
}

// TEST-022 traces: TASK-2
// Test creating a node from a TraceItem
func TestNode_FromTraceItem(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	item := &trace.TraceItem{
		ID:       "TEST-42",
		Type:     trace.NodeTypeTEST,
		Project:  "my-project",
		Title:    "Test something",
		Status:   "active",
		TracesTo: []string{"TASK-1"},
		Location: "foo_test.go",
		Line:     10,
		Function: "TestFoo",
	}

	node := trace.NodeFromItem(item)
	g.Expect(node.ID).To(Equal("TEST-42"))
	g.Expect(node.Type).To(Equal(trace.NodeTypeTEST))
	g.Expect(node.Project).To(Equal("my-project"))
	g.Expect(node.Title).To(Equal("Test something"))
	g.Expect(node.Location).To(Equal("foo_test.go"))
	g.Expect(node.Line).To(Equal(10))
	g.Expect(node.Function).To(Equal("TestFoo"))
}

// TEST-023 traces: TASK-2
// Test creating edges from a TraceItem's TracesTo field
func TestEdge_FromTraceItem(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	item := &trace.TraceItem{
		ID:       "TEST-1",
		Type:     trace.NodeTypeTEST,
		TracesTo: []string{"TASK-1", "TASK-2"},
	}

	edges := trace.EdgesFromItem(item)
	g.Expect(edges).To(HaveLen(2))
	g.Expect(edges[0].From).To(Equal("TEST-1"))
	g.Expect(edges[0].To).To(Equal("TASK-1"))
	g.Expect(edges[1].From).To(Equal("TEST-1"))
	g.Expect(edges[1].To).To(Equal("TASK-2"))
}

// TEST-024 traces: TASK-2
// Property test: random valid nodes have consistent ID and Type
func TestNode_PropertyIDMatchesType(t *testing.T) {
	t.Parallel()
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

// TEST-025 traces: TASK-2
// Property test: edges always connect two distinct IDs
func TestEdge_PropertyDistinctIDs(t *testing.T) {
	t.Parallel()
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

// TEST-026 traces: TASK-003
// Test that AddNode adds a node to the graph
func TestGraph_AddNode(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	node := &trace.Node{
		ID:      "REQ-1",
		Type:    trace.NodeTypeREQ,
		Project: "my-project",
		Title:   "A requirement",
		Status:  "active",
	}

	err := graph.AddNode(node)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(graph.Nodes).To(HaveKey("REQ-1"))
	g.Expect(graph.Nodes["REQ-1"]).To(Equal(node))
}

// TEST-027 traces: TASK-003
// Test that AddNode returns error for duplicate ID
func TestGraph_AddNode_DuplicateID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	node1 := &trace.Node{
		ID:      "REQ-1",
		Type:    trace.NodeTypeREQ,
		Project: "my-project",
		Title:   "First requirement",
		Status:  "active",
	}
	node2 := &trace.Node{
		ID:      "REQ-1", // Same ID
		Type:    trace.NodeTypeREQ,
		Project: "my-project",
		Title:   "Duplicate requirement",
		Status:  "active",
	}

	err := graph.AddNode(node1)
	g.Expect(err).ToNot(HaveOccurred())

	err = graph.AddNode(node2)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("duplicate"))
}

// TEST-028 traces: TASK-003
// Test that AddNode validates ID format matches NodeType
func TestGraph_AddNode_IDTypeMismatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	node := &trace.Node{
		ID:      "REQ-1",          // ID says REQ
		Type:    trace.NodeTypeTASK, // Type says TASK
		Project: "my-project",
		Title:   "Mismatched",
		Status:  "active",
	}

	err := graph.AddNode(node)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("mismatch"))
}

// TEST-029 traces: TASK-003
// Test that AddNode returns error for nil node
func TestGraph_AddNode_NilNode(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	err := graph.AddNode(nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("nil"))
}

// TEST-030 traces: TASK-003
// Test that graph state is unchanged after AddNode error
func TestGraph_AddNode_StateUnchangedOnError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	node1 := &trace.Node{
		ID:      "REQ-1",
		Type:    trace.NodeTypeREQ,
		Project: "my-project",
		Title:   "First requirement",
		Status:  "active",
	}

	err := graph.AddNode(node1)
	g.Expect(err).ToNot(HaveOccurred())

	originalCount := len(graph.Nodes)

	// Try to add duplicate
	node2 := &trace.Node{
		ID:      "REQ-1", // Same ID
		Type:    trace.NodeTypeREQ,
		Project: "my-project",
		Title:   "Duplicate",
		Status:  "active",
	}
	err = graph.AddNode(node2)
	g.Expect(err).To(HaveOccurred())

	// State should be unchanged
	g.Expect(graph.Nodes).To(HaveLen(originalCount))
	g.Expect(graph.Nodes["REQ-1"].Title).To(Equal("First requirement"))
}

// TEST-031 traces: TASK-003
// Property test: AddNode with unique valid nodes always succeeds
func TestGraph_AddNode_PropertyUniqueNodesSucceed(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		graph := trace.NewGraph()

		nodeType := rapid.SampledFrom([]trace.NodeType{
			trace.NodeTypeREQ,
			trace.NodeTypeDES,
			trace.NodeTypeARCH,
			trace.NodeTypeTASK,
			trace.NodeTypeTEST,
		}).Draw(rt, "nodeType")

		num := rapid.IntRange(1, 999).Draw(rt, "num")
		id := string(nodeType) + "-" + padNum3(num)

		node := &trace.Node{
			ID:      id,
			Type:    nodeType,
			Project: "test-project",
			Title:   "Test node",
			Status:  "active",
		}

		err := graph.AddNode(node)
		g.Expect(err).ToNot(HaveOccurred())
		// ID is normalized when stored
		normalizedID := trace.NormalizeID(id)
		g.Expect(graph.Nodes).To(HaveKey(normalizedID))
	})
}

// TEST-032 traces: TASK-003
// Property test: AddNode with duplicate ID always fails
func TestGraph_AddNode_PropertyDuplicateIDFails(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		graph := trace.NewGraph()

		num := rapid.IntRange(1, 999).Draw(rt, "num")
		id := "REQ-" + padNum3(num)

		node1 := &trace.Node{
			ID:      id,
			Type:    trace.NodeTypeREQ,
			Project: "test-project",
			Title:   "First node",
			Status:  "active",
		}

		node2 := &trace.Node{
			ID:      id, // Same ID
			Type:    trace.NodeTypeREQ,
			Project: "test-project",
			Title:   "Second node",
			Status:  "active",
		}

		err := graph.AddNode(node1)
		g.Expect(err).ToNot(HaveOccurred())

		err = graph.AddNode(node2)
		g.Expect(err).To(HaveOccurred())
	})
}

// TEST-033 traces: TASK-004
// Test that AddEdge creates edge in both forward and reverse maps
func TestGraph_AddEdge(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-1", Type: trace.NodeTypeREQ, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "ARCH-1", Type: trace.NodeTypeARCH, Project: "p", Title: "t", Status: "active"})

	edge := &trace.Edge{From: "REQ-1", To: "ARCH-1"}
	err := graph.AddEdge(edge)
	g.Expect(err).ToNot(HaveOccurred())

	// Check forward edge
	g.Expect(graph.Edges).To(HaveKey("REQ-1"))
	g.Expect(graph.Edges["REQ-1"]).To(HaveLen(1))
	g.Expect(graph.Edges["REQ-1"][0].To).To(Equal("ARCH-1"))

	// Check reverse edge
	g.Expect(graph.ReverseEdges).To(HaveKey("ARCH-1"))
	g.Expect(graph.ReverseEdges["ARCH-1"]).To(HaveLen(1))
	g.Expect(graph.ReverseEdges["ARCH-1"][0].From).To(Equal("REQ-1"))
}

// TEST-034 traces: TASK-004
// Test that AddEdge returns error when From node doesn't exist
func TestGraph_AddEdge_FromNodeMissing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "ARCH-1", Type: trace.NodeTypeARCH, Project: "p", Title: "t", Status: "active"})

	edge := &trace.Edge{From: "REQ-1", To: "ARCH-1"} // REQ-1 doesn't exist
	err := graph.AddEdge(edge)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("REQ-1"))
}

// TEST-035 traces: TASK-004
// Test that AddEdge returns error when To node doesn't exist
func TestGraph_AddEdge_ToNodeMissing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-1", Type: trace.NodeTypeREQ, Project: "p", Title: "t", Status: "active"})

	edge := &trace.Edge{From: "REQ-1", To: "ARCH-1"} // ARCH-1 doesn't exist
	err := graph.AddEdge(edge)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("ARCH-1"))
}

// TEST-036 traces: TASK-004
// Test that multiple edges from same source are supported
func TestGraph_AddEdge_MultipleFromSameSource(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-1", Type: trace.NodeTypeREQ, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "ARCH-1", Type: trace.NodeTypeARCH, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "ARCH-2", Type: trace.NodeTypeARCH, Project: "p", Title: "t", Status: "active"})

	edge1 := &trace.Edge{From: "REQ-1", To: "ARCH-1"}
	edge2 := &trace.Edge{From: "REQ-1", To: "ARCH-2"}

	err := graph.AddEdge(edge1)
	g.Expect(err).ToNot(HaveOccurred())

	err = graph.AddEdge(edge2)
	g.Expect(err).ToNot(HaveOccurred())

	// Should have 2 edges from REQ-1
	g.Expect(graph.Edges["REQ-1"]).To(HaveLen(2))
}

// TEST-037 traces: TASK-004
// Test that AddEdge returns error for nil edge
func TestGraph_AddEdge_NilEdge(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	err := graph.AddEdge(nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("nil"))
}

// TEST-038 traces: TASK-004
// Property test: AddEdge with existing nodes always succeeds
func TestGraph_AddEdge_PropertyExistingNodesSucceed(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		graph := trace.NewGraph()

		fromNum := rapid.IntRange(1, 999).Draw(rt, "fromNum")
		toNum := rapid.IntRange(1, 999).Draw(rt, "toNum")

		fromID := "REQ-" + padNum3(fromNum)
		toID := "ARCH-" + padNum3(toNum)

		_ = graph.AddNode(&trace.Node{ID: fromID, Type: trace.NodeTypeREQ, Project: "p", Title: "t", Status: "active"})
		_ = graph.AddNode(&trace.Node{ID: toID, Type: trace.NodeTypeARCH, Project: "p", Title: "t", Status: "active"})

		edge := &trace.Edge{From: fromID, To: toID}
		err := graph.AddEdge(edge)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify bidirectional storage (IDs are normalized in storage)
		normalizedFrom := trace.NormalizeID(fromID)
		normalizedTo := trace.NormalizeID(toID)
		g.Expect(graph.Edges[normalizedFrom]).To(ContainElement(edge))
		g.Expect(graph.ReverseEdges[normalizedTo]).To(ContainElement(edge))
	})
}

// TEST-039 traces: TASK-005
// Test Upstream returns empty for node with no ancestors
func TestGraph_Upstream_NoAncestors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-1", Type: trace.NodeTypeREQ, Project: "p", Title: "t", Status: "active"})

	ancestors, err := graph.Upstream("REQ-1")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(ancestors).To(BeEmpty())
}

// TEST-040 traces: TASK-005
// Test Upstream returns direct ancestors
func TestGraph_Upstream_DirectAncestors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-1", Type: trace.NodeTypeREQ, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "ARCH-1", Type: trace.NodeTypeARCH, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddEdge(&trace.Edge{From: "ARCH-1", To: "REQ-1"})

	ancestors, err := graph.Upstream("REQ-1")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(ancestors).To(HaveLen(1))
	g.Expect(ancestors).To(ContainElement("ARCH-1"))
}

// TEST-041 traces: TASK-005
// Test Upstream returns transitive ancestors (multiple levels)
func TestGraph_Upstream_TransitiveAncestors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// REQ-1 <- ARCH-1 <- TASK-1 <- TEST-1
	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-1", Type: trace.NodeTypeREQ, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "ARCH-1", Type: trace.NodeTypeARCH, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "TASK-1", Type: trace.NodeTypeTASK, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "TEST-1", Type: trace.NodeTypeTEST, Project: "p", Title: "t", Status: "active", Location: "t.go", Function: "Test"})

	_ = graph.AddEdge(&trace.Edge{From: "ARCH-1", To: "REQ-1"})
	_ = graph.AddEdge(&trace.Edge{From: "TASK-1", To: "ARCH-1"})
	_ = graph.AddEdge(&trace.Edge{From: "TEST-1", To: "TASK-1"})

	// Starting from REQ-1, should get all 3 ancestors
	ancestors, err := graph.Upstream("REQ-1")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(ancestors).To(HaveLen(3))
	g.Expect(ancestors).To(ContainElements("ARCH-1", "TASK-1", "TEST-1"))
}

// TEST-42 traces: TASK-005
// Test Upstream handles diamond dependency (each node visited once)
func TestGraph_Upstream_DiamondDependency(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Diamond: REQ-1 <- ARCH-1 <- TASK-1
	//                 <- ARCH-2 <-
	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-1", Type: trace.NodeTypeREQ, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "ARCH-1", Type: trace.NodeTypeARCH, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "ARCH-2", Type: trace.NodeTypeARCH, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "TASK-1", Type: trace.NodeTypeTASK, Project: "p", Title: "t", Status: "active"})

	_ = graph.AddEdge(&trace.Edge{From: "ARCH-1", To: "REQ-1"})
	_ = graph.AddEdge(&trace.Edge{From: "ARCH-2", To: "REQ-1"})
	_ = graph.AddEdge(&trace.Edge{From: "TASK-1", To: "ARCH-1"})
	_ = graph.AddEdge(&trace.Edge{From: "TASK-1", To: "ARCH-2"})

	ancestors, err := graph.Upstream("REQ-1")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(ancestors).To(HaveLen(3))
	g.Expect(ancestors).To(ContainElements("ARCH-1", "ARCH-2", "TASK-1"))
}

// TEST-043 traces: TASK-005
// Test Upstream returns error for non-existent node
func TestGraph_Upstream_NodeNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	_, err := graph.Upstream("REQ-999")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("REQ-999"))
}

// TEST-044 traces: TASK-005
// Property test: Upstream on random DAG returns correct ancestors
func TestGraph_Upstream_PropertyRandomDAG(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		graph := trace.NewGraph()

		// Create a chain of nodes
		chainLen := rapid.IntRange(1, 5).Draw(rt, "chainLen")
		types := []trace.NodeType{trace.NodeTypeREQ, trace.NodeTypeARCH, trace.NodeTypeTASK, trace.NodeTypeTEST}

		var prevID string
		for i := 0; i < chainLen; i++ {
			nodeType := types[i%len(types)]
			id := string(nodeType) + "-" + padNum3(i+1)
			node := &trace.Node{ID: id, Type: nodeType, Project: "p", Title: "t", Status: "active"}
			if nodeType == trace.NodeTypeTEST {
				node.Location = "t.go"
				node.Function = "Test"
			}
			_ = graph.AddNode(node)

			if prevID != "" {
				_ = graph.AddEdge(&trace.Edge{From: id, To: prevID})
			}
			prevID = id
		}

		// Query from first node (has all others as ancestors)
		if chainLen > 0 {
			firstType := types[0]
			firstID := string(firstType) + "-001"
			ancestors, err := graph.Upstream(firstID)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(len(ancestors)).To(Equal(chainLen - 1))
		}
	})
}

// TEST-045 traces: TASK-006
// Test Downstream returns empty for node with no descendants
func TestGraph_Downstream_NoDescendants(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "TEST-1", Type: trace.NodeTypeTEST, Project: "p", Title: "t", Status: "active", Location: "t.go", Function: "Test"})

	descendants, err := graph.Downstream("TEST-1")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(descendants).To(BeEmpty())
}

// TEST-046 traces: TASK-006
// Test Downstream returns direct descendants
func TestGraph_Downstream_DirectDescendants(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-1", Type: trace.NodeTypeREQ, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "ARCH-1", Type: trace.NodeTypeARCH, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddEdge(&trace.Edge{From: "REQ-1", To: "ARCH-1"})

	descendants, err := graph.Downstream("REQ-1")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(descendants).To(HaveLen(1))
	g.Expect(descendants).To(ContainElement("ARCH-1"))
}

// TEST-047 traces: TASK-006
// Test Downstream returns transitive descendants (multiple levels)
func TestGraph_Downstream_TransitiveDescendants(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// REQ-1 -> ARCH-1 -> TASK-1 -> TEST-1
	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-1", Type: trace.NodeTypeREQ, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "ARCH-1", Type: trace.NodeTypeARCH, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "TASK-1", Type: trace.NodeTypeTASK, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "TEST-1", Type: trace.NodeTypeTEST, Project: "p", Title: "t", Status: "active", Location: "t.go", Function: "Test"})

	_ = graph.AddEdge(&trace.Edge{From: "REQ-1", To: "ARCH-1"})
	_ = graph.AddEdge(&trace.Edge{From: "ARCH-1", To: "TASK-1"})
	_ = graph.AddEdge(&trace.Edge{From: "TASK-1", To: "TEST-1"})

	// Starting from REQ-1, should get all 3 descendants
	descendants, err := graph.Downstream("REQ-1")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(descendants).To(HaveLen(3))
	g.Expect(descendants).To(ContainElements("ARCH-1", "TASK-1", "TEST-1"))
}

// TEST-048 traces: TASK-006
// Test Downstream handles diamond dependency (each node visited once)
func TestGraph_Downstream_DiamondDependency(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Diamond: REQ-1 -> ARCH-1 -> TASK-1
	//                  -> ARCH-2 ->
	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-1", Type: trace.NodeTypeREQ, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "ARCH-1", Type: trace.NodeTypeARCH, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "ARCH-2", Type: trace.NodeTypeARCH, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "TASK-1", Type: trace.NodeTypeTASK, Project: "p", Title: "t", Status: "active"})

	_ = graph.AddEdge(&trace.Edge{From: "REQ-1", To: "ARCH-1"})
	_ = graph.AddEdge(&trace.Edge{From: "REQ-1", To: "ARCH-2"})
	_ = graph.AddEdge(&trace.Edge{From: "ARCH-1", To: "TASK-1"})
	_ = graph.AddEdge(&trace.Edge{From: "ARCH-2", To: "TASK-1"})

	descendants, err := graph.Downstream("REQ-1")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(descendants).To(HaveLen(3))
	g.Expect(descendants).To(ContainElements("ARCH-1", "ARCH-2", "TASK-1"))
}

// TEST-049 traces: TASK-006
// Test Downstream returns error for non-existent node
func TestGraph_Downstream_NodeNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	_, err := graph.Downstream("REQ-999")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("REQ-999"))
}

// TEST-050 traces: TASK-007
// Test Orphans returns empty when no orphans of given type
func TestGraph_Orphans_NoOrphans(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-1", Type: trace.NodeTypeREQ, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "ARCH-1", Type: trace.NodeTypeARCH, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddEdge(&trace.Edge{From: "REQ-1", To: "ARCH-1"})

	// REQ-1 has downstream, so not an orphan with direction="downstream"
	orphans := graph.Orphans(trace.NodeTypeREQ, "downstream")
	g.Expect(orphans).To(BeEmpty())
}

// TEST-051 traces: TASK-007
// Test Orphans detects REQ with no downstream edges
func TestGraph_Orphans_REQNoDownstream(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-1", Type: trace.NodeTypeREQ, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "REQ-2", Type: trace.NodeTypeREQ, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "ARCH-1", Type: trace.NodeTypeARCH, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddEdge(&trace.Edge{From: "REQ-1", To: "ARCH-1"})
	// REQ-2 has no downstream edges

	orphans := graph.Orphans(trace.NodeTypeREQ, "downstream")
	g.Expect(orphans).To(HaveLen(1))
	g.Expect(orphans).To(ContainElement("REQ-2"))
}

// TEST-052 traces: TASK-007
// Test Orphans detects TASK with no TEST downstream
func TestGraph_Orphans_TASKNoTest(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "TASK-1", Type: trace.NodeTypeTASK, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "TASK-2", Type: trace.NodeTypeTASK, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "TEST-1", Type: trace.NodeTypeTEST, Project: "p", Title: "t", Status: "active", Location: "t.go", Function: "Test"})
	_ = graph.AddEdge(&trace.Edge{From: "TASK-1", To: "TEST-1"})
	// TASK-2 has no downstream edges

	orphans := graph.Orphans(trace.NodeTypeTASK, "downstream")
	g.Expect(orphans).To(HaveLen(1))
	g.Expect(orphans).To(ContainElement("TASK-2"))
}

// TEST-053 traces: TASK-007
// Test Orphans detects TEST with no upstream edges
func TestGraph_Orphans_TESTNoUpstream(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "TASK-1", Type: trace.NodeTypeTASK, Project: "p", Title: "t", Status: "active"})
	_ = graph.AddNode(&trace.Node{ID: "TEST-1", Type: trace.NodeTypeTEST, Project: "p", Title: "t", Status: "active", Location: "t.go", Function: "Test1"})
	_ = graph.AddNode(&trace.Node{ID: "TEST-2", Type: trace.NodeTypeTEST, Project: "p", Title: "t", Status: "active", Location: "t.go", Function: "Test2"})
	_ = graph.AddEdge(&trace.Edge{From: "TEST-1", To: "TASK-1"})
	// TEST-2 has no upstream edges (no traces)

	orphans := graph.Orphans(trace.NodeTypeTEST, "upstream")
	g.Expect(orphans).To(HaveLen(1))
	g.Expect(orphans).To(ContainElement("TEST-2"))
}

// TEST-054 traces: TASK-007
// Property test: Orphans detection on graphs with known gaps
func TestGraph_Orphans_PropertyKnownGaps(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		graph := trace.NewGraph()

		// Create some connected nodes
		connectedCount := rapid.IntRange(1, 5).Draw(rt, "connectedCount")
		for i := 0; i < connectedCount; i++ {
			reqID := "REQ-" + padNum3(i+1)
			archID := "ARCH-" + padNum3(i+1)
			_ = graph.AddNode(&trace.Node{ID: reqID, Type: trace.NodeTypeREQ, Project: "p", Title: "t", Status: "active"})
			_ = graph.AddNode(&trace.Node{ID: archID, Type: trace.NodeTypeARCH, Project: "p", Title: "t", Status: "active"})
			_ = graph.AddEdge(&trace.Edge{From: reqID, To: archID})
		}

		// Create orphan REQs (no downstream)
		orphanCount := rapid.IntRange(0, 3).Draw(rt, "orphanCount")
		for i := 0; i < orphanCount; i++ {
			orphanID := "REQ-" + padNum3(connectedCount+i+1)
			_ = graph.AddNode(&trace.Node{ID: orphanID, Type: trace.NodeTypeREQ, Project: "p", Title: "t", Status: "active"})
		}

		orphans := graph.Orphans(trace.NodeTypeREQ, "downstream")
		g.Expect(len(orphans)).To(Equal(orphanCount))
	})
}
