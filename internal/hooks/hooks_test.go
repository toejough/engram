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
	g.Expect(script).To(ContainSubstring("CLAUDE_USER_MESSAGE"))
	g.Expect(script).To(ContainSubstring("CLAUDE_PLUGIN_ROOT"))
	g.Expect(script).To(ContainSubstring("set -euo pipefail"))
	g.Expect(script).To(ContainSubstring("ENGRAM_API_TOKEN"))
}

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
