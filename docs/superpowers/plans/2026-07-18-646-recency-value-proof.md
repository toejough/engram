# Recency Recall Value-Proof (#646) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. **This plan STOPS at the pilot gate (Milestone P) for a user go/no-go before any batch spend.**

**Goal:** Measure whether engram's **recency channel** — the resurfacing of an agent's *own recent transcript narration* after context loss (FEATURES.md:159's un-eval'd "recency channel delivery") — measurably helps a real opus agent reach the right answer, and at what time/$ cost, by running one context-loss continuity scenario with the recency channel **ON vs OFF**, and record the verdict in `dev/eval/LEDGER.md` + `docs/FEATURES.md:159`.

**Architecture:** Each trial is **two fresh `claude -p` sessions** sharing one per-trial-isolated vault + chunk index. **Phase 1** does real work and, forced by a validator, discovers + narrates an idiosyncratic money-unit convention. Its transcript is ingested (`engram ingest`) — legitimate self-capture, no planted note (note 288). **Phase 2** is a brand-new session (no `--resume`, zero phase-1 context) with a **natural** task prompt; the agent decides on its own whether to recall. The only systematic variable between arms is the recency channel: **ON** = default; **OFF** = the existing, documented `ENGRAM_RECENT_FILL=-1` flag (channel off). The recency toggle is invisible to the agent, so recall-firing stays balanced across arms and recency-channel delivery is the only systematic difference. Whether recall fired is recorded as a diagnostic (note 83) so the firing gap is never confused with the ranking-delivery gap.

**Tech Stack:** Python 3 eval harness under `dev/eval/traps/` (the C5 family's pattern: `claude -p` headless, isolated `CLAUDE_CONFIG_DIR`/`ENGRAM_VAULT_PATH`/`ENGRAM_CHUNKS_DIR`/`ENGRAM_TRANSCRIPT_DIR`, real `/recall` skill, real `engram` binary); a tiny fictional "orders-cli" sandbox as the build task. **No production Go change in the main plan** — the OFF arm uses an existing flag. (A Go `ENGRAM_RECENCY` toggle is a *pilot-gated contingency* — Appendix A — built only if the pilot shows the re-rank bias leaks the phase-1 chunk into the OFF arm.)

## Revision log

- **2026-07-18 — Pilot-1 (milli-dollar scenario) measured VACUOUS; scenario redesigned.** The original scenario made phase-1's idiosyncratic decision (a milli-dollar money unit) topically *identical* to the phase-2 task (compute a revenue report). Pilot n=1/arm (real opus, real engram): both arms recovered the decision via plain cosine in the matched set (`provenances:[direct]`), both correct, `surfaced_via_recency=False` even in ON — the recency channel (which dedups against the matched set) added nothing. Confirmed cause + lesson: **vault note 294** (`recency-channel-eval-needs-topically-distant-target`). A second finding: the *carrier* (phase-1's transcript) must itself be topically distant from phase-2, not just the decision — else the whole transcript cosine-matches and drags the convention in. **Disposition (Joe, 2026-07-18):** I recommended recording the finding + closing #646 (the recency channel's distinct value is the topically-distant case, already shown by C5/C4i); **Joe chose to redesign for topical distance** (self-capture-in-the-distant-case is the specific thing to prove). Redesign below (§2 still describes the old milli-dollar scenario pending re-pilot validation; the live design is the `hq`/RUNLOG two-command scenario): a multi-command `hq` CLI where **phase 1 implements an UNRELATED command (`greet`)** and discovers an idiosyncratic **house logging convention** (`RUNLOG v1 :: <CMD> :: rows=<N> :: sig=QX7Z`), and **phase 2 implements `report`** (orders revenue) and is correct only if it emits the recalled RUNLOG line. Phase-1's greet transcript won't cosine-match "implement report", so the recency channel is the only delivery path. Per-trial cost measured at **~$0.75/trial** (batch estimates use this, not the old $3–8 guess).

## Global Constraints

