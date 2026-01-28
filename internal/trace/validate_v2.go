package trace

// ValidateV2Result contains the results of graph-based validation.
type ValidateV2Result struct {
	Pass      bool     `json:"pass"`
	Errors    []string `json:"errors,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
	NodeCount int      `json:"node_count"`
}

// ValidateV2 validates traceability using the graph-based system.
// Takes pre-collected trace items, builds a graph, and validates.
func ValidateV2(items []*TraceItem) (*ValidateV2Result, error) {
	// 1. Build the graph
	graph, buildWarnings, err := BuildGraph(items)
	if err != nil {
		return nil, err
	}

	// 2. Validate the graph
	validation := ValidateGraph(graph)

	// 3. Combine results
	result := &ValidateV2Result{
		Pass:      validation.Pass,
		Errors:    validation.Errors,
		Warnings:  validation.Warnings,
		NodeCount: len(graph.Nodes),
	}

	// Add build warnings to result warnings
	result.Warnings = append(result.Warnings, buildWarnings...)

	return result, nil
}
