// Package state manages the project state machine stored as a TOML file.
package state

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

// StateFile is the filename for the project state.
const StateFile = "state.toml"

// Project holds the project identification and phase.
type Project struct {
	Name     string    `toml:"name"`
	Created  time.Time `toml:"created"`
	Phase    string    `toml:"phase"`
	Workflow string    `toml:"workflow"` // new | adopt | align | task
	Issue    string    `toml:"issue,omitempty"`
	RepoDir  string    `toml:"repo_dir,omitempty"` // Repository root for code artifacts
}

// Progress tracks implementation progress.
type Progress struct {
	CurrentTask     string   `toml:"current_task"`
	CurrentSubphase string   `toml:"current_subphase"`
	TasksComplete   int      `toml:"tasks_complete"`
	TasksTotal      int      `toml:"tasks_total"`
	TasksEscalated  int      `toml:"tasks_escalated"`
	CompletedTasks  []string `toml:"completed_tasks,omitempty"`
}

// Conflicts tracks conflict state.
type Conflicts struct {
	Open          int      `toml:"open"`
	BlockingTasks []string `toml:"blocking_tasks"`
}

// Meta tracks meta-audit state.
type Meta struct {
	CorrectionsSinceLastAudit int       `toml:"corrections_since_last_audit"`
	LastMetaAudit             time.Time `toml:"last_meta_audit,omitempty"`
}

// PhaseTransition records a state transition.
type PhaseTransition struct {
	Timestamp time.Time `toml:"timestamp"`
	Phase     string    `toml:"phase"`
}

// ErrorInfo captures details about a failed transition.
type ErrorInfo struct {
	LastPhase   string    `toml:"last_phase"`
	LastTask    string    `toml:"last_task"`
	TargetPhase string    `toml:"target_phase"` // The phase we were trying to transition to
	ErrorType   string    `toml:"error_type"`   // "illegal_transition", "precondition_failed"
	Message     string    `toml:"message"`
	Timestamp   time.Time `toml:"timestamp"`
	RetryCount  int       `toml:"retry_count"`
}

// PairState tracks the state of a PAIR LOOP for a phase or task.
type PairState struct {
	Iteration           int      `toml:"iteration"`
	MaxIterations       int      `toml:"max_iterations"`
	ProducerComplete    bool     `toml:"producer_complete"`
	ProducerTranscript  string   `toml:"producer_transcript,omitempty"`  // Path to saved producer transcript for QA review
	QAVerdict           string   `toml:"qa_verdict,omitempty"`           // approved, improvement-request, escalate-phase, escalate-user
	ImprovementRequest  string   `toml:"improvement_request,omitempty"`  // Feedback from QA if verdict is improvement-request
	SpawnAttempts       int      `toml:"spawn_attempts"`
	FailedModels        []string `toml:"failed_models,omitempty"`
}

// WorktreeState tracks the state of a git worktree for a task.
type WorktreeState struct {
	Path    string    `toml:"path"`
	Branch  string    `toml:"branch"`
	Created time.Time `toml:"created"`
	Status  string    `toml:"status"` // active, merged, failed
}

// State is the complete project state.
type State struct {
	Project   Project                  `toml:"project"`
	Progress  Progress                 `toml:"progress"`
	Conflicts Conflicts                `toml:"conflicts"`
	Meta      Meta                     `toml:"meta"`
	History   []PhaseTransition        `toml:"history"`
	Error     *ErrorInfo               `toml:"error,omitempty"`
	Pairs     map[string]PairState     `toml:"pairs,omitempty"`
	Worktrees map[string]WorktreeState `toml:"worktrees,omitempty"`
}

// InitOpts holds optional parameters for Init.
type InitOpts struct {
	Workflow string // new | adopt | align | task (defaults to "new")
	Issue    string // optional issue ID to link
	RepoDir  string // repository root for code artifacts (optional)
}

// Init creates a new state file in the given directory.
func Init(dir string, name string, now func() time.Time, opts ...InitOpts) (State, error) {
	statePath := filepath.Join(dir, StateFile)

	if _, err := os.Stat(statePath); err == nil {
		return State{}, fmt.Errorf("state file already exists: %s", statePath)
	}

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return State{}, fmt.Errorf("directory does not exist: %s", dir)
	}

	t := now()

	// Apply options
	workflow := "new"
	issue := ""
	repoDir := ""
	if len(opts) > 0 {
		if opts[0].Workflow != "" {
			workflow = opts[0].Workflow
		}
		issue = opts[0].Issue
		repoDir = opts[0].RepoDir
	}

	s := State{
		Project: Project{
			Name:     name,
			Created:  t,
			Phase:    "init",
			Workflow: workflow,
			Issue:    issue,
			RepoDir:  repoDir,
		},
		History: []PhaseTransition{
			{Timestamp: t, Phase: "init"},
		},
	}

	f, err := os.Create(statePath)
	if err != nil {
		return State{}, fmt.Errorf("failed to create state file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := toml.NewEncoder(f).Encode(s); err != nil {
		return State{}, fmt.Errorf("failed to encode state: %w", err)
	}

	return s, nil
}

