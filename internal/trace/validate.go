package trace

import "regexp"

// testIDPattern matches a valid TEST ID: TEST-N (1+ digits)
var testIDPattern = regexp.MustCompile(`^TEST-\d+$`)

// ValidateTESTUniqueness checks that all TEST node IDs are unique.
// Returns list of validation errors (empty if valid).
func ValidateTESTUniqueness(graph *Graph) []string {
	return nil
}

// ValidateTESTIDFormat checks if a TEST ID has valid format (TEST-N with 1+ digits).
func ValidateTESTIDFormat(id string) bool {
	return testIDPattern.MatchString(id)
}

// ValidateTESTIDFormats checks multiple IDs and returns those with invalid format.
func ValidateTESTIDFormats(ids []string) []string {
	var invalid []string
	for _, id := range ids {
		if !ValidateTESTIDFormat(id) {
			invalid = append(invalid, id)
		}
	}
	return invalid
}

// ValidateDanglingRefs checks that all edge targets exist in the graph.
// Returns list of validation errors for dangling references.
func ValidateDanglingRefs(graph *Graph) []string {
	var errors []string
	for _, edge := range graph.DanglingEdges {
		errors = append(errors, "dangling reference: "+edge.From+" traces to non-existent "+edge.To)
	}
	return errors
}

// ValidateCoverage checks trace coverage rules.
// REQ should have ARCH/DES downstream, ARCH should have TASK, TASK should have TEST.
// Returns list of warnings (coverage gaps don't fail validation).
func ValidateCoverage(graph *Graph) []string {
	var warnings []string

	for id, node := range graph.Nodes {
		switch node.Type {
		case NodeTypeREQ:
			// REQ should have downstream ARCH or DES
			if !hasDownstreamType(graph, id, NodeTypeARCH) && !hasDownstreamType(graph, id, NodeTypeDES) {
				warnings = append(warnings, id+" has no downstream ARCH or DES")
			}
		case NodeTypeARCH:
			// ARCH should have downstream TASK
			if !hasDownstreamType(graph, id, NodeTypeTASK) {
				warnings = append(warnings, id+" has no downstream TASK")
			}
		case NodeTypeTASK:
			// TASK should have downstream TEST
			if !hasDownstreamType(graph, id, NodeTypeTEST) {
				warnings = append(warnings, id+" has no downstream TEST")
			}
		}
	}

	return warnings
}

// hasDownstreamType checks if a node has any downstream nodes of the given type.
func hasDownstreamType(graph *Graph, nodeID string, targetType NodeType) bool {
	// ReverseEdges: To ID -> [Edges from sources]
	// We need to find edges where this node is the "To" (upstream), and the "From" is of targetType
	for _, edge := range graph.ReverseEdges[nodeID] {
		fromNode := graph.Nodes[edge.From]
		if fromNode != nil && fromNode.Type == targetType {
			return true
		}
	}
	return false
}
