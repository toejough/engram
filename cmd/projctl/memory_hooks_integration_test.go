//go:build integration

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

// TestMemoryHooksInstall_CreatesNewFile tests that CLI install command works.
func TestMemoryHooksInstall_CreatesNewFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Setup
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Execute
	args := memoryHooksInstallArgs{
		SettingsPath: settingsPath,
	}
	err := memoryHooksInstall(args)
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
	g.Expect(hooks).To(HaveKey("UserPromptSubmit"))
	g.Expect(hooks).To(HaveKey("PreToolUse"))
}

// TestMemoryHooksInstall_DefaultsToHomeDir tests that CLI defaults to ~/.claude/settings.json.
func TestMemoryHooksInstall_DefaultsToHomeDir(t *testing.T) {
	g := NewWithT(t)

	// Setup - create temporary home directory
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		_ = os.Setenv("HOME", originalHome)
	})
	err := os.Setenv("HOME", tmpHome)
	g.Expect(err).ToNot(HaveOccurred())

	// Create .claude directory
	claudeDir := filepath.Join(tmpHome, ".claude")
	err = os.MkdirAll(claudeDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	expectedPath := filepath.Join(claudeDir, "settings.json")

	// Execute with empty settings path (should use default)
	args := memoryHooksInstallArgs{
		SettingsPath: "",
	}
	err = memoryHooksInstall(args)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify file was created at default location
	g.Expect(expectedPath).To(BeAnExistingFile())
}

// TestMemoryHooksShow_DisplaysHooks tests that CLI show command works.
func TestMemoryHooksShow_DisplaysHooks(t *testing.T) {
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
		},
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(settingsPath, data, 0600)
	g.Expect(err).ToNot(HaveOccurred())

	// Execute - capture output by testing the function directly
	args := memoryHooksShowArgs{
		SettingsPath: settingsPath,
	}
	err = memoryHooksShow(args)
	g.Expect(err).ToNot(HaveOccurred())
	// Note: We can't easily capture stdout in unit tests, but we verify no error
}

// TestMemoryHooksShow_DefaultsToHomeDir tests that CLI defaults to ~/.claude/settings.json.
func TestMemoryHooksShow_DefaultsToHomeDir(t *testing.T) {
	g := NewWithT(t)

	// Setup - create temporary home directory
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		_ = os.Setenv("HOME", originalHome)
	})
	err := os.Setenv("HOME", tmpHome)
	g.Expect(err).ToNot(HaveOccurred())

	// Create .claude directory with settings
	claudeDir := filepath.Join(tmpHome, ".claude")
	err = os.MkdirAll(claudeDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	settingsPath := filepath.Join(claudeDir, "settings.json")
	settings := map[string]any{
		"hooks": map[string]any{
			"Stop": []map[string]any{
				{
					"type":    "command",
					"command": "test",
				},
			},
		},
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(settingsPath, data, 0600)
	g.Expect(err).ToNot(HaveOccurred())

	// Execute with empty settings path (should use default)
	args := memoryHooksShowArgs{
		SettingsPath: "",
	}
	err = memoryHooksShow(args)
	g.Expect(err).ToNot(HaveOccurred())
}