// TransitionOpts holds optional fields for a state transition.
type TransitionOpts struct {
	Task     string
	Subphase string
	Force    bool   // Bypass precondition checks (not transition graph checks)
	RepoDir  string // Repository root for code-related checks (populated from state if not set)
}

// PreconditionChecker validates preconditions for phase transitions.
type PreconditionChecker interface {
	RequirementsExist(dir string) bool
	RequirementsHaveIDs(dir string) bool
	DesignExists(dir string) bool
	DesignHasIDs(dir string) bool
	TraceValidationPasses(dir string, phase string) bool // phase enables phase-aware validation
	TestsExist(dir string) bool
	TestsFail(dir string) bool
	TestsPass(dir string) bool
	AcceptanceCriteriaComplete(dir, taskID string) bool
	IncompleteAcceptanceCriteria(dir, taskID string) []string // Returns list of incomplete AC items
	UnblockedTasks(dir string, failedTask string) []string    // Returns unblocked tasks excluding the failed one
	RetroExists(dir string) bool                              // Check for retro.md in project dir
	SummaryExists(dir string) bool                            // Check for summary.md in project dir
	IssueACComplete(repoDir, issueID string) bool             // Check if issue's AC are all complete
	IncompleteIssueAC(repoDir, issueID string) []string       // Returns list of incomplete issue AC items
}

// Preconditions maps phases to their required preconditions.
// The function takes dir, opts, and checker so preconditions can access task ID.
var Preconditions = map[string]func(dir string, opts TransitionOpts, checker PreconditionChecker) error{
	"pm-complete": func(dir string, opts TransitionOpts, c PreconditionChecker) error {
		if !c.RequirementsExist(dir) {
			return fmt.Errorf("precondition failed: requirements.md must exist")
		}
		if !c.RequirementsHaveIDs(dir) {
			return fmt.Errorf("precondition failed: requirements.md must contain REQ-NNN IDs")
		}
		return nil
	},
	"design-complete": func(dir string, opts TransitionOpts, c PreconditionChecker) error {
		if !c.DesignExists(dir) {
			return fmt.Errorf("precondition failed: design.md must exist")
		}
		if !c.DesignHasIDs(dir) {
			return fmt.Errorf("precondition failed: design.md must contain DES-NNN IDs")
		}
		return nil
	},
	"architect-complete": func(dir string, opts TransitionOpts, c PreconditionChecker) error {
		if !c.TraceValidationPasses(dir, "architect-complete") {
			return fmt.Errorf("precondition failed: trace validation must pass")
		}
		return nil
	},
	"task-complete": func(dir string, opts TransitionOpts, c PreconditionChecker) error {
		if !c.TraceValidationPasses(dir, "task-complete") {
			return fmt.Errorf("precondition failed: trace validation must pass")
		}
		if opts.Task != "" && !c.AcceptanceCriteriaComplete(dir, opts.Task) {
			return fmt.Errorf("precondition failed: acceptance criteria for %s must be complete", opts.Task)
		}
		return nil
	},
	"tdd-green": func(dir string, opts TransitionOpts, c PreconditionChecker) error {
		// Use repo dir for code-related checks, fallback to project dir
		codeDir := opts.RepoDir
		if codeDir == "" {
			codeDir = dir
		}
		if !c.TestsExist(codeDir) {
			return fmt.Errorf("precondition failed: test files must exist")
		}
		if !c.TestsFail(codeDir) {
			return fmt.Errorf("precondition failed: tests must currently fail")
		}
		return nil
	},
	"tdd-refactor": func(dir string, opts TransitionOpts, c PreconditionChecker) error {
		if !c.TestsPass(dir) {
			return fmt.Errorf("precondition failed: all tests must pass")
		}
		return nil
	},
	"retro-complete": func(dir string, opts TransitionOpts, c PreconditionChecker) error {
		if !c.RetroExists(dir) {
			return fmt.Errorf("precondition failed: retro.md must exist")
		}
		return nil
	},
	"summary-complete": func(dir string, opts TransitionOpts, c PreconditionChecker) error {
		if !c.SummaryExists(dir) {
			return fmt.Errorf("precondition failed: summary.md must exist")
		}
		return nil
	},
	"issue-update": func(dir string, opts TransitionOpts, c PreconditionChecker) error {
		// Read state to get linked issue
		s, err := Get(dir)
		if err != nil {
			return fmt.Errorf("precondition failed: could not read state: %w", err)
		}

		// If no linked issue, skip AC check
		if s.Project.Issue == "" {
			return nil
		}

		// Get repo dir for issue lookup (issues.md is in repo, not project dir)
		repoDir := s.Project.RepoDir
		if repoDir == "" {
			repoDir = dir // Fallback to project dir
		}

		// Check issue AC
		if !c.IssueACComplete(repoDir, s.Project.Issue) {
			return fmt.Errorf("precondition failed: acceptance criteria for %s are incomplete", s.Project.Issue)
		}
		return nil
	},
}

