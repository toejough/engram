package worktree

import (
	"fmt"
	"os"
	"text/tabwriter"
)

// CleanupAllArgs holds arguments for the worktree cleanup-all command.
type CleanupAllArgs struct {
	RepoDir string `targ:"flag,short=r,desc=Repository root (defaults to current directory)"`
}

// CleanupArgs holds arguments for the worktree cleanup command.
type CleanupArgs struct {
	TaskID  string `targ:"flag,short=t,required,desc=Task ID to cleanup"`
	RepoDir string `targ:"flag,short=r,desc=Repository root (defaults to current directory)"`
}

// CreateArgs holds arguments for the worktree create command.
type CreateArgs struct {
	TaskID     string `targ:"flag,short=t,required,desc=Task ID (e.g. TASK-001)"`
	BaseBranch string `targ:"flag,short=b,desc=Base branch (auto-detected if not provided)"`
	RepoDir    string `targ:"flag,short=r,desc=Repository root (defaults to current directory)"`
}

// CreateProjectArgs holds arguments for the worktree create-project command.
type CreateProjectArgs struct {
	ProjectName string `targ:"flag,short=p,required,desc=Project name"`
	BaseBranch  string `targ:"flag,short=b,desc=Base branch (auto-detected if not provided)"`
	RepoDir     string `targ:"flag,short=r,desc=Repository root (defaults to current directory)"`
}

// ListArgs holds arguments for the worktree list command.
type ListArgs struct {
	RepoDir string `targ:"flag,short=r,desc=Repository root (defaults to current directory)"`
}

// MergeArgs holds arguments for the worktree merge command.
type MergeArgs struct {
	TaskID  string `targ:"flag,short=t,required,desc=Task ID to merge"`
	Onto    string `targ:"flag,short=o,desc=Target branch (defaults to main)"`
	RepoDir string `targ:"flag,short=r,desc=Repository root (defaults to current directory)"`
}

// MergeProjectArgs holds arguments for the worktree merge-project command.
type MergeProjectArgs struct {
	ProjectName string `targ:"flag,short=p,required,desc=Project name to merge"`
	Onto        string `targ:"flag,short=o,desc=Target branch (defaults to base branch)"`
	RepoDir     string `targ:"flag,short=r,desc=Repository root (defaults to current directory)"`
}

// RunCleanup removes a task worktree.
func RunCleanup(args CleanupArgs) error {
	repoDir := args.RepoDir
	if repoDir == "" {
		repoDir = "."
	}

	mgr := NewManager(repoDir)

	err := mgr.Cleanup(args.TaskID)
	if err != nil {
		return err
	}

	fmt.Printf("Cleaned up worktree for %s\n", args.TaskID)

	return nil
}

// RunCleanupAll removes all task worktrees.
func RunCleanupAll(args CleanupAllArgs) error {
	repoDir := args.RepoDir
	if repoDir == "" {
		repoDir = "."
	}

	mgr := NewManager(repoDir)

	err := mgr.CleanupAll()
	if err != nil {
		return err
	}

	fmt.Println("Cleaned up all worktrees")

	return nil
}

// RunCreate creates a worktree for a task.
func RunCreate(args CreateArgs) error {
	repoDir := args.RepoDir
	if repoDir == "" {
		repoDir = "."
	}

	mgr := NewManager(repoDir)

	baseBranch := args.BaseBranch
	if baseBranch == "" {
		var err error

		baseBranch, err = mgr.DetectBaseBranch()
		if err != nil {
			return fmt.Errorf("failed to detect base branch: %w", err)
		}
	}

	path, err := mgr.Create(args.TaskID, baseBranch)
	if err != nil {
		return err
	}

	fmt.Printf("Created worktree for %s at %s (base: %s)\n", args.TaskID, path, baseBranch)

	return nil
}

// RunCreateProject creates a worktree for a project.
func RunCreateProject(args CreateProjectArgs) error {
	repoDir := args.RepoDir
	if repoDir == "" {
		repoDir = "."
	}

	mgr := NewManager(repoDir)

	baseBranch := args.BaseBranch
	if baseBranch == "" {
		var err error

		baseBranch, err = mgr.DetectBaseBranch()
		if err != nil {
			return fmt.Errorf("failed to detect base branch: %w", err)
		}
	}

	path, err := mgr.CreateProject(args.ProjectName, baseBranch)
	if err != nil {
		return err
	}

	fmt.Printf("Created project worktree for %s at %s (base: %s)\n", args.ProjectName, path, baseBranch)

	return nil
}

// RunList lists all worktrees.
func RunList(args ListArgs) error {
	repoDir := args.RepoDir
	if repoDir == "" {
		repoDir = "."
	}

	mgr := NewManager(repoDir)

	worktrees, err := mgr.List()
	if err != nil {
		return err
	}

	if len(worktrees) == 0 {
		fmt.Println("No worktrees found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "TYPE\tNAME\tBRANCH\tPATH"); err != nil {
		return err
	}

	for _, wt := range worktrees {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", wt.Type, wt.TaskID, wt.Branch, wt.Path); err != nil {
			return err
		}
	}

	return w.Flush()
}

// RunMerge merges a task worktree onto target branch.
func RunMerge(args MergeArgs) error {
	repoDir := args.RepoDir
	if repoDir == "" {
		repoDir = "."
	}

	mgr := NewManager(repoDir)

	onto := args.Onto
	if onto == "" {
		var err error

		onto, err = mgr.DetectBaseBranch()
		if err != nil {
			return fmt.Errorf("failed to detect base branch: %w", err)
		}
	}

	err := mgr.Merge(args.TaskID, onto)
	if err != nil {
		return err
	}

	fmt.Printf("Merged %s onto %s\n", args.TaskID, onto)

	return nil
}

// RunMergeProject merges a project worktree onto target branch.
func RunMergeProject(args MergeProjectArgs) error {
	repoDir := args.RepoDir
	if repoDir == "" {
		repoDir = "."
	}

	mgr := NewManager(repoDir)

	onto := args.Onto
	if onto == "" {
		var err error

		onto, err = mgr.DetectBaseBranch()
		if err != nil {
			return fmt.Errorf("failed to detect base branch: %w", err)
		}
	}

	err := mgr.MergeProject(args.ProjectName, onto)
	if err != nil {
		return err
	}

	fmt.Printf("Merged project %s onto %s\n", args.ProjectName, onto)

	return nil
}
