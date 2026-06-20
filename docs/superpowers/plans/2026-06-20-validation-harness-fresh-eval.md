# Fresh Validation of the Engram Memory System — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax. Code phases follow RED→GREEN→REFACTOR; the run ladder is a human-gated loop, not a code task.

**Goal:** Produce trustworthy, fresh evidence — on the *currently shipping* `recall-v2` skills + binary — about whether engram memory delivers on six axes (C1 faster, C2 cheaper, C3 fewer human interactions, C4 learns from standards changes, C5 remembers recent history, C6 builds compounding lessons not directly taught), across Anthropic models, n=5, with adversarial-retro + human gates at each rung.

**Architecture:** Reuse the `dev/eval/cumulative/` infrastructure (3-app accumulation chain `notes→links→feeds`, real-skill invocation, isolated index/vault, name-agnostic structural scorer) — but FIRST repair its drift against `recall-v2`, RETIRE the dead pre-recall-v2 tiered arms, and ADD fresh instrumentation for the unmeasured axes (C4/C5/C6 + C1 split + C5 recent-channel). All prior results are void; every number is produced fresh.

**Tech Stack:** Python harness (`dev/eval/cumulative/`), the real `engram` Go binary, the real `skills/{recall,learn}` skills, `claude` CLI subprocess per cell.

## Global Constraints

- **Single harness version per reported number.** Any *material* change to harness or instrumentation voids all prior rungs and forces a restart from haiku n=1. (Per the stale-results rule.)
- **Real artifacts only.** The agent INVOKES `/recall` and `/learn`; never hand-inline `engram` commands as a stand-in. Verify via session-transcript skill markers.
- **Isolated state.** Every run sets `ENGRAM_CHUNKS_DIR` + `ENGRAM_VAULT_PATH` + `ENGRAM_TRANSCRIPT_DIR` to per-cell throwaway dirs. Never read or write the prod vault.
- **Fail loud.** Missing eval inputs raise; never silently fall back to a strawman baseline.
- **Honest statistics.** Report mean ± CI over n trials. Size the noise floor from the SAME contrast (warm-vs-warm). Any effect below that floor is reported "underpowered / indistinguishable," never "tie" or "no effect."
- **Test where the no-memory baseline fails**, not just average outcomes.
- **No mid-run spend cap.** Cost is estimated and confirmed up front (Phase 2), then the run completes.
- **Python harness changes:** keep `validate.py` (zero-LLM structural checks) green at every step — it is the harness's own test suite.

---

## Phase 0 — Repair validity (no fresh run until this is green)

> **Re-audit status (2026-06-20, post-deep-clean).** The deep clean ran since this plan was written and **partially** swept the harness — it updated the query/build path to bare `engram query` (the `--synthesize-l2`/`--tier` flags are gone from the binary) but left the learn path and regime definitions stale, AND stacked the real-skill regimes on top of the legacy ones. Net effect: **`python dev/eval/cumulative/validate.py` now FAILS 14/20** — the harness is *more* broken than when this plan was written, not less. Per-task status: **0.1 still-needed (now critical)**, **0.2 still-needed (+ a new cell-count assertion to fix)**, **0.3 partial** (query swept; learn-prompt/`real.eager` residue remains), **0.4 done** (CLI surface confirmed clean). Phase 0's exit criterion (validate.py green) is unchanged but the starting hole is deeper. All `~:NNN` line refs below were re-confirmed against current `main`.

### Task 0.1: Remove the dead `learn episode` path

**Files:** Modify `dev/eval/cumulative/harness.py` (`_deterministic_learn` ~:1155); `dev/eval/cumulative/validate.py` (`check_learn_tiers`).

**Problem (CRITICAL — confirmed at `harness.py:1155`).** `_deterministic_learn` still calls `eg_learn(..., "episode", ...)` → `engram learn episode`, which no longer exists (binary ships only `learn feedback`/`learn fact`, `targets.go:157`), and still passes the removed flags `boundary-rationale`/`session`/`transcript-range`/`transcript-text` (`:1158-1160`). It returns nil → seeds an EMPTY vault → cascades into every `check_learn_tiers` failure. This single fix unblocks the bulk of the current validate.py failures.

