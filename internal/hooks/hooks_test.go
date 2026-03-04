package hooks_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/hooks"
)

func TestT42_StopHookInvokesExtractAndCatchup(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// When hooks.StopScript() is called
	script := hooks.StopScript()
	// Then returned string contains "engram extract", "engram catchup", transcript var, and strict mode
	g.Expect(script).To(ContainSubstring("extract"))
	g.Expect(script).To(ContainSubstring("catchup"))
	g.Expect(script).To(ContainSubstring("bin/engram"))
	g.Expect(script).To(ContainSubstring("CLAUDE_SESSION_TRANSCRIPT"))
	g.Expect(script).To(ContainSubstring("set -euo pipefail"))
}

func TestT43_UserPromptSubmitHookInvokesCorrect(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// When hooks.UserPromptSubmitScript() is called
	script := hooks.UserPromptSubmitScript()
	// Then returned string contains "engram correct", user message var, strict mode, and Keychain lookup
	g.Expect(script).To(ContainSubstring("correct"))
	g.Expect(script).To(ContainSubstring("bin/engram"))
	g.Expect(script).To(ContainSubstring("CLAUDE_USER_MESSAGE"))
	g.Expect(script).To(ContainSubstring("set -euo pipefail"))
	g.Expect(script).To(ContainSubstring("ENGRAM_API_TOKEN"))
	g.Expect(script).To(ContainSubstring("security find-generic-password"))
}

func TestT55_SessionStartHookInvokesSurface(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// When test calls hooks.SessionStartScript()
	script := hooks.SessionStartScript()
	// Then returned string contains "surface", "--hook session-start", "CLAUDE_PROJECT_DIR", strict mode, and binary path
	g.Expect(script).To(ContainSubstring("surface"))
	g.Expect(script).To(ContainSubstring("--hook session-start"))
	g.Expect(script).To(ContainSubstring("CLAUDE_PROJECT_DIR"))
	g.Expect(script).To(ContainSubstring("set -euo pipefail"))
	g.Expect(script).To(ContainSubstring("bin/engram"))
}

func TestT56_UserPromptSubmitHookInvokesCorrectAndSurface(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// When test calls hooks.UserPromptSubmitScript()
	script := hooks.UserPromptSubmitScript()
	// Then returned string contains "correct", "surface", "--hook user-prompt", "CLAUDE_USER_MESSAGE"
	g.Expect(script).To(ContainSubstring("correct"))
	g.Expect(script).To(ContainSubstring("surface"))
	g.Expect(script).To(ContainSubstring("--hook user-prompt"))
	g.Expect(script).To(ContainSubstring("CLAUDE_USER_MESSAGE"))
	// And index of "correct" < index of "surface" (correction first per DES-4)
	g.Expect(strings.Index(script, "correct")).
		To(BeNumerically("<", strings.Index(script, "surface")))
}

func TestT57_PreToolUseHookInvokesSurface(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// When test calls hooks.PreToolUseScript()
	script := hooks.PreToolUseScript()
	// Then returned string contains "surface", "--hook pre-tool-use", "CLAUDE_TOOL_INPUT", strict mode, and binary path
	g.Expect(script).To(ContainSubstring("surface"))
	g.Expect(script).To(ContainSubstring("--hook pre-tool-use"))
	g.Expect(script).To(ContainSubstring("CLAUDE_TOOL_INPUT"))
	g.Expect(script).To(ContainSubstring("set -euo pipefail"))
	g.Expect(script).To(ContainSubstring("bin/engram"))
}
