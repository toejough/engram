# Fix Hooks stdin Parsing, Fire-and-Forget Stop, Agent Session Filter, and Skill Memory Path

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix four bugs: hook scripts read stdin JSON instead of env vars (#564), agent-stop.sh blocks on exit (#566), stop hook fires for engram-agent sessions (#573), and engram-agent skill is missing the memory directory path (#572).

**Architecture:** Three independent change areas: (1) Go — stamp `ENGRAM_IS_AGENT=1` on the subprocess env in `buildRunClaude` so hooks can identify agent-spawned sessions; (2) Bash — rewrite two hook scripts to parse stdin JSON and skip when `ENGRAM_IS_AGENT=1`; (3) Skill — add memory directory path and discovery guidance to `engram-agent/SKILL.md` via the writing-skills skill.

**Tech Stack:** Go, gomega, pgregory.net/rapid, Bash, targ

---

## File Map

| File | Change |
|------|--------|
| `internal/cli/cli_server.go` | Add `cmd.Env = append(os.Environ(), "ENGRAM_IS_AGENT=1")` after cmd construction |
| `internal/cli/cli_server_test.go` | Add `TestBuildRunClaude_SetsEngramIsAgentEnvVar` |
| `internal/hooks/hooks_test.go` | **New** — Go integration tests for hook shell scripts (exec-planning rule 9) |
| `hooks/user-prompt.sh` | Read `.prompt` from stdin JSON; exit early if `ENGRAM_IS_AGENT=1` |
| `hooks/agent-stop.sh` | Consume stdin; switch `engram intent` → `engram post`; exit early if `ENGRAM_IS_AGENT=1` |
| `hooks/hooks.json` | Stop hook timeout: 30 → 10 |
| `skills/engram-agent/SKILL.md` | Add **Memory Location** section with path, discovery, read, and write guidance |

---

### Task 1: RED — Go test for ENGRAM_IS_AGENT env var

**Files:**
- Modify: `internal/cli/cli_server_test.go`

- [ ] **Step 1: Append the failing test**

Add this test to `internal/cli/cli_server_test.go` (inside `package cli_test`, after the existing tests). The fake binary prints `ENGRAM_IS_AGENT=<value>` from its environment so we can assert the value without depending on any specific shell.

```go
func TestBuildRunClaude_SetsEngramIsAgentEnvVar(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	// Fake binary that prints the ENGRAM_IS_AGENT env var.
	fakeSrc := filepath.Join(dir, "envcheck.go")
	g.Expect(os.WriteFile(fakeSrc, []byte(`package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "ENGRAM_IS_AGENT=") {
			fmt.Print(e)
			return
		}
	}
	fmt.Print("ENGRAM_IS_AGENT not set")
}
`), 0o600)).To(Succeed())

	fakeBinary := filepath.Join(dir, "envcheck")
	if runtime.GOOS == "windows" {
		fakeBinary += ".exe"
	}

	buildCmd := exec.Command("go", "build", "-o", fakeBinary, fakeSrc)
	buildOut, buildErr := buildCmd.CombinedOutput()
	g.Expect(buildErr).NotTo(HaveOccurred(), string(buildOut))

	if buildErr != nil {
		return
	}

	runner := cli.ExportBuildRunClaude(fakeBinary)

	out, err := runner(t.Context(), "prompt", "")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(ContainSubstring("ENGRAM_IS_AGENT=1"))
}
```

- [ ] **Step 2: Run to confirm RED**

```bash
targ test
```

Expected: `FAIL` — output contains `"ENGRAM_IS_AGENT not set"` (env var not set yet).

---

### Task 2: GREEN — Set ENGRAM_IS_AGENT=1 in buildRunClaude

**Files:**
- Modify: `internal/cli/cli_server.go` (the `buildRunClaude` function, line ~43)

- [ ] **Step 1: Add `cmd.Env` after `cmd` construction**

