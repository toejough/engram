package auto_test

import (
	"encoding/json"
	"fmt"
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

func Test855d4a1b_Moment_TheInstructionsCountTheAgentsPlainCorrectionOfItsOwnClaim(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	step := auto.Moment([]byte(userPrompt("Guess the port, read zorblax.conf and say what it is.")), auto.MomentInput{
		Session: "4f1c0a9e-6f1d-4f5e-9d58-0f3d9a3c8e11", Cue: "turn-end", Origin: "composer", Turn: "a-turn",
		Text: "Guess the port, read zorblax.conf and say what it is.", Answer: "My guess of 8080 was wrong: it is 7731.",
	})

	g.Expect(strings.ToLower(step.Request.System)).To(ContainSubstring("plainly corrects its own earlier claim"),
		"a guess that the agent then corrects from what it read is a lesson, though a guess alone is not")
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

// checkMomentIsSafe asks the moment for a prompt and for each tool cue, with a conversation and an event of hostile
// texts that hold the two credentials, and fails when a request can be closed, reshaped or read with a credential.
func checkMomentIsSafe(rt *rapid.T, first, second credential) {
	rt.Helper()

	prompt := tagText().Draw(rt, "prompt")
	message := tagText().Draw(rt, "message")
	record := userPrompt(jsonText("word "+message+" "+first.text)) + "\n" +
		assistantText(jsonText("word "+message+" "+second.text))
	hostile := json.RawMessage(`"` + jsonText("zq9x "+message+" "+first.text) + `"`)
	event := auto.ToolEvent{
		Tool: "zq9x" + prompt, Input: hostile, Error: "zq9x " + prompt + " " + second.text,
		Calls: []auto.ToolCall{{Tool: "Bash", Input: hostile, Response: hostile}},
	}

	for _, cue := range []string{"prompt", "before-tool", "failure", "batch-done"} {
		step := auto.Moment([]byte(record), auto.MomentInput{
			Cue: cue, Origin: "composer", Text: "zq9x " + prompt + " " + second.text, Event: event,
		})

		if counts := tagCounts(step.Request.Prompt); !slices.Equal(counts[:2], []int{1, 1}) {
			rt.Fatalf("the tags of the window are %v in the prompt %q (cue %s)", counts, step.Request.Prompt, cue)
		}

		for _, line := range windowOf(step.Request.Prompt) {
			if strings.Contains(line, "<") {
				rt.Fatalf("a line of the window holds a <: %q (cue %s)", line, cue)
			}
		}

		for _, secret := range []string{first.secret, second.secret} {
			if strings.Contains(step.Request.Prompt, secret) {
				rt.Fatalf("the request holds the secret %q: %q (cue %s)", secret, step.Request.Prompt, cue)
			}
		}

		if strings.Contains(step.Request.System, "zq9x") {
			rt.Fatalf("the system holds text of the session: %q (cue %s)", step.Request.System, cue)
		}
	}
}

func Test855d4a1c_Moment_EachToolCuePutsItsEventInTheWindow(t *testing.T) {
	t.Parallel()

	const (
		turnText = "Tag v1.4.0 and tell me when the release is done."
		input    = `{"command":"make ci"}`
	)

	record := strings.Join([]string{
		userPrompt(turnText),
		assistantText("Running the build."),
	}, "\n")
	turnLine := "User: " + turnText
	calls := []auto.ToolCall{
		{Tool: "Bash", Input: json.RawMessage(`{"command":"touch tag-v1.4.0"}`), Response: json.RawMessage(`"tagged"`)},
		{Tool: "Read", Input: json.RawMessage(`{"file_path":"CHANGELOG.md"}`), Response: json.RawMessage(`{"lines":3}`)},
	}

	cases := []struct {
		name  string
		input auto.MomentInput
		want  []string
	}{
		{
			name: "before-tool", input: auto.MomentInput{
				Cue:   "before-tool",
				Event: auto.ToolEvent{Tool: "Bash", Input: json.RawMessage(input)},
			},
			want: []string{turnLine, "Agent: Running the build.", `Agent used Bash with {"command":"make ci"}`},
		},
		{
			name: "failure", input: auto.MomentInput{
				Cue:   "failure",
				Event: auto.ToolEvent{Tool: "Bash", Input: json.RawMessage(input), Error: "make: *** [ci] Error 1"},
			},
			want: []string{
				turnLine, "Agent: Running the build.", `Agent used Bash with {"command":"make ci"}`,
				"Agent used Bash failed: make: *** [ci] Error 1",
			},
		},
		{
			name:  "batch-done",
			input: auto.MomentInput{Cue: "batch-done", Event: auto.ToolEvent{Calls: calls}},
			want: []string{
				turnLine, "Agent: Running the build.",
				`Agent used Bash with {"command":"touch tag-v1.4.0"}`, "Agent used Bash returned: tagged",
				`Agent used Read with {"file_path":"CHANGELOG.md"}`, `Agent used Read returned: {"lines":3}`,
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			testCase.input.Session = "4f1c0a9e-6f1d-4f5e-9d58-0f3d9a3c8e11"
			testCase.input.Origin = "composer"
			testCase.input.Text = turnText

			step := auto.Moment([]byte(record), testCase.input)

			g.Expect(windowOf(step.Request.Prompt)).To(Equal(testCase.want), "the window ends with the line of the event")
			g.Expect(step.Request.Model).To(Equal("haiku"))
			g.Expect(step.Request.TimeoutMs).To(BeZero(), "a recall cue has no time limit")
			g.Expect(step.Request.System).To(And(ContainSubstring("Agent used <tool> with <input>"),
				ContainSubstring("Agent used <tool> failed: <error>"),
				ContainSubstring("Agent used <tool> returned: <response>")), "the system tells the lines of an event")
			g.Expect(step.Carry.Cue).To(Equal(testCase.input.Cue))
			g.Expect(step.Carry.Window).To(Equal(strings.Join(windowOf(step.Request.Prompt), "\n")))
		})
	}

	t.Run("a record without the turn's prompt still has its text in the window", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		step := auto.Moment(nil, auto.MomentInput{
			Cue: "before-tool", Origin: "composer", Text: turnText,
			Event: auto.ToolEvent{Tool: "Bash", Input: json.RawMessage(input)},
		})

		g.Expect(windowOf(step.Request.Prompt)).To(Equal([]string{turnLine, `Agent used Bash with {"command":"make ci"}`}))
	})
}

func Test855d4a1d_Moment_ACredentialInEscapedFormInAnEventIsRedacted(t *testing.T) {
	t.Parallel()

	// the issuer's prefix and the body, built here and written whole in no literal; each character is written as a
	// unicode escape of the JSON, which the redaction must read decoded (the {{AKIA}} placeholder convention)
	secret := "AK" + "IA" + "IOSFODNN7EXAMPLE"
	escaped := ""

	for _, r := range secret {
		escaped += fmt.Sprintf(`\u%04x`, r)
	}

	object := json.RawMessage(`{"command":"echo ` + escaped + ` done","note":"` + escaped + `"}`)
	inputs := map[string]auto.MomentInput{
		"an object input": {Cue: "before-tool", Event: auto.ToolEvent{Tool: "Bash", Input: object}},
		"a batch response": {
			Cue: "batch-done", Event: auto.ToolEvent{Calls: []auto.ToolCall{{Tool: "Bash", Input: object, Response: object}}},
		},
		"a string response": {
			Cue: "batch-done", Event: auto.ToolEvent{Calls: []auto.ToolCall{{Tool: "Bash", Response: json.RawMessage(
				`"` + escaped + `"`)}}},
		},
		"a failure input": {Cue: "failure", Event: auto.ToolEvent{Tool: "Bash", Input: object, Error: "x"}},
		"a key": {Cue: "before-tool", Event: auto.ToolEvent{Tool: "Bash", Input: json.RawMessage(
			`{"` + escaped + `":"x"}`)}},
	}

	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			input.Origin = "composer"
			input.Text = "go"

			step := auto.Moment(nil, input)

			g.Expect(step.Request.Prompt).NotTo(ContainSubstring(secret), "the decoded credential")
			g.Expect(step.Request.Prompt).NotTo(ContainSubstring("EXAMPLE"), "the body of the credential")
			g.Expect(step.Request.Prompt).NotTo(ContainSubstring(escaped[:12]), "the credential, still escaped")
		})
	}
}
