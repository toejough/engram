package auto_test

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/auto"
)

func Test855d4b1i_Recall_ACandidateCannotReshapeTheRequestOrHoldACredential(t *testing.T) {
	t.Parallel()

	t.Run("named example: a note of 400 and more is whole, a chunk of 1,200 and more is cut", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		note := strings.Repeat("a note word ", 40)
		passage := strings.Repeat("a chunk word ", 100)

		step := auto.Recall(recallOf("glance"), []auto.Candidate{
			{ID: "n1.md", Kind: auto.KindNote, Date: "2026-10-17", Situation: "s", Text: note},
			{ID: "c1#turn-1", Kind: auto.KindPassage, Date: "2026-10-14", Text: passage},
		})

		lines := candidatesOf(step.Request.Prompt)

		g.Expect(lines).To(HaveLen(2))

		if len(lines) == 2 {
			g.Expect(lines[0]).To(Equal("- n1.md (2026-10-17; s): " + strings.TrimSpace(note)))
			g.Expect(lines[1]).To(Equal("- c1#turn-1 (2026-10-14; passage): " + strings.TrimSpace(passage)[:1200]))
		}

		g.Expect(step.Carry.Candidates[1].Text).To(Equal(passage), "the carry holds the chunk in full")
	})

	rapid.Check(t, checkRecallIsSafe)
}

// candidatesOf returns the lines of the section of candidates of a prompt: the lines between the tags that hold it.
func candidatesOf(prompt string) []string {
	_, rest, _ := strings.Cut(prompt, "<candidates>\n")
	section, _, _ := strings.Cut(rest, "\n</candidates>")

	return strings.Split(section, "\n")
}

// checkRecallIsSafe asks the recall about a note and a chunk of hostile texts that hold the two credentials, and fails
// when the request can be closed or reshaped, holds a credential, or holds a chunk of more than 1,200 characters.
func checkRecallIsSafe(rt *rapid.T) {
	rt.Helper()

	first := credentialShape().Draw(rt, "first")
	second := credentialShape().Draw(rt, "second")
	noteText := tagText().Draw(rt, "note") + " " + first.text
	passageText := tagText().Draw(rt, "passage") + " " + second.text +
		strings.Repeat(" word", rapid.IntRange(0, 400).Draw(rt, "padding"))

	step := auto.Recall(recallOf("deep"), []auto.Candidate{
		{ID: "n1.md", Kind: auto.KindNote, Date: "2026-10-17", Situation: tagText().Draw(rt, "situation"), Text: noteText},
		{ID: "c1#turn-1", Kind: auto.KindPassage, Date: "2026-10-14", Text: passageText},
	})

	if counts := tagCounts(step.Request.Prompt); !slices.Equal(counts[:2], []int{1, 1}) ||
		strings.Count(step.Request.Prompt, "<candidates>") != 1 ||
		strings.Count(step.Request.Prompt, "</candidates>") != 1 {
		rt.Fatalf("the tags are %v in the prompt %q", counts, step.Request.Prompt)
	}

	lines := candidatesOf(step.Request.Prompt)
	if len(lines) != 2 {
		rt.Fatalf("two candidates made %d lines: %q", len(lines), lines)
	}

	for _, line := range slices.Concat(windowOf(step.Request.Prompt), lines) {
		if strings.Contains(line, "<") {
			rt.Fatalf("a line of the window or of the candidates holds a <: %q", line)
		}
	}

	for _, secret := range []string{first.secret, second.secret} {
		if strings.Contains(step.Request.Prompt, secret) {
			rt.Fatalf("the request holds the secret %q: %q", secret, step.Request.Prompt)
		}
	}

	// the cut comes before the escape, which writes a < as four characters
	_, text, _ := strings.Cut(lines[1], "(2026-10-14; passage): ")
	if length := utf8.RuneCountInString(strings.ReplaceAll(text, "&lt;", "<")); length > 1200 {
		rt.Fatalf("a chunk of %d characters is in the request", length)
	}
}

// recallOf is the reply of the phrases judgement to the recall, with a window and a depth.
func recallOf(depth string) auto.RecallInput {
	return auto.RecallInput{
		Session: "4f1c0a9e-6f1d-4f5e-9d58-0f3d9a3c8e11", Cue: "prompt", Reply: `{"phrases":["a"]}`,
		Carry: auto.Carry{Cue: "prompt", Depth: depth, Window: "User: Prepare a test run."},
	}
}
