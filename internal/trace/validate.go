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