- **Commit-gating (note 265) — OVERRIDES the skill default:** every build step is **stage-only** (`git add`); **HOLD every `git commit` for Joe's explicit go-ahead.** The natural checkpoint is the pilot STOP gate (Milestone P) — present the staged work + pilot result, commit on go. The subagent-driven-development skill's per-task auto-commit default does not know Joe's "commit only when asked" rule, so this plan encodes the override.
- **Natural recall firing (no scripting):** phase-2's prompt is a normal task ("implement `report` per SPEC.md") with the deployed recall guidance present in the config — the agent decides whether to `/recall`. Do NOT script "recall then build". Record `recall_fired` (grep the phase-2 transcript for an `engram query` call, per `wrun.py:65-73`) as a diagnostic. Report BOTH end-to-end (all valid trials) AND ranking-conditioned (fired-only trials) results (note 83 — separate the firing gap from the delivery gap).
- **Non-tautological validity (notes 286 + 98 + 288):** the ON-arm "memory" is the agent's **own phase-1 transcript**, surfaced by real engram recency — **never a pre-seeded note or planted answer** (tautological — re-proves note 99). Do NOT `engram learn` a note in phase 1; the recency channel is chunk-based, so phase 1 only needs its transcript ingested as chunks.
- **Real engram end-to-end, zero stubs (note 197):** the retrieval under test is the real `engram query`. No mock retrieval, no injected payloads. Both arms are real-engram configs (ON = default; OFF = existing `ENGRAM_RECENT_FILL=-1`). The pilot IS the zero-stub anchor.
- **Recency-only recoverable (note 195 + the vacuous-verdict trap):** phase-2's task query must NOT semantically match the phase-1 convention — else plain cosine surfaces it even with the recency channel OFF and the eval measures nothing. Phase-2's source feed (`orders.csv`) is withheld from phase-2's workspace so the unit cannot be re-derived from the data (alternative-neutralization, note 195).
- **Idiosyncratic, un-derivable convention (notes 99/195):** the money unit (integer tenths-of-a-cent / "milli-dollars") is arbitrary; a cold agent defaults to dollars or cents. It must be *discovered by running the validator*, not stated in any spec phase-2 can read.
- **Mandatory per-trial isolation (LEDGER `harder-regime-op-cost-unmeasurable`, the contamination bug):** every phase, arm, trial sets its own `ENGRAM_VAULT_PATH` (empty), `ENGRAM_CHUNKS_DIR` (per-trial), `ENGRAM_TRANSCRIPT_DIR` (phase-1 transcript), `CLAUDE_CONFIG_DIR`. Leaving `ENGRAM_CHUNKS_DIR` unset reads the operator's real global index = answer-key contamination (note 287).
- **Raw all-runs cost reported (issue requirement):** the aggregator and report table include a raw mean cost/turns column over ALL valid trials, alongside the judged correct-only efficiency. Raw cost is reported, not judged on.
- **Efficiency only among unassisted-correct runs (note 292):** JUDGE time/$/turns only across trials that reached correctness with **no corrective prompts**. State the claim as "recency is the cheaper / faster / or only path to *unassisted* correctness."
- **Diagnostic sub-metric paired with the end-metric (note 83):** every trial records whether phase-1 narration *surfaced* in phase-2's recall payload, so a retrieval miss is never scored as a synthesis success.
- **Pilot before batch (note 196):** N=1/arm first; validate the contrast is non-vacuous, decide whether the contingency toggle is needed, AND capture the real per-trial $ before any batch. Estimate spend from real per-trial cost × fan-out (note 101).
- **Pre-register the acceptable-outcomes list (§8) BEFORE the batch.**
- **No spend cap that interrupts a run (memory-note):** estimate + confirm up front, then let it finish.
- **Model = opus** (`--model opus`, `claude-opus-4-8`).

---

## 1. Goal & Done-Condition

**Surviving question (re-scoped issue):** does engram's recency channel change whether a context-lost opus agent recovers its own prior decision and reaches the right result on its own — and at what time/$?

**Done-condition:** a pilot-validated, then adequately-run (N confirmed post-pilot) two-arm (recency channel ON vs OFF) measurement on the context-loss continuity scenario that either (a) shows the recency channel is the only path to unassisted correctness, or is cheaper/faster at equal unassisted correctness — beyond the same-contrast spread; or (b) returns the pre-registered null; recorded as a new `dev/eval/LEDGER.md` row + `docs/FEATURES.md:159` update + the `docs/ROADMAP.md` #646 row flipped to the result with #648 unblocked.

---

## 2. Design of the Scenario

### 2.1 The sandbox ("orders-cli")

