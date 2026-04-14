# Spawn Intent Semantic Context Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix `postSpawnIntentAndWait` so the spawn intent message uses semantic context (task description, role, and goal) instead of mechanical pane ID details that defeat memory surfacing.

**Architecture:** The function `postSpawnIntentAndWait` in `internal/cli/cli_agent.go` (lines 633-644) currently embeds the pane ID in the intent text unconditionally. The fix removes `paneID` from the intent text entirely: when `intentMsg` is empty the message falls back to a generic semantic template; when it is provided the message uses it directly. No new parameters are added — `flags.intentMsg` (already parsed from `--intent-text`) carries the task string.

**Tech Stack:** Go, `pgregory.net/rapid` for property-based tests, `gomega` for assertions, `targ test` to run tests.

---

## File Map

| File | Change |
|------|--------|
| `internal/cli/cli_agent.go` | Rewrite intent text construction in `postSpawnIntentAndWait` (lines 633–644) |
| `internal/cli/cli_test.go` | Add two new tests: one property-based, one example-based |

No changes to `export_test.go` — `ExportRunAgentSpawn` is already exported.

---

### Task 1: Property-based test — pane ID never appears in intent text

**Files:**
- Test: `internal/cli/cli_test.go`
- Modify: `internal/cli/cli_agent.go:633-644`

- [ ] **Step 1: Write the failing test**

Add this test to `internal/cli/cli_test.go` after `TestRunAgentSpawn_WritesStateFileAndOutputsPaneID`:

