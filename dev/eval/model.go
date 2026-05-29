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

// Deps is the injected I/O surface (nil-able for pure-logic tests).
type Deps struct {
	Cloner  VaultCloner
	Config  ConfigBuilder
	Runner  AgentRunner
	Results ResultsWriter
}

// Layer1Metrics are the cost/efficiency signals.
type Layer1Metrics struct {
	DurationMS  int
	Turns       int
	TotalTokens int
	CostUSD     float64
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

// Scenario is one build task the agent performs.
type Scenario struct {
	Name          string
	Prompt        string
	ExpectedVault []string // documented vault lessons this task should exercise
	SuccessCmd    []string // command run in the workspace to check task correctness ([] = none)
	Checks        []BehaviorCheck
}
