// Package integrate handles merging per-project documentation into top-level docs.
package integrate

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// MergeFS provides file system operations for merge.
type MergeFS interface {
	ReadFile(path string) (string, error)
	WriteFile(path string, content string) error
	FileExists(path string) bool
	RemoveAll(path string) error
}

// MergeResult holds the results of a merge operation.
type MergeResult struct {
	RequirementsAdded  int    // Number of requirements added
	DesignAdded        int    // Number of design decisions added
	ArchitectureAdded  int    // Number of architecture decisions added
	TasksAdded         int    // Number of tasks added
	IDsRenumbered      int    // Number of IDs that were renumbered due to conflicts
	LinksUpdated       int    // Number of traceability links updated
	Summary            string // Human-readable summary
}

// docItem represents a parsed documentation item.
type docItem struct {
	ID      string // e.g., "REQ-001"
	Prefix  string // e.g., "REQ"
	Number  int    // e.g., 1
	Title   string // Title after the ID
	Content string // Full markdown content including header
}

// docFile represents a documentation file type for merging.
type docFile struct {
	Name   string // e.g., "requirements"
	Prefix string // e.g., "REQ"
}

var docFiles = []docFile{
	{"requirements", "REQ"},
	{"design", "DES"},
	{"architecture", "ARCH"},
	{"tasks", "TASK"},
}

// Merge merges per-project documentation into top-level docs.
func Merge(projectDir string, projectName string, fs MergeFS) (*MergeResult, error) {
	result := &MergeResult{}
	idMapping := make(map[string]string) // old ID -> new ID

	docsDir := filepath.Join(projectDir, "docs")
	perProjectDir := filepath.Join(docsDir, "projects", projectName)

	for _, df := range docFiles {
		topLevelPath := filepath.Join(docsDir, df.Name+".md")
		perProjectPath := filepath.Join(perProjectDir, df.Name+".md")

		// Skip if per-project file doesn't exist
		if !fs.FileExists(perProjectPath) {
			continue
		}

		added, renumbered, err := mergeFile(topLevelPath, perProjectPath, df.Prefix, fs, idMapping)
		if err != nil {
			return nil, fmt.Errorf("merging %s: %w", df.Name, err)
		}

		result.IDsRenumbered += renumbered

		switch df.Name {
		case "requirements":
			result.RequirementsAdded = added
		case "design":
			result.DesignAdded = added
		case "architecture":
			result.ArchitectureAdded = added
		case "tasks":
			result.TasksAdded = added
		}
	}

	// Update traceability if there were renumbered IDs
	if len(idMapping) > 0 {
		linksUpdated, err := mergeTraceability(docsDir, perProjectDir, fs, idMapping)
		if err != nil {
			return nil, fmt.Errorf("merging traceability: %w", err)
		}
		result.LinksUpdated = linksUpdated
	}

	result.Summary = buildSummary(result)
	return result, nil
}

// mergeFile merges a single documentation file.
func mergeFile(topLevelPath, perProjectPath, prefix string, fs MergeFS, idMapping map[string]string) (added, renumbered int, err error) {
	topContent, err := fs.ReadFile(topLevelPath)
	if err != nil {
		return 0, 0, fmt.Errorf("reading top-level file: %w", err)
	}

	perProjectContent, err := fs.ReadFile(perProjectPath)
	if err != nil {
		return 0, 0, fmt.Errorf("reading per-project file: %w", err)
	}

	topItems := parseItems(topContent, prefix)
	perProjectItems := parseItems(perProjectContent, prefix)

	if len(perProjectItems) == 0 {
		return 0, 0, nil
	}

	// Find max ID in top-level
	maxID := findMaxID(topItems)

	// Build set of existing IDs
	existingIDs := make(map[string]bool)
	for _, item := range topItems {
		existingIDs[item.ID] = true
	}

	// Process per-project items
	var itemsToAdd []string
	for _, item := range perProjectItems {
		newContent := item.Content
		newID := item.ID

		// Check for conflict
		if existingIDs[item.ID] {
			// Renumber
			maxID++
			newID = fmt.Sprintf("%s-%03d", prefix, maxID)
			newContent = strings.Replace(item.Content, item.ID, newID, 1)
			idMapping[item.ID] = newID
			renumbered++
		}

		itemsToAdd = append(itemsToAdd, newContent)
		added++
	}

	// Append to top-level content
	merged := topContent
	if !strings.HasSuffix(merged, "\n") {
		merged += "\n"
	}
	merged += strings.Join(itemsToAdd, "\n")

	if err := fs.WriteFile(topLevelPath, merged); err != nil {
		return 0, 0, fmt.Errorf("writing merged file: %w", err)
	}

	return added, renumbered, nil
}

