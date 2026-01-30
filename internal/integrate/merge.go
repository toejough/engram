// Package integrate handles merging per-project documentation into top-level docs.
package integrate

// MergeFS provides file system operations for merge.
type MergeFS interface {
	ReadFile(path string) (string, error)
	WriteFile(path string, content string) error
	FileExists(path string) bool
	RemoveAll(path string) error
}

// MergeResult holds the results of a merge operation.
type MergeResult struct {
	RequirementsAdded  int    // Number of requirements added
	DesignAdded        int    // Number of design decisions added
	ArchitectureAdded  int    // Number of architecture decisions added
	TasksAdded         int    // Number of tasks added
	IDsRenumbered      int    // Number of IDs that were renumbered due to conflicts
	LinksUpdated       int    // Number of traceability links updated
	Summary            string // Human-readable summary
}

// Merge merges per-project documentation into top-level docs.
func Merge(projectDir string, projectName string, fs MergeFS) (*MergeResult, error) {
	// TODO: implement
	return &MergeResult{}, nil
}
