package promote

import (
	"errors"
	"fmt"
	"strings"
)

// SectionEditor implements ClaudeMDEditor using ## heading-based section parsing.
type SectionEditor struct{}

// AddEntry appends a new section to CLAUDE.md content.
func (e *SectionEditor) AddEntry(content, entry string) (string, error) {
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return entry, nil
	}

	return content + "\n\n" + entry, nil
}

// ExtractEntry returns the content of the section containing the
// promoted-from marker for the given entryID.
func (e *SectionEditor) ExtractEntry(
	content, entryID string,
) (string, error) {
	marker := "<!-- promoted from " + entryID + " -->"

	sections := splitSections(content)

	for _, section := range sections {
		if strings.Contains(section, marker) {
			return strings.TrimSpace(section), nil
		}
	}

	return "", fmt.Errorf("%w: %q", errEntryNotFound, entryID)
}

// RemoveEntry removes the section containing the promoted-from marker
// for the given entryID.
func (e *SectionEditor) RemoveEntry(
	content, entryID string,
) (string, error) {
	marker := "<!-- promoted from " + entryID + " -->"

	sections := splitSections(content)

	result := make([]string, 0, len(sections))
	found := false

	for _, section := range sections {
		if strings.Contains(section, marker) {
			found = true

			continue
		}

		result = append(result, section)
	}

	if !found {
		return "", fmt.Errorf("%w: %q", errEntryNotFound, entryID)
	}

	return joinSections(result), nil
}

// unexported variables.
var (
	errEntryNotFound = errors.New("entry not found in CLAUDE.md")
)

// joinSections reassembles sections, trimming trailing whitespace
// between them.
func joinSections(sections []string) string {
	trimmed := make([]string, 0, len(sections))

	for _, s := range sections {
		trimmed = append(trimmed, strings.TrimRight(s, "\n"))
	}

	return strings.Join(trimmed, "\n\n")
}

// splitSections splits CLAUDE.md content into sections delimited by
// ## headings. The preamble (content before the first ## heading) is
// the first element if present.
func splitSections(content string) []string {
	lines := strings.Split(content, "\n")
	sections := make([]string, 0)

	var buf strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") && buf.Len() > 0 {
			sections = append(sections, buf.String())
			buf.Reset()
		}

		buf.WriteString(line)
		buf.WriteString("\n")
	}

	if buf.Len() > 0 {
		sections = append(sections, buf.String())
	}

	return sections
}
