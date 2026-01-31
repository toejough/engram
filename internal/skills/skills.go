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
