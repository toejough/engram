# Remove MaxConcurrentWorkers Cap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the undocumented 3-worker concurrency cap from the binary so engram-tmux-lead can spawn up to 8 workers without rejection.

**Architecture:** Pure deletion — remove the `MaxConcurrentWorkers` constant, the `errWorkerQueueFull` sentinel, and the enforcement gate in `preSpawnGuards`. The test for the removed behavior is also deleted. A RED test is added first to prove the cap exists, then removed cap makes it GREEN.

**Tech Stack:** Go, gomega, `targ check-full` for verification.

---

## File Map

| File | Change |
|------|--------|
| `internal/cli/cli_test.go` | Add RED test (step 1), delete old MaxWorkers test (step 3) |
| `internal/agent/agent.go` | Remove `MaxConcurrentWorkers = 3` const block |
| `internal/cli/cli_agent.go` | Remove `errWorkerQueueFull`, remove cap check in `preSpawnGuards`, update comment |

---

### Task 1: Write the RED test proving the 4th spawn is blocked

**Files:**
- Modify: `internal/cli/cli_test.go` — add test after `TestRunAgentSpawn_MaxWorkers_ReturnsError` (around line 2323)

- [ ] **Step 1: Add the failing test**

Insert this function after `TestRunAgentSpawn_MaxWorkers_ReturnsError` in `internal/cli/cli_test.go`:

```go
func TestRunAgentSpawn_FourthWorker_IsNotRejectedByCap(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cli.SetTestSpawnAckMaxWait(t, 100*time.Millisecond)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// Pre-populate state file with 3 active agents (the old cap).
	for _, name := range []string{"exec-1", "exec-2", "exec-3"} {
		prePopErr := cli.ExportReadModifyWriteStateFile(stateFile, func(sf agentpkg.StateFile) agentpkg.StateFile {
			return agentpkg.AddAgent(sf, agentpkg.AgentRecord{Name: name, State: "ACTIVE"})
		})
		g.Expect(prePopErr).NotTo(HaveOccurred())

		if prePopErr != nil {
			return
		}
	}

	err := cli.ExportRunAgentSpawn([]string{
		"--name", "exec-4",
		"--prompt", "You are exec-4.",
		"--chat-file", chatFile,
		"--state-file", stateFile,
	}, io.Discard, func(_ context.Context, _, _ string) (string, string, error) {
		return "main:1.4", "sess456", nil
	})

	// Cap must be gone: any error should not mention "worker queue full".
	if err != nil {
		g.Expect(err.Error()).NotTo(ContainSubstring("worker queue full"))
	}
}
```

- [ ] **Step 2: Run test to confirm RED**

```bash
targ test 2>&1 | grep -A 3 "FourthWorker"
```

Expected: FAIL — `Expected "agent spawn: worker queue full (max 3 concurrent)" not to contain substring "worker queue full"`

---

### Task 2: Remove the cap (GREEN)

**Files:**
- Modify: `internal/agent/agent.go:13–16` — delete `MaxConcurrentWorkers` const block
- Modify: `internal/cli/cli_agent.go:46` — delete `errWorkerQueueFull` line
- Modify: `internal/cli/cli_agent.go:562–564` — update `preSpawnGuards` doc comment
- Modify: `internal/cli/cli_agent.go:586–588` — delete cap check block

- [ ] **Step 3: Remove `MaxConcurrentWorkers` from agent.go**

In `internal/agent/agent.go`, delete the entire exported constants block:

```go
// Exported constants.
const (
	MaxConcurrentWorkers = 3
)
```

The file will have no `const` block at all after this removal (the imports block stays).

- [ ] **Step 4: Remove `errWorkerQueueFull` from cli_agent.go**

In `internal/cli/cli_agent.go`, delete this line from the `var` block (line 46):

```go
	errWorkerQueueFull          = errors.New("agent spawn: worker queue full (max 3 concurrent)")
```

- [ ] **Step 5: Remove the cap check from `preSpawnGuards` in cli_agent.go**

Replace the `preSpawnGuards` doc comment and cap-check block. Current state (lines 560–590):

```go
// rejectDuplicateAgentName returns an error if the state file already contains an agent
// with the given name, preventing duplicate spawns from creating orphan panes.
// preSpawnGuards runs pre-spawn validation checks: duplicate name and worker queue limit.
// preSpawnGuards reads the state file once and checks both duplicate name and
// worker queue limit. Returns nil if no state file exists (first spawn).
func preSpawnGuards(stateFilePath, name string) error {
	data, err := os.ReadFile(stateFilePath) //nolint:gosec
	if errors.Is(err, os.ErrNotExist) {
		...
	}
	...
	for _, record := range state.Agents {
		if record.Name == name {
			return fmt.Errorf("%w: %s", errDuplicateAgentName, name)
		}
	}

	if agentpkg.ActiveWorkerCount(state) >= agentpkg.MaxConcurrentWorkers {
		return errWorkerQueueFull
	}

	return nil
}
```

After edits (replace the stale doc comment and delete the 3-line cap check):

```go
// preSpawnGuards runs pre-spawn validation: duplicate name check.
// Returns nil if no state file exists (first spawn).
func preSpawnGuards(stateFilePath, name string) error {
```

And delete these three lines from the body:

```go
	if agentpkg.ActiveWorkerCount(state) >= agentpkg.MaxConcurrentWorkers {
		return errWorkerQueueFull
	}
```

- [ ] **Step 6: Run tests to confirm GREEN**

```bash
targ test 2>&1 | grep -E "FAIL|PASS|FourthWorker|MaxWorkers"
```

Expected: `TestRunAgentSpawn_FourthWorker_IsNotRejectedByCap` PASS. `TestRunAgentSpawn_MaxWorkers_ReturnsError` will now FAIL (its behavior is gone — deletion in Task 3 cleans it up). Do not stop here; proceed directly to Task 3.

---

### Task 3: Delete the old cap test and run full checks

**Files:**
- Modify: `internal/cli/cli_test.go` — delete `TestRunAgentSpawn_MaxWorkers_ReturnsError`

- [ ] **Step 7: Delete `TestRunAgentSpawn_MaxWorkers_ReturnsError`**

In `internal/cli/cli_test.go`, delete the entire function `TestRunAgentSpawn_MaxWorkers_ReturnsError` (lines 2284–2323). This is the test that validated the now-removed behavior. It tests for the "worker queue full" error, which no longer occurs.

- [ ] **Step 8: Run full quality check**

```bash
targ check-full
```

Expected: all lint, vet, and tests pass with no errors. No references to `MaxConcurrentWorkers` or `errWorkerQueueFull` remain.

- [ ] **Step 9: Verify no stale references**

```bash
grep -r "MaxConcurrentWorkers\|errWorkerQueueFull\|worker queue full" --include="*.go" .
```

Expected: no output.

- [ ] **Step 10: Commit**

```bash
git add internal/agent/agent.go internal/cli/cli_agent.go internal/cli/cli_test.go
git commit -m "$(cat <<'EOF'
fix(cli): remove undocumented MaxConcurrentWorkers=3 spawn cap (#542)

Removes the 3-worker cap that blocked engram-tmux-lead's 9-pane parallel
workflows. The binary enforced an arbitrary limit with no spec or design
doc; the skill is the correct enforcement boundary via tmux pane allocation.

AI-Used: [claude]
EOF
)"
```
