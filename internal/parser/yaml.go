// Package parser provides document parsing functionality for traceability files.
package parser

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/toejough/projctl/internal/trace"
)

// frontmatterData represents the raw YAML structure before conversion to TraceItem.
type frontmatterData struct {
	ID       string    `yaml:"id"`
	Type     string    `yaml:"type"`
	Project  string    `yaml:"project"`
	Title    string    `yaml:"title"`
	Status   string    `yaml:"status"`
	TracesTo []string  `yaml:"traces_to"`
	Tags     []string  `yaml:"tags"`
	Created  time.Time `yaml:"created"`
	Updated  time.Time `yaml:"updated"`

	// TEST-specific fields
	Location string `yaml:"location"`
	Line     int    `yaml:"line"`
	Function string `yaml:"function"`
}

// ParseResult represents a parsed traceability item with its body content.
type ParseResult struct {
	Item *trace.TraceItem
	Body string
}

// ParseDocument parses a document containing multiple frontmatter items.
// Returns all successfully parsed items and any errors encountered.
// Malformed items are skipped with errors collected.
func ParseDocument(content string) ([]ParseResult, []error) {
	items, err := SplitFrontmatter(content)
	if err != nil {
		return nil, []error{err}
	}

	var results []ParseResult
	var errs []error

	for _, item := range items {
		parsed, err := ParseFrontmatter(item.Frontmatter)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		results = append(results, ParseResult{
			Item: parsed,
			Body: item.Body,
		})
	}

	return results, errs
}

// ParseFrontmatter parses YAML frontmatter into a TraceItem.
// Returns error if YAML is invalid or required fields are missing.
func ParseFrontmatter(frontmatter string) (*trace.TraceItem, error) {
	var data frontmatterData
	if err := yaml.Unmarshal([]byte(frontmatter), &data); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	item := &trace.TraceItem{
		ID:       data.ID,
		Type:     trace.NodeType(data.Type),
		Project:  data.Project,
		Title:    data.Title,
		Status:   data.Status,
		TracesTo: data.TracesTo,
		Tags:     data.Tags,
		Created:  data.Created,
		Updated:  data.Updated,
		Location: data.Location,
		Line:     data.Line,
		Function: data.Function,
	}

	if err := item.Validate(); err != nil {
		return nil, err
	}

	return item, nil
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
