package auto_test

import (
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/auto"
)

func Test855d4a1a_Moment_APromptBecomesARequestForTheCheapTier(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := strings.Join([]string{
		userPrompt("How do I build the Zorblax project?"),
		assistantText("Try make all."),
	}, "\n")

	step := auto.Moment([]byte(record), auto.MomentInput{
		Session: "4f1c0a9e-6f1d-4f5e-9d58-0f3d9a3c8e11", Cue: "prompt", Origin: "composer",
		Text: "Prepare a test run against the Zorblax staging database tonight.",
	})

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

func Test855d4a1b_Moment_ATurnThatEndedBecomesALearnMomentRequest(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := strings.Join([]string{
		userPrompt("How do I build the Zorblax project?"),
		assistantText("Try make all."),
		userPrompt("No - build with make zorblax-build."),
		assistantText("Understood."),
	}, "\n")

	step := auto.Moment([]byte(record), auto.MomentInput{
		Session: "4f1c0a9e-6f1d-4f5e-9d58-0f3d9a3c8e11", Cue: "turn-end", Origin: "composer", Turn: "a-turn",
		Text: "No - build with make zorblax-build.", Answer: "Understood.",
	})

	request := step.Request
	g.Expect(request.Model).To(Equal("haiku"))
	g.Expect(request.Effort).To(Equal("low"))
	g.Expect(request.TimeoutMs).To(Equal(10000), "the row of the learn-moment judgement")
	g.Expect(request.MaxTokens).To(BeNumerically(">", 0))
	g.Expect(request.System).To(ContainSubstring("explicit save requests"), "the description of the learn skill")
	g.Expect(request.System).To(And(ContainSubstring(`{"learn":true}`), ContainSubstring(`{"learn":false}`)),
		"the reply shape")
	g.Expect(request.System).NotTo(ContainSubstring(`"depth"`), "no depth for a learn moment")
	g.Expect(request.System).NotTo(ContainSubstring("Zorblax"), "the system holds nothing of the session")
	g.Expect(windowOf(request.Prompt)).To(Equal([]string{
		"User: How do I build the Zorblax project?",
		"Agent: Try make all.",
		"--- the turn ---",
		"User: No - build with make zorblax-build.",
		"Agent: Understood.",
	}))
	g.Expect(request.Prompt).NotTo(ContainSubstring("explicit save requests"), "none of the instructions")
	g.Expect(step.Carry.Cue).To(Equal("turn-end"))
	g.Expect(step.Carry.Window).To(Equal(strings.Join(windowOf(request.Prompt), "\n")))
}

func Test855d4a1d_Moment_APromptAMessageOrAnEventCannotCloseTheWindowOrHoldACredential(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		checkMomentIsSafe(rt, credentialShape().Draw(rt, "first"), credentialShape().Draw(rt, "second"))
	})

	// the named example: one credential of every shape
	for _, spec := range credentialSpecs() {
		t.Run(spec.name, func(t *testing.T) {
			t.Parallel()

			rapid.Check(t, func(rt *rapid.T) { checkMomentIsSafe(rt, spec.make(rt), spec.make(rt)) })
		})
	}
}

// checkMomentIsSafe asks the moment for a prompt and a conversation of hostile texts that hold the two credentials, and
// fails when the request can be closed, reshaped or read with a credential.
func checkMomentIsSafe(rt *rapid.T, first, second credential) {
	rt.Helper()

	prompt := tagText().Draw(rt, "prompt")
	message := tagText().Draw(rt, "message")
	record := userPrompt(jsonText("word "+message+" "+first.text)) + "\n" +
		assistantText(jsonText("word "+message+" "+second.text))

	step := auto.Moment([]byte(record), auto.MomentInput{
		Cue: "prompt", Origin: "composer", Text: "zq9x " + prompt + " " + second.text,
	})

	if counts := tagCounts(step.Request.Prompt); !slices.Equal(counts[:2], []int{1, 1}) {
		rt.Fatalf("the tags of the window are %v in the prompt %q", counts, step.Request.Prompt)
	}

	for _, line := range windowOf(step.Request.Prompt) {
		if strings.Contains(line, "<") {
			rt.Fatalf("a line of the window holds a <: %q", line)
		}
	}

	for _, secret := range []string{first.secret, second.secret} {
		if strings.Contains(step.Request.Prompt, secret) {
			rt.Fatalf("the request holds the secret %q: %q", secret, step.Request.Prompt)
		}
	}

	if strings.Contains(step.Request.System, "zq9x") {
		rt.Fatalf("the system holds text of the session: %q", step.Request.System)
	}
}

func Test855d4a1b_Moment_TheInstructionsCountTheAgentsPlainCorrectionOfItsOwnClaimEvenAfterAGuessItWasAsked(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	step := auto.Moment([]byte(userPrompt("Guess the port, read zorblax.conf and say what it is.")), auto.MomentInput{
		Session: "4f1c0a9e-6f1d-4f5e-9d58-0f3d9a3c8e11", Cue: "turn-end", Origin: "composer", Turn: "a-turn",
		Text: "Guess the port, read zorblax.conf and say what it is.", Answer: "My guess of 8080 was wrong: it is 7731.",
	})

	g.Expect(strings.ToLower(step.Request.System)).To(ContainSubstring("plainly corrects its own earlier claim"),
		"a guess that the agent then corrects from what it read is a lesson, though a guess alone is not")
}
