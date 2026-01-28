package trace_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/trace"
)

// TEST-139 traces: TASK-021
// Test no cycle in acyclic graph
func TestDetectCycle_Acyclic(t *testing.T) {
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-001", Type: trace.NodeTypeREQ})
	_ = graph.AddNode(&trace.Node{ID: "TASK-001", Type: trace.NodeTypeTASK})
	_ = graph.AddEdge(&trace.Edge{From: "TASK-001", To: "REQ-001"})

	hasCycle, path := trace.DetectCycle(graph)
	g.Expect(hasCycle).To(BeFalse())
	g.Expect(path).To(BeEmpty())
}

// TEST-140 traces: TASK-021
// Test detects simple cycle
func TestDetectCycle_SimpleCycle(t *testing.T) {
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-001", Type: trace.NodeTypeREQ})
	_ = graph.AddNode(&trace.Node{ID: "REQ-002", Type: trace.NodeTypeREQ})
	_ = graph.AddEdge(&trace.Edge{From: "REQ-001", To: "REQ-002"})
	_ = graph.AddEdge(&trace.Edge{From: "REQ-002", To: "REQ-001"})

	hasCycle, path := trace.DetectCycle(graph)
	g.Expect(hasCycle).To(BeTrue())
	g.Expect(path).To(HaveLen(2))
}

// TEST-141 traces: TASK-021
// Test detects self-loop
func TestDetectCycle_SelfLoop(t *testing.T) {
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-001", Type: trace.NodeTypeREQ})
	_ = graph.AddEdge(&trace.Edge{From: "REQ-001", To: "REQ-001"})

	hasCycle, path := trace.DetectCycle(graph)
	g.Expect(hasCycle).To(BeTrue())
	g.Expect(path).To(ContainElement("REQ-001"))
}

// TEST-142 traces: TASK-021
// Test diamond dependency is not a cycle
func TestDetectCycle_Diamond(t *testing.T) {
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-001", Type: trace.NodeTypeREQ})
	_ = graph.AddNode(&trace.Node{ID: "TASK-001", Type: trace.NodeTypeTASK})
	_ = graph.AddNode(&trace.Node{ID: "TASK-002", Type: trace.NodeTypeTASK})
	_ = graph.AddNode(&trace.Node{ID: "ARCH-001", Type: trace.NodeTypeARCH})
	// Diamond: REQ-001 <- TASK-001 <- ARCH-001 and REQ-001 <- TASK-002 <- ARCH-001
	_ = graph.AddEdge(&trace.Edge{From: "TASK-001", To: "REQ-001"})
	_ = graph.AddEdge(&trace.Edge{From: "TASK-002", To: "REQ-001"})
	_ = graph.AddEdge(&trace.Edge{From: "ARCH-001", To: "TASK-001"})
	_ = graph.AddEdge(&trace.Edge{From: "ARCH-001", To: "TASK-002"})

	hasCycle, path := trace.DetectCycle(graph)
	g.Expect(hasCycle).To(BeFalse())
	g.Expect(path).To(BeEmpty())
}

// TEST-143 traces: TASK-021
// Test empty graph has no cycle
func TestDetectCycle_EmptyGraph(t *testing.T) {
	g := NewWithT(t)

	graph := trace.NewGraph()

	hasCycle, path := trace.DetectCycle(graph)
	g.Expect(hasCycle).To(BeFalse())
	g.Expect(path).To(BeEmpty())
}

// TEST-144 traces: TASK-021
// Test detects longer cycle
func TestDetectCycle_LongerCycle(t *testing.T) {
	g := NewWithT(t)

	graph := trace.NewGraph()
	_ = graph.AddNode(&trace.Node{ID: "REQ-001", Type: trace.NodeTypeREQ})
	_ = graph.AddNode(&trace.Node{ID: "REQ-002", Type: trace.NodeTypeREQ})
	_ = graph.AddNode(&trace.Node{ID: "REQ-003", Type: trace.NodeTypeREQ})
	_ = graph.AddEdge(&trace.Edge{From: "REQ-001", To: "REQ-002"})
	_ = graph.AddEdge(&trace.Edge{From: "REQ-002", To: "REQ-003"})
	_ = graph.AddEdge(&trace.Edge{From: "REQ-003", To: "REQ-001"})

	hasCycle, path := trace.DetectCycle(graph)
	g.Expect(hasCycle).To(BeTrue())
	g.Expect(path).To(HaveLen(3))
}