Files under `dev/eval/traps/recency_value/fixtures/`:
- `SPEC.md` — task spec, says **nothing** about the money unit: `import <orders.csv>` → writes `orders.db.json` (array of `{id, customer, amt}`); `report` → prints `total revenue: $<X>` in dollars; "run `./validate_import.py orders.db.json` to check the importer."
- `orders.csv` — sample orders with a `dollars` column (10 rows, known dollar total).
- `validate_import.py` — the forcing function: asserts each `amt == round(dollars*1000)` (integer tenths-of-a-cent); on mismatch fails with a *symptom* message ("row 1: amt out of range — expected integer tenths-of-a-cent, got 19.99") that forces discovery-by-running without writing a copy-paste formula into any file phase 2 reads.
- `expected.json` — `{"dollar_total": <sum>}`, used only by the scorer (never in an agent workspace).

**Why milli-dollars is a valid closed lever (note 195):** idiosyncratic (cold defaults to cents/dollars → 10×/1000× wrong), discovered empirically, and every escape is neutralized — phase 2 gets ONLY `{SPEC.md, orders.db.json}` (opaque integer `amt`); it does NOT get `orders.csv` (dollars), `validate_import.py` (contains the formula), OR **phase-1's own importer source** (which narrates the `*1000` math — a phase-2 agent extending the same CLI would otherwise read the neighboring `import` code and recover the unit with zero recall, in both arms → vacuous). WS2 is a **whitelist** of exactly those two files (Gate B finding 1), not a copy-minus-some.

### 2.2 The two phases (per trial, per arm)

1. **Phase 1 — self-capture.** Fresh `claude -p` in workspace `WS1`, isolated engram env. Prompt: *"Implement `import` in orders-cli per SPEC.md, then make `./validate_import.py orders.db.json` pass."* The agent runs the validator, hits the milli-dollar failure, fixes it, narrates the discovery. Deliverables: importer code + `orders.db.json` + a phase-1 session transcript.
2. **Ingest.** `engram ingest --transcript <phase-1 .jsonl> --chunks-dir <per-trial ENGRAM_CHUNKS_DIR>` folds phase-1's narration into the per-trial chunk index as the newest chunks. **Optional volume padding** (see §2.5): ingest a few extra recent unrelated chunks to simulate realistic recent-session volume.
3. **Phase 2 — context-lost measurement.** A **brand-new** `claude -p` session (no `--resume`) in `WS2`, constructed as a **whitelist** = exactly `{SPEC.md (fixture), orders.db.json (phase-1 output)}` — NO phase-1 importer source, no `orders.csv`, no `validate_import.py` (Gate B finding 1; `run_trial` asserts the WS2 file set). Same isolated engram env; `ENGRAM_RECENT_FILL=-1` iff `arm=="off"`. **Natural prompt:** *"Implement `report` in orders-cli per SPEC.md."* The agent decides whether to `/recall`; if it does, `engram query` surfaces the phase-1 narration via the recency channel only when the channel is ON. Correct `report` divides `amt` by 1000.

### 2.3 Arms
- **ON** (default `ENGRAM_RECENT_FILL` unset → 25): phase-1 narration surfaces via the recency channel (newest-by-ingest chunks) even though it is a weak cosine match to "implement report".
- **OFF** (`ENGRAM_RECENT_FILL=-1` → channel off, `resolveRecentFill`: negative → 0): the recency channel is empty; phase-1 narration competes on raw cosine only (weak). **NOTE (Gate B finding 2):** the recency *re-rank bias* (`scoreChunkForPhrase`, `query_chunks.go`) is a SEPARATE mechanism NOT gated by `ENGRAM_RECENT_FILL` — it still boosts matched-set chunks (tagged `provenanceDirect`, not `recent`) in the OFF arm. The pilot (P2, using `surfaced_any`) confirms whether this leaks the weak-cosine chunk into a readable position; if it does, build Appendix A (which also disables the re-rank bias).

### 2.4 Why this satisfies the hard constraints
Non-tautological (own transcript, no seed — 288) ✓ · recency-only recoverable (topical distance + withheld CSV — 195; pilot-validated) ✓ · idiosyncratic (99/195) ✓ · real engram, zero stubs, both arms real configs (197) ✓ · recall-firing natural + recorded, not scripted (ask-faithful; note 83 separation) ✓.

