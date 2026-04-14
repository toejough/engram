# Issue 548: Outer Watch Loop No-Marker Path Test

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a test that exercises `runConversationLoopWith` with a non-nil `watchForIntent` where the inner session emits no markers, confirming the outer loop calls `watchAndResume` and re-enters correctly.

**Architecture:** Single new test in `internal/cli/cli_test.go` using the existing `ExportRunConversationLoopWith` export and a flag-file fake claude binary. No production code changes anticipated. Two misleading comments also fixed in Phase 3.

**Tech Stack:** Go, gomega, `internal/agent` (ParseStateFile), `internal/chat` (chat.Message), `internal/cli` (ExportRunConversationLoopWith)

---

## File Map

| File | Change |
|------|--------|
| `internal/cli/cli_test.go` | Add `TestRunConversationLoopWith_NoMarkers_OuterLoopRewatches`; fix two misleading comments in Phase 3 |

---

## Phase 1 — RED: Write the Test

### Task 1: Add the new test

**Files:**
- Modify: `internal/cli/cli_test.go` (insert after line ~2784, after `TestRunConversationLoopWith_IntentThenDone`)

- [ ] **Step 1.1: Insert the new test after `TestRunConversationLoopWith_IntentThenDone`**

Locate the closing `}` of `TestRunConversationLoopWith_IntentThenDone` (around line 2784) and insert the following test immediately after:

```go
// TestRunConversationLoopWith_NoMarkers_OuterLoopRewatches verifies that when the
// inner session emits no markers, the outer loop calls watchForIntent and re-enters
// the inner loop. The second session emits DONE so the loop exits cleanly.
func TestRunConversationLoopWith_NoMarkers_OuterLoopRewatches(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	stateToml := "[[agent]]\nname = \"worker-1\"\npane_id = \"\"\n" +
		"session_id = \"\"\nstate = \"STARTING\"\nspawned_at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	dir2 := t.TempDir()
	fakeClaude := filepath.Join(dir2, "claude")

	plainJSON := `{"type":"assistant","session_id":"sess-no-markers",` +
		`"message":{"content":[{"type":"text","text":"Here is your answer."}]}}`
	doneJSON := `{"type":"assistant","session_id":"sess-no-markers",` +
		`"message":{"content":[{"type":"text","text":"DONE: All done."}]}}`

	// First call: plain text (no markers). Second call: DONE.
	flagFile := filepath.Join(dir2, "called")
	script := "#!/bin/sh\n" +
		"if [ -f " + flagFile + " ]; then\n" +
		"  printf '%s\\n' '" + doneJSON + "'\nelse\n" +
		"  touch " + flagFile + "\n" +
		"  printf '%s\\n' '" + plainJSON + "'\nfi\n"

	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	// stubPromptBuilder: never called (inner loop exits immediately on no-markers).
	stubBuilder := func(_ context.Context, _, _ string, _ int) (string, error) {
		return "Proceed.", nil
	}

	// stubWatchForIntent: returns a fake intent immediately; counts calls.
	watchCallCount := 0
	stubWatchForIntent := func(_ context.Context, _, _ string, cursor int) (chat.Message, int, error) {
		watchCallCount++

		return chat.Message{
			From: "lead",
			Type: "intent",
			Text: "Resume now.",
		}, cursor + 1, nil
	}

	err := cli.ExportRunConversationLoopWith(
		context.Background(),
		"worker-1", "hello", chatFile, stateFile, fakeClaude,
		io.Discard,
		stubBuilder,
		stubWatchForIntent,
		nil,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// watchForIntent must have been called once (outer loop entered after no-marker session).
	g.Expect(watchCallCount).To(Equal(1))

	// State file must contain state = "SILENT" for worker-1
	// (written by watchAndResume before calling watchForIntent).
	stateData, readErr := os.ReadFile(stateFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	parsedState, parseErr := agentpkg.ParseStateFile(stateData)
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	var worker1State string
	for _, rec := range parsedState.Agents {
		if rec.Name == "worker-1" {
			worker1State = rec.State
		}
	}

	g.Expect(worker1State).To(Equal("SILENT"))
}
```

- [ ] **Step 1.2: Verify imports are present**

The test uses `agentpkg` and `chat`. Check the import block at the top of `cli_test.go` for these lines:

```go
agentpkg "engram/internal/agent"
"engram/internal/chat"
```

If either is missing, add it to the import block.

- [ ] **Step 1.3: Run the new test in isolation**

```bash
cd /Users/joe/repos/personal/engram && targ test 2>&1 | grep -A5 "OuterLoopRewatches\|FAIL\|ok"
```

