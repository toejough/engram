package corrections_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/corrections"
	"pgregory.net/rapid"
)

// TEST-715 traces: TASK-042
// Test Analyze identifies repeated corrections (count >= 2)
func TestAnalyze_IdentifiesRepeatedCorrections(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{}

	// Log same correction twice
	_ = corrections.Log("testdir", "Never amend pushed commits", "git workflow", corrections.LogOpts{}, nowFunc(), fs)
	_ = corrections.Log("testdir", "Never amend pushed commits", "git workflow", corrections.LogOpts{}, nowFunc(), fs)
	_ = corrections.Log("testdir", "Different correction", "other context", corrections.LogOpts{}, nowFunc(), fs)

	patterns, err := corrections.Analyze("testdir", corrections.AnalyzeOpts{
		MinOccurrences: 2,
	}, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(patterns).To(HaveLen(1))
	g.Expect(patterns[0].Message).To(Equal("Never amend pushed commits"))
	g.Expect(patterns[0].Count).To(Equal(2))
}

// TEST-716 traces: TASK-042
// Test Analyze groups similar corrections with fuzzy matching
func TestAnalyze_FuzzyMatchingSimilarCorrections(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{}

	// These should be grouped together despite different wording
	_ = corrections.Log("testdir", "Never amend pushed commits", "context1", corrections.LogOpts{}, nowFunc(), fs)
	_ = corrections.Log("testdir", "Don't amend commits that are pushed", "context2", corrections.LogOpts{}, nowFunc(), fs)
	_ = corrections.Log("testdir", "Avoid amending pushed commits", "context3", corrections.LogOpts{}, nowFunc(), fs)

	patterns, err := corrections.Analyze("testdir", corrections.AnalyzeOpts{
		MinOccurrences: 2,
	}, fs)
	g.Expect(err).ToNot(HaveOccurred())

	// Should find one pattern with 3 occurrences
	g.Expect(patterns).To(HaveLen(1))
	g.Expect(patterns[0].Count).To(Equal(3))

	// Pattern message should represent the group
	g.Expect(patterns[0].Message).To(ContainSubstring("amend"))
	g.Expect(patterns[0].Message).To(ContainSubstring("push"))
}

// TEST-717 traces: TASK-042
// Test Analyze respects MinOccurrences threshold
func TestAnalyze_RespectsMinOccurrences(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{}

	// Log correction 3 times
	_ = corrections.Log("testdir", "Check VCS type before git commands", "vcs", corrections.LogOpts{}, nowFunc(), fs)
	_ = corrections.Log("testdir", "Check VCS type before git commands", "vcs", corrections.LogOpts{}, nowFunc(), fs)
	_ = corrections.Log("testdir", "Check VCS type before git commands", "vcs", corrections.LogOpts{}, nowFunc(), fs)

	// With threshold of 2, should find pattern
	patterns, err := corrections.Analyze("testdir", corrections.AnalyzeOpts{
		MinOccurrences: 2,
	}, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(patterns).To(HaveLen(1))

	// With threshold of 4, should find nothing
	patterns, err = corrections.Analyze("testdir", corrections.AnalyzeOpts{
		MinOccurrences: 4,
	}, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(patterns).To(BeEmpty())
}

// TEST-718 traces: TASK-042
// Test Analyze default MinOccurrences is 2
func TestAnalyze_DefaultMinOccurrencesIsTwo(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{}

	// Single occurrence
	_ = corrections.Log("testdir", "Some correction", "context", corrections.LogOpts{}, nowFunc(), fs)

	// With default opts, should not find pattern
	patterns, err := corrections.Analyze("testdir", corrections.AnalyzeOpts{}, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(patterns).To(BeEmpty())

	// Add second occurrence
	_ = corrections.Log("testdir", "Some correction", "context", corrections.LogOpts{}, nowFunc(), fs)

	// Now should find pattern
	patterns, err = corrections.Analyze("testdir", corrections.AnalyzeOpts{}, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(patterns).To(HaveLen(1))
}

// TEST-719 traces: TASK-042
// Test Pattern contains proposal for CLAUDE.md
func TestAnalyze_PatternIncludesProposal(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{}

	_ = corrections.Log("testdir", "Never use git checkout -- .", "destroys work", corrections.LogOpts{}, nowFunc(), fs)
	_ = corrections.Log("testdir", "Never use git checkout -- .", "destroys work", corrections.LogOpts{}, nowFunc(), fs)

	patterns, err := corrections.Analyze("testdir", corrections.AnalyzeOpts{
		MinOccurrences: 2,
	}, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(patterns).To(HaveLen(1))

	// Should have a proposed CLAUDE.md addition
	g.Expect(patterns[0].Proposal).ToNot(BeEmpty())
	g.Expect(patterns[0].Proposal).To(ContainSubstring("git checkout"))
}

// TEST-720 traces: TASK-042
// Test Pattern output includes all required fields
func TestAnalyze_PatternFieldsComplete(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{}

	_ = corrections.Log("testdir", "Use build tool commands", "context1", corrections.LogOpts{}, nowFunc(), fs)
	_ = corrections.Log("testdir", "Use build tool commands", "context2", corrections.LogOpts{}, nowFunc(), fs)

	patterns, err := corrections.Analyze("testdir", corrections.AnalyzeOpts{
		MinOccurrences: 2,
	}, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(patterns).To(HaveLen(1))

	pattern := patterns[0]
	// Required fields from acceptance criteria
	g.Expect(pattern.Message).ToNot(BeEmpty())    // Pattern
	g.Expect(pattern.Count).To(Equal(2))          // Count
	g.Expect(pattern.Proposal).ToNot(BeEmpty())   // Proposed rule
}

// TEST-721 traces: TASK-042
// Test Analyze with no corrections returns empty patterns
func TestAnalyze_NoCorrections(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{}

	patterns, err := corrections.Analyze("testdir", corrections.AnalyzeOpts{}, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(patterns).To(BeEmpty())
}

// TEST-722 traces: TASK-042
// Test Analyze groups by keywords not exact match
func TestAnalyze_GroupsByKeywords(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{}

	// Different messages but same keywords
	_ = corrections.Log("testdir", "Check for plan documents when resuming", "planning", corrections.LogOpts{}, nowFunc(), fs)
	_ = corrections.Log("testdir", "After compaction, check plan documents exist", "planning", corrections.LogOpts{}, nowFunc(), fs)
	_ = corrections.Log("testdir", "Always check for plan docs on resume", "planning", corrections.LogOpts{}, nowFunc(), fs)

	patterns, err := corrections.Analyze("testdir", corrections.AnalyzeOpts{
		MinOccurrences: 2,
	}, fs)
	g.Expect(err).ToNot(HaveOccurred())

	// Should group by "plan" and "document" keywords
	g.Expect(patterns).To(HaveLen(1))
	g.Expect(patterns[0].Count).To(Equal(3))
}

// TEST-723 traces: TASK-042
// Test Analyze property: patterns always sorted by count descending
func TestAnalyze_Property_PatternsSortedByCount(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		fs := &MockFS{}

		// Generate random corrections with varying repetition
		numPatterns := rapid.IntRange(2, 5).Draw(rt, "num_patterns")
		for i := 0; i < numPatterns; i++ {
			message := rapid.StringMatching(`[a-z ]+`).Draw(rt, "message")
			repetitions := rapid.IntRange(2, 10).Draw(rt, "repetitions")
			for j := 0; j < repetitions; j++ {
				_ = corrections.Log("testdir", message, "context", corrections.LogOpts{}, nowFunc(), fs)
			}
		}

		patterns, err := corrections.Analyze("testdir", corrections.AnalyzeOpts{
			MinOccurrences: 2,
		}, fs)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify sorted by count descending
		for i := 1; i < len(patterns); i++ {
			g.Expect(patterns[i-1].Count).To(BeNumerically(">=", patterns[i].Count))
		}
	})
}

// TEST-724 traces: TASK-042
// Test Analyze fuzzy matching uses keyword extraction
func TestAnalyze_KeywordExtraction(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{}

	// Should extract "amend", "push", "commit" as keywords
	_ = corrections.Log("testdir", "Never amend pushed commits - check git status first", "git", corrections.LogOpts{}, nowFunc(), fs)
	_ = corrections.Log("testdir", "Don't amend if commits are pushed upstream", "git", corrections.LogOpts{}, nowFunc(), fs)

	patterns, err := corrections.Analyze("testdir", corrections.AnalyzeOpts{
		MinOccurrences: 2,
	}, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(patterns).To(HaveLen(1))

	// Pattern message should contain key terms
	msg := patterns[0].Message
	g.Expect(msg).To(Or(
		ContainSubstring("amend"),
		ContainSubstring("push"),
		ContainSubstring("commit"),
	))
}

// TEST-725 traces: TASK-042
// Test Analyze ignores stop words in fuzzy matching
func TestAnalyze_IgnoresStopWords(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{}

	// Same meaning despite different stop words
	_ = corrections.Log("testdir", "Never use the git checkout command with -- .", "git", corrections.LogOpts{}, nowFunc(), fs)
	_ = corrections.Log("testdir", "Don't use git checkout with -- .", "git", corrections.LogOpts{}, nowFunc(), fs)
	_ = corrections.Log("testdir", "Avoid using git checkout -- .", "git", corrections.LogOpts{}, nowFunc(), fs)

	patterns, err := corrections.Analyze("testdir", corrections.AnalyzeOpts{
		MinOccurrences: 2,
	}, fs)
	g.Expect(err).ToNot(HaveOccurred())

	// Should group despite different stop words (the, with, using)
	g.Expect(patterns).To(HaveLen(1))
	g.Expect(patterns[0].Count).To(Equal(3))
}

// TEST-726 traces: TASK-042
// Test Pattern proposal format is markdown-compatible
func TestAnalyze_ProposalIsMarkdown(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{}

	_ = corrections.Log("testdir", "Use dependency injection for time.Now", "testing", corrections.LogOpts{}, nowFunc(), fs)
	_ = corrections.Log("testdir", "Use dependency injection for time.Now", "testing", corrections.LogOpts{}, nowFunc(), fs)

	patterns, err := corrections.Analyze("testdir", corrections.AnalyzeOpts{
		MinOccurrences: 2,
	}, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(patterns).To(HaveLen(1))

	proposal := patterns[0].Proposal
	// Should be formatted for CLAUDE.md (markdown list or header)
	g.Expect(proposal).To(Or(
		HavePrefix("-"),
		HavePrefix("*"),
		HavePrefix("**"),
	))
}

// TEST-727 traces: TASK-042
// Test Analyze with MinOccurrences of 1 returns all corrections
func TestAnalyze_MinOccurrencesOne(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{}

	_ = corrections.Log("testdir", "Unique correction one", "context1", corrections.LogOpts{}, nowFunc(), fs)
	_ = corrections.Log("testdir", "Unique correction two", "context2", corrections.LogOpts{}, nowFunc(), fs)
	_ = corrections.Log("testdir", "Repeated correction", "context3", corrections.LogOpts{}, nowFunc(), fs)
	_ = corrections.Log("testdir", "Repeated correction", "context4", corrections.LogOpts{}, nowFunc(), fs)

	patterns, err := corrections.Analyze("testdir", corrections.AnalyzeOpts{
		MinOccurrences: 1,
	}, fs)
	g.Expect(err).ToNot(HaveOccurred())

	// Should return all patterns including single occurrences
	g.Expect(len(patterns)).To(BeNumerically(">=", 3))
}

// TEST-728 traces: TASK-042
// Test Analyze property: MinOccurrences filters correctly
func TestAnalyze_Property_MinOccurrencesFilters(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		fs := &MockFS{}

		threshold := rapid.IntRange(1, 5).Draw(rt, "threshold")

		// Create corrections with known counts
		for i := 1; i <= 5; i++ {
			message := rapid.StringMatching(`pattern[0-9]+`).Draw(rt, "message")
			for j := 0; j < i; j++ {
				_ = corrections.Log("testdir", message, "ctx", corrections.LogOpts{}, nowFunc(), fs)
			}
		}

		patterns, err := corrections.Analyze("testdir", corrections.AnalyzeOpts{
			MinOccurrences: threshold,
		}, fs)
		g.Expect(err).ToNot(HaveOccurred())

		// All returned patterns must meet threshold
		for _, p := range patterns {
			g.Expect(p.Count).To(BeNumerically(">=", threshold))
		}
	})
}

// TEST-729 traces: TASK-042
// Test Analyze includes example entries in pattern
func TestAnalyze_PatternIncludesExamples(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{}

	_ = corrections.Log("testdir", "Stop words in patterns", "context A", corrections.LogOpts{SessionID: "sess-1"}, nowFunc(), fs)
	_ = corrections.Log("testdir", "Stop words in patterns", "context B", corrections.LogOpts{SessionID: "sess-2"}, nowFunc(), fs)

	patterns, err := corrections.Analyze("testdir", corrections.AnalyzeOpts{
		MinOccurrences: 2,
	}, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(patterns).To(HaveLen(1))

	// Pattern should include example entries for context
	g.Expect(patterns[0].Examples).ToNot(BeEmpty())
	g.Expect(len(patterns[0].Examples)).To(BeNumerically("<=", patterns[0].Count))
}
