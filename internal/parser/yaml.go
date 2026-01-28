// Package parser provides document parsing functionality for traceability files.
package parser

// FrontmatterItem represents a parsed item with frontmatter and body.
type FrontmatterItem struct {
	Frontmatter string // YAML frontmatter content (without delimiters)
	Body        string // Markdown body content
}

// SplitFrontmatter splits multi-item markdown content into frontmatter/body pairs.
// Each item is delimited by `---` for the start and end of frontmatter.
// Returns error if frontmatter is opened but not closed.
func SplitFrontmatter(content string) ([]FrontmatterItem, error) {
	// TODO: Implement
	return nil, nil
}
