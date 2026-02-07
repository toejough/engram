package step

import (
	"fmt"
	"strings"
	"time"

	"github.com/toejough/projctl/internal/state"
	"github.com/toejough/projctl/internal/task"
	"github.com/toejough/projctl/internal/worktree"
)

// HandshakeInstruction is prepended to every generated TaskParams.Prompt.
// It instructs the teammate to respond with its model name before doing any work.
const HandshakeInstruction = "First, respond with your model name so I can verify you're running the correct model."

// isQAOnlyPhase returns true for phases that only run QA (no producer).
func isQAOnlyPhase(phase string) bool {
	switch phase {
	case "tdd-red-qa", "tdd-green-qa", "tdd-refactor-qa",
		"commit-red-qa", "commit-green-qa", "commit-refactor-qa":
		return true
	default:
		return false
	}
}

// buildPrompt assembles the full prompt for a spawn action.
func buildPrompt(skillName string, ctx StepContext) string {
	prompt := HandshakeInstruction + "\n\nThen invoke /" + skillName + "."

	if ctx.Issue != "" {
		prompt += "\n\nIssue: " + ctx.Issue
	}

	if ctx.QAFeedback != "" {
		prompt += "\n\nQA feedback:\n" + ctx.QAFeedback
	}

	if len(ctx.PriorArtifacts) > 0 {
		prompt += "\n\nPrior artifacts:"
		for _, a := range ctx.PriorArtifacts {
			prompt += "\n- " + a
		}
	}

	return prompt
}

// StepContext provides contextual information for the action.
type StepContext struct {
	Issue          string   `json:"issue,omitempty"`
	PriorArtifacts []string `json:"prior_artifacts,omitempty"`
	QAFeedback     string   `json:"qa_feedback,omitempty"`
}

// TaskParams holds the exact parameters for a Task tool call.
type TaskParams struct {
	SubagentType string `json:"subagent_type"`
	Name         string `json:"name"`
	Model        string `json:"model"`
	TeamName     string `json:"team_name,omitempty"`
	Prompt       string `json:"prompt,omitempty"`
}

// TaskInfo holds information about a single task for parallel execution.
// Traces to: TASK-1, ARCH-1, DES-1, REQ-3
type TaskInfo struct {
	ID       string  `json:"id"`       // Task identifier (e.g., "TASK-1")
	Command  string  `json:"command"`  // Command to execute (e.g., "projctl run TASK-1")
	Worktree *string `json:"worktree"` // Worktree path (null for sequential, path for parallel)
}

// NextResult holds the structured output of step next.
type NextResult struct {
	Action        string      `json:"action"`                   // spawn-producer, spawn-qa, commit, transition, all-complete
	Skill         string      `json:"skill,omitempty"`          // Skill name
	SkillPath     string      `json:"skill_path,omitempty"`     // Path to SKILL.md
	Model         string      `json:"model,omitempty"`          // Model to use
	Artifact      string      `json:"artifact,omitempty"`       // Artifact produced
	Phase         string      `json:"phase,omitempty"`          // Current or target phase
	Context       StepContext `json:"context"`                  // Contextual information
	TaskParams    *TaskParams `json:"task_params,omitempty"`    // Task tool call parameters (non-nil for spawn actions)
	ExpectedModel string      `json:"expected_model,omitempty"` // Expected model for handshake validation
	Details       string      `json:"details,omitempty"`        // Details for escalation actions
	Tasks         []TaskInfo  `json:"tasks"`                    // Array of unblocked tasks for parallel execution (TASK-1)
}

// CompleteResult holds the input to step complete.
type CompleteResult struct {
	Action        string `json:"action"`                   // What was completed
	Status        string `json:"status"`                   // done, failed
	QAVerdict     string `json:"qa_verdict,omitempty"`     // approved, improvement-request, escalate-phase, escalate-user
	QAFeedback    string `json:"qa_feedback,omitempty"`    // Feedback text from QA
	Phase         string `json:"phase,omitempty"`          // Target phase for transition actions
	ReportedModel string `json:"reported_model,omitempty"` // Model reported by teammate (for failed spawns)
}

// buildSpawnResult constructs a spawn action result with common fields populated.
func buildSpawnResult(action, skill, skillPath, model, artifact, phase string, ctx StepContext, teamName string, tasks []TaskInfo) NextResult {
	return NextResult{
		Action:    action,
		Skill:     skill,
		SkillPath: skillPath,
		Model:     model,
		Artifact:  artifact,
		Phase:     phase,
		Context:   ctx,
		TaskParams: &TaskParams{
			SubagentType: "general-purpose",
			Name:         skill,
			Model:        model,
			TeamName:     teamName,
			Prompt:       buildPrompt(skill, ctx),
		},
		ExpectedModel: model,
		Tasks:         tasks,
	}
}

