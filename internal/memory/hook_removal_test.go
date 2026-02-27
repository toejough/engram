//go:build sqlite_fts5

package memory_test

import (
	"reflect"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestFormatMarkdownNoSkillsSection verifies that FormatMarkdown output
// never contains "## Relevant Skills" section regardless of input.
func TestFormatMarkdownNoSkillsSection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Test with memories only (no skills field exists to populate)
	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results: []memory.QueryResult{
			{
				Content:    "Test memory",
				Confidence: 0.9,
				Score:      0.8,
				Source:     "memory",
				SourceType: "internal",
			},
		},
		MinConfidence: 0.0,
		MaxEntries:    10,
		MaxTokens:     2000,
	})

	g.Expect(output).ToNot(ContainSubstring("## Relevant Skills"),
		"FormatMarkdown output should never contain skills section after hook removal")
}

// TestFormatMarkdownOptsHasNoSkillsField verifies that FormatMarkdownOpts
// does NOT have a Skills field (hook injection removed).
func TestFormatMarkdownOptsHasNoSkillsField(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	typ := reflect.TypeFor[memory.FormatMarkdownOpts]()
	_, found := typ.FieldByName("Skills")
	g.Expect(found).To(BeFalse(), "FormatMarkdownOpts should not have Skills field after hook removal")
}

// TestQueryResultsHasNoSkillsField verifies that QueryResults struct
// does NOT have a Skills field (hook injection removed).
func TestQueryResultsHasNoSkillsField(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	typ := reflect.TypeFor[memory.QueryResults]()
	_, found := typ.FieldByName("Skills")
	g.Expect(found).To(BeFalse(), "QueryResults should not have Skills field after hook removal")
}
