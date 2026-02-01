// Package conflict manages conflict records stored in conflicts.md.
package conflict

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ConflictFile is the filename for conflict records.
const ConflictFile = "conflicts.md"

// Status values for conflicts.
const (
	StatusOpen        = "open"
	StatusNegotiating = "negotiating"
	StatusResolved    = "resolved"
)

// Conflict represents a cross-skill conflict record.
type Conflict struct {
	ID            string
	Skills        string
	Traceability  string
	Description   string
	Status        string
	Resolution    string
}

// Create appends a new conflict entry to conflicts.md with an auto-incremented CONF- ID.
func Create(dir, skills, traceability, description string) (string, error) {
	path := filepath.Join(dir, ConflictFile)

	existing, err := loadAll(path)
	if err != nil {
		return "", err
	}

	nextID := fmt.Sprintf("CONF-%03d", len(existing)+1)

	entry := fmt.Sprintf(`
### %s

**Skills:** %s
**Traceability:** %s
**Status:** %s
**Description:** %s
**Resolution:** (pending)
`, nextID, skills, traceability, StatusNegotiating, description)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return "", fmt.Errorf("failed to open conflicts file: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Write header if file is empty
	info, _ := f.Stat()
	if info.Size() == 0 {
		if _, err := f.WriteString("# Conflicts\n"); err != nil {
			return "", fmt.Errorf("failed to write header: %w", err)
		}
	}

	if _, err := f.WriteString(entry); err != nil {
		return "", fmt.Errorf("failed to write conflict: %w", err)
	}

	return nextID, nil
}

// CheckResult holds newly resolved conflicts.
type CheckResult struct {
	Resolved []Conflict `json:"resolved"`
}

// Check parses conflicts.md for entries with a non-empty Resolution field.
func Check(dir string) (CheckResult, error) {
	path := filepath.Join(dir, ConflictFile)

	all, err := loadAll(path)
	if err != nil {
		return CheckResult{}, err
	}

	var resolved []Conflict
	for _, c := range all {
		if c.Status == StatusResolved || (c.Resolution != "" && c.Resolution != "(pending)") {
			resolved = append(resolved, c)
		}
	}

	return CheckResult{Resolved: resolved}, nil
}

// ListResult holds all conflicts with summary.
type ListResult struct {
	Conflicts []Conflict `json:"conflicts"`
	Open      int        `json:"open"`
	Resolved  int        `json:"resolved"`
	Negotiating int      `json:"negotiating"`
}

// List returns all conflicts with optional status filter.
func List(dir, statusFilter string) (ListResult, error) {
	path := filepath.Join(dir, ConflictFile)

	all, err := loadAll(path)
	if err != nil {
		return ListResult{}, err
	}

	result := ListResult{}

	for _, c := range all {
		switch c.Status {
		case StatusOpen:
			result.Open++
		case StatusResolved:
			result.Resolved++
		case StatusNegotiating:
			result.Negotiating++
		}

		if statusFilter == "" || c.Status == statusFilter {
			result.Conflicts = append(result.Conflicts, c)
		}
	}

	return result, nil
}

var (
	confIDPattern     = regexp.MustCompile(`### (CONF-\d{3})`)
	fieldPattern      = regexp.MustCompile(`\*\*(\w+):\*\*\s*(.*)`)
)

func loadAll(path string) ([]Conflict, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to read conflicts file: %w", err)
	}

	return parseConflicts(string(data)), nil
}

func parseConflicts(content string) []Conflict {
	var conflicts []Conflict
	sections := confIDPattern.FindAllStringIndex(content, -1)

	for i, loc := range sections {
		var section string
		if i+1 < len(sections) {
			section = content[loc[0]:sections[i+1][0]]
		} else {
			section = content[loc[0]:]
		}

		idMatch := confIDPattern.FindStringSubmatch(section)
		if idMatch == nil {
			continue
		}

		c := Conflict{ID: idMatch[1]}

		fields := fieldPattern.FindAllStringSubmatch(section, -1)
		for _, f := range fields {
			key := strings.ToLower(f[1])
			val := strings.TrimSpace(f[2])

			switch key {
			case "skills":
				c.Skills = val
			case "traceability":
				c.Traceability = val
			case "status":
				c.Status = val
			case "description":
				c.Description = val
			case "resolution":
				c.Resolution = val
			}
		}

		conflicts = append(conflicts, c)
	}

	return conflicts
}