Replace:
```go
cmd := exec.CommandContext(ctx, claudeBinary, args...)

output, runErr := cmd.Output()
```

With:
```go
cmd := exec.CommandContext(ctx, claudeBinary, args...)
cmd.Env = append(os.Environ(), "ENGRAM_IS_AGENT=1")

output, runErr := cmd.Output()
```

The full function after the change:
```go
func buildRunClaude(claudeBinary string) server.RunClaudeFunc {
	return func(ctx context.Context, prompt, sessionID string) (string, error) {
		args := []string{"-p",
			"--dangerously-skip-permissions",
			"--verbose",
			"--output-format=stream-json",
		}
		if sessionID != "" {
			args = append(args, "--resume", sessionID)
		}

		args = append(args, prompt)

		cmd := exec.CommandContext(ctx, claudeBinary, args...)
		cmd.Env = append(os.Environ(), "ENGRAM_IS_AGENT=1")

		output, runErr := cmd.Output()
		if runErr != nil {
			return "", fmt.Errorf("running claude: %w", runErr)
		}

		return string(output), nil
	}
}
```

- [ ] **Step 2: Run to confirm GREEN**

```bash
targ test
```

Expected: all tests pass.

- [ ] **Step 3: Commit**

```bash
git add internal/cli/cli_server.go internal/cli/cli_server_test.go
git commit -m "$(cat <<'EOF'
fix(cli): set ENGRAM_IS_AGENT=1 when spawning engram-agent subprocess

Hook scripts check this var to skip firing for agent-managed sessions,
preventing the stop hook from posting to engram-agent about itself.

Closes #573 (Go side — hook scripts fixed in follow-up commit).

AI-Used: [claude]
EOF
)"
```

---

### Task 3: RED — Hook integration tests

**Files:**
- Create: `internal/hooks/hooks_test.go`

These tests run the actual bash scripts with a controlled stdin and a fake `engram` binary on `PATH`. The fake binary writes its space-separated arguments to `$ARGS_FILE`. Tests verify the scripts call the right engram subcommand with the right arguments, and that they exit early when appropriate.

- [ ] **Step 1: Create `internal/hooks/` directory and `hooks_test.go`**

```go
package hooks_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

// hooksDir returns the absolute path to the hooks/ directory at the repo root.
// Uses runtime.Caller so it works regardless of the working directory at test time.
func hooksDir(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	// filename: .../internal/hooks/hooks_test.go → repo root: ../..
	return filepath.Join(filepath.Dir(filename), "..", "..", "hooks")
}

// buildFakeEngram compiles a Go binary named "engram" into dir.
// When invoked, the fake binary writes its space-separated arguments to $ARGS_FILE.
// Tests set ARGS_FILE in the hook's environment and read it after the hook runs.
func buildFakeEngram(t *testing.T, dir string) {
	t.Helper()
	g := NewWithT(t)

	fakeSrc := filepath.Join(dir, "fakeengram.go")
	g.Expect(os.WriteFile(fakeSrc, []byte(`package main

import (
	"os"
	"strings"
)

