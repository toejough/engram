package skills

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// DocsArgs holds arguments for the skills docs command.
type DocsArgs struct {
	SkillsDir string `targ:"flag,short=d,desc=Skills directory (default: ~/.claude/skills)"`
	SkillName string `targ:"positional,required,desc=Skill name"`
	Section   string `targ:"flag,short=s,desc=Specific section to output"`
}

// InstallArgs holds arguments for the skills install command.
type InstallArgs struct {
	RepoDir   string `targ:"flag,short=r,desc=Project repository directory"`
	TargetDir string `targ:"flag,short=t,desc=Target skills directory (default: ~/.claude/skills)"`
	SkillName string `targ:"positional,desc=Optional: specific skill to install"`
	Force     bool   `targ:"flag,short=f,desc=Overwrite conflicting directories"`
}

// ListArgs holds arguments for the skills list command.
type ListArgs struct {
	SkillsDir string `targ:"flag,short=d,desc=Skills directory (default: ~/.claude/skills)"`
}

// StatusArgs holds arguments for the skills status command.
type StatusArgs struct {
	RepoDir   string `targ:"flag,short=r,desc=Project repository directory"`
	TargetDir string `targ:"flag,short=t,desc=Target skills directory (default: ~/.claude/skills)"`
}

// UninstallArgs holds arguments for the skills uninstall command.
type UninstallArgs struct {
	RepoDir   string `targ:"flag,short=r,desc=Project repository directory"`
	TargetDir string `targ:"flag,short=t,desc=Target skills directory (default: ~/.claude/skills)"`
	SkillName string `targ:"positional,desc=Optional: specific skill to uninstall"`
}

// RunDocs shows full skill documentation.
func RunDocs(args DocsArgs) error {
	if args.SkillName == "" {
		return errors.New("skill name is required")
	}

	skillsDir := args.SkillsDir
	if skillsDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		skillsDir = filepath.Join(home, ".claude", "skills")
	}

	var (
		content string
		err     error
	)

	if args.Section != "" {
		content, err = DocsSection(skillsDir, args.SkillName, args.Section)
	} else {
		content, err = Docs(skillsDir, args.SkillName)
	}

	if err != nil {
		return err
	}

	fmt.Print(content)

	return nil
}

// RunInstall installs skills by creating symlinks.
func RunInstall(args InstallArgs) error {
	repoDir := args.RepoDir
	if repoDir == "" {
		var err error

		repoDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	targetDir := args.TargetDir
	if targetDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		targetDir = filepath.Join(home, ".claude", "skills")
	}

	skillsDir := filepath.Join(repoDir, "skills")
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		return fmt.Errorf("skills directory not found: %s", skillsDir)
	}

	result, err := Install(skillsDir, targetDir, InstallOpts{
		SkillName: args.SkillName,
		Force:     args.Force,
	})
	if err != nil {
		return err
	}

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

// RunList lists available skills.
func RunList(args ListArgs) error {
	skillsDir := args.SkillsDir
	if skillsDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		skillsDir = filepath.Join(home, ".claude", "skills")
	}

	names, err := List(skillsDir)
	if err != nil {
		return err
	}

	for _, name := range names {
		fmt.Println(name)
	}

	return nil
}

// RunStatus shows skill installation status.
func RunStatus(args StatusArgs) error {
	return RunStatusCore(args, os.Exit)
}

// RunStatusCore is the testable core of RunStatus with an injectable exit function.
func RunStatusCore(args StatusArgs, exit func(int)) error {
	repoDir := args.RepoDir
	if repoDir == "" {
		var err error

		repoDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	targetDir := args.TargetDir
	if targetDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		targetDir = filepath.Join(home, ".claude", "skills")
	}

	skillsDir := filepath.Join(repoDir, "skills")
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		return fmt.Errorf("skills directory not found: %s", skillsDir)
	}

	result, err := Status(skillsDir, targetDir)
	if err != nil {
		return err
	}

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
		exit(1)
	}

	return nil
}

// RunUninstall uninstalls skills by removing symlinks.
func RunUninstall(args UninstallArgs) error {
	repoDir := args.RepoDir
	if repoDir == "" {
		var err error

		repoDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	targetDir := args.TargetDir
	if targetDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		targetDir = filepath.Join(home, ".claude", "skills")
	}

	skillsDir := filepath.Join(repoDir, "skills")
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		return fmt.Errorf("skills directory not found: %s", skillsDir)
	}

	result, err := Uninstall(skillsDir, targetDir, UninstallOpts{
		SkillName: args.SkillName,
	})
	if err != nil {
		return err
	}

	if len(result.Removed) > 0 {
		fmt.Printf("Removed %d skills:\n", len(result.Removed))

		for _, name := range result.Removed {
			fmt.Printf("  %s\n", name)
		}
	}

	if len(result.Skipped) > 0 {
		fmt.Printf("Skipped %d skills (not symlinks):\n", len(result.Skipped))

		for _, name := range result.Skipped {
			fmt.Printf("  %s\n", name)
		}
	}

	if len(result.Removed) == 0 && len(result.Skipped) == 0 {
		fmt.Println("Nothing to uninstall")
	}

	return nil
}
