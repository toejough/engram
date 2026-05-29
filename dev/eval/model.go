//go:build targ

// Package eval is dev-only tooling that measures whether engram memory
// helps a Claude Code agent, by running real headless agents through
// build tasks under different memory configurations ("arms").
package eval

// Arm is one memory configuration under test.
type Arm struct {
	Name         string   // "nothing", "skills-only", "current-state"
	Skills       []string // engram skill names to install ([] = none)
	BinaryOnPATH bool     // whether the engram binary is reachable
}

// Deps is the injected I/O surface (nil-able for pure-logic tests).
type Deps struct {
	Cloner  VaultCloner
	Config  ConfigBuilder
	Runner  AgentRunner
	Results ResultsWriter
}

// RunConfig holds run-level knobs.
type RunConfig struct {
	Trials   int    // trials per scenario
	Model    string // claude model for the agent (e.g. "haiku")
	VaultSrc string // path to the live vault to clone
	OutDir   string // where results JSONL is written
}