func main() {
	argsFile := os.Getenv("ARGS_FILE")
	if argsFile == "" {
		return
	}

	_ = os.WriteFile(argsFile, []byte(strings.Join(os.Args[1:], " ")), 0o600)
}
`), 0o600)).To(Succeed())

	fakeBinary := filepath.Join(dir, "engram")

	buildCmd := exec.Command("go", "build", "-o", fakeBinary, fakeSrc)
	buildOut, buildErr := buildCmd.CombinedOutput()
	g.Expect(buildErr).NotTo(HaveOccurred(), string(buildOut))
}

// setupMarker creates the ENGRAM_DATA directory structure and writes the agent-name
// marker file for workDir. The hook reads this file to learn the current agent name.
// The slug computation mirrors the hook: echo $PWD | tr '/' '-'.
func setupMarker(t *testing.T, engramData, workDir, agentName string) {
	t.Helper()
	g := NewWithT(t)

	chatDir := filepath.Join(engramData, "chat")
	g.Expect(os.MkdirAll(chatDir, 0o755)).To(Succeed())

	slug := strings.ReplaceAll(workDir, "/", "-")
	markerPath := filepath.Join(chatDir, slug+".agent-name")
	g.Expect(os.WriteFile(markerPath, []byte(agentName), 0o600)).To(Succeed())
}

// hookEnv returns a base environment for running hook scripts.
// fakeDir is on PATH so hooks find the fake engram binary.
// argsFile is where the fake engram writes its arguments.
func hookEnv(fakeDir, engramData, argsFile string, extra ...string) []string {
	env := append(os.Environ(),
		"ENGRAM_DATA="+engramData,
		"ARGS_FILE="+argsFile,
		"PATH="+fakeDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)

	return append(env, extra...)
}

// TestUserPromptHook_ReadsSituationFromStdinPrompt verifies that user-prompt.sh
// reads the .prompt field from stdin JSON and passes it to engram intent.
func TestUserPromptHook_ReadsSituationFromStdinPrompt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fakeDir := t.TempDir()
	buildFakeEngram(t, fakeDir)

	argsFile := filepath.Join(fakeDir, "engram.args")
	workDir := t.TempDir()
	engramData := t.TempDir()
	setupMarker(t, engramData, workDir, "lead-1")

	cmd := exec.Command("bash", filepath.Join(hooksDir(t), "user-prompt.sh"))
	cmd.Dir = workDir
	cmd.Stdin = strings.NewReader(`{"prompt":"the user typed this"}`)
	cmd.Env = hookEnv(fakeDir, engramData, argsFile)

	out, err := cmd.CombinedOutput()
	g.Expect(err).NotTo(HaveOccurred(), string(out))

	if err != nil {
		return
	}

	args, readErr := os.ReadFile(argsFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	argsStr := string(args)
	g.Expect(argsStr).To(ContainSubstring("intent"))
	g.Expect(argsStr).To(ContainSubstring("the user typed this"))
}

// TestUserPromptHook_ExitsEarlyForAgentSession verifies that user-prompt.sh
// does NOT call engram when ENGRAM_IS_AGENT=1 is set in the environment.
func TestUserPromptHook_ExitsEarlyForAgentSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fakeDir := t.TempDir()
	buildFakeEngram(t, fakeDir)

	argsFile := filepath.Join(fakeDir, "engram.args")
	workDir := t.TempDir()
	engramData := t.TempDir()
	setupMarker(t, engramData, workDir, "lead-1")

	cmd := exec.Command("bash", filepath.Join(hooksDir(t), "user-prompt.sh"))
	cmd.Dir = workDir
	cmd.Stdin = strings.NewReader(`{"prompt":"should never reach engram"}`)
	cmd.Env = hookEnv(fakeDir, engramData, argsFile, "ENGRAM_IS_AGENT=1")

	out, err := cmd.CombinedOutput()
	g.Expect(err).NotTo(HaveOccurred(), string(out))

	// Hook must exit before calling engram — argsFile must not exist.
	_, statErr := os.Stat(argsFile)
	g.Expect(statErr).To(MatchError(os.ErrNotExist))
}

// TestUserPromptHook_PropAnyPromptPassedAsSituation is a property test verifying
// that for any alphanumeric prompt string, user-prompt.sh passes it to engram.
func TestUserPromptHook_PropAnyPromptPassedAsSituation(t *testing.T) {
	t.Parallel()

	fakeDir := t.TempDir()
	buildFakeEngram(t, fakeDir)

	workDir := t.TempDir()
	engramData := t.TempDir()
	setupMarker(t, engramData, workDir, "lead-1")

	rapid.Check(t, func(rt *rapid.T) {
		// Alphanumeric-only to avoid shell quoting edge cases in the fake binary args check.
		prompt := rapid.StringMatching(`[a-zA-Z0-9]{1,40}`).Draw(rt, "prompt")

		argsFile := filepath.Join(fakeDir, fmt.Sprintf("args-%s.txt", prompt))
		_ = os.Remove(argsFile) // clean up from prior iterations

		stdinPayload := fmt.Sprintf(`{"prompt":%q}`, prompt)

		cmd := exec.Command("bash", filepath.Join(hooksDir(t), "user-prompt.sh"))
		cmd.Dir = workDir
		cmd.Stdin = strings.NewReader(stdinPayload)
		cmd.Env = hookEnv(fakeDir, engramData, argsFile)

		cmdOut, cmdErr := cmd.CombinedOutput()
		if cmdErr != nil {
			rt.Fatalf("hook failed: %v\n%s", cmdErr, cmdOut)
		}

		argsBytes, readErr := os.ReadFile(argsFile)
		if readErr != nil {
			rt.Fatalf("engram was not called (no args file): %v", readErr)
		}

		if !strings.Contains(string(argsBytes), prompt) {
			rt.Fatalf("expected engram args to contain prompt %q, got: %s", prompt, argsBytes)
		}
	})
}

// TestAgentStopHook_PostsLastAssistantMessage verifies that agent-stop.sh calls
// engram post (not engram intent) and passes last_assistant_message as --text.
func TestAgentStopHook_PostsLastAssistantMessage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fakeDir := t.TempDir()
	buildFakeEngram(t, fakeDir)

	argsFile := filepath.Join(fakeDir, "engram.args")
	workDir := t.TempDir()
	engramData := t.TempDir()
	setupMarker(t, engramData, workDir, "lead-1")

	cmd := exec.Command("bash", filepath.Join(hooksDir(t), "agent-stop.sh"))
	cmd.Dir = workDir
	cmd.Stdin = strings.NewReader(`{"last_assistant_message":"the agent finished doing a thing","transcript_path":"/tmp/t.json"}`)
	cmd.Env = hookEnv(fakeDir, engramData, argsFile)

	out, err := cmd.CombinedOutput()
	g.Expect(err).NotTo(HaveOccurred(), string(out))

	if err != nil {
		return
	}

	args, readErr := os.ReadFile(argsFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	argsStr := string(args)
	g.Expect(argsStr).To(ContainSubstring("post"))
	g.Expect(argsStr).NotTo(ContainSubstring("intent"))
	g.Expect(argsStr).To(ContainSubstring("the agent finished doing a thing"))
}

// TestAgentStopHook_ExitsEarlyForAgentSession verifies that agent-stop.sh
// does NOT call engram when ENGRAM_IS_AGENT=1 is set.
func TestAgentStopHook_ExitsEarlyForAgentSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fakeDir := t.TempDir()
	buildFakeEngram(t, fakeDir)

	argsFile := filepath.Join(fakeDir, "engram.args")
	workDir := t.TempDir()
	engramData := t.TempDir()
	setupMarker(t, engramData, workDir, "lead-1")

	cmd := exec.Command("bash", filepath.Join(hooksDir(t), "agent-stop.sh"))
	cmd.Dir = workDir
	cmd.Stdin = strings.NewReader(`{}`)
	cmd.Env = hookEnv(fakeDir, engramData, argsFile, "ENGRAM_IS_AGENT=1")

	out, err := cmd.CombinedOutput()
	g.Expect(err).NotTo(HaveOccurred(), string(out))

	// Hook must exit before calling engram — argsFile must not exist.
	_, statErr := os.Stat(argsFile)
	g.Expect(statErr).To(MatchError(os.ErrNotExist))
}
```