- [ ] **Step 1 (RED):** Add a `validate.py` check `check_no_dead_subcommands()` that greps the harness for `learn episode` (and `transcript --mark`, `--from-transcript-range`, `nearest_l2`) and asserts zero occurrences. Run `python dev/eval/cumulative/validate.py` → expect FAIL listing the offenders.
- [ ] **Step 2 (GREEN):** In `_deterministic_learn`, replace the `learn episode` invocation. The deterministic (`--stub`) learn now seeds via `engram learn fact`/`learn feedback` only (one note per stated convention), matching the shipping write surface. Delete `--summary/--boundary-rationale/--session/--transcript-range` plumbing.
- [ ] **Step 3:** Run `python dev/eval/cumulative/validate.py` → expect PASS.
- [ ] **Step 4: Commit** `fix(eval): drop dead 'learn episode' from stub-learn path`.

### Task 0.2: Retire the 7 pre-recall-v2 tiered regimes

**Files:** `harness.py` (REGIMES table), `matrix.py` (`cells_for` legacy path ~:124), `aggregate.py` (legacy tables), `validate.py` (`check_cellgen`, `check_stub_pipeline` counts).

Still defined at `harness.py:47-58`; `matrix.py` still generates the 26-op legacy chain. **New issue:** the deep clean added the real-skill regimes *without* retiring the legacy ones, so the harness now generates **41 ops** while `validate.py:38-39` still asserts **26** — fix that assertion as part of this task.

- [ ] **Step 1 (RED):** Update `validate.py:check_cellgen` (incl. the `:38-39` op-count assertion) to assert the regime set is exactly the real-skill set (`real.lazy`, `real.auto`, `real.autol2` — see Task 0.3 re: `real.eager`) and the new cell counts. Run validate → FAIL (legacy still present).
- [ ] **Step 2 (GREEN):** Delete the 7 legacy regimes (`l1`, `l2.l1l2`, `l2.l2`, `l2.lazy`, `l3.l1l2l3`, `l3.l2l3`, `l3.l3`) and the legacy `cells_for` path; keep `real_cells_for`. Remove now-dead legacy aggregate tables. This also clears the orphaned `--tier` construction (`harness.py:209`) and `nearest_l2`/`nearest_l3` references in the legacy recall branches.
- [ ] **Step 3:** validate → PASS. Grep confirms no legacy regime names remain.
- [ ] **Step 4: Commit** `refactor(eval): retire pre-recall-v2 tiered regimes`.

### Task 0.3: Reconcile the real-skill learn arms with the shipping skill

**Files:** `harness.py` (`skill_learn_prompt` ~:430-432; `real.eager` regime ~:66; `LEARN_TIER_GUIDE["L3"]` ~:415; `score_learn_capture` `episode_extracted`); `dev/eval/run-chain-stage.sh` (~:35).

**Problem (PARTIAL — query path swept, learn path not).** `skill_learn_prompt:430-432` still references removed workflows (`engram transcript --mark`, `--from-transcript-range`); `LEARN_TIER_GUIDE["L3"]:415` references the removed `engram query --synthesis`; `real.eager` (`:66`) expects eager-L2 distillation the shipping `/learn` explicitly forbids; `episode_extracted` is always-False dead scoring; `run-chain-stage.sh:35` still describes the removed L3-synthesis workflow.

- [ ] **Step 1 (RED):** Add `validate.py` assertion that learn prompts contain no removed-workflow strings and that the regime set contains no eager-distillation arm. Run → FAIL.
- [ ] **Step 2 (GREEN):** Strip the dead workflow references from `skill_learn_prompt` so it only asks the agent to invoke `/learn` (the skill decides what to crystallize). Remove `real.eager` (or, if a contrast is wanted, redefine it as `real.auto` vs `real.lazy` chunk-ingest timing — NOT eager L2). Delete the `episode_extracted` check from `score_learn_capture`.
- [ ] **Step 3:** validate → PASS.
- [ ] **Step 4: Commit** `fix(eval): align real-skill learn arms with shipping /learn`.

### Task 0.4: Verify remaining CLI surface — DONE (deep clean), keep the smoke

**Status:** the remaining CLI surface is clean — `engram embed apply --all` (`harness.py:912/1110`) is valid per `internal/cli/embed.go:18-28`, and the deep clean already swept the `engram query` invocations to the flag-free form. No fix needed.

