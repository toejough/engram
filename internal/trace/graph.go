package trace

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
