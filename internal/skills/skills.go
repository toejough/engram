// Package skills manages skill installation and symlink management.
package skills

import (
	"fmt"
	"os"
	"path/filepath"
)

// InstallOpts configures the Install operation.
type InstallOpts struct {
	SkillName string // If set, install only this skill
	Force     bool   // If true, overwrite conflicting directories
}

// InstallResult contains the results of an install operation.
type InstallResult struct {
	Linked    []string // Skills that were newly linked
	Updated   []string // Skills whose symlinks were updated
	Conflicts []string // Skills that had conflicts (non-symlink dirs)
	Skipped   []string // Skills that were already correctly linked
}

// Install creates symlinks from repoSkillsDir to targetDir.
func Install(repoSkillsDir, targetDir string, opts InstallOpts) (InstallResult, error) {
	var result InstallResult

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return result, fmt.Errorf("failed to create target directory: %w", err)
	}

	// Get list of skills to install
	var skillNames []string
	if opts.SkillName != "" {
		// Check if specific skill exists
		skillPath := filepath.Join(repoSkillsDir, opts.SkillName)
		if _, err := os.Stat(skillPath); os.IsNotExist(err) {
			return result, fmt.Errorf("skill not found: %s", opts.SkillName)
		}
		skillNames = []string{opts.SkillName}
	} else {
		// List all skills in repo
		entries, err := os.ReadDir(repoSkillsDir)
		if err != nil {
			return result, fmt.Errorf("failed to read skills directory: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				skillNames = append(skillNames, entry.Name())
			}
		}
	}

	// Install each skill
	for _, name := range skillNames {
		srcPath := filepath.Join(repoSkillsDir, name)
		dstPath := filepath.Join(targetDir, name)

		// Check if destination exists
		info, err := os.Lstat(dstPath)
		if err == nil {
			// Destination exists
			if info.Mode()&os.ModeSymlink != 0 {
				// It's a symlink - check if it points to the right place
				currentTarget, err := os.Readlink(dstPath)
				if err != nil {
					return result, fmt.Errorf("failed to read symlink %s: %w", dstPath, err)
				}
				if currentTarget == srcPath {
					// Already correctly linked
					result.Skipped = append(result.Skipped, name)
					continue
				}
				// Points elsewhere - update it
				if err := os.Remove(dstPath); err != nil {
					return result, fmt.Errorf("failed to remove old symlink %s: %w", dstPath, err)
				}
				if err := os.Symlink(srcPath, dstPath); err != nil {
					return result, fmt.Errorf("failed to create symlink %s: %w", dstPath, err)
				}
				result.Updated = append(result.Updated, name)
			} else {
				// It's a regular directory/file - conflict
				if opts.Force {
					// Remove and replace
					if err := os.RemoveAll(dstPath); err != nil {
						return result, fmt.Errorf("failed to remove conflicting path %s: %w", dstPath, err)
					}
					if err := os.Symlink(srcPath, dstPath); err != nil {
						return result, fmt.Errorf("failed to create symlink %s: %w", dstPath, err)
					}
					result.Linked = append(result.Linked, name)
				} else {
					result.Conflicts = append(result.Conflicts, name)
				}
			}
		} else if os.IsNotExist(err) {
			// Destination doesn't exist - create symlink
			if err := os.Symlink(srcPath, dstPath); err != nil {
				return result, fmt.Errorf("failed to create symlink %s: %w", dstPath, err)
			}
			result.Linked = append(result.Linked, name)
		} else {
			return result, fmt.Errorf("failed to check destination %s: %w", dstPath, err)
		}
	}

	return result, nil
}

// StatusResult contains the status of all skills.
type StatusResult struct {
	Linked    []string // Skills properly symlinked to repo
	Missing   []string // Repo skills not installed
	Local     []string // Skills only in target (not in repo)
	Conflicts []string // Non-symlink directories with same name as repo skill
	Stale     []string // Symlinks pointing to wrong location
}

// Status returns the installation status of all skills.
func Status(repoSkillsDir, targetDir string) (StatusResult, error) {
	var result StatusResult

	// Build map of repo skills
	repoSkills := make(map[string]bool)
	if entries, err := os.ReadDir(repoSkillsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				repoSkills[entry.Name()] = true
			}
		}
	}

	// Build map of target skills
	targetSkills := make(map[string]bool)
	if entries, err := os.ReadDir(targetDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				targetSkills[entry.Name()] = true
			}
		}
	}

	// Check each repo skill
	for name := range repoSkills {
		srcPath := filepath.Join(repoSkillsDir, name)
		dstPath := filepath.Join(targetDir, name)

		info, err := os.Lstat(dstPath)
		if os.IsNotExist(err) {
			// Not installed
			result.Missing = append(result.Missing, name)
			continue
		}
		if err != nil {
			return result, fmt.Errorf("failed to check %s: %w", dstPath, err)
		}

		if info.Mode()&os.ModeSymlink != 0 {
			// It's a symlink - check where it points
			target, err := os.Readlink(dstPath)
			if err != nil {
				return result, fmt.Errorf("failed to read symlink %s: %w", dstPath, err)
			}
			if target == srcPath {
				result.Linked = append(result.Linked, name)
			} else {
				result.Stale = append(result.Stale, name)
			}
		} else {
			// It's a regular directory - conflict
			result.Conflicts = append(result.Conflicts, name)
		}
	}

	// Check for local-only skills (in target but not in repo)
	for name := range targetSkills {
		if !repoSkills[name] {
			result.Local = append(result.Local, name)
		}
	}

	return result, nil
}
