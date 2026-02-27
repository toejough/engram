package memory

import (
	"testing"

	. "github.com/onsi/gomega"
)

// TestExtractSignificantWords_FiltersShortWords verifies words <=3 chars are excluded.
func TestExtractSignificantWords_FiltersShortWords(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	words := extractSignificantWords("git do a task now")

	g.Expect(words).To(ContainElement("task"))
	g.Expect(words).ToNot(ContainElement("git"))
	g.Expect(words).ToNot(ContainElement("do"))
	g.Expect(words).ToNot(ContainElement("a"))
	g.Expect(words).ToNot(ContainElement("now"))
}

// TestExtractSignificantWords_StripsPunctuation verifies punctuation is stripped.
func TestExtractSignificantWords_StripsPunctuation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	words := extractSignificantWords("testing, workflow. quality!")

	g.Expect(words).To(ContainElement("testing"))
	g.Expect(words).To(ContainElement("workflow"))
	g.Expect(words).To(ContainElement("quality"))
}

// TestTopicsMatch_NoMatch verifies false when no significant overlap.
func TestTopicsMatch_NoMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := topicsMatch("git commit trailer", "memory storage", "pizza delivery service")

	g.Expect(result).To(BeFalse())
}

// TestTopicsMatch_QueryWordInCorrection verifies match when query word appears in correction.
func TestTopicsMatch_QueryWordInCorrection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := topicsMatch("testing workflow", "some result content", "testing strategy overview")

	g.Expect(result).To(BeTrue())
}

// TestTopicsMatch_ResultWordInCorrection verifies match when result word appears in correction.
func TestTopicsMatch_ResultWordInCorrection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := topicsMatch("unrelated query", "testing workflow step", "testing strategy overview")

	g.Expect(result).To(BeTrue())
}

// TestTopicsMatch_ShortWords verifies short words (<=3 chars) are not matched.
func TestTopicsMatch_ShortWords(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// "git", "do", "a" are all 3 chars or fewer, should not match
	result := topicsMatch("git do a", "do it now", "git do a thing")

	g.Expect(result).To(BeFalse())
}
