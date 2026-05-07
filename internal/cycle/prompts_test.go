package cycle_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cycle"
)

func TestLearnExtractionPrompt_IncludesTranscript(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	prompt := cycle.LearnExtractionPrompt("the transcript body")
	g.Expect(prompt).To(ContainSubstring("the transcript body"))
	g.Expect(prompt).To(ContainSubstring("Output a JSON array"))
	g.Expect(prompt).To(ContainSubstring("feedback"))
	g.Expect(prompt).To(ContainSubstring("fact"))
}

func TestQueryProposalPrompt_IncludesTranscript(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	prompt := cycle.QueryProposalPrompt("transcript here")
	g.Expect(prompt).To(ContainSubstring("transcript here"))
	g.Expect(prompt).To(ContainSubstring("NO QUERIES"))
	g.Expect(prompt).To(ContainSubstring("1-5 targeted recall queries"))
}
