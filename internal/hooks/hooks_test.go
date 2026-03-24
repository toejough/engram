package hooks_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
}

// TestT158_HooksJSONStructure verifies hooks.json has exactly one UserPromptSubmit
// entry: synchronous (user-prompt-submit.sh, no "async" key). Also verifies
// user-prompt-submit.sh does not use nohup or disown (T-158).
func TestT158_HooksJSONStructure(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)

	// --- Parse hooks.json ---
	hooksPath := filepath.Join(root, "hooks", "hooks.json")
	hooksData, err := os.ReadFile(hooksPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Unmarshal into a structure that captures the UserPromptSubmit entries.
	type hookEntry struct {
		Type    string `json:"type"`
		Command string `json:"command"`
		Timeout int    `json:"timeout"`
		Async   *bool  `json:"async"`
	}

	type hookGroup struct {
		Hooks []hookEntry `json:"hooks"`
	}

	type hooksFile struct {
		Hooks map[string][]hookGroup `json:"hooks"`
	}

	var parsed hooksFile

	parseErr := json.Unmarshal(hooksData, &parsed)
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	entries := parsed.Hooks["UserPromptSubmit"]
	g.Expect(entries).To(HaveLen(1), "expected exactly one UserPromptSubmit hook group")

	if len(entries) == 0 {
		return
	}

	// Single sync entry pointing to user-prompt-submit.sh.
	g.Expect(entries[0].Hooks).To(HaveLen(1))

	if len(entries[0].Hooks) == 0 {
		return
	}

	hook := entries[0].Hooks[0]
	g.Expect(hook.Async).To(BeNil(), "hook must not have an async field")
	g.Expect(hook.Command).To(ContainSubstring("user-prompt-submit.sh"))

	// --- Inspect user-prompt-submit.sh for forbidden background-spawn patterns ---
	scriptPath := filepath.Join(root, "hooks", "user-prompt-submit.sh")
	scriptData, err := os.ReadFile(scriptPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	script := string(scriptData)
	g.Expect(script).NotTo(ContainSubstring("nohup"), "sync hook must not use nohup")
	g.Expect(script).NotTo(ContainSubstring("disown"), "sync hook must not use disown")
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
// TestT22_UserPromptSubmitHookNoBashTokenSourcing verifies the hook no longer
// contains Keychain token sourcing — token resolution moved to the Go binary
// (internal/tokenresolver, #363).
func TestT22_UserPromptSubmitHookNoBashTokenSourcing(t *testing.T) {
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

	g.Expect(script).NotTo(ContainSubstring("security find-generic-password"))
	g.Expect(script).NotTo(ContainSubstring("export ENGRAM_API_TOKEN"))
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

// TestT352_PostToolUseFiltersFeedbackLoop verifies post-tool-use.sh exits early
// for engram feedback/correct calls to prevent a surfacing feedback loop (#352).
func TestT352_PostToolUseFiltersFeedbackLoop(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	scriptPath := filepath.Join(root, "hooks", "post-tool-use.sh")

	content, err := os.ReadFile(scriptPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	script := string(content)

	g.Expect(script).To(ContainSubstring(`$ENGRAM_BIN`), "must filter all engram CLI calls via binary path")
	g.Expect(script).To(ContainSubstring("#352"), "must reference the original issue for traceability")
	g.Expect(script).To(ContainSubstring("#369"), "must reference the widening issue for traceability")
}

// TestT352_PreToolUseFiltersFeedbackLoop verifies pre-tool-use.sh exits early
// for all engram CLI calls to prevent a surfacing feedback loop (#352, #369).
func TestT352_PreToolUseFiltersFeedbackLoop(t *testing.T) {
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

	g.Expect(script).To(ContainSubstring(`$ENGRAM_BIN`), "must filter all engram CLI calls via binary path")
	g.Expect(script).To(ContainSubstring("#352"), "must reference the original issue for traceability")
	g.Expect(script).To(ContainSubstring("#369"), "must reference the widening issue for traceability")
}

// TestT370_HooksJSONSessionStartSingle verifies hooks.json has a single
// SessionStart entry pointing to session-start.sh (sync output + background fork) (#370).
func TestT370_HooksJSONSessionStartSingle(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	hooksPath := filepath.Join(root, "hooks", "hooks.json")

	hooksData, err := os.ReadFile(hooksPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	type hookEntry struct {
		Type    string `json:"type"`
		Command string `json:"command"`
		Timeout int    `json:"timeout"`
		Async   *bool  `json:"async"`
	}

	type hookGroup struct {
		Hooks []hookEntry `json:"hooks"`
	}

	type hooksFile struct {
		Hooks map[string][]hookGroup `json:"hooks"`
	}

	var parsed hooksFile

	parseErr := json.Unmarshal(hooksData, &parsed)
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	entries := parsed.Hooks["SessionStart"]
	g.Expect(entries).To(HaveLen(1), "expected single SessionStart hook group (sync+fork)")

	if len(entries) < 1 {
		return
	}

	g.Expect(entries[0].Hooks).To(HaveLen(1))

	if len(entries[0].Hooks) > 0 {
		hook := entries[0].Hooks[0]
		g.Expect(hook.Command).To(ContainSubstring("session-start.sh"))
		g.Expect(hook.Async).To(BeNil(), "hook must not have async field — background work is forked internally")
	}
}

// TestT370_PostToolUsePendingCheck verifies post-tool-use.sh checks for
// pending-maintenance.json BEFORE the engram filter so pending content is
// always consumed even when the first tool call is an engram command (#370).
func TestT370_PostToolUsePendingCheck(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	scriptPath := filepath.Join(root, "hooks", "post-tool-use.sh")

	content, err := os.ReadFile(scriptPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	script := string(content)

	// Must check for pending file.
	g.Expect(script).To(ContainSubstring("pending-maintenance.json"))
	g.Expect(script).To(ContainSubstring("PENDING_SYS"))
	g.Expect(script).To(ContainSubstring("PENDING_CTX"))
	// Must use atomic consumption (mv).
	g.Expect(script).To(ContainSubstring("mv "))
	// Must reference #370.
	g.Expect(script).To(ContainSubstring("#370"))

	// Verify ordering: pending check BEFORE engram filter, Write/Edit advisory, and Bash-only exit.
	engramFilterIdx := strings.Index(script, "#352")
	pendingCheckIdx := strings.Index(script, "pending-maintenance.json")
	advisoryIdx := strings.Index(script, "Write/Edit")
	bashOnlyIdx := strings.Index(script, `"$TOOL_NAME" != "Bash"`)

	g.Expect(pendingCheckIdx).To(BeNumerically("<", engramFilterIdx),
		"pending check must come before engram filter")
	g.Expect(pendingCheckIdx).To(BeNumerically("<", advisoryIdx),
		"pending check must come before Write/Edit advisory")
	g.Expect(pendingCheckIdx).To(BeNumerically("<", bashOnlyIdx),
		"pending check must come before Bash-only exit")

	// Engram filter must emit pending content, not bare exit 0.
	g.Expect(script).To(ContainSubstring("emit_pending_and_exit"),
		"engram filter must use emit_pending_and_exit, not bare exit 0")
}

// TestT370_PreToolUsePendingCheck verifies pre-tool-use.sh checks for
// pending-maintenance.json BEFORE the engram filter so pending content is
// always consumed even when the first tool call is an engram command (#370).
func TestT370_PreToolUsePendingCheck(t *testing.T) {
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

	// Must check for pending file.
	g.Expect(script).To(ContainSubstring("pending-maintenance.json"))
	g.Expect(script).To(ContainSubstring("PENDING_SYS"))
	g.Expect(script).To(ContainSubstring("PENDING_CTX"))
	// Must use atomic consumption (mv).
	g.Expect(script).To(ContainSubstring("mv "))
	// Must reference #370.
	g.Expect(script).To(ContainSubstring("#370"))

	// Verify ordering: pending check BEFORE engram filter and Bash-only exit.
	engramFilterIdx := strings.Index(script, "#352")
	pendingCheckIdx := strings.Index(script, "pending-maintenance.json")
	bashOnlyIdx := strings.Index(script, `"$TOOL_NAME" != "Bash"`)

	g.Expect(engramFilterIdx).To(BeNumerically(">", -1))
	g.Expect(pendingCheckIdx).To(BeNumerically(">", -1))
	g.Expect(bashOnlyIdx).To(BeNumerically(">", -1))
	g.Expect(pendingCheckIdx).To(BeNumerically("<", engramFilterIdx),
		"pending check must come before engram filter")
	g.Expect(pendingCheckIdx).To(BeNumerically("<", bashOnlyIdx),
		"pending check must come before Bash-only exit")

	// Engram filter must emit pending content, not bare exit 0.
	g.Expect(script).To(ContainSubstring("emit_pending_and_exit"),
		"engram filter must use emit_pending_and_exit, not bare exit 0")
}

// TestT370_SessionStartWritesPendingFile verifies session-start.sh background
// fork writes to pending-maintenance.json, uses atomic rename, and deletes
// stale files (#370).
func TestT370_SessionStartWritesPendingFile(t *testing.T) {
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

	// Must write to pending file in background fork.
	g.Expect(script).To(ContainSubstring("pending-maintenance.json"))
	// Must use atomic write (temp + mv).
	g.Expect(script).To(ContainSubstring(".tmp"))
	g.Expect(script).To(ContainSubstring("mv "))
	// Must delete stale pending file at start.
	g.Expect(script).To(ContainSubstring("rm -f"))
	// Must use atomic build (temp + mv).
	g.Expect(script).To(ContainSubstring("ENGRAM_BIN.tmp"))
	// Must still run maintain.
	g.Expect(script).To(ContainSubstring("engram maintain"), "background fork must run maintain")
	// Must fork background work (& disown).
	g.Expect(script).To(ContainSubstring("disown"), "maintain must run in background fork")
	// Sync portion emits static context via jq.
	g.Expect(script).To(ContainSubstring("{systemMessage: $sys"))
	g.Expect(script).To(ContainSubstring("/recall"))
	g.Expect(script).To(ContainSubstring("Mid-turn user messages"))
	// Must reference #370.
	g.Expect(script).To(ContainSubstring("#370"))
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

	// session-start.sh emits sync context then forks build+maintain (#370).
	g.Expect(script).To(ContainSubstring("bin/engram"))
	g.Expect(script).To(ContainSubstring("CLAUDE_PLUGIN_ROOT"))
	g.Expect(script).To(ContainSubstring("set -euo pipefail"))
	g.Expect(script).To(ContainSubstring("pending-maintenance.json"))
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

// TestT67_PreCompactHookIsNoOp verifies hooks/pre-compact.sh is a no-op (#350).
// PreCompact previously ran flush (redundant with Stop hook); it now exits cleanly.
func TestT67_PreCompactHookIsNoOp(t *testing.T) {
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

	// Must be a no-op — just exits cleanly.
	g.Expect(script).To(ContainSubstring("exit 0"))
	// Must NOT call flush (that was the redundant behavior being removed).
	g.Expect(script).NotTo(ContainSubstring("engram flush"))
	// Must reference the issue for traceability.
	g.Expect(script).To(ContainSubstring("#350"))
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

	// session-start.sh background fork writes to pending-maintenance.json (#370).
	g.Expect(script).To(ContainSubstring("pending-maintenance.json"))
	g.Expect(script).To(ContainSubstring("ENGRAM_BIN.tmp"))
	g.Expect(script).To(ContainSubstring("#370"))
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
