# Issue 544: Explicit agentName Parameter for runConversationLoopWith

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `agentName string` as an explicit top-level parameter to `runConversationLoopWith`, then remove the `flags agentRunFlags` parameter entirely by also extracting `initialPrompt string`, so Phase 6 dispatcher can call the function without any CLI-layer flags struct.

**Architecture:** Pure refactor in three phases. Phase 1 (RED): add a new export shim and property-based test that fail to compile against the current signature. Phase 2 (GREEN): add `agentName string` to `runConversationLoopWith`, replace `flags.name` uses, fix all call sites. Phase 3 (REFACTOR): replace `flags agentRunFlags` with `initialPrompt string`, remove the `flags` parameter entirely, clean up the now-redundant export shim.

**Tech Stack:** Go, `pgregory.net/rapid` (property-based testing), `github.com/onsi/gomega`, `targ` build tool.

---

## File Map

| File | Change |
|------|--------|
| `internal/cli/cli_agent.go` | Modify `runConversationLoopWith` signature (add `agentName`, then replace `flags`); update `runConversationLoop` call site |
| `internal/cli/export_test.go` | Add `ExportRunConversationLoopWithName` shim (Phase 1); update `ExportRunConversationLoopWith` (Phase 2); collapse shim and clean up (Phase 3) |
| `internal/cli/cli_test.go` | Add `TestRunConversationLoopWith_AgentNamePropagated` rapid property test |

---

## Phase 1 — RED: Property-Based Test That Fails to Compile

### Task 1: Add the new export shim (compile-error RED)

**Files:**
- Modify: `internal/cli/export_test.go`

The shim calls `runConversationLoopWith` with a new `agentName` positional arg that does not yet exist in the function signature. This guarantees a compilation failure.

- [ ] **Step 1: Add `ExportRunConversationLoopWithName` to `export_test.go`**

Open `internal/cli/export_test.go`. After the closing `}` of `ExportRunConversationLoopWith` (currently line 111), add:

```go
// ExportRunConversationLoopWithName is the property-test shim for issue 544.
// It passes agentName as an explicit parameter, separate from any flags construction,
// to verify the param is correctly threaded through runConversationLoopWith.
func ExportRunConversationLoopWithName(
	ctx context.Context,
	agentName, prompt, chatFile, stateFile, claudeBinary string,
	stdout io.Writer,
	promptBuilder func(ctx context.Context, agentName, chatFilePath string, cursor int) (string, error),
	watchForIntent func(ctx context.Context, agentName, chatFilePath string, cursor int) (chat.Message, int, error),
	memFileSelector func(homeDir string, maxFiles int) ([]string, error),
) error {
	flags := agentRunFlags{name: agentName, prompt: prompt, chatFile: chatFile, stateFile: stateFile}
	runner := buildAgentRunner(flags, stateFile, chatFile, stdout)

	return runConversationLoopWith(
		ctx, runner, agentName, flags, chatFile, stateFile,
		//               ^^^^^^^^^ new explicit param — compile error until Phase 2
		claudeBinary, stdout, promptBuilder,
		watchForIntent, memFileSelector,
	)
}
```

### Task 2: Add the property-based test

**Files:**
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Add import for `rapid` and `fmt` if not already present**

The existing `cli_test.go` imports block (lines 3–24) does not include `rapid` or `fmt`. Add them:

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	agentpkg "engram/internal/agent"
	"engram/internal/chat"
	"engram/internal/cli"
)
```

- [ ] **Step 2: Add the property test at the end of the `runConversationLoopWith` section**

After the last test in the `// runConversationLoopWith: INTENT path coverage` section (around line 2825), add:

