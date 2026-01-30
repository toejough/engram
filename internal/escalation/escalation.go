// Package escalation handles user escalation workflow for project adoption.
package escalation

// EscalationFS provides file system operations for escalation handling.
type EscalationFS interface {
	WriteFile(path string, content string) error
	ReadFile(path string) (string, error)
}

// Escalation represents a question that needs user input.
type Escalation struct {
	ID       string // Unique ID (ESC-NNN)
	Category string // requirement, design, architecture, task
	Context  string // What was being analyzed
	Question string // The question needing resolution
	Status   string // pending, resolved, deferred, issue
	Notes    string // User's answer or notes
}

// IsValidStatus checks if a status string is valid.
func IsValidStatus(status string) bool {
	// TODO: implement
	return false
}

// WriteEscalationFile writes escalations to a markdown file.
func WriteEscalationFile(path string, escalations []Escalation, fs EscalationFS) error {
	// TODO: implement
	return nil
}

// ParseEscalationFile reads escalations from a markdown file.
func ParseEscalationFile(path string, fs EscalationFS) ([]Escalation, error) {
	// TODO: implement
	return nil, nil
}