```go
// TestRunAgentSpawn_IntentNeverContainsPaneID verifies that the spawn intent
// message posted to the chat file never contains the pane ID regardless of
// whether --intent-text is supplied.
// Property: for any valid agent name, any pane ID, and any intent text (including empty),
// no intent message in the chat file contains the pane ID string.
func TestRunAgentSpawn_IntentNeverContainsPaneID(t *testing.T) {
	t.Parallel()

	cli.SetTestSpawnAckMaxWait(t, 6*time.Second)

	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		agentName := rapid.StringMatching(`[a-z][a-z0-9-]{1,19}`).Draw(rt, "agentName")
		// Pane IDs look like "%42" in tmux — generate a realistic-looking one.
		paneNum := rapid.IntRange(1, 999).Draw(rt, "paneNum")
		paneID := fmt.Sprintf("%%%d", paneNum)
		sessionID := "$sess" + agentName
		intentText := rapid.StringOf(
			rapid.RuneFrom(nil, unicode.Letter, unicode.Digit, unicode.Space),
		).Draw(rt, "intentText")

		dir := rt.TempDir()
		chatFile := filepath.Join(dir, "chat.toml")
		stateFile := filepath.Join(dir, "state.toml")

		if err := os.WriteFile(chatFile, []byte(""), 0o600); err != nil {
			rt.Fatal(err)
		}

		spawnArgs := []string{
			"--name", agentName,
			"--prompt", "You are " + agentName + ".",
			"--chat-file", chatFile,
			"--state-file", stateFile,
		}
		if intentText != "" {
			spawnArgs = append(spawnArgs, "--intent-text", intentText)
		}

		spawnDone := make(chan error, 1)

		go func() {
			spawnDone <- cli.ExportRunAgentSpawn(spawnArgs, io.Discard,
				func(_ context.Context, _, _ string) (string, string, error) {
					return paneID, sessionID, nil
				},
			)
		}()

		// Give the intent time to be posted, then ACK it.
		time.Sleep(50 * time.Millisecond)

		_ = cli.Run([]string{
			"engram", "chat", "post",
			"--chat-file", chatFile,
			"--from", "engram-agent",
			"--to", "system",
			"--thread", "lifecycle",
			"--type", "ack",
			"--text", "No relevant memories. Proceed.",
		}, io.Discard, io.Discard, nil)

		select {
		case err := <-spawnDone:
			g.Expect(err).NotTo(HaveOccurred())
			if err != nil {
				return
			}
		case <-time.After(6 * time.Second):
			rt.Fatal("agent spawn did not complete within 6s")
		}

		messages, loadErr := cli.ExportLoadChatMessages(chatFile)
		g.Expect(loadErr).NotTo(HaveOccurred())
		if loadErr != nil {
			return
		}

		for _, msg := range messages {
			if msg.Type == "intent" {
				g.Expect(msg.Text).NotTo(ContainSubstring(paneID),
					"spawn intent must not contain pane ID %q", paneID)
			}
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — the property test finds a case where `msg.Text` contains the pane ID (e.g. `"%42"`), because the current implementation always formats `paneID` into the intent text.

- [ ] **Step 3: Write minimal implementation**

In `internal/cli/cli_agent.go`, replace the intent text construction in `postSpawnIntentAndWait` (lines 633–644):

**Before:**
```go
func postSpawnIntentAndWait(ctx context.Context, chatFilePath, name, paneID, intentMsg string) error {
	intentText := fmt.Sprintf(
		"Situation: About to spawn agent %q in pane %s.\nBehavior: Agent will post ready when initialized.",
		name, paneID,
	)

	if intentMsg != "" {
		intentText = fmt.Sprintf(
			"Situation: About to spawn agent %q in pane %s. Task: %s\nBehavior: Agent will post ready when initialized.",
			name, paneID, intentMsg,
		)
	}
```

**After:**
```go
func postSpawnIntentAndWait(ctx context.Context, chatFilePath, name, paneID, intentMsg string) error {
	var intentText string

	if intentMsg != "" {
		intentText = fmt.Sprintf(
			"Situation: %s. Behavior: Will spawn agent %q to handle the above.",
			intentMsg, name,
		)
	} else {
		intentText = fmt.Sprintf(
			"Situation: Spawning agent %q as requested. Behavior: Will spawn agent %q to handle the assigned task.",
			name, name,
		)
	}
```

Note: `paneID` is kept in the function signature as it may be used elsewhere in the caller. It is simply no longer embedded in `intentText`.

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS — all rapid cases confirm no pane ID appears in any intent message.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/cli_agent.go internal/cli/cli_test.go
git commit -m "$(cat <<'EOF'
fix(cli): use semantic context in spawn intent instead of pane ID

postSpawnIntentAndWait embedded the tmux pane ID in the intent text,
which changes every session and carries no memory-matching signal.
The skill template (fixed in cd169dd) already uses semantic framing —
this aligns the CLI binary to the same format.

When --intent-text is provided the intent reads:
  Situation: <task>. Behavior: Will spawn <name> to handle the above.
When omitted it falls back to a generic semantic template with no pane ID.

Fixes #535.

AI-Used: [claude]
EOF
)"
```

---

### Task 2: Example-based test — intent text content with and without --intent-text

**Files:**
- Test: `internal/cli/cli_test.go`

- [ ] **Step 1: Write the failing test**

Add this test to `internal/cli/cli_test.go` immediately after `TestRunAgentSpawn_IntentNeverContainsPaneID`:

```go
// TestRunAgentSpawn_IntentText_SemanticFormat verifies that:
// (a) when --intent-text is supplied, the intent message contains the task text
//     and the agent name but not the pane ID;
// (b) when --intent-text is omitted, the intent message still contains the agent
//     name and still does not contain the pane ID.
func TestRunAgentSpawn_IntentText_SemanticFormat(t *testing.T) {
	t.Parallel()

	cli.SetTestSpawnAckMaxWait(t, 6*time.Second)

	cases := []struct {
		name       string
		intentFlag []string
		wantTask   string
		wantName   string
		wantAbsent string
	}{
		{
			name:       "with task",
			intentFlag: []string{"--intent-text", "implement the auth feature"},
			wantTask:   "implement the auth feature",
			wantName:   "auth-worker",
			wantAbsent: "%99",
		},
		{
			name:       "without task",
			intentFlag: nil,
			wantTask:   "",
			wantName:   "auth-worker",
			wantAbsent: "%99",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			dir := t.TempDir()
			chatFile := filepath.Join(dir, "chat.toml")
			stateFile := filepath.Join(dir, "state.toml")

			g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

			spawnArgs := []string{
				"--name", "auth-worker",
				"--prompt", "You are auth-worker.",
				"--chat-file", chatFile,
				"--state-file", stateFile,
			}
			spawnArgs = append(spawnArgs, tc.intentFlag...)

			spawnDone := make(chan error, 1)

			go func() {
				spawnDone <- cli.ExportRunAgentSpawn(spawnArgs, io.Discard,
					func(_ context.Context, _, _ string) (string, string, error) {
						return "%99", "$sess-auth", nil
					},
				)
			}()

			time.Sleep(50 * time.Millisecond)

			g.Expect(cli.Run([]string{
				"engram", "chat", "post",
				"--chat-file", chatFile,
				"--from", "engram-agent",
				"--to", "system",
				"--thread", "lifecycle",
				"--type", "ack",
				"--text", "No relevant memories. Proceed.",
			}, io.Discard, io.Discard, nil)).To(Succeed())

			select {
			case err := <-spawnDone:
				g.Expect(err).NotTo(HaveOccurred())
				if err != nil {
					return
				}
			case <-time.After(6 * time.Second):
				t.Fatal("agent spawn did not complete within 6s")
			}

			messages, loadErr := cli.ExportLoadChatMessages(chatFile)
			g.Expect(loadErr).NotTo(HaveOccurred())
			if loadErr != nil {
				return
			}

			var intentMsg string
			for _, msg := range messages {
				if msg.Type == "intent" {
					intentMsg = msg.Text
					break
				}
			}

			g.Expect(intentMsg).NotTo(BeEmpty(), "expected an intent message in chat")

			if tc.wantTask != "" {
				g.Expect(intentMsg).To(ContainSubstring(tc.wantTask))
			}

			g.Expect(intentMsg).To(ContainSubstring(tc.wantName))
			g.Expect(intentMsg).NotTo(ContainSubstring(tc.wantAbsent),
				"pane ID must not appear in intent text")
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `with task` case fails because pane ID appears in intent text and task text does not.

- [ ] **Step 3: Confirm implementation from Task 1 is in place**

No additional code changes needed. The fix in Task 1 Step 3 covers both cases.

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS — both sub-cases pass. All prior spawn tests continue to pass.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/cli_test.go
git commit -m "$(cat <<'EOF'
test(cli): add example-based test for spawn intent semantic format

Covers both the --intent-text supplied and omitted branches, asserting
the task text appears and the pane ID does not appear in the intent message.
Complements the property test added in the #535 fix commit.

AI-Used: [claude]
EOF
)"
```
