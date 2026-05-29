//go:build targ

package eval

import "context"

// AgentInvocation is one headless agent run request.
type AgentInvocation struct {
	Prompt     string
	Model      string
	Workspace  string // cwd / --add-dir for the agent
	ConfigDir  string // CLAUDE_CONFIG_DIR
	PathPrefix string // prepended to PATH (may be empty)
	VaultPath  string // exported as ENGRAM_VAULT_PATH (the per-run vault clone)
}

// AgentResult is the raw output of a headless run.
type AgentResult struct {
	ResultJSON    []byte // stdout from --output-format json
	TranscriptRaw []byte // the session JSONL bytes (located via session_id)
}

// AgentRunner runs one headless Claude Code agent and collects its output.
type AgentRunner interface {
	Run(ctx context.Context, inv AgentInvocation) (AgentResult, error)
}

// ConfigBuilder prepares a per-arm CLAUDE_CONFIG_DIR (credentials, settings,
// optional engram skills) and returns the dir plus the PATH prefix to use
// (empty when the engram binary should be unreachable).
type ConfigBuilder interface {
	Build(ctx context.Context, arm Arm, root string) (configDir, pathPrefix string, err error)
}

// ResultsWriter appends a run result row to durable storage (JSONL).
type ResultsWriter interface {
	Append(ctx context.Context, r RunResult) error
}

// VaultCloner makes an isolated copy of the live vault per run.
type VaultCloner interface {
	Clone(ctx context.Context, srcVault, destDir string) error
}
