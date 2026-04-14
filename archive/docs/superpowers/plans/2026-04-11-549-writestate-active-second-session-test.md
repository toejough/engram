# WriteState(ACTIVE) Second-Session Test Coverage (#549) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a test that runs the outer watch loop through two sessions and asserts `WriteState("ACTIVE")` is called in both.

**Architecture:** Add a new test export `ExportRunConversationLoopWithStateHook` to `internal/cli/export_test.go` that wraps the runner's `WriteState` with a caller-supplied observer. The observer records every state value passed across all sessions, making it possible to assert ACTIVE was called twice. One new test in `cli_test.go` exercises this export with a two-session fake claude script.

**Tech Stack:** Go, gomega (testify-style assertions via `NewWithT`), os/exec (fake claude shell script), sync.Mutex for parallel-safe state recording.

---

## File Map

| File | Action | Purpose |
|------|--------|---------|
| `internal/cli/export_test.go` | Modify | Add `ExportRunConversationLoopWithStateHook` |
| `internal/cli/cli_test.go` | Modify | Add `TestOuterWatchLoop_WriteStateActiveOnSecondSession` |

No production code is modified. The behavior under test (`WriteState("ACTIVE")` per READY: marker) already works; this plan adds test coverage only.

---

## Phase 1 — RED: Write the failing test

### Task 1: Write the test that calls the not-yet-existing export

**Files:**
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Add the test function**

Insert after `TestOuterWatchLoop_WriteStateSilentAfterSession` (around line 1205 in `cli_test.go`):

```go
func TestOuterWatchLoop_WriteStateActiveOnSecondSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	stateToml := "[[agent]]\nname = \"worker-1\"\npane_id = \"\"\n" +
		"session_id = \"\"\nstate = \"STARTING\"\nspawned_at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	// Fake claude: each invocation emits READY: then DONE:.
	fakeClaude := filepath.Join(dir, "claude")
	readyJSON := `{"type":"assistant","session_id":"sess-abc",` +
		`"message":{"content":[{"type":"text","text":"READY: Online."}]}}`
	doneJSON := `{"type":"assistant","session_id":"sess-abc",` +
		`"message":{"content":[{"type":"text","text":"DONE: All done."}]}}`
	script := "#!/bin/sh\nprintf '%s\\n' '" + readyJSON + "'\nprintf '%s\\n' '" + doneJSON + "'\n"
	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// watchForIntent: first call bridges to session 2, second cancels.
	watchCallCount := 0
	watchForIntent := func(_ context.Context, _, _ string, cursor int) (chat.Message, int, error) {
		watchCallCount++
		if watchCallCount >= 2 {
			cancel()

			return chat.Message{}, 0, context.Canceled
		}

		return chat.Message{
			From: "test-agent",
			To:   "engram-agent",
			Type: "intent",
			Text: "Situation: test. Behavior: respond.",
		}, cursor + 10, nil
	}

	memFileSelector := func(_ string, _ int) ([]string, error) {
		return nil, nil
	}

	stubBuilder := func(_ context.Context, _, _ string, _ int) (string, error) {
		return "Proceed.", nil
	}

	// Record all WriteState calls across both sessions.
	var mu sync.Mutex
	var stateHistory []string
	writeStateObserver := func(state string) error {
		mu.Lock()
		defer mu.Unlock()
		stateHistory = append(stateHistory, state)

		return nil
	}

	err := cli.ExportRunConversationLoopWithStateHook(
		ctx,
		"worker-1", "hello", chatFile, stateFile, fakeClaude,
		io.Discard,
		stubBuilder,
		watchForIntent,
		memFileSelector,
		writeStateObserver,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Both sessions must have called WriteState("ACTIVE") — one per READY: marker.
	mu.Lock()
	defer mu.Unlock()

	activeCount := 0
	for _, state := range stateHistory {
		if state == "ACTIVE" {
			activeCount++
		}
	}

	g.Expect(activeCount).To(Equal(2), "WriteState(ACTIVE) must be called once per session start")
}
```

- [ ] **Step 2: Run the test to confirm RED (compilation failure)**

```bash
cd /Users/joe/repos/personal/engram && targ test 2>&1 | grep -A3 "ExportRunConversationLoopWithStateHook\|FAIL\|undefined"
```

Expected: compilation error — `undefined: cli.ExportRunConversationLoopWithStateHook`. This confirms RED.

---

## Phase 2 — GREEN: Add the export, verify pass

### Task 2: Add `ExportRunConversationLoopWithStateHook` to `export_test.go`

**Files:**
- Modify: `internal/cli/export_test.go`

- [ ] **Step 1: Add the new export function**

Insert after `ExportRunConversationLoopWith` (around line 111 in `export_test.go`):

