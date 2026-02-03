package main

import (
	"encoding/json"
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
	Mode    string `targ:"flag,short=m,desc=Workflow mode: new (default), adopt, align, task"`
	Issue   string `targ:"flag,short=i,desc=Issue ID to link (e.g. ISSUE-042)"`
	RepoDir string `targ:"flag,short=r,desc=Repository root (auto-detected if not provided)"`
}

func stateInit(args stateInitArgs) error {
	// Default mode is "new"
	mode := args.Mode
	if mode == "" {
		mode = "new"
	}

	// Validate mode
	validModes := map[string]bool{"new": true, "adopt": true, "align": true, "task": true}
	if !validModes[mode] {
		return fmt.Errorf("unknown mode: %s (valid: new, adopt, align, task)", mode)
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

	// Auto-transition to the first state for the workflow (if not "new")
	if mode != "new" {
		var firstState string
		switch mode {
		case "adopt":
			firstState = "adopt-explore"
		case "align":
			firstState = "align-explore"
		case "task":
			firstState = "task-implementation"
		}

		s, err = state.Transition(dir, firstState, state.TransitionOpts{}, time.Now)
		if err != nil {
			return fmt.Errorf("failed to transition to %s: %w", firstState, err)
		}
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

type stateNextArgs struct {
	Dir string `targ:"flag,short=d,required,desc=Project directory"`
}

func stateNext(args stateNextArgs) error {
	result := state.Next(args.Dir)

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode result: %w", err)
	}

	fmt.Println(string(data))

	// Return exit code based on action
	if result.Action == "stop" && result.Reason != "all_complete" {
		return fmt.Errorf("stop: %s", result.Reason)
	}

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

type stateRecoveryArgs struct {
	Dir string `targ:"flag,short=d,required,desc=Project directory"`
}

func stateRecovery(args stateRecoveryArgs) error {
	recovery := state.GetRecovery(args.Dir)

	data, err := json.MarshalIndent(recovery, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode recovery info: %w", err)
	}

	fmt.Println(string(data))

	return nil
}

type stateSetArgs struct {
	Dir      string `targ:"flag,short=d,required,desc=Project directory"`
	Issue    string `targ:"flag,short=i,desc=Issue ID to link (e.g. ISSUE-042)"`
	Task     string `targ:"flag,short=t,desc=Current task ID (e.g. TASK-007)"`
	Workflow string `targ:"flag,short=w,desc=Workflow type (new, adopt, align, task)"`
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

type statePairSetArgs struct {
	Dir                string `targ:"flag,short=d,required,desc=Project directory"`
	Key                string `targ:"flag,short=k,required,desc=Phase or task ID (e.g. pm, TASK-007)"`
	Iteration          int    `targ:"flag,short=i,desc=Current iteration (default: 1)"`
	MaxIterations      int    `targ:"flag,short=m,desc=Maximum iterations (default: 3)"`
	ProducerComplete   bool   `targ:"flag,short=p,desc=Producer has completed this iteration"`
	QAVerdict          string `targ:"flag,short=v,desc=QA verdict (approved, improvement-request, escalate-phase, escalate-user)"`
	ImprovementRequest string `targ:"flag,short=r,desc=Feedback if verdict is improvement-request"`
}

func statePairSet(args statePairSetArgs) error {
	// Defaults
	if args.Iteration == 0 {
		args.Iteration = 1
	}
	if args.MaxIterations == 0 {
		args.MaxIterations = 3
	}

	s, err := state.SetPair(args.Dir, args.Key, state.PairState{
		Iteration:          args.Iteration,
		MaxIterations:      args.MaxIterations,
		ProducerComplete:   args.ProducerComplete,
		QAVerdict:          args.QAVerdict,
		ImprovementRequest: args.ImprovementRequest,
	})
	if err != nil {
		return err
	}

	ps := s.Pairs[args.Key]
	fmt.Printf("Pair %q: iteration=%d/%d, producer_complete=%v, qa_verdict=%s\n",
		args.Key, ps.Iteration, ps.MaxIterations, ps.ProducerComplete, ps.QAVerdict)

	return nil
}

type statePairClearArgs struct {
	Dir string `targ:"flag,short=d,required,desc=Project directory"`
	Key string `targ:"flag,short=k,required,desc=Phase or task ID to clear"`
}

func statePairClear(args statePairClearArgs) error {
	_, err := state.ClearPair(args.Dir, args.Key)
	if err != nil {
		return err
	}

	fmt.Printf("Cleared pair loop state for %q\n", args.Key)

	return nil
}

type stateYieldSetArgs struct {
	Dir         string `targ:"flag,short=d,required,desc=Project directory"`
	Type        string `targ:"flag,short=t,required,desc=Yield type (need-user-input, need-context, need-decision, blocked, error)"`
	Agent       string `targ:"flag,short=a,required,desc=Agent that yielded"`
	ContextFile string `targ:"flag,short=c,desc=Path to context file for resumption"`
}

func stateYieldSet(args stateYieldSetArgs) error {
	s, err := state.SetYield(args.Dir, &state.YieldState{
		Pending:     true,
		Type:        args.Type,
		Agent:       args.Agent,
		ContextFile: args.ContextFile,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Yield set: pending=%v, type=%s, agent=%s, context_file=%s\n",
		s.Yield.Pending, s.Yield.Type, s.Yield.Agent, s.Yield.ContextFile)

	return nil
}

type stateYieldClearArgs struct {
	Dir string `targ:"flag,short=d,required,desc=Project directory"`
}

func stateYieldClear(args stateYieldClearArgs) error {
	_, err := state.ClearYield(args.Dir)
	if err != nil {
		return err
	}

	fmt.Println("Cleared pending yield")

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