Expected: the test passes (GREEN immediately — the production code already handles this path). If it fails, record the failure message and proceed to Phase 2.

- [ ] **Step 1.4: Commit Phase 1**

```bash
cd /Users/joe/repos/personal/engram
git add internal/cli/cli_test.go
git commit -m "$(cat <<'EOF'
test(cli): add outer-loop no-marker rewatch test (#548)

Exercises runConversationLoopWith with non-nil watchForIntent where the
inner session emits no markers, verifying the outer loop calls
watchAndResume and re-enters correctly.

AI-Used: [claude]
EOF
)"
```

---

## Phase 2 — GREEN: Fix if Needed

### Task 2: Diagnose and fix if Phase 1 test failed

- [ ] **Step 2.1: Check if Phase 1 passed**

If the test in Task 1 passed: **skip this entire task**. Phase 2 has no work to do.

If the test failed: read the failure message carefully. The most likely failure points are:

- `watchCallCount` is 0 → `runConversationLoopWith` is not calling `watchAndResume` after a no-marker session
- State is not `"SILENT"` → `writeAgentState` is not finding `worker-1` in the state file (check if the agent is registered)
- `err` is non-nil → `watchAndResume` returned an error

- [ ] **Step 2.2 (only if test failed): Inspect `runConversationLoopWith` outer-loop condition**

Read `internal/cli/cli_agent.go` lines ~1075-1100. Verify the condition:

```go
if watchForIntent == nil {
    return nil
}
prompt, err = watchAndResume(...)
```

If the condition is `!= nil` but the call is absent or guarded incorrectly, fix it.

- [ ] **Step 2.3 (only if test failed): Re-run after fix**

```bash
cd /Users/joe/repos/personal/engram && targ test 2>&1 | grep -A5 "OuterLoopRewatches\|FAIL\|ok"
```

Expected: PASS.

- [ ] **Step 2.4 (only if test failed): Commit the fix**

```bash
cd /Users/joe/repos/personal/engram
git add internal/cli/cli_agent.go internal/cli/cli_test.go
git commit -m "$(cat <<'EOF'
fix(cli): outer loop watchAndResume not called on no-marker session (#548)

AI-Used: [claude]
EOF
)"
```

---

## Phase 3 — Refactor: Fix Misleading Comments and Verify

### Task 3: Fix two incorrect comments

- [ ] **Step 3.1: Fix comment in `TestRunAgentRun_FakeClaude_NoMarkers_ExitsClean`**

In `internal/cli/cli_test.go` around line 2205–2208, find:

```go
// TestRunAgentRun_FakeClaude_NoMarkers_ExitsClean verifies conversation ends when claude
// emits no INTENT or DONE markers (treated as complete).
// Uses ExportRunConversationLoopWith with nil watchForIntent to test inner-loop exit;
// the outer watch loop is covered by TestRunConversationLoopWith_IntentThenDone.
```

Replace with:

```go
// TestRunAgentRun_FakeClaude_NoMarkers_ExitsClean verifies conversation ends when claude
// emits no INTENT or DONE markers (treated as complete).
// Uses ExportRunConversationLoopWith with nil watchForIntent to test inner-loop exit;
// the outer watch loop is covered by TestRunConversationLoopWith_NoMarkers_OuterLoopRewatches.
```

- [ ] **Step 3.2: Fix comment in `TestRunConversationLoopWith_IntentThenDone`**

In `internal/cli/cli_test.go` around line 2729–2731, find:

```go
// TestRunConversationLoopWith_IntentThenDone exercises the INTENT → ack-wait → DONE path
// using a fake claude binary and an immediate stub prompt builder.
```

This comment does not mention the outer loop, so no update is needed unless it also references it somewhere. Scan the full comment block to confirm.

If there is a reference to "outer watch loop" in this test's comment, remove it. If there is none, skip this step.

- [ ] **Step 3.3: Run full quality checks**

```bash
cd /Users/joe/repos/personal/engram && targ check-full 2>&1
```

Expected: all tests pass, lint clean, coverage thresholds met.

If any lint errors appear: fix them before committing.

- [ ] **Step 3.4: Commit Phase 3**

```bash
cd /Users/joe/repos/personal/engram
git add internal/cli/cli_test.go
git commit -m "$(cat <<'EOF'
refactor(cli): fix misleading outer-loop coverage comments (#548)

Two test comments incorrectly claimed the outer watch loop was covered
by TestRunConversationLoopWith_IntentThenDone. Updated to reference the
new TestRunConversationLoopWith_NoMarkers_OuterLoopRewatches test.

AI-Used: [claude]
EOF
)"
```
