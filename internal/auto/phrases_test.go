package auto_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
)

func Test855d4v1a_Phrases_TheRequestHoldsTheTenAnglesAndTheWindow(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	carry := auto.Carry{
		Cue:    "prompt",
		Window: "User: How do I build the Zorblax project?\nUser: Prepare a test run against the Zorblax staging.",
	}

	step, found, err := auto.Phrases(auto.PhrasesInput{
		Session: "4f1c0a9e-6f1d-4f5e-9d58-0f3d9a3c8e11", Cue: "prompt", Reply: `{"depth":"deep"}`, Carry: carry,
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())

	request := step.Request
	g.Expect(request.Model).To(Equal("sonnet"))
	g.Expect(request.Effort).To(BeEmpty(), "the effort is the cheap tier's")
	g.Expect(request.TimeoutMs).To(BeZero(), "a recall cue has no time limit")
	g.Expect(request.MaxTokens).To(BeNumerically(">", 0))
	g.Expect(request.System).To(And(
		ContainSubstring("situation or setting"), ContainSubstring("intent"), ContainSubstring("concrete action"),
		ContainSubstring("problem or blocker"), ContainSubstring("candidate approach"), ContainSubstring("tooling"),
		ContainSubstring("prior related work"), ContainSubstring("adjacent technique"),
		ContainSubstring("failure mode to avoid"), ContainSubstring("domain"),
	), "the ten angles")
	g.Expect(request.System).To(ContainSubstring("implementing Claude Code hooks"), "the keying to the task")
	g.Expect(request.System).To(ContainSubstring(`{"phrases":[`), "the reply shape")
	g.Expect(request.System).NotTo(ContainSubstring("Zorblax"), "the system holds nothing of the session")
	g.Expect(request.Prompt).To(Equal("<window>\n"+carry.Window+"\n</window>"), "the window of the carry")
	g.Expect(step.Carry).To(Equal(auto.Carry{Cue: "prompt", Depth: "deep", Window: carry.Window}))
}
