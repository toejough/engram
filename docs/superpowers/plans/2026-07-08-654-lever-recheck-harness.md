# Plan — #654: C7 lever-recheck harness — runner, RED fixtures, live baseline

## Ask (verbatim intent)

"Let's tackle 654 then. /please take it end-to-end." — #654: *eval: C7 lever-recheck (anti-amnesia)
harness — reproduce the re-proposed-closed-lever miss as RED.*

## Verified current state (orientation findings)

**The issue text is stale in two ways** (verified against main): (a) it says the harness "does not
exist yet" — the core instrument is **merged on main** (commit `0e04082e`); (b) its comment references
branch `investigate/recall-miss-cost-round3`, which no longer exists — the increment was merged.

Already on main (verified first-hand, no TODOs):
- `dev/eval/cumulative/lever_recheck/stub_engram.py` — fake `engram` on PATH: returns the buried
  closed-lever note ONLY for lever-keyed queries (`STUB_ENGRAM_LEVER_TERMS` AND-groups), logs every
  query (`STUB_ENGRAM_LOG` JSONL: phrases / lever_keyed / returned_buried); write ops are no-op successes.
- `dev/eval/cumulative/lever_recheck_scorer.py` — per-lever adversarial judge: default-AMNESIA,
  majority-of-3 (sonnet), meaning-vs-`closed_levers.json` ground truth, deterministic
  reconciliation-by-vocabulary guard, fail-LOUD loaders, `stub=True` offline mode.
- `dev/eval/cumulative/recheck.py` — pure core (`recheck_result`: outcome + mechanism kept separate:
  `note_surfaced`, `lever_query_issued`, `n_queries` never folded into pass/fail) + `live_recall` via
  `harness.claude()` with the stub shim on PATH. **Library only — no CLI.**
- `dev/eval/cumulative/lever_recheck/fixture1/` — "Orchestra pipeline" fictional domain; consult-memory
  framing; `closed_levers.json` (cheap-retrieval-model ≅ real note 80); `vault_open/` (7 distractors) +
  `vault_with_closed/` (+1 closed-lever note), each note with a `.vec.json` sidecar.
- Unit tests: `test_recheck.py`, `test_lever_recheck_scorer.py`.

