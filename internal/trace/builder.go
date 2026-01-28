package trace

import "fmt"

// BuildGraph constructs a Graph from a slice of TraceItems.
// Returns the graph, any warnings (e.g., dangling edges), and error if build fails.
func BuildGraph(items []*TraceItem) (*Graph, []string, error) {
	graph := NewGraph()
	var warnings []string

	// First pass: add all nodes
	for _, item := range items {
		node := NodeFromItem(item)
		if err := graph.AddNode(node); err != nil {
			return nil, nil, fmt.Errorf("duplicate node ID: %s", item.ID)
		}
	}

	// Second pass: add edges
	for _, item := range items {
		edges := EdgesFromItem(item)
		for _, edge := range edges {
			if _, exists := graph.Nodes[edge.To]; !exists {
				warnings = append(warnings, fmt.Sprintf("dangling edge: %s traces to non-existent %s", edge.From, edge.To))
				graph.DanglingEdges = append(graph.DanglingEdges, edge)
				continue
			}
			// Edge targets exist, safe to add
			_ = graph.AddEdge(edge)
		}
	}

	return graph, warnings, nil
}
