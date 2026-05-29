//go:build targ

package eval

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// KeychainCredentialAvailable reports whether the Claude Code keychain
// credential can be read (used to skip integration tests where it's absent).
func KeychainCredentialAvailable() bool {
	out, err := readKeychainCredential(context.Background())
	return err == nil && len(bytes.TrimSpace(out)) > 0
}

// NewJSONLResultsWriter appends run results to a JSONL file.
func NewJSONLResultsWriter(path string) ResultsWriter {
	return &jsonlResultsWriter{path: path}
}

// NewOSAgentRunner runs headless claude and collects result JSON + transcript.
func NewOSAgentRunner() AgentRunner { return osAgentRunner{} }

// NewOSConfigBuilder returns a ConfigBuilder. enginePath is the engram binary
// to expose on PATH for binary-bearing arms.
func NewOSConfigBuilder(enginePath string) ConfigBuilder {
	return &osConfigBuilder{enginePath: enginePath}
}

// NewOSVaultCloner returns a vault cloner (plain cp -R).
func NewOSVaultCloner() VaultCloner { return osVaultCloner{} }

// unexported constants.
const (
	keychainService = "Claude Code-credentials"
)

// --- ResultsWriter ---

type jsonlResultsWriter struct{ path string }

func (w *jsonlResultsWriter) Append(_ context.Context, r RunResult) error {
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644) //nolint:gosec // 0644 is intentional for results file readable by user
	if err != nil {
		return fmt.Errorf("opening results file: %w", err)
	}
	defer f.Close()

	data, err := marshalResult(r)
	if err != nil {
		return err
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("appending result: %w", err)
	}

	return nil
}

// --- AgentRunner ---

type osAgentRunner struct{}

func (osAgentRunner) Run(ctx context.Context, inv AgentInvocation) (AgentResult, error) {
	args := []string{
		"-p", inv.Prompt,
		"--output-format", "json",
		"--model", inv.Model,
		"--add-dir", inv.Workspace,
		"--permission-mode", "bypassPermissions",
	}
	cmd := exec.CommandContext(ctx, "claude", args...) //nolint:gosec // args are constructed from trusted eval config
	cmd.Dir = inv.Workspace
	cmd.Env = agentEnv(inv)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return AgentResult{}, fmt.Errorf("running claude: %w", err)
	}

	rs, err := ParseResult(stdout.Bytes())
	if err != nil {
		return AgentResult{}, fmt.Errorf("parsing agent result: %w", err)
	}

	transcript, terr := readSessionTranscript(inv.ConfigDir, rs.SessionID)
	if terr != nil {
		transcript = nil // non-fatal: behavioral scoring degrades, cost metrics survive
	}

	return AgentResult{ResultJSON: stdout.Bytes(), TranscriptRaw: transcript}, nil
}

// --- ConfigBuilder ---

type osConfigBuilder struct{ enginePath string }

func (b *osConfigBuilder) Build(ctx context.Context, arm Arm, root string) (string, string, error) {
	cfgDir := filepath.Join(root, "cfg-"+arm.Name)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil { //nolint:gosec // 0755 is intentional for config dir
		return "", "", fmt.Errorf("mkdir config: %w", err)
	}

	creds, err := readKeychainCredential(ctx)
	if err != nil {
		return "", "", err
	}

	if err := os.WriteFile(filepath.Join(cfgDir, ".credentials.json"), creds, 0o600); err != nil {
		return "", "", fmt.Errorf("writing credentials: %w", err)
	}

	home, _ := os.UserHomeDir()
	if data, rerr := os.ReadFile(filepath.Join(home, ".claude", "settings.json")); rerr == nil {
		_ = os.WriteFile(filepath.Join(cfgDir, "settings.json"), data, 0o644) //nolint:gosec // 0644 is intentional for settings
	}

	if len(arm.Skills) > 0 {
		// cp -R cannot create the intermediate skills/ dir, so make it first.
		if err := os.MkdirAll(filepath.Join(cfgDir, "skills"), 0o755); err != nil { //nolint:gosec // 0755 intentional for skills dir
			return "", "", fmt.Errorf("mkdir skills dir: %w", err)
		}
	}

	for _, skill := range arm.Skills {
		src := filepath.Join(home, ".claude", "skills", skill)
		dst := filepath.Join(cfgDir, "skills", skill)
		if err := copyTree(ctx, src, dst); err != nil {
			return "", "", fmt.Errorf("copying skill %q: %w", skill, err)
		}
	}

	pathPrefix := ""
	if arm.BinaryOnPATH {
		pathPrefix = filepath.Dir(b.enginePath)
	}

	return cfgDir, pathPrefix, nil
}

// --- VaultCloner ---

type osVaultCloner struct{}

func (osVaultCloner) Clone(ctx context.Context, srcVault, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil { //nolint:gosec // 0755 is intentional for vault clone dir
		return fmt.Errorf("mkdir vault dest: %w", err)
	}

	cmd := exec.CommandContext(ctx, "cp", "-R", srcVault+string(os.PathSeparator)+".", destDir) //nolint:gosec // srcVault/destDir are trusted caller-supplied paths
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cloning vault: %w: %s", err, out)
	}

	return nil
}

func agentEnv(inv AgentInvocation) []string {
	env := append([]string{}, os.Environ()...)
	env = append(env, "CLAUDE_CONFIG_DIR="+inv.ConfigDir)

	if inv.VaultPath != "" {
		env = append(env, "ENGRAM_VAULT_PATH="+inv.VaultPath)
	}

	if inv.PathPrefix != "" {
		env = append(env, "PATH="+inv.PathPrefix+string(os.PathListSeparator)+os.Getenv("PATH"))
	}

	return env
}

func copyTree(ctx context.Context, src, dst string) error {
	if err := exec.CommandContext(ctx, "cp", "-R", src, dst).Run(); err != nil { //nolint:gosec // src/dst are constructed from trusted config paths
		return fmt.Errorf("copying tree %s: %w", src, err)
	}

	return nil
}

func readKeychainCredential(ctx context.Context) ([]byte, error) {
	out, err := exec.CommandContext(ctx, "security", //nolint:gosec // fixed args, no user input
		"find-generic-password", "-s", keychainService, "-w").Output()
	if err != nil {
		return nil, fmt.Errorf("reading keychain credentials: %w", err)
	}

	return out, nil
}

// readSessionTranscript finds <configDir>/projects/<cwd-slug>/<session>.jsonl
// by walking projects/ for a file named <session>.jsonl.
func readSessionTranscript(configDir, sessionID string) ([]byte, error) {
	root := filepath.Join(configDir, "projects")

	var found string

	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error { //nolint:gosec // configDir is a trusted eval-managed directory
		if err == nil && info != nil && !info.IsDir() && info.Name() == sessionID+".jsonl" {
			found = p
		}

		return nil
	})

	if found == "" {
		return nil, fmt.Errorf("session transcript %s.jsonl not found under %s", sessionID, root)
	}

	data, err := os.ReadFile(found) //nolint:gosec // found is resolved from a trusted eval-managed directory
	if err != nil {
		return nil, fmt.Errorf("reading session transcript: %w", err)
	}

	return data, nil
}
