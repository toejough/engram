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

	// Verify Stop hook structure
	stopHooks, ok := hooks["Stop"].([]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(stopHooks).To(HaveLen(1))

	stopHook, ok := stopHooks[0].(map[string]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(stopHook["type"]).To(Equal("command"))
	g.Expect(stopHook["command"]).To(ContainSubstring("projctl memory extract-session"))
	g.Expect(stopHook["command"]).To(ContainSubstring("--transcript $TRANSCRIPT_PATH"))
	g.Expect(stopHook["command"]).To(ContainSubstring("&")) // async

	// Verify PreCompact hook structure
	preCompactHooks, ok := hooks["PreCompact"].([]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(preCompactHooks).To(HaveLen(1))

	preCompactHook, ok := preCompactHooks[0].(map[string]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(preCompactHook["type"]).To(Equal("command"))
	g.Expect(preCompactHook["command"]).To(ContainSubstring("projctl memory extract-session"))
	g.Expect(preCompactHook["command"]).To(ContainSubstring("--transcript $TRANSCRIPT_PATH"))
	g.Expect(preCompactHook["command"]).To(ContainSubstring("&")) // async

	// Verify SessionStart hook structure
	sessionStartHooks, ok := hooks["SessionStart"].([]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(sessionStartHooks).To(HaveLen(1))

	sessionStartHook, ok := sessionStartHooks[0].(map[string]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(sessionStartHook["type"]).To(Equal("command"))
	g.Expect(sessionStartHook["command"]).To(Equal("projctl memory context-inject"))
	g.Expect(sessionStartHook["command"]).ToNot(ContainSubstring("&")) // sync
}

// TestHooksInstall_MergesWithExistingHooks tests that install preserves existing hooks.
func TestHooksInstall_MergesWithExistingHooks(t *testing.T) {
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

	// Verify permissions preserved
	g.Expect(settings).To(HaveKey("permissions"))
}

// TestHooksInstall_OverwritesExistingProjectHooks tests that install replaces existing projctl hooks.
func TestHooksInstall_OverwritesExistingProjectHooks(t *testing.T) {
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

	stopHooks, ok := hooks["Stop"].([]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(stopHooks).To(HaveLen(1))

	stopHook, ok := stopHooks[0].(map[string]any)
	g.Expect(ok).To(BeTrue())
	// Should have new format
	g.Expect(stopHook["command"]).To(ContainSubstring("--transcript $TRANSCRIPT_PATH &"))
	g.Expect(stopHook["command"]).ToNot(ContainSubstring("--old-format"))
}

// TestHooksShow_DisplaysCurrentConfig tests that show returns current hook configuration.
func TestHooksShow_DisplaysCurrentConfig(t *testing.T) {
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
					"command": "projctl memory context-inject",
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
}

// TestHooksShow_ReturnsEmptyWhenNoHooks tests that show handles missing hooks gracefully.
func TestHooksShow_ReturnsEmptyWhenNoHooks(t *testing.T) {
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
