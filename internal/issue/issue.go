// Package issue provides functionality for managing issues.md files.
package issue

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// IssuesFile is the default filename for issues.
const IssuesFile = "docs/issues.md"

// Issue represents a tracked issue.
type Issue struct {
	ID       string
	Title    string
	Priority string
	Status   string
	Created  string
	Body     string // Full markdown body after metadata
}

var issueHeaderRe = regexp.MustCompile(`^## (ISSUE-\d+): (.+)$`)
var priorityRe = regexp.MustCompile(`^\*\*Priority:\*\* (.+)$`)
var statusRe = regexp.MustCompile(`^\*\*Status:\*\* (.+)$`)
var createdRe = regexp.MustCompile(`^\*\*Created:\*\* (.+)$`)

// Parse reads issues from the issues file.
func Parse(dir string) ([]Issue, error) {
	path := filepath.Join(dir, IssuesFile)
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No issues file yet
		}
		return nil, fmt.Errorf("failed to read issues file: %w", err)
	}

	return ParseContent(string(content)), nil
}

// ParseContent parses issues from markdown content.
func ParseContent(content string) []Issue {
	var issues []Issue
	var current *Issue
	var bodyLines []string

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()

		// Check for new issue header
		if matches := issueHeaderRe.FindStringSubmatch(line); matches != nil {
			// Save previous issue
			if current != nil {
				current.Body = strings.TrimSpace(strings.Join(bodyLines, "\n"))
				issues = append(issues, *current)
			}
			current = &Issue{
				ID:    matches[1],
				Title: matches[2],
			}
			bodyLines = nil
			continue
		}

		if current == nil {
			continue
		}

		// Parse metadata fields
		if matches := priorityRe.FindStringSubmatch(line); matches != nil {
			current.Priority = matches[1]
			continue
		}
		if matches := statusRe.FindStringSubmatch(line); matches != nil {
			current.Status = matches[1]
			continue
		}
		if matches := createdRe.FindStringSubmatch(line); matches != nil {
			current.Created = matches[1]
			continue
		}

		// Accumulate body
		bodyLines = append(bodyLines, line)
	}

	// Save last issue
	if current != nil {
		current.Body = strings.TrimSpace(strings.Join(bodyLines, "\n"))
		issues = append(issues, *current)
	}

	return issues
}

// Get returns a single issue by ID.
func Get(dir string, id string) (*Issue, error) {
	issues, err := Parse(dir)
	if err != nil {
		return nil, err
	}

	for _, issue := range issues {
		if issue.ID == id {
			return &issue, nil
		}
	}

	return nil, fmt.Errorf("issue not found: %s", id)
}

// NextID returns the next available issue ID.
func NextID(dir string) (string, error) {
	issues, err := Parse(dir)
	if err != nil {
		return "", err
	}

	maxNum := 0
	for _, issue := range issues {
		// Extract number from ISSUE-NNN
		numStr := strings.TrimPrefix(issue.ID, "ISSUE-")
		if num, err := strconv.Atoi(numStr); err == nil && num > maxNum {
			maxNum = num
		}
	}

	return fmt.Sprintf("ISSUE-%03d", maxNum+1), nil
}

// CreateOpts holds options for creating an issue.
type CreateOpts struct {
	Title    string
	Priority string // Defaults to "Medium"
	Body     string // Optional detailed body
}

// Create adds a new issue to the issues file.
func Create(dir string, opts CreateOpts, now func() time.Time) (Issue, error) {
	id, err := NextID(dir)
	if err != nil {
		return Issue{}, err
	}

	priority := opts.Priority
	if priority == "" {
		priority = "Medium"
	}

	issue := Issue{
		ID:       id,
		Title:    opts.Title,
		Priority: priority,
		Status:   "Open",
		Created:  now().Format("2006-01-02"),
		Body:     opts.Body,
	}

	// Read existing content
	path := filepath.Join(dir, IssuesFile)
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return Issue{}, fmt.Errorf("failed to read issues file: %w", err)
	}

	// Build new content
	var builder strings.Builder
	if len(existing) > 0 {
		builder.Write(existing)
		if !strings.HasSuffix(string(existing), "\n") {
			builder.WriteString("\n")
		}
		builder.WriteString("\n---\n\n")
	} else {
		// Create header for new file
		builder.WriteString("# projctl Issues\n\n")
		builder.WriteString("Tracked issues for future work beyond the current task list.\n\n")
		builder.WriteString("---\n\n")
	}

	// Write new issue
	builder.WriteString(fmt.Sprintf("## %s: %s\n\n", issue.ID, issue.Title))
	builder.WriteString(fmt.Sprintf("**Priority:** %s\n", issue.Priority))
	builder.WriteString(fmt.Sprintf("**Status:** %s\n", issue.Status))
	builder.WriteString(fmt.Sprintf("**Created:** %s\n", issue.Created))
	if issue.Body != "" {
		builder.WriteString("\n")
		builder.WriteString(issue.Body)
		builder.WriteString("\n")
	}

	// Ensure docs directory exists
	docsDir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		return Issue{}, fmt.Errorf("failed to create docs directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, []byte(builder.String()), 0o644); err != nil {
		return Issue{}, fmt.Errorf("failed to write issues file: %w", err)
	}

	return issue, nil
}

// UpdateOpts holds options for updating an issue.
type UpdateOpts struct {
	Status  string // New status (Open, Closed, etc.)
	Comment string // Optional comment to append
}

// Update modifies an existing issue.
func Update(dir string, id string, opts UpdateOpts) error {
	path := filepath.Join(dir, IssuesFile)
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read issues file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var result []string
	inIssue := false
	foundIssue := false
	issuePattern := fmt.Sprintf("## %s:", id)

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check if entering target issue
		if strings.HasPrefix(line, issuePattern) {
			inIssue = true
			foundIssue = true
			result = append(result, line)
			continue
		}

		// Check if leaving target issue (next issue or end)
		if inIssue && strings.HasPrefix(line, "## ISSUE-") {
			// Add comment before leaving if specified
			if opts.Comment != "" {
				result = append(result, "")
				result = append(result, "### Comment")
				result = append(result, "")
				result = append(result, opts.Comment)
			}
			inIssue = false
		}

		// Update status line if in target issue
		if inIssue && strings.HasPrefix(line, "**Status:**") && opts.Status != "" {
			result = append(result, fmt.Sprintf("**Status:** %s", opts.Status))
			continue
		}

		result = append(result, line)
	}

	// Add comment at end if still in issue (last issue in file)
	if inIssue && opts.Comment != "" {
		result = append(result, "")
		result = append(result, "### Comment")
		result = append(result, "")
		result = append(result, opts.Comment)
	}

	if !foundIssue {
		return fmt.Errorf("issue not found: %s", id)
	}

	if err := os.WriteFile(path, []byte(strings.Join(result, "\n")), 0o644); err != nil {
		return fmt.Errorf("failed to write issues file: %w", err)
	}

	return nil
}

// List returns all issues, optionally filtered by status.
func List(dir string, statusFilter string) ([]Issue, error) {
	issues, err := Parse(dir)
	if err != nil {
		return nil, err
	}

	if statusFilter == "" {
		return issues, nil
	}

	var filtered []Issue
	for _, issue := range issues {
		if strings.EqualFold(issue.Status, statusFilter) {
			filtered = append(filtered, issue)
		}
	}

	return filtered, nil
}