```go
// TestRunConversationLoopWith_AgentNamePropagated is a property-based test (issue 544).
// Property: for any valid agent name string, runConversationLoopWith completes without error
// when the explicit agentName param matches the agent record in the state file.
// Uses ExportRunConversationLoopWithName (the Phase 1 shim) which calls the new explicit-param
// signature — this test fails to compile until Phase 2 adds the agentName parameter.
func TestRunConversationLoopWith_AgentNamePropagated(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		// Generate a valid agent name: starts with a letter, followed by letters/digits/hyphens.
		agentName := rapid.StringMatching(`[a-z][a-z0-9-]{1,19}`).Draw(rt, "agentName")

		dir := t.TempDir()
		chatFile := filepath.Join(dir, "chat.toml")
		stateFile := filepath.Join(dir, "state.toml")

		g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())
		if err := os.WriteFile(chatFile, []byte(""), 0o600); err != nil {
			return
		}

		// Seed state file with the generated agent name so state writes succeed.
		stateToml := fmt.Sprintf(
			"[[agent]]\nname = %q\npane_id = \"\"\n"+
				"session_id = \"\"\nstate = \"STARTING\"\nspawned_at = 2026-04-06T00:00:00Z\n",
			agentName,
		)
		g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())
		if err := os.WriteFile(stateFile, []byte(stateToml), 0o600); err != nil {
			return
		}

		// Fake claude: emits DONE immediately so the loop exits cleanly.
		fakeClaude := filepath.Join(dir, "claude")
		doneJSON := `{"type":"assistant","session_id":"sess-prop",` +
			`"message":{"content":[{"type":"text","text":"DONE: All done."}]}}`
		script := "#!/bin/sh\nprintf '%s\\n' '" + doneJSON + "'\n"
		g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())
		if err := os.WriteFile(fakeClaude, []byte(script), 0o700); err != nil {
			return
		}

		err := cli.ExportRunConversationLoopWithName(
			context.Background(),
			agentName, "initial prompt", chatFile, stateFile, fakeClaude,
			io.Discard,
			cli.ExportWaitAndBuildPrompt,
			nil, nil,
		)
		g.Expect(err).NotTo(HaveOccurred())
		if err != nil {
			return
		}
	})
}
```

### Task 3: Confirm RED

**Files:** none (read-only verification step)

- [ ] **Step 1: Run tests and confirm compilation failure**

```bash
targ test
```

Expected output contains something like:
```
./export_test.go:NNN: too many arguments in call to runConversationLoopWith
```

or:

```
# engram/internal/cli
internal/cli/export_test.go:...: cannot use agentName (variable of type string) as type ...
```

The important thing is a **compilation error** referencing `runConversationLoopWith` in `export_test.go`. If it compiles and tests pass, the RED phase is broken — re-check that the shim actually passes `agentName` as a distinct extra argument.

- [ ] **Step 2: Commit RED state**

```bash
git add internal/cli/export_test.go internal/cli/cli_test.go
git commit -m "$(cat <<'EOF'
test(cli): RED property test for explicit agentName param in runConversationLoopWith (#544)

Adds ExportRunConversationLoopWithName shim and rapid property test.
Fails to compile until agentName is added as explicit parameter.

AI-Used: [claude]
EOF
)"
```

---

## Phase 2 — GREEN: Add `agentName string` Parameter

### Task 4: Update `runConversationLoopWith` signature

**Files:**
- Modify: `internal/cli/cli_agent.go:1063-1109`

- [ ] **Step 1: Add `agentName string` after `runner`, replace `flags.name` uses**

Current `runConversationLoopWith` (lines 1063–1109):

```go
func runConversationLoopWith(
	ctx context.Context,
	runner claudepkg.Runner,
	flags agentRunFlags,
	chatFilePath, stateFilePath, claudeBinary string,
	stdout io.Writer,
	promptBuilder promptBuilderFunc,
	watchForIntent watchForIntentFunc,
	memFileSelector memFileSelectorFunc,
) error {
	prompt := flags.prompt
	sessionID := ""

	for {
		result, _, cursor, err := runWithinSessionLoop(
			ctx, runner, prompt, sessionID,
			flags.name, chatFilePath, claudeBinary,
			stdout, promptBuilder,
		)
		if err != nil {
			return err
		}

		// No outer watch loop: Phase 4 exit behavior.
		if watchForIntent == nil {
			return nil
		}

		// Phase 5: watch for next intent after session ends.
		prompt, err = watchAndResume(
			ctx, flags.name, chatFilePath, stateFilePath,
			cursor, result, stdout,
			watchForIntent, memFileSelector,
		)
```

Replace with:

