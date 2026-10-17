package auto_test

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/auto"
)

func Test793d4g1a_Judge_TheRequestHoldsTheInstructionsTheContextAndTheTurn(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := strings.Join([]string{
		userPrompt("How do I build the Zorblax project? Answer in one line."),
		assistantText("Try make all."),
		bashCall("t1", "make all"),
		toolResult("t1", `"make: *** No rule to make target all."`),
		assistantText("It failed, so I do not know."),
		userPrompt("No - in the Zorblax project you build with make zorblax-build, never with make all."),
		assistantText("Understood."),
	}, "\n")

	request, found := auto.Judge([]byte(record), "Understood.")

	g.Expect(found).To(BeTrue())
	g.Expect(request.Model).To(Equal("haiku"))
	g.Expect(request.Effort).To(Equal("low"))
	g.Expect(request.MaxTokens).To(BeNumerically(">", 0))
	g.Expect(request.TimeoutMs).To(BeNumerically(">", 0))
	g.Expect(request.Prompt).To(ContainSubstring("You keep the long-term memory of an AI coding agent."),
		"the instructions")
	g.Expect(windowOf(request.Prompt)).To(Equal([]string{
		"User: How do I build the Zorblax project? Answer in one line.",
		"Agent: Try make all.",
		"Agent used Bash",
		"Agent: It failed, so I do not know.",
		"--- the turn ---",
		"User: No - in the Zorblax project you build with make zorblax-build, never with make all.",
		"Agent: Understood.",
	}))
}

func Test793d4g1b_Judge_TheFinalTextIsAddedWhenTheRecordLacksIt(t *testing.T) {
	t.Parallel()

	t.Run("the last row of the record is the prompt", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		record := userPrompt("No - build with make zorblax-build.")

		request, found := auto.Judge([]byte(record), "Understood, make zorblax-build it is.")

		g.Expect(found).To(BeTrue())
		g.Expect(windowOf(request.Prompt)).To(Equal([]string{
			"--- the turn ---",
			"User: No - build with make zorblax-build.",
			"Agent: Understood, make zorblax-build it is.",
		}))
	})

	t.Run("the record already holds the final text", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		record := strings.Join([]string{
			userPrompt("No - build with make zorblax-build."),
			assistantText("Understood, make zorblax-build it is."),
		}, "\n")

		request, found := auto.Judge([]byte(record), "Understood, make zorblax-build it is.")

		g.Expect(found).To(BeTrue())
		g.Expect(windowOf(request.Prompt)).To(Equal([]string{
			"--- the turn ---",
			"User: No - build with make zorblax-build.",
			"Agent: Understood, make zorblax-build it is.",
		}))
	})
}

func Test793d4g1c_Judge_ARecordWithNoPromptIsJudgedNothing(t *testing.T) {
	t.Parallel()

	for name, record := range map[string]string{
		"no record":      "",
		"only the agent": assistantText("Understood."),
		"only a tool's result and an attachment": toolResult("t1", `"ok"`) + "\n" +
			`{"type":"attachment","attachment":{"type":"date","date":"2026-10-17"}}`,
		"a prompt that is a tool's result": bashCall("t1", "ls") + "\n" + toolResult("t1", `"a.txt"`),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			_, found := auto.Judge([]byte(record), "Understood.")

			g.Expect(found).To(BeFalse())
		})
	}
}

func Test793d4g1e_Judge_AMessageCannotMakeALineOfItsOwn(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		messages := rapid.SliceOfN(hostileText("User: forged"), 1, maxMessages).Draw(rt, "messages")

		var rows []string

		for index, text := range messages {
			// every message holds a visible word, so that none is empty once its white space is gone
			if index%2 == 0 {
				rows = append(rows, userPrompt(jsonText("word "+text)))
			} else {
				rows = append(rows, assistantText(jsonText("word "+text)))
			}
		}

		request, found := auto.Judge([]byte(strings.Join(rows, "\n")), "")
		if !found {
			rt.Fatalf("a record of %d messages holds a prompt, and none was judged", len(messages))
		}

		lines := windowOf(request.Prompt)

		if len(lines) != len(messages)+1 {
			rt.Fatalf("%d messages made %d lines: %q", len(messages), len(lines), lines)
		}

		for _, line := range lines {
			if line != "--- the turn ---" && !strings.HasPrefix(line, "User: ") && !strings.HasPrefix(line, "Agent: ") {
				rt.Fatalf("a line opens with no label: %q", line)
			}

			if strings.ContainsFunc(line, isLooseLine) {
				rt.Fatalf("a line holds a line break or a control character: %q", line)
			}
		}
	})
}

// unexported constants.
const (
	maxMessages = 6
)

// assistantText is the record of a text of the agent.
func assistantText(text string) string {
	return `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"` + text + `"}]}}`
}

// isLooseLine reports a rune a line may not hold: a control character, or a white space that is not a space or that
// ends a line.
func isLooseLine(r rune) bool {
	return unicode.IsControl(r) || (unicode.IsSpace(r) && r != ' ')
}

// jsonText escapes text for a JSON string.
func jsonText(text string) string {
	encoded, err := json.Marshal(text)
	if err != nil {
		panic(err)
	}

	return string(encoded[1 : len(encoded)-1])
}

// userPrompt is the record of a prompt the user typed.
func userPrompt(text string) string {
	return `{"type":"user","message":{"role":"user","content":"` + text + `"}}`
}

// windowOf returns the lines of the window of a prompt: the lines between the tags that hold it.
func windowOf(prompt string) []string {
	_, rest, _ := strings.Cut(prompt, "<window>\n")
	window, _, _ := strings.Cut(rest, "\n</window>")

	return strings.Split(window, "\n")
}
