package context_test

import (
	"encoding/json"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	sessionctx "github.com/toejough/engram/internal/context"
)

// Test792b15c3a1_TranscriptReader_SystemReminderDropped verifies a user entry that is a system-reminder block
// yields no line, so the only line is the real question.
func Test792b15c3a1_TranscriptReader_SystemReminderDropped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	lines := []string{
		trc2vUserLine(t, "<system-reminder>hook output the user never typed</system-reminder>"),
		trc2vUserLine(t, "real question"),
	}

	result := trc2vRead(lines)

	g.Expect(result).To(Equal([]string{"USER: real question"}))
}

// Test792b15c3a2_TranscriptReader_InjectedTextsDropped verifies skill bodies, local command output and caveats and
// task notifications yield no line; a real question after them is the only line.
func Test792b15c3a2_TranscriptReader_InjectedTextsDropped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	lines := []string{
		trc2vUserLine(t, "Base directory for this skill: /skills/learn\n\n# Learn\nThe whole skill body."),
		trc2vUserLine(t, "<local-command-stdout>Catch you later!</local-command-stdout>"),
		trc2vUserLine(t, "<local-command-caveat>Caveat: generated while running local commands.</local-command-caveat>"),
		trc2vUserLine(t, "<task-notification><task-id>bo3apd2s2</task-id><status>completed</status></task-notification>"),
		trc2vUserLine(t, "real question"),
	}

	result := trc2vRead(lines)

	g.Expect(result).To(Equal([]string{"USER: real question"}))
}

// Test792b15c3a3_TranscriptReader_SlashCommandKeepsArguments verifies a slash command is replaced by its trimmed
// arguments and a slash command with no arguments yields no line.
func Test792b15c3a3_TranscriptReader_SlashCommandKeepsArguments(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	lines := []string{
		trc2vUserLine(t, "<command-message>learn</command-message>\n<command-name>/learn</command-name>\n"+
			"<command-args>  remember X\n</command-args>"),
		trc2vUserLine(t, "<command-message>clear</command-message>\n<command-name>/clear</command-name>"),
	}

	result := trc2vRead(lines)

	g.Expect(result).To(Equal([]string{"USER: remember X"}))
}

// Test792b15c4a1_TranscriptReader_Base64Replaced verifies a 200-character base64 run is replaced by the placeholder
// while the text around it stays.
func Test792b15c4a1_TranscriptReader_Base64Replaced(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const runLen = 200

	run := strings.Repeat("QUFB", runLen/len("QUFB"))
	g.Expect(run).To(HaveLen(runLen))

	lines := []string{trc2vUserLine(t, "screenshot: "+run+" end")}

	result := trc2vRead(lines)

	g.Expect(result).To(Equal([]string{"USER: screenshot: [base64 removed] end"}))
}

// Test792b15c4a2_TranscriptReader_OversizedLineCut verifies a 3000-character user text is cut, prefix included, to
// its first 2000 characters followed by [truncated].
func Test792b15c4a2_TranscriptReader_OversizedLineCut(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		textLen   = 3000
		keptChars = 2000
		unit      = "abcdefghi "
	)

	text := strings.Repeat(unit, textLen/len(unit))
	g.Expect(text).To(HaveLen(textLen))

	lines := []string{trc2vUserLine(t, text)}

	result := trc2vRead(lines)

	g.Expect(result).To(HaveLen(1))

	if len(result) != 1 {
		return
	}

	g.Expect(result[0]).To(Equal(("USER: " + text)[:keptChars] + "[truncated]"))
}

// trc2vRead reads the entries the way the transcript reader does: tool-summary mode.
func trc2vRead(lines []string) []string {
	return sessionctx.StripWithConfig(lines, sessionctx.StripConfig{ToolSummaryMode: true})
}

// trc2vUserLine builds a JSONL user entry with a string content field.
func trc2vUserLine(t *testing.T, content string) string {
	t.Helper()

	marshaled, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("marshaling content: %v", err)
	}

	return `{"type":"user","message":{"role":"user","content":` + string(marshaled) + `}}`
}