```go
func runConversationLoopWith(
	ctx context.Context,
	runner claudepkg.Runner,
	agentName string,
	flags agentRunFlags,
	chatFilePath, stateFilePath, claudeBinary string,
	stdout io.Writer,
	promptBuilder promptBuilderFunc,
	watchForIntent watchForIntentFunc,
	memFileSelector memFileSelectorFunc,
) error {
	prompt := flags.prompt
	sessionID := ""

	for {
		result, _, cursor, err := runWithinSessionLoop(
			ctx, runner, prompt, sessionID,
			agentName, chatFilePath, claudeBinary,
			stdout, promptBuilder,
		)
		if err != nil {
			return err
		}

		// No outer watch loop: Phase 4 exit behavior.
		if watchForIntent == nil {
			return nil
		}

		// Phase 5: watch for next intent after session ends.
		prompt, err = watchAndResume(
			ctx, agentName, chatFilePath, stateFilePath,
			cursor, result, stdout,
			watchForIntent, memFileSelector,
		)
```

- [ ] **Step 2: Update `runConversationLoop` call site (lines 1046–1058)**

Current:

```go
func runConversationLoop(
	ctx context.Context,
	runner claudepkg.Runner,
	flags agentRunFlags,
	chatFilePath, stateFilePath, claudeBinary string,
	stdout io.Writer,
) error {
	return runConversationLoopWith(
		ctx, runner, flags, chatFilePath, stateFilePath,
		claudeBinary, stdout, waitAndBuildPrompt,
		defaultWatchForIntent, defaultMemFileSelector,
	)
}
```

Replace with:

```go
func runConversationLoop(
	ctx context.Context,
	runner claudepkg.Runner,
	flags agentRunFlags,
	chatFilePath, stateFilePath, claudeBinary string,
	stdout io.Writer,
) error {
	return runConversationLoopWith(
		ctx, runner, flags.name, flags, chatFilePath, stateFilePath,
		claudeBinary, stdout, waitAndBuildPrompt,
		defaultWatchForIntent, defaultMemFileSelector,
	)
}
```

### Task 5: Update export shims to pass `agentName` explicitly

**Files:**
- Modify: `internal/cli/export_test.go:95-111`

- [ ] **Step 1: Update `ExportRunConversationLoopWith`**

Current (lines 95–111):

```go
func ExportRunConversationLoopWith(
	ctx context.Context,
	name, prompt, chatFile, stateFile, claudeBinary string,
	stdout io.Writer,
	promptBuilder func(ctx context.Context, agentName, chatFilePath string, turn int) (string, error),
	watchForIntent func(ctx context.Context, agentName, chatFilePath string, cursor int) (chat.Message, int, error),
	memFileSelector func(homeDir string, maxFiles int) ([]string, error),
) error {
	flags := agentRunFlags{name: name, prompt: prompt, chatFile: chatFile, stateFile: stateFile}
	runner := buildAgentRunner(flags, stateFile, chatFile, stdout)

	return runConversationLoopWith(
		ctx, runner, flags, chatFile, stateFile,
		claudeBinary, stdout, promptBuilder,
		watchForIntent, memFileSelector,
	)
}
```

Replace with:

```go
func ExportRunConversationLoopWith(
	ctx context.Context,
	name, prompt, chatFile, stateFile, claudeBinary string,
	stdout io.Writer,
	promptBuilder func(ctx context.Context, agentName, chatFilePath string, turn int) (string, error),
	watchForIntent func(ctx context.Context, agentName, chatFilePath string, cursor int) (chat.Message, int, error),
	memFileSelector func(homeDir string, maxFiles int) ([]string, error),
) error {
	flags := agentRunFlags{name: name, prompt: prompt, chatFile: chatFile, stateFile: stateFile}
	runner := buildAgentRunner(flags, stateFile, chatFile, stdout)

	return runConversationLoopWith(
		ctx, runner, name, flags, chatFile, stateFile,
		claudeBinary, stdout, promptBuilder,
		watchForIntent, memFileSelector,
	)
}
```

- [ ] **Step 2: Update `ExportRunConversationLoopWithName`**