- [ ] **Step 2: Run to confirm RED**

```bash
targ test
```

Expected: all four new hook tests FAIL. `TestUserPromptHook_ReadsSituationFromStdinPrompt` fails because `${PROMPT:-}` is empty (the hook doesn't parse stdin). `TestAgentStopHook_UsesFireAndForget` fails because the hook calls `intent`, not `post`. Both `ExitsEarlyForAgentSession` tests fail because the hooks don't check `ENGRAM_IS_AGENT`.

---

### Task 4: GREEN — Fix hook scripts

**Files:**
- Modify: `hooks/user-prompt.sh`
- Modify: `hooks/agent-stop.sh`
- Modify: `hooks/hooks.json`

- [ ] **Step 1: Rewrite `hooks/user-prompt.sh`**

Replace the entire file with:

```bash
#!/usr/bin/env bash
# UserPromptSubmit hook: posts user prompt to engram API server.
# Uses engram intent (blocking) to get surfaced memories before the next turn.
# Agent name is read from a marker file written by /engram-up.

set -euo pipefail

# Skip for engram-agent sessions (spawned via claude -p --resume by the server).
if [ "${ENGRAM_IS_AGENT:-}" = "1" ]; then
  exit 0
fi

# Claude Code delivers hook data as JSON on stdin.
HOOK_JSON=$(cat)

ENGRAM_DATA="${XDG_DATA_HOME:-$HOME/.local/share}/engram"
SLUG=$(echo "$PWD" | tr '/' '-')
MARKER="$ENGRAM_DATA/chat/${SLUG}.agent-name"

if [ ! -f "$MARKER" ]; then
  exit 0  # Engram not active for this session.
fi

ENGRAM_AGENT_NAME=$(cat "$MARKER")
if [ -z "$ENGRAM_AGENT_NAME" ]; then
  exit 0
fi

SITUATION=$(echo "$HOOK_JSON" | jq -r '.prompt // empty')

engram intent \
  --from "${ENGRAM_AGENT_NAME}:user" \
  --to engram-agent \
  --situation "$SITUATION" \
  --planned-action ""
```

- [ ] **Step 2: Rewrite `hooks/agent-stop.sh`**

Replace the entire file with:

```bash
#!/usr/bin/env bash
# Stop hook: posts the agent's last response to engram for learning.
# Uses engram post (fire-and-forget) — agent is exiting, blocking adds no value.
# Agent name is read from a marker file written by /engram-up.

set -euo pipefail

# Skip for engram-agent sessions (spawned via claude -p --resume by the server).
if [ "${ENGRAM_IS_AGENT:-}" = "1" ]; then
  exit 0
fi

# Claude Code Stop hook provides last_assistant_message directly in stdin JSON.
HOOK_JSON=$(cat)
LAST_MESSAGE=$(echo "$HOOK_JSON" | jq -r '.last_assistant_message // empty')

ENGRAM_DATA="${XDG_DATA_HOME:-$HOME/.local/share}/engram"
SLUG=$(echo "$PWD" | tr '/' '-')
MARKER="$ENGRAM_DATA/chat/${SLUG}.agent-name"

if [ ! -f "$MARKER" ]; then
  exit 0
fi

ENGRAM_AGENT_NAME=$(cat "$MARKER")
if [ -z "$ENGRAM_AGENT_NAME" ]; then
  exit 0
fi

engram post \
  --from "${ENGRAM_AGENT_NAME}" \
  --to engram-agent \
  --text "$LAST_MESSAGE"
```

- [ ] **Step 3: Update `hooks/hooks.json` — Stop timeout 30 → 10**

Replace the `Stop` block only:

```json
"Stop": [
  {
    "hooks": [
      {
        "type": "command",
        "command": "${CLAUDE_PLUGIN_ROOT}/hooks/agent-stop.sh",
        "timeout": 10
      }
    ]
  }
],
```

---

### Task 5: Verify hook tests GREEN + quality check

- [ ] **Step 1: Run tests**

```bash
targ test
```

Expected: all tests pass, including the four new hook integration tests.

- [ ] **Step 2: Run full quality check**

```bash
targ check-full
```

Expected: no lint errors, coverage thresholds met.

If `HOOK_JSON` triggers an unused-variable linter (unlikely for bash, but check any Go linters), note that it's intentionally retained as documentation that we consumed stdin.

- [ ] **Step 3: Commit hook changes**

```bash
git add hooks/user-prompt.sh hooks/agent-stop.sh hooks/hooks.json internal/hooks/hooks_test.go
git commit -m "$(cat <<'EOF'
fix(hooks): parse stdin JSON, fire-and-forget stop, filter agent sessions

user-prompt.sh (#564): read .prompt from stdin JSON via jq instead of
reading $PROMPT env var (Claude Code delivers hook data on stdin, not env).

agent-stop.sh (#564+#566): consume stdin, switch from blocking engram intent
to fire-and-forget engram post. Agent is exiting — no value in blocking 30s
waiting for memories that will never be used.

Both scripts (#573): exit early when ENGRAM_IS_AGENT=1, which the server now
sets when spawning the engram-agent subprocess. Prevents the stop hook from
posting a message to engram-agent about itself.

hooks.json: reduce Stop timeout from 30s to 10s (fire-and-forget needs < 1s).

Adds integration tests in internal/hooks/ that pipe representative JSON
payloads via stdin and verify the downstream engram command and arguments.

Closes #564, #566, #573.

AI-Used: [claude]
EOF
)"
```

---

### Task 6: Fix #572 — Add memory path to engram-agent SKILL.md

**Files:**
- Modify: `skills/engram-agent/SKILL.md`

**REQUIRED:** Use the `superpowers:writing-skills` skill for this task. Do not edit the skill file without invoking it.

- [ ] **Step 1: Invoke writing-skills skill**

Call `superpowers:writing-skills` with the following context:

> Add a **Memory Location** section to `skills/engram-agent/SKILL.md` explaining where memories are stored and how to discover/read/write them. The agent currently reports an empty memory store despite 374 memories on disk because the skill never mentions the directory path.
>
> Add after the existing "Memory Judgment" section:
>
> **Memory Location**
>
> Memories are stored at `${XDG_DATA_HOME:-$HOME/.local/share}/engram/memory/` with two subdirectories:
> - `feedback/` — behavioral feedback memories (TOML files)
> - `facts/` — extracted fact memories (TOML files)
>
> **To count available memories on startup:** list files in both subdirectories.
>
> **To read memories:** The server pre-selects the most semantically relevant files by recency and injects them into your prompt context. Read injected memories from your context. For conflict resolution or full scans, read individual TOML files directly from the paths above.
>
> **To write a memory:** use the path `${XDG_DATA_HOME:-$HOME/.local/share}/engram/memory/<type>/<descriptive-slug>.toml` with the atomic write protocol (per-file lock → write to `.tmp-<slug>.toml` → rename atomically → unlock).

- [ ] **Step 2: Commit via `/commit`**

After the writing-skills skill completes its TDD cycle and updates the file, commit using `/commit`.

---

## Self-Review

**Spec coverage:**
- #564 (stdin JSON parsing): ✓ Tasks 3+4 (user-prompt.sh reads `.prompt`, agent-stop.sh consumes stdin)
- #566 (fire-and-forget stop): ✓ Tasks 3+4 (agent-stop.sh calls `engram post`, timeout reduced)
- #572 (skill memory path): ✓ Task 6
- #573 (hook filter): ✓ Tasks 1+2 (Go env var) + Tasks 3+4 (hook early exit)
- exec-planning rule 9 (hook stdin tests): ✓ Task 3 (four integration tests including one property test)

**Placeholder scan:** None found. All steps include actual code or exact commands.

**Type consistency:** No new types introduced. Shell variable names are consistent across hook scripts.

**Follow-up:** `hooks/subagent-stop.sh` has the same stdin bug (#574, already filed).