```go
// ExportRunConversationLoopWithStateHook is like ExportRunConversationLoopWith but wraps
// the runner's WriteState with an observer. The observer is called before every real
// WriteState write; if it returns an error, the write is aborted.
// Use this in tests that need to verify WriteState call patterns across sessions.
func ExportRunConversationLoopWithStateHook(
	ctx context.Context,
	name, prompt, chatFile, stateFile, claudeBinary string,
	stdout io.Writer,
	promptBuilder func(ctx context.Context, agentName, chatFilePath string, turn int) (string, error),
	watchForIntent func(ctx context.Context, agentName, chatFilePath string, cursor int) (chat.Message, int, error),
	memFileSelector func(homeDir string, maxFiles int) ([]string, error),
	writeStateObserver func(state string) error,
) error {
	flags := agentRunFlags{name: name, prompt: prompt, chatFile: chatFile, stateFile: stateFile}
	runner := buildAgentRunner(flags, stateFile, chatFile, stdout)

	if writeStateObserver != nil {
		original := runner.WriteState
		runner.WriteState = func(state string) error {
			if observeErr := writeStateObserver(state); observeErr != nil {
				return observeErr
			}

			if original != nil {
				return original(state)
			}

			return nil
		}
	}

	return runConversationLoopWith(
		ctx, runner, flags, chatFile, stateFile,
		claudeBinary, stdout, promptBuilder,
		watchForIntent, memFileSelector,
	)
}
```

- [ ] **Step 2: Run the test to confirm GREEN**

```bash
cd /Users/joe/repos/personal/engram && targ test 2>&1 | grep -E "PASS|FAIL|TestOuterWatchLoop_WriteStateActiveOnSecondSession"
```

Expected: `TestOuterWatchLoop_WriteStateActiveOnSecondSession` appears in PASS output. No FAIL lines.

- [ ] **Step 3: Run full check to confirm no new issues**

```bash
cd /Users/joe/repos/personal/engram && targ check-full 2>&1 | tail -20
```

Expected: zero new errors or linter issues.

- [ ] **Step 4: Commit the RED→GREEN work**

```bash
cd /Users/joe/repos/personal/engram && git add internal/cli/export_test.go internal/cli/cli_test.go && git commit -m "$(cat <<'EOF'
test(cli): verify WriteState(ACTIVE) called on second outer-loop session (#549)

No test exercised the second watch-loop iteration's ACTIVE state write.
Adds ExportRunConversationLoopWithStateHook and a two-session test.

AI-Used: [claude]
EOF
)"
```

---

## Phase 3 — REFACTOR: Simplify if warranted

### Task 3: Review for duplication and clean up

**Files:**
- Possibly modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Check for shared setup duplication**

Compare the new test with `TestOuterWatchLoop_DoneThenWatchFires` (cli_test.go:784) and `TestOuterWatchLoop_WriteStateSilentAfterSession` (cli_test.go:1149). Both share this scaffold:
- Create temp dir, chat file, state file with STARTING state
- Write a fake claude script with a specific JSON payload
- Define `watchForIntent`, `memFileSelector`, `stubBuilder`
- Call `ExportRunConversationLoopWith*`

If 3 or more tests share 10+ lines of identical setup, extract a helper:

```go
// outerLoopTestSetup creates temp files and a fake claude script for outer-loop tests.
// The script emits the provided JSON lines in order and exits.
// Returns dir, chatFile, stateFile, and fakeClaude path.
func outerLoopTestSetup(t *testing.T, scriptLines ...string) (string, string, string, string) {
	t.Helper()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	stateToml := "[[agent]]\nname = \"worker-1\"\npane_id = \"\"\n" +
		"session_id = \"\"\nstate = \"STARTING\"\nspawned_at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	fakeClaude := filepath.Join(dir, "claude")
	printLines := ""
	for _, line := range scriptLines {
		printLines += "printf '%s\\n' '" + line + "'\n"
	}
	g.Expect(os.WriteFile(fakeClaude, []byte("#!/bin/sh\n"+printLines), 0o700)).To(Succeed())

	return dir, chatFile, stateFile, fakeClaude
}
```

Only extract this helper if it genuinely reduces duplication across 3+ tests. If the tests differ enough in setup, leave them as-is. YAGNI.

- [ ] **Step 2: Run check-full to confirm refactor didn't break anything**

```bash
cd /Users/joe/repos/personal/engram && targ check-full 2>&1 | tail -20
```

Expected: zero errors. If any appear, fix before committing.

- [ ] **Step 3: Commit refactor (only if step 1 produced changes)**

If no duplication was found worth extracting, skip this commit.

```bash
cd /Users/joe/repos/personal/engram && git add internal/cli/cli_test.go && git commit -m "$(cat <<'EOF'
refactor(cli): extract outerLoopTestSetup helper for outer-loop tests (#549)

Reduces duplicated file setup across TestOuterWatchLoop_* tests.

AI-Used: [claude]
EOF
)"
```

---

## Acceptance Criteria Checklist

- [ ] `TestOuterWatchLoop_WriteStateActiveOnSecondSession` exists in `cli_test.go`
- [ ] `ExportRunConversationLoopWithStateHook` exists in `export_test.go`
- [ ] Test asserts `activeCount == 2` — one per session
- [ ] `targ check-full` passes with zero new issues
- [ ] No production code modified
- [ ] Commits use `AI-Used: [claude]` trailer (not Co-Authored-By)