The shim body (added in Task 1) calls `runConversationLoopWith` with `agentName` in position 3. After Phase 2 the signature accepts it there, so no body change is needed — only verify it compiles.

### Task 6: Verify GREEN

- [ ] **Step 1: Run all tests**

```bash
targ test
```

Expected: all tests pass with no compilation errors. If any test fails, check the two `flags.name` replacement sites in `runConversationLoopWith` (lines 1079 and 1093 of the pre-edit file) and the two call sites in `runConversationLoop` and `ExportRunConversationLoopWith`.

- [ ] **Step 2: Commit GREEN state**

```bash
git add internal/cli/cli_agent.go internal/cli/export_test.go
git commit -m "$(cat <<'EOF'
feat(cli): add explicit agentName parameter to runConversationLoopWith (#544)

Adds agentName string as top-level param (Phase 6 extraction point).
Replaces flags.name usage inside the loop. Updates runConversationLoop
and ExportRunConversationLoopWith call sites.

AI-Used: [claude]
EOF
)"
```

---

## Phase 3 — REFACTOR: Remove `flags agentRunFlags`, Extract `initialPrompt`

After Phase 2, `flags` is used in `runConversationLoopWith` for exactly one thing: `flags.prompt` (line 1073: `prompt := flags.prompt`). Extract it as `initialPrompt string` and remove `flags` from the signature entirely.

### Task 7: Replace `flags agentRunFlags` with `initialPrompt string`

**Files:**
- Modify: `internal/cli/cli_agent.go`

- [ ] **Step 1: Update `runConversationLoopWith` signature and body**

Current (after Phase 2):

```go
func runConversationLoopWith(
	ctx context.Context,
	runner claudepkg.Runner,
	agentName string,
	flags agentRunFlags,
	chatFilePath, stateFilePath, claudeBinary string,
	stdout io.Writer,
	promptBuilder promptBuilderFunc,
	watchForIntent watchForIntentFunc,
	memFileSelector memFileSelectorFunc,
) error {
	prompt := flags.prompt
```

Replace with:

```go
func runConversationLoopWith(
	ctx context.Context,
	runner claudepkg.Runner,
	agentName, initialPrompt string,
	chatFilePath, stateFilePath, claudeBinary string,
	stdout io.Writer,
	promptBuilder promptBuilderFunc,
	watchForIntent watchForIntentFunc,
	memFileSelector memFileSelectorFunc,
) error {
	prompt := initialPrompt
```

- [ ] **Step 2: Update `runConversationLoop` call site**

Current (after Phase 2):

```go
return runConversationLoopWith(
	ctx, runner, flags.name, flags, chatFilePath, stateFilePath,
	claudeBinary, stdout, waitAndBuildPrompt,
	defaultWatchForIntent, defaultMemFileSelector,
)
```

Replace with:

```go
return runConversationLoopWith(
	ctx, runner, flags.name, flags.prompt, chatFilePath, stateFilePath,
	claudeBinary, stdout, waitAndBuildPrompt,
	defaultWatchForIntent, defaultMemFileSelector,
)
```

### Task 8: Update export shims and collapse `ExportRunConversationLoopWithName`

**Files:**
- Modify: `internal/cli/export_test.go`

After Phase 3, `ExportRunConversationLoopWithName` and `ExportRunConversationLoopWith` are functionally identical — both pass `(name/agentName, prompt)` and no longer need `flags` inside the call to `runConversationLoopWith`. Remove `ExportRunConversationLoopWithName` and update the property test to use `ExportRunConversationLoopWith`.

- [ ] **Step 1: Update `ExportRunConversationLoopWith` body**

Current (after Phase 2):

```go
func ExportRunConversationLoopWith(
	ctx context.Context,
	name, prompt, chatFile, stateFile, claudeBinary string,
	stdout io.Writer,
	promptBuilder func(ctx context.Context, agentName, chatFilePath string, turn int) (string, error),
	watchForIntent func(ctx context.Context, agentName, chatFilePath string, cursor int) (chat.Message, int, error),
	memFileSelector func(homeDir string, maxFiles int) ([]string, error),
) error {
	flags := agentRunFlags{name: name, prompt: prompt, chatFile: chatFile, stateFile: stateFile}
	runner := buildAgentRunner(flags, stateFile, chatFile, stdout)

	return runConversationLoopWith(
		ctx, runner, name, flags, chatFile, stateFile,
		claudeBinary, stdout, promptBuilder,
		watchForIntent, memFileSelector,
	)
}
```

