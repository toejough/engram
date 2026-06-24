# C7 lever-recheck (anti-amnesia) harness — implementation plan

**Issue:** #654 · **Design:** `docs/design/2026-06-24-recall-miss-and-cost-round3-findings.md` §4.
**Status:** plan (to be executed risk-first via TDD).

## Ask → coverage

| Ask element (from #654) | Where covered |
|---|---|
| fixture family (`vault_with_closed` + distractors + `vault_open` control) | §A Fixtures |
| meaning-not-words scorer reusing `synthesis_judge` plumbing | §B Scorer |
| diagnostic "did the note surface in recall at all?" sub-metric | §B Scorer + §C Mode |
| recheck mode runs the **real** skill, measures re-proposal of the closed lever | §C Mode |
| validate current skill reproduces the miss at **~0.0 RED** | §D Validation (risk-first) |

## The load-bearing tension (raised in orientation; must be resolved here)

#654 says "tune distractor density + the in-context excerpt until the current skill reliably scores
~0.0 RED." Vault note 70 (`red-baseline-can-falsify-the-premise`) says the opposite discipline: when the
RED baseline contradicts the design premise, **STOP and surface it — do not engineer the fixture to
reproduce the assumed failure.** Resolution adopted here:

- **Tune only toward *fidelity*** to the real miss conditions (note 67: capture the concrete triggering
  condition). The real miss had: (1) a closed-lever note in the vault, (2) the disproof present in
  context but *not* pre-summarized as a verdict, (3) a task whose natural answer *is* the closed lever,
  (4) the lever invented during synthesis, not keyed by the task's own phrasing.
- **A faithful fixture that does NOT reproduce RED is a FINDING, not a fixture to contort** — it means
  the behavioral trap needs rich session context (`behavioral-traps-need-context`), and the honest
  output is "C7 cannot be cheaply reproduced as a toy." We do not manufacture a 0.0.

## §A — Fixtures (`lever_recheck/fixture1/`)

- `vault_with_closed/` — the closed-lever note (an analogue of note 80: "cheaper-tier recall split =
  −14%, rolled back") + 6–10 distractor notes (force clustering, mirror real vault density). Each `.md`
  gets a `.vec.json` sidecar via `engram embed apply` so `engram query` ranks it.
- `vault_open/` — **control**: identical minus the closed-lever note. Proves the agent freely proposes
  the lever here → a pass means "reconciled when it had the evidence," not "never mentions the lever."
- `context.md` — an EXPERIMENT-LOG-style excerpt that **states the −14%/rolled-back numbers** but does
  **not** re-summarize the lever's closed status in a verdict line (forces reconciliation against the
  seeded note, matching the real in-context-but-not-flagged condition).
- `task.txt` — asks for the single highest-leverage cost cut for a memory-augmented build loop; the
  natural answer is the closed lever.
- `closed_levers.json` — ground truth: `canonical_action`, `closure_reason`, `measured_outcome`
  (the scorer judges *meaning* against this, never the note's literal words).

## §B — Scorer (`lever_recheck_scorer.py`)

Mirror `synthesis_judge.py`: `STUB_MODE`, `JUDGE_RUNS=3`, majority vote, **default verdict = AMNESIA**
(must be refuted to flip to RECONCILED), reuse the `_call_claude_judge` / `_parse_judge_json` pattern.

- Inputs: the recommendation text + `closed_levers.json` (NOT the note's wording) + the sub-metric flags.
- Per proposed closed lever, decide: **proposed?** (advocates the `canonical_action`, paraphrase counts)
  and **reconciled?** (acknowledges the prior attempt AND outcome before advocating). A deterministic
  guard rejects "reconciliation by vocabulary": closure markers present but the lever still advocated as
  top pick without engaging the reason ⇒ AMNESIA.
- **Diagnostic sub-metric:** `note_surfaced` — did the closed-lever note appear in the recall output at
  all? Separates retrieval-failure (note never surfaced) from synthesis-failure (surfaced, ignored).
- **Stub mode:** deterministic per vault path (`vault_with_closed` → AMNESIA-expected; `vault_open` →
  control N/A), zero cost for CI.
- Scorer **unit tests** (cheap, no LLM): feed a hand-written "amnesia" recommendation and a
  "reconciled" recommendation; assert the deterministic guard + stub classify them correctly. This is
  the scorer's own RED/GREEN.

## §C — Recheck mode (`recheck.py`, reusing `harness.py` `claude()`)

- Set `ENGRAM_VAULT_PATH` → the fixture vault, `ENGRAM_CHUNKS_DIR` → the `context.md` chunk index.
- Run the **real** skill: `claude -p "<task.txt> — invoke your /recall (or /please) skill"` (never a
  proxy — `eval-dont-bypass-component-under-test`).
- Extract from the transcript: the recommendation text, `recall_fired` (reuse `harness.recall_fired`),
  and `note_surfaced` (did the closed-lever note appear in the recall payload).
- Hand to the scorer. Inputs fail loud — a missing fixture file raises, no silent strawman
  (`eval-fail-loud`).

## §D — Validation (RISK-FIRST — do this before hardening)

1. Author `fixture1` + control.
2. **RED-reproduction probe:** run the real skill (a fresh opus agent with the fixture vault + task)
   ONCE. Observe: does it re-propose the closed lever without reconciling?
   - **RED reproduces (~0.0)** → proceed to harden scorer + mode + add fixtures.
   - **Does NOT reproduce** → STOP (note 70). Surface as the `behavioral-traps-need-context` finding;
     do not contort the fixture. Report and re-scope.
3. Stub-mode + scorer unit tests must be green regardless (deterministic).

## Scope this pass

`fixture1` + control + scorer (stub + real + unit tests) + recheck mode + the RED-reproduction result.
**Deferred (documented, not silently dropped):** the ≥4 additional distinct closed-lever fixtures that
#654 requires before GREEN is trustworthy — filed as the next increment.
