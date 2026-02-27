package main

import "github.com/toejough/projctl/internal/worktree"

func worktreeCleanup(args worktree.CleanupArgs) error {
	return worktree.RunCleanup(args)
}

func worktreeCleanupAll(args worktree.CleanupAllArgs) error {
	return worktree.RunCleanupAll(args)
}

func worktreeCreate(args worktree.CreateArgs) error {
	return worktree.RunCreate(args)
}

func worktreeCreateProject(args worktree.CreateProjectArgs) error {
	return worktree.RunCreateProject(args)
}

func worktreeList(args worktree.ListArgs) error {
	return worktree.RunList(args)
}

func worktreeMerge(args worktree.MergeArgs) error {
	return worktree.RunMerge(args)
}

func worktreeMergeProject(args worktree.MergeProjectArgs) error {
	return worktree.RunMergeProject(args)
}
