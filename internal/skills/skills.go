// Package skills manages skill installation and symlink management.
package skills

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
	// TODO: Implement
	return InstallResult{}, nil
}
