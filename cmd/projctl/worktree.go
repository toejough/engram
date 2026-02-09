package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/toejough/projctl/internal/worktree"
)

type worktreeCreateArgs struct {
	TaskID     string `targ:"flag,short=t,required,desc=Task ID (e.g. TASK-001)"`
	BaseBranch string `targ:"flag,short=b,desc=Base branch (auto-detected if not provided)"`
	RepoDir    string `targ:"flag,short=r,desc=Repository root (defaults to current directory)"`
}

func worktreeCreate(args worktreeCreateArgs) error {
	repoDir := args.RepoDir
	if repoDir == "" {
		repoDir = "."
	}

	mgr := worktree.NewManager(repoDir)

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

type worktreeCreateProjectArgs struct {
	ProjectName string `targ:"flag,short=p,required,desc=Project name"`
	BaseBranch  string `targ:"flag,short=b,desc=Base branch (auto-detected if not provided)"`
	RepoDir     string `targ:"flag,short=r,desc=Repository root (defaults to current directory)"`
}

func worktreeCreateProject(args worktreeCreateProjectArgs) error {
	repoDir := args.RepoDir
	if repoDir == "" {
		repoDir = "."
	}

	mgr := worktree.NewManager(repoDir)

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

type worktreeMergeProjectArgs struct {
	ProjectName string `targ:"flag,short=p,required,desc=Project name to merge"`
	Onto        string `targ:"flag,short=o,desc=Target branch (defaults to base branch)"`
	RepoDir     string `targ:"flag,short=r,desc=Repository root (defaults to current directory)"`
}

func worktreeMergeProject(args worktreeMergeProjectArgs) error {
	repoDir := args.RepoDir
	if repoDir == "" {
		repoDir = "."
	}

	mgr := worktree.NewManager(repoDir)

	onto := args.Onto
	if onto == "" {
		var err error
		onto, err = mgr.DetectBaseBranch()
		if err != nil {
			return fmt.Errorf("failed to detect base branch: %w", err)
		}
	}

	if err := mgr.MergeProject(args.ProjectName, onto); err != nil {
		return err
	}

	fmt.Printf("Merged project %s onto %s\n", args.ProjectName, onto)
	return nil
}

type worktreeListArgs struct {
	RepoDir string `targ:"flag,short=r,desc=Repository root (defaults to current directory)"`
}

func worktreeList(args worktreeListArgs) error {
	repoDir := args.RepoDir
	if repoDir == "" {
		repoDir = "."
	}

	mgr := worktree.NewManager(repoDir)
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
	if err := w.Flush(); err != nil {
		return err
	}

	return nil
}

type worktreeMergeArgs struct {
	TaskID  string `targ:"flag,short=t,required,desc=Task ID to merge"`
	Onto    string `targ:"flag,short=o,desc=Target branch (defaults to main)"`
	RepoDir string `targ:"flag,short=r,desc=Repository root (defaults to current directory)"`
}

func worktreeMerge(args worktreeMergeArgs) error {
	repoDir := args.RepoDir
	if repoDir == "" {
		repoDir = "."
	}

	mgr := worktree.NewManager(repoDir)

	onto := args.Onto
	if onto == "" {
		var err error
		onto, err = mgr.DetectBaseBranch()
		if err != nil {
			return fmt.Errorf("failed to detect base branch: %w", err)
		}
	}
	if err := mgr.Merge(args.TaskID, onto); err != nil {
		return err
	}

	fmt.Printf("Merged %s onto %s\n", args.TaskID, onto)
	return nil
}

type worktreeCleanupArgs struct {
	TaskID  string `targ:"flag,short=t,required,desc=Task ID to cleanup"`
	RepoDir string `targ:"flag,short=r,desc=Repository root (defaults to current directory)"`
}

func worktreeCleanup(args worktreeCleanupArgs) error {
	repoDir := args.RepoDir
	if repoDir == "" {
		repoDir = "."
	}

	mgr := worktree.NewManager(repoDir)
	if err := mgr.Cleanup(args.TaskID); err != nil {
		return err
	}

	fmt.Printf("Cleaned up worktree for %s\n", args.TaskID)
	return nil
}

type worktreeCleanupAllArgs struct {
	RepoDir string `targ:"flag,short=r,desc=Repository root (defaults to current directory)"`
}

func worktreeCleanupAll(args worktreeCleanupAllArgs) error {
	repoDir := args.RepoDir
	if repoDir == "" {
		repoDir = "."
	}

	mgr := worktree.NewManager(repoDir)
	if err := mgr.CleanupAll(); err != nil {
		return err
	}

	fmt.Println("Cleaned up all worktrees")
	return nil
}
