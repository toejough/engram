package context_test

import (
	"encoding/json"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	sessionctx "engram/internal/context"
)

// TestStripWithConfig_DropsSystemReminders verifies system-reminders are dropped in both modes.
func TestStripWithConfig_DropsSystemReminders(t *testing.T) {
	t.Parallel()

	sysReminderLine := `{"type":"user","message":{"role":"user",` +
		`"content":[{"type":"text","text":"<system-reminder>hook noise</system-reminder>"}]}}`
	normalLine := `{"type":"user","message":{"role":"user","content":"real question"}}`

	lines := []string{sysReminderLine, normalLine}

	for _, keepTools := range []bool{false, true} {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			g := NewGomegaWithT(t)
			cfg := sessionctx.StripConfig{KeepToolCalls: keepTools}
			result := sessionctx.StripWithConfig(lines, cfg)

			g.Expect(result).To(HaveLen(1))
			g.Expect(result[0]).To(Equal("USER: real question"))
		})
	}
}

// TestStripWithConfig_DropsUnknownBlockTypes verifies that unknown content block types are ignored.
func TestStripWithConfig_DropsUnknownBlockTypes(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// A line with only an unknown block type should produce no output.
	unknownBlockLine := `{"type":"assistant","message":{"role":"assistant","content":[` +
		`{"type":"image","source":{"type":"base64","data":"abc"}}` +
		`]}}`

	cfg := sessionctx.StripConfig{KeepToolCalls: true}
	result := sessionctx.StripWithConfig([]string{unknownBlockLine}, cfg)

	g.Expect(result).To(BeEmpty())
}

// TestStripWithConfig_ErrorToolResult verifies is_error=true produces [error] label.
func TestStripWithConfig_ErrorToolResult(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	toolResultLine := buildToolResultLine("command not found: foo", true)

	cfg := sessionctx.StripConfig{KeepToolCalls: true}
	result := sessionctx.StripWithConfig([]string{toolResultLine}, cfg)

	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]).To(HavePrefix("TOOL_RESULT [error]:"))
	g.Expect(result[0]).To(ContainSubstring("command not found: foo"))
}

// TestStripWithConfig_HandlesStringContent verifies content as plain string (not array) works in both modes.
func TestStripWithConfig_HandlesStringContent(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lines := []string{
		`{"type":"user","message":{"role":"user","content":"plain string message"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":"plain assistant reply"}}`,
	}

	for _, keepTools := range []bool{false, true} {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			g := NewGomegaWithT(t)
			cfg := sessionctx.StripConfig{KeepToolCalls: keepTools}
			result := sessionctx.StripWithConfig(lines, cfg)

			g.Expect(result).To(HaveLen(2))
			g.Expect(result[0]).To(Equal("USER: plain string message"))
			g.Expect(result[1]).To(Equal("ASSISTANT: plain assistant reply"))
		})
	}

	// Also verify zero-value config matches Strip() behavior.
	result := sessionctx.StripWithConfig(lines, sessionctx.StripConfig{})
	g.Expect(result).To(Equal(sessionctx.Strip(lines)))
}

// TestStripWithConfig_KeepsToolCalls verifies SBIA mode preserves tool_use and tool_result blocks.
func TestStripWithConfig_KeepsToolCalls(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	toolUseLine := buildToolUseLine("Let me run tests", "Bash", map[string]string{"command": "go test ./..."})
	toolResultLine := buildToolResultLine("ok engram/internal/context", false)

	lines := []string{
		`{"type":"user","message":{"role":"user","content":"run the tests"}}`,
		toolUseLine,
		toolResultLine,
	}

	cfg := sessionctx.StripConfig{KeepToolCalls: true}
	result := sessionctx.StripWithConfig(lines, cfg)

	// Expect: USER line, ASSISTANT text + TOOL_USE, TOOL_RESULT.
	g.Expect(result).To(HaveLen(4))
	g.Expect(result[0]).To(Equal("USER: run the tests"))
	g.Expect(result[1]).To(Equal("ASSISTANT: Let me run tests"))
	g.Expect(result[2]).To(ContainSubstring("TOOL_USE [Bash]:"))
	g.Expect(result[2]).To(ContainSubstring("go test ./..."))
	g.Expect(result[3]).To(ContainSubstring("TOOL_RESULT [ok]:"))
	g.Expect(result[3]).To(ContainSubstring("ok engram/internal/context"))
}

