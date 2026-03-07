package hooks_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/gomega"
)

// TestDES3_StaticHookScriptMatchesGenerated verifies the static hook script at
// hooks/user-prompt-submit.sh references the expected commands and variables (DES-3).
func TestDES3_StaticHookScriptMatchesGenerated(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	scriptPath := filepath.Join(root, "hooks", "user-prompt-submit.sh")

	content, err := os.ReadFile(scriptPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	script := string(content)

	// DES-3: hook script invokes engram correct with user message and plugin root paths.
	g.Expect(script).To(ContainSubstring("correct"))
	g.Expect(script).To(ContainSubstring("bin/engram"))
	g.Expect(script).To(ContainSubstring("jq"))
	g.Expect(script).To(ContainSubstring(".prompt"))
	g.Expect(script).To(ContainSubstring(".transcript_path"))
	g.Expect(script).To(ContainSubstring("CLAUDE_PLUGIN_ROOT"))
	g.Expect(script).To(ContainSubstring("set -euo pipefail"))
	g.Expect(script).To(ContainSubstring("ENGRAM_API_TOKEN"))
}

// T-20: Plugin manifest exists
// TestT20_PluginManifestExists verifies .claude-plugin/plugin.json exists with
// the correct name and description.
func TestT20_PluginManifestExists(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	manifestPath := filepath.Join(root, ".claude-plugin", "plugin.json")

	content, err := os.ReadFile(manifestPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	manifest := string(content)

	g.Expect(manifest).To(ContainSubstring(`"name": "engram"`))
	g.Expect(manifest).To(ContainSubstring(`"description"`))
}

// T-21: Hooks JSON has UserPromptSubmit
// TestT21_HooksJSONHasUserPromptSubmit verifies hooks/hooks.json contains a
// UserPromptSubmit hook entry pointing to user-prompt-submit.sh.
func TestT21_HooksJSONHasUserPromptSubmit(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	hooksPath := filepath.Join(root, "hooks", "hooks.json")

	content, err := os.ReadFile(hooksPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	hooksJSON := string(content)

	g.Expect(hooksJSON).To(ContainSubstring("UserPromptSubmit"))
	g.Expect(hooksJSON).To(ContainSubstring("user-prompt-submit.sh"))
}

// T-22: UserPromptSubmit hook script has platform-aware token retrieval
// TestT22_UserPromptSubmitHookCrossPlatformToken verifies the static hook script at
// hooks/user-prompt-submit.sh has platform-aware token retrieval (ARCH-6, DES-3).
func TestT22_UserPromptSubmitHookCrossPlatformToken(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	scriptPath := filepath.Join(root, "hooks", "user-prompt-submit.sh")

	content, err := os.ReadFile(scriptPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	script := string(content)

	// Must check platform before attempting Keychain
	g.Expect(script).To(ContainSubstring("uname"))
	g.Expect(script).To(ContainSubstring("Darwin"))
	// Must still have Keychain lookup for macOS
	g.Expect(script).To(ContainSubstring("security find-generic-password"))
	// Must export token regardless of source
	g.Expect(script).To(ContainSubstring("export ENGRAM_API_TOKEN"))
}

// T-23: bin/ is in .gitignore
// TestT23_BinDirInGitignore verifies that the bin/ directory is gitignored (ARCH-8, REQ-8).
func TestT23_BinDirInGitignore(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	gitignorePath := filepath.Join(root, ".gitignore")

	content, err := os.ReadFile(gitignorePath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(content)).To(ContainSubstring("bin/"))
}

// TestT43_SessionStartHookSurfaces verifies hooks/session-start.sh calls
// engram surface with --mode session-start (T-43).
func TestT43_SessionStartHookSurfaces(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	scriptPath := filepath.Join(root, "hooks", "session-start.sh")

	content, err := os.ReadFile(scriptPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	script := string(content)

	g.Expect(script).To(ContainSubstring("surface"))
	g.Expect(script).To(ContainSubstring("--mode session-start"))
	g.Expect(script).To(ContainSubstring("bin/engram"))
	g.Expect(script).To(ContainSubstring("CLAUDE_PLUGIN_ROOT"))
	g.Expect(script).To(ContainSubstring("set -euo pipefail"))
}

// TestT44_UserPromptSubmitHookSurfaces verifies hooks/user-prompt-submit.sh
// calls both engram correct and engram surface --mode prompt (T-44).
func TestT44_UserPromptSubmitHookSurfaces(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	scriptPath := filepath.Join(root, "hooks", "user-prompt-submit.sh")

	content, err := os.ReadFile(scriptPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	script := string(content)

	g.Expect(script).To(ContainSubstring("correct"))
	g.Expect(script).To(ContainSubstring("surface"))
	g.Expect(script).To(ContainSubstring("--mode prompt"))
}

// TestT45_HooksJSONHasPreToolUse verifies hooks/hooks.json contains a
// PreToolUse hook entry pointing to pre-tool-use.sh (T-45).
func TestT45_HooksJSONHasPreToolUse(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	hooksPath := filepath.Join(root, "hooks", "hooks.json")

	content, err := os.ReadFile(hooksPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	hooksJSON := string(content)

	g.Expect(hooksJSON).To(ContainSubstring("PreToolUse"))
	g.Expect(hooksJSON).To(ContainSubstring("pre-tool-use.sh"))
}

// TestT46_PreToolUseHookReadsSdin verifies hooks/pre-tool-use.sh reads stdin
// JSON and calls engram surface --mode tool (T-46).
func TestT46_PreToolUseHookSurfaces(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	scriptPath := filepath.Join(root, "hooks", "pre-tool-use.sh")

	content, err := os.ReadFile(scriptPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	script := string(content)

	g.Expect(script).To(ContainSubstring("jq"))
	g.Expect(script).To(ContainSubstring(".tool_name"))
	g.Expect(script).To(ContainSubstring(".tool_input"))
	g.Expect(script).To(ContainSubstring("surface"))
	g.Expect(script).To(ContainSubstring("--mode tool"))
	g.Expect(script).To(ContainSubstring("--tool-name"))
	g.Expect(script).To(ContainSubstring("--tool-input"))
}

// TestT65_HooksJSONHasPreCompact verifies hooks/hooks.json contains a
// PreCompact hook entry pointing to pre-compact.sh (T-65).
func TestT65_HooksJSONHasPreCompact(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	hooksPath := filepath.Join(root, "hooks", "hooks.json")

	content, err := os.ReadFile(hooksPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	hooksJSON := string(content)

	g.Expect(hooksJSON).To(ContainSubstring("PreCompact"))
	g.Expect(hooksJSON).To(ContainSubstring("pre-compact.sh"))
}

// TestT66_HooksJSONHasSessionEnd verifies hooks/hooks.json contains a
// SessionEnd hook entry pointing to session-end.sh (T-66).
func TestT66_HooksJSONHasSessionEnd(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	hooksPath := filepath.Join(root, "hooks", "hooks.json")

	content, err := os.ReadFile(hooksPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	hooksJSON := string(content)

	g.Expect(hooksJSON).To(ContainSubstring("SessionEnd"))
	g.Expect(hooksJSON).To(ContainSubstring("session-end.sh"))
}

// TestT67_PreCompactHookReadsTranscript verifies hooks/pre-compact.sh reads
// transcript from stdin JSON, retrieves OAuth token, and calls engram learn (T-67).
func TestT67_PreCompactHookReadsTranscript(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	scriptPath := filepath.Join(root, "hooks", "pre-compact.sh")

	content, err := os.ReadFile(scriptPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	script := string(content)

	// Must read stdin JSON for transcript
	g.Expect(script).To(ContainSubstring("jq"))
	g.Expect(script).To(ContainSubstring("set -euo pipefail"))
	// Must use transcript_path, not inline transcript content
	g.Expect(script).To(ContainSubstring(".transcript_path"))
	g.Expect(script).NotTo(ContainSubstring(".transcript //"))
	// Must warn when no transcript available
	g.Expect(script).To(ContainSubstring("no transcript available"))
	// Must retrieve OAuth token (DES-3 pattern)
	g.Expect(script).To(ContainSubstring("uname"))
	g.Expect(script).To(ContainSubstring("Darwin"))
	g.Expect(script).To(ContainSubstring("security find-generic-password"))
	g.Expect(script).To(ContainSubstring("export ENGRAM_API_TOKEN"))
	// Must call engram learn with --data-dir
	g.Expect(script).To(ContainSubstring("learn"))
	g.Expect(script).To(ContainSubstring("--data-dir"))
	g.Expect(script).To(ContainSubstring("bin/engram"))
}

// TestT68_SessionEndHookReadsTranscript verifies hooks/session-end.sh reads
// transcript from stdin JSON, retrieves OAuth token, and calls engram learn (T-68).
func TestT68_SessionEndHookReadsTranscript(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	scriptPath := filepath.Join(root, "hooks", "session-end.sh")

	content, err := os.ReadFile(scriptPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	script := string(content)

	// Must read stdin JSON for transcript
	g.Expect(script).To(ContainSubstring("jq"))
	g.Expect(script).To(ContainSubstring("set -euo pipefail"))
	// Must use transcript_path, not inline transcript content
	g.Expect(script).To(ContainSubstring(".transcript_path"))
	g.Expect(script).NotTo(ContainSubstring(".transcript //"))
	// Must warn when no transcript available
	g.Expect(script).To(ContainSubstring("no transcript available"))
	// Must retrieve OAuth token (DES-3 pattern)
	g.Expect(script).To(ContainSubstring("uname"))
	g.Expect(script).To(ContainSubstring("Darwin"))
	g.Expect(script).To(ContainSubstring("security find-generic-password"))
	g.Expect(script).To(ContainSubstring("export ENGRAM_API_TOKEN"))
	// Must call engram learn with --data-dir
	g.Expect(script).To(ContainSubstring("learn"))
	g.Expect(script).To(ContainSubstring("--data-dir"))
	g.Expect(script).To(ContainSubstring("bin/engram"))
}

// TestT98_UserPromptSubmitCreationInSystemMessage verifies hooks/user-prompt-submit.sh
// places creation output from engram correct into systemMessage (not additionalContext) (T-98).
func TestT98_UserPromptSubmitCreationInSystemMessage(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	scriptPath := filepath.Join(root, "hooks", "user-prompt-submit.sh")

	content, err := os.ReadFile(scriptPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	script := string(content)

	// When surface + correct both exist: systemMessage must contain surface summary AND correct output.
	// The jq expression must build systemMessage from surface summary + correct output.
	g.Expect(script).To(ContainSubstring("systemMessage: (.summary + "))
	// Correct output must NOT be put into additionalContext alone — it goes in systemMessage.
	g.Expect(script).NotTo(ContainSubstring("additionalContext: ($correct +"))
	// When only correct output (no surface): must emit JSON with systemMessage, not bare plain text.
	g.Expect(script).To(ContainSubstring(`{systemMessage: $correct`))
}

// TestT99_SessionStartCreationInSystemMessage verifies hooks/session-start.sh
// places both creation report summary and recency summary into systemMessage (T-99).
func TestT99_SessionStartCreationInSystemMessage(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	scriptPath := filepath.Join(root, "hooks", "session-start.sh")

	content, err := os.ReadFile(scriptPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	script := string(content)

	// Must call surface with --mode session-start --format json.
	g.Expect(script).To(ContainSubstring("--mode session-start"))
	g.Expect(script).To(ContainSubstring("--format json"))
	// Must reshape so summary goes to systemMessage.
	g.Expect(script).To(ContainSubstring("systemMessage: .summary"))
	g.Expect(script).To(ContainSubstring("additionalContext: .context"))
}

// repoRoot returns the engram repository root by walking up from the test file.
func repoRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine test file path")
	}

	// Walk up from internal/hooks/hooks_test.go -> repo root
	return filepath.Dir(filepath.Dir(filepath.Dir(filename)))
}