- [ ] **Step 1 (still do):** Add a `validate.py` smoke that dry-runs each distinct `engram` invocation string in the harness against `engram <sub> --help`, so a future CLI-surface drift (like the `--synthesize-l2` removal that broke recall mid-deep-clean) fails the harness's own check instead of a paid run. Commit `test(eval): CLI-surface smoke against the live binary`.

**Phase 0 exit criterion:** `python dev/eval/cumulative/validate.py` fully green; every `engram` invocation in the harness validated against the live binary; harness is real-skill-only.

---

## Phase 1 — Fresh instrumentation of all six axes

Each axis emits a metric per cell into the result JSON (schema bump). C1/C2/C3 read off the existing accumulation chain; C4/C5/C6 need new probes. **One accumulation run yields C1/C2/C3/C5/C6 signals at once; C4 needs a dedicated reversal variant.**

### Task 1.1: C1 speed (per-phase split) + C2 cost + C3 interventions + CI

**Files:** `harness.py` (timing/token capture), `aggregate.py` (CI reporting).

- [ ] Split wall-time into `recall_s` / `build_s` / `learn_s` per op (C1); keep token→$ per op (C2); keep convention-restatement count (C3).
- [ ] `aggregate.py`: report mean ± 95% CI (bootstrap over trials) for every metric; compute the warm-vs-warm noise floor and label any cold-vs-warm gap below it "underpowered."
- [ ] `validate.py` fixture check: synthetic results in → expected CI/floor out. RED→GREEN→commit.

### Task 1.2: C5 recent-history surfacing probe

**Files:** `harness.py` (recall-payload parse), new `recency_probe.py`, `aggregate.py`.

**Design:** recall-v2 emits Channel 2 (`provenance: recent`). Seed a vault + chunk index where a *recently-ingested* lesson R is the correct resolution for the next build's first failure. Measure: (a) did the recall payload surface R in the recent channel? (b) did the build APPLY R (structural/behavioral check passes on the R-governed checkpoint) without re-teaching?

- [ ] Build the probe: parse recall YAML for the recent channel; a scorer checkpoint keyed to R.
- [ ] RED (fixture: payload with/without R) → GREEN → commit.

### Task 1.3: C4 standards-change-over-time variant

**Files:** new `dev/eval/cumulative/reversal_spec.json`, `harness.py` (reversal injection), `matrix.py` (`real.reversal` arm), scorer hook.

**Design:** Teach standard X during app1 (e.g. "wrap errors with %w"). Between app1 and app2, inject a REVERSAL into the taught standard (e.g. "convention changed: return bare sentinel errors, do NOT wrap") — delivered the same way a real standard change arrives (a correction the agent should `/learn`). Measure in app3 whether the build follows X' (new) and has DROPPED X (old) — i.e. recall-v2's recency-weighting + amend-on-conflict supersede the stale lesson.

