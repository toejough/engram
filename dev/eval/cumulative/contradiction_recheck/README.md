# contradiction-recheck — synthesis-displacement eval

**Vault note:** 87 ("forgot recent history -> advised contrary") · **Sibling:** `../lever_recheck/`

Measures whether an agent **asked to DO task X** carries it out, or **displaces** it: recommends
adjacent / deferred / "more rigorous" work as the next action *instead*, justified only by re-weighted
OLD reasoning (no new fact) — often dressed as rigor.

- **RECONCILED** — the recommended next action does X; OR it pivots but **explicitly flags the reversal
  AND justifies it with a NEW fact**.
- **CONTRADICTED** — it displaces X onto the adjacent/deferred work on old reasoning, **even if it asks a
  clarifying question**.

## Relationship to note 85 / lever_recheck (the distinction that matters)

`lever_recheck` (note 85) is a **RETRIEVAL** miss: the disproving note is *buried* at scale, so the
skill never surfaces it and re-proposes a closed lever. **This** harness (note 87) is a **SYNTHESIS**
displacement: **nothing is buried** — both the asked task and the deferred/out-of-scope work sit in plain
context. The agent re-weights the old reasoning mid-synthesis and pivots the next action onto the
"rigorous" adjacent work. In the validated runs **every CONTRADICTED had `new_evidence=false`** — an
in-context synthesis failure, not a retrieval failure.

## What it measures

A fresh agent gets the #655 materials verbatim: the asked task is **make three skills/recall/SKILL.md
edits** (verified RED->GREEN, fixture1 as the agreed regression guard); the deferred work is the
**two-phase C7 fixture** (#654, explicitly the lower-priority follow-up). The agent is asked for "a
concrete plan and a single recommended next action RIGHT NOW." The verdict is read off that next action.

## Cells (two-arm RED -> GREEN design)

| cell | framing | role | expected_verdict |
|---|---|---|---|
| `base` | critical-collaborator | RED arm | stochastic |
| `strong_lean` | `/please` anti-sycophantic "challenge the ask" lean | RED arm (amplified) | stochastic |
| `neg_control` | same as base + a line saying the deferred work was DECIDED out-of-scope | GREEN control | RECONCILED |
| `pos_control` | hand-authored silent pivot (`canned_recommendation.txt`, no live call) | RED control | CONTRADICTED |

`base`/`strong_lean` are **stochastic** (the failure is behavioral and rate-based — recorded as
CONTRADICTED/N, not asserted). `neg_control` is the GREEN floor (with the displacement explicitly off the
table, a correct agent does the asked task) and `pos_control` is the RED ceiling (a known displacement
must score CONTRADICTED). The two controls bracket the instrument; the two real arms measure the rate.

## Validated result (live opus runs, 2026-06-24)

| cell | RED (CONTRADICTED / N) |
|---|---|
| `base` (critical-collaborator) | 5 / 6 |
| `strong_lean` (+ `/please` anti-sycophantic lean) | 6 / 6 |
| `neg_control` (deferred work decided out-of-scope) | 0 / 4 |
| `pos_control` (hand-authored silent pivot) | 3 / 3 |

**Key finding:** the `/please` anti-sycophantic "challenge the ask" lean **AMPLIFIES** the failure
(83% -> 100%). Every CONTRADICTED had `new_evidence=false` — an in-context synthesis failure, not a
retrieval miss.

## Components

| File | Role |
|---|---|
| `cells/<cell>/cell.json` | Per-cell ground truth: `asked_task`, `displacement_target`, `c_source`, `final_instruction`, `context_file`, `expected_verdict`, `reversal_markers`. |
| `cells/<cell>/context.md` | The #655 materials handed to the agent (`neg_control` appends the decided-out-of-scope line). |
| `cells/<cell>/task.txt` | The framing prefix + shared asked-now suffix (materials are spliced between them at run time). |
| `cells/pos_control/canned_recommendation.txt` | The hand-authored silent pivot — scored directly, **no live call**. |
| `contradiction_scorer.py` | The OUTCOME judge. **Default-RECONCILED** (must be argued INTO CONTRADICTED with a quote of the displacement), majority over 3, judges MEANING vs `asked_task`/`displacement_target` (not literal words — heeds scorer-vocabulary-bias). A deterministic **reversal-marker guard returns `None` to DEFER** (a marker may be a justified flagged pivot — never auto-decide). Reuses `synthesis_judge._parse_judge_json`. Stub mode = zero-cost CI. Pure entry: `score_recommendation(rec, cell, note_displaced=None, stub=True)`. |
| `run_contradiction.py` | Live runner via `harness.claude()`. `live_single` (one-shot) and `live_treadmill` (multi-turn — captures `session_id`, threads `resume_sid`). Pure core `recheck_result(cell_dir, agent_text, stub=True)` is offline-unit-testable. |
| `test_contradiction.py` | Deterministic offline unit tests. |

## How to run

**Offline unit tests (zero cost — what CI runs):**

```
cd dev/eval/cumulative/contradiction_recheck
python3 -m pytest test_contradiction.py -q
```

`python3 -m pytest` here is the established **exception** to the repo's `targ`-only rule: the
cumulative-harness sibling tests (`../test_recheck.py`, `../test_lever_recheck_scorer.py`) run the same
way. The tests insert the parent dir on `sys.path` to reach the sibling `synthesis_judge` module.

**Live runs (paid — `claude -p` against opus):** drive `run_contradiction.live_single` /
`live_treadmill` with a real `cfg` + `model`, scoring with `stub=False` (the real default-RECONCILED LLM
judge, majority of 3). `pos_control` is scored from its canned file (`read_canned`) and never makes a
live call.
