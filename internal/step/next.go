package step

import (
	"fmt"
	"strings"
	"time"

	"github.com/toejough/projctl/internal/state"
)

// HandshakeInstruction is prepended to every generated TaskParams.Prompt.
// It instructs the teammate to respond with its model name before doing any work.
const HandshakeInstruction = "First, respond with your model name so I can verify you're running the correct model."

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

// NextResult holds the structured output of step next.
type NextResult struct {
	Action        string      `json:"action"`                    // spawn-producer, spawn-qa, commit, transition, all-complete
	Skill         string      `json:"skill,omitempty"`           // Skill name
	SkillPath     string      `json:"skill_path,omitempty"`      // Path to SKILL.md
	Model         string      `json:"model,omitempty"`           // Model to use
	Artifact      string      `json:"artifact,omitempty"`        // Artifact produced
	Phase         string      `json:"phase,omitempty"`           // Current or target phase
	Context       StepContext `json:"context"`                   // Contextual information
	TaskParams    *TaskParams `json:"task_params,omitempty"`     // Task tool call parameters (non-nil for spawn actions)
	ExpectedModel string     `json:"expected_model,omitempty"`  // Expected model for handshake validation
	Details       string     `json:"details,omitempty"`         // Details for escalation actions
}

// CompleteResult holds the input to step complete.
type CompleteResult struct {
	Action        string `json:"action"`                  // What was completed
	Status        string `json:"status"`                  // done, failed
	QAVerdict     string `json:"qa_verdict,omitempty"`     // approved, improvement-request, escalate-phase, escalate-user
	QAFeedback    string `json:"qa_feedback,omitempty"`    // Feedback text from QA
	Phase         string `json:"phase,omitempty"`          // Target phase for transition actions
	ReportedModel string `json:"reported_model,omitempty"` // Model reported by teammate (for failed spawns)
}

// Next determines the next action based on the current project state.
// It reads the state file, checks the phase registry, and returns structured JSON
// telling the LLM exactly what to do.
func Next(dir string) (NextResult, error) {
	s, err := state.Get(dir)
	if err != nil {
		return NextResult{}, fmt.Errorf("failed to read state: %w", err)
	}

	currentPhase := s.Project.Phase

	// Check for terminal state
	targets := state.LegalTargets(currentPhase)
	if len(targets) == 0 {
		return NextResult{
			Action: "all-complete",
			Phase:  currentPhase,
		}, nil
	}

	// Look up phase in registry
	info, registered := Registry.Lookup(currentPhase)
	if !registered {
		// Non-registered phases are transitions (like pm-complete, design-complete, etc.)
		// Just suggest the next transition
		return NextResult{
			Action: "transition",
			Phase:  targets[0],
		}, nil
	}

	// Determine sub-phase from pair state
	pair, hasPair := s.Pairs[currentPhase]

	ctx := StepContext{
		Issue:          s.Project.Issue,
		PriorArtifacts: []string{},
	}

	// Sub-phase logic based on pair state
	switch {
	case !hasPair || (!pair.ProducerComplete && pair.QAVerdict == ""):
		// No pair state or producer not done yet: spawn producer
		if pair.SpawnAttempts >= 3 {
			return escalateResult(currentPhase, "producer", info.ProducerModel, pair.FailedModels), nil
		}
		if pair.ImprovementRequest != "" {
			ctx.QAFeedback = pair.ImprovementRequest
		}
		return NextResult{
			Action:    "spawn-producer",
			Skill:     info.Producer,
			SkillPath: info.ProducerPath,
			Model:     info.ProducerModel,
			Artifact:  info.Artifact,
			Phase:     currentPhase,
			Context:   ctx,
			TaskParams: &TaskParams{
				SubagentType: "general-purpose",
				Name:         info.Producer,
				Model:        info.ProducerModel,
				TeamName:     s.Project.Name,
				Prompt:       buildPrompt(info.Producer, ctx),
			},
			ExpectedModel: info.ProducerModel,
		}, nil

	case pair.ProducerComplete && pair.QAVerdict == "":
		// Producer done, no QA yet: spawn QA
		if pair.SpawnAttempts >= 3 {
			return escalateResult(currentPhase, "qa", info.QAModel, pair.FailedModels), nil
		}
		return NextResult{
			Action:    "spawn-qa",
			Skill:     info.QA,
			SkillPath: info.QAPath,
			Model:     info.QAModel,
			Artifact:  info.Artifact,
			Phase:     currentPhase,
			Context:   ctx,
			TaskParams: &TaskParams{
				SubagentType: "general-purpose",
				Name:         info.QA,
				Model:        info.QAModel,
				TeamName:     s.Project.Name,
				Prompt:       buildPrompt(info.QA, ctx),
			},
			ExpectedModel: info.QAModel,
		}, nil

	case pair.QAVerdict == "improvement-request":
		// QA requested improvements: re-run producer with feedback
		if pair.SpawnAttempts >= 3 {
			return escalateResult(currentPhase, "producer", info.ProducerModel, pair.FailedModels), nil
		}
		ctx.QAFeedback = pair.ImprovementRequest
		return NextResult{
			Action:    "spawn-producer",
			Skill:     info.Producer,
			SkillPath: info.ProducerPath,
			Model:     info.ProducerModel,
			Artifact:  info.Artifact,
			Phase:     currentPhase,
			Context:   ctx,
			TaskParams: &TaskParams{
				SubagentType: "general-purpose",
				Name:         info.Producer,
				Model:        info.ProducerModel,
				TeamName:     s.Project.Name,
				Prompt:       buildPrompt(info.Producer, ctx),
			},
			ExpectedModel: info.ProducerModel,
		}, nil

	case pair.QAVerdict == "approved":
		// QA approved: commit
		return NextResult{
			Action:  "commit",
			Phase:   currentPhase,
			Context: ctx,
		}, nil

	case pair.QAVerdict == "committed":
		// Committed: transition to completion phase
		return NextResult{
			Action: "transition",
			Phase:  info.CompletionPhase,
		}, nil

	default:
		return NextResult{
			Action: "transition",
			Phase:  targets[0],
		}, nil
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
