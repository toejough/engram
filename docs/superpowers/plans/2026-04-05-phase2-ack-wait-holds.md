# Phase 2 — ACK-Wait Binary + Hold Commands

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the ACK-wait background subagent template with a single binary blocking call (`engram chat ack-wait`). Add chat-file-native hold commands (`engram hold acquire/release/list/check`). Fixes the Phase 1 latent ACK-wait race condition and eliminates per-hold background tasks from the lead skill.

**Spec:** `docs/superpowers/specs/engram-deterministic-coordination-design.md` §3.1, §4.1, §7 (Phase 2)

**Codesign session:** planners 31 (arch), 32 (skill), 33 (agent E2E), 34 (user E2E) — 2026-04-05. All four perspectives converged before this plan was written.

**Tech Stack:** Go, BurntSushi/toml (already in go.mod), github.com/fsnotify/fsnotify (added in Phase 1)

---

## Codesign Decisions

These decisions were argued and resolved during planning. Do NOT revisit without reading the codesign thread.

| Decision | Resolved | Rationale |
|----------|----------|-----------|
| FileAckWaiter in `internal/chat/` | Yes | Alongside FilePoster/FileWatcher — same DI func injection pattern |
| Func injection for Watch (not Watcher interface) | Yes | Consistent with Phase 1 codebase pattern |
| TIMEOUT as third AckResult type | Yes | Clean JSON output; exit 0 for all three — caller parses result field |
| Hold state: chat-file-native | Yes | TOML messages only; no separate registry file |
| ScanActiveHolds + EvaluateCondition are pure functions | Yes | No I/O in domain logic; wired at CLI layer |
| Background Monitor Pattern survives Phase 2 | Yes | Phase 2 eliminates WAITER side (ack-wait subagent); RESPONDER side (engram-agent watch loop) survives to Phase 5 |
| `hold-acquire`/`hold-release` in type catalog | Yes | Binary-only types; must not be posted with `engram chat post` directly |
| `--hold-id` flag (not positional) for hold release | Yes | Consistent with targ struct flag pattern |
| `engram hold check` manual-only in Phase 2 | Yes | Auto-invocation from `engram agent kill` ships Phase 3 |
| Online-silent TIMEOUT result (not stderr) | Yes | All three results to stdout JSON; non-zero exit only for system errors |
| Reading New Content section NOT deleted in Phase 2 | Yes | Background Monitor Pattern still needs cursor-based tail instructions |
| `--help` on subcommands exits 0 | Yes | ContinueOnError + --help must return nil, not error (small CLI fix) |
| Argument continuation uses `engram chat watch` directly | Yes | Single-recipient bounded watch; acceptable as direct Bash per two-pattern distinction |
| Phase 1 `--agent RECIPIENT` bug documented and fixed | Yes | Phase 1 subagent filtered by TO field; Phase 2 ack-wait filters by FROM field — correct |

---

## Package Structure

```
internal/
  chat/
    chat.go          (existing — add AckResult/WaitResult/TimeoutResult types)
    ackwaiter.go     (NEW: FileAckWaiter)
    ackwaiter_test.go
    hold.go          (NEW: HoldRecord, ScanActiveHolds, EvaluateCondition)
    hold_test.go
    poster.go        (existing — unchanged)
    watcher.go       (existing — unchanged)

internal/cli/
  cli.go             (add runChatAckWait, runHoldDispatch + 4 hold runners)
  targets.go         (add ChatAckWaitArgs, HoldAcquireArgs, BuildHoldGroup)
  cli_test.go        (add tests for new subcommands)
```

No new packages. No new external deps (`crypto/rand` is stdlib; TOML and fsnotify already present).

---

## Task 1: AckResult Types + FileAckWaiter

**Files:**
- Modify: `internal/chat/chat.go` — add AckResult, WaitResult, TimeoutResult
- Create: `internal/chat/ackwaiter.go`
- Create: `internal/chat/ackwaiter_test.go`

### Step 1: Write failing tests for AckResult JSON round-trip

- [ ] **Step 1: Write failing tests for AckResult types**

```go
// internal/chat/ackwaiter_test.go

func TestAckResult_ACK_JSONRoundTrip(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    result := chat.AckResult{Result: "ACK", NewCursor: 1234}
    data, err := json.Marshal(result)
    g.Expect(err).NotTo(HaveOccurred())
    if err != nil { return }

    var got chat.AckResult
    g.Expect(json.Unmarshal(data, &got)).To(Succeed())
    g.Expect(got.Result).To(Equal("ACK"))
    g.Expect(got.NewCursor).To(Equal(1234))
    g.Expect(got.Wait).To(BeNil())
    g.Expect(got.Timeout).To(BeNil())
}

func TestAckResult_WAIT_JSONRoundTrip(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    result := chat.AckResult{
        Result:    "WAIT",
        NewCursor: 5678,
        Wait:      &chat.WaitResult{From: "engram-agent", Text: "objection text"},
    }
    data, err := json.Marshal(result)
    g.Expect(err).NotTo(HaveOccurred())
    if err != nil { return }

    // Verify CLI format: {"result":"WAIT","from":"engram-agent","cursor":5678,"text":"..."}
    g.Expect(string(data)).To(ContainSubstring(`"result":"WAIT"`))
    g.Expect(string(data)).To(ContainSubstring(`"from":"engram-agent"`))
    g.Expect(string(data)).To(ContainSubstring(`"cursor":5678`))
    g.Expect(string(data)).To(ContainSubstring(`"text":"objection text"`))
}

func TestAckResult_TIMEOUT_JSONRoundTrip(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    result := chat.AckResult{
        Result:    "TIMEOUT",
        NewCursor: 999,
        Timeout:   &chat.TimeoutResult{Recipient: "engram-agent"},
    }
    data, err := json.Marshal(result)
    g.Expect(err).NotTo(HaveOccurred())
    if err != nil { return }

    g.Expect(string(data)).To(ContainSubstring(`"result":"TIMEOUT"`))
    g.Expect(string(data)).To(ContainSubstring(`"recipient":"engram-agent"`))
}
```

- [ ] **Step 2: Add AckResult/WaitResult/TimeoutResult to `internal/chat/chat.go`**

```go
// AckResult is the result of an AckWait call.
// Result is "ACK", "WAIT", or "TIMEOUT".
// Exit code 0 for all three; non-zero only for system errors.
type AckResult struct {
    Result    string         `json:"result"`
    Wait      *WaitResult    `json:"wait,omitempty"`
    Timeout   *TimeoutResult `json:"timeout,omitempty"`
    NewCursor int            `json:"cursor"`
}

// WaitResult carries the WAIT message details.
type WaitResult struct {
    From string `json:"from"`
    Text string `json:"text"`
}

// TimeoutResult names the online-but-silent recipient.
type TimeoutResult struct {
    Recipient string `json:"recipient"`
}

// AckWaiter blocks until all recipients respond.
// Invariants: when AckResult.Result=="WAIT", AckResult.Wait is non-nil.
//             when AckResult.Result=="TIMEOUT", AckResult.Timeout is non-nil.
//             when AckResult.Result=="ACK", both Wait and Timeout are nil.
type AckWaiter interface {
    AckWait(ctx context.Context, agent string, cursor int, recipients []string) (AckResult, error)
}
```

**Plan supersedes spec §4.1 AckResult shape:** The spec's `AckResult{AllAcked bool, ...}` is
superseded by this plan's `AckResult{Result string, ...}`. The codesign decision "TIMEOUT as
third AckResult type — Yes" changed the entire struct shape. Spec §4.1 is stale on this point.

**Note on CLI JSON output format** (flat, not nested):

The CLI layer (not the domain struct) marshals to the flat format expected by skills:
```
{"result":"ACK","cursor":1234}
{"result":"WAIT","from":"agent","cursor":1234,"text":"..."}
{"result":"TIMEOUT","recipient":"agent","cursor":1234}
```

### Step 3–4: FileAckWaiter unit tests

- [ ] **Step 3: Write failing unit tests for FileAckWaiter**

