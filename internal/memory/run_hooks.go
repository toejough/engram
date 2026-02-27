package memory

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// RunHooksCheckClaudeMD checks CLAUDE.md size constraints.
func RunHooksCheckClaudeMD(args HooksCheckClaudeMDArgs, homeDir string) error {
	claudeMDPath := args.ClaudeMDPath
	if claudeMDPath == "" {
		claudeMDPath = filepath.Join(homeDir, ".claude", "CLAUDE.md")
	}

	maxLines := args.MaxLines
	if maxLines == 0 {
		maxLines = 260
	}

	memoryRoot := filepath.Join(homeDir, ".claude", "memory")

	return CheckClaudeMDSize(CheckClaudeMDSizeOpts{
		ClaudeMDPath: claudeMDPath,
		MaxLines:     maxLines,
		MemoryRoot:   memoryRoot,
	})
}

// RunHooksCheckEmbedding checks embedding metadata from a hook event.
func RunHooksCheckEmbedding(args HooksCheckEmbeddingArgs, homeDir string, stdin io.Reader) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		memoryRoot = filepath.Join(homeDir, ".claude", "memory")
	}

	return CheckEmbeddingMetadata(CheckEmbeddingMetaOpts{
		MemoryRoot: memoryRoot,
		Stdin:      stdin,
	})
}

// RunHooksCheckSkill checks skill contract compliance.
func RunHooksCheckSkill(args HooksCheckSkillArgs, homeDir string) error {
	skillsDir := args.SkillsDir
	if skillsDir == "" {
		skillsDir = filepath.Join(homeDir, ".claude", "skills")
	}

	memoryRoot := filepath.Join(homeDir, ".claude", "memory")

	return CheckSkillContract(CheckSkillContractOpts{
		SkillsDir:  skillsDir,
		MemoryRoot: memoryRoot,
	})
}

// RunHooksInstall installs memory hooks into Claude settings.
func RunHooksInstall(args HooksInstallArgs, homeDir string) error {
	settingsPath := args.SettingsPath
	if settingsPath == "" {
		settingsPath = filepath.Join(homeDir, ".claude", "settings.json")
	}

	claudeDir := filepath.Dir(settingsPath)

	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	if err := InstallHooks(InstallHooksOpts{SettingsPath: settingsPath}); err != nil {
		return fmt.Errorf("failed to install hooks: %w", err)
	}

	fmt.Printf("Hooks installed successfully to %s\n", settingsPath)
	fmt.Println("\nInstalled hooks:")
	fmt.Println("  - Stop:")
	fmt.Println("      * projctl memory extract-session")
	fmt.Println("      * projctl memory hooks check-claudemd --max-lines=260")
	fmt.Println("  - PreCompact: projctl memory extract-session")
	fmt.Println("  - SessionStart: projctl memory query --primacy --stdin-project --min-confidence=30 --max-tokens=1000 -n 10 \"recent important learnings\"")
	fmt.Println("  - UserPromptSubmit: projctl memory query --primacy --stdin-prompt --min-confidence=30 --max-tokens=2000 -n 10")
	fmt.Println("  - PreToolUse: projctl memory query --stdin-tool --min-confidence=50 --max-tokens=1000 -n 5")
	fmt.Println("  - PostToolUse (matcher: Bash): projctl memory hooks check-embedding")
	fmt.Println("  - TeammateIdle: projctl memory hooks check-skill")

	return nil
}

// RunHooksShow shows the current hook configuration.
func RunHooksShow(args HooksShowArgs, homeDir string) error {
	settingsPath := args.SettingsPath
	if settingsPath == "" {
		settingsPath = filepath.Join(homeDir, ".claude", "settings.json")
	}

	result, err := ShowHooks(ShowHooksOpts{SettingsPath: settingsPath})
	if err != nil {
		return fmt.Errorf("failed to show hooks: %w", err)
	}

	fmt.Println(result)

	return nil
}

// RunHooksStats shows hook execution statistics.
func RunHooksStats(args HooksStatsArgs, homeDir string) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		memoryRoot = filepath.Join(homeDir, ".claude", "memory")
	}

	db, err := InitDBForTest(memoryRoot)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer func() { _ = db.Close() }()

	stats, err := GetHookStats(db)
	if err != nil {
		return fmt.Errorf("failed to get hook stats: %w", err)
	}

	if len(stats) == 0 {
		fmt.Println("No hook events recorded")
		return nil
	}

	fmt.Printf("%-30s %10s %12s %15s %25s\n", "Hook Name", "Fire Count", "Success Rate", "Avg Duration", "Last Fired")
	fmt.Println(string(make([]byte, 100)))

	for i := range stats {
		fmt.Printf("%-30s %10d %11.1f%% %12dms %25s\n",
			stats[i].HookName,
			stats[i].FireCount,
			stats[i].SuccessRate*100,
			stats[i].AvgDuration,
			stats[i].LastFired)
	}

	return nil
}
