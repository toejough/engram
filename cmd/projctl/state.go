package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/toejough/projctl/internal/state"
	"github.com/toejough/projctl/internal/task"
)

type stateInitArgs struct {
	Name    string `targ:"flag,short=n,required,desc=Project name"`
	Dir     string `targ:"flag,short=d,desc=Project directory (defaults to .claude/projects/<name>/)"`
	Mode    string `targ:"flag,short=m,desc=Workflow mode: new (default), scoped, align"`
	Issue   string `targ:"flag,short=i,desc=Issue ID to link (e.g. ISSUE-42)"`
	RepoDir string `targ:"flag,short=r,desc=Repository root (auto-detected if not provided)"`
}

func stateInit(args stateInitArgs) error {
	// Default mode is "new"
	mode := args.Mode
	if mode == "" {
		mode = "new"
	}

	// Validate mode
	validModes := map[string]bool{"new": true, "scoped": true, "align": true}
	if !validModes[mode] {
		return fmt.Errorf("unknown mode: %s (valid: new, scoped, align)", mode)
	}

	// Default dir to .claude/projects/<name>/
	dir := args.Dir
	if dir == "" {
		dir = filepath.Join(".claude", "projects", args.Name)
	}

	// Auto-detect repo dir if not provided
	repoDir := args.RepoDir
	if repoDir == "" {
		detected, err := state.FindRepoRoot(".")
		if err == nil {
			repoDir = detected
		}
		// If not in a git repo, repoDir stays empty (that's OK)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	s, err := state.Init(dir, args.Name, time.Now, state.InitOpts{
		Workflow: mode,
		Issue:    args.Issue,
		RepoDir:  repoDir,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Initialized project %q in %s (workflow: %s, phase: %s)\n",
		s.Project.Name, dir, s.Project.Workflow, s.Project.Phase)

	return nil
}

type stateGetArgs struct {
	Dir string `targ:"flag,short=d,required,desc=Project directory"`
}

func stateGet(args stateGetArgs) error {
	s, err := state.Get(args.Dir)
	if err != nil {
		return err
	}

	return toml.NewEncoder(os.Stdout).Encode(s)
}

type stateTransitionArgs struct {
	Dir      string `targ:"flag,short=d,required,desc=Project directory"`
	To       string `targ:"flag,short=t,required,desc=Target phase"`
	Task     string `targ:"flag,desc=Current task ID (e.g. TASK-004)"`
	Subphase string `targ:"flag,desc=Current subphase (e.g. tdd-green)"`
	Force    bool   `targ:"flag,short=f,desc=Force transition, bypassing precondition checks"`
}

func stateTransition(args stateTransitionArgs) error {
	checker := &DefaultChecker{}
	s, err := state.TransitionWithChecker(args.Dir, args.To, state.TransitionOpts{
		Task:     args.Task,
		Subphase: args.Subphase,
		Force:    args.Force,
	}, time.Now, checker)
	if err != nil {
		return err
	}

	fmt.Printf("Transitioned to %q (task: %s, subphase: %s)\n",
		s.Project.Phase,
		s.Progress.CurrentTask,
		s.Progress.CurrentSubphase,
	)

	return nil
}

type stateRetryArgs struct {
	Dir string `targ:"flag,short=d,required,desc=Project directory"`
}

func stateRetry(args stateRetryArgs) error {
	s, err := state.Retry(args.Dir, time.Now, nil)
	if err != nil {
		return err
	}

	fmt.Printf("Retried successfully, now in phase %q\n", s.Project.Phase)

	return nil
}

type stateSetArgs struct {
	Dir      string `targ:"flag,short=d,required,desc=Project directory"`
	Issue    string `targ:"flag,short=i,desc=Issue ID to link (e.g. ISSUE-42)"`
	Task     string `targ:"flag,short=t,desc=Current task ID (e.g. TASK-007)"`
	Workflow string `targ:"flag,short=w,desc=Workflow type (new, scoped, align)"`
}

func stateSet(args stateSetArgs) error {
	if args.Issue == "" && args.Task == "" && args.Workflow == "" {
		return fmt.Errorf("at least one of --issue, --task, or --workflow must be specified")
	}

	s, err := state.Set(args.Dir, state.SetOpts{
		Issue:    args.Issue,
		Task:     args.Task,
		Workflow: args.Workflow,
	})
	if err != nil {
		return err
	}

	fmt.Printf("State updated: workflow=%s, issue=%s, task=%s\n",
		s.Project.Workflow, s.Project.Issue, s.Progress.CurrentTask)

	return nil
}

type stateCompleteArgs struct {
	Dir  string `targ:"flag,short=d,required,desc=Project directory"`
	Task string `targ:"flag,short=t,required,desc=Task ID to mark complete (e.g. TASK-001)"`
}

func stateComplete(args stateCompleteArgs) error {
	// Validate task exists in tasks.md
	result := task.ValidateAcceptanceCriteria(args.Dir, args.Task)
	if result.Error != "" {
		if strings.Contains(result.Error, "not found") {
			return fmt.Errorf("task %s not found in tasks.md", args.Task)
		}
		if strings.Contains(result.Error, "could not read") {
			return fmt.Errorf("tasks.md not found in %s/docs/", args.Dir)
		}
		// Other errors pass through
		return fmt.Errorf("task validation error: %s", result.Error)
	}

	// Mark task complete in state
	s, err := state.MarkTaskComplete(args.Dir, args.Task)
	if err != nil {
		return err
	}

	fmt.Printf("Marked %s complete (total completed: %d)\n",
		args.Task, len(s.Progress.CompletedTasks))

	return nil
}
