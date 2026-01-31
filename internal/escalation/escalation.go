// Package escalation handles user escalation workflow for project adoption.
package escalation

import (
	"fmt"
	"regexp"
	"strings"
)

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

// validStatuses lists all valid escalation status values.
var validStatuses = map[string]bool{
	"pending":  true,
	"resolved": true,
	"deferred": true,
	"issue":    true,
}

// IsValidStatus checks if a status string is valid.
func IsValidStatus(status string) bool {
	return validStatuses[status]
}

// WriteEscalationFile writes escalations to a markdown file.
func WriteEscalationFile(path string, escalations []Escalation, fs EscalationFS) error {
	var b strings.Builder

	b.WriteString("# Escalations\n\n")
	b.WriteString("Review each escalation and update the **Status** field:\n")
	b.WriteString("- `pending` - Not yet reviewed\n")
	b.WriteString("- `resolved` - Add your answer in **Notes**\n")
	b.WriteString("- `deferred` - Create an issue for later\n")
	b.WriteString("- `issue` - Create an issue with your description in **Notes**\n\n")
	b.WriteString("---\n\n")

	for _, e := range escalations {
		b.WriteString(fmt.Sprintf("## %s\n\n", e.ID))
		b.WriteString(fmt.Sprintf("**Category:** %s\n", e.Category))
		b.WriteString(fmt.Sprintf("**Context:** %s\n", e.Context))
		b.WriteString(fmt.Sprintf("**Question:** %s\n\n", e.Question))
		b.WriteString(fmt.Sprintf("**Status:** %s\n", e.Status))
		b.WriteString(fmt.Sprintf("**Notes:** %s\n\n", e.Notes))
		b.WriteString("---\n\n")
	}

	return fs.WriteFile(path, b.String())
}

// escHeaderPattern matches ## ESC-NNN headers
var escHeaderPattern = regexp.MustCompile(`^## (ESC-\d+)`)

// fieldPattern matches **Field:** Value lines
var fieldPattern = regexp.MustCompile(`^\*\*(\w+):\*\*\s*(.*)$`)

// Resolve updates an escalation's status and notes by ID.
// Returns the updated slice or error if ID not found or status invalid.
func Resolve(escalations []Escalation, id, status, notes string) ([]Escalation, error) {
	if !IsValidStatus(status) {
		return nil, fmt.Errorf("invalid status %q", status)
	}

	found := false
	result := make([]Escalation, len(escalations))
	for i, e := range escalations {
		if e.ID == id {
			e.Status = status
			e.Notes = notes
			found = true
		}
		result[i] = e
	}

	if !found {
		return nil, fmt.Errorf("escalation %q not found", id)
	}

	return result, nil
}

// ParseEscalationFile reads escalations from a markdown file.
func ParseEscalationFile(path string, fs EscalationFS) ([]Escalation, error) {
	content, err := fs.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var escalations []Escalation
	var current *Escalation

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")

		// Check for new escalation header
		if match := escHeaderPattern.FindStringSubmatch(line); match != nil {
			if current != nil {
				escalations = append(escalations, *current)
			}
			current = &Escalation{ID: match[1]}
			continue
		}

		// Skip if not in an escalation block
		if current == nil {
			continue
		}

		// Check for field lines
		if match := fieldPattern.FindStringSubmatch(line); match != nil {
			field := match[1]
			value := strings.TrimSpace(match[2])

			switch field {
			case "Category":
				current.Category = value
			case "Context":
				current.Context = value
			case "Question":
				current.Question = value
			case "Status":
				current.Status = value
			case "Notes":
				current.Notes = value
			}
		}
	}

	// Don't forget the last escalation
	if current != nil {
		escalations = append(escalations, *current)
	}

	// Validate statuses
	for _, e := range escalations {
		if !IsValidStatus(e.Status) {
			return nil, fmt.Errorf("invalid status %q for escalation %s", e.Status, e.ID)
		}
	}

	return escalations, nil
}