### 2.5 Coverage of the issue's "settle during execution" items
- **Fade-rate default:** the binary ON/OFF proof does NOT vary the fade rate — a fade-rate sweep is a constants-tuning task, which is **#648's scope** (roadmap: "#648 tune usefulness-activation constants", blocked on #646). #646 establishes whether there is delivery value to tune at all; its `surfaced_rate` diagnostic is the raw input #648 tunes on. (Note: the issue text says "currently 3 days"; the real code default is `halfLifeDays=60` — flag this discrepancy to #648.)
- **Floor guarantees a count, not a specific item (note 34):** partially tested here via the optional volume padding (§2.2 step 2) — ingest K extra recent chunks so phase-2's recall faces competing recent volume, and the `surfaced_rate` diagnostic reports whether the *specific* milli-dollar narration still survives. Full realistic-volume tuning is #648 scope. Pilot runs with K=0 first (cleanest contrast); the batch may set K>0 to probe robustness.

---

## 3. Unit — the harness (Python, `dev/eval/traps/`)

### Task 1: the sandbox fixture + deterministic scorers

**Files:**
- Create: `dev/eval/traps/recency_value/fixtures/{SPEC.md,orders.csv,validate_import.py,expected.json}`
- Create: `dev/eval/traps/recency_value/score.py`
- Test: `dev/eval/traps/recency_value/test_score.py`

**Interfaces — `score.py` produces:**
- `import_ok(db_path, csv_path) -> bool` — every `amt == round(dollars*1000)`.
- `report_revenue_ok(stdout, expected_dollar_total) -> bool` — parses `total revenue: $X`; True iff `abs(X - expected_dollar_total) < 0.005`.
- `surfaced_any(query_payload_yaml) -> bool` — True iff a phase-1 chunk mentioning the milli-dollar unit appears ANYWHERE in the payload (recency channel OR matched set). Serves the P2 vacuous-contrast gate — catches the re-rank-bias leak the recent-channel filter would miss (Gate B finding 2).
- `surfaced_via_recency(query_payload_yaml) -> bool` — True iff such an item appears with `provenances` containing `"recent"` (the recency channel specifically). The note-83 diagnostic — separates "recency channel delivered it" from "cosine/re-rank delivered it". Mirrors `dev/eval/cumulative/recency_probe.py:97` `score_recency_hit` faithfully; **validate against a REAL payload** (`dev/eval/cumulative/testdata/recency_with_R.yaml`), not the plan's synthetic fixture (whose indentation was wrong — fix that fixture to the real shape).
- `recall_fired(transcript_path) -> bool` — True iff the phase-2 transcript contains an `engram query` invocation (per `wrun.py:65-73`).

- [ ] **Step 1: Write failing scorer tests** in `test_score.py`:

```python
def test_report_revenue_ok_accepts_milli_dollar_math():
    assert report_revenue_ok("total revenue: $24.99\n", 24.99) is True
def test_report_revenue_ok_rejects_cents_misread():
    assert report_revenue_ok("total revenue: $2499.00\n", 24.99) is False   # amt read as cents => 100x
def test_narration_surfaced_true_when_recent_chunk_mentions_unit():
    payload = "items:\n- kind: chunk\n  provenances:\n  - recent\n  content: '...milli-dollar (dollars*1000)...'\n"
    assert narration_surfaced(payload) is True
def test_narration_surfaced_false_when_absent():
    assert narration_surfaced("items: []\n") is False
def test_recall_fired_detects_engram_query(tmp_path):
    p = tmp_path / "t.jsonl"; p.write_text('{"content":"... engram query --phrase ..."}')
    assert recall_fired(str(p)) is True
```

- [ ] **Step 2: Run to verify fail** — `python3 -m pytest dev/eval/traps/recency_value/test_score.py -v`. Expected: FAIL (module/functions missing).
- [ ] **Step 3: Implement `score.py`** (four pure functions) + author the four fixture files. `validate_import.py` fails the naive attempt with a symptom message (no formula). Confirm the exact recency-channel YAML key by reading `dev/eval/cumulative/recency_probe.py` (it matches `provenances` containing `"recent"`).
- [ ] **Step 4: Run to verify pass** — same pytest. Expected: PASS.
- [ ] **Step 5: STAGE only** (`git add`), do not commit (commit-gating).

### Task 2: the two-phase per-arm trial runner

**Files:** Create `dev/eval/traps/recency_value.py`