Replace with:

```go
func ExportRunConversationLoopWith(
	ctx context.Context,
	name, prompt, chatFile, stateFile, claudeBinary string,
	stdout io.Writer,
	promptBuilder func(ctx context.Context, agentName, chatFilePath string, turn int) (string, error),
	watchForIntent func(ctx context.Context, agentName, chatFilePath string, cursor int) (chat.Message, int, error),
	memFileSelector func(homeDir string, maxFiles int) ([]string, error),
) error {
	flags := agentRunFlags{name: name, prompt: prompt, chatFile: chatFile, stateFile: stateFile}
	runner := buildAgentRunner(flags, stateFile, chatFile, stdout)

	return runConversationLoopWith(
		ctx, runner, name, prompt, chatFile, stateFile,
		claudeBinary, stdout, promptBuilder,
		watchForIntent, memFileSelector,
	)
}
```

- [ ] **Step 2: Remove `ExportRunConversationLoopWithName`**

Delete the entire `ExportRunConversationLoopWithName` function added in Task 1. It is now redundant — `ExportRunConversationLoopWith` is the canonical form.

- [ ] **Step 3: Update the property test to use `ExportRunConversationLoopWith`**

In `internal/cli/cli_test.go`, in `TestRunConversationLoopWith_AgentNamePropagated`, change the call from:

```go
err := cli.ExportRunConversationLoopWithName(
    context.Background(),
    agentName, "initial prompt", chatFile, stateFile, fakeClaude,
    io.Discard,
    cli.ExportWaitAndBuildPrompt,
    nil, nil,
)
```

to:

```go
err := cli.ExportRunConversationLoopWith(
    context.Background(),
    agentName, "initial prompt", chatFile, stateFile, fakeClaude,
    io.Discard,
    cli.ExportWaitAndBuildPrompt,
    nil, nil,
)
```

### Task 9: Final quality check and commit

- [ ] **Step 1: Run full quality check**

```bash
targ check-full
```

Expected: no lint errors, no vet errors, coverage thresholds pass. If `targ check-full` reports issues, address them before committing. Do not suppress linter warnings with `//nolint` without first understanding the issue.

- [ ] **Step 2: Commit refactor**

```bash
git add internal/cli/cli_agent.go internal/cli/export_test.go internal/cli/cli_test.go
git commit -m "$(cat <<'EOF'
refactor(cli): remove flags from runConversationLoopWith, extract initialPrompt (#544)

Completes Phase 6 extraction point: runConversationLoopWith now takes
agentName and initialPrompt as explicit strings with no dependency on
agentRunFlags. Collapses ExportRunConversationLoopWithName into the
existing ExportRunConversationLoopWith shim.

AI-Used: [claude]
EOF
)"
```

---

## Self-Review

**Spec coverage check:**
- [x] Add `agentName string` as explicit top-level param → Task 4
- [x] Replace `flags.name` uses (2 occurrences) → Task 4 Step 1
- [x] Update `runConversationLoop` call site → Task 4 Step 2
- [x] Update `ExportRunConversationLoopWith` → Task 5 Step 1
- [x] Property-based test with `rapid` → Task 2
- [x] RED compile failure → Task 3
- [x] Remove `flags agentRunFlags`, extract `initialPrompt` → Task 7
- [x] Clean up redundant shim → Task 8 Steps 2–3
- [x] `targ check-full` → Task 9 Step 1

**Placeholder scan:** No TBD, TODO, or "handle edge cases" language present.

**Type consistency:** `agentName string` and `initialPrompt string` used consistently across Tasks 4, 5, 7, 8. `ExportRunConversationLoopWith` signature unchanged externally (same call-site API for existing tests).

**Note on rapid test temp dirs:** The property test uses `t.TempDir()` inside the `rapid.Check` callback. Each call to `t.TempDir()` creates a distinct directory, so parallel rapid iterations do not share state.