// TestStripWithConfig_RecallMode verifies default config (KeepToolCalls=false) drops tools like Strip().
func TestStripWithConfig_RecallMode(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	toolUseLine := buildToolUseLine("Let me check", "Read", map[string]string{"path": "/foo.go"})
	toolResultLine := buildToolResultLine("package main", false)

	lines := []string{
		`{"type":"user","message":{"role":"user","content":"check the code"}}`,
		toolUseLine,
		toolResultLine,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Done."}]}}`,
	}

	cfg := sessionctx.StripConfig{KeepToolCalls: false}
	result := sessionctx.StripWithConfig(lines, cfg)

	stripResult := sessionctx.Strip(lines)

	// StripWithConfig with default config should match Strip() exactly.
	g.Expect(result).To(Equal(stripResult))
	// Strip keeps the text block from the mixed assistant message, drops tool blocks.
	g.Expect(result).To(HaveLen(3))
	g.Expect(result[0]).To(Equal("USER: check the code"))
	g.Expect(result[1]).To(Equal("ASSISTANT: Let me check"))
	g.Expect(result[2]).To(Equal("ASSISTANT: Done."))
}

// TestStripWithConfig_TruncatesLongToolArgs verifies args over ToolArgsTruncate are truncated.
func TestStripWithConfig_TruncatesLongToolArgs(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Use spaces to avoid base64 replacement (base64 alphabet excludes spaces).
	longCmd := "echo " + strings.Repeat("hello world ", 30)
	toolUseLine := buildToolUseLine("Doing something", "Bash", map[string]string{"command": longCmd})

	const argLimit = 50

	cfg := sessionctx.StripConfig{KeepToolCalls: true, ToolArgsTruncate: argLimit}
	result := sessionctx.StripWithConfig([]string{toolUseLine}, cfg)

	// Expect two lines: text line and tool_use line.
	g.Expect(result).To(HaveLen(2))

	toolLine := result[1]
	g.Expect(toolLine).To(HavePrefix("TOOL_USE [Bash]:"))

	// The args portion (after the prefix) should not exceed the truncation limit.
	const toolUsePrefix = "TOOL_USE [Bash]: "

	args := toolLine[len(toolUsePrefix):]
	g.Expect(len(args)).To(BeNumerically("<=", argLimit+len("[truncated]")))
	g.Expect(toolLine).To(ContainSubstring("[truncated]"))
}

// TestStripWithConfig_TruncatesLongToolResult verifies results over ToolResultTruncate are truncated.
func TestStripWithConfig_TruncatesLongToolResult(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Use spaces to avoid base64 replacement (base64 alphabet excludes spaces).
	longResult := "output: " + strings.Repeat("line output ", 30)
	toolResultLine := buildToolResultLine(longResult, false)

	const resultLimit = 50

	cfg := sessionctx.StripConfig{KeepToolCalls: true, ToolResultTruncate: resultLimit}
	result := sessionctx.StripWithConfig([]string{toolResultLine}, cfg)

	g.Expect(result).To(HaveLen(1))

	toolLine := result[0]
	g.Expect(toolLine).To(HavePrefix("TOOL_RESULT [ok]:"))

	const toolResultPrefix = "TOOL_RESULT [ok]: "

	content := toolLine[len(toolResultPrefix):]
	g.Expect(len(content)).To(BeNumerically("<=", resultLimit+len("[truncated]")))
	g.Expect(toolLine).To(ContainSubstring("[truncated]"))
}

// buildToolResultLine builds a JSONL user line with a tool_result block.
func buildToolResultLine(content string, isError bool) string {
	isErrorStr := "false"
	if isError {
		isErrorStr = "true"
	}

	return `{"type":"user","message":{"role":"user","content":[` +
		`{"type":"tool_result","tool_use_id":"t1","content":"` + content + `","is_error":` + isErrorStr + `}` +
		`]}}`
}

// buildToolUseLine builds a JSONL assistant line with a text block and a tool_use block.
func buildToolUseLine(text, toolName string, input map[string]string) string {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		panic("buildToolUseLine: marshal failed: " + err.Error())
	}

	return `{"type":"assistant","message":{"role":"assistant","content":[` +
		`{"type":"text","text":"` + text + `"},` +
		`{"type":"tool_use","id":"t1","name":"` + toolName + `","input":` + string(inputJSON) + `}` +
		`]}}`
}
