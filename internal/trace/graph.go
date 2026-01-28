package trace

import (
	"fmt"
	"strings"
)

// Graph represents the complete traceability graph.
type Graph struct {
	Nodes        map[string]*Node  // ID -> Node
	Edges        map[string][]*Edge // From ID -> [Edges]
	ReverseEdges map[string][]*Edge // To ID -> [Edges] (for upstream queries)
}

// Node represents a traceability item in the graph.
type Node struct {
	ID      string   // e.g., "TASK-003", "TEST-001"
	Type    NodeType // REQ, DES, ARCH, TASK, TEST
	Project string   // Project identifier
	Title   string   // One-line summary
	Status  string   // draft|active|completed|deprecated
	Tags    []string // Optional metadata

	// TEST-specific fields (empty for other types)
	Location string // File path (e.g., "user_test.go")
	Line     int    // Line number
	Function string // Test function name
}

// Edge represents a traces-to relationship.
type Edge struct {
	From string // Source node ID
	To   string // Target node ID
}

// NewGraph creates a new empty graph with initialized maps.
func NewGraph() *Graph {
	return &Graph{
		Nodes:        make(map[string]*Node),
		Edges:        make(map[string][]*Edge),
		ReverseEdges: make(map[string][]*Edge),
	}
}

// NodeFromItem creates a Node from a TraceItem.
func NodeFromItem(item *TraceItem) *Node {
	return &Node{
		ID:       item.ID,
		Type:     item.Type,
		Project:  item.Project,
		Title:    item.Title,
		Status:   item.Status,
		Tags:     item.Tags,
		Location: item.Location,
		Line:     item.Line,
		Function: item.Function,
	}
}

// EdgesFromItem creates Edge objects from a TraceItem's TracesTo field.
func EdgesFromItem(item *TraceItem) []*Edge {
	edges := make([]*Edge, 0, len(item.TracesTo))
	for _, target := range item.TracesTo {
		edges = append(edges, &Edge{
			From: item.ID,
			To:   target,
		})
	}
	return edges
}

// AddNode inserts a node into the graph.
// Returns error if node is nil, has duplicate ID, or ID doesn't match Type.
func (g *Graph) AddNode(n *Node) error {
	if n == nil {
		return fmt.Errorf("node is nil")
	}

	// Validate ID prefix matches Type
	expectedPrefix := string(n.Type) + "-"
	if !strings.HasPrefix(n.ID, expectedPrefix) {
		return fmt.Errorf("id prefix mismatch: ID %q does not match Type %q", n.ID, n.Type)
	}

	// Check for duplicate
	if _, exists := g.Nodes[n.ID]; exists {
		return fmt.Errorf("duplicate node ID: %s", n.ID)
	}

	g.Nodes[n.ID] = n
	return nil
}

// AddEdge inserts an edge into the graph.
// Returns error if edge is nil or either From/To nodes don't exist.
func (g *Graph) AddEdge(e *Edge) error {
	if e == nil {
		return fmt.Errorf("edge is nil")
	}

	// Validate From node exists
	if _, exists := g.Nodes[e.From]; !exists {
		return fmt.Errorf("from node not found: %s", e.From)
	}

	// Validate To node exists
	if _, exists := g.Nodes[e.To]; !exists {
		return fmt.Errorf("to node not found: %s", e.To)
	}

	// Add to forward adjacency list
	g.Edges[e.From] = append(g.Edges[e.From], e)

	// Add to reverse adjacency list
	g.ReverseEdges[e.To] = append(g.ReverseEdges[e.To], e)

	return nil
}

// Upstream finds all ancestor nodes (what this node traces to) recursively.
// Uses ReverseEdges to traverse upstream.
// Returns error if source node doesn't exist.
func (g *Graph) Upstream(nodeID string) ([]string, error) {
	// TODO: Implement
	return nil, nil
}