**Interfaces:**
- Consumes: `score.py`; the C5 isolation block (`dev/eval/traps/c5.py:53-61`), the `subprocess.run(["claude","-p",...,"--output-format","json"])` shape with the `(0,15,45,120)` degraded-build backoff (`c5.py:62-68`).
- Produces: `build_trial_env(arm, trial_dir) -> dict`; `run_trial(arm, idx, model) -> dict` with keys `{arm, idx, phase1_ok, recall_fired, correct, surfaced, phase2_cost, phase2_turns, phase2_dur_ms, phase1_cost}`; `main()` with `--arm on|off --trials N --model opus --recent-volume K --out <path>`.

**`run_trial` sequence (each step isolated):**
1. Fresh `WS1`; copy fixtures (EXCLUDING `expected.json` — the scorer answer key); `build_trial_env` (empty vault, per-trial chunks dir, transcript dir); **phase-1 config has NO engram skills** (it only builds; the transcript is captured by the explicit `engram ingest` regardless), **phase-2 config has `/recall` ONLY** (no `/learn` — a phase-1 `/learn` would write a vault note that surfaces via cosine in both arms, confounding the contrast; Gate B finding 3). Set `ENGRAM_RECENT_FILL=-1` in the phase-2 env iff `arm=="off"`.
2. Phase-1 `claude -p` (importer prompt). Record `phase1_cost`; gate `phase1_ok = import_ok(WS1/orders.db.json, csv)` — a trial with `phase1_ok == False` is **excluded and counted** (not silently dropped): phase 1 must have captured the lesson for the phase-2 measurement to be meaningful.
3. `engram ingest --transcript <phase1 .jsonl> --chunks-dir <CHUNKS>` (+ K padding chunks if `--recent-volume K`).
4. `WS2` = copy of `WS1` minus `orders.csv`.
5. Phase-2 `claude -p` (natural report prompt). Capture stdout, `total_cost_usd`, `num_turns`, `duration_ms`, and the phase-2 transcript path.
6. **Surfacing capture:** immediately after phase 2, run an out-of-band `engram query --lazy-chunks --phrase "implement the revenue report" --phrase "orders-cli report total"` with the SAME `ENGRAM_CHUNKS_DIR`/`ENGRAM_RECENT_FILL` env, teeing the YAML to `<trial_dir>/recall_payload.yaml`. This does NOT feed the agent (the agent's own recall already ran), so it does not bypass the component under test — it only scores what the arm's ranking would surface.
7. Score: `recall_fired` (phase-2 transcript), `correct = report_revenue_ok(stdout, expected)`, `surfaced = narration_surfaced(open(recall_payload.yaml))`.

- [ ] **Step 1: Write a failing unit test** `test_build_trial_env` (in `test_score.py` or a new `test_runner.py`): asserts `ENGRAM_CHUNKS_DIR` set + per-trial-unique, `ENGRAM_VAULT_PATH` set, `ENGRAM_RECENT_FILL=="-1"` present iff `arm=="off"` and ABSENT iff `arm=="on"`. (The full `run_trial` is a `@pytest.mark.slow` smoke, run in the pilot, not CI.)
- [ ] **Step 2: Run to verify fail.**
- [ ] **Step 3: Implement `build_trial_env` + `run_trial` + `main`.**
- [ ] **Step 4: Run the env unit test to pass.**
- [ ] **Step 5: STAGE only** (commit-gating).

### Task 3: aggregation + metrics report

**Files:** Create `dev/eval/traps/recency_value_agg.py`; Test `dev/eval/traps/recency_value/test_agg.py`

**Interfaces — `aggregate(trials) -> dict` per arm:**
- `n_valid` (excludes `phase1_ok == False`), `recall_fired_rate`.
- `correct_rate_all` (over n_valid), `correct_rate_fired` (over fired trials only — the note-83 conditioning).
- `surfaced_any_rate` and `surfaced_via_recency_rate` (both over fired trials; None-safe when a trial's recall didn't fire) — the dual metric from Gate B finding 2.
- `raw_cost` — mean phase-2 cost/turns/dur over ALL valid trials (the issue's raw-cost requirement).
- `efficiency` — `{cost_usd, turns, dur_ms}` computed only over `correct == True` trials (note 292), or `None` if no correct trials.
- `verdict_inputs`: only-path? (ON correct & OFF ≈ 0); cheaper/faster among correct?; sized vs the within-arm spread.

- [ ] **Step 1: Write failing aggregation tests** — synthetic trial dicts; assert `efficiency` ignores incorrect trials, `raw_cost` includes all valid trials, `correct_rate_fired` conditions on `recall_fired`, `n_valid` excludes `phase1_ok=False`.
- [ ] **Step 2–4: RED → implement `aggregate` → GREEN.**
- [ ] **Step 5: STAGE only** (commit-gating).

---

## 4. Pilot Milestone (P) — STOP for user go/no-go

**Purpose (note 196):** validate the contrast is non-vacuous, decide whether Appendix A is needed, AND capture the real per-trial $ before any batch. Zero-stub real-engram anchor (note 197).

### Commands
```bash
python3 dev/eval/traps/recency_value.py --arm on  --trials 1 --model opus --recent-volume 0 --out /tmp/646-pilot-on.json
python3 dev/eval/traps/recency_value.py --arm off --trials 1 --model opus --recent-volume 0 --out /tmp/646-pilot-off.json
python3 dev/eval/traps/recency_value_agg.py /tmp/646-pilot-on.json /tmp/646-pilot-off.json
```

### Pass/fail signals (pre-registered)
- **P1 — phase-1 capture works:** both trials `phase1_ok == True` and the phase-1 `.jsonl` contains the milli-dollar narration (grep). Else tune validator/prompt, re-pilot.
- **P2 — non-vacuous contrast (critical gate):** using `surfaced_any` (the whole-payload check that sees the re-rank leak), ON `surfaced_any == True` AND OFF `surfaced_any == False`. Batch bar: **OFF `surfaced_any_rate ≤ 0.2 × ON`** (or OFF 0 while ON ≥ 1 at pilot n=1). Also record `surfaced_via_recency` both arms (the note-83 read). If OFF `surfaced_any` is True → the re-rank bias is leaking the weak-cosine chunk → **build Appendix A** (the `ENGRAM_RECENCY` full toggle, which also disables the re-rank bias) and re-pilot; do NOT batch on a vacuous contrast (note-195 guard).
- **P3 — the mechanism can bite:** OFF `correct == False` on the pilot trial is the signal the dead-end is real; ON `correct == True` is encouraging (batch decides — one trial is anecdotal).
- **P4 — clean run:** no exhausted degraded-build retries; both payloads confirm the per-trial `ENGRAM_CHUNKS_DIR` was used (not the global index); `recall_fired` recorded for both.

### Cost/time capture
Record real `phase1_cost + phase2_cost` per trial → per-trial $. Batch estimate = per-trial $ × 2 arms × N (note 101).

### STOP
Report to Joe: the two payloads, P1–P4 verdicts, whether Appendix A is needed, per-trial $, and a proposed batch **N** (+ $ estimate). **Commit the staged harness on Joe's go; do not run the batch without go.**

---

## 5. Full Run (post-approval)

```bash
python3 dev/eval/traps/recency_value.py --arm on  --trials $N --model opus --recent-volume $K --out /tmp/646-on.json
python3 dev/eval/traps/recency_value.py --arm off --trials $N --model opus --recent-volume $K --out /tmp/646-off.json
python3 dev/eval/traps/recency_value_agg.py /tmp/646-on.json /tmp/646-off.json > /tmp/646-report.txt
```

### Metrics — a labeled criteria table (units, arms side-by-side, Δ)
Columns: arm · n_valid · recall_fired_rate · correct_rate_all · correct_rate_fired · surfaced_rate · **raw mean phase-2 $ (all)** · mean $ (correct-only) · mean turns (correct-only) · mean dur ms (correct-only) · Δ vs OFF. Present BOTH the all-trials row and the fired-conditioned row. Never narrate numbers in prose; never collapse to a bare %.

### Win-bar arithmetic
- **Only-path win:** ON correct high AND OFF ≈ 0 → recency channel is the only path to unassisted correctness (note 99 capability case).
- **Cheaper/faster win:** both reach correctness but ON cheaper/faster among correct trials, beyond the within-arm spread (note 292).
- A gap below the within-arm spread is underpowered, not a tie.

---

## 6. Analysis, LEDGER & Docs (Definition-of-Done)

### 6.1 Doc-surface enumeration grep — disposition (verified 2026-07-18; re-verified after the OFF-via-existing-flag decision)

| File:line | Disposition | Reason |
|---|---|---|
| `docs/FEATURES.md:159` ("the recency channel's delivery is not separately eval'd") | **UPDATE** | The core claim #646 closes — replace with the verdict + LEDGER anchor. |
| `dev/eval/LEDGER.md` | **ADD ROW** `#646-recency-channel-value` | DoD; verdict vocab `proven\|refuted\|unmeasured\|superseded` + figure + raw-data path. No existing #646 row. |
| `docs/ROADMAP.md:83` (#646 row) | **UPDATE** | Flip to result; move out of NOW rank 1. |
| `docs/ROADMAP.md:96` (#648 row) | **UPDATE** | Unblock #648; note the fade-rate-default discrepancy (issue says 3d, code is 60d) as #648 input. |
| `docs/GLOSSARY.md:308,561` · `docs/architecture/{c1-system-context.md:110, c2-containers.md:42,160,166, c3-components.md:103,105,160,190}` · `README.md:93` | **KEEP (N/A) for the main plan** | These describe `--recent-fill`/`ENGRAM_RECENT_FILL` (Channel 2), which the main plan **uses as-is** (existing flag) and does NOT change. They become **UPDATE** targets ONLY if Appendix A (the new `ENGRAM_RECENCY` toggle) is built — then document the new toggle alongside the existing `--recent-fill` mentions at each site. (Docs-reviewer's c1:110 / c2 / c3-diagram findings folded here.) |
| `docs/ROADMAP.md:111,158,170,213`, `LEDGER.md#glance-fails-c5-delivery`, `#crowded-vault-capability-robustness` | **KEEP (N/A)** | Other recency levers (cluster centroid, C5-apply, robustness sweep), not this delivery-value claim. |

### 6.2 Tasks
- [ ] **Task C1:** Run the aggregator; write the results table + verdict.
- [ ] **Task C2:** Add the `dev/eval/LEDGER.md` row (verdict + vintage + n + raw-data path).
- [ ] **Task C3:** Update `docs/FEATURES.md:159`, `docs/ROADMAP.md` (#646 result + #648 unblock). If Appendix A was built, also update the `ENGRAM_RECENT_FILL` doc sites per 6.1.
- [ ] **Task C4:** Deliver the results table to Joe (attach the report file — memory: attach deliverables).
- [ ] **Task C5:** On Joe's go, commit the full change set; then close #646.

---

## 7. Pre-Registered Acceptable Outcomes (honest null — write BEFORE the batch)

1. **Only-path win:** ON reaches unassisted correctness where OFF cannot (OFF correct_rate ≈ 0). LEDGER: proven.
2. **Efficiency win:** both reach correctness but ON cheaper/faster among correct trials beyond the spread. LEDGER: proven (efficiency).
3. **Null — situational:** among correct runs ON is neither the only path nor cheaper/faster, OR the ON−OFF gap is below the within-arm spread. → "recency channel helps only situationally"; #648 tunes on the `surfaced_rate` diagnostic alone. LEDGER: unmeasured/refuted as op-value, surfacing figure recorded.
4. **Firing-dominated:** if `recall_fired_rate` is low in both arms and correctness tracks firing more than the arm, report that recall-firing (not ranking) is the binding gap — the note-83 conditioned view (`correct_rate_fired`) is then the primary read, and firing is flagged as the real lever (a #695/#685-adjacent finding, honestly surfaced — not pre-assumed).
5. **Vacuous (pilot must prevent):** OFF also surfaces the unit → contrast invalid → build Appendix A or redesign; not a reportable null.

## 8. Cost Estimate
Per trial = two small opus `claude -p` builds (few turns each). Rough prior ~$3–8/trial (note 95); the **pilot replaces this with the real number** (note 101). Pilot ≈ 2 trials ≈ ~$6–16. Batch N=10/arm ≈ ~$60–160 on opus (confirmed against pilot per-trial × 20 before launch). No interrupting spend cap.

## 9. Risks & Open Questions
- **R1 — vacuous contrast (highest):** the re-rank bias surfaces the weak-cosine chunk even with the channel off. Mitigation: recency-only design + the P2 pilot gate; if it fails → Appendix A.
- **R2 — phase-1 non-capture:** agent fixes the validator without narrating why. Mitigation: P1 gate; if it recurs, a phase-1 nudge ("note any non-obvious conventions you discover"), identical across arms.
- **R3 — low recall firing:** the agent skips `/recall`. This is now a MEASURED outcome (`recall_fired`), not a failure — reported (outcome 4), arm-balanced so unbiased; the fired-conditioned view isolates ranking.
- **R4 — under-powered N:** report the gap-vs-spread honestly; a small gap is underpowered, not a tie.

## Appendix A — Contingency: the `ENGRAM_RECENCY=off` toggle (build ONLY if pilot P2 fails)

Only if the pilot shows the re-rank bias leaks the phase-1 chunk into the OFF arm (P2 vacuous with `ENGRAM_RECENT_FILL=-1` alone). Then the OFF arm needs both recency paths off.

**Files:** Modify `internal/cli/query.go`; Test `internal/cli/recency_eval_test.go` (+ an `Export` wrapper in `internal/cli/export_test.go`).

**Design:** add a `Recency string` field to `QueryArgs` (targ struct-tag `flag,name=recency,env=ENGRAM_RECENCY,...`, sibling of `RecentFill` at query.go:39). `recencyEnabled(args) = !strings.EqualFold(strings.TrimSpace(args.Recency),"off")` (`strings` already imported, query.go:13). Gate both seams:
- Recent channel (query.go:594): `fill := resolveRecentFill(args.RecentFill); if !recencyEnabled(args) { fill = 0 }`.
- Re-rank bias — **gate at the call site, not by changing `queryRecencyNow`'s signature.** `runQuery` (query.go:1699-1716) already has `args` in scope: `nowL2 := queryRecencyNow(timer, deps); if !recencyEnabled(args) { nowL2 = time.Time{} }`. Zero "now" makes chunk (`query_chunks.go:217`, `if !record.IngestedAt.IsZero() && !now.IsZero()`) and note (`query.go:1443`, `if !now.IsZero()`) re-rank skip. This preserves `boundary()`'s unconditional call (coverage invariant).

- [ ] **Step 1: RED test** in `internal/cli/recency_eval_test.go` (package `cli_test` — blackbox). Add `cli.ExportRecencyEnabled(cli.QueryArgs) bool` to `internal/cli/export_test.go` (package `cli`) to reach the unexported helper. Assert `cli.ExportRecencyEnabled(cli.QueryArgs{})` true, `{Recency:"off"}`/`{Recency:"OFF"}` false; and drive the query path over `buildSyntheticPool`/`rankOf` (in `recency_eval_test.go`) to assert a recent low-cosine chunk present under on, absent under off, off ordering = pure cosine. If `--timings` is exercised, assert timing still works under `Recency:"off"` (R5 — the `boundary()` coverage branch).
- [ ] **Step 2: `targ test`** (NOT `targ test-dev` — that is scoped to `./dev/...` and never runs `internal/cli`; confirmed `dev/targs.go:86` + `CLAUDE.md:55`). Expected: FAIL.
- [ ] **Step 3: Implement** the field + helper + both gates.
- [ ] **Step 4: `targ test`** — Expected: PASS. Then `go install ./cmd/engram` and probe `ENGRAM_RECENCY=off engram query ...` shows no `recent` items (note-197 anchor).
- [ ] **Step 5: `targ check-full`** — no new lint/coverage failures.
- [ ] **Step 6: Re-pilot** with the OFF arm using `ENGRAM_RECENCY=off`; then document the toggle at the 6.1 doc sites. STAGE only.

## Self-Review
- **Spec coverage:** mechanism (§2), harness (§3), settle-items (§2.5), pilot gate (§4), batch (§5), LEDGER/docs (§6), null incl. firing-dominated (§7), cost (§8), contingency toggle (Appendix A) — every approved-design element + every Gate-A finding maps to a task.
- **Gate-A resolution:** commit-gating (Global Constraints); natural firing + diagnostic (§ constraints, §2.2, §3 Task 2/3); raw-cost column (§3 Task 3, §5); settle-items disposition (§2.5); silent-wrong-answer accepted-with-rationale (a redo validator would leak the unit — §2.1); OFF via existing flag (§2.3, doc burden dropped — §6.1); `targ test`, blackbox `Export` wrapper, call-site gating, line re-pins (Appendix A); out-of-band capture → `recall_payload.yaml` (§3 Task 2 step 6); numeric P2 threshold (§4); efficiency dict shape (§3 Task 3); helper refs → `recency_eval_test.go` (Appendix A).
- **Placeholders:** none. **Type consistency:** `run_trial` keys consumed identically by `aggregate`; `recencyEnabled`/`Recency`/`ExportRecencyEnabled` consistent in Appendix A.
- Open item deferred to pilot by design: batch **N** and whether Appendix A is needed.
