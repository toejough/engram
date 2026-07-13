# #690 Pre-Query Phase Composition — Measure-First Plan (rev 2)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Measure the internal composition of recall's pre-query phase (~15–21 s, the largest remaining segmented recall phase after the payload-shape lever class closed at #689), then — only if the measurement reveals a cuttable slice — design and gate a cut behind trap-coverage + segmented-time bars.

**Architecture:** Additive, non-invasive sub-analysis over the FROZEN `recall_time.py` segmenter. A new `compute_pre_query_split()` decomposes the existing `pre_query_s` outer span into mechanically-identifiable sub-phases from the SAME transcript records — it does not alter the pinned 4-phase model (#684, comparability-frozen). All sub-phase boundaries are **tool_use timestamps** (not tool_results — see the marker rules), so the split never depends on the ack-vs-meta record ambiguity. The baseline composition is measured by RE-ANALYZING the 8 existing #689 after-measure transcripts (already gate-PASS full-recall runs; Variant A never touched Step 0/0.5/1, so their pre-query behavior == current main) — **zero new API spend for the measurement**. A cut, if warranted, is a separate conditional unit gated exactly like #689.

**Tech Stack:** Python 3 (dev/eval tooling; `recall_time.py` is Python, stdlib `json`/`datetime` only). Tests are pytest (`pytest 9.0.2` installed; convention `test_*.py`; `dev/targs.go` has NO python target, so tests run via plain `python3 -m pytest`, not `targ`). Trap gate (`dev/eval/traps/gate.py --tier smoke`) and the `recall_cost` `$METER` for any behavior change — both already built (vault note 102); do not rebuild.

## Global Constraints

- **Never touch the win-nucleus** (vault note 100): the Step-3 conventions-as-requirements directive, Step-2.5B recency-weight, Step-2 matched-note retrieval, and the frontmatter `description` field. A pre-query cut may not weaken any of these.
- **The lever space stays OPEN pending the measurement — with one foreclosure and one high bar:**
  - **Model-split recall is FORECLOSED** (note 80 — built, rolled back at ~14% op-cost, complexity not worth it; do not re-propose as fresh).
  - **Fewer-phrases is NOT foreclosed** — it is the highest-coverage-bar candidate. Notes 100/72 establish that the 10-phrase breadth is load-bearing for retrieval coverage, so a phrase-count cut must clear a *retrieval-coverage gate* (proving retrieval does not regress), not just a stopwatch — which is exactly the condition #690's own lever space places on it. It remains a choosable disposition at the Task-2 checkpoint, gated harder than template-assist. (This reconciles note 100's DEAD-for-cost verdict with the issue: note 100's objection is a coverage objection; the coverage gate is what answers it.)
  - **Template-assisted composition and any other composition mechanism** that cuts time without cutting coverage stay open at the standard bar.
- **A working mechanism ≠ time bought** (note 257, the #689 lesson): the keep/revert bar is on measured *segmented time* AND coverage. A cut that functions but does not reduce median `pre_query_s` beyond the noise floor is a REVERT.
- **The segmenter is comparability-frozen** (#684 pinned phase model; #688 tracks its fragility): the inner-split function is PURELY ADDITIVE. It must not change the values or the validity gate of the existing 4 phases.
- **Measure-first**: no cut is designed before the composition table is in hand and Joe has disposed the checkpoint. Joe disposes the cut decision and any issue-state mutation, choosing from the FULL preserved lever space (fewer-phrases-behind-coverage-gate, template-assist, other, or no-cut).
- **Mechanical or STOP** (mirror `compute_phases`, recall_time.py:270): every sub-phase boundary is located mechanically; a missing marker returns `(None, reason)` and is reported — never estimated.

---

## File Structure

- `dev/eval/traps/recall_time.py` — ADD `compute_pre_query_split(records, start_ts, first_query_use_ts)`, `PRE_QUERY_SUBFIELDS`, and `summarize_pre_query_split(rows)`; extend the `--segment` JSON/print output with a `pre_query_split` block. No change to `compute_phases` or the 4-phase model.
- `dev/eval/traps/test_recall_time_prequery.py` — CREATE (pytest, `test_*.py` convention): golden-fixture tests. Imports via `sys.path.insert(0, os.path.dirname(__file__)); import recall_time` (the established pattern in `test_gate.py`).
- `dev/eval/traps/testdata/prequery_trial0.jsonl` — CREATE: a committed copy of the trial-0 fixture (NEW path — `dev/eval/traps/` has no existing `testdata/`; the only established fixture dir is `dev/eval/testdata/` one level up. Placing it beside its test is deliberate; acknowledged as a new convention).
- `docs/superpowers/plans/2026-07-13-690-pre-query-split.md` — this plan.
- (Conditional, Task 3 only, added as a plan amendment at the checkpoint) the disposed cut's target file.

---

## Task 1: Pre-query inner-split instrument (additive to recall_time.py)

**Files:**
- Modify: `dev/eval/traps/recall_time.py` (add functions; extend `--segment` output)
- Test: `dev/eval/traps/test_recall_time_prequery.py`
- Fixture: `dev/eval/traps/testdata/prequery_trial0.jsonl`

**Interfaces:**
- Consumes: existing `_tool_use_blocks`, `_bash_command`, `find_query_calls`, `parse_ts` helpers (recall_time.py:89–236 — verified present with these signatures). The per-trial `records` list and `start_ts` (from `find_span`). `first_query_use_ts` is `find_query_calls(records)[0]["tool_use_ts"]` — computed ONCE by the caller and passed in (do not recompute inside the function; Task 2 passes the already-derived marker).
- Produces: `compute_pre_query_split(records, start_ts, first_query_use_ts) -> (split_dict, None)` or `(None, reason)`. All boundaries are **tool_use** timestamps:
  - `ttft_invoke_s` — `start_ts` → the `Skill` tool_use invoking `recall` (`name=="Skill"`, `input.skill=="recall"`; fires unambiguously in all 8 fixtures). Initial latency + invoke decision; NOT ours to cut — reported for completeness.
  - `skill_read_step0_s` — the `Skill` tool_use → the `engram ingest` sweep tool_use. This span bundles reading the skill body AND generating the Step-0 judgement in one model turn; **skill-reading and Step-0 generation are NOT mechanically separable within it** (no marker between "done reading" and "began emitting Step-0" — it is one forward pass). This is a stated resolution limit (see Self-Review / the AC1 note), not a clean three-way split. The Step-0 text record timestamp is emitted as `step0_text_ts` metadata for transparency, but is NOT used as a phase boundary.
  - `sweep_s` — the `engram ingest` tool_use → its tool_result (the `engram ingest --auto` Bash call). The one genuinely-mechanical, cleanly-cuttable slice.
  - `compose_s` — the `engram ingest` tool_result → `first_query_use_ts` (writing the 10 phrases + assembling the query call).
  - `unattributed_s` — `pre_query_s` minus the sum of the four above (expected ≈ 0).
  - `split_gate_ok` (bool: all five ≥ 0 AND their sum == `pre_query_s` ± 1.0 s), `split_gate_detail` (str), and the raw marker timestamps used.
- STOP rules (return `(None, reason)`, never estimate):
  - No `Skill` tool_use with `input.skill=="recall"` before `first_query_use_ts` → `"no recall Skill tool_use — cannot split pre-query"`.
  - No `engram ingest` Bash tool_use→result before `first_query_use_ts` → `"no engram ingest sweep before query — cannot separate step0 from compose"`. (Sweep-absent is a STOP, not a silent fold: without the sweep anchor, `skill_read_step0_s` and `compose_s` cannot be separated. All 8 fixtures contain the sweep, so this branch is defensive; it must never estimate.)
  - No `engram query` tool_use → `"no engram query tool_use found — cannot split pre-query"`.

- [ ] **Step 1: Create the committed fixture + write the failing golden test**

Copy the trial-0 transcript to the fixture path (it currently lives in the ephemeral `$CLAUDE_JOB_DIR`):
```bash
mkdir -p dev/eval/traps/testdata
cp "$CLAUDE_JOB_DIR/tmp/689-after-transcripts/trial-0.jsonl" dev/eval/traps/testdata/prequery_trial0.jsonl
```
Golden values are MEASURED from trial-0's real record timestamps (start `15:14:13.805`; Skill:recall tool_use `15:14:17.946`; `engram ingest` tool_use `15:14:25.076`, tool_result `15:14:26.970`; first `engram query` tool_use `15:14:32.310`) → `ttft_invoke_s`=4.141→**4.1**, `skill_read_step0_s`=(25.076−17.946)=7.130→**7.1**, `sweep_s`=(26.970−25.076)=1.894→**1.9**, `compose_s`=(32.310−26.970)=5.340→**5.3**, `unattributed_s`≈**0.0**, sum=18.505→matches `pre_query_s`=round(32.310−13.805,1)=**18.5** (within ±1.0 gate). All boundaries are tool_use timestamps, so the values do not depend on the ack-vs-meta skill-result record.

```python
import os, sys
sys.path.insert(0, os.path.dirname(__file__))
import recall_time as rt

FIX = os.path.join(os.path.dirname(__file__), "testdata", "prequery_trial0.jsonl")

def _load(path):
    return [__import__("json").loads(l) for l in open(path) if l.strip()]

def test_pre_query_split_golden_trial0():
    records = _load(FIX)
    start_ts = rt.find_span(records)["start_ts"]
    first_q = rt.find_query_calls(records)[0]["tool_use_ts"]
    split, err = rt.compute_pre_query_split(records, start_ts, first_q)
    assert err is None
    assert split["split_gate_ok"] is True
    assert split["ttft_invoke_s"] == 4.1
    assert split["skill_read_step0_s"] == 7.1
    assert split["sweep_s"] == 1.9
    assert split["compose_s"] == 5.3
    assert abs(split["unattributed_s"]) <= 1.0
```

- [ ] **Step 2: Run it to verify it fails**

Run: `python3 -m pytest dev/eval/traps/test_recall_time_prequery.py -v`
Expected: FAIL — `AttributeError: module 'recall_time' has no attribute 'compute_pre_query_split'`.

- [ ] **Step 3: Implement `compute_pre_query_split` minimally**

Add the function per the Interfaces contract. Reuse `_tool_use_blocks`/`_bash_command`/`parse_ts`; do not duplicate them. Pure function of `(records, start_ts, first_query_use_ts)`. Locate the Skill marker by scanning `_tool_use_blocks` for `block["name"]=="Skill"` and `(block.get("input") or {}).get("skill")=="recall"`; the sweep by the first Bash `tool_use` whose command contains `engram ingest` before `first_query_use_ts` (+ its tool_result via the existing forward-scan helper).

- [ ] **Step 4: Run the golden test to verify it passes**

Run: `python3 -m pytest dev/eval/traps/test_recall_time_prequery.py -v`
Expected: PASS.

- [ ] **Step 5: Add the STOP-branch + regression-guard tests**

(a) A synthetic records list with the `engram ingest` call removed → assert `compute_pre_query_split` returns `(None, "no engram ingest sweep before query — cannot separate step0 from compose")` (covers the STOP branch no real fixture exercises). (b) A cheap regression guard: assert `compute_phases(records, start, end)` returns identical values whether or not `compute_pre_query_split` was called on the same records — this guards against a FUTURE in-place mutation of the shared `records`/helpers; it is not a proof of independence (the two functions are already independent pure functions), just a mutation tripwire.

- [ ] **Step 6: Wire the split into `--segment` output**

In `main` near `summarize_phases`, compute the split per gate-PASS trial (reusing each trial's already-derived `first_query_use_ts`) and add `summarize_pre_query_split(rows)` — median + range per sub-phase over `split_gate_ok` trials ONLY (discard-never-pool, mirroring `summarize_phases`). Add a `pre_query_split` block to the segmented JSON and the `PHASE SUMMARY` print. No change to existing fields.

- [ ] **Step 7: Commit**

Subject: `feat(eval): pre-query inner-split analyzer over recall_time.py (#690)` (verify ≤ 72 bytes with `printf '%s' "<subject>" | wc -c`; trim if over) + `AI-Used: [claude]` trailer.

---

## Task 2: Measure the baseline composition (re-analyze existing transcripts)

**Files:**
- Use: `dev/eval/traps/recall_time.py` (Task 1 output); the 8 transcripts at `$CLAUDE_JOB_DIR/tmp/689-after-transcripts/trial-{0..7}.jsonl`.
- Output: `$CLAUDE_JOB_DIR/tmp/690-prequery-composition.json` (measurement artifact; NOT committed — recorded in the LEDGER).

- [ ] **Step 1: Measure `pre_query_s` per trial (sanity) + the split**

For each trial-{0..7}.jsonl: load records, run `compute_phases` (record each trial's `pre_query_s`; the observed set from the #689 after-measure was 18.5/17.6/19.6/18.6/17.4/19.9/17.7/18.8 — source `$CLAUDE_JOB_DIR/tmp/689-after.json`), and run `compute_pre_query_split`. If any trial's `pre_query_s` falls outside ~16–21 s, investigate before pooling. Pool median + range per sub-phase over `split_gate_ok` trials.

- [ ] **Step 2: Verify the split gate + sums**

Assert every trial's split gate PASSED and each trial's five sub-phases sum to its `pre_query_s` ± 1.0 s. Report `n_split_gate_pass / 8`. If < 6 pass, STOP and report the marker-failure reason before drawing conclusions — do not pool a thin set silently.

- [ ] **Step 3: Produce the composition table (median + range, seconds), mapped to cuttability**

| sub-phase | median | range | cuttable by us? |
|---|---|---|---|
| ttft_invoke | — | — | No (model TTFT + invoke) |
| skill_read_step0 | — | — | Only by shortening skill body / Step-0 (nucleus-adjacent) — and the two are bundled, so which of skill-read vs Step-0-gen dominates is unresolved |
| sweep | — | — | **Yes** — skip-if-recent / async (cleanest mechanical slice) |
| compose | — | — | Fewer-phrases (HIGH coverage bar, notes 100/72), template-assist, or any other composition mechanism — all OPEN, disposed at checkpoint |

- [ ] **Step 4: CHECKPOINT — six-part briefing to Joe (inside the AskUserQuestion box)**

Present, in the refined six-part form (part 2 = relevant artifacts + their relationships; part 3 = current states verified live): the composition, which slice dominates, the resolution limit on skill_read_step0, and the disposition options — Joe chooses from the FULL preserved lever space: (a) fewer-phrases behind a retrieval-coverage gate [highest bar], (b) template-assisted composition, (c) another composition mechanism, (d) an async/skip-if-recent sweep cut, or (e) close #690 measured-no-cut. Do NOT proceed to Task 3 without his disposition.

---

## Task 3 (CONDITIONAL — only if Joe disposes a cut): the cut, gated

> This task is a FRAMEWORK. The concrete cut (target file, mechanism, exact steps, and — for a phrase-count cut — the specific retrieval-coverage gate designed at Gate A) is added as a plan amendment at the Task-2 checkpoint once Joe names the disposition, mirroring the #689 revision pattern. It is not pre-written because the measurement has not yet named a cuttable slice.

**Pre-registered keep/revert bar (RULE fixed now; threshold instantiated from the Task-2 baseline):**
- **KEEP** iff: median `pre_query_s` improvement ≥ `T` s (where `T` = the greater of 3.0 s or the Task-2 baseline `pre_query_s` inter-quartile spread — set numerically at the checkpoint) AND the trap gate smoke stays GREEN (C3/C4i/C5/C6, before+after) AND the retrieval-delivery/coverage check for the specific change holds (designed at Gate A; MANDATORY and hardest for a fewer-phrases cut).
- **REVERT** otherwise — including a mechanism that works but buys < `T` s (note 257).

- [ ] RED: a repeatable test capturing the cut's intended behavior change (writing-skills TDD if a SKILL.md edit; Go TDD if a binary change).
- [ ] GREEN minimal; REFACTOR; Gate B (design-fit).
- [ ] Trap gate smoke BEFORE (pre-cut tree) + AFTER (cut tree); `$METER` recall-only segment before/after; the coverage check for the specific change.
- [ ] Re-measure the pre-query split (Task-1 analyzer) on fresh post-cut trials; apply the pre-registered bar.
- [ ] Checkpoint: present the measured result (labeled table, units); Joe disposes KEEP/REVERT.

---

## Self-Review

**Spec coverage:** #690 AC1 (measure inner structure before designing) → Tasks 1–2, with a stated resolution limit (below). AC2 (any change gates on time AND coverage) → Task 3 bar + Global Constraints. AC3 (pre-registered bar at plan stage; Joe disposes at checkpoint) → Task 3 bar (rule fixed) + Task-2 Step-4 checkpoint.

**AC1 resolution limit (disclosed, not hidden):** the instrument resolves the pre-query span into ttft_invoke / skill_read_step0 / sweep / compose. It does NOT cleanly separate AC1's "skill-reading" from "Step-0 printing" — they occur in one model generation turn with no mechanical marker between them, so they are reported as one bundled `skill_read_step0_s` (with `step0_text_ts` emitted as transparency metadata). This is a measurement-resolution limit of the transcript, flagged for Joe to weigh at the checkpoint — not full three-way coverage.

**Lever-space fidelity (not just task order):** the plan preserves the FULL lever space of #690 into the Task-2 checkpoint — fewer-phrases (behind a coverage gate), template-assist, other composition mechanisms, sweep cut, and no-cut are all choosable dispositions. Only model-split is foreclosed (note 80). Global Constraints does not unilaterally close any lever the issue kept open.

**Anti-displacement:** the plan does the ASKED task (measure the pre-query composition) as its first deliverable; the orientation challenge (cut-space may be empty) is recorded, and Task 2's no-cut branch is a legitimate disposed outcome, not a refusal to measure.

**Provenance:** Task-1 golden values are measured from `dev/eval/traps/testdata/prequery_trial0.jsonl` (the committed trial-0 copy); the `pre_query_s` set is from `689-after.json`. Both are cited inline.

**Placeholder scan:** Task 3 is intentionally a conditional framework (the cut is unknown pre-measurement) with a concrete keep/revert RULE — not a TBD. Tasks 1–2 carry complete code, commands, and expected output.

**Type consistency:** `compute_pre_query_split` signature and the five sub-phase field names (`ttft_invoke_s`, `skill_read_step0_s`, `sweep_s`, `compose_s`, `unattributed_s`) are used identically across Tasks 1–2 and the tests.
