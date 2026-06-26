# Cost-meter + trap regression gate — design

**Date:** 2026-06-26
**Status:** approved (brainstorming) — pending spec review
**Author flow:** `/please` → brainstorming → (this doc) → writing-plans

## Motivation

Two prerequisites must exist before any engram cost/usage optimization ships. They came out of
the 2026-06-26 verified benefit ledger (vault notes `99`, `100`):

- Memory's only clean, adversarially-survived wins are **capability on idiosyncratic content**
  (C3 0/25→25/25, C4-idio 5/5, C5 0/5→5/5, C6 0/8→8/8). Cost/speed are net-negative on easy builds.
- "Cheaper without losing impact" and "lighter prompts for more usage" reduce to the **same lever**:
  shave the post-fire procedure tax without eroding the capability wins.

The asymmetry that makes this dangerous: an optimization cut feels productive immediately, but a
lost capability win is **invisible** until a trap harness catches it — sessions later. And today the
dollar axis is literally unmeasurable: recall's cost is bundled inside `build_cost`. So before
optimizing we need (1) a way to *see* recall's dollars and (2) a gate that *catches* a capability
regression. Neither exists yet.

## Component 1 — `$METER`: directly-billed `recall_cost`

**File:** `dev/eval/cumulative/harness.py` (`run_build`). **Method:** session-split (Option A).

Today `build_prompt()` fuses the recall instruction and the build instruction into one prompt run in
a single `claude` session call (`do_build`, line ~674). Consequences: `recall_s` actually times
recall + the entire first build (the note-93 mislabel), and recall's dollars are summed into
`build_cost` (line ~803) with **no `recall_cost` field**.

**Change:** split round-1 into two `claude` calls.

1. **1a — recall-only.** New `recall_prompt` = "invoke `/recall`, print the impact summary, then
   STOP — write no code." Bracket it → `recall_s` (true recall-only); `recall_cost` = that call's
   billed `total_cost_usd`. Capture its `session_id`.
2. **1b — build.** Existing build prompt *minus* the recall paragraph, run with `--resume <sid>` so
   the warm cache/context carries over. This is round-1 of the existing feedback loop.

`build_cost = sum(rounds)` then naturally **excludes** recall.

**Schema + consumers (note 66 — first-class):**
- Add `recall_cost` and `axis_c2_recall_cost` to the `out` dict next to `build_cost`.
- Bump `SCHEMA_VERSION 4 → 5`.
- Update every downstream consumer of the changed fields and **run `validate.py` to green**:
  `validate.py` (`axis_fields`), `aggregate.py` (axes/reporting), `matrix.py` (chain `$` rollup).
  A partial sweep breaks consumers worse than a no-op.

**Faithfulness caveat:** splitting the session adds one small extra cache-write on the recall call;
`--resume` keeps the session so cache stays warm. This is recorded as a known, accepted cost — far
cleaner than transcript-attribution math (the rejected Option B).

**TDD (pytest, matching `cumulative/` house style — plain `def test_*` + `assert`):** inject a fake
claude-runner returning canned `{total_cost_usd, session_id}`; assert `recall_cost` equals the
recall call's cost, `build_cost` excludes it, and the v5 schema emits the new field. The pure
split/assembly logic is unit-tested; the real `claude` shell-out stays the I/O boundary.

## Component 2 — trap regression gate

**New files in `dev/eval/traps/`:** `seed_c3.py` and `gate.py`.

### `seed_c3.py` — the missing fixture
C3's warm vault was seeded manually; there is no committed seed script. Add one that plants the 5
convention notes via `engram learn fact`, mirroring `c4_idio.seed_vaults`:
`req-with-context`→`NewRequestWithContext`, `nocolor`→`NO_COLOR` gate, `t-parallel`→`t.Parallel()`,
`nil-guard-split`→nil/len guard, `wrapped-error`→`%w` wrap.

### `gate.py` — tiered orchestrator
`--tier smoke|full`:

| Tier | C3 | C4-idio | C5 | C6 | Trials | ~Cost | ~Time |
| --- | --- | --- | --- | --- | --- | --- | --- |
| **smoke** (per-edit) | 5×1 | 1 | 1 | 2×1 | 9 | ~$3 | ~3 min |
| **full** (pre-merge) | 5×5 | 5 | 5 | 2×4 | 43 | ~$18 | ~15 min |

C3 smoke keeps all 5 conventions at 1 rep (coverage beats reps — a regression kills a *specific*
convention).

- **Runs the existing warm harnesses** (reuse, do not reimplement): `seed_c3`+`wrun.py` (C3),
  `c4_idio.py` (self-seeds), `seed_c5`+`c5.py` (C5), `c6_clean.py --arm warm` (C6 — the verified
  0/8→8/8 path, not `c6.py`).
- Reads each axis's JSON output, computes per-axis pass-rate **over valid trials only**.
- **Contamination guard:** exclude `built=False` / `JUDGE_ERROR` / exhausted-retry trials; if an
  axis's contamination rate exceeds a threshold (default 20%), that axis is **INCONCLUSIVE**
  (re-run), not RED. (Encodes the degraded-build / auth-thrash-is-transient lessons.)
- **Verdict (exact bars):** GREEN iff every axis hits its exact bar over valid trials and none is
  INCONCLUSIVE; RED on any valid miss; INCONCLUSIVE if contamination too high. The verified results
  were 100%, so any valid-trial miss is a real capability drop. Smoke tier is necessarily exact (n=1).
- Emits a verdict JSON + a labeled per-axis table (valid / contaminated / pass counts).

**Fail loud (MEMORY.md feedback):** a missing fixture or seed must raise, never silently strawman to
a pass; the gate must run the *real* trap harnesses (do not bypass the component under test).

**TDD (pytest):** the verdict function is pure — feed canned axis-JSON fixtures and assert
all-pass→GREEN, one valid miss→RED, high-contamination→INCONCLUSIVE, contaminated-but-rest-pass→
GREEN-over-valid. Orchestration shell-out is the I/O boundary. Verify the whole thing with a real
`--tier smoke` run before declaring done.

## Build order
`$METER` first (small, crisp, unblocks the dollar axis), then the gate. Each ships behind its own
TDD cycle; neither depends on the other. (Considered gate-first since it is the more critical
guardrail; user chose `$METER` first.)

## Out of scope (YAGNI)
The actual optimization levers (payload-prune-after-Step-3, Step-2 paging restructure, async learn,
etc.) are NOT in this work — these are only the two prerequisites that make those levers safe to
attempt and their effect measurable.

## Risks
- **Session-split changes the cost numbers vs prior v4 runs.** Accepted: prior results are
  architecture-bound and were already to be re-validated; v5 is a clean re-baseline.
- **Full gate is real money (~$18/run).** Mitigated by the cheap smoke tier for the per-edit cadence.
- **C6 uses an LLM (sonnet) judge** that can flake → the contamination guard routes judge errors to
  INCONCLUSIVE rather than RED.
