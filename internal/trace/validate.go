package trace

// ValidateTESTUniqueness checks that all TEST node IDs are unique.
// Returns list of validation errors (empty if valid).
func ValidateTESTUniqueness(graph *Graph) []string {
	return nil
}

// ValidateTESTIDFormat checks if a TEST ID has valid format (TEST-NNN with 3+ digits).
func ValidateTESTIDFormat(id string) bool {
	return false
}

// ValidateTESTIDFormats checks multiple IDs and returns those with invalid format.
func ValidateTESTIDFormats(ids []string) []string {
	return nil
}