// Transition moves the project to a new phase, validating the transition is legal.
// Writes atomically (temp file + rename) to avoid corruption.
// This variant does not check preconditions - use TransitionWithChecker for that.
func Transition(dir string, to string, opts TransitionOpts, now func() time.Time) (State, error) {
	return TransitionWithChecker(dir, to, opts, now, nil)
}

// TransitionWithChecker moves the project to a new phase, validating both
// the transition graph and preconditions. If checker is nil, preconditions
// are not validated.
func TransitionWithChecker(dir string, to string, opts TransitionOpts, now func() time.Time, checker PreconditionChecker) (State, error) {
	s, err := Get(dir)
	if err != nil {
		return State{}, err
	}

	// Populate RepoDir from state if not explicitly set
	if opts.RepoDir == "" && s.Project.RepoDir != "" {
		opts.RepoDir = s.Project.RepoDir
	}

	from := s.Project.Phase
	t := now()

	if !IsLegalTransition(from, to) {
		targets := LegalTargets(from)
		transitionErr := fmt.Errorf(
			"illegal transition: %s → %s (legal targets: %v)",
			from, to, targets,
		)

		// Capture error in state
		captureError(&s, to, "illegal_transition", transitionErr.Error(), t)
		_ = writeAtomic(dir, s)

		return State{}, transitionErr
	}

	// Check preconditions if checker provided and not forcing
	if checker != nil && !opts.Force {
		if precondCheck, ok := Preconditions[to]; ok {
			if err := precondCheck(dir, opts, checker); err != nil {
				// Capture error in state
				captureError(&s, to, "precondition_failed", err.Error(), t)
				_ = writeAtomic(dir, s)

				return State{}, err
			}
		}
	}

	// Clear error on successful transition
	s.Error = nil

	s.Project.Phase = to
	s.History = append(s.History, PhaseTransition{Timestamp: t, Phase: to})

	if opts.Task != "" {
		s.Progress.CurrentTask = opts.Task
	}

	if opts.Subphase != "" {
		s.Progress.CurrentSubphase = opts.Subphase
	}

	if err := writeAtomic(dir, s); err != nil {
		return State{}, err
	}

	return s, nil
}

// captureError records error details in state, incrementing retry count if same error type.
func captureError(s *State, targetPhase, errorType, message string, t time.Time) {
	retryCount := 1
	if s.Error != nil && s.Error.ErrorType == errorType {
		retryCount = s.Error.RetryCount + 1
	}

	s.Error = &ErrorInfo{
		LastPhase:   s.Project.Phase,
		LastTask:    s.Progress.CurrentTask,
		TargetPhase: targetPhase,
		ErrorType:   errorType,
		Message:     message,
		Timestamp:   t,
		RetryCount:  retryCount,
	}
}

