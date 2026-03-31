# SBIA Step 4: Evaluate

> **Depends on:** Step 2 (extract pipeline) ✅, Step 3 (surface upgrades) ✅

**Goal:** Replace LLM self-report (`engram feedback`) with automated Haiku evaluation at the async stop hook. Memories accumulate effectiveness data without relying on the agent to grade itself.

---

## Architecture

```
stop.sh (async, 120s timeout)
  → reads hook JSON from stdin (transcript_path, session_id)
  → calls: engram evaluate --transcript-path <path> --session-id <id>

engram evaluate:
  1. Scan all memory files for pending_evaluations matching session_id
  2. Read transcript delta (full session content via context.Strip)
  3. For each pending evaluation:
     a. Build Haiku prompt: memory SBIA fields + transcript context
     b. Haiku assesses: situation relevant? action taken?
     c. Increment followed_count / not_followed_count / irrelevant_count
     d. Remove consumed pending_evaluation entry
  4. Write updated memory files atomically
```

---

## Steps

### S1: Add `evaluate_haiku` prompt to policy (TDD)

**Files:** `internal/policy/policy.go`, `internal/policy/policy_test.go`

- Add `EvaluateHaikuPrompt string` field to `Policy` struct
- Add `defaultEvaluateHaikuPrompt` constant — template with `{situation}`, `{behavior}`, `{action}`, `{transcript}` placeholders
- Add `EvaluateHaiku string` to `policyFilePrompts` struct
- Wire into `mergePrompts()`
- Test: `Load()` returns default prompt; file override replaces it

**Prompt content:**
```
You are evaluating whether a memory was relevant and followed during a conversation.

Memory:
- Situation: {situation}
- Behavior to avoid: {behavior}
- Action: {action}

Transcript (agent's response after memory was surfaced):
{transcript}

Assess:
1. Was the situation relevant to what was happening in the conversation? (yes/no)
2. If relevant, was the action taken by the agent? (yes/no)

Return exactly one of: FOLLOWED, NOT_FOLLOWED, IRRELEVANT
Do not explain. Return only the verdict.
```

### S2: Add evaluate package with core logic (TDD)

**Files:** `internal/evaluate/evaluate.go`, `internal/evaluate/evaluate_test.go`

**Types:**
```go
// HaikuCallerFunc matches the signature used elsewhere.
type HaikuCallerFunc func(ctx context.Context, model, system, user string) (string, error)

// TranscriptReaderFunc reads and strips a transcript file.
type TranscriptReaderFunc func(path string) (string, error)

// MemoryScannerFunc finds memories with pending evaluations for a session.
type MemoryScannerFunc func(sessionID string) ([]PendingMemory, error)

// ModifyFunc atomically updates a memory file.
type ModifyFunc func(path string, mutate func(*memory.MemoryRecord)) error

type PendingMemory struct {
    Path   string
    Record *memory.MemoryRecord
    Eval   memory.PendingEvaluation
}

type Evaluator struct {
    caller   HaikuCallerFunc
    scanner  MemoryScannerFunc
    modifier ModifyFunc
    reader   TranscriptReaderFunc
    prompt   string
    model    string
}
```

**Core function:** `func (e *Evaluator) Run(ctx context.Context, sessionID, transcriptPath string) ([]Result, error)`

- Calls scanner to find pending evals for this session
- Reads transcript via reader
- For each pending eval: calls Haiku, parses verdict, updates counters via modifier
- Returns results slice for logging/testing

