package trace_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/trace"
)

// TEST-139 traces: TASK-21
// Test no cycle in acyclic graph
func TestDetectCycle_Acyclic(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-1", Type: trace.NodeTypeREQ})
	_ = graph.AddNode(&trace.Node{ID: "TASK-1", Type: trace.NodeTypeTASK})
	_ = graph.AddEdge(&trace.Edge{From: "TASK-1", To: "REQ-1"})

	hasCycle, path := trace.DetectCycle(graph)
	g.Expect(hasCycle).To(BeFalse())
	g.Expect(path).To(BeEmpty())
}

// TEST-140 traces: TASK-21
// Test detects simple cycle
func TestDetectCycle_SimpleCycle(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-1", Type: trace.NodeTypeREQ})
	_ = graph.AddNode(&trace.Node{ID: "REQ-2", Type: trace.NodeTypeREQ})
	_ = graph.AddEdge(&trace.Edge{From: "REQ-1", To: "REQ-2"})
	_ = graph.AddEdge(&trace.Edge{From: "REQ-2", To: "REQ-1"})

	hasCycle, path := trace.DetectCycle(graph)
	g.Expect(hasCycle).To(BeTrue())
	g.Expect(path).To(HaveLen(2))
}

// TEST-141 traces: TASK-21
// Test detects self-loop
func TestDetectCycle_SelfLoop(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-1", Type: trace.NodeTypeREQ})
	_ = graph.AddEdge(&trace.Edge{From: "REQ-1", To: "REQ-1"})

	hasCycle, path := trace.DetectCycle(graph)
	g.Expect(hasCycle).To(BeTrue())
	g.Expect(path).To(ContainElement("REQ-1"))
}

// TEST-142 traces: TASK-21
// Test diamond dependency is not a cycle
func TestDetectCycle_Diamond(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-1", Type: trace.NodeTypeREQ})
	_ = graph.AddNode(&trace.Node{ID: "TASK-1", Type: trace.NodeTypeTASK})
	_ = graph.AddNode(&trace.Node{ID: "TASK-2", Type: trace.NodeTypeTASK})
	_ = graph.AddNode(&trace.Node{ID: "ARCH-1", Type: trace.NodeTypeARCH})
	// Diamond: REQ-1 <- TASK-1 <- ARCH-1 and REQ-1 <- TASK-2 <- ARCH-1
	_ = graph.AddEdge(&trace.Edge{From: "TASK-1", To: "REQ-1"})
	_ = graph.AddEdge(&trace.Edge{From: "TASK-2", To: "REQ-1"})
	_ = graph.AddEdge(&trace.Edge{From: "ARCH-1", To: "TASK-1"})
	_ = graph.AddEdge(&trace.Edge{From: "ARCH-1", To: "TASK-2"})

	hasCycle, path := trace.DetectCycle(graph)
	g.Expect(hasCycle).To(BeFalse())
	g.Expect(path).To(BeEmpty())
}

// TEST-143 traces: TASK-21
// Test empty graph has no cycle
func TestDetectCycle_EmptyGraph(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()

	hasCycle, path := trace.DetectCycle(graph)
	g.Expect(hasCycle).To(BeFalse())
	g.Expect(path).To(BeEmpty())
}

// TEST-144 traces: TASK-21
// Test detects longer cycle
func TestDetectCycle_LongerCycle(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-1", Type: trace.NodeTypeREQ})
	_ = graph.AddNode(&trace.Node{ID: "REQ-2", Type: trace.NodeTypeREQ})
	_ = graph.AddNode(&trace.Node{ID: "REQ-3", Type: trace.NodeTypeREQ})
	_ = graph.AddEdge(&trace.Edge{From: "REQ-1", To: "REQ-2"})
	_ = graph.AddEdge(&trace.Edge{From: "REQ-2", To: "REQ-3"})
	_ = graph.AddEdge(&trace.Edge{From: "REQ-3", To: "REQ-1"})

	hasCycle, path := trace.DetectCycle(graph)
	g.Expect(hasCycle).To(BeTrue())
	g.Expect(path).To(HaveLen(3))
}
