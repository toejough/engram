# C7 lever-recheck (anti-amnesia) harness — implementation plan (rev 2, Gate-A-revised)

**Issue:** #654 · **Design:** `docs/design/2026-06-24-recall-miss-and-cost-round3-findings.md` §4.
**Status:** plan (execute risk-first via TDD). Rev 2 folds in all four Gate-A reviews.

## Ask → coverage

| Ask / #654 AC element | Where covered |
|---|---|
| fixture family (`vault_with_closed` + distractors + `vault_open` control) | §A |
| meaning-not-words scorer reusing `synthesis_judge` plumbing | §B |
| diagnostic "did the note surface in recall at all?" sub-metric | §B5 + §C |
| recheck mode runs the **real** skill, measures re-proposal of the closed lever | §C |
| validate current skill reproduces the miss at **~0.0 RED** | §D |
| (#654) ≥4–5 distinct fixtures before GREEN is trustworthy | §E (deferred — *prerequisite for any GREEN*) |
| (#654) size deltas against judge-run variance, not zero | §B4 (record per-run votes) |
| (#654) adversarial-paraphrase spot-check as a hard gate | §B6 (scorer adversarial test) |

## The load-bearing tension — #654 "tune until ~0.0 RED" vs note 70

#654/§4 say the skill must score **~0.0 RED or the fixture proves nothing**, and to *tune* distractor
density + the in-context excerpt until it does. Note 70 (`red-baseline-can-falsify-the-premise`) says:
do **not** engineer a fixture to reproduce an *assumed* failure. Reconciliation (sharpened per Gate A —
the two are not the same situation, so name the boundary precisely):

- **Tuning toward FIDELITY is allowed and is exactly what #654's "tune" means.** The target is the four
  concrete triggering conditions of the real miss (note 67 — capture the triggering condition):
  1. a closed-lever **note** is in the vault;
  2. the disproof (the −14% / rolled-back numbers) is **present in context but NOT pre-summarized as a
     verdict** the agent can read off the page;
  3. the task's **natural answer is the closed lever**;
  4. the lever is **invented during synthesis**, not keyed by the task's own phrasing.
- **A FAITHFUL fixture** (all four conditions met, verified by the §D2 checklist) **that still does not
  reproduce RED is a FINDING, not a fixture to contort.** That is the note-70 case — but it applies
  *only after* the fidelity checklist passes. It means the behavioral trap needs rich session context
  (`behavioral-traps-need-context`); we do **not** add unrealistic traps to manufacture a 0.0.
- Distinguishing the two is the §D2 checklist: a sparse/unfaithful fixture scoring ≠0.0 is a fidelity
  bug (fix it); a *faithful* fixture scoring ≠0.0 is the finding (stop, report, file follow-up).

## Execution order (RISK-FIRST — this is the sequence, not §A→§E)

1. **§A** Author `fixture1` (vault_with_closed + vault_open control + closed_levers.json + context.md +
   task.txt) and generate sidecars.
2. **§D1 RED-reproduction probe** — run the real skill ONCE against `fixture1`; eyeball whether it
   re-proposes the closed lever (no scorer needed to see this).
3. **§D2 fidelity gate** on the result (checklist below). RED reproduces → continue; faithful-but-no-RED
   → STOP, file the behavioral-trap finding, report to the user.
4. **§B** Build `lever_recheck_scorer.py` (+ stub + unit tests + adversarial-paraphrase test).
5. **§C** Build `recheck.py` (real-skill run + extraction + scoring), wired to the scorer.
6. **§E** Note deferred fixtures as a prerequisite for any GREEN.

(Scorer + mode are built **after** the probe so they are not hardened against a fixture that proves
useless — the Gate-A sequencing fix.)

## §A — Fixtures (`lever_recheck/fixture1/`)

- `vault_with_closed/` — the closed-lever note (analogue of note 80: "cheaper-tier recall split −14%,
  rolled back") + 6–10 distractor notes (mirror real vault density / force clustering). Sidecars via
  `engram embed apply --all` with `ENGRAM_VAULT_PATH` set to the dir (verified mechanism).
- `vault_open/` — **control**: identical minus the closed-lever note. A pass = "reconciled when it had
  the evidence," not "never mentions the lever."
- `context.md` — EXPERIMENT-LOG-style excerpt stating the −14%/rolled-back numbers, **no verdict line**.
  **Delivered in-context by pasting it into the task prompt** (matches the real miss, where the log was
  in the agent's context). NOT via `ENGRAM_CHUNKS_DIR` (that needs a `.jsonl` chunk dir, not a `.md`
  file — Gate-A code finding).
- `task.txt` — asks for the single highest-leverage cost cut for a memory-augmented build loop; natural
  answer is the closed lever.
- `closed_levers.json` — ground truth, array of objects:
  `{ "id", "canonical_action", "closure_reason", "measured_outcome", "note_basename" }`. The scorer
  judges *meaning* against `canonical_action`/`closure_reason`, never the note's literal words.

## §B — Scorer (`lever_recheck_scorer.py`)

Mirror `synthesis_judge.py`'s structure (do **not** call `list_fixtures`/`_stub_judge` directly — they
hardcode `vault_with_cluster`/`vault_covered` + `expected_synthesis_z.txt`; write lever_recheck
equivalents). **Reuse by import:** `_call_claude_judge`, `_parse_judge_json`.

1. `STUB_MODE`, `JUDGE_RUNS=3`, majority vote, **default verdict = AMNESIA** (must be refuted to flip to
   RECONCILED).
2. Inputs: recommendation text + `closed_levers.json` + sub-metric flags. Per proposed closed lever:
   **proposed?** (advocates the `canonical_action`; paraphrase counts) and **reconciled?** (acknowledges
   the prior attempt AND its outcome before advocating).
3. Deterministic guard against "reconciliation by vocabulary": closure markers present but lever still
   advocated as top pick *without engaging the reason* ⇒ AMNESIA. Concretely (Gate-A clarity example):
   - **RECONCILED:** "We tried cheaper-tier recall (−14%, rolled back as not worth it); the real lever
     is the build loop."
   - **AMNESIA:** "Cheaper-tier recall is the highest-leverage cut." (or name-drops "−14%/rolled back"
     then still picks it as #1 without saying why that's now OK).
4. **Record per-run judge votes** (not just the majority) so a later GREEN delta is sized against judge
   variance, not zero (#654 AC).
5. **Sub-metric `note_surfaced`** — passed in by §C; separates retrieval-failure (note never surfaced)
   from synthesis-failure (surfaced, ignored). Persisted with the verdict.
6. **Adversarial-paraphrase test (hard gate, #654 AC):** a unit test feeding a paraphrase that
   name-drops the closure vocabulary but still advocates the lever without engaging the reason — assert
   the scorer returns AMNESIA. This is the scorer's own adversarial RED, distinct from the plain
   hand-written unit tests.
- **Stub mode:** deterministic per vault-path sentinel (`vault_with_closed` → AMNESIA-expected;
  `vault_open` → control), zero-cost CI.
- **Unit tests** (`test_lever_recheck_scorer.py`, pytest plain `def test_*` + assert, run via `pytest`
  — NOT `targ`, which is Go-only here): stub classification + the deterministic guard + §B6.

## §C — Recheck mode (`recheck.py`, reusing `harness.py` `claude()` + `recall_fired`)

- `ENGRAM_VAULT_PATH` → the fixture `vault_with_closed/` (or `vault_open/`). No `ENGRAM_CHUNKS_DIR`
  (context.md is in the prompt).
- Run the **real** skill: `claude -p "<task.txt> + <context.md> — invoke your /recall (or /please)
  skill"` (never a proxy — `eval-dont-bypass-component-under-test`).
- Extract from the session JSONL (under `cfg/projects/<slug>/<sid>.jsonl`): the recommendation text;
  `recall_fired` (reuse `harness.recall_fired`); and **`note_surfaced`** — a NEW extractor that finds
  the `engram query` tool-result in the transcript and checks whether the closed-lever note's basename
  appears in its matched items (no existing function does this — Gate-A code finding; model the YAML
  parse on `recency_probe.py` but check Channel-1 items, not the recent channel).
- Hand to the scorer. Inputs **fail loud** — a missing fixture file raises, no silent strawman
  (`eval-fail-loud`).

## §D — Validation (risk-first; see Execution order)

1. **RED-reproduction probe** (real skill, fresh opus, fixture1 vault + task, context.md in prompt).
   Record per-run behavior + `note_surfaced`.
2. **Fidelity gate** before any conclusion — verify all four §-tension conditions hold (closed-lever
   note in vault; disproof in-context but unsummarized; task's natural answer is the lever; lever
   emerges in synthesis). Only then:
   - **RED reproduces (~0.0)** → proceed to §B/§C.
   - **Faithful but NOT ~0.0** → STOP. File a follow-up issue ("C7 needs rich-context fixture; toy does
     not reproduce — `behavioral-traps-need-context`") and report to the user. Do not contort.
3. Stub + unit tests green regardless (deterministic, no LLM).

## §E — Scope & deferral (honest)

This pass delivers `fixture1` + control + scorer (stub + real + unit + adversarial-paraphrase tests) +
recheck mode + the **RED-reproduction result**. **Deferred — and a PREREQUISITE for any GREEN, not
"additional":** the ≥4 further distinct closed-lever fixtures #654 requires before a GREEN is
trustworthy. **A GREEN on `fixture1` alone is therefore NOT a reportable result** — only the RED result
is reportable this increment. Filed as the next increment.
