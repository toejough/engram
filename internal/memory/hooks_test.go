package memory_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/memory"
)

// TestHooksInstall_CreatesNewFile tests that install creates settings.json if it doesn't exist.
func TestHooksInstall_CreatesNewFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Setup
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Execute
	err := memory.InstallHooks(memory.InstallHooksOpts{
		SettingsPath: settingsPath,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Verify file was created
	g.Expect(settingsPath).To(BeAnExistingFile())

	// Verify content
	data, err := os.ReadFile(settingsPath)
	g.Expect(err).ToNot(HaveOccurred())

	var settings map[string]any
	err = json.Unmarshal(data, &settings)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify hooks were installed
	hooks, ok := settings["hooks"].(map[string]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(hooks).To(HaveKey("Stop"))
	g.Expect(hooks).To(HaveKey("PreCompact"))
	g.Expect(hooks).To(HaveKey("SessionStart"))

	// Verify Stop hook structure (new format: { hooks: [...] })
	stopEntries, ok := hooks["Stop"].([]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(stopEntries).To(HaveLen(1))

	stopEntry, ok := stopEntries[0].(map[string]any)
	g.Expect(ok).To(BeTrue())
	stopHooksList, ok := stopEntry["hooks"].([]any)
	g.Expect(ok).To(BeTrue(), "Stop entry should have 'hooks' array")
	g.Expect(stopHooksList).To(HaveLen(1))

	stopHook, ok := stopHooksList[0].(map[string]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(stopHook["type"]).To(Equal("command"))
	g.Expect(stopHook["command"]).To(ContainSubstring("projctl memory extract-session"))
	g.Expect(stopHook["command"]).To(ContainSubstring("--transcript $TRANSCRIPT_PATH"))
	g.Expect(stopHook["command"]).To(ContainSubstring("&")) // async

	// Verify PreCompact hook structure
	preCompactEntries, ok := hooks["PreCompact"].([]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(preCompactEntries).To(HaveLen(1))

	preCompactEntry, ok := preCompactEntries[0].(map[string]any)
	g.Expect(ok).To(BeTrue())
	preCompactHooksList, ok := preCompactEntry["hooks"].([]any)
	g.Expect(ok).To(BeTrue(), "PreCompact entry should have 'hooks' array")
	g.Expect(preCompactHooksList).To(HaveLen(1))

	preCompactHook, ok := preCompactHooksList[0].(map[string]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(preCompactHook["type"]).To(Equal("command"))
	g.Expect(preCompactHook["command"]).To(ContainSubstring("projctl memory extract-session"))
	g.Expect(preCompactHook["command"]).To(ContainSubstring("--transcript $TRANSCRIPT_PATH"))
	g.Expect(preCompactHook["command"]).To(ContainSubstring("&")) // async

	// Verify SessionStart hook structure
	sessionStartEntries, ok := hooks["SessionStart"].([]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(sessionStartEntries).To(HaveLen(1))

	sessionStartEntry, ok := sessionStartEntries[0].(map[string]any)
	g.Expect(ok).To(BeTrue())
	sessionStartHooksList, ok := sessionStartEntry["hooks"].([]any)
	g.Expect(ok).To(BeTrue(), "SessionStart entry should have 'hooks' array")
	g.Expect(sessionStartHooksList).To(HaveLen(1))

	sessionStartHook, ok := sessionStartHooksList[0].(map[string]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(sessionStartHook["type"]).To(Equal("command"))
	g.Expect(sessionStartHook["command"]).To(ContainSubstring("projctl memory query"))
	g.Expect(sessionStartHook["command"]).To(ContainSubstring("--primacy"))
	g.Expect(sessionStartHook["command"]).To(ContainSubstring("--stdin-project"))
	g.Expect(sessionStartHook["command"]).ToNot(ContainSubstring("&")) // sync

	// Verify UserPromptSubmit hook structure
	g.Expect(hooks).To(HaveKey("UserPromptSubmit"))
	userPromptEntries, ok := hooks["UserPromptSubmit"].([]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(userPromptEntries).To(HaveLen(1))

	userPromptEntry, ok := userPromptEntries[0].(map[string]any)
	g.Expect(ok).To(BeTrue())
	userPromptHooksList, ok := userPromptEntry["hooks"].([]any)
	g.Expect(ok).To(BeTrue(), "UserPromptSubmit entry should have 'hooks' array")
	g.Expect(userPromptHooksList).To(HaveLen(1))

	userPromptHook, ok := userPromptHooksList[0].(map[string]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(userPromptHook["type"]).To(Equal("command"))
	g.Expect(userPromptHook["command"]).To(ContainSubstring("projctl memory query"))
	g.Expect(userPromptHook["command"]).To(ContainSubstring("--stdin-prompt"))

	// Verify PreToolUse hook structure
	g.Expect(hooks).To(HaveKey("PreToolUse"))
	preToolUseEntries, ok := hooks["PreToolUse"].([]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(preToolUseEntries).To(HaveLen(1))

	preToolUseEntry, ok := preToolUseEntries[0].(map[string]any)
	g.Expect(ok).To(BeTrue())
	preToolUseHooksList, ok := preToolUseEntry["hooks"].([]any)
	g.Expect(ok).To(BeTrue(), "PreToolUse entry should have 'hooks' array")
	g.Expect(preToolUseHooksList).To(HaveLen(1))

	preToolUseHookCmd, ok := preToolUseHooksList[0].(map[string]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(preToolUseHookCmd["type"]).To(Equal("command"))
	g.Expect(preToolUseHookCmd["command"]).To(ContainSubstring("projctl memory query"))
	g.Expect(preToolUseHookCmd["command"]).To(ContainSubstring("--stdin-tool"))
	g.Expect(preToolUseHookCmd["command"]).To(ContainSubstring("--min-confidence=50"))
}

// TestHooksInstall_MergesWithExistingHooks tests that install preserves existing hooks.
func TestHooksInstall_MergesWithExistingHooks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Setup
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Create existing settings with other hooks
	existingSettings := map[string]any{
		"permissions": map[string]any{
			"allow": []string{"mcp__pencil"},
		},
		"hooks": map[string]any{
			"PreToolUse": []map[string]any{
				{
					"type":    "command",
					"command": "echo 'existing hook'",
				},
			},
		},
	}

	data, err := json.MarshalIndent(existingSettings, "", "  ")
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(settingsPath, data, 0600)
	g.Expect(err).ToNot(HaveOccurred())

	// Execute
	err = memory.InstallHooks(memory.InstallHooksOpts{
		SettingsPath: settingsPath,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Verify
	data, err = os.ReadFile(settingsPath)
	g.Expect(err).ToNot(HaveOccurred())

	var settings map[string]any
	err = json.Unmarshal(data, &settings)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify existing hooks preserved
	hooks, ok := settings["hooks"].(map[string]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(hooks).To(HaveKey("PreToolUse"))

	// Verify new hooks added
	g.Expect(hooks).To(HaveKey("Stop"))
	g.Expect(hooks).To(HaveKey("PreCompact"))
	g.Expect(hooks).To(HaveKey("SessionStart"))
	g.Expect(hooks).To(HaveKey("UserPromptSubmit"))
	g.Expect(hooks).To(HaveKey("PreToolUse"))

	// Verify permissions preserved
	g.Expect(settings).To(HaveKey("permissions"))
}

// TestHooksInstall_OverwritesExistingProjectHooks tests that install replaces existing projctl hooks.
func TestHooksInstall_OverwritesExistingProjectHooks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Setup
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Create existing settings with old projctl hooks
	existingSettings := map[string]any{
		"hooks": map[string]any{
			"Stop": []map[string]any{
				{
					"type":    "command",
					"command": "projctl memory extract-session --old-format &",
				},
			},
		},
	}

	data, err := json.MarshalIndent(existingSettings, "", "  ")
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(settingsPath, data, 0600)
	g.Expect(err).ToNot(HaveOccurred())

	// Execute
	err = memory.InstallHooks(memory.InstallHooksOpts{
		SettingsPath: settingsPath,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Verify
	data, err = os.ReadFile(settingsPath)
	g.Expect(err).ToNot(HaveOccurred())

	var settings map[string]any
	err = json.Unmarshal(data, &settings)
	g.Expect(err).ToNot(HaveOccurred())

	hooks, ok := settings["hooks"].(map[string]any)
	g.Expect(ok).To(BeTrue())

	stopEntries, ok := hooks["Stop"].([]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(stopEntries).To(HaveLen(1))

	stopEntry, ok := stopEntries[0].(map[string]any)
	g.Expect(ok).To(BeTrue())
	stopHooksList, ok := stopEntry["hooks"].([]any)
	g.Expect(ok).To(BeTrue(), "Should use new nested hooks format")
	g.Expect(stopHooksList).To(HaveLen(1))

	stopHook, ok := stopHooksList[0].(map[string]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(stopHook["command"]).To(ContainSubstring("--transcript $TRANSCRIPT_PATH &"))
	g.Expect(stopHook["command"]).ToNot(ContainSubstring("--old-format"))
}

// TestHooksShow_DisplaysCurrentConfig tests that show returns current hook configuration.
func TestHooksShow_DisplaysCurrentConfig(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Setup
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Create settings with hooks
	settings := map[string]any{
		"hooks": map[string]any{
			"Stop": []map[string]any{
				{
					"type":    "command",
					"command": "projctl memory extract-session --transcript $TRANSCRIPT_PATH &",
				},
			},
			"PreCompact": []map[string]any{
				{
					"type":    "command",
					"command": "projctl memory extract-session --transcript $TRANSCRIPT_PATH &",
				},
			},
			"SessionStart": []map[string]any{
				{
					"type":    "command",
					"command": "projctl memory query --primacy --stdin-project --min-confidence=30 --max-tokens=1000 -n 10 \"recent important learnings\"",
				},
			},
			"UserPromptSubmit": []map[string]any{
				{
					"type":    "command",
					"command": "projctl memory query --primacy --stdin-prompt --min-confidence=30 --max-tokens=2000 -n 10",
				},
			},
			"PreToolUse": []map[string]any{
				{
					"type":    "command",
					"command": "projctl memory query --stdin-tool --min-confidence=50 --max-tokens=1000 -n 5",
				},
			},
		},
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(settingsPath, data, 0600)
	g.Expect(err).ToNot(HaveOccurred())

	// Execute
	result, err := memory.ShowHooks(memory.ShowHooksOpts{
		SettingsPath: settingsPath,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Verify
	g.Expect(result).ToNot(BeEmpty())

	// Verify it's valid JSON
	var output map[string]any
	err = json.Unmarshal([]byte(result), &output)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify it contains our hooks
	g.Expect(output).To(HaveKey("Stop"))
	g.Expect(output).To(HaveKey("PreCompact"))
	g.Expect(output).To(HaveKey("SessionStart"))
	g.Expect(output).To(HaveKey("UserPromptSubmit"))
	g.Expect(output).To(HaveKey("PreToolUse"))
}

// TestHooksShow_ReturnsEmptyWhenNoHooks tests that show handles missing hooks gracefully.
func TestHooksShow_ReturnsEmptyWhenNoHooks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Setup
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Create settings without hooks
	settings := map[string]any{
		"permissions": map[string]any{
			"allow": []string{"mcp__pencil"},
		},
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(settingsPath, data, 0600)
	g.Expect(err).ToNot(HaveOccurred())

	// Execute
	result, err := memory.ShowHooks(memory.ShowHooksOpts{
		SettingsPath: settingsPath,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Verify
	g.Expect(result).To(Equal("{}"))
}

// TestHooksShow_HandlesNonExistentFile tests that show handles missing settings file.
func TestHooksShow_HandlesNonExistentFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Setup
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Execute
	result, err := memory.ShowHooks(memory.ShowHooksOpts{
		SettingsPath: settingsPath,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Verify
	g.Expect(result).To(Equal("{}"))
}
