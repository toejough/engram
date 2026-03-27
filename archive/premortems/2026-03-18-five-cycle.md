# Premortem: Five More Cycles of Mixed Fixup + Feature Work

**Date:** 2026-03-18
**Scenario:** "It is 2026-06-01. We completed five more cycles — a mix of fixup and feature
work through B-2 and beyond. The project is in trouble. What went wrong?"

## Prior Premortem Status

The 2026-03-17 premortem identified 5 risks. Current standing:

| Prior Risk | Status |
|-----------|--------|
| P0: `traced verify` broken | 🔴 Still open — #324 filed, not fixed |
| P1: Test brittleness (makeTestEvaluator) | ✅ Resolved — #321 done |
| P2: Concurrent write corruption | 🔴 Still open — no mitigation |
| P3: Coverage theater | 🟡 Partially addressed — #323 open |
| P4: No binary smoke test | 🟡 Partially addressed — #322 open |

Three of the five original risks persist. This premortem focuses on **new risks**
that emerge over a longer multi-cycle horizon.

---

## Feature

Five more cycles of engram development, mixing fixup cycles (correctness, test
hygiene) with feature cycles (B-2: cluster dedup, link recompute, TF-IDF, UC-34,
UC memory unification). Success = features ship correctly, memories improve in
quality, no data loss, spec stays trustworthy.

---

### Failure 1: Cluster Merge Permanently Corrupts Memory Files

**What went wrong:**
B-2 introduces P4-full (cluster dedup + cross-source suppression) and P5-full
(re-compute links after merge). Both operations write to real `.toml` files in the
user's `memories/` directory. The cluster merge picks a "survivor" memory and
deletes the others. If the merge logic selects the wrong survivor (e.g., the
shorter/lower-quality memory due to a frecency bug), or writes a partial TOML
(mid-crash), the user's memory data is permanently altered with no undo path.
Unit tests pass because they mock the file system; the real `memories/` directory
is never touched in tests. The bug lives in production for multiple sessions before
anyone notices memories have regressed.

**Principle violated:** "Passing tests ≠ usable system — verify entry points compile,
wiring uses real dependencies, a user can actually run it."

**What would have caught it:**
1. A dry-run mode (`--dry-run`) that prints the merge plan without executing it.
2. An integration test using a real tempdir with pre-seeded TOML files, calling
   the binary directly and asserting survivor identity.
3. A pre-merge backup step (copy to `memories/.backup/` before overwrite).

**Remediation:**
Add `--dry-run` flag to any CLI command that mutates memory files. Require a
real-filesystem integration test for every merge/delete operation before B-2 ships.

**Likelihood × Impact:** HIGH × HIGH = **#1 risk**

---

### Failure 2: Evaluation Data Format Drift Makes Historical Data Unreadable

**What went wrong:**
The `evaluations/<timestamp>.jsonl` format is defined by `persistEvaluationLog`
and read by future `runEvaluate` calls. As cycles add fields (new outcome types,
new memory metadata), the reader silently ignores unknown fields — but old records
missing required new fields cause panics or zeroed results. After 5 cycles, the
evaluations directory has 3 months of data in 3 different format versions. The
LLM correlation analysis starts producing wrong results (wrong outcome rates,
missing evidence) because old records are partially parsed. No one notices because
the summary output still looks plausible.

**Principle violated:** "Content quality > mechanical sophistication."

**What would have caught it:**
A round-trip test that writes a v1 record, reads it with v2 reader, and asserts
all fields survive correctly. A schema version field in the JSONL record.

**Remediation:**
Add `"schema_version": 1` to evaluation JSONL records now, before format diverges.
Add a round-trip test asserting backward compatibility when fields are added.

**Likelihood × Impact:** HIGH × MEDIUM = **#2 risk**

---

### Failure 3: Spec Item Count Becomes Navigation Debt

**What went wrong:**
Current spec: 429 T-items, 84 ARCH items, 133 REQ items = 714 items total. Over
5 feature cycles (each adding 20-40 items), the spec grows to 900+ items. No
triage or pruning mechanism exists. Implemented test items are never marked
`status = "done"`. Superseded ARCH items are never retired. Engineers stop reading
the spec before implementing because it's too expensive to find what's relevant.
New features get implemented without spec items. The spec becomes archaeology.