// handleProducerPhase handles the producer spawn phase logic.
func handleProducerPhase(result NextResult, pair state.PairState, info PhaseInfo, phase string, ctx StepContext, teamName string) (NextResult, error) {
	if pair.SpawnAttempts >= 3 {
		escalated := escalateResult(phase, "producer", info.ProducerModel, pair.FailedModels)
		escalated.Tasks = result.Tasks
		return escalated, nil
	}
	if pair.ImprovementRequest != "" {
		ctx.QAFeedback = pair.ImprovementRequest
	}
	return buildSpawnResult("spawn-producer", info.Producer, info.ProducerPath, info.ProducerModel, info.Artifact, phase, ctx, teamName, result.Tasks), nil
}

// handleQAPhase handles the QA spawn phase logic.
func handleQAPhase(result NextResult, pair state.PairState, info PhaseInfo, phase string, ctx StepContext, teamName string) (NextResult, error) {
	if pair.SpawnAttempts >= 3 {
		escalated := escalateResult(phase, "qa", info.QAModel, pair.FailedModels)
		escalated.Tasks = result.Tasks
		return escalated, nil
	}
	return buildSpawnResult("spawn-qa", info.QA, info.QAPath, info.QAModel, info.Artifact, phase, ctx, teamName, result.Tasks), nil
}

// handleImprovementRequest handles the improvement request phase logic.
func handleImprovementRequest(result NextResult, pair state.PairState, info PhaseInfo, phase string, ctx StepContext, teamName string) (NextResult, error) {
	if pair.Iteration > pair.MaxIterations {
		escalated := escalateIterationResult(phase, pair.Iteration)
		escalated.Tasks = result.Tasks
		return escalated, nil
	}
	if pair.SpawnAttempts >= 3 {
		escalated := escalateResult(phase, "producer", info.ProducerModel, pair.FailedModels)
		escalated.Tasks = result.Tasks
		return escalated, nil
	}
	ctx.QAFeedback = pair.ImprovementRequest
	return buildSpawnResult("spawn-producer", info.Producer, info.ProducerPath, info.ProducerModel, info.Artifact, phase, ctx, teamName, result.Tasks), nil
}

// Next determines the next action based on the current project state.
// It reads the state file, checks the phase registry, and returns structured JSON
// telling the LLM exactly what to do.
// Traces to: TASK-2, ARCH-2, DES-1, DES-6, REQ-1, REQ-5
func Next(dir string) (NextResult, error) {
	s, err := state.Get(dir)
	if err != nil {
		return NextResult{}, fmt.Errorf("failed to read state: %w", err)
	}

	currentPhase := s.Project.Phase

	// Populate parallel tasks array (TASK-2)
	// This detects all unblocked tasks for parallel execution
	result := NextResult{}
	result.Tasks, err = populateTasks(dir)
	if err != nil {
		// If task detection fails, continue with empty array (non-blocking)
		result.Tasks = []TaskInfo{}
	}

	// Check for terminal state
	targets := state.LegalTargets(currentPhase)
	if len(targets) == 0 {
		result.Action = "all-complete"
		result.Phase = currentPhase
		return result, nil
	}

	// Look up phase in registry
	info, registered := Registry.Lookup(currentPhase)
	if !registered {
		// Non-registered phases are transitions (like pm-complete, design-complete, etc.)
		// Just suggest the next transition
		result.Action = "transition"
		result.Phase = targets[0]
		return result, nil
	}

	// Initialize context early for all registered phases
	ctx := StepContext{
		Issue:          s.Project.Issue,
		PriorArtifacts: []string{},
	}

	// Special handling for QA-only phases (tdd-red-qa, tdd-green-qa, etc.)
	// These phases only run QA, no producer
	if isQAOnlyPhase(currentPhase) {
		pair, hasPair := s.Pairs[currentPhase]
		if !hasPair || pair.QAVerdict == "" {
			// No QA verdict yet: spawn QA
			if pair.SpawnAttempts >= 3 {
				escalated := escalateResult(currentPhase, "qa", info.QAModel, pair.FailedModels)
				escalated.Tasks = result.Tasks
				return escalated, nil
			}
			return buildSpawnResult("spawn-qa", info.QA, info.QAPath, info.QAModel, "", currentPhase, ctx, s.Project.Name, result.Tasks), nil
		}
		// QA complete: transition to next phase
		result.Action = "transition"
		result.Phase = targets[0]
		return result, nil
	}

	// Determine sub-phase from pair state
	pair, hasPair := s.Pairs[currentPhase]

	// Sub-phase logic based on pair state
	switch {
	case !hasPair || (!pair.ProducerComplete && pair.QAVerdict == ""):
		return handleProducerPhase(result, pair, info, currentPhase, ctx, s.Project.Name)

	case pair.ProducerComplete && pair.QAVerdict == "":
		return handleQAPhase(result, pair, info, currentPhase, ctx, s.Project.Name)

	case pair.QAVerdict == "improvement-request":
		return handleImprovementRequest(result, pair, info, currentPhase, ctx, s.Project.Name)

	case pair.QAVerdict == "approved":
		result.Action = "commit"
		result.Phase = currentPhase
		result.Context = ctx
		return result, nil

	case pair.QAVerdict == "committed":
		result.Action = "transition"
		result.Phase = targets[0]
		return result, nil

	default:
		result.Action = "transition"
		result.Phase = targets[0]
		return result, nil
	}
}

