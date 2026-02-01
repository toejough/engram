package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/toejough/projctl/internal/skills"
)

type skillsInstallArgs struct {
	RepoDir   string `targ:"--repo,-r,Project repository directory"`
	TargetDir string `targ:"--target,-t,Target skills directory (default: ~/.claude/skills)"`
	SkillName string `targ:"[skill],Optional: specific skill to install"`
	Force     bool   `targ:"--force,-f,Overwrite conflicting directories"`
}

func skillsInstall(args skillsInstallArgs) error {
	// Default repo dir to current directory
	repoDir := args.RepoDir
	if repoDir == "" {
		var err error
		repoDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Default target to ~/.claude/skills
	targetDir := args.TargetDir
	if targetDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		targetDir = filepath.Join(home, ".claude", "skills")
	}

	// Skills dir is repo/skills
	skillsDir := filepath.Join(repoDir, "skills")
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		return fmt.Errorf("skills directory not found: %s", skillsDir)
	}

	result, err := skills.Install(skillsDir, targetDir, skills.InstallOpts{
		SkillName: args.SkillName,
		Force:     args.Force,
	})
	if err != nil {
		return err
	}

	// Report results
	if len(result.Linked) > 0 {
		fmt.Printf("Linked %d skills:\n", len(result.Linked))
		for _, name := range result.Linked {
			fmt.Printf("  %s\n", name)
		}
	}
	if len(result.Updated) > 0 {
		fmt.Printf("Updated %d skills:\n", len(result.Updated))
		for _, name := range result.Updated {
			fmt.Printf("  %s\n", name)
		}
	}
	if len(result.Skipped) > 0 {
		fmt.Printf("Skipped %d skills (already linked):\n", len(result.Skipped))
		for _, name := range result.Skipped {
			fmt.Printf("  %s\n", name)
		}
	}
	if len(result.Conflicts) > 0 {
		fmt.Printf("Conflicts (%d skills, use --force to overwrite):\n", len(result.Conflicts))
		for _, name := range result.Conflicts {
			fmt.Printf("  %s\n", name)
		}
	}

	return nil
}

type skillsStatusArgs struct {
	RepoDir   string `targ:"--repo,-r,Project repository directory"`
	TargetDir string `targ:"--target,-t,Target skills directory (default: ~/.claude/skills)"`
}

func skillsStatus(args skillsStatusArgs) error {
	// Default repo dir to current directory
	repoDir := args.RepoDir
	if repoDir == "" {
		var err error
		repoDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Default target to ~/.claude/skills
	targetDir := args.TargetDir
	if targetDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		targetDir = filepath.Join(home, ".claude", "skills")
	}

	// Skills dir is repo/skills
	skillsDir := filepath.Join(repoDir, "skills")
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		return fmt.Errorf("skills directory not found: %s", skillsDir)
	}

	result, err := skills.Status(skillsDir, targetDir)
	if err != nil {
		return err
	}

	// Report results
	hasIssues := false

	if len(result.Linked) > 0 {
		fmt.Printf("Linked (%d):\n", len(result.Linked))
		for _, name := range result.Linked {
			fmt.Printf("  ✓ %s\n", name)
		}
	}
	if len(result.Missing) > 0 {
		hasIssues = true
		fmt.Printf("Missing (%d):\n", len(result.Missing))
		for _, name := range result.Missing {
			fmt.Printf("  ✗ %s\n", name)
		}
	}
	if len(result.Stale) > 0 {
		hasIssues = true
		fmt.Printf("Stale (%d, needs update):\n", len(result.Stale))
		for _, name := range result.Stale {
			fmt.Printf("  ~ %s\n", name)
		}
	}
	if len(result.Conflicts) > 0 {
		hasIssues = true
		fmt.Printf("Conflicts (%d, use --force to overwrite):\n", len(result.Conflicts))
		for _, name := range result.Conflicts {
			fmt.Printf("  ! %s\n", name)
		}
	}
	if len(result.Local) > 0 {
		fmt.Printf("Local only (%d):\n", len(result.Local))
		for _, name := range result.Local {
			fmt.Printf("  ? %s\n", name)
		}
	}

	if hasIssues {
		os.Exit(1)
	}
	return nil
}

type skillsUninstallArgs struct {
	TargetDir string `targ:"--target,-t,Target skills directory (default: ~/.claude/skills)"`
	SkillName string `targ:"[skill],Optional: specific skill to uninstall"`
}

func skillsUninstall(args skillsUninstallArgs) error {
	// TODO: Implement in TASK-058
	return fmt.Errorf("not implemented")
}