```go
func TestFileAckWaiter_AllACK(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    // Two recipients; both post ack after cursor.
    // ReadFile returns a file with their ack messages.
    // Watch returns the first ack, then the second.
    // Expected: AckResult{Result:"ACK", NewCursor:N}

    ackMessages := []chat.Message{
        {From: "engram-agent", To: "caller", Thread: "t", Type: "ack", Text: "ok"},
        {From: "reviewer",     To: "caller", Thread: "t", Type: "ack", Text: "ok"},
    }
    // ... table-driven setup with fakeWatch returning ack messages ...
}

func TestFileAckWaiter_WAITReturnedImmediately(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    // First response is a WAIT from engram-agent.
    // Expected: AckResult{Result:"WAIT", Wait:&WaitResult{From:"engram-agent",...}}
}

func TestFileAckWaiter_OfflineImplicitACKAfter5s(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    // Recipient has no messages in last 15 min.
    // NowFunc returns +6s on second call.
    // Expected: AckResult{Result:"ACK"} after implicit offline timeout.
}

func TestFileAckWaiter_OnlineSilentTIMEOUT(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    // Recipient posted a message within last 15 min (online).
    // Watch never returns matching message before max-wait.
    // Expected: AckResult{Result:"TIMEOUT", Timeout:&TimeoutResult{Recipient:"engram-agent"}}
}

func TestFileAckWaiter_MultiRecipient_BothMustACK(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    // Two recipients; first ACKs immediately; second is offline and needs 5s implicit.
    // Expected: ACK only after both are resolved.
}

func TestFileAckWaiter_CtxCancellation(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    // Cancel ctx while waiting. Expect ctx.Err() returned.
}
```

- [ ] **Step 4: Implement `internal/chat/ackwaiter.go`**

```go
// FileAckWaiter waits for ACK/WAIT responses from all named recipients.
// All I/O is injected — no os.* calls in this package.
type FileAckWaiter struct {
    FilePath string
    Watch    func(ctx context.Context, agent string, cursor int, msgTypes []string) (Message, int, error)
    ReadFile func(path string) ([]byte, error)
    NowFunc  func() time.Time // injectable for online/offline detection tests
    MaxWait  time.Duration    // default 30s; per-online-silent-recipient timeout
}

// recipientState tracks per-recipient ACK progress.
type recipientState struct {
    responded bool
    online    bool      // true if posted any message within last 15 min
    waitStart time.Time // when we started waiting for this recipient
}

// AckWait blocks until all recipients respond or a timeout/error occurs.
// Algorithm:
//  1. Read full chat file → detect online/offline per recipient (full-file scan, not cursor-bounded)
//  2. Build per-recipient state
//  3. Loop:
//     a. Scan messages after cursor for ack/wait FROM each recipient TO callerAgent
//        Correctness: filter msg.From == recipient (not msg.To == recipient — Phase 1 bug fixed)
//     b. WAIT found → return AckResult{Result:"WAIT", ...} immediately
//     c. ACK found → mark recipient as responded
//     d. Sweep offline recipients: elapsed ≥ 5s → implicit ACK
//     e. Check online+silent: elapsed ≥ MaxWait → return TIMEOUT for first violator
//     f. All responded → return AckResult{Result:"ACK", NewCursor:N}
//     g. Block on Watch(ctx, callerAgent, cursor, ["ack","wait"]) until next event
func (w *FileAckWaiter) AckWait(ctx context.Context, callerAgent string, cursor int, recipients []string) (AckResult, error) {
    // ... implementation ...
}
```

**CRITICAL correctness fix from Phase 1:**
Phase 1 subagent used `engram chat watch --agent RECIPIENT`, which matched messages addressed TO the recipient. ACKs from the recipient are addressed TO the callerAgent. Phase 2 fixes this: `Watch(ctx, callerAgent, cursor, ["ack","wait"])` watches for messages addressed to callerAgent, then filters `msg.From == recipient`.

Run `targ check-full` after Step 4. Commit with `/commit`.

---

## Task 2: Hold Domain Logic

**Files:**
- Create: `internal/chat/hold.go`
- Create: `internal/chat/hold_test.go`

### HoldRecord + ScanActiveHolds + EvaluateCondition

- [ ] **Step 5: Write failing unit tests for HoldRecord JSON round-trip**

```go
func TestHoldRecord_JSONRoundTrip(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    original := chat.HoldRecord{
        HoldID:     "h-12345",
        Holder:     "reviewer-1",
        Target:     "executor-1",
        Condition:  "done:reviewer-1",
        AcquiredTS: time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC),
    }
    data, err := json.Marshal(original)
    g.Expect(err).NotTo(HaveOccurred())
    if err != nil { return }

    var got chat.HoldRecord
    g.Expect(json.Unmarshal(data, &got)).To(Succeed())
    g.Expect(got).To(Equal(original))
}
```

- [ ] **Step 6: Write failing unit tests for ScanActiveHolds**

```go
func TestScanActiveHolds_EmptyMessages(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)
    holds := chat.ScanActiveHolds(nil)
    g.Expect(holds).To(BeEmpty())
}

func TestScanActiveHolds_AcquireWithNoRelease(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    record := chat.HoldRecord{HoldID: "h1", Holder: "lead", Target: "exec-1", Condition: "done:lead"}
    text, _ := json.Marshal(record)
    messages := []chat.Message{
        {From: "lead", To: "exec-1", Type: "hold-acquire", Text: string(text)},
    }
    holds := chat.ScanActiveHolds(messages)
    g.Expect(holds).To(HaveLen(1))
    g.Expect(holds[0].HoldID).To(Equal("h1"))
}

func TestScanActiveHolds_AcquireAndRelease_ReturnsEmpty(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    record := chat.HoldRecord{HoldID: "h2", Holder: "lead", Target: "exec-1"}
    text, _ := json.Marshal(record)
    messages := []chat.Message{
        {From: "lead", To: "exec-1", Type: "hold-acquire", Text: string(text)},
        {From: "lead", To: "exec-1", Type: "hold-release", Text: string(text)},
    }
    holds := chat.ScanActiveHolds(messages)
    g.Expect(holds).To(BeEmpty())
}

func TestScanActiveHolds_MultipleHolds_IndependentTracking(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)
    // Two holds acquired; one released; one remains.
}
```

- [ ] **Step 7: Write failing unit tests for EvaluateCondition**

```go
func TestEvaluateCondition_DoneAgent_ConditionMet(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    hold := chat.HoldRecord{HoldID: "h1", Condition: "done:reviewer-1", AcquiredTS: time.Now().Add(-1 * time.Minute)}
    messages := []chat.Message{
        {From: "reviewer-1", Type: "done", TS: time.Now()},
    }
    met, reason := chat.EvaluateCondition(hold, messages)
    g.Expect(met).To(BeTrue())
    g.Expect(reason).NotTo(BeEmpty())
}

func TestEvaluateCondition_DoneAgent_ConditionNotMet(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    hold := chat.HoldRecord{HoldID: "h1", Condition: "done:reviewer-1", AcquiredTS: time.Now().Add(-1 * time.Minute)}
    messages := []chat.Message{
        {From: "reviewer-1", Type: "info", TS: time.Now()},
    }
    met, _ := chat.EvaluateCondition(hold, messages)
    g.Expect(met).To(BeFalse())
}

func TestEvaluateCondition_FirstIntent_Met(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)
    // Condition: "first-intent:exec-1"; exec-1 posts intent after AcquiredTS → met.
}

func TestEvaluateCondition_LeadRelease_NeverAutoMet(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)
    // Condition: "lead-release:tag"; no messages ever meet this automatically.
    hold := chat.HoldRecord{HoldID: "h1", Condition: "lead-release:tag", AcquiredTS: time.Now().Add(-1 * time.Minute)}
    messages := []chat.Message{
        {From: "lead", Type: "done", TS: time.Now()},
    }
    met, _ := chat.EvaluateCondition(hold, messages)
    g.Expect(met).To(BeFalse(), "lead-release conditions never auto-evaluate to true")
}

func TestEvaluateCondition_EmptyCondition_NeverMet(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)
    // No condition set → never auto-releases; requires explicit release.
    hold := chat.HoldRecord{HoldID: "h1", Condition: "", AcquiredTS: time.Now().Add(-1 * time.Minute)}
    met, _ := chat.EvaluateCondition(hold, nil)
    g.Expect(met).To(BeFalse())
}
```

- [ ] **Step 8: Implement `internal/chat/hold.go`**

