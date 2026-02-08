package state

import "github.com/toejough/projctl/internal/workflow"

// IsLegalTransition checks whether transitioning from one state to another
// is allowed within the given workflow.
func IsLegalTransition(from, to, wf string) bool {
	return workflow.DefaultConfig.IsLegalTransition(from, to, wf)
}

// LegalTargets returns the valid next states for a given state within a workflow.
func LegalTargets(from, wf string) []string {
	return workflow.DefaultConfig.LegalTargets(from, wf)
}

// WorkflowInitState returns the initial state for a workflow.
func WorkflowInitState(wf string) (string, error) {
	return workflow.DefaultConfig.InitState(wf)
}

// TransitionsForWorkflow returns the merged transition map for a workflow.
func TransitionsForWorkflow(wf string) map[string][]string {
	transitions, err := workflow.DefaultConfig.TransitionsFor(wf)
	if err != nil {
		return nil
	}
	return transitions
}