**Principle violated:** "Content quality > mechanical sophistication." A spec that
isn't read provides no value.

**What would have caught it:**
A cycle-gate check: `gh issue list --label spec-triage` ensures outstanding
triage issues are closed before a new feature cycle opens. A spec item count
trend chart in the retro (items added vs. items retired).

**Remediation:**
Add a "spec triage" step to the cycle boundary protocol: at each cycle end, review
T-items for the just-completed cycle and mark them `status = "implemented"`. Archive
superseded ARCH items. Track spec item delta in retro metrics.

**Likelihood × Impact:** MEDIUM × HIGH = **#3 risk**

---

### Failure 4: Concurrent Write Corruption Surfaces Under Real Usage

**What went wrong:**
Two stop.sh invocations fire within the same second (fast turn rate, long stop.sh
runtime). Both generate evaluation filenames from `time.Now()` at the same
resolution and collide in `evaluations/`. The second write clobbers the first.
Five cycles of dedup work (P4-full, P5-full) have made memory files more valuable
— losing one evaluation means losing a link that would have identified a
low-quality memory for repair. This was predicted in the 2026-03-17 premortem
(Failure 3) and remained unaddressed for 5 more cycles.

**Principle violated:** Unaddressed known risk compounds.

**What would have caught it:**
A goroutine-parallel test invoking the evaluator twice concurrently against a shared
tempdir, asserting both evaluation files are present after both goroutines complete.

**Remediation:**
Switch evaluation filenames from second-resolution timestamp to nanosecond or UUID
suffix. Add the concurrent-write test before B-2 ships. (Carries forward from P2 of
prior premortem.)

**Likelihood × Impact:** MEDIUM × HIGH = **#4 risk**

---

### Failure 5: Session Crash Recovery Overhead Accumulates

**What went wrong:**
The Cycle 2 session crash left 7 orphaned worktrees, 2 with usable partial work.
Recovery required a full diagnosis pass (~30 min). Over 5 more cycles, if parallel
agent sessions crash routinely (network, context exhaustion), this overhead
compounds. Worse: partial work from crashed agents is silently abandoned (not all
engineers will check worktrees). Features appear done in one session but
are half-implemented, and the next session starts on assumptions of completeness.

**Principle violated:** Invisible partial state creates false progress signals.

**What would have caught it:**
A manifest file at `.claude/worktrees/manifest.json` recording `{worktree: issue}`
mappings, written atomically before agent dispatch. Resume logic reads the manifest
and reports stranded work before starting new work.

**Remediation:**
Add a pre-dispatch step to the parallel agent workflow that writes a manifest.
Until tooling supports this, add "check for orphaned worktrees" as the first step
of every session resume (already in place now — keep it).

**Likelihood × Impact:** MEDIUM × MEDIUM = **#5 risk**

---

## Priority Matrix

| Failure | Likelihood | Impact | Priority |
|---------|-----------|--------|----------|
| 1: Cluster merge data loss | High | High | 🔴 P0 |
| 2: Evaluation format drift | High | Medium | 🔴 P1 |
| 3: Spec item count debt | Medium | High | 🟡 P2 |
| 4: Concurrent write corruption (carry-forward) | Medium | High | 🟡 P3 |
| 5: Session crash partial state | Medium | Medium | 🟡 P4 |

---

## Mitigations (Top 2)

### Mitigation A: Dry-Run Gate for Memory-Mutating Operations (Failure 1)

Before any B-2 feature (P4-full, P5-full) ships, require:
1. A `--dry-run` flag on the CLI command.
2. A real-filesystem integration test that seeds real TOML files in a tempdir,
   runs the merge, and asserts the correct survivor by content (not path).

**Action:** Create issue for `--dry-run` requirement on all memory-mutating CLI
commands before Cycle 3 work starts.

### Mitigation B: Evaluation Format Version Field Now (Failure 2)

Add `schema_version = 1` to the evaluation JSONL record struct before any more
cycles write data. Cost: 1 struct field + 1 round-trip test. The alternative —
retroactively versioning 3 months of records — is expensive and error-prone.

**Action:** File as a Cycle 2/3 boundary item alongside #322 and #323.