```go
// Package chat — hold.go

// HoldRecord is the payload of a hold-acquire message.
type HoldRecord struct {
    HoldID     string    `json:"hold-id"`
    Holder     string    `json:"holder"`
    Target     string    `json:"target"`
    Condition  string    `json:"condition,omitempty"` // "done:<agent>", "first-intent:<agent>", "lead-release:<tag>", or empty
    Tag        string    `json:"tag,omitempty"`       // workflow label for bulk operations (e.g., "codesign-1", "plan-review-1")
    AcquiredTS time.Time `json:"acquired-ts"`
}

// REVIEWER NOTE: Tag field is required. The lead skill uses tag-based bulk dissolve for
// codesign (lead_release("codesign-N") dissolves ALL barrier holds at once) and merge
// queue. Without Tag in HoldRecord, ScanActiveHolds cannot support the --tag filter.

// ScanActiveHolds returns holds with no matching release in messages.
// Pure function — no I/O. O(n) per hold; acceptable for session-length files.
//
// HoldRecord is stored as JSON in Message.Text. Both acquire and release messages
// must be unmarshaled to extract HoldID for matching. Messages that fail to unmarshal
// are silently skipped (they are not valid hold messages).
func ScanActiveHolds(messages []Message) []HoldRecord {
    // 1. For each message with Type=="hold-acquire":
    //    json.Unmarshal([]byte(msg.Text), &record)
    //    On unmarshal error: skip (not a valid hold message).
    //    Otherwise: add to acquired map keyed by record.HoldID.
    //
    // 2. For each message with Type=="hold-release":
    //    json.Unmarshal([]byte(msg.Text), &record)
    //    On unmarshal error: skip.
    //    Otherwise: delete acquired[record.HoldID] (release cancels acquire).
    //
    // 3. Return values of acquired map as []HoldRecord slice.
    //
    // NOTE: release Text need only contain {"hold-id":"..."} — see runHoldRelease wiring.
}

// EvaluateCondition checks if hold condition is met against messages after hold.AcquiredTS.
// Conditions:
//   "done:<agent>"         → true when agent posts type="done" after AcquiredTS
//   "first-intent:<agent>" → true when agent posts type="intent" after AcquiredTS
//   "lead-release:<tag>"   → NEVER auto-evaluates to true (requires explicit release)
//   ""                     → NEVER auto-evaluates to true (requires explicit release)
// Pure function — no I/O.
func EvaluateCondition(hold HoldRecord, messages []Message) (met bool, reason string) {
    // ...
}
```

**UUID generation:** Use `crypto/rand` (per CLAUDE.md — never `math/rand`). UUID generation happens at the CLI layer in `runHoldAcquire`, not in domain logic.

Run `targ check-full` after Step 8. Commit with `/commit`.

---

## Task 3: CLI Wiring

**Files:**
- Modify: `internal/cli/cli.go` — add `runChatAckWait`, `runHoldDispatch`, `runHoldAcquire`, `runHoldRelease`, `runHoldList`, `runHoldCheck`
- Modify: `internal/cli/targets.go` — add `ChatAckWaitArgs`, `HoldAcquireArgs`, `HoldReleaseArgs`, `HoldListArgs`, `HoldCheckArgs`, `targ.Group("hold", ...)`
- Modify: `internal/cli/cli_test.go` — add tests for all new subcommands

### Step 9: CLI ack-wait tests

- [ ] **Step 9: Write failing tests for `runChatAckWait`**

```go
func TestRun_ChatAckWait_AllACK(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    dir := t.TempDir()
    chatFile := filepath.Join(dir, "chat.toml")

    // Pre-populate chat file with an ack from engram-agent addressed to "tester"
    // ... write TOML with ack message ...

    var stdout bytes.Buffer
    err := cli.Run([]string{
        "engram", "chat", "ack-wait",
        "--chat-file", chatFile,
        "--agent", "tester",
        "--cursor", "0",
        "--recipients", "engram-agent",
        "--max-wait", "5",
    }, &stdout, io.Discard, nil)
    g.Expect(err).NotTo(HaveOccurred())
    if err != nil { return }

    var result map[string]interface{}
    g.Expect(json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &result)).To(Succeed())
    g.Expect(result["result"]).To(Equal("ACK"))
    g.Expect(result).To(HaveKey("cursor"))
}

func TestRun_ChatAckWait_WAIT(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)
    // Pre-populate chat file with a wait message from engram-agent to "tester".
    // Expect JSON output with result="WAIT", from="engram-agent", text=...
}

func TestRun_ChatAckWait_OfflineImplicitACK(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)
    // Recipient "nonexistent" has no messages in chat file.
    // Expect ACK after ~5s (offline implicit ACK).
}

func TestRun_ChatAckWait_MaxWaitFlag_NoTargCollision(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)
    // Regression test: verify --max-wait 1 does not cause targ flag parsing error.
    // Run with an offline recipient; expect ACK in ~5s (not a flag error).
    // This is the Phase 1 --timeout collision lesson applied to Phase 2.
}
```

### Step 10: CLI hold command tests

- [ ] **Step 10: Write failing tests for hold subcommands**

```go
func TestRun_HoldAcquire_PostsMessage(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    dir := t.TempDir()
    chatFile := filepath.Join(dir, "chat.toml")

    var stdout bytes.Buffer
    err := cli.Run([]string{
        "engram", "hold", "acquire",
        "--chat-file", chatFile,
        "--holder", "lead",
        "--target", "executor-1",
        "--condition", "done:lead",
    }, &stdout, io.Discard, nil)
    g.Expect(err).NotTo(HaveOccurred())
    if err != nil { return }

    // stdout should be the hold-id (UUID string)
    holdID := strings.TrimSpace(stdout.String())
    g.Expect(holdID).NotTo(BeEmpty())

    // chat file should have a hold-acquire message
    data, _ := os.ReadFile(chatFile)
    var parsed struct{ Message []chat.Message `toml:"message"` }
    g.Expect(toml.Unmarshal(data, &parsed)).To(Succeed())
    g.Expect(parsed.Message).To(HaveLen(1))
    g.Expect(parsed.Message[0].Type).To(Equal("hold-acquire"))

    var record chat.HoldRecord
    g.Expect(json.Unmarshal([]byte(parsed.Message[0].Text), &record)).To(Succeed())
    g.Expect(record.HoldID).To(Equal(holdID))
    g.Expect(record.Condition).To(Equal("done:lead"))
}

func TestRun_HoldRelease_PostsMessage(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)
    // Acquire then release. Verify hold-release message posted with matching hold-id.
}

func TestRun_HoldList_FiltersCorrectly(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)
    // Two holds; filter by --holder lead; verify only one returned.
}

func TestRun_HoldList_FilterByTag(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)
    // Two holds with different tags; filter by --tag codesign-1; verify only matching hold returned.
    // Critical: lead_release("codesign-N") needs this to enumerate hold IDs before releasing.
}

func TestRun_HoldCheck_AutoReleasesMetCondition(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)
    // Hold with condition "done:reviewer-1". Chat file has reviewer-1 posting done.
    // Hold check should post hold-release and output the released hold-id.
}

func TestRun_HoldHelp_ExitsZero(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)
    // Regression: --help must exit 0 (ContinueOnError fix).
    err := cli.Run([]string{"engram", "hold", "acquire", "--help"}, io.Discard, io.Discard, nil)
    g.Expect(err).NotTo(HaveOccurred())
}
```

### Step 11: Implement CLI wiring

- [ ] **Step 11: Implement `runChatAckWait` and hold dispatch in `cli.go`**

**WIRING CHECKLIST — must all be done in Step 11 (easy to miss):**

1. Add `"ack-wait"` case to `runChatDispatch` switch (cli.go:380-390):
   ```go
   case "ack-wait":
       return runChatAckWait(subArgs[1:], stdout)
   ```
2. Add `"hold"` case to top-level `Run()` switch (cli.go:49-58):
   ```go
   case "hold":
       return runHoldDispatch(subArgs, stdout)
   ```
3. Add `--help` nil return to every new `flag.FlagSet` parse block (ContinueOnError returns `flag.ErrHelp` on --help; callers expect exit 0):
   ```go
   if errors.Is(parseErr, flag.ErrHelp) { return nil }
   ```
4. Update `errUsage` string to include `hold` (currently `"usage: engram <recall|show> [flags]"`).

