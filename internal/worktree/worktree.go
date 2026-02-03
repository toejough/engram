package worktree

import (
	"fmt"
	"os"
	"os/exec"
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

// Create creates a new worktree for the given task.
// Creates the branch and worktree directory.
func (m *Manager) Create(taskID string) (string, error) {
	wtPath := m.Path(taskID)
	branch := m.BranchName(taskID)

	// Create parent directory if needed
	if err := os.MkdirAll(m.ParentDir(), 0o755); err != nil {
		return "", fmt.Errorf("failed to create worktree parent dir: %w", err)
	}

	// Check if worktree already exists
	if _, err := os.Stat(wtPath); err == nil {
		return "", fmt.Errorf("worktree already exists at %s", wtPath)
	}

	// Create branch and worktree in one command
	output, err := m.git("worktree", "add", "-b", branch, wtPath)
	if err != nil {
		return "", fmt.Errorf("failed to create worktree: %s: %w", output, err)
	}

	return wtPath, nil
}

// WorktreeInfo contains information about a worktree.
type WorktreeInfo struct {
	TaskID string
	Path   string
	Branch string
}

// List returns all worktrees managed by this manager.
func (m *Manager) List() ([]WorktreeInfo, error) {
	output, err := m.git("worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktrees []WorktreeInfo
	parentDir := m.ParentDir()

	// Parse porcelain output: "worktree <path>\nHEAD <sha>\nbranch refs/heads/<branch>\n\n"
	lines := strings.Split(output, "\n")
	var currentPath, currentBranch string
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			currentPath = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "branch refs/heads/") {
			currentBranch = strings.TrimPrefix(line, "branch refs/heads/")
		} else if line == "" && currentPath != "" {
			// Only include worktrees in our managed directory
			if strings.HasPrefix(currentPath, parentDir) && strings.HasPrefix(currentBranch, "task/") {
				taskID := strings.TrimPrefix(currentBranch, "task/")
				worktrees = append(worktrees, WorktreeInfo{
					TaskID: taskID,
					Path:   currentPath,
					Branch: currentBranch,
				})
			}
			currentPath = ""
			currentBranch = ""
		}
	}

	return worktrees, nil
}

// CleanupAll removes all worktrees managed by this manager.
func (m *Manager) CleanupAll() error {
	worktrees, err := m.List()
	if err != nil {
		return err
	}

	var errs []string
	for _, wt := range worktrees {
		if err := m.Cleanup(wt.TaskID); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", wt.TaskID, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to cleanup some worktrees: %s", strings.Join(errs, "; "))
	}

	return nil
}

// Cleanup removes a worktree and its branch.
func (m *Manager) Cleanup(taskID string) error {
	wtPath := m.Path(taskID)
	branch := m.BranchName(taskID)

	// Remove worktree
	if _, err := m.git("worktree", "remove", "--force", wtPath); err != nil {
		// Try manual removal if git command fails
		if rmErr := os.RemoveAll(wtPath); rmErr != nil {
			return fmt.Errorf("failed to remove worktree directory: %w", rmErr)
		}
		// Prune worktree list
		_, _ = m.git("worktree", "prune")
	}

	// Delete branch
	_, _ = m.git("branch", "-D", branch)

	// Try to remove parent dir if empty
	_ = os.Remove(m.ParentDir())

	return nil
}

// CleanupAll removes all task worktrees and their branches.

// Merge rebases a task branch onto the target and fast-forward merges.
func (m *Manager) Merge(taskID, onto string) error {
	branch := m.BranchName(taskID)
	wtPath := m.Path(taskID)

	// First remove the worktree (but keep the branch)
	// Can't rebase a branch that's checked out in a worktree
	if _, err := os.Stat(wtPath); err == nil {
		if _, err := m.git("worktree", "remove", "--force", wtPath); err != nil {
			// Try manual removal
			if rmErr := os.RemoveAll(wtPath); rmErr != nil {
				return fmt.Errorf("failed to remove worktree before merge: %w", rmErr)
			}
			_, _ = m.git("worktree", "prune")
		}
	}

	// Rebase task branch onto target
	output, err := m.git("rebase", onto, branch)
	if err != nil {
		// Check if it's a conflict
		if strings.Contains(output, "CONFLICT") || strings.Contains(output, "could not apply") {
			// Abort the rebase
			_, _ = m.git("rebase", "--abort")
			return &MergeConflictError{
				TaskID:  taskID,
				Message: output,
			}
		}
		return fmt.Errorf("rebase failed: %s: %w", output, err)
	}

	// Fast-forward merge
	output, err = m.git("checkout", onto)
	if err != nil {
		return fmt.Errorf("checkout failed: %s: %w", output, err)
	}

	output, err = m.git("merge", "--ff-only", branch)
	if err != nil {
		return fmt.Errorf("merge failed: %s: %w", output, err)
	}

	// Delete the branch (worktree already removed)
	_, _ = m.git("branch", "-D", branch)

	// Try to remove parent dir if empty
	_ = os.Remove(m.ParentDir())

	return nil
}

// MergeConflictError indicates a merge conflict occurred.
type MergeConflictError struct {
	TaskID  string
	Message string
}

func (e *MergeConflictError) Error() string {
	return fmt.Sprintf("merge conflict for task %s: %s", e.TaskID, e.Message)
}

// git runs a git command in the repo directory.
func (m *Manager) git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = m.repoDir
	output, err := cmd.CombinedOutput()
	return string(output), err
}
