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
func (m *Manager) CleanupAll() error {
	parentDir := m.ParentDir()

	// Check if parent directory exists
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		if os.IsNotExist(err) {
			// No worktrees to clean up
			return nil
		}
		return fmt.Errorf("failed to read worktrees directory: %w", err)
	}

	// Clean up each worktree found in the parent directory
	for _, entry := range entries {
		if entry.IsDir() {
			taskID := entry.Name()
			if err := m.Cleanup(taskID); err != nil {
				return fmt.Errorf("failed to cleanup worktree %s: %w", taskID, err)
			}
		}
	}

	// Try to remove parent dir if empty (should be after all cleanups)
	_ = os.Remove(parentDir)

	return nil
}

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

// Worktree represents an active task worktree.
type Worktree struct {
	TaskID string // The task ID (e.g., "TASK-001")
	Path   string // Absolute path to worktree directory
	Branch string // Branch name (e.g., "task/TASK-001")
}

// List returns all active task worktrees.
// It parses `git worktree list` output and filters to task/* branches only.
func (m *Manager) List() ([]Worktree, error) {
	output, err := m.git("worktree", "list")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktrees []Worktree
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse format: <path> <commit> [<branch>]
		// Example: /path/to/worktree  abc1234 [task/TASK-001]
		wt, ok := parseWorktreeLine(line)
		if !ok {
			continue
		}

		// Filter to task/* branches only
		if !strings.HasPrefix(wt.Branch, "task/") {
			continue
		}

		// Extract task ID from branch name
		wt.TaskID = strings.TrimPrefix(wt.Branch, "task/")
		worktrees = append(worktrees, wt)
	}

	return worktrees, nil
}

// parseWorktreeLine parses a single line from `git worktree list` output.
// Returns the Worktree and true if parsing succeeded, otherwise empty and false.
func parseWorktreeLine(line string) (Worktree, bool) {
	// Format: <path> <commit> [<branch>]
	// The path can contain spaces, but is followed by multiple spaces before commit
	// Branch is in square brackets at the end

	// Find the branch in brackets at the end
	bracketStart := strings.LastIndex(line, "[")
	bracketEnd := strings.LastIndex(line, "]")
	if bracketStart == -1 || bracketEnd == -1 || bracketEnd < bracketStart {
		return Worktree{}, false
	}

	branch := line[bracketStart+1 : bracketEnd]

	// Everything before the bracket section is path + commit
	// Split by whitespace to get path (first part) and commit (middle part)
	beforeBracket := strings.TrimSpace(line[:bracketStart])
	fields := strings.Fields(beforeBracket)
	if len(fields) < 2 {
		return Worktree{}, false
	}

	// Path is everything except the last field (which is commit)
	path := strings.Join(fields[:len(fields)-1], " ")

	return Worktree{
		Path:   path,
		Branch: branch,
	}, true
}

// git runs a git command in the repo directory.
func (m *Manager) git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = m.repoDir
	output, err := cmd.CombinedOutput()
	return string(output), err
}
