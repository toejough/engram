package step

// StatusResult holds the structured output of step status.
type StatusResult struct {
	ActiveTasks    []ActiveTaskInfo    `json:"active_tasks"`
	CompletedTasks []CompletedTaskInfo `json:"completed_tasks"`
	BlockedTasks   []BlockedTaskInfo   `json:"blocked_tasks"`
}

// ActiveTaskInfo represents an actively running task.
type ActiveTaskInfo struct {
	ID        string `json:"id"`
	Worktree  string `json:"worktree"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at"`
}

// CompletedTaskInfo represents a completed task.
type CompletedTaskInfo struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	CompletedAt string `json:"completed_at"`
}

// BlockedTaskInfo represents a blocked task.
type BlockedTaskInfo struct {
	ID        string   `json:"id"`
	BlockedBy []string `json:"blocked_by"`
}

// Status returns the current status of all tasks.
// TODO: TASK-4 implementation
func Status(dir string) (StatusResult, error) {
	// Not implemented yet - this is a stub for tests to compile
	return StatusResult{}, nil
}
