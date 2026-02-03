package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/toejough/projctl/internal/worktree"
)

type worktreeCreateArgs struct {
	TaskID  string `targ:"flag,short=t,required,desc=Task ID (e.g. TASK-001)"`
	RepoDir string `targ:"flag,short=r,desc=Repository root (defaults to current directory)"`
}

func worktreeCreate(args worktreeCreateArgs) error {
	repoDir := args.RepoDir
	if repoDir == "" {
		repoDir = "."
	}

	mgr := worktree.NewManager(repoDir)
	path, err := mgr.Create(args.TaskID)
	if err != nil {
		return err
	}

	fmt.Printf("Created worktree for %s at %s\n", args.TaskID, path)
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
	fmt.Fprintln(w, "TASK\tBRANCH\tPATH")
	for _, wt := range worktrees {
		fmt.Fprintf(w, "%s\t%s\t%s\n", wt.TaskID, wt.Branch, wt.Path)
	}
	w.Flush()

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

	onto := args.Onto
	if onto == "" {
		onto = "main"
	}

	mgr := worktree.NewManager(repoDir)
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
