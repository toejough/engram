// Package parser provides document parsing functionality for traceability files.
package parser

import (
	"fmt"
	"strings"

	"github.com/toejough/projctl/internal/trace"
)

// ParseFrontmatter parses YAML frontmatter into a TraceItem.
// Returns error if YAML is invalid or required fields are missing.
func ParseFrontmatter(frontmatter string) (*trace.TraceItem, error) {
	return nil, fmt.Errorf("not implemented")
}

// FrontmatterItem represents a parsed item with frontmatter and body.
type FrontmatterItem struct {
	Frontmatter string // YAML frontmatter content (without delimiters)
	Body        string // Markdown body content
}

// SplitFrontmatter splits multi-item markdown content into frontmatter/body pairs.
// Each item is delimited by `---` for the start and end of frontmatter.
// Returns error if frontmatter is opened but not closed.
func SplitFrontmatter(content string) ([]FrontmatterItem, error) {
	if content == "" {
		return nil, nil
	}

	var items []FrontmatterItem
	lines := strings.Split(content, "\n")

	i := 0
	for i < len(lines) {
		// Skip leading empty lines
		for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
			i++
		}

		if i >= len(lines) {
			break
		}

		// Look for opening ---
		if strings.TrimSpace(lines[i]) != "---" {
			// No frontmatter delimiter, skip this content
			i++
			continue
		}

		// Found opening ---, now find closing ---
		i++
		frontmatterStart := i

		for i < len(lines) && strings.TrimSpace(lines[i]) != "---" {
			i++
		}

		if i >= len(lines) {
			return nil, fmt.Errorf("frontmatter missing closing delimiter")
		}

		frontmatter := strings.Join(lines[frontmatterStart:i], "\n")
		i++ // Skip closing ---

		// Collect body until next opening --- or end of content
		bodyStart := i
		for i < len(lines) {
			// Check if this is a new frontmatter opening
			// A new item starts with --- on its own line followed by non-empty content
			if strings.TrimSpace(lines[i]) == "---" {
				// Look ahead to see if this is a frontmatter opening
				// (followed by content) or just random dashes
				if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[i+1]), "---") {
					break
				}
			}
			i++
		}

		body := strings.TrimSpace(strings.Join(lines[bodyStart:i], "\n"))

		items = append(items, FrontmatterItem{
			Frontmatter: strings.TrimSpace(frontmatter),
			Body:        body,
		})
	}

	return items, nil
}