**Missing (this plan's scope):** a checkpointing CLI runner; the RED-producing diagnostic-framing
fixtures; the live RED baseline (≥4–5 fixtures) with validity gates; LEDGER row + ROADMAP/triage
updates + issue closure.

## Binding vault constraints (recalled; applied as requirements)

- **Note 85** (from this harness's own first increment): the miss is NOT toy-reproducible when the
  note surfaces — the model reconciles every time; a small vault can't bury a note; the 10-angle
  recall already queries any lever *conceivable at recall time*. The miss requires the lever conceived
  **after** the single recall. Tractable RED = the **deterministic mechanism** (no lever-keyed query
  fired), instrumented by the stub; the outcome judge corroborates.
- **Notes 83/185**: ≥1 fixture must be a **pure retrieval-miss instance** — closure verdict absent
  from context, note buried; pair the end metric with the surfaced/issued diagnostic sub-metric.
- **Note 138**: neutral task framing — never name the moment ("consult your memory" spotlights;
  fixture1 keeps that framing as a *control*, not a RED cell).
- **Note 146**: bars are model-specific — the RED baseline is measured fresh on the primary model,
  never inherited from the 06-24 opus-era numbers.
- **Notes 159/168/160**: per-trial JSONL checkpoint + resume + nohup; a per-trial treatment-delivery
  gate (stub log non-empty = recall ran against the stub) before scoring; fixtures carry no real-repo
  absolute paths; post-batch `git status` contamination check; near-zero-cost trials discarded as
  degraded, never pooled.
- **Notes 97/101**: pilot 1–2 live cells first, price the batch from measured per-cell cost.
- **Note 142/86**: the real `/recall` skill is the SUT (never stubbed); the *engram binary* stub is the
  legitimate scale-faking instrument, verified against measured behavior.

## Design

### Fixture set (5 total; fictional domains, each closed lever isomorphic to a real vault rejection)

Common shape per fixture (mirrors fixture1): `task.txt` + `context.md` + `closed_levers.json` +
`vault_open/` (7 distractor notes) + `vault_with_closed/` (+1 closed-lever note), every note with a
`.vec.json` sidecar. **Sidecar generation (verified mechanism, from the first increment's retired
PLAN.md + the `dev/eval/traps/cake_fixtures.py` precedent):** `ENGRAM_VAULT_PATH=<fixture>/<vault_dir>
engram embed apply --all`, then fail-loud verify every `.md` has a sibling `.vec.json`. No real-repo paths, no real
engram/lever names — fictional isomorphs only (sandbox hygiene + no-priors).

| Fixture | Fictional domain | Closed lever (fictional) | Real isomorph |
| --- | --- | --- | --- |
| fixture1 (exists) | Orchestra doc pipeline | cheap-retrieval-model | note 80 (haiku split rollback) |
| fixture2 (new) | log-analytics ingest | shrink-archive-page-size (measured ~no cost change; closed) | notes 77/79 (size caps null) |
| fixture3 (new) | CI build farm | move-flaky-suite-to-background (rejected: relocates, doesn't reduce) | note 108 (async-learn) |
| fixture4 (new) | support-KB search | index-by-ticket-question (lost retrieval; parked) | note 153 (qanchor park) |
| fixture5 (new) | data-sync daemon | per-shard-subprocess-isolation (rejected: complexity; reopen on lighter route) | note 191 (payload-prune build) |

**Diagnostic framing (the RED cells):** `task.txt` asks a *neutral* diagnose-and-recommend question
("throughput fell / costs rose — find the highest-leverage fix"); `context.md`'s data makes the closed
lever the natural answer that **emerges mid-analysis** (not nameable from the task alone, so the
Step-0 10-angle recall won't phrase it); the closure verdict lives ONLY in the buried note (pure
retrieval instance). Expected mechanics per note 85: no lever-keyed query → note never surfaces →
recommendation re-proposes the lever → AMNESIA.

### Arms (cells) per fixture

| Arm | Vault | Framing | Expected (current skill) | Role |
| --- | --- | --- | --- | --- |
| A | with_closed | diagnostic | **AMNESIA + no lever-keyed query** | the RED cell |
| B | open | diagnostic | lever proposed legitimately; judge must NOT flag | degenerate-scorer control |
| C | with_closed | consult-memory (fixture1 style) | lever conceivable at recall → note surfaces → RECONCILED | judge positive-class + regression cell |

Primary bar = A vs B across fixtures; C runs at lower n (fixture1 already has this framing).

### Runner — new `dev/eval/cumulative/run_recheck.py` (CLI)

- argparse: `--fixtures f1,f2,.../all`, `--arm A|B|C`, `--n`, `--model` (default `opus`; add
  `fable → claude-fable-5` to `harness.MODELS` — a **disclosed one-line scope-add to a shared file**,
  see Models below), `--judge stub|live`, `--out results.jsonl`, `--resume`.
- Per trial: build isolated cfg via **`matrix.build_cfg_template(dst, warm=True)`**
  (`dev/eval/cumulative/matrix.py` ~L49 — the existing cumulative-side builder; with `warm=True` it
  copies BOTH `skills/recall` and `skills/learn` into an isolated `CLAUDE_CONFIG_DIR`; `harness.py`
  itself has no cfg builder), temp cwd, stub env via
  `recheck._stub_env`, run `recheck.live_recall`, then `recheck.recheck_result`; **append the full
  trial record to JSONL immediately** (flush), with cost from the harness output; resume skips
  completed (fixture, arm, trial) keys.
- **Per-trial validity gate before scoring** (note 168): stub log non-empty AND `total_cost_usd ≥
  $0.02` (the harness family's existing degraded-call heuristic — `dev/eval/traps/wrun.py` treats < $0.02 as a
  rate-limited/degraded call) AND non-empty agent text; failures recorded as INVALID, excluded, and
  re-run.
- **Stub-payload caveat (accepted, recorded):** the stub's payload is functionally sufficient but not
  byte-faithful engram output — `candidate_l2s` is always empty and there is no `budget` object. Inert
  for this measurement (Step 3 reads `items[]` content, which the stub populates; Step 2.5's write side
  is not what C7 measures) and already proven against live `/recall` in the merged fixture1 control
  runs. Note the caveat in `RESULTS.md`.
- Post-batch: `git status --porcelain` on the repo must be clean of eval-caused changes.

### Pre-registered bars

- **RED bar (arm A), stated as decision procedure:** for each of the 5 fixtures, run arm A until it
  has **≥3 VALID trials** (re-running INVALID ones; **retry cap: ≤6 total arm-A attempts per
  fixture** — <3 valid at the cap → NOT-RED). A fixture is **RED iff 0 of its valid arm-A trials
  score RECONCILED**. The C7 RED baseline is **established iff ≥4 of 5 fixtures are RED**. A fixture
  that cannot reach 3 valid trials counts as NOT-RED (fails toward the bar, never rescues it).
  Mechanism corroboration (lever_query_issued = false, note_surfaced = false per trial) is reported
  alongside, never folded into pass/fail.
- **Named amendment of the AC's "non-waivable" tune-until-RED gate:** #654 says "must score ~0.0 RED
  … (tune … until it does)". We deliberately amend that to **capped tuning (≤2 rounds/fixture) + an
  honest non-reproducing record + the ≥4-of-5 bar** — per note 85's established finding that
  behavioral RED is not always toy-reproducible, and per the issue's own precedent (its first
  increment re-scoped fixture1 to a control rather than forcing RED). This is a decision, not drift;
  it ships in the issue-closing comment.
- **Control bar (arm B):** false-AMNESIA = 0. Any false flag stops the batch (fix judge/ground truth first).
- **Tuning rule (≤2 rounds/fixture; each round = one n=1 re-pilot).** Approved moves only:
  (a) reword `task.txt` to remove lever-adjacent nouns (keep the neutral diagnose-and-recommend shape);
  (b) move the lever-suggesting data deeper into `context.md` / dilute with additional plausible lines;
  (c) add distractor notes up to the AC's cap of 10; (d) tighten `STUB_ENGRAM_LEVER_TERMS` AND-groups.
  Still-GREEN after round 2 → record as non-reproducing.
- **Paraphrase hard gate (protocol):** 1 randomly-selected AMNESIA transcript per RED fixture;
  for each: (1) transcript grep confirms the buried note's content never appeared in recall output;
  (2) stub log confirms no lever-keyed query; (3) re-judge a meaning-preserving, vocabulary-shifted
  paraphrase of the recommendation — the verdict must be stable. Any check failing → fix judge/ground
  truth/framing before the RED verdict is recorded.
- **Judge-variance measurement (operationalizes the AC's "size deltas against judge-run variance"):**
  re-run the full 3-judge vote 3× on ≥5 transcripts spanning fixtures and both verdict classes; report
  the per-lever verdict flip-rate. This number goes in the LEDGER row; any future GREEN claim (#655)
  must exceed it.
- **Models:** primary = **opus** (continuity with every prior C7-family number; proven plumbing).
  Secondary = **fable** pilot (n=1–2, arm A only; the model daily sessions now run) — reported
  separately, no bar attached (note 146). Disclosed scope-add beyond #654's letter: one line in
  `harness.MODELS` + the pilot spend; cut freely if not wanted.
- **Judge determinism cross-check:** stub-judge pass over all live transcripts; disagreements with the
  live judge examined before verdicts ship.

### Cost (order-of-magnitude, refined by pilot before batch)

5 fixtures × (A n=3 + B n=2) = 25, **+ C n=2 on fixture1 only** = 27 recall-only runs ≈ $11–25
(opus recall-only ~$0.4–0.9) + live judge (sonnet ×3/lever/trial) ≈ $2–6 + judge-variance re-votes
(~$1) + pilot/tuning margin → **~$20–45 total**. Pilot (fixture2 A+B, n=1) prices the batch first;
honest tally kept in results. No mid-run cap.

### TDD order

1. **RED (unit):** tests for the runner core — checkpoint append/resume-skip, validity-gate
   invalidation, arm-matrix expansion, cost tally (pure; `stub=True`; no LLM).
2. **GREEN:** implement `run_recheck.py` to those tests.
3. Fixtures 2–5 authored + vec sidecars + stub-judge smoke on canned AMNESIA/RECONCILED texts +
   absolute-path grep gate.
4. **Pilot (live):** fixture2 arms A+B, n=1, opus → treatment gate + cost + does RED fire mechanically?
   Tune if needed (≤2 rounds).
5. **Batch:** all fixtures per the bars, checkpointed, nohup; paraphrase spot-checks; fable pilot.
6. Analysis + `lever_recheck/RESULTS.md`.

## Documentation & close (steps 5–6)

- **LEDGER.md** new row (definition-of-done per its charter): C7 lever-recheck RED baseline — verdict
  per the pre-registered bars, honest bounds (n, model, judge variance).
- **ROADMAP.md**: Track A "Filed — recall-before-recommend" paragraph + triage snapshot: #654 → built
  + RED measured (or honest outcome); #655's retrieval half becomes actionable (gate exists).
- **Issue #654**: closing comment with the results table + correction of the two stale claims (core
  merged at `0e04082e`; branch gone); close if the AC's RED bar is met. GREEN validation belongs to
  #655, not this issue.
- Plan doc retired at cycle close.

## Named amendment 2 (post-pilot, 2026-07-08 late) — two-phase trial layout

Pilot 2 (fixture2 A+B, n=1, opus, $1.77, `results_pilot2.jsonl`) measured three defects in the
original inline-context design:
(a) with `context.md` in the prompt, the lever is conceivable at recall time — arm A's Step-1
phrasing issued a lever-keyed query and the buried note SURFACED (`lever_query_issued=True,
note_surfaced=True`), which cannot reproduce the retrieval miss (note 85's mechanism, confirmed
live); (b) with no delimited recommendation, `extract_recommendation` falls back to whole-text and
the deterministic guard fires on recall *narration* (false AMNESIA — the negated-advocacy risk the
runner Gate B flagged); (c) arm B's final message was recall narration with no recommendation →
vacuous advocacy=False.

**Amendment (reproduces the original incident's structure — data encountered after recall):**
- The trial cwd is a bare temp dir containing ONLY the fixture's `context.md`, copied in as
  `scratch-notes.md` (domain-plausible name). The fixture dir (vaults, `closed_levers.json`,
  ground truth) is never inside or reachable from the cwd (also closes a leak: cwd=fixture_dir let
  the agent read `closed_levers.json`).
- The prompt = RECALL_PREFIX + task text + a pointer: the team's scratch log is in
  `scratch-notes.md`, read it before deciding + a format directive: end the reply with one line
  `RECOMMENDATION: <the single recommended change>`. The format directive is content-neutral.
- Expected RED mechanics (current skill): recall fires on the diagnostic ask (no lever phrases) →
  buried note NOT returned → agent reads the scratch file → conceives the lever mid-analysis → no
  re-query exists → recommends the lever → AMNESIA with `lever_query_issued=False,
  note_surfaced=False` (the pure retrieval miss, end-to-end).
- Scoring reads the `RECOMMENDATION:` line (extraction regex unchanged); the guard/judge see the
  recommendation, not narration.

## Named amendment 3 (post-pilot-3) — two-TURN resume structure

Pilot 3 (fixture2 A+B, n=1, $1.09, `results_pilot3.jsonl`) measured that the cwd-scratch-file form
still cannot produce the retrieval miss: the agent reads `scratch-notes.md` before/while phrasing
its recall, so the lever is conceivable at recall time (`lever_query_issued=True` in BOTH arms;
arm A surfaced → RECONCILED with guard clean — the judge pipeline is now sound). Instructional
ordering cannot be trusted (note 145: mechanisms, not prose). Two further findings: (a) fixture2's
closed lever points the wrong DIRECTION (data tempts toward fewer/larger page reads; the lever says
shrink) — fixed as its tuning round 1; (b) arm A issued a second, lever-keyed query mid-work — the
shipped #655-criterion-3 reconcile-rule may already partially cover the retrieval half; whether it
does so RELIABLY is now a measured output of this harness, and "criterion 3 already covers it" is
an acceptable honest outcome (recorded as such, per the amended non-reproducing rule).

**Amendment: enforce the phase split mechanically with two `claude` calls per trial:**
- **Turn 1:** RECALL_PREFIX + task.txt ONLY — no scratch file in cwd, no pointer. The task text ends
  with a neutral note that the team's data is still being gathered and will follow. Recall fires on
  diagnostic-only phrasing; the buried note cannot be returned (fixture gate: no lever_terms group
  satisfiable from task.txt — already enforced).
- **Turn 2:** `claude --resume <sid>` (the harness's own recall/build split pattern): the scratch
  data is delivered (written to cwd as `scratch-notes.md` + pointed to), with the
  `RECOMMENDATION:` format directive. The lever is conceived here; whether ANY lever-keyed query
  fires in the whole session (turn 1 + turn 2, one stub log spans both) is the mechanism metric.
- Validity gate: turn-1 stub log non-empty (recall ran); turn-2 text non-empty; cost = sum of both
  calls, floor applied to the sum.
- RED expected signature (reported alongside, not folded into pass/fail): no lever-keyed query in
  either turn, note never surfaced, RECOMMENDATION re-proposes the lever → AMNESIA. GREEN (#655
  later, or criterion-3 firing today): a turn-2 lever-keyed re-query surfaces the note → RECONCILED.

## Gates

Gate A over this plan (ask-alignment / code-alignment / docs-alignment / clarity-standards, fresh
reviewers, argue to resolution). Gate B design-fit after each refactor unit (runner; fixtures; any
harness.py touch). Gate C over LEDGER/ROADMAP/RESULTS edits. Gate D over commits + issue prose.
Amendment 2 re-gated: targeted ask-alignment re-ACK (the AC's fixture-shape language) + Gate B on
the implementing diff. Amendment 3 re-gated the same way (ask-alignment targeted ACK + Gate B on
the diff); its Gate-B advisories (docstring "expected signature" wording, `rec_line_found` flag,
last-match extraction) fold into the implementing round.
