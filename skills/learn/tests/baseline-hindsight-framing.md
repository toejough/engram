# Baseline pressure test — explicit save-request with hindsight-baked wording

## Scenario

The user says: "remember: when fixing context cancellation in concurrent code, always pass the parent context through to spawned goroutines."

## Expected current-skill behavior (PASS)

This is an explicit save-request ("remember"). The current skill writes it.

- **Step 1 — sweep:** `engram ingest --auto`.
- **Step 2 — scan:** explicit save-request detected. Write one `engram learn feedback` note.
- **`--situation` phrasing:** the skill says to phrase the situation "the way a future task would be described" — a retrieval handle, not a hindsight description. The agent must reframe "when fixing context cancellation" → "when writing concurrent Go code that spawns goroutines with context" (pre-fix phrasing). This is the only wording judgment the current skill exercises.
- **Result:** 1 write. Situation is forward-looking (pre-fix phrasing), not hindsight-baked.

```bash
engram learn feedback --slug parent-context-propagates-to-goroutines \
  --position top \
  --source "session <date>, context: user correction on goroutine context propagation" \
  --situation "when writing concurrent Go code that spawns goroutines with context" \
  --behavior "not propagating the parent context to spawned goroutines" \
  --impact "cancellation signals do not reach goroutines; they outlive their cancellation scope" \
  --action "always pass parent context (or context.WithCancel/WithTimeout(ctx)) through to spawned goroutines"
```

## Failure modes that must FAIL this test

- Not writing anything (user said "remember" — this must be written).
- Writing the situation as "when fixing context cancellation" verbatim (hindsight baked in).
- Running Gate 1 / Gate 2 / Gate 3 explicit checks.
- Running `engram transcript` or writing an episode.

## What changed from the pre-v2 expected behavior

Pre-v2: run Gate 2 (Activity+Domain) → FAIL hindsight → explicit reframe-and-rerun sequence. Current skill: write it (it IS an explicit save-request); the only judgment is phrasing `--situation` in forward-looking language. The gate machinery is gone — the test now checks that `--situation` is retrieval-shaped, not that a formal gate was invoked.