**`runChatAckWait` DI wiring:**
```go
func runChatAckWait(args []string, stdout io.Writer) error {
    fs := flag.NewFlagSet("chat ack-wait", flag.ContinueOnError)
    fs.SetOutput(io.Discard)

    agent    := fs.String("agent", "", "calling agent name")
    cursor   := fs.Int("cursor", 0, "line position to start watching from")
    recips   := fs.String("recipients", "", "comma-separated recipient names")
    maxWaitS := fs.Int("max-wait", 0, "seconds to wait for online-silent recipients (default 30)")
    chatFile := fs.String("chat-file", "", "override chat file path (testing only)")

    parseErr := fs.Parse(args)
    if errors.Is(parseErr, flag.ErrHelp) { return nil }
    if parseErr != nil { return fmt.Errorf("chat ack-wait: %w", parseErr) }

    chatFilePath, pathErr := deriveChatFilePath(*chatFile, os.UserHomeDir, os.Getwd)
    if pathErr != nil { return fmt.Errorf("chat ack-wait: %w", pathErr) }

    watcher := &chat.FileWatcher{
        FilePath:  chatFilePath,
        FSWatcher: &watch.FSNotifyWatcher{},
        ReadFile:  os.ReadFile,
    }

    maxWait := time.Duration(*maxWaitS) * time.Second
    if maxWait == 0 {
        maxWait = 30 * time.Second // default
    }

    recipients := strings.Split(*recips, ",")

    waiter := &chat.FileAckWaiter{
        FilePath: chatFilePath,
        Watch:    watcher.Watch,
        ReadFile: os.ReadFile,
        NowFunc:  time.Now, // REQUIRED: nil NowFunc panics on online/offline detection
        MaxWait:  maxWait,
    }

    ctx, cancel := signalContext() // use existing helper — consistent with runChatWatch/runRecall
    defer cancel()

    result, ackErr := waiter.AckWait(ctx, *agent, *cursor, recipients)
    if ackErr != nil { return fmt.Errorf("chat ack-wait: %w", ackErr) }

    // Output flat JSON format (not nested struct)
    return outputAckResult(stdout, result)
}
```

**`outputAckResult` flat JSON:**
```go
func outputAckResult(w io.Writer, result chat.AckResult) error {
    var out map[string]interface{}
    switch result.Result {
    case "ACK":
        out = map[string]interface{}{"result": "ACK", "cursor": result.NewCursor}
    case "WAIT":
        // AckWaiter invariant: Wait is non-nil when Result=="WAIT".
        // Guard anyway — a nil dereference here would be silent data loss in production.
        if result.Wait == nil {
            return fmt.Errorf("outputAckResult: WAIT result has nil Wait field")
        }
        out = map[string]interface{}{
            "result": "WAIT",
            "from":   result.Wait.From,
            "cursor": result.NewCursor,
            "text":   result.Wait.Text,
        }
    case "TIMEOUT":
        // Same invariant guard.
        if result.Timeout == nil {
            return fmt.Errorf("outputAckResult: TIMEOUT result has nil Timeout field")
        }
        out = map[string]interface{}{"result": "TIMEOUT", "recipient": result.Timeout.Recipient, "cursor": result.NewCursor}
    default:
        return fmt.Errorf("outputAckResult: unexpected result type %q", result.Result)
    }
    data, err := json.Marshal(out)
    if err != nil { return err }
    _, err = fmt.Fprintln(w, string(data))
    return err
}
```

**`runHoldAcquire` UUID generation:**
```go
func runHoldAcquire(args []string, stdout io.Writer) error {
    fs := flag.NewFlagSet("hold acquire", flag.ContinueOnError)
    fs.SetOutput(io.Discard)
    holder   := fs.String("holder", "", "agent acquiring the hold")
    target   := fs.String("target", "", "agent being held")
    condition := fs.String("condition", "", "auto-release condition")
    chatFile := fs.String("chat-file", "", "override chat file path (testing only)")
    parseErr := fs.Parse(args)
    if errors.Is(parseErr, flag.ErrHelp) { return nil }
    if parseErr != nil { return fmt.Errorf("hold acquire: %w", parseErr) }

    chatFilePath, pathErr := deriveChatFilePath(*chatFile, os.UserHomeDir, os.Getwd)
    if pathErr != nil { return fmt.Errorf("hold acquire: %w", pathErr) }

    holdID, err := generateHoldID() // uses crypto/rand
    if err != nil { return fmt.Errorf("generating hold id: %w", err) }

    record := chat.HoldRecord{
        HoldID:     holdID,
        Holder:     *holder,
        Target:     *target,
        Condition:  *condition,
        AcquiredTS: time.Now().UTC(),
    }
    text, err := json.Marshal(record)
    if err != nil { return fmt.Errorf("marshaling hold record: %w", err) }

    poster := &chat.FilePoster{
        FilePath:   chatFilePath,
        Lock:       osLockFile,
        AppendFile: osAppendFile,
        LineCount:  osLineCount,
    }
    // From=holder, To=target, Thread="hold", Type="hold-acquire", Text=full HoldRecord JSON
    _, postErr := poster.Post(chat.Message{From: *holder, To: *target, Thread: "hold", Type: "hold-acquire", Text: string(text)})
    if postErr != nil { return fmt.Errorf("hold acquire: posting: %w", postErr) }

    _, err = fmt.Fprintln(stdout, holdID)
    return err
}
```

**`runHoldRelease` — release message fields:**
```go
func runHoldRelease(args []string, stdout io.Writer) error {
    fs := flag.NewFlagSet("hold release", flag.ContinueOnError)
    fs.SetOutput(io.Discard)
    holdID   := fs.String("hold-id", "", "hold ID returned by engram hold acquire")
    chatFile := fs.String("chat-file", "", "override chat file path (testing only)")
    parseErr := fs.Parse(args)
    if errors.Is(parseErr, flag.ErrHelp) { return nil }
    if parseErr != nil { return fmt.Errorf("hold release: %w", parseErr) }

    chatFilePath, pathErr := deriveChatFilePath(*chatFile, os.UserHomeDir, os.Getwd)
    if pathErr != nil { return fmt.Errorf("hold release: %w", pathErr) }

    // Release message Text: minimal JSON containing hold-id for ScanActiveHolds matching.
    // From="system", To="all", Thread="hold" — consistent with acquire's Thread="hold".
    releasePayload, _ := json.Marshal(map[string]string{"hold-id": *holdID})
    poster := &chat.FilePoster{
        FilePath:   chatFilePath,
        Lock:       osLockFile,
        AppendFile: osAppendFile,
        LineCount:  osLineCount,
    }
    _, postErr := poster.Post(chat.Message{From: "system", To: "all", Thread: "hold", Type: "hold-release", Text: string(releasePayload)})
    if postErr != nil { return fmt.Errorf("hold release: posting: %w", postErr) }

    _, err := fmt.Fprintln(stdout, "OK")
    return err
}
```

**NOTE on ScanActiveHolds matching:** acquire messages have full HoldRecord JSON in Text;
release messages have `{"hold-id":"..."}` in Text. ScanActiveHolds unmarshals both:
both formats contain `hold-id` at the JSON root, so `record.HoldID` is populated for both.
The hold-release Text intentionally omits Holder/Target/Condition — only HoldID is needed
for set-subtraction matching.

**`runHoldList` — scans active holds, filters by holder/target/tag, prints tab-separated:**
```go
func runHoldList(args []string, stdout io.Writer) error {
    fs := flag.NewFlagSet("hold list", flag.ContinueOnError)
    fs.SetOutput(io.Discard)
    holder   := fs.String("holder", "", "filter by holder agent name")
    target   := fs.String("target", "", "filter by target agent name")
    tag      := fs.String("tag", "", "filter by workflow tag")
    chatFile := fs.String("chat-file", "", "override chat file path (testing only)")
    parseErr := fs.Parse(args)
    if errors.Is(parseErr, flag.ErrHelp) { return nil }
    if parseErr != nil { return fmt.Errorf("hold list: %w", parseErr) }

    chatFilePath, pathErr := deriveChatFilePath(*chatFile, os.UserHomeDir, os.Getwd)
    if pathErr != nil { return fmt.Errorf("hold list: %w", pathErr) }

    data, readErr := os.ReadFile(chatFilePath)
    if os.IsNotExist(readErr) { return nil } // no chat file = no holds (not an error)
    if readErr != nil { return fmt.Errorf("hold list: reading chat file: %w", readErr) }

    var parsed struct{ Message []chat.Message `toml:"message"` }
    if unmarshalErr := toml.Unmarshal(data, &parsed); unmarshalErr != nil {
        return fmt.Errorf("hold list: parsing chat file: %w", unmarshalErr)
    }

    activeHolds := chat.ScanActiveHolds(parsed.Message)
    for _, hold := range activeHolds {
        if *holder != "" && hold.Holder != *holder { continue }
        if *target != "" && hold.Target != *target { continue }
        if *tag != "" && hold.Tag != *tag { continue }
        fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", hold.HoldID, hold.Holder, hold.Target, hold.Condition)
    }
    return nil
}
```

