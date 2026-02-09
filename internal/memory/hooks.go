package memory

import (
	"encoding/json"
	"fmt"
	"os"
)

// InstallHooksOpts contains options for installing Claude Code hooks.
type InstallHooksOpts struct {
	// SettingsPath is the path to Claude Code settings.json
	SettingsPath string
}

// ShowHooksOpts contains options for showing hook configuration.
type ShowHooksOpts struct {
	// SettingsPath is the path to Claude Code settings.json
	SettingsPath string
}

// hookConfig represents a single hook configuration.
type hookConfig struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// InstallHooks installs projctl memory hooks into Claude Code settings.json.
// It creates the file if it doesn't exist, merges with existing settings,
// and replaces any existing projctl memory hooks.
func InstallHooks(opts InstallHooksOpts) error {
	// Define the hooks to install
	stopHook := hookConfig{
		Type:    "command",
		Command: "projctl memory extract-session --transcript $TRANSCRIPT_PATH &",
	}

	preCompactHook := hookConfig{
		Type:    "command",
		Command: "projctl memory extract-session --transcript $TRANSCRIPT_PATH &",
	}

	sessionStartHook := hookConfig{
		Type:    "command",
		Command: "projctl memory context-inject",
	}

	// Read existing settings or create new
	var settings map[string]any
	data, err := os.ReadFile(opts.SettingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create new settings
			settings = make(map[string]any)
		} else {
			return fmt.Errorf("failed to read settings file: %w", err)
		}
	} else {
		// Parse existing settings
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("failed to parse settings file: %w", err)
		}
	}

	// Ensure hooks section exists
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		hooks = make(map[string]any)
		settings["hooks"] = hooks
	}

	// Install/replace hooks
	hooks["Stop"] = []hookConfig{stopHook}
	hooks["PreCompact"] = []hookConfig{preCompactHook}
	hooks["SessionStart"] = []hookConfig{sessionStartHook}

	// Write updated settings
	data, err = json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(opts.SettingsPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	return nil
}

// ShowHooks returns the current hook configuration as formatted JSON.
// Returns "{}" if no hooks are configured or the file doesn't exist.
func ShowHooks(opts ShowHooksOpts) (string, error) {
	// Read settings file
	data, err := os.ReadFile(opts.SettingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "{}", nil
		}
		return "", fmt.Errorf("failed to read settings file: %w", err)
	}

	// Parse settings
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return "", fmt.Errorf("failed to parse settings file: %w", err)
	}

	// Extract hooks
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok || len(hooks) == 0 {
		return "{}", nil
	}

	// Format as JSON
	output, err := json.MarshalIndent(hooks, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal hooks: %w", err)
	}

	return string(output), nil
}