- [ ] Scorer detects which standard (X vs X') the app3 build follows (name-agnostic structural check on both forms).
- [ ] Metric: `supersession_rate` = fraction of trials where X' adopted AND X dropped.
- [ ] RED (fixture builds following X vs X') → GREEN → commit.

### Task 1.4: C6 emergent-synthesis probe (highest risk) — via clustering + "absent" crystallization

**Mechanism under test (corrected):** the shipping `/recall` runs `engram query --synthesize-l2`, which unifies matched **notes + chunks**, clusters them once (`dispatchSynthesisMode`, `query.go:86`), and emits `candidate_l2s` per cluster. The recall skill (Step 2.5) judges each cluster covered/near/**absent**; on **absent** — a cluster of related evidence that **no existing note covers centrally enough** — it **crystallizes a NEW note** synthesizing the cluster. THAT is the "compounding lesson not directly taught": a note that no prior note stated, composed from a cluster of related notes+chunks. (NOT the 3-hop wikilink subgraph — that path exists in the binary but `/recall` never calls it.)

**Files:** new `dev/eval/cumulative/synthesis_fixtures/` (curated isolated vault+chunk states), `harness.py` (synthesis probe + recall-payload cluster parse), an adversarial **semantic judge** (separate model) `synthesis_judge.py`.

**Design:** Seed an isolated vault + chunk index containing a **cluster of related-but-decentralized evidence** about a theme — several notes/chunks that each touch a facet, but where **no single note states the integrative lesson Z** (e.g. chunks/notes evidencing "validate URLs on input" and "dedup on import" as separate facts; the unstated integrative Z = "on URL import, validate-then-dedup in one pass"). Then run `/recall` with phrases that match the theme. Measure the mechanism end-to-end:
- **(a) Clustering:** did `--synthesize-l2` group the related evidence into ONE cluster (parse the payload's `clusters` + `candidate_l2s`)?
- **(b) Coverage verdict:** did the skill judge it **absent/near** (no centrally-covering note) rather than spuriously "covered"?
- **(c) Crystallization correctness:** did `/recall` write a NEW note, and does it correctly state Z (semantic judge, separate model, blind to arm, majority over ≥3 runs)?
- **(d) Downstream use:** in the subsequent build, does the agent ACT on Z where the cold baseline does not?

- [ ] Curate ≥3 fixtures (designed decentralized cluster + a task whose correct solution needs Z + cold-baseline control).
- [ ] Semantic judge scores (c) and (d) against a rubric; must distinguish real synthesis from coincidence/restatement of an existing note.
- [ ] **Honesty guard:** a NULL result (clustering doesn't group them, or the skill marks "covered" and never crystallizes, or Z is wrong) is a valid, reportable finding — design the probe to cleanly separate null vs signal, not to manufacture a positive.
- [ ] RED (fixture where Z-crystallization is obvious vs a fixture where it must NOT fire — a cluster already centrally covered) → GREEN → commit.

### Task 1.5: Schema + aggregate wiring

- [ ] Bump result schema to carry the new metrics; `aggregate.py` emits a per-axis headline table with CI; `validate.py` covers the new schema. Commit.

**Phase 1 exit:** `validate.py` green; a dry `--stub` (zero-LLM) pass produces all six axis fields with sane shapes.

---

## Phase 2 — Cost model (before any paid run)

- [ ] Compute exact projected $ for each rung from the price sheet (`harness.py` price table) × measured token profile of a single `--stub`-calibrated cell × cell count:
  - haiku n=1 pilot; haiku n=5; sonnet n=5; opus n=5; and the C4 reversal + C6 synthesis variants.
- [ ] Present the cost table to the user and get explicit go-ahead. **No paid run starts without this confirmation.**

---

## Phase 3 — The iterative run ladder (human-gated)

Loop structure (NOT a code task — a controlled experiment with gates):

1. **Run** the current rung (start: haiku **n=1** pilot) against the instrumented, isolated harness.
2. **Adversarial retro:** dispatch fresh-context reviewer subagents (per the `route` rubric) to critique the run ALONGSIDE the runner's self-report. Retro angles: (a) signal cleanliness — is any axis below the noise floor / confounded? (b) metric validity — does each probe measure what it claims (esp. C6 judge)? (c) cost-efficiency — cheaper path to the same confidence? (d) artifact fidelity — did the agent really invoke the shipping skills? Reviewers argue with the runner to resolution.
3. **Human gate:** present the retro findings to the user, who decides **iterate** (adjust + re-run this rung) or **move on**.
4. **Material-change rule:** if iterating made a *material* harness/instrumentation change, RESTART from haiku n=1 (single-version guarantee).
5. **Ladder order:** haiku n=1 → (confident) → haiku n=5 → retro/gate → sonnet n=5 → retro/gate → opus n=5 → retro/gate.

Exit: a cross-model findings report (mean ± CI per axis per model) the user accepts.

---

## Self-Review

- **Spec coverage:** all six axes have a fresh probe (C1 split, C2 cost, C3 interventions, C4 reversal variant, C5 recent-channel probe, C6 synthesis judge); the audit's must-fix list maps to Phase 0 Tasks 0.1–0.4. ✓
- **Real-artifact / isolation / fail-loud / honest-stats constraints** are encoded as Global Constraints and per-task checks. ✓
- **Risk flagged:** C6 (Task 1.4) may yield a clean null; that is acceptable and the probe is designed to show null-vs-signal honestly, not to fish for a positive. ✓
- **Cost gate** (Phase 2) precedes any paid run; ladder (Phase 3) is human-gated per rung with restart-on-material-change. ✓
