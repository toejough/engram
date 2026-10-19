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

func Test855d4a1b_Moment_TheInstructionsCountARequestToRememberWhateverTheAgentAnswers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	step := auto.Moment(nil, auto.MomentInput{
		Session: "4f1c0a9e-6f1d-4f5e-9d58-0f3d9a3c8e11", Cue: "turn-end", Origin: "composer", Turn: "a-turn",
		Text:   "Remember that the Zorblax staging database is reset on Sundays, but do not save it yourself.",
		Answer: "I did not save it.",
	})

	g.Expect(strings.Join(strings.Fields(strings.ToLower(step.Request.System)), " ")).To(
		ContainSubstring("whatever the agent answers, even when it says that it did not save it"),
		"a request to remember is a learn moment though the agent says it saved nothing (793a2c1i)")
}

func Test855d4a1b_Moment_TheInstructionsCountAnUncertainApproachThatSucceededWithEvidence(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	step := auto.Moment([]byte(userPrompt("Say you do not know, run the check, and say if it passed.")), auto.MomentInput{
		Session: "4f1c0a9e-6f1d-4f5e-9d58-0f3d9a3c8e11", Cue: "turn-end", Origin: "composer", Turn: "a-turn",
		Text: "Say you do not know, run the check, and say if it passed.", Answer: "The check passed.",
	})

	g.Expect(strings.ToLower(step.Request.System)).To(
		ContainSubstring("tried it, and reports the outcome that it observed"),
		"an approach the agent said it did not know would work, that it tried and saw succeed, is a lesson")
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
			words := strings.Join(strings.Fields(step.Request.System), " ")

			g.Expect(words).To(And(
				ContainSubstring("the tool call failed and the agent cannot immediately explain it"),
				ContainSubstring("the agent is about to start building a new approach")),
				"the cues of the recall skill, in its words")
			g.Expect(words).NotTo(Or(ContainSubstring("plain and trivial"), ContainSubstring("begins a change")),
				"no wording of the prompt's own")
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

	var spelled strings.Builder

	for _, r := range secret {
		fmt.Fprintf(&spelled, `\u%04x`, r)
	}

	escaped := spelled.String()

	object := json.RawMessage(
		`{"command":"echo ` + escaped + ` done","note":"` + escaped + `","list":["` + escaped + `",1]}`)
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

func Test855d4a1d_Moment_ALargeIntegerKeepsItsDigits(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	step := auto.Moment(nil, auto.MomentInput{
		Cue: "before-tool", Origin: "composer", Text: "go",
		Event: auto.ToolEvent{Tool: "Bash", Input: json.RawMessage(`{"id":12345678901234567890,"ratio":0.5}`)},
	})

	g.Expect(windowOf(step.Request.Prompt)).To(
		ContainElement(`Agent used Bash with {"id":12345678901234567890,"ratio":0.5}`))
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

func Test855d4a1d_Moment_AValueOfASensitiveNameIsRedactedWholeWhateverItHolds(t *testing.T) {
	t.Parallel()

	// the value holds a quote, which the encoded text spells \" and which ends a match that reads the encoded text
	// (a {{...}} placeholder, no real secret)
	tail := "er2 very secret-tail"
	value := "{{HUNT}}" + `\"` + tail
	inner := `{"password":"` + value + `"}`
	doubled := rawJSON(t, inner)
	listed := rawJSON(t, []string{strings.Replace(inner, "password", "token", 1)})
	inList := rawJSON(t, map[string][]string{"list": {strings.Replace(inner, "password", "secret", 1)}})

	inputs := map[string]json.RawMessage{
		"an object":          json.RawMessage(inner),
		"a nested object":    json.RawMessage(`{"headers":{"x":[{"api_key":"` + value + `"}]}}`),
		"a number":           json.RawMessage(`{"pwd":1234567}`),
		"a string of JSON":   doubled,
		"an array of JSON":   listed,
		"a string in a list": inList,
	}

	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			step := auto.Moment(nil, auto.MomentInput{
				Cue: "batch-done", Origin: "composer", Text: "go",
				Event: auto.ToolEvent{Calls: []auto.ToolCall{{Tool: "Bash", Input: input, Response: input}}},
			})

			g.Expect(step.Request.Prompt).NotTo(ContainSubstring("secret-tail"), "the rest of the value")
			g.Expect(step.Request.Prompt).NotTo(ContainSubstring("1234567"), "a number of a sensitive name")
			g.Expect(step.Request.Prompt).To(ContainSubstring("[redacted]"))
		})
	}

	t.Run("the value is replaced whole", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		step := auto.Moment(nil, auto.MomentInput{
			Cue: "before-tool", Origin: "composer", Text: "go",
			Event: auto.ToolEvent{Tool: "Bash", Input: json.RawMessage(inner)},
		})

		g.Expect(windowOf(step.Request.Prompt)).To(ContainElement(`Agent used Bash with {"password":[redacted]}`))
	})
}

func Test855d4a1i_Moment_AQuestionOfStatusWithNoDecisionInItIsNeverDeep(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	step := auto.Moment([]byte(userPrompt("what's next?")), auto.MomentInput{
		Session: "4f1c0a9e-6f1d-4f5e-9d58-0f3d9a3c8e11", Cue: "prompt", Origin: "composer", Text: "what's next?",
	})

	g.Expect(flattened(step.Request.System)).To(ContainSubstring(
		"a question about the status or progress of the work or about what comes next, "+
			"with no decision in it, is never deep"),
		"the deep recall is for a decision, and a status question has none (the dogfood of 2026-10-19)")
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

// rawJSON is the JSON text of a value.
func rawJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encoding %v: %v", value, err)
	}

	return encoded
}
