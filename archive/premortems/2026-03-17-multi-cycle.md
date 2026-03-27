# Premortem: Multi-Cycle Development (Cycles 2–N)

**Date:** 2026-03-17
**Scenario:** "It is 2026-05-01. We completed Cycles 2 and 3 and several more like them.
The project is harder to work on than it was in March. What went wrong?"

---

## Feature

Engram continues to grow across test hygiene (Cycle 2), B-2 integration (Cycle 3),
and further cycles. Success = new features ship cleanly, existing behavior is reliable,
and the spec → code traceability stays trustworthy.

---

### Failure 1: Spec Tree Becomes Documentation Theater

**What went wrong:**
`traced verify` has been broken since `state.toml` was removed (`3c3a852`). After
Cycles 2–3, the spec tree has ~400 T-items, ~100 ARCH items. Nobody updates hashes
after edits. Spec bodies drift from code. Engineers stop trusting the spec and treat it
as read-only history. New features get coded without spec items. Traceability
collapses silently.

**Principle violated:** Spec as single source of truth (the entire traced workflow).

**What would have caught it:** `traced verify` passing in CI before any merge.
Currently it fails with `state.toml not found` — already broken, already invisible.

**Remediation:** Fix `traced stamp` / `traced verify` compatibility with TOML-only
spec (no state.toml). Until fixed, every cycle compounds drift. Filed against
`toejough/traced`.

**Likelihood × Impact:** HIGH × HIGH = **#1 risk**

---

### Failure 2: Test Brittleness Taxes Every Structural Change

**What went wrong:**
The Cycle 1 retro already measured this: one 10-line rename change required updating
16 test handlers (fix/feat ratio 33%). The evaluator test file is now 1,182 lines.
Cycle 3 adds P4-full (cluster dedup) and P5-full (link recompute) — both touch the
surfacing → evaluate → registry pipeline. Each structural change cascades into mass
test updates. Engineers start avoiding refactors. Dead paths accumulate. Maintainability
degrades.

**Principle violated:** "Test hard-to-test code by refactoring for DI, not writing
integration tests around I/O."

**What would have caught it:** A test infrastructure helper (`testFS` or rename-aware
`makeTestEvaluator`) that decouples test logic from internal path naming. Issue #321
was filed but not addressed before Cycle 3.

**Remediation:** Resolve #321 before Cycle 3. Add a `testFS` struct shared by
evaluator tests that handles rename→redirect transparently. Bound: 1 day of work.

**Likelihood × Impact:** HIGH × MEDIUM = **#2 risk**

---

### Failure 3: Concurrent Stop Hook Writes Corrupt Evaluations

**What went wrong:**
The `evaluations/<timestamp>.jsonl` atomic write uses `time.Now()` for the filename.
Two stop.sh invocations firing within the same second (rapid turn rate, long stop.sh
runtime) collide: both write to `evaluations/2026-04-01T10-30-00Z.jsonl.tmp`, the
second clobbers the first, one turn's outcomes are silently lost. The registry has
similar no-lock TOML writes: two concurrent `learn` calls can corrupt the same
memory file. In active sessions (multiple quick turns), this produces mysterious
missing evaluations and truncated memory TOML files.

**Principle violated:** DI-everywhere doesn't prevent OS-level race conditions on
shared files.

**What would have caught it:** A concurrent-write integration test (two goroutines
invoking the evaluator simultaneously against a shared tempdir). Not currently in
test suite.

**Remediation:** Use sub-second timestamp (nanoseconds) or UUID suffix for evaluation
log filenames. Add registry file locking or append-only semantics for learn.

**Likelihood × Impact:** MEDIUM × HIGH = **#3 risk**

---

### Failure 4: Coverage Metric Passes but Error Paths Are Untested

**What went wrong:**
`targ check-coverage-for-fail` measures per-function coverage at 80% threshold, but
`internal/evaluate` has 4.2% statement coverage. The per-function check passes because
small functions (option setters, constants) are "covered" by any call. All
fire-and-forget error branches (`|| true` in stop.sh, `_ = e.registry.RecordEvaluation`)
are untested. Cycle 3 adds P4-full cluster merge — 300+ lines of error-branch logic.
A silent failure in merge (e.g., wrong survivor chosen) goes undetected. Real memory
data is permanently altered with no observable error.

**Principle violated:** "Content quality > mechanical sophistication."

**What would have caught it:** Statement coverage threshold (not just function
threshold), plus explicit tests for each fire-and-forget error path.

**Remediation:** Add per-package statement coverage floor (50%+) to `targ check-full`.
File issue. Pre-Cycle 3: add tests for merge error paths specifically.

**Likelihood × Impact:** MEDIUM × MEDIUM = **#4 risk**

---

### Failure 5: No End-to-End Integration Test for the Binary

**What went wrong:**
All tests mock the LLM and I/O via DI. The `engram` binary is tested only at the
unit level. When Cycle 3 integrates P4-full (cluster dedup) and P5-full (link
recompute) into `runSignalDetect`, the CLI wiring changes. A test passes in unit tests
but the binary panics at runtime (e.g., nil interface in CLI wiring, wrong arg order
to `MergeExecutor`). The hook fires silently (`|| true` in stop.sh) and users see no
output, no error, no improvement in memory quality. The bug lives for weeks.

**Principle violated:** "Passing tests ≠ usable system — verify entry points compile,
wiring uses real dependencies, a user can actually run it."

**What would have caught it:** A smoke test: build the binary, run `engram evaluate
--data-dir /tmp/empty` and verify exit code 0. Currently no such test exists.

**Likelihood × Impact:** MEDIUM × HIGH = **#5 risk**

---

## Priority Matrix

| Failure | Likelihood | Impact | Priority |
|---------|-----------|--------|----------|
| 1: Spec drift via broken traced verify | High | High | 🔴 P0 |
| 2: Test brittleness taxes structural changes | High | Medium | 🔴 P1 |
| 3: Concurrent write corruption | Medium | High | 🟡 P2 |
| 4: Coverage theater misses error paths | Medium | Medium | 🟡 P3 |
| 5: No binary smoke test | Medium | High | 🟡 P4 |

---

## Mitigations (Top 2)

### Mitigation A: Fix `traced verify` (Failure 1)

This is already filed against `toejough/traced`. Until it's fixed, add a manual
"spec stamp check" to the Cycle gate: before closing a cycle, run `traced stamp` and
commit. If `traced stamp` fails, treat the cycle as incomplete.

**Process gate (immediate):** Add to roadmap cycle checklist: "run `traced stamp`
before declaring cycle complete."

### Mitigation B: Resolve #321 Before Cycle 3 (Failure 2)

The `makeTestEvaluator` test brittleness will compound with every pipeline change in
Cycle 3. Fixing it before P4-full/P5-full work bounds the rework cost.

**Action:** Schedule #321 as Cycle 2 item (it's small — 1 day). Add `testFS` helper.
Verify by re-running the 16-handler change with zero manual updates needed.

---

## Files Created

- `docs/premortems/2026-03-17-multi-cycle.md` (this file)