// parseItems extracts documentation items from markdown content.
func parseItems(content, prefix string) []docItem {
	var items []docItem

	// Pattern to match headers like "## REQ-001: Title"
	pattern := regexp.MustCompile(`(?m)^##\s+(` + prefix + `-(\d+)):\s*(.*)$`)
	matches := pattern.FindAllStringSubmatchIndex(content, -1)

	for i, match := range matches {
		if len(match) < 8 {
			continue
		}

		id := content[match[2]:match[3]]
		numStr := content[match[4]:match[5]]
		title := content[match[6]:match[7]]
		num, _ := strconv.Atoi(numStr)

		// Extract content from this header to the next header or end
		start := match[0]
		end := len(content)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}

		itemContent := strings.TrimRight(content[start:end], "\n") + "\n"

		items = append(items, docItem{
			ID:      id,
			Prefix:  prefix,
			Number:  num,
			Title:   title,
			Content: itemContent,
		})
	}

	return items
}

// findMaxID returns the highest ID number from items.
func findMaxID(items []docItem) int {
	max := 0
	for _, item := range items {
		if item.Number > max {
			max = item.Number
		}
	}
	return max
}

// mergeTraceability merges traceability files and updates IDs.
func mergeTraceability(docsDir, perProjectDir string, fs MergeFS, idMapping map[string]string) (int, error) {
	topPath := filepath.Join(docsDir, "traceability.toml")
	perProjectPath := filepath.Join(perProjectDir, "traceability.toml")

	if !fs.FileExists(perProjectPath) {
		return 0, nil
	}

	topContent := ""
	if fs.FileExists(topPath) {
		var err error
		topContent, err = fs.ReadFile(topPath)
		if err != nil {
			return 0, fmt.Errorf("reading top-level traceability: %w", err)
		}
	}

	perProjectContent, err := fs.ReadFile(perProjectPath)
	if err != nil {
		return 0, fmt.Errorf("reading per-project traceability: %w", err)
	}

	// Update IDs in per-project content
	updatedContent := perProjectContent
	linksUpdated := 0
	for oldID, newID := range idMapping {
		if strings.Contains(updatedContent, oldID) {
			updatedContent = strings.ReplaceAll(updatedContent, oldID, newID)
			linksUpdated++
		}
	}

	// Merge contents
	merged := topContent
	if !strings.HasSuffix(merged, "\n") && merged != "" {
		merged += "\n"
	}
	merged += updatedContent

	if err := fs.WriteFile(topPath, merged); err != nil {
		return 0, fmt.Errorf("writing merged traceability: %w", err)
	}

	return linksUpdated, nil
}

// buildSummary creates a human-readable summary.
func buildSummary(r *MergeResult) string {
	var parts []string

	if r.RequirementsAdded > 0 {
		parts = append(parts, fmt.Sprintf("%d requirement(s)", r.RequirementsAdded))
	}
	if r.DesignAdded > 0 {
		parts = append(parts, fmt.Sprintf("%d design decision(s)", r.DesignAdded))
	}
	if r.ArchitectureAdded > 0 {
		parts = append(parts, fmt.Sprintf("%d architecture decision(s)", r.ArchitectureAdded))
	}
	if r.TasksAdded > 0 {
		parts = append(parts, fmt.Sprintf("%d task(s)", r.TasksAdded))
	}

	if len(parts) == 0 {
		return "No items merged"
	}

	summary := "Merged " + strings.Join(parts, ", ")
	if r.IDsRenumbered > 0 {
		summary += fmt.Sprintf(" (%d ID(s) renumbered)", r.IDsRenumbered)
	}
	return summary
}
