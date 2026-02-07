package step

import (
	"github.com/toejough/projctl/internal/state"
	"github.com/toejough/projctl/internal/task"
	"github.com/toejough/projctl/internal/worktree"
)

// StatusResult holds the structured output of step status.
// Traces to: TASK-4, ARCH-4, DES-3
type StatusResult struct {
	ActiveTasks    []ActiveTaskInfo    `json:"active_tasks"`
	CompletedTasks []CompletedTaskInfo `json:"completed_tasks"`
	BlockedTasks   []BlockedTaskInfo   `json:"blocked_tasks"`
}

// ActiveTaskInfo represents an actively running task.
// Traces to: TASK-4, ARCH-4, DES-3
type ActiveTaskInfo struct {
	ID        string `json:"id"`
	Worktree  string `json:"worktree"`
	Status    string `json:"status,omitempty"`
	StartedAt string `json:"started_at,omitempty"`
}

// CompletedTaskInfo represents a completed task.
// Traces to: TASK-4, ARCH-4, DES-3
type CompletedTaskInfo struct {
	ID          string `json:"id"`
	Status      string `json:"status,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
}

// BlockedTaskInfo represents a blocked task.
// Traces to: TASK-4, ARCH-4, DES-3
type BlockedTaskInfo struct {
	ID        string   `json:"id"`
	BlockedBy []string `json:"blocked_by"`
}

// Status returns the current status of all tasks by combining:
// - Active tasks from worktree list
// - Completed tasks from state file
// - Blocked tasks from dependency graph
// Traces to: TASK-4, ARCH-4, DES-3
func Status(dir string) (StatusResult, error) {
	result := StatusResult{
		ActiveTasks:    []ActiveTaskInfo{},
		CompletedTasks: []CompletedTaskInfo{},
		BlockedTasks:   []BlockedTaskInfo{},
	}

	// Get active tasks from worktree list
	mgr := worktree.NewManager(dir)
	activeWorktrees, err := mgr.List()
	if err == nil {
		for _, wt := range activeWorktrees {
			result.ActiveTasks = append(result.ActiveTasks, ActiveTaskInfo{
				ID:       wt.TaskID,
				Worktree: wt.Path,
			})
		}
	}

	// Get completed tasks from state file
	_, err = state.Get(dir)
	if err == nil {
		// Parse tasks from tasks.md to find completed ones
		graph, err := task.ParseDependencies(dir)
		if err == nil {
			for _, taskID := range graph.Tasks {
				if graph.Status[taskID] == "complete" {
					result.CompletedTasks = append(result.CompletedTasks, CompletedTaskInfo{
						ID: taskID,
					})
				}
			}
		}
	}

	// Get blocked tasks from dependency graph
	graph, err := task.ParseDependencies(dir)
	if err == nil {
		for _, taskID := range graph.Tasks {
			// Skip completed tasks
			if graph.Status[taskID] == "complete" {
				continue
			}

			// Check if task is blocked
			blocked := false
			var blockers []string
			for _, dep := range graph.Deps[taskID] {
				if graph.Status[dep] != "complete" {
					blocked = true
					blockers = append(blockers, dep)
				}
			}

			if blocked {
				result.BlockedTasks = append(result.BlockedTasks, BlockedTaskInfo{
					ID:        taskID,
					BlockedBy: blockers,
				})
			}
		}
	}

	return result, nil
}