func writeAtomic(dir string, s State) error {
	statePath := filepath.Join(dir, StateFile)
	tmpPath := statePath + ".tmp"

	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	if err := toml.NewEncoder(f).Encode(s); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)

		return fmt.Errorf("failed to encode state: %w", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)

		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, statePath); err != nil {
		_ = os.Remove(tmpPath)

		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Get reads the state file from the given directory.
// Automatically migrates legacy phase="tdd" to phase="tdd-red".
func Get(dir string) (State, error) {
	statePath := filepath.Join(dir, StateFile)

	var s State

	if _, err := toml.DecodeFile(statePath, &s); err != nil {
		return State{}, fmt.Errorf("failed to read state file: %w", err)
	}

	// Migrate legacy TDD phase to TDD-red sub-phase
	if s.Project.Phase == "tdd" {
		s.Project.Phase = "tdd-red"

		// Initialize pair state for tdd-red with iteration 0
		if s.Pairs == nil {
			s.Pairs = make(map[string]PairState)
		}
		// Only initialize if not already set (idempotent)
		if _, exists := s.Pairs["tdd-red"]; !exists {
			s.Pairs["tdd-red"] = PairState{
				Iteration:     0,
				MaxIterations: 3, // Default max iterations
			}
		}

		// Persist migration to disk
		if err := writeAtomic(dir, s); err != nil {
			return State{}, fmt.Errorf("failed to persist migration: %w", err)
		}
	}

	return s, nil
}

// NextResult holds the result of Next().
type NextResult struct {
	Action     string `json:"action"`               // "continue" or "stop"
	NextPhase  string `json:"next_phase,omitempty"` // Next phase when action is continue
	NextTask   string `json:"next_task,omitempty"`  // Next task when action is continue
	Reason     string `json:"reason,omitempty"`     // Reason when action is stop
	Escalation string `json:"escalation,omitempty"` // Escalation ID if reason is escalation_pending
	Details    string `json:"details,omitempty"`    // Details if reason is validation_failed
}

// RecoveryInfo holds information about recovery options after a failure.
type RecoveryInfo struct {
	HasError         bool     `json:"has_error"`
	AvailableActions []string `json:"available_actions,omitempty"`
	LastError        string   `json:"last_error,omitempty"`
}

// GetRecovery returns recovery information for the current state.
func GetRecovery(dir string) RecoveryInfo {
	s, err := Get(dir)
	if err != nil {
		return RecoveryInfo{}
	}

	if s.Error == nil {
		return RecoveryInfo{HasError: false}
	}

	return RecoveryInfo{
		HasError:         true,
		AvailableActions: []string{"retry", "skip", "escalate"},
		LastError:        s.Error.Message,
	}
}

// LastFailedTransition holds info about the last failed transition for retry.
type LastFailedTransition struct {
	FromPhase string
	ToPhase   string
}

// SetOpts holds fields to update via Set().
// Empty strings are ignored (not cleared).
type SetOpts struct {
	Issue    string // Set linked issue ID
	Task     string // Set current task ID
	Workflow string // Set workflow type
}

// Set updates state fields without triggering a phase transition.
func Set(dir string, opts SetOpts) (State, error) {
	s, err := Get(dir)
	if err != nil {
		return State{}, err
	}

	if opts.Issue != "" {
		s.Project.Issue = opts.Issue
	}
	if opts.Task != "" {
		s.Progress.CurrentTask = opts.Task
	}
	if opts.Workflow != "" {
		s.Project.Workflow = opts.Workflow
	}

	if err := writeAtomic(dir, s); err != nil {
		return State{}, err
	}

	return s, nil
}

// SetPair updates the pair loop state for a phase or task.
func SetPair(dir string, key string, ps PairState) (State, error) {
	s, err := Get(dir)
	if err != nil {
		return State{}, err
	}

	if s.Pairs == nil {
		s.Pairs = make(map[string]PairState)
	}
	s.Pairs[key] = ps

	if err := writeAtomic(dir, s); err != nil {
		return State{}, err
	}

	return s, nil
}

// ClearPair removes the pair loop state for a phase or task.
func ClearPair(dir string, key string) (State, error) {
	s, err := Get(dir)
	if err != nil {
		return State{}, err
	}

	if s.Pairs != nil {
		delete(s.Pairs, key)
	}

	if err := writeAtomic(dir, s); err != nil {
		return State{}, err
	}

	return s, nil
}

// SetWorktree updates the worktree state for a task.
func SetWorktree(dir string, key string, ws WorktreeState) (State, error) {
	s, err := Get(dir)
	if err != nil {
		return State{}, err
	}

	if s.Worktrees == nil {
		s.Worktrees = make(map[string]WorktreeState)
	}
	s.Worktrees[key] = ws

	if err := writeAtomic(dir, s); err != nil {
		return State{}, err
	}

	return s, nil
}

// ClearWorktree removes the worktree state for a task.
func ClearWorktree(dir string, key string) (State, error) {
	s, err := Get(dir)
	if err != nil {
		return State{}, err
	}

	if s.Worktrees != nil {
		delete(s.Worktrees, key)
	}

	if err := writeAtomic(dir, s); err != nil {
		return State{}, err
	}

	return s, nil
}

// MarkTaskComplete marks a task as complete and persists to state file.
// This is idempotent - marking the same task complete multiple times
// will not add duplicates.
func MarkTaskComplete(dir string, taskID string) (State, error) {
	s, err := Get(dir)
	if err != nil {
		return State{}, err
	}

	// Check if already complete (idempotent)
	for _, completed := range s.Progress.CompletedTasks {
		if completed == taskID {
			return s, nil
		}
	}

	s.Progress.CompletedTasks = append(s.Progress.CompletedTasks, taskID)

	if err := writeAtomic(dir, s); err != nil {
		return State{}, err
	}

	return s, nil
}

// IsTaskComplete checks if a task has been marked as complete.
func IsTaskComplete(dir string, taskID string) (bool, error) {
	s, err := Get(dir)
	if err != nil {
		return false, err
	}

	for _, completed := range s.Progress.CompletedTasks {
		if completed == taskID {
			return true, nil
		}
	}

	return false, nil
}

// Retry re-attempts the last failed transition.
// This is a simplified implementation that requires the caller to know the target phase.
func Retry(dir string, now func() time.Time, checker PreconditionChecker) (State, error) {
	s, err := Get(dir)
	if err != nil {
		return State{}, err
	}

	if s.Error == nil {
		return State{}, fmt.Errorf("no previous failure to retry")
	}

	// For now, we need to store the target phase in Error
	// Let's enhance ErrorInfo to include TargetPhase
	if s.Error.TargetPhase == "" {
		return State{}, fmt.Errorf("no target phase recorded for retry")
	}

	return TransitionWithChecker(dir, s.Error.TargetPhase, TransitionOpts{}, now, checker)
}

// Next determines the next action based on current state.
// Returns "continue" with next phase/task, or "stop" with reason.
func Next(dir string) NextResult {
	return NextWithChecker(dir, nil)
}

// NextWithChecker determines the next action, optionally checking preconditions.
// If checker is provided and we're at task-audit, validates AC before suggesting task-complete.
func NextWithChecker(dir string, checker PreconditionChecker) NextResult {
	s, err := Get(dir)
	if err != nil {
		return NextResult{
			Action:  "stop",
			Reason:  "state_error",
			Details: err.Error(),
		}
	}

	// Check for error state - but see if there are unblocked tasks
	if s.Error != nil {
		failedTask := s.Error.LastTask
		if checker != nil && failedTask != "" {
			unblockedTasks := checker.UnblockedTasks(dir, failedTask)
			if len(unblockedTasks) > 0 {
				// Continue with unblocked work
				return NextResult{
					Action:    "continue",
					NextPhase: "task-start",
					NextTask:  unblockedTasks[0],
					Details:   "continuing with unblocked work despite failure in " + failedTask,
				}
			}
		}
		return NextResult{
			Action:  "stop",
			Reason:  "error_pending",
			Details: s.Error.Message,
		}
	}

	currentPhase := s.Project.Phase

	// Check for legal targets
	targets := LegalTargets(currentPhase)
	if len(targets) == 0 {
		// Terminal state
		return NextResult{
			Action: "stop",
			Reason: "all_complete",
		}
	}

	// If at task-audit and checker provided, verify AC before suggesting task-complete
	if currentPhase == "task-audit" && checker != nil {
		taskID := s.Progress.CurrentTask
		if taskID != "" && !checker.AcceptanceCriteriaComplete(dir, taskID) {
			incompleteItems := checker.IncompleteAcceptanceCriteria(dir, taskID)
			details := "acceptance criteria for " + taskID + " are incomplete:"
			for _, item := range incompleteItems {
				details += "\n- " + item
			}
			return NextResult{
				Action:  "stop",
				Reason:  "validation_failed",
				Details: details,
			}
		}
	}

	// If at task-complete, check for remaining incomplete tasks
	if currentPhase == "task-complete" && checker != nil {
		// Get unblocked tasks from checker, filter out completed ones
		unblockedTasks := checker.UnblockedTasks(dir, "")
		nextTask := filterFirstIncompleteTask(unblockedTasks, s.Progress.CompletedTasks)
		if nextTask != "" {
			return NextResult{
				Action:    "continue",
				NextPhase: "task-start",
				NextTask:  nextTask,
			}
		}
		// All tasks complete, suggest implementation-complete
		return NextResult{
			Action:    "continue",
			NextPhase: "implementation-complete",
		}
	}

	// Default: continue with first legal target
	return NextResult{
		Action:    "continue",
		NextPhase: targets[0],
		NextTask:  s.Progress.CurrentTask,
	}
}

// filterFirstIncompleteTask returns the first task from candidates that is not in completed.
func filterFirstIncompleteTask(candidates []string, completed []string) string {
	completedSet := make(map[string]bool, len(completed))
	for _, c := range completed {
		completedSet[c] = true
	}

	for _, candidate := range candidates {
		if !completedSet[candidate] {
			return candidate
		}
	}

	return ""
}
