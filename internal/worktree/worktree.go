package worktree

import (
	"path/filepath"
	"strings"
)

// Manager handles git worktree operations for parallel task execution.
type Manager struct {
	repoDir string
}

// NewManager creates a Manager for the given repository directory.
func NewManager(repoDir string) *Manager {
	// Resolve to absolute path
	absPath, err := filepath.Abs(repoDir)
	if err != nil {
		absPath = repoDir
	}
	return &Manager{repoDir: absPath}
}

// RepoDir returns the repository directory.
func (m *Manager) RepoDir() string {
	return m.repoDir
}

// ParentDir returns the parent directory for all worktrees.
// Pattern: <repo>/../<repo-name>-worktrees/
func (m *Manager) ParentDir() string {
	repoName := filepath.Base(m.repoDir)
	parent := filepath.Dir(m.repoDir)
	return filepath.Join(parent, repoName+"-worktrees")
}

// Path returns the canonical worktree path for a task.
// Pattern: <repo>/../<repo-name>-worktrees/<task-id>/
func (m *Manager) Path(taskID string) string {
	// Sanitize task ID: dots to dashes
	sanitized := strings.ReplaceAll(taskID, ".", "-")
	return filepath.Join(m.ParentDir(), sanitized)
}

// BranchName returns the branch name for a task.
// Pattern: task/<task-id>
func (m *Manager) BranchName(taskID string) string {
	return "task/" + taskID
}
