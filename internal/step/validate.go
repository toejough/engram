package step

import (
	"fmt"

	"github.com/toejough/projctl/internal/state"
)

// CompleteOpts holds options for step completion validation.
type CompleteOpts struct {
	// Currently empty, reserved for future use
}

// ValidationResult holds the result of step completion validation.
type ValidationResult struct {
	Valid bool   // Whether the step can be completed
	Error string // Error message if validation failed
}

// Complete validates whether the current step can be completed.
// For plan_produce phase, it verifies that both EnterPlanMode and ExitPlanMode
// tool calls have been made.
func Complete(dir string, opts CompleteOpts) (ValidationResult, error) {
	s, err := state.Get(dir)
	if err != nil {
		return ValidationResult{Valid: false, Error: "failed to read state"}, err
	}

	currentPhase := s.Project.Phase

	// Plan produce phase requires EnterPlanMode and ExitPlanMode tool calls
	if currentPhase == "plan_produce" {
		hasEnter, err := state.HasToolCall(dir, "EnterPlanMode")
		if err != nil {
			return ValidationResult{Valid: false, Error: "failed to check tool calls"}, err
		}

		hasExit, err := state.HasToolCall(dir, "ExitPlanMode")
		if err != nil {
			return ValidationResult{Valid: false, Error: "failed to check tool calls"}, err
		}

		if !hasEnter && !hasExit {
			return ValidationResult{
				Valid: false,
				Error: "plan mode required EnterPlanMode and ExitPlanMode tool calls: both are missing",
			}, fmt.Errorf("plan mode validation failed: EnterPlanMode and ExitPlanMode required")
		}

		if !hasEnter {
			return ValidationResult{
				Valid: false,
				Error: "plan mode required EnterPlanMode tool call: EnterPlanMode is required but missing",
			}, fmt.Errorf("plan mode validation failed: EnterPlanMode required")
		}

		if !hasExit {
			return ValidationResult{
				Valid: false,
				Error: "plan mode required ExitPlanMode tool call: ExitPlanMode is required but missing",
			}, fmt.Errorf("plan mode validation failed: ExitPlanMode required")
		}

		// Both tool calls present - clear them for next phase
		if err := state.ClearToolCalls(dir); err != nil {
			return ValidationResult{Valid: false, Error: "failed to clear tool calls"}, err
		}

		return ValidationResult{Valid: true, Error: ""}, nil
	}

	// Other phases pass validation without tool call checks
	return ValidationResult{Valid: true, Error: ""}, nil
}