**`runHoldCheck` — reads all messages, evaluates conditions, posts releases for met holds. Non-trivial — fully specified here to prevent implementation drift:**

```go
func runHoldCheck(args []string, stdout io.Writer) error {
    fs := flag.NewFlagSet("hold check", flag.ContinueOnError)
    fs.SetOutput(io.Discard)
    chatFile := fs.String("chat-file", "", "override chat file path (testing only)")
    parseErr := fs.Parse(args)
    if errors.Is(parseErr, flag.ErrHelp) { return nil }
    if parseErr != nil { return fmt.Errorf("hold check: %w", parseErr) }

    chatFilePath, pathErr := deriveChatFilePath(*chatFile, os.UserHomeDir, os.Getwd)
    if pathErr != nil { return fmt.Errorf("hold check: %w", pathErr) }

    data, readErr := os.ReadFile(chatFilePath)
    if os.IsNotExist(readErr) { return nil } // no chat file = no holds
    if readErr != nil { return fmt.Errorf("hold check: reading chat file: %w", readErr) }

    var parsed struct{ Message []chat.Message `toml:"message"` }
    if unmarshalErr := toml.Unmarshal(data, &parsed); unmarshalErr != nil {
        return fmt.Errorf("hold check: parsing chat file: %w", unmarshalErr)
    }

    activeHolds := chat.ScanActiveHolds(parsed.Message)
    if len(activeHolds) == 0 { return nil }

    poster := &chat.FilePoster{
        FilePath:   chatFilePath,
        Lock:       osLockFile,
        AppendFile: osAppendFile,
        LineCount:  osLineCount,
    }

    for _, hold := range activeHolds {
        met, _ := chat.EvaluateCondition(hold, parsed.Message)
        if !met { continue }

        // Release text: minimal JSON (same format as runHoldRelease)
        releaseText, marshalErr := json.Marshal(map[string]string{"hold-id": hold.HoldID})
        if marshalErr != nil { return fmt.Errorf("hold check: marshaling release: %w", marshalErr) }

        _, postErr := poster.Post(chat.Message{
            From:   "system",
            To:     "all",
            Thread: "hold",
            Type:   "hold-release",
            Text:   string(releaseText),
        })
        if postErr != nil { return fmt.Errorf("hold check: posting release for %s: %w", hold.HoldID, postErr) }

        fmt.Fprintln(stdout, hold.HoldID)
    }
    return nil
}
```

- [ ] **Step 12: Add targ structs, `BuildHoldGroup`, `BuildChatAckWaitTarget`, and update `Targets()` in `targets.go`**

```go
type ChatAckWaitArgs struct {
    Agent      string `targ:"flag,name=agent,desc=calling agent name"`
    Cursor     int    `targ:"flag,name=cursor,desc=line position to start watching from"`
    Recipients string `targ:"flag,name=recipients,desc=comma-separated recipient names"`
    MaxWait    int    `targ:"flag,name=max-wait,desc=seconds to wait for online-silent recipients (default 30)"`
    ChatFile   string `targ:"flag,name=chat-file,desc=override chat file path (testing only)"`
}

type HoldAcquireArgs struct {
    Holder    string `targ:"flag,name=holder,desc=agent acquiring the hold"`
    Target    string `targ:"flag,name=target,desc=agent being held"`
    Condition string `targ:"flag,name=condition,desc=auto-release condition: done:<agent>, first-intent:<agent>, lead-release:<tag>"`
    Tag       string `targ:"flag,name=tag,desc=workflow label for bulk operations (e.g. codesign-1, plan-review-1)"`
    ChatFile  string `targ:"flag,name=chat-file,desc=override chat file path (testing only)"`
}

type HoldReleaseArgs struct {
    HoldID   string `targ:"flag,name=hold-id,desc=hold ID returned by engram hold acquire"`
    ChatFile string `targ:"flag,name=chat-file,desc=override chat file path (testing only)"`
}

type HoldListArgs struct {
    Holder   string `targ:"flag,name=holder,desc=filter by holder agent name"`
    Target   string `targ:"flag,name=target,desc=filter by target agent name"`
    Tag      string `targ:"flag,name=tag,desc=filter by workflow tag"`
    ChatFile string `targ:"flag,name=chat-file,desc=override chat file path (testing only)"`
}

// REVIEWER NOTE: --tag is required. The lead skill has bulk tag-dissolve patterns
// (codesign, merge queue, plan-review/handoff) that call lead_release("tag"). Without
// --tag, the lead cannot enumerate hold IDs by tag to release them. Without it, these
// patterns break entirely in Phase 2.

type HoldCheckArgs struct {
    ChatFile string `targ:"flag,name=chat-file,desc=override chat file path (testing only)"`
}
```

**`BuildChatAckWaitTarget` — add to `BuildChatGroup` call site OR as a separate target added to the chat group:**

The `ack-wait` subcommand lives under `chat`, so add it inside `BuildChatGroup`:
```go
func BuildChatGroup(stdout, stderr io.Writer, stdin io.Reader) *targ.TargetGroup {
    return targ.Group("chat",
        // existing: post, watch, cursor targets ...
        targ.Targ(func(a ChatAckWaitArgs) {
            args := append([]string{"engram", "chat", "ack-wait"}, ChatAckWaitFlags(a)...)
            RunSafe(args, stdout, stderr, stdin)
        }).Name("ack-wait").Description("Block until all recipients ACK or WAIT; returns JSON result"),
    )
}
```

Add the corresponding `ChatAckWaitFlags` helper:
```go
func ChatAckWaitFlags(a ChatAckWaitArgs) []string {
    flags := BuildFlags(
        "--agent", a.Agent,
        "--recipients", a.Recipients,
        "--chat-file", a.ChatFile,
    )
    if a.Cursor != 0 {
        flags = append(flags, "--cursor", strconv.Itoa(a.Cursor))
    }
    if a.MaxWait != 0 {
        flags = append(flags, "--max-wait", strconv.Itoa(a.MaxWait))
    }
    return flags
}
```

**`BuildHoldGroup` — add this function, then add it to `Targets()`:**
```go
// BuildHoldGroup builds the targ group for hold subcommands.
func BuildHoldGroup(stdout, stderr io.Writer, stdin io.Reader) *targ.TargetGroup {
    return targ.Group("hold",
        targ.Targ(func(a HoldAcquireArgs) {
            args := append([]string{"engram", "hold", "acquire"}, HoldAcquireFlags(a)...)
            RunSafe(args, stdout, stderr, stdin)
        }).Name("acquire").Description("Place a hold on an agent (outputs UUID hold-id)"),
        targ.Targ(func(a HoldReleaseArgs) {
            args := append([]string{"engram", "hold", "release"}, HoldReleaseFlags(a)...)
            RunSafe(args, stdout, stderr, stdin)
        }).Name("release").Description("Release a hold by hold-id"),
        targ.Targ(func(a HoldListArgs) {
            args := append([]string{"engram", "hold", "list"}, HoldListFlags(a)...)
            RunSafe(args, stdout, stderr, stdin)
        }).Name("list").Description("List active (unreleased) holds"),
        targ.Targ(func(a HoldCheckArgs) {
            args := append([]string{"engram", "hold", "check"}, HoldCheckFlags(a)...)
            RunSafe(args, stdout, stderr, stdin)
        }).Name("check").Description("Evaluate auto-conditions and release met holds"),
    )
}
```

**Add hold flag helpers** (same pattern as `ChatWatchFlags`/`ChatCursorFlags`):
```go
func HoldAcquireFlags(a HoldAcquireArgs) []string {
    return BuildFlags("--holder", a.Holder, "--target", a.Target, "--condition", a.Condition, "--chat-file", a.ChatFile)
}
func HoldReleaseFlags(a HoldReleaseArgs) []string {
    return BuildFlags("--hold-id", a.HoldID, "--chat-file", a.ChatFile)
}
func HoldListFlags(a HoldListArgs) []string {
    return BuildFlags("--holder", a.Holder, "--target", a.Target, "--chat-file", a.ChatFile)
}
func HoldCheckFlags(a HoldCheckArgs) []string {
    return BuildFlags("--chat-file", a.ChatFile)
}
```