**Verdict parsing:** Trim whitespace, case-insensitive match on FOLLOWED/NOT_FOLLOWED/IRRELEVANT. Unknown → skip (don't corrupt counters on garbage response).

**Tests:**
- Haiku returns FOLLOWED → followed_count++, pending eval removed
- Haiku returns NOT_FOLLOWED → not_followed_count++, pending eval removed
- Haiku returns IRRELEVANT → irrelevant_count++, pending eval removed
- Haiku returns garbage → pending eval NOT removed (retry next session)
- No pending evals → no Haiku calls, no errors
- Multiple memories with pending evals → each evaluated independently
- Haiku error on one → continue with others (errors.Join)

### S3: Add memory scanner function (TDD)

**Files:** `internal/evaluate/scanner.go`, `internal/evaluate/scanner_test.go`

**Function:** `func NewFileScanner(dir string, readFile func(string, error) ([]byte, error), listDir func(string) ([]os.DirEntry, error)) MemoryScannerFunc`

- Lists `*.toml` files in `dir/memories/`
- Reads each, unmarshals, checks for pending_evaluations where session_id matches
- Returns `[]PendingMemory` with path + record + matching eval entry

**Tests:**
- Memory with matching session_id → returned
- Memory with different session_id → not returned
- Memory with no pending evals → not returned
- Multiple pending evals on one memory, only one matches → returns that one
- Empty directory → empty result

### S4: Wire CLI command `engram evaluate` (TDD)

**Files:** `internal/cli/cli.go`, `internal/cli/evaluate.go`, `internal/cli/evaluate_test.go`

- Add `case "evaluate":` to CLI dispatch
- Parse flags: `--transcript-path`, `--session-id`, `--data-dir`
- Load policy for `evaluate_haiku` prompt
- Create Anthropic caller (reuse `makeAnthropicCaller`)
- Build transcript reader (reuse `context.Strip`)
- Build scanner (from S3)
- Build modifier (reuse `memory.NewModifier`)
- Construct `evaluate.Evaluator`, call `Run()`
- Print results as JSON for hook consumption

**Tests:**
- Missing `--session-id` → error
- Missing `--transcript-path` → error
- Successful evaluation → counters updated, pending evals removed

### S5: Update `stop.sh` to call `engram evaluate` (no TDD needed — shell)

**File:** `hooks/stop.sh`

- Read hook JSON from stdin (same pattern as `stop-surface.sh`)
- Extract `transcript_path` and `session_id`
- Call `engram evaluate --transcript-path <path> --session-id <id>`
- Exit 0 always (async hook, fire-and-forget)
- Include binary build check (same pattern as other hooks)

### S6: Update surface injection preamble — remove `engram feedback` instruction

**Files:** `internal/policy/policy.go`, `internal/policy/policy_test.go`

- Change `defaultSurfaceInjectionPreamble` to remove the `engram feedback` call instruction
- New preamble: `"[engram] Memories — for any relevant memory, call \`engram show --name <name>\` for full details:"`
- Test: verify default preamble no longer mentions `feedback`

### S7: Remove `engram feedback` command

**Files:** `internal/cli/cli.go`, `internal/cli/feedback.go`, `internal/cli/feedback_test.go`

- Remove `case "feedback":` from CLI dispatch
- Delete `feedback.go` and `feedback_test.go`
- Verify no other code references `RunFeedback` or `applyFeedbackCounters`

### S8: Clean up stale files

- Remove `surfacing-log.jsonl` if present in data dir
- Remove `learn-offset.json` if present in data dir
- Check for any other stale references to removed commands

### S9: Run `targ check-full`, fix coverage/lint

- All 8 checks must pass
- Coverage ≥ 80% on all packages including new `evaluate` package

---

## Hook State After Step 4

| Hook | Script | What it calls | Status |
|------|--------|---------------|--------|
| SessionStart | `session-start.sh` | `engram maintain` (async bg) | Unchanged |
| UserPromptSubmit | `user-prompt-submit.sh` | `engram correct`, `engram surface` | Unchanged |
| Stop (sync) | `stop-surface.sh` | `engram surface --mode stop` | Unchanged |
| Stop (async) | `stop.sh` | `engram evaluate` | **Updated** |

## Invariants

- Memory files are the sole source of truth for evaluation state
- Pending evaluations are consumed per-session (other sessions' entries remain)
- Unknown Haiku verdicts are silently skipped (pending eval preserved for retry)
- Haiku errors on individual memories don't block evaluation of others
- No LLM self-report anywhere in the system
