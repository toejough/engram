# Plan: Parallelize correct + surface in hooks (#450)

## Problem

`hooks/user-prompt-submit.sh` runs `engram correct` and `engram surface` sequentially (lines 77-83). Both involve LLM calls (correct: detect + extract; surface: semantic gate), so the sequential pipeline adds unnecessary latency on every user prompt.

## Analysis

### Independence Assessment

**correct pipeline** (5 steps):
1. Detect — fast-path keywords OR Haiku classification
2. Context — read transcript tail
3. BM25 — read all memories, score
4. Extract — Sonnet LLM call
5. Disposition — write memory file (create/update/merge)

**surface pipeline** (prompt mode):
1. List — read all memories
2. BM25 + filter — score, threshold, cap
3. Semantic gate — Haiku LLM call
4. Write pending evaluations — ReadModifyWrite to surfaced memory files
5. Output — JSON with summary + context

**Shared inputs:** `$USER_MESSAGE`, `$ENGRAM_BIN`, `$SESSION_ID`, memory files directory.
**Shared outputs:** None — each writes to separate stdout/stderr. Both contribute independently to `FINAL_SYS`/`FINAL_CTX`.

**Read overlap:** Both read memory files from the same directory. Both read policy.toml. No conflict (concurrent reads are safe).

**Write overlap:** Both write to memory files via `ReadModifyWrite` (atomic: temp file + rename). correct writes disposition results; surface writes pending evaluations + surfaced_count. If both target the same memory file simultaneously, last writer wins (no corruption, but one mutation lost). This is extremely unlikely (correct targets the corrected memory; surface targets relevant memories) and the consequence is minor (one counter increment lost, recoverable on next run). Acceptable for now; file-level locking can be added later if needed.

**Ordering:** Neither depends on the other's output. The shell script merges their outputs into a single JSON response after both complete.

**Verdict:** Safe to parallelize.

### Approach: Shell-level parallelization

Change only `hooks/user-prompt-submit.sh`. No Go code changes.

Why shell-level (not a new Go subcommand):
- Minimal change — one file, ~15 lines modified
- The main latency win is from concurrent LLM calls (seconds), not from shared process startup (~100ms)
- Keeps correct and surface as independent, testable subcommands
- Go-level optimization (shared policy/token/memory listing) is a separate, additive improvement

## Tasks

### Task 1: Parallelize correct and surface in user-prompt-submit.sh

**Files:** `hooks/user-prompt-submit.sh`

**Changes:**

Replace lines 69-86 (the sequential block):

```bash
CORRECT_OUTPUT=""
CORRECT_ERR=""
SURFACE_OUTPUT=""
SURFACE_ERR=""

if [[ -n "$USER_MESSAGE" ]]; then
    TMPFILE=$(mktemp)

    CORRECT_OUTPUT=$("$ENGRAM_BIN" "${CORRECT_ARGS[@]}" 2>"$TMPFILE") || true
    CORRECT_ERR=$(cat "$TMPFILE" 2>/dev/null)

    # UC-2: Surface relevant memories
    SURFACE_OUTPUT=$("$ENGRAM_BIN" surface --mode prompt \
        --message "$USER_MESSAGE" --session-id "$SESSION_ID" --format json 2>"$TMPFILE") || true
    SURFACE_ERR=$(cat "$TMPFILE" 2>/dev/null)

    rm -f "$TMPFILE"
fi
```

With parallel execution using separate temp files:

```bash
CORRECT_OUTPUT=""
CORRECT_ERR=""
SURFACE_OUTPUT=""
SURFACE_ERR=""

if [[ -n "$USER_MESSAGE" ]]; then
    CORRECT_STDOUT=$(mktemp)
    CORRECT_STDERR=$(mktemp)
    SURFACE_STDOUT=$(mktemp)
    SURFACE_STDERR=$(mktemp)

    # Run correct and surface in parallel
    "$ENGRAM_BIN" "${CORRECT_ARGS[@]}" >"$CORRECT_STDOUT" 2>"$CORRECT_STDERR" &
    CORRECT_PID=$!

    "$ENGRAM_BIN" surface --mode prompt \
        --message "$USER_MESSAGE" --session-id "$SESSION_ID" --format json \
        >"$SURFACE_STDOUT" 2>"$SURFACE_STDERR" &
    SURFACE_PID=$!

    wait "$CORRECT_PID" || true
    wait "$SURFACE_PID" || true

    CORRECT_OUTPUT=$(cat "$CORRECT_STDOUT" 2>/dev/null)
    CORRECT_ERR=$(cat "$CORRECT_STDERR" 2>/dev/null)
    SURFACE_OUTPUT=$(cat "$SURFACE_STDOUT" 2>/dev/null)
    SURFACE_ERR=$(cat "$SURFACE_STDERR" 2>/dev/null)

    rm -f "$CORRECT_STDOUT" "$CORRECT_STDERR" "$SURFACE_STDOUT" "$SURFACE_STDERR"
fi
```

**Key differences:**
1. Four separate temp files instead of one shared one (eliminates stderr clobbering)
2. Both commands run as background processes with `&`
3. `wait` collects exit statuses (with `|| true` to prevent `set -e` abort)
4. Results read from files after both processes complete

**Verification:**
- `targ check-full` (no Go code changed, but verify nothing broke)
- Manual test: confirm hook still produces valid JSON output
- Commit with `AI-Used: [claude]` trailer

### Known Risks

1. **Theoretical write race:** If correct's disposition and surface's pending-eval both target the same memory file simultaneously, last writer wins. Extremely unlikely; consequence is one lost counter increment. Acceptable.
2. **Process cleanup:** If the hook is killed mid-execution, background processes could orphan. `set -e` + `wait` mitigates this. The hook timeout (30s) will kill the process group.
3. **Resource usage:** Two concurrent `engram` processes instead of one sequential pair. Each process is lightweight (Go binary + one LLM call). No concern.