**Update `Targets()` to include `BuildHoldGroup`** — without this the targ CLI never exposes hold subcommands:
```go
func Targets(stdout, stderr io.Writer, stdin io.Reader) []any {
    run := func(subcmd string, flags []string) {
        args := append([]string{"engram", subcmd}, flags...)
        RunSafe(args, stdout, stderr, stdin)
    }
    return append(BuildTargets(run), BuildChatGroup(stdout, stderr, stdin), BuildHoldGroup(stdout, stderr, stdin))
}
```

Run `targ check-full` after Step 12. Commit with `/commit`.

---

## Task 4: Concurrent + Integration Tests

**Files:**
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 13: Write concurrent hold-acquire safety test**

```go
func TestRun_HoldAcquire_ConcurrentWritesSafe(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    dir := t.TempDir()
    chatFile := filepath.Join(dir, "chat.toml")

    const holdCount = 5
    var wg sync.WaitGroup
    wg.Add(holdCount)

    for i := range holdCount {
        go func(n int) {
            defer wg.Done()
            _ = cli.Run([]string{
                "engram", "hold", "acquire",
                "--chat-file", chatFile,
                "--holder", fmt.Sprintf("lead-%d", n),
                "--target", fmt.Sprintf("exec-%d", n),
            }, io.Discard, io.Discard, nil)
        }(i)
    }

    wg.Wait()

    data, readErr := os.ReadFile(chatFile)
    g.Expect(readErr).NotTo(HaveOccurred())
    if readErr != nil { return }

    var parsed struct{ Message []chat.Message `toml:"message"` }
    g.Expect(toml.Unmarshal(data, &parsed)).To(Succeed())
    g.Expect(parsed.Message).To(HaveLen(holdCount))
    for _, msg := range parsed.Message {
        g.Expect(msg.Type).To(Equal("hold-acquire"))
    }
}
```

- [ ] **Step 14: Write ack-wait + hold integration test**

```go
// TestRun_AckWait_With_HoldAcquire_E2E exercises the full intent→ack→hold→check→release cycle.
// This is the E2E scenario a lead runs: post intent, get ACK, acquire hold, executor works,
// executor posts done, hold check auto-releases, lead confirms hold cleared.
func TestRun_AckWait_With_HoldAcquire_E2E(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    dir := t.TempDir()
    chatFile := filepath.Join(dir, "chat.toml")

    // Step 1: Get cursor, post ack-wait in goroutine, then post ACK from executor side.
    var cursorOut bytes.Buffer
    g.Expect(cli.Run([]string{"engram", "chat", "cursor", "--chat-file", chatFile}, &cursorOut, io.Discard, nil)).To(Succeed())
    cursor := strings.TrimSpace(cursorOut.String())

    ackWaitDone := make(chan string, 1)
    go func() {
        var out bytes.Buffer
        _ = cli.Run([]string{
            "engram", "chat", "ack-wait",
            "--chat-file", chatFile,
            "--agent", "lead",
            "--cursor", cursor,
            "--recipients", "executor-1",
            "--max-wait", "5",
        }, &out, io.Discard, nil)
        ackWaitDone <- strings.TrimSpace(out.String())
    }()

    // Post ACK from executor-1 to lead.
    time.Sleep(50 * time.Millisecond) // ensure ack-wait is watching
    g.Expect(cli.Run([]string{
        "engram", "chat", "post",
        "--chat-file", chatFile,
        "--from", "executor-1",
        "--to", "lead",
        "--thread", "e2e",
        "--type", "ack",
        "--text", "proceeding",
    }, io.Discard, io.Discard, nil)).To(Succeed())

    // Step 2: Verify ack-wait resolved with ACK.
    select {
    case result := <-ackWaitDone:
        g.Expect(result).To(ContainSubstring(`"result":"ACK"`))
    case <-time.After(6 * time.Second):
        t.Fatal("ack-wait did not resolve within 6s")
    }

    // Step 3: Acquire hold — executor-1 must not be killed until done.
    var holdOut bytes.Buffer
    g.Expect(cli.Run([]string{
        "engram", "hold", "acquire",
        "--chat-file", chatFile,
        "--holder", "lead",
        "--target", "executor-1",
        "--condition", "done:executor-1",
    }, &holdOut, io.Discard, nil)).To(Succeed())
    holdID := strings.TrimSpace(holdOut.String())
    g.Expect(holdID).NotTo(BeEmpty())

    // Step 4: Verify hold is active.
    var listOut bytes.Buffer
    g.Expect(cli.Run([]string{"engram", "hold", "list", "--chat-file", chatFile}, &listOut, io.Discard, nil)).To(Succeed())
    g.Expect(listOut.String()).To(ContainSubstring(holdID))

    // Step 5: Executor-1 posts done.
    g.Expect(cli.Run([]string{
        "engram", "chat", "post",
        "--chat-file", chatFile,
        "--from", "executor-1",
        "--to", "all",
        "--thread", "e2e",
        "--type", "done",
        "--text", "task complete",
    }, io.Discard, io.Discard, nil)).To(Succeed())

    // Step 6: Hold check evaluates condition and auto-releases.
    var checkOut bytes.Buffer
    g.Expect(cli.Run([]string{"engram", "hold", "check", "--chat-file", chatFile}, &checkOut, io.Discard, nil)).To(Succeed())
    g.Expect(strings.TrimSpace(checkOut.String())).To(Equal(holdID))

    // Step 7: Verify hold is cleared.
    var listOut2 bytes.Buffer
    g.Expect(cli.Run([]string{"engram", "hold", "list", "--chat-file", chatFile}, &listOut2, io.Discard, nil)).To(Succeed())
    g.Expect(listOut2.String()).To(BeEmpty())
}
```

Run `targ check-full` after Step 14. Commit with `/commit`.

---

## Task 5: Skill Rewrites

**Files:**
- Modify: `skills/use-engram-chat-as/SKILL.md`
- Modify: `skills/engram-tmux-lead/SKILL.md`

**IMPORTANT:** Use `superpowers:writing-skills` skill for ALL skill edits. This enforces the TDD baseline→update→verify cycle.

### Deletions from `use-engram-chat-as`

- [ ] **Step 15: Delete ACK-wait subagent template (~35 lines)**

Remove the entire block starting "Spawn a background ACK-wait Agent using this template:" through the end of that code block.

- [ ] **Step 16: Delete online/offline detection bash block (~15 lines)**

Remove the `FIFTEEN_MIN_AGO`/`LAST_TS`/`grep` block in the Timing section.

- [ ] **Step 17: Delete Timing paragraph (~8 lines)**

Remove the paragraph: "Offline: 5s implicit ACK. Online + silent: post info, wait 30s, escalate."

- [ ] **Step 18: Update Background Monitor Pattern phase note**

Replace:
> "This subagent survives Phase 1. Phase 2 (`engram chat ack-wait`) will eliminate it entirely."

With:
> "This pattern survives through Phase 4. Phase 5 (`engram agent resume`) eliminates it by converting engram-agent to a stateless worker."

- [ ] **Step 19: Add ACK-Wait Protocol section to use-engram-chat-as**

Add after the intent protocol flow:

```
## ACK-Wait Protocol

After posting an intent, block on all recipients responding:

  PRE_CURSOR=$(engram chat cursor)                          # BEFORE intent post
  engram chat post --from <name> --to engram-agent --type intent ...
  RESULT=$(engram chat ack-wait \
    --agent <your-name> \
    --cursor $PRE_CURSOR \
    --recipients engram-agent[,reviewer,...] \
    --max-wait 30)

Parse response:
  RESULT_TYPE=$(echo "$RESULT" | jq -r '.result')    # ACK, WAIT, or TIMEOUT
  FROM=$(echo "$RESULT" | jq -r '.from')              # who objected (WAIT only)
  CURSOR=$(echo "$RESULT" | jq -r '.cursor')          # advance cursor
  TEXT=$(echo "$RESULT" | jq -r '.text')              # objection text (WAIT only)

If RESULT_TYPE=ACK: proceed immediately.
If RESULT_TYPE=WAIT: pause, engage argument protocol with FROM using TEXT.
  For argument continuation, use engram chat watch directly:
    RESULT=$(engram chat watch --agent <name> --cursor $CURSOR --type ack,wait --max-wait 30)
If RESULT_TYPE=TIMEOUT: recipient is online but silent. Post escalate to lead.

HARD RULE: Capture PRE_CURSOR=$(engram chat cursor) BEFORE posting the intent. Pass this cursor to ack-wait. Any ACK posted between your intent-post and ack-wait-start is safe because the cursor was captured first.

Two-pattern distinction:
- Active agent blocking on ack-wait (bounded): call engram chat ack-wait directly as Bash tool.
- Reactive agent watching for incoming messages (indefinite): use Background Monitor Pattern.
```

