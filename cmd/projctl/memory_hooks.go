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

// memoryHooksCheckClaudeMDArgs holds the command-line arguments for hooks check-claudemd.
type memoryHooksCheckClaudeMDArgs struct {
	ClaudeMDPath string `targ:"flag,desc=Path to CLAUDE.md (default: ~/.claude/CLAUDE.md)"`
	MaxLines     int    `targ:"flag,desc=Maximum line count (default: 260)"`
}

// memoryHooksCheckSkillArgs holds the command-line arguments for hooks check-skill.
type memoryHooksCheckSkillArgs struct {
	SkillsDir string `targ:"flag,desc=Path to skills directory (default: ~/.claude/skills)"`
}

// memoryHooksCheckEmbeddingArgs holds the command-line arguments for hooks check-embedding.
type memoryHooksCheckEmbeddingArgs struct {
	MemoryRoot string `targ:"flag,desc=Path to memory root directory (default: ~/.claude/memory)"`
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
	fmt.Println("  - Stop:")
	fmt.Println("      * projctl memory extract-session --transcript $TRANSCRIPT_PATH &")
	fmt.Println("      * projctl memory hooks check-claudemd --max-lines=260")
	fmt.Println("  - PreCompact: projctl memory extract-session --transcript $TRANSCRIPT_PATH &")
	fmt.Println("  - SessionStart: projctl memory query --primacy --stdin-project --min-confidence=30 --max-tokens=1000 -n 10 \"recent important learnings\"")
	fmt.Println("  - UserPromptSubmit: projctl memory query --primacy --stdin-prompt --min-confidence=30 --max-tokens=2000 -n 10")
	fmt.Println("  - PreToolUse: projctl memory query --stdin-tool --min-confidence=50 --max-tokens=1000 -n 5")
	fmt.Println("  - PostToolUse (matcher: Bash): projctl memory hooks check-embedding")
	fmt.Println("  - TeammateIdle: projctl memory hooks check-skill")

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

// memoryHooksCheckClaudeMD checks if CLAUDE.md exceeds the maximum line count.
func memoryHooksCheckClaudeMD(args memoryHooksCheckClaudeMDArgs) error {
	// Determine CLAUDE.md path
	claudeMDPath := args.ClaudeMDPath
	if claudeMDPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		claudeMDPath = filepath.Join(home, ".claude", "CLAUDE.md")
	}

	// Determine max lines
	maxLines := args.MaxLines
	if maxLines == 0 {
		maxLines = 260
	}

	// Call internal check function
	if err := memory.CheckClaudeMDSize(memory.CheckClaudeMDSizeOpts{
		ClaudeMDPath: claudeMDPath,
		MaxLines:     maxLines,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(2)
	}

	return nil
}

// memoryHooksCheckSkill validates SKILL.md files in the skills directory.
func memoryHooksCheckSkill(args memoryHooksCheckSkillArgs) error {
	// Determine skills directory
	skillsDir := args.SkillsDir
	if skillsDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		skillsDir = filepath.Join(home, ".claude", "skills")
	}

	// Call internal check function
	if err := memory.CheckSkillContract(memory.CheckSkillContractOpts{
		SkillsDir: skillsDir,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(2)
	}

	return nil
}

// memoryHooksCheckEmbedding validates embedding metadata completeness.
func memoryHooksCheckEmbedding(args memoryHooksCheckEmbeddingArgs) error {
	// Determine memory root
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = filepath.Join(home, ".claude", "memory")
	}

	// Call internal check function (reads from stdin)
	if err := memory.CheckEmbeddingMetadata(memory.CheckEmbeddingMetaOpts{
		MemoryRoot: memoryRoot,
		Stdin:      os.Stdin,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(2)
	}

	return nil
}
