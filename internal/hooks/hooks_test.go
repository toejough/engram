package hooks_test

// Tests for ARCH-9: Hook Shell Scripts (pure implementation, no I/O).
// Won't compile yet — RED phase.

import (
	"testing"

	"engram/internal/hooks"
	"github.com/onsi/gomega"
)

// T-42: The Stop hook script invokes both "engram extract" and "engram catchup"
// with the session transcript path.
func TestHookScript_StopInvokesExtractAndCatchup(t *testing.T) {
	g := gomega.NewWithT(t)
	script := hooks.StopScript()

	g.Expect(script).To(gomega.ContainSubstring("engram extract"))
	g.Expect(script).To(gomega.ContainSubstring("engram catchup"))
	g.Expect(script).To(gomega.ContainSubstring("CLAUDE_SESSION_TRANSCRIPT"))
	g.Expect(script).To(gomega.ContainSubstring("set -euo pipefail"))
}

// T-43: The UserPromptSubmit hook script invokes "engram correct" with the user's message.
func TestHookScript_UserPromptSubmitInvokesCorrect(t *testing.T) {
	g := gomega.NewWithT(t)
	script := hooks.UserPromptSubmitScript()

	g.Expect(script).To(gomega.ContainSubstring("engram correct"))
	g.Expect(script).To(gomega.ContainSubstring("CLAUDE_USER_MESSAGE"))
	g.Expect(script).To(gomega.ContainSubstring("set -euo pipefail"))
}
