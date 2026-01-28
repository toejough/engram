package trace

import (
	"fmt"
	"strings"
)

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

// ValidationResult contains the results of graph validation.
type ValidationResult struct {
	Pass     bool     // True if validation passed (no errors)
	Errors   []string // Validation errors (cause failure)
	Warnings []string // Validation warnings (informational)
}

// ValidateGraph runs all validation checks on the graph.
// Returns ValidationResult with pass/fail status, errors, and warnings.
func ValidateGraph(graph *Graph) *ValidationResult {
	_ = strings.TrimSpace // Silence unused import
	result := &ValidationResult{}

	// Check for cycles (error)
	if hasCycle, cyclePath := DetectCycle(graph); hasCycle {
		result.Errors = append(result.Errors, "cycle detected: "+strings.Join(cyclePath, " -> "))
	}

	// Check for dangling references (error)
	danglingErrors := ValidateDanglingRefs(graph)
	result.Errors = append(result.Errors, danglingErrors...)

	// Check coverage gaps (warning)
	coverageWarnings := ValidateCoverage(graph)
	result.Warnings = append(result.Warnings, coverageWarnings...)

	// Pass only if no errors
	result.Pass = len(result.Errors) == 0

	return result
}
