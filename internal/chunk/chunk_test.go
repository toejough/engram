package chunk_test

import (
	"strings"
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
)

func TestChunkMarkdownIgnoresFencedHeadings(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	raw := strings.Join([]string{
		"## Real section",
		"Content before the fence shows the convention applied in real code below.",
		"```",
		"# not a heading, just a comment inside a code fence",
		"```",
		"Content after the fence continues the very same section without a split.",
	}, "\n")

	chunks := chunk.Markdown(raw, 1500)

	g.Expect(chunks).To(gomega.HaveLen(1))

	if len(chunks) != 1 {
		return
	}

	g.Expect(chunks[0].Text).To(gomega.ContainSubstring("not a heading"))
}

func TestChunkMarkdownSplitsOnHeadings(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	raw := strings.Join([]string{
		"# Title",
		"Intro paragraph with enough words to clear the minimum noise threshold easily.",
		"## Section A",
		"Section A content explains the first convention in reasonable detail for retrieval.",
		"## Section B",
		"Section B content explains the second convention in reasonable detail for retrieval.",
	}, "\n")

	chunks := chunk.Markdown(raw, 1500)

	g.Expect(chunks).To(gomega.HaveLen(3))

	if len(chunks) != 3 {
		return
	}

	g.Expect(chunks[0].Anchor).To(gomega.Equal("Title"))
	g.Expect(chunks[1].Anchor).To(gomega.Equal("Section A"))
	g.Expect(chunks[2].Text).To(gomega.ContainSubstring("second convention"))
}

func TestChunkMarkdownSplitsOversizedSectionOnParagraphs(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	para := strings.Repeat("paragraph words for the oversized section test ", 20) // ~940 chars
	raw := "## Big\n" + para + "\n\n" + para + "\n\n" + para

	chunks := chunk.Markdown(raw, 1500)

	g.Expect(len(chunks)).To(gomega.BeNumerically(">", 1))

	for _, c := range chunks {
		g.Expect(len(c.Text)).To(gomega.BeNumerically("<=", 1500))
		g.Expect(c.Anchor).To(gomega.Equal("Big"))
	}
}

func TestChunkTranscriptContentBeforeFirstUserTurnIsPreamble(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	stripped := strings.Join([]string{
		"ASSISTANT: leftover assistant text from a resumed session goes first here",
		"USER: now the actual first user turn with plenty of text to keep around",
	}, "\n")

	chunks := chunk.Transcript(stripped, 30, 1500)

	g.Expect(chunks).NotTo(gomega.BeEmpty())

	if len(chunks) == 0 {
		return
	}

	g.Expect(chunks[0].Anchor).To(gomega.Equal("preamble"))
}

func TestChunkTranscriptDropsNoise(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	chunks := chunk.Transcript("USER: ok", 500, 1500)

	g.Expect(chunks).To(gomega.BeEmpty())
}

func TestChunkTranscriptMergesTurnsToTarget(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Three short user turns; with a target larger than any single turn, they
	// merge into one chunk rather than three noisy fragments.
	stripped := strings.Join([]string{
		"USER: set up the linter config for the project",
		"ASSISTANT: done, added golangci config with the standard rules",
		"USER: also wire it into the build",
		"ASSISTANT: wired into targ check",
		"USER: thanks, looks good now",
		"ASSISTANT: anytime",
	}, "\n")

	chunks := chunk.Transcript(stripped, 500, 1500)

	g.Expect(chunks).To(gomega.HaveLen(1))

	if len(chunks) != 1 {
		return
	}

	g.Expect(chunks[0].Text).To(gomega.ContainSubstring("linter config"))
	g.Expect(chunks[0].Text).To(gomega.ContainSubstring("targ check"))
	g.Expect(chunks[0].Anchor).To(gomega.Equal("turn-1"))
}

func TestChunkTranscriptSplitsOversizedTurn(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// A single turn far beyond max must be split so every chunk embeds fully.
	lines := make([]string, 0, 81)
	lines = append(lines, "USER: giant dump follows")

	for range 80 {
		lines = append(lines, "ASSISTANT: "+strings.Repeat("x", 50))
	}

	stripped := strings.Join(lines, "\n")

	chunks := chunk.Transcript(stripped, 500, 1500)

	g.Expect(len(chunks)).To(gomega.BeNumerically(">", 1))

	for _, c := range chunks {
		g.Expect(len(c.Text)).To(gomega.BeNumerically("<=", 1500))
	}
}

func TestChunkTranscriptStartsNewChunkPastTarget(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	long := strings.Repeat("convention detail words here ", 20) // ~580 chars
	stripped := strings.Join([]string{
		"USER: first ask " + long,
		"ASSISTANT: first answer",
		"USER: second ask, distinct topic entirely",
		"ASSISTANT: second answer",
	}, "\n")

	chunks := chunk.Transcript(stripped, 500, 1500)

	g.Expect(chunks).To(gomega.HaveLen(2))

	if len(chunks) != 2 {
		return
	}

	g.Expect(chunks[0].Text).To(gomega.ContainSubstring("first ask"))
	g.Expect(chunks[1].Text).To(gomega.ContainSubstring("second ask"))
	g.Expect(chunks[1].Anchor).To(gomega.Equal("turn-2"))
}
