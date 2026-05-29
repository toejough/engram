//go:build targ

// Package eval is dev-only tooling that measures whether engram memory
// helps a Claude Code agent, by running real headless agents through
// build tasks under different memory configurations ("arms").
package eval

import "regexp"

// Exported constants.
const (
	ConventionViolation BehaviorKind = "convention_violation" // e.g. used `go test`
	KnownBadPath        BehaviorKind = "known_bad_path"       // did a thing a memory warns against
	ReSearchKnown       BehaviorKind = "re_search_known"      // queried a fact already known
)

// Arm is one memory configuration under test.
type Arm struct {
	Name         string   // "nothing", "skills-only", "current-state"
	Skills       []string // engram skill names to install ([] = none)
	BinaryOnPATH bool     // whether the engram binary is reachable
}

// BehaviorCheck matches an agent's Bash command stream. A match means the
// (undesirable) behavior occurred — lower match-rate is better.
type BehaviorCheck struct {
	Name    string
	Kind    BehaviorKind
	Pattern *regexp.Regexp
}

// BehaviorKind labels what a check detects.
type BehaviorKind string

// BehaviorOutcome records whether a check's behavior occurred in a run.
type BehaviorOutcome struct {
	Name     string       `json:"name"`
	Kind     BehaviorKind `json:"kind"`
	Occurred bool         `json:"occurred"`
}

// CellStats aggregates trials for one (arm × scenario) cell.
type CellStats struct {
	Arm        string
	Scenario   string
	Trials     int
	MeanTurns  float64
	MeanCost   float64
	violations map[string]int // check name → occurrences
}

// ViolationRate is the fraction of trials in which the named check occurred.
func (c CellStats) ViolationRate(check string) float64 {
	if c.Trials == 0 {
		return 0
	}
	return float64(c.violations[check]) / float64(c.Trials)
}

// Deps is the injected I/O surface (nil-able for pure-logic tests).
type Deps struct {
	Cloner  VaultCloner
	Config  ConfigBuilder
	Runner  AgentRunner
	Results ResultsWriter
}

// Layer1Metrics are the cost/efficiency signals.
type Layer1Metrics struct {
	DurationMS  int     `json:"duration_ms"`
	Turns       int     `json:"turns"`
	TotalTokens int     `json:"total_tokens"`
	CostUSD     float64 `json:"cost_usd"`
}

// ResultSummary is the headless `claude -p --output-format json` result.
type ResultSummary struct {
	Type       string  `json:"type"`
	IsError    bool    `json:"is_error"`
	Result     string  `json:"result"`
	SessionID  string  `json:"session_id"`
	DurationMS int     `json:"duration_ms"`
	NumTurns   int     `json:"num_turns"`
	TotalCost  float64 `json:"total_cost_usd"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// RunConfig holds run-level knobs.
type RunConfig struct {
	Trials   int    // trials per scenario
	Model    string // claude model for the agent (e.g. "haiku")
	VaultSrc string // path to the live vault to clone
	OutDir   string // where results JSONL is written
}

// RunResult is one (arm × scenario × trial) outcome.
type RunResult struct {
	Arm       string            `json:"arm"`
	Scenario  string            `json:"scenario"`
	Trial     int               `json:"trial"`
	Layer1    Layer1Metrics     `json:"layer1"`
	Behaviors []BehaviorOutcome `json:"behaviors"`
	TaskOK    bool              `json:"task_ok"`
}

// Scenario is one build task the agent performs.
type Scenario struct {
	Name          string
	Prompt        string
	ExpectedVault []string // documented vault lessons this task should exercise
	SuccessCmd    []string // command run in the workspace to check task correctness ([] = none)
	Checks        []BehaviorCheck
}

// Summary holds all aggregated cells.
type Summary struct {
	cells map[string]CellStats // key: arm + "\x00" + scenario
}