// Complete records the result of a completed step and advances sub-phase state.
func Complete(dir string, result CompleteResult, now func() time.Time) error {
	s, err := state.Get(dir)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}

	currentPhase := s.Project.Phase

	switch result.Action {
	case "spawn-producer":
		pair := getPair(s, currentPhase)
		if result.Status == "failed" {
			pair.SpawnAttempts++
			pair.FailedModels = append(pair.FailedModels, result.ReportedModel)
			_, err = state.SetPair(dir, currentPhase, pair)
			return err
		}
		// done (or empty for backward compat)
		pair.ProducerComplete = true
		pair.SpawnAttempts = 0
		pair.FailedModels = nil
		if pair.Iteration == 0 {
			pair.Iteration = 1
			pair.MaxIterations = 3
		}
		pair.QAVerdict = ""
		pair.ImprovementRequest = ""
		_, err = state.SetPair(dir, currentPhase, pair)
		return err

	case "spawn-qa":
		pair := getPair(s, currentPhase)
		if result.Status == "failed" {
			pair.SpawnAttempts++
			pair.FailedModels = append(pair.FailedModels, result.ReportedModel)
			_, err = state.SetPair(dir, currentPhase, pair)
			return err
		}
		// done (or empty for backward compat)
		pair.SpawnAttempts = 0
		pair.FailedModels = nil
		pair.QAVerdict = result.QAVerdict
		if result.QAVerdict == "improvement-request" {
			pair.ImprovementRequest = result.QAFeedback
			pair.ProducerComplete = false
			pair.Iteration++
		}
		_, err = state.SetPair(dir, currentPhase, pair)
		return err

	case "commit":
		// Commit: verify QA was approved first
		pair := getPair(s, currentPhase)
		if pair.QAVerdict != "approved" {
			return fmt.Errorf("cannot commit: QA has not approved (verdict: %q)", pair.QAVerdict)
		}
		pair.QAVerdict = "committed"
		_, err = state.SetPair(dir, currentPhase, pair)
		return err

	case "transition":
		// Transition: advance the state machine
		targetPhase := result.Phase
		if targetPhase == "" {
			return fmt.Errorf("transition requires a target phase")
		}
		// Clear pair state for current phase
		_, _ = state.ClearPair(dir, currentPhase)
		_, err = state.Transition(dir, targetPhase, state.TransitionOpts{}, now)
		return err

	default:
		return fmt.Errorf("unknown action: %q", result.Action)
	}
}

// escalateResult builds a NextResult for the escalate-user action.
func escalateResult(phase, subPhase, expectedModel string, failedModels []string) NextResult {
	details := fmt.Sprintf(
		"spawn failed 3 times for %s %s: expected model '%s', got models: ['%s']",
		phase, subPhase, expectedModel, strings.Join(failedModels, "', '"),
	)
	return NextResult{
		Action:  "escalate-user",
		Phase:   phase,
		Details: details,
	}
}

// escalateIterationResult builds a NextResult for max iteration escalation.
func escalateIterationResult(phase string, iterations int) NextResult {
	details := fmt.Sprintf(
		"max iterations (%d) exceeded for phase %s",
		iterations, phase,
	)
	return NextResult{
		Action:  "escalate-user",
		Phase:   phase,
		Details: details,
	}
}

// getPair returns the PairState for the current phase, initializing it if needed.
func getPair(s state.State, phase string) state.PairState {
	if s.Pairs == nil {
		return state.PairState{}
	}
	pair, ok := s.Pairs[phase]
	if !ok {
		return state.PairState{}
	}
	return pair
}

// populateTasks detects all unblocked tasks and creates TaskInfo entries.
// Implements TASK-2: Modify Next() to Return Array of Unblocked Tasks
// Traces to: TASK-2, ARCH-2, DES-1, DES-6, REQ-1, REQ-5
func populateTasks(dir string) ([]TaskInfo, error) {
	// Get all unblocked tasks using task.Parallel()
	unblocked, err := task.Parallel(dir)
	if err != nil {
		return []TaskInfo{}, err
	}

	// If no tasks or only one task, worktree is null (sequential execution)
	// If multiple tasks, assign worktree paths (parallel execution)
	if len(unblocked) == 0 {
		return []TaskInfo{}, nil
	}

	tasks := make([]TaskInfo, 0, len(unblocked))
	mgr := worktree.NewManager(dir)

	for _, taskID := range unblocked {
		taskInfo := TaskInfo{
			ID:      taskID,
			Command: "projctl run " + taskID,
		}

		// TASK-3: Assign worktree paths only for parallel execution (multiple tasks)
		// Traces to: TASK-3, ARCH-3, DES-2, DES-5, REQ-2
		if len(unblocked) > 1 {
			worktreePath := mgr.Path(taskID)
			taskInfo.Worktree = &worktreePath
		}

		tasks = append(tasks, taskInfo)
	}

	return tasks, nil
}
