package memory

import (
	"testing"

	. "github.com/onsi/gomega"
)

// TestFormatActionExplanation_AllCases verifies all switch cases produce expected output.
func TestFormatActionExplanation_AllCases(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tests := []struct {
		action   string
		tier     string
		expected string
	}{
		{"prune", "skills", "archive"},
		{"prune", "claude-md", "Remove this line"},
		{"prune", "embeddings", "Delete this memory"},
		{"decay", "", "confidence"},
		{"consolidate", "claude-md", "Merge duplicate CLAUDE.md"},
		{"consolidate", "embeddings", "Delete the second"},
		{"split", "", "Break this"},
		{"promote", "embeddings", "Create a new skill"},
		{"promote", "skills", "Add this skill"},
		{"promote", "other", "higher tier"},
		{"demote", "claude-md", "Remove from CLAUDE.md"},
		{"demote", "embeddings", "lower tier"},
		{"rewrite", "", "LLM-improved"},
		{"add-rationale", "", "WHY"},
		{"extract-examples", "", "concrete examples"},
		{"unknown", "", ""},
	}

	for _, tc := range tests {
		result := formatActionExplanation(tc.action, tc.tier)

		if tc.expected == "" {
			g.Expect(result).To(BeEmpty(), "action=%q tier=%q", tc.action, tc.tier)
		} else {
			g.Expect(result).To(ContainSubstring(tc.expected), "action=%q tier=%q", tc.action, tc.tier)
		}
	}
}