- [ ] **Step 20: Add hold-acquire/hold-release to Message Type Catalog**

Add to the type catalog table:
| `hold-acquire` | Binary-only — place a hold on an agent. Use `engram hold acquire`. Do NOT post with `engram chat post`. | N/A | No |
| `hold-release` | Binary-only — lift a hold from an agent. Use `engram hold release`. Do NOT post with `engram chat post`. | N/A | No |

- [ ] **Step 21: Expand startup binary check**

Add after the existing `engram chat post --help` check:
```bash
if ! engram chat ack-wait --help >/dev/null 2>&1; then
  echo "ERROR: engram binary missing 'chat ack-wait' subcommand. Run: targ build"; exit 1
fi
if ! engram hold acquire --help >/dev/null 2>&1; then
  echo "ERROR: engram binary missing 'hold' subcommand. Run: targ build"; exit 1
fi
```

### Deletions + Additions in `engram-tmux-lead`

- [ ] **Step 22: Delete Hold Registry in-context section (~10 lines)**

Remove the `Holds:` list with `task_id`/`cursor` fields from the lead's in-context state tracking.

- [ ] **Step 23: Delete per-hold background detection tasks (~20 lines)**

Remove the per-hold background Agent with fswatch loop watching for release conditions.

- [ ] **Step 24: Add Hold Commands section to engram-tmux-lead**

```
## Hold Commands

Acquire a hold before spawning agents that must stay alive:
  HOLD_ID=$(engram hold acquire --holder <holder-agent> --target <target-agent> [--condition <cond>])

Release a hold explicitly:
  engram hold release --hold-id $HOLD_ID

List active (unreleased) holds:
  engram hold list [--holder <name>] [--target <name>]

Evaluate auto-conditions and release met holds (manual in Phase 2):
  engram hold check

Condition DSL: done:<agent>, first-intent:<agent>, lead-release:<tag>
Note: In Phase 2, holds do not auto-release. hold check is manual-only.
      Auto-invocation from engram agent kill ships in Phase 3.

DONE state entry: agent posted done AND engram hold list --target <name> returns empty.
PENDING-RELEASE: agent posted done AND engram hold list --target <name> returns non-empty.

When to run hold check (Phase 2 manual trigger):
  After any agent posts done, run: engram hold check
  This releases any holds whose auto-condition is now met.
  Without this step, holds remain forever — agents stay "alive" indefinitely.
  Phase 3 automates this; for now the lead MUST call it manually after each done.
```

- [ ] **Step 25: Update old condition syntax in pattern recipes**

Replace:
- `{release: done('reviewer-1')}` → `--condition done:reviewer-1`
- `{release: first_intent('exec-1')}` → `--condition first-intent:exec-1`
- `{release: lead_release('tag')}` → `--condition lead-release:tag`

**WARNING: Incomplete.** The lead skill has the full dict form `{id: "h1", holder: "...", target: "...", release: done("..."), tag: "..."}` across Hold Definition, all pipeline phases, and all pattern recipes. Each must become a `HOLD_ID=$(engram hold acquire ...)` call. Steps 26–34 below cover the remainder.

- [ ] **Step 26: Rewrite Hold Definition block in `engram-tmux-lead`**

Replace `Hold { id, holder, target, release: Condition, tag }` struct + condition type table with:

```
#### Hold Definition

  HOLD_ID=$(engram hold acquire \
    --holder <holder-agent> \
    --target <target-agent> \
    [--condition done:<agent> | first-intent:<agent> | lead-release:<tag>] \
    [--tag <workflow-label>])

Conditions:
  done:<agent>          Auto-releases when agent posts type="done" after AcquiredTS
  first-intent:<agent>  Auto-releases on agent's first type="intent" after AcquiredTS
  lead-release:<tag>    Never auto-releases; requires explicit hold release
  (empty)               Never auto-releases; requires explicit hold release
```

- [ ] **Step 27: Update `lead_release` operation description**

Replace: "The lead posts an info message to chat documenting the release, then removes the holds from its registry."

With:
```
`lead_release(tag)` is now:
1. Post info to chat documenting the release
2. engram hold list --tag <tag>   (NDJSON output, one hold per line)
3. engram hold release --hold-id <id>  for each result
```

- [ ] **Step 28: Rewrite pipeline phase hold creation syntax (Section 4)**

Change all dict-based hold creations to `HOLD_ID=$(engram hold acquire ...)`. Delete all "Capture cursor and start hold detection background task" lines. Locations:

- Phase 1b step 2: `HOLD_PLANREV_N=$(engram hold acquire --holder reviewer-R --target planner-N --condition lead-release:plan-review-N --tag plan-review-N)`
- Phase 1b step 4: **delete** "Create hold detection background task for h-planrev-N"
- Phase 2 step 2b: `HOLD_HANDOFF_N=$(engram hold acquire --holder exec-N --target planner-N --condition first-intent:exec-N --tag plan-handoff-N)` (no background task)
- Phase 2 step 2c: `lead_release("plan-review-N")` → `engram hold release --hold-id $HOLD_PLANREV_N`
- Phase 3 step 3: `HOLD_IMPLREV_N=$(engram hold acquire --holder reviewer-R --target exec-N --condition done:reviewer-R --tag impl-review-N)`
- Phase 3 step 5: **delete** "Create hold detection background task for h-implrev-N"
- Merge queue: dict form → `HOLD_MERGE_K=$(engram hold acquire --holder lead --target exec-K --condition lead-release:merge-N-exec-K --tag merge-queue-N)`
- Fan-in: dict form → `HOLD_FANIN_K=$(engram hold acquire --holder synthesizer-N --target researcher-K --condition done:synthesizer-N --tag synthesis-N)`
- Barrier: dict form → `HOLD_BARRIER_K=$(engram hold acquire --holder lead --target member-K --condition lead-release:codesign-N --tag codesign-N)`

- [ ] **Step 29: Specify `engram hold list` output format**

The plan specifies no output format for `engram hold list`. Add to the Step 11 CLI wiring:

`runHoldList` outputs NDJSON, one object per active hold:
```json
{"hold-id":"h-12345","holder":"reviewer-1","target":"exec-1","condition":"done:reviewer-1","tag":"plan-review-1","acquired-ts":"2026-04-05T12:00:00Z"}
```
Empty list = no output (exit 0). Add `TestRun_HoldList_OutputFormat` to Step 10 asserting NDJSON format.

- [ ] **Step 30: Delete Section 6.4 Rules 7–10 from `engram-tmux-lead` and patch Rule 3**

Delete (these govern per-hold background tasks that no longer exist):
- Rule 7: "One persistent background task per hold"
- Rule 8: "Drain on lead_release"
- Rule 9: "Hold detection tasks do not replace each other"
- Rule 10: "Hold watchers replace standard agent wait tasks"

Patch Rule 3: remove "and all hold detection task IDs" from drain-on-shutdown text.

**Failing to delete these is a correctness bug** — executors will follow instructions to spawn background tasks that Phase 2 eliminated.

- [ ] **Step 31: Rewrite "When a hold fires" section in `engram-tmux-lead`**

Replace the background-task-drain section with:

```
#### When a Hold Is Released

Explicit release: engram hold release --hold-id $HOLD_ID

Auto-release check (manual in Phase 2):
  engram hold check   # evaluates conditions, releases met holds, prints released hold-ids

After any release:
1. engram hold list --target <target-agent>
2. Empty → send shutdown → KILL-PANE → DONE
3. Non-empty → stays in PENDING-RELEASE
```

- [ ] **Step 32: Update Common Mistakes entry in `use-engram-chat-as`**

Replace the entry "Let ACK-wait subagent re-derive cursor at startup" with:

> "Let ack-wait miss early ACK | Critical bug: same race. Capture `PRE_CURSOR=$(engram chat cursor)` BEFORE posting intent. Pass `--cursor $PRE_CURSOR` to `engram chat ack-wait`. ACKs posted between intent-post and ack-wait invocation are captured because the cursor was taken first."

- [ ] **Step 33: Update Hold Commands section (Step 24) with --tag and bulk release**

Update `engram hold list` line to show `[--tag <label>]` and add:
```
Bulk release by tag (lead_release pattern):
  engram hold list --tag <label> | jq -r '.["hold-id"]' | xargs -I{} engram hold release --hold-id {}
```

- [ ] **Step 34: Update Agent Lifecycle step 11 in `use-engram-chat-as`**

Replace steps 11a–b with:
```
11. If acting:
    a. PRE_CURSOR=$(engram chat cursor)   # BEFORE posting intent
    b. Post intent to (engram-agent + any other relevant recipients)
    c. RESULT=$(engram chat ack-wait --agent <name> --cursor $PRE_CURSOR --recipients <names> --max-wait 30)
       Parse: RESULT_TYPE / FROM / CURSOR / TEXT (see ACK-Wait Protocol)
    d. If ACK: proceed. If WAIT: engage argument protocol. If TIMEOUT: escalate to lead.
```

Run `targ check-full` after all Step 25–34 skill edits. Commit with `/commit`.

---

## Verification Protocol

These are the explicit steps to run before declaring Phase 2 done. Run them in order.

**Step 1 — Binary smoke test:**
```bash
targ build
engram chat ack-wait --help
engram hold acquire --help
engram hold release --help
engram hold list --help
engram hold check --help
```
Verify: no `flag provided but not defined` errors; all subcommands recognized; exit code 0.

**Step 2 — ACK-wait correctness (two terminals):**
```bash
# Terminal 1:
CURSOR=$(engram chat cursor)
engram chat ack-wait --agent tester --cursor $CURSOR --recipients responder --max-wait 10
# Terminal 2 (within 10s):
engram chat post --from responder --to tester --thread smoke --type ack --text 'test ACK'
```
Verify: Terminal 1 prints `{"result":"ACK","cursor":N}` and exits immediately (not after 10s timeout).

**Step 3 — WAIT response:**
```bash
# Terminal 1: (same ack-wait command)
# Terminal 2:
engram chat post --from responder --to tester --thread smoke --type wait --text 'test WAIT'
```
Verify: Terminal 1 prints `{"result":"WAIT","from":"responder","cursor":N,"text":"test WAIT"}`.

**Step 4 — Offline implicit ACK:**
```bash
CURSOR=$(engram chat cursor)
engram chat ack-wait --agent tester --cursor $CURSOR --recipients nonexistent-agent --max-wait 10
```
Verify: exits in ~5s (not 10s) with `{"result":"ACK","cursor":N}`. Offline agent = implicit ACK after 5s.

**Step 5 — --max-wait flag regression (Phase 1 lesson):**
```bash
engram chat ack-wait --agent tester --cursor 0 --recipients nonexistent --max-wait 1
```
Verify: exits cleanly without flag parse error. This guards against targ reserved-flag collision.

**Step 6 — Hold lifecycle:**
```bash
HOLD_ID=$(engram hold acquire --holder lead --target executor-1 --condition done:lead)
echo "HOLD_ID: $HOLD_ID"    # expect a UUID
engram hold list             # expect one entry
engram hold release --hold-id $HOLD_ID
engram hold list             # expect empty
```
Verify: hold-id is a valid UUID; list shows correct count; release removes it.

**Step 7 — hold check with met condition:**
```bash
HOLD_ID=$(engram hold acquire --holder lead --target executor-1 --condition done:lead)
engram chat post --from lead --to all --thread smoke --type done --text 'done'
engram hold check            # expect hold-id output (auto-released)
engram hold list             # expect empty
```
Verify: hold check evaluates condition and posts hold-release.

**Step 8 — Binary-verifiable session sanity (no live Claude session required):**
```bash
# Verify no binary spawns background processes (ack-wait should block in foreground and exit):
CURSOR=$(engram chat cursor)
timeout 6 engram chat ack-wait --agent tester --cursor $CURSOR --recipients ghost-agent --max-wait 5
echo "Exit: $?"  # expect 0 (clean exit after 5s offline implicit ACK)

# Verify hold list and check produce no output when no holds exist:
engram hold list
echo "List exit: $?"   # expect 0, no output

# Verify hold check is a no-op when there are no holds:
engram hold check
echo "Check exit: $?"  # expect 0, no output

# Verify new message types appear in TOML format (spot-check):
HOLD_ID=$(engram hold acquire --holder lead --target executor-1)
grep 'type = "hold-acquire"' ~/.local/share/engram/chat/*.toml | tail -1
```
Verify: all commands exit 0; hold-acquire message visible in TOML; no stray background processes.

**NOTE:** "No background ACK-wait Agent tool calls" is a behavioral property of skill compliance, not binary behavior — it cannot be verified by running commands. It is verified by the test suite (no goroutine leak in ack-wait tests) and by the skill TDD baseline in Task 5 (writing-skills enforces this via behavior test).

---

## Done Criteria

- [ ] `targ check-full` passes (all linters, all tests, coverage thresholds)
- [ ] `engram chat ack-wait` exits with `{"result":"ACK","cursor":N}` when all recipients ACK
- [ ] `engram chat ack-wait` exits with `{"result":"WAIT","from":...}` when any recipient WAITs
- [ ] `engram chat ack-wait` exits with `{"result":"TIMEOUT","recipient":...}` for online-silent recipients
- [ ] `engram chat ack-wait` exits with implicit ACK after ~5s for offline recipients
- [ ] `--max-wait` flag works without targ collision (regression from Phase 1 lesson)
- [ ] `--help` exits 0 on all subcommands
- [ ] `engram hold acquire` posts hold-acquire message, outputs UUID hold-id
- [ ] `engram hold release` posts hold-release message
- [ ] `engram hold list` returns only active (unmatched) holds, filterable by holder/target
- [ ] `engram hold check` evaluates conditions and auto-releases met holds
- [ ] `engram hold check` does NOT auto-release `lead-release:<tag>` conditions
- [ ] Concurrent hold-acquire test passes (5 concurrent writers, valid TOML after)
- [ ] Startup binary check in use-engram-chat-as covers ack-wait + hold subcommands
- [ ] ACK-wait subagent template deleted from use-engram-chat-as
- [ ] Online/offline detection bash deleted from use-engram-chat-as
- [ ] Background Monitor Pattern phase note corrected: "Phase 5" not "Phase 2"
- [ ] hold-acquire/hold-release in Message Type Catalog (binary-only note)
- [ ] Hold Registry + per-hold background tasks deleted from engram-tmux-lead
- [ ] Hold Commands section added to engram-tmux-lead with Phase 2 manual-only note
- [ ] HoldRecord has `Tag` field; `engram hold acquire` has `--tag` flag; `engram hold list` has `--tag` filter
- [ ] `engram hold list` outputs NDJSON (one hold per line) — output format specified and tested
- [ ] Hold Definition block rewritten to show `engram hold acquire` CLI syntax
- [ ] `lead_release` description updated: uses `engram hold list --tag` + `engram hold release`
- [ ] Section 4 pipeline phases rewritten: dict-based hold syntax replaced with `HOLD_ID=$(engram hold acquire ...)`
- [ ] Section 4: all "hold detection background task" creation lines deleted
- [ ] Section 6.4 Rules 7–10 deleted; Rule 3 drain-on-shutdown updated
- [ ] "When a hold fires" section rewritten: uses `engram hold check` / `engram hold release`
- [ ] Common Mistakes entry updated: ACK-wait subagent cursor → ack-wait CLI cursor
- [ ] Hold Commands section shows --tag flag and bulk release pattern
- [ ] Agent Lifecycle step 11 updated: uses `engram chat ack-wait` instead of "wait for explicit ACK"
- [ ] Skill still E2E-functional: agent lifecycle, intent protocol, hold commands all work
- [ ] E2E integration test (Step 14) passes: intent→ack→hold-acquire→done→hold-check→hold-cleared
- [ ] `engram hold check` manual trigger documented in engram-tmux-lead: "after each agent done, run hold check"
- [ ] Verification Step 8 commands run without error; hold-acquire TOML message visible in chat file
