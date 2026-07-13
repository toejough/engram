# #690 Pre-Query Phase Composition — Measure-First Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Measure the internal composition of recall's pre-query phase (~15–21 s, the largest remaining segmented recall phase after the payload-shape lever class closed at #689), then — only if the measurement reveals a cuttable slice — design and gate a cut behind trap-coverage + segmented-time bars.

**Architecture:** Additive, non-invasive sub-analysis over the FROZEN `recall_time.py` segmenter. A new `compute_pre_query_split()` decomposes the existing `pre_query_s` outer span into mechanically-identifiable sub-phases from the SAME transcript records — it does not alter the pinned 4-phase model (#684, comparability-frozen). The baseline composition is measured by RE-ANALYZING the 8 existing #689 after-measure transcripts (already gate-PASS full-recall runs; Variant A never touched Step 0/0.5/1, so their pre-query behavior == current main) — **zero new API spend for the measurement**. A cut, if one is warranted, is a separate conditional unit gated exactly like #689.

**Tech Stack:** Python 3 (dev/eval tooling; `recall_time.py` is Python, no external deps — stdlib `json`/`datetime` only, matching the existing file). Trap gate (`dev/eval/traps/gate.py --tier smoke`) and the `recall_cost` `$METER` (cumulative harness) for any behavior change — both already built (vault note 102); do not rebuild.

## Global Constraints

- **Never touch the win-nucleus** (vault note 100): the Step-3 conventions-as-requirements directive, Step-2.5B recency-weight, Step-2 matched-note retrieval, and the frontmatter `description` field. A pre-query cut may not weaken any of these.
- **Two obvious cuts are already REFUTED — do not re-propose as fresh** (anti-amnesia): cutting the 10 query phrases (note 100 lists it DEAD — breadth surfaces the un-guessable notes; note 72 multiphrase-subsumes-graph-expansion) and splitting recall across models (note 80 — built, rolled back at ~14% op-cost, complexity not worth it). Any phrase-count OR composition change gates on retrieval COVERAGE, not just a stopwatch.
- **A working mechanism ≠ time bought** (note 257, the #689 lesson): the keep/revert bar is on measured *segmented time* AND coverage. A cut that functions but does not reduce median `pre_query_s` beyond the noise floor is a REVERT.
- **The segmenter is comparability-frozen** (#684 pinned phase model; #688 tracks its fragility): the inner-split function is PURELY ADDITIVE. It must not change the values or the validity gate of the existing 4 phases.
- **Measure-first**: no cut is designed before the composition table is in hand and Joe has disposed the checkpoint. Joe disposes the cut decision and any issue-state mutation.
- **Instrument correctness gates on golden fixtures**, not new live runs: sub-phase values are hand-verified against real transcript timestamps before the analyzer is trusted.

---

## File Structure

- `dev/eval/traps/recall_time.py` — ADD `compute_pre_query_split(records, start_ts, first_query_use_ts)` and a `PRE_QUERY_SUBFIELDS` tuple + `summarize_pre_query_split(rows)`; extend the `--segment` JSON/print output with a `pre_query_split` block. No change to `compute_phases` or the 4-phase model.
- `dev/eval/traps/recall_time_prequery_test.py` — CREATE: golden-fixture tests asserting exact sub-phase seconds for a known transcript, the validity gate, and additive-non-interference with `compute_phases`.
- `docs/superpowers/plans/2026-07-13-690-pre-query-split.md` — this plan.
- (Conditional, Task 3 only, added as a plan amendment at the checkpoint) the cut's target file — `skills/recall/SKILL.md` and/or `internal/cli/*` depending on the disposed cut.

---

## Task 1: Pre-query inner-split instrument (additive to recall_time.py)

**Files:**
- Modify: `dev/eval/traps/recall_time.py` (add functions; extend `--segment` output)
- Test: `dev/eval/traps/recall_time_prequery_test.py`

**Interfaces:**
- Consumes: the existing `_tool_use_blocks`, `_bash_command`, `_find_tool_result`, `find_query_calls`, `parse_ts`, `record_text` helpers already in `recall_time.py`; the per-trial `records` list and `start_ts`; `find_query_calls(records)[0]["tool_use_ts"]` as `first_query_use_ts`.
- Produces: `compute_pre_query_split(records, start_ts, first_query_use_ts) -> (split_dict, None)` or `(None, reason)`. `split_dict` keys:
  - `ttft_invoke_s` — `start_ts` → the `Skill` tool_use invoking `recall` (initial latency + invoke decision; NOT ours to cut — reported for completeness).
  - `skill_read_step0_s` — recall Skill tool_result → the `engram ingest --auto` sweep tool_use (reading the skill body + generating the Step-0 judgement).
  - `sweep_s` — `engram ingest --auto` tool_use → its tool_result.
  - `compose_s` — sweep tool_result → `first_query_use_ts` (writing the 10 phrases + assembling the query call).
  - `unattributed_s` — `pre_query_s` minus the sum of the four above (any records not captured by a marker; expected ≈0).
  - `split_gate_ok` (bool), `split_gate_detail` (str), plus the raw marker timestamps used.
- Marker rules (all mechanical; STOP → `(None, reason)`, never estimate — mirrors `compute_phases`):
  - Skill-invoke marker: first assistant `tool_use` with `name == "Skill"` and `input.skill == "recall"`. If absent (a run that inlines recall without the Skill tool), return `(None, "no recall Skill tool_use — cannot split pre-query")`.
  - Sweep marker: first Bash `tool_use` whose command contains `engram ingest` occurring before `first_query_use_ts`. If absent, sweep did not run this trial → set `sweep_s = 0.0` and fold its span into `compose_s` (record `sweep_present: false`); this is valid, not a STOP.
  - All four sub-phases must be ≥ 0 and sum to `pre_query_s` ± 1.0 s → `split_gate_ok`.

- [ ] **Step 1: Write the failing golden test**

Hand-compute the expected sub-phases for `689-after-transcripts/trial-0.jsonl` from its record timestamps (start `15:14:13.805`; Skill:recall use `15:14:17.946`; skill tool_result `15:14:17.952`; `engram ingest` use `15:14:25.076`, result `15:14:26.970`; first `engram query` use `15:14:32.310`) → `ttft_invoke_s≈4.1`, `skill_read_step0_s≈7.1`, `sweep_s≈1.9`, `compose_s≈5.3`, `unattributed_s≈0`, `split_gate_ok=True`, and the four sum to `pre_query_s` (18.5) ± 1.0. Copy the fixture transcript into a stable test-fixtures path the test reads (do NOT read from `$CLAUDE_JOB_DIR`, which is ephemeral — copy trial-0.jsonl to `dev/eval/traps/testdata/prequery_trial0.jsonl`). Assert exact rounded values.

```python
def test_pre_query_split_golden_trial0():
    records = load_records("dev/eval/traps/testdata/prequery_trial0.jsonl")
    start_ts = find_span(records)["start_ts"]
    first_q = find_query_calls(records)[0]["tool_use_ts"]
    split, err = compute_pre_query_split(records, start_ts, first_q)
    assert err is None
    assert split["split_gate_ok"] is True
    assert split["sweep_s"] == 1.9
    assert abs(split["ttft_invoke_s"] - 4.1) <= 0.1
    assert abs(split["skill_read_step0_s"] - 7.1) <= 0.2
    assert abs(split["compose_s"] - 5.3) <= 0.2
    assert abs(split["unattributed_s"]) <= 1.0
```

- [ ] **Step 2: Run it to verify it fails**

Run: `python3 -m pytest dev/eval/traps/recall_time_prequery_test.py -v` (or the repo's Python test runner if `targ` wraps it — check `dev/targs.go`; if dev/eval Python has no targ target, run pytest directly and note it in the task report).
Expected: FAIL with `compute_pre_query_split` not defined.

- [ ] **Step 3: Implement `compute_pre_query_split` minimally**

Add the function per the Interfaces contract above. Reuse existing helpers; do not duplicate `parse_ts`/`_tool_use_blocks`. Keep it a pure function of `(records, start_ts, first_query_use_ts)`.

- [ ] **Step 4: Run the golden test to verify it passes**

Run: `python3 -m pytest dev/eval/traps/recall_time_prequery_test.py -v`
Expected: PASS.

- [ ] **Step 5: Add the additive-non-interference test**

Assert `compute_phases(records, start, end)` returns byte-identical values with and without the split code path exercised (the split is read-only over the same records) — i.e. the 4-phase dict is unchanged. This defends the frozen-segmenter constraint.

- [ ] **Step 6: Wire the split into `--segment` output**

In the batch path (`main`, near `summarize_phases`), compute the split per gate-PASS trial and add `summarize_pre_query_split(rows)` (median + range per sub-phase over `split_gate_ok` trials only — discard-never-pool, mirroring `summarize_phases`). Add a `pre_query_split` block to the segmented JSON and the `PHASE SUMMARY` print. No change to existing fields.

- [ ] **Step 7: Commit**

Subject: `feat(eval): pre-query inner-split analyzer over recall_time.py (#690)` (measure byte length ≤ 72 with `wc -c`; trim if over) + `AI-Used: [claude]` trailer.

---

## Task 2: Measure the baseline composition (re-analyze existing transcripts)

**Files:**
- Use: `dev/eval/traps/recall_time.py` (Task 1 output); the 8 transcripts at `$CLAUDE_JOB_DIR/tmp/689-after-transcripts/trial-{0..7}.jsonl`.
- Output: `$CLAUDE_JOB_DIR/tmp/690-prequery-composition.json` (analysis artifact; NOT committed — it's a measurement, recorded in the LEDGER instead).

- [ ] **Step 1: Run the split analyzer over all 8 #689-after transcripts**

For each trial-{0..7}.jsonl: load records, compute the 4-phase segmentation (sanity: `pre_query_s` matches the recorded 17.4–19.9 s) and the pre-query split. Pool median + range per sub-phase over `split_gate_ok` trials. Write the composition JSON.

- [ ] **Step 2: Verify the composition sums and gate**

Assert every trial's split gate PASSED and each trial's four sub-phases sum to its `pre_query_s` ± 1.0 s. Report `n_split_gate_pass / 8`. If < 6 pass, STOP and report why (marker fragility) before drawing conclusions — do not pool a thin set silently.

- [ ] **Step 3: Produce the composition table (median + range, seconds), mapped to cuttability**

| sub-phase | median | range | cuttable by us? |
|---|---|---|---|
| ttft_invoke | — | — | No (model TTFT + invoke) |
| skill_read_step0 | — | — | Only by shortening skill body / Step-0 — nucleus-adjacent, high risk |
| sweep | — | — | **Yes** — skip-if-recent / async (cleanest mechanical slice) |
| compose | — | — | Phrase-count cut REFUTED; template-assist is the only open lever |

- [ ] **Step 4: CHECKPOINT — six-part briefing to Joe (inside the AskUserQuestion box)**

Present, in the refined six-part form (part 2 = relevant artifacts + their relationships; part 3 = current states verified live): the composition, which slice dominates, which slices are cuttable vs refuted vs not-ours, and the disposition options — (a) design a specific cut [named] behind the gate, (b) close #690 measured-no-cut (recall pre-query time is dominated by irreducible reasoning). Joe disposes. Do NOT proceed to Task 3 without his disposition.

---

## Task 3 (CONDITIONAL — only if Joe disposes "design a cut"): the cut, gated

> This task is a FRAMEWORK. The concrete cut (target file, mechanism, exact steps) is added as a plan amendment at the Task-2 checkpoint once Joe names the disposition — mirroring the #689 revision pattern. It is not pre-written here because the measurement has not yet named a cuttable slice.

**Pre-registered keep/revert bar (RULE fixed now; threshold instantiated from the Task-2 baseline):**
- **KEEP** iff: median `pre_query_s` improvement ≥ `T` s (where `T` = the greater of 3.0 s or the Task-2 baseline's `pre_query_s` inter-quartile spread — set numerically at the checkpoint) AND the trap gate smoke stays GREEN (C3/C4i/C5/C6, before+after) AND the retrieval-delivery/coverage check for the specific change holds (designed at Gate A for that change).
- **REVERT** otherwise — including a mechanism that works but buys < `T` s (note 257).

- [ ] RED: a repeatable test capturing the cut's intended behavior change (writing-skills TDD if the cut is a SKILL.md edit; Go TDD if it's a binary change).
- [ ] GREEN minimal; REFACTOR; Gate B (design-fit).
- [ ] Trap gate smoke BEFORE (on the pre-cut tree) + AFTER (on the cut tree); `$METER` recall-only segment before/after.
- [ ] Re-measure the pre-query split (Task-1 analyzer) on fresh post-cut trials; apply the pre-registered bar.
- [ ] Checkpoint: present the measured result (labeled table, units); Joe disposes KEEP/REVERT.

---

## Self-Review

**Spec coverage:** #690 AC bullet 1 (measure inner structure before designing) → Tasks 1–2. AC bullet 2 (any change gates on time AND coverage) → Task 3 bar + Global Constraints. AC bullet 3 (pre-registered bar at plan stage; Joe disposes at checkpoint) → Task 3 bar (rule fixed) + Task-2 Step-4 checkpoint. Covered.

**Placeholder scan:** Task 3 is intentionally a framework (the cut is unknown pre-measurement) — this is a deliberate conditional-on-disposition structure, not a TBD; the bar RULE is concrete. Tasks 1–2 have complete code/commands.

**Type consistency:** `compute_pre_query_split` signature and key names are used identically in Task 1 tests and Task 2 analysis. Sub-phase field names (`ttft_invoke_s`, `skill_read_step0_s`, `sweep_s`, `compose_s`, `unattributed_s`) are fixed across tasks.

**Anti-displacement note:** the plan does the ASKED task (measure the pre-query composition) as its first deliverable. The orientation challenge (cut-space may be empty) is recorded, not acted on as a substitution — Task 2's no-cut branch is a legitimate disposed outcome, not a refusal to measure.
