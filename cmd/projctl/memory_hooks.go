package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/toejough/projctl/internal/memory"
)

// memoryHooksInstallArgs holds the command-line arguments for hooks install.
type memoryHooksInstallArgs struct {
	SettingsPath string `targ:"flag,desc=Path to Claude Code settings.json (default: ~/.claude/settings.json)"`
}

// memoryHooksShowArgs holds the command-line arguments for hooks show.
type memoryHooksShowArgs struct {
	SettingsPath string `targ:"flag,desc=Path to Claude Code settings.json (default: ~/.claude/settings.json)"`
}

// memoryHooksInstall installs projctl memory hooks into Claude Code settings.json.
func memoryHooksInstall(args memoryHooksInstallArgs) error {
	// Determine settings path
	settingsPath := args.SettingsPath
	if settingsPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		settingsPath = filepath.Join(home, ".claude", "settings.json")
	}

	// Create .claude directory if it doesn't exist
	claudeDir := filepath.Dir(settingsPath)
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Call internal InstallHooks function
	opts := memory.InstallHooksOpts{
		SettingsPath: settingsPath,
	}

	if err := memory.InstallHooks(opts); err != nil {
		return fmt.Errorf("failed to install hooks: %w", err)
	}

	// Print success message
	fmt.Printf("Hooks installed successfully to %s\n", settingsPath)
	fmt.Println("\nInstalled hooks:")
	fmt.Println("  - Stop: projctl memory extract-session --transcript $TRANSCRIPT_PATH &")
	fmt.Println("  - PreCompact: projctl memory extract-session --transcript $TRANSCRIPT_PATH &")
	fmt.Println("  - SessionStart: projctl memory query --primacy --stdin-project --min-confidence=0.3 --max-tokens=1000 -n 10 \"recent important learnings\"")
	fmt.Println("  - UserPromptSubmit: projctl memory query --primacy --stdin-prompt --min-confidence=0.3 --max-tokens=2000 -n 10")
	fmt.Println("  - PreToolUse: projctl memory query --stdin-tool --min-confidence=0.5 --max-tokens=1000 -n 5")

	return nil
}

// memoryHooksShow displays the current hook configuration.
func memoryHooksShow(args memoryHooksShowArgs) error {
	// Determine settings path
	settingsPath := args.SettingsPath
	if settingsPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		settingsPath = filepath.Join(home, ".claude", "settings.json")
	}

	// Call internal ShowHooks function
	opts := memory.ShowHooksOpts{
		SettingsPath: settingsPath,
	}

	result, err := memory.ShowHooks(opts)
	if err != nil {
		return fmt.Errorf("failed to show hooks: %w", err)
	}

	// Print result
	fmt.Println(result)

	return nil
}
