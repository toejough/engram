//go:build targ

package eval

// AgentRunner runs a headless Claude Code agent against a prepared workspace.
// Full definition lands in Task 8.
type AgentRunner interface{}

// ConfigBuilder assembles a Claude Code settings.json for an arm.
// Full definition lands in Task 8.
type ConfigBuilder interface{}

// ResultsWriter persists trial outcomes to JSONL.
// Full definition lands in Task 8.
type ResultsWriter interface{}

// VaultCloner duplicates a vault into a temporary directory.
// Full definition lands in Task 8.
type VaultCloner interface{}
