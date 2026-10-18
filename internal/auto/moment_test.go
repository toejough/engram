package auto_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
)

func Test855d4a1a_Moment_APromptBecomesARequestForTheCheapTier(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := strings.Join([]string{
		userPrompt("How do I build the Zorblax project?"),
		assistantText("Try make all."),
	}, "\n")

	step, found, err := auto.Moment([]byte(record), auto.MomentInput{
		Session: "4f1c0a9e-6f1d-4f5e-9d58-0f3d9a3c8e11", Cue: "prompt", Origin: "composer",
		Text: "Prepare a test run against the Zorblax staging database tonight.",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())

	request := step.Request
	g.Expect(request.Model).To(Equal("haiku"))
	g.Expect(request.Effort).To(Equal("low"))
	g.Expect(request.TimeoutMs).To(BeZero(), "a recall cue has no time limit")
	g.Expect(request.MaxTokens).To(BeNumerically(">", 0))
	g.Expect(request.System).To(ContainSubstring("more than a single tool call"), "the rule of the recall skill")
	g.Expect(request.System).To(ContainSubstring(`{"depth":`), "the reply shape")
	g.Expect(request.System).To(And(ContainSubstring("none"), ContainSubstring("glance"), ContainSubstring("deep")))
	g.Expect(request.System).NotTo(ContainSubstring(`"phrases"`), "the moment gives no phrase")
	g.Expect(request.System).NotTo(ContainSubstring("Zorblax"), "the system holds nothing of the session")
	g.Expect(windowOf(request.Prompt)).To(Equal([]string{
		"User: How do I build the Zorblax project?",
		"Agent: Try make all.",
		"User: Prepare a test run against the Zorblax staging database tonight.",
	}))
	g.Expect(request.Prompt).NotTo(ContainSubstring("more than a single tool call"), "none of the instructions")
	g.Expect(step.Carry.Cue).To(Equal("prompt"))
	g.Expect(step.Carry.Window).To(Equal(strings.Join(windowOf(request.Prompt), "\n")))
}
