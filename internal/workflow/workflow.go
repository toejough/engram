// Package workflow provides the declarative TOML-based state machine for projctl.
// All workflow definitions are embedded via go:embed and loaded at init time.
package workflow

import (
	_ "embed"
	"fmt"

	"github.com/BurntSushi/toml"
)

//go:embed workflows.toml
var workflowsTOML string

// StateType describes what the Go interpreter does for a given state.
type StateType string

const (
	StateTypeProduce       StateType = "produce"
	StateTypeQA            StateType = "qa"
	StateTypeDecide        StateType = "decide"
	StateTypeCommit        StateType = "commit"
	StateTypeSelect        StateType = "select"
	StateTypeFork          StateType = "fork"
	StateTypeJoin          StateType = "join"
	StateTypeAssess        StateType = "assess"
	StateTypeWorktree      StateType = "worktree"
	StateTypeLock          StateType = "lock"
	StateTypeRebase        StateType = "rebase"
	StateTypeMerge         StateType = "merge"
	StateTypeCleanup       StateType = "cleanup"
	StateTypeEscalateItem  StateType = "escalate-item"
	StateTypeEscalatePhase StateType = "escalate-phase"
	StateTypeAction        StateType = "action"
	StateTypeApprove       StateType = "approve"
	StateTypeTaskList      StateType = "tasklist"
	StateTypeInterview     StateType = "interview"
)

// StateDef defines a single state in the workflow.
type StateDef struct {
	Type          StateType `toml:"type"`
	Skill         string    `toml:"skill,omitempty"`
	SkillPath     string    `toml:"skill_path,omitempty"`
	FallbackModel string    `toml:"fallback_model,omitempty"`
	Artifact      string    `toml:"artifact,omitempty"`
	IDFormat      string    `toml:"id_format,omitempty"`
}

// StateGroup defines a reusable group of states and transitions.
type StateGroup struct {
	States      map[string]bool     `toml:"states"`
	Transitions map[string][]string `toml:"transitions"`
}

// WorkflowDef defines a complete workflow.
type WorkflowDef struct {
	InitState     string              `toml:"init_state"`
	IncludeGroups []string            `toml:"include_groups"`
	Transitions   map[string][]string `toml:"transitions"`
}

// Config holds the complete parsed workflow configuration.
type Config struct {
	States      map[string]StateDef    `toml:"states"`
	StateGroups map[string]StateGroup  `toml:"state-groups"`
	Workflows   map[string]WorkflowDef `toml:"workflows"`
}

// TransitionsFor returns the merged transition map for a workflow,
// combining the workflow's own transitions with all included state groups.
func (c *Config) TransitionsFor(workflow string) (map[string][]string, error) {
	wf, ok := c.Workflows[workflow]
	if !ok {
		return nil, fmt.Errorf("unknown workflow: %s", workflow)
	}

	merged := make(map[string][]string)

	// Include state group transitions first
	for _, groupName := range wf.IncludeGroups {
		group, ok := c.StateGroups[groupName]
		if !ok {
			return nil, fmt.Errorf("unknown state group: %s", groupName)
		}
		for state, targets := range group.Transitions {
			merged[state] = targets
		}
	}

	// Workflow-specific transitions override group transitions
	for state, targets := range wf.Transitions {
		merged[state] = targets
	}

	// Inject init → init_state so callers don't need special-case logic.
	if wf.InitState != "" {
		merged["init"] = []string{wf.InitState}
	}

	return merged, nil
}

// LookupState returns the state definition for a given state name.
func (c *Config) LookupState(name string) (StateDef, bool) {
	s, ok := c.States[name]
	return s, ok
}

// WorkflowNames returns all defined workflow names.
func (c *Config) WorkflowNames() []string {
	names := make([]string, 0, len(c.Workflows))
	for name := range c.Workflows {
		names = append(names, name)
	}
	return names
}

// InitState returns the initial state for a workflow.
func (c *Config) InitState(workflow string) (string, error) {
	wf, ok := c.Workflows[workflow]
	if !ok {
		return "", fmt.Errorf("unknown workflow: %s", workflow)
	}
	return wf.InitState, nil
}

// IsLegalTransition checks whether transitioning from one state to another
// is allowed within the given workflow.
func (c *Config) IsLegalTransition(from, to, workflow string) bool {
	transitions, err := c.TransitionsFor(workflow)
	if err != nil {
		return false
	}

	for _, target := range transitions[from] {
		if target == to {
			return true
		}
	}
	return false
}

// LegalTargets returns the valid next states for a given state within a workflow.
func (c *Config) LegalTargets(from, workflow string) []string {
	transitions, err := c.TransitionsFor(workflow)
	if err != nil {
		return nil
	}
	return transitions[from]
}

// AllStatesForWorkflow returns all state names reachable in a workflow
// (from included groups + workflow-specific transitions).
func (c *Config) AllStatesForWorkflow(workflow string) ([]string, error) {
	transitions, err := c.TransitionsFor(workflow)
	if err != nil {
		return nil, err
	}

	stateSet := make(map[string]bool)
	for state, targets := range transitions {
		stateSet[state] = true
		for _, t := range targets {
			stateSet[t] = true
		}
	}

	states := make([]string, 0, len(stateSet))
	for s := range stateSet {
		states = append(states, s)
	}
	return states, nil
}

// Load parses the embedded TOML and returns a Config.
func Load() (*Config, error) {
	var cfg Config
	if _, err := toml.Decode(workflowsTOML, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse workflows.toml: %w", err)
	}
	return &cfg, nil
}

// MustLoad loads the config or panics. Used for package-level initialization.
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}

// DefaultConfig is the package-level workflow configuration, loaded from the embedded TOML.
var DefaultConfig = MustLoad()
