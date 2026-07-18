# Recency Recall Value-Proof (#646) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. **This plan STOPS at the pilot gate (Milestone P) for a user go/no-go before any batch spend.**

**Goal:** Measure whether engram's recency recall — the resurfacing of an agent's *own recent transcript narration* after context loss — measurably helps a real opus agent reach the right answer (and at what time/$ cost), by running one context-loss continuity scenario with recency **ON vs OFF**, and record the verdict in `dev/eval/LEDGER.md` + `docs/FEATURES.md:159`.

**Architecture:** Each trial is **two fresh `claude -p` sessions** sharing one per-trial-isolated vault + chunk index. **Phase 1** does real work and, forced by a validator, discovers + narrates an idiosyncratic money-unit convention. Its transcript is ingested (`engram ingest`) — legitimate self-capture, no planted note (note 288). **Phase 2** is a brand-new session (no `--resume`, zero phase-1 context) asked to build a topically-different consumer that is correct *only if* it recovers the convention; its only recovery path is engram recall. The **only** variable between arms is a new `ENGRAM_RECENCY=off` production toggle that disables both recency paths — so the OFF arm is a validated real-engram config, not a hack (note 197). Recall is forced to fire in **both** arms, so recency *ranking* is isolated from the separate recall-*firing* axis (#695/#685).

**Tech Stack:** Go (the `ENGRAM_RECENCY` toggle in `internal/cli/query.go`, tested with imptest+rapid+gomega); Python 3 eval harness under `dev/eval/traps/` (the C5 family's pattern: `claude -p` headless, isolated `CLAUDE_CONFIG_DIR`/`ENGRAM_VAULT_PATH`/`ENGRAM_CHUNKS_DIR`/`ENGRAM_TRANSCRIPT_DIR`, real `/recall` skill, real `engram` binary); a tiny fictional "orders-cli" Go/script sandbox as the build task.

## Global Constraints

- **Non-tautological validity (notes 286 + 98 + 288):** the ON-arm "memory" is the agent's **own phase-1 transcript**, surfaced by real engram recency — **never a pre-seeded note or planted answer** (that is tautological — it merely re-proves note 99). Do NOT `engram learn` a note in phase 1; recency operates on the recent-activity **chunk** channel, so phase 1 only needs its transcript ingested as chunks.
- **Real engram end-to-end, zero stubs (note 197):** the retrieval under test is the real `engram query`. No mock retrieval, no injected payloads. The pilot IS the zero-stub anchor.
- **Recency-only recoverable (note 195 + the vacuous-verdict trap):** phase-2's task query must NOT semantically match the phase-1 convention note — else plain cosine surfaces it even with recency OFF and the eval measures nothing. Phase 2's source feed (the dollars CSV) is withheld from phase-2's workspace so the unit cannot be re-derived from the data (alternative-neutralization, note 195).
- **Idiosyncratic, un-derivable convention (notes 99/195):** the money unit (integer tenths-of-a-cent / "milli-dollars") is arbitrary; a cold agent defaults to dollars or cents. It must be *discovered by running the validator*, not stated in any spec phase-2 can read.
- **Recall forced to fire in BOTH arms:** both phase-2 configs carry the `/recall` skill and both prompts instruct recall-then-build. Firing is held constant; recency ranking is the only variable. #646 does NOT measure recall-firing behavior (that is #695/#685).
- **Mandatory per-trial isolation (LEDGER `harder-regime-op-cost-unmeasurable`, the contamination bug):** every phase, every arm, every trial sets its own `ENGRAM_VAULT_PATH` (empty), `ENGRAM_CHUNKS_DIR` (per-trial), `ENGRAM_TRANSCRIPT_DIR` (phase-1 transcript), `CLAUDE_CONFIG_DIR`. Leaving `ENGRAM_CHUNKS_DIR` unset reads the operator's real global index = answer-key contamination.
- **Efficiency only among unassisted-correct runs (note 292):** report raw all-runs cost but JUDGE time/$/turns only across trials that reached correctness with **no corrective prompts**. State the claim as "recency is the cheaper / faster / or only path to *unassisted* correctness."
- **Diagnostic sub-metric paired with the end-metric (note 83):** every trial records whether phase-1 narration actually *surfaced* in phase-2's recall payload, so a retrieval miss is never scored as a synthesis success (or vice-versa).
- **Pilot before batch (note 196):** N=1/arm first; validate the contrast is non-vacuous AND capture the real per-trial $ before any batch. Estimate spend from real per-trial cost × fan-out (note 101).
- **Pre-register the acceptable-outcomes list (Section 8) BEFORE the batch** so results can't be p-hacked.
- **No spend cap that interrupts a run (memory-note):** estimate + confirm up front, then let it finish.
- **Model = opus** (`--model opus`, i.e. `claude-opus-4-8`) for the value claim's real-use fidelity (notes 98/99: a win on a strong model is the strongest evidence).
- **DI everywhere in Go (ADR-0001):** the `ENGRAM_RECENCY` env read happens at the CLI edge via the existing `os.Getenv` seam / targ struct-tag wiring — no `os.*` inside the ranking logic.

---

## 1. Goal & Done-Condition

**Surviving question (from the re-scoped issue):** Given recall fires, does engram's recency ranking change whether a context-lost opus agent recovers its own prior decision and gets the task right — and does it cost less time/$ to get there?

**Done-condition:** A pilot-validated, then adequately-run (N confirmed post-pilot) two-arm (recency ON vs OFF) measurement on the context-loss continuity scenario that either (a) shows recency is the only path to unassisted correctness, or is cheaper/faster at equal unassisted correctness — beyond the same-contrast noise floor; or (b) returns the pre-registered null ("recency situational"); and **records the verdict** as a new `dev/eval/LEDGER.md` row + updates `docs/FEATURES.md:159` (currently "the recency channel's delivery is not separately eval'd") + flips the `docs/ROADMAP.md` #646 row to the result and unblocks #648.

---

## 2. Design of the Scenario

### 2.1 The sandbox ("orders-cli")

A fictional, greenfield records tool. Files live under `dev/eval/traps/recency_value/fixtures/`:

- `SPEC.md` — the task spec. Describes two commands and says **nothing** about the money unit:
  - `import <orders.csv>` → writes `orders.db.json` (an array of `{id, customer, amt}`).
  - `report` → reads `orders.db.json`, prints `total revenue: $<X>` in dollars.
  - "Run `./validate_import.py orders.db.json` to check the importer."
- `orders.csv` — sample orders with a `dollars` column (e.g. `1,acme,19.99` / `2,globex,5.00` / …), 10 rows, a known dollar total.
- `validate_import.py` — the forcing function: loads `orders.db.json` and asserts each `amt == round(dollars * 1000)` (integer tenths-of-a-cent). On mismatch it fails with a symptom-level message ("row 1: amt out of range — expected integer tenths-of-a-cent, got 19.99") that forces the agent to *discover* the milli-dollar unit by running, without handing a copy-paste formula into any file phase 2 reads.
- `expected.json` — `{"dollar_total": <sum of dollars>}`, used only by the scorer (never in a workspace an agent sees).

**Why milli-dollars is a valid closed lever (note 195):** the unit is idiosyncratic (a cold agent defaults to cents or dollars → 10×/1000× wrong), it is discovered empirically (validator fails the naive attempt), and the escape route is neutralized — phase 2 gets `orders.db.json` (opaque integer `amt`) but NOT `orders.csv`, so the unit cannot be re-derived by comparing `amt` to `dollars`.

### 2.2 The two phases (per trial, per arm)

1. **Phase 1 — self-capture.** Fresh `claude -p` in workspace `WS1` with isolated engram env. Prompt: *"Implement `import` in orders-cli per SPEC.md, then make `./validate_import.py orders.db.json` pass."* The agent runs the validator, hits the milli-dollar failure, fixes it, and narrates the discovery ("amt must be tenths-of-a-cent, i.e. dollars×1000"). Deliverables: the importer code + `orders.db.json` + a session transcript in `ENGRAM_TRANSCRIPT_DIR`.
2. **Ingest.** `engram ingest --transcript <phase-1 session .jsonl> --chunks-dir <per-trial ENGRAM_CHUNKS_DIR>` folds phase-1's narration into the per-trial chunk index as the newest chunks.
3. **Phase 2 — context-lost measurement.** A **brand-new** `claude -p` session (no `--resume`) in workspace `WS2` = a copy of `WS1` with `orders.csv` **removed** (only importer code + `orders.db.json` remain). Same isolated engram env (same chunk index, `ENGRAM_RECENCY` per arm). Prompt: *"Recall anything relevant, then implement `report` in orders-cli per SPEC.md."* The `/recall` skill fires (both arms); it runs `engram query`, which honors `ENGRAM_RECENCY`. Correct `report` divides `amt` by 1000.

### 2.3 Arms

- **ON** (`recency=on`, the default / production): phase-1 narration surfaces via the recent-activity channel (+ re-rank) even though it is a weak cosine match to "implement report".
- **OFF** (`ENGRAM_RECENCY=off`): both recency paths disabled — the recent channel is off and the re-rank bias is off, so phase-1 narration only competes on raw cosine (weak) and is not floated up.

### 2.4 Why this satisfies the hard constraints

- Non-tautological: the memory is the agent's own transcript, not a seed (note 288). ✓
- Recency-only recoverable: "implement report" ≠ "amt is milli-dollars" semantically; source CSV withheld (note 195). ✓ (validated in the pilot)
- Idiosyncratic + un-derivable (notes 99/195). ✓
- Real engram, zero stubs (note 197). ✓
- Recall-firing held constant; recency ranking isolated. ✓

---

## 3. Unit A — the `ENGRAM_RECENCY` production toggle (Go, TDD)

**Files:**
- Modify: `internal/cli/query.go` (add the `Recency` arg + gate the two recency paths on it)
- Test: `internal/cli/recency_eval_test.go` (add the toggle test alongside the existing surfacing test)

**Interfaces:**
- Consumes: `QueryArgs` (query.go:25), `resolveRecentFill(raw int)` (query.go:1664), `queryRecencyNow(timer, deps)` (query.go:1403), `QueryDeps.Now` (query.go:64).
- Produces: `recencyEnabled(args QueryArgs) bool`; when false, `buildRecentFillItems` receives 0 (channel off) AND `queryRecencyNow` returns the zero time (re-rank skipped), independent of the timer clock.

**Design:** `QueryArgs` auto-wires flags/env from `targ:` struct tags (see `RecentFill` at query.go:39). Add a sibling string field so the toggle needs no new plumbing:

```go
// Recency, when "off", disables BOTH recency paths — the recent-activity
// channel and the recency re-rank bias — yielding pure-cosine ranking with
// no recent channel. Default ("" / "on") is production behavior. Used by the
// #646 value-proof OFF arm to make a validated real-engram no-recency config.
Recency string `targ:"flag,name=recency,env=ENGRAM_RECENCY,desc=recency ranking: on (default) or off (disables re-rank bias AND the recent channel)"` //nolint:lll // single unbreakable struct-tag string
```

Add a pure helper and gate the two seams:

```go
func recencyEnabled(args QueryArgs) bool {
	return !strings.EqualFold(strings.TrimSpace(args.Recency), "off")
}
```

- Recent channel (query.go:594): when disabled, force the count to 0.
  `fill := resolveRecentFill(args.RecentFill); if !recencyEnabled(args) { fill = 0 }`
  then `buildRecentFillItems(chunkRecords, chunkUnion, fill)`.
- Re-rank bias: `queryRecencyNow` (query.go:1403) is the single source of recency's "now" for ranking. Gate it so a disabled toggle returns the zero time **without** disturbing the phase timer's own clock:
  `if !recencyEnabled(args) { return time.Time{} }` as the first line of `queryRecencyNow` (thread `args` in, or capture `enabled` at the call site and pass it). A zero "now" makes both the chunk re-rank (`query_chunks.go`) and the note re-rank (`query.go:1441-1446`) skip the multiplier (`if !now.IsZero()`), per the scout's confirmation.

- [ ] **Step 1: Write the failing test** — add to `internal/cli/recency_eval_test.go`. It drives the real query path over a synthetic pool where one *recent, low-cosine* chunk surfaces under default recency, and asserts the toggle removes it (both channels). Use the existing export/test seams (`ExportNewRecencyParams`, the synthetic pool builder) plus a `QueryArgs{Recency:"off"}` path. Assert: (a) with `Recency:"on"` the recent low-cosine chunk appears (recent channel and/or re-rank floats it); (b) with `Recency:"off"` it is absent from the payload and ordering is pure cosine; (c) `recencyEnabled(QueryArgs{})` is true, `recencyEnabled(QueryArgs{Recency:"off"})` is false, case-insensitive.

```go
func TestEngramRecencyOffDisablesBothPaths(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(recencyEnabled(QueryArgs{})).To(BeTrue())
	g.Expect(recencyEnabled(QueryArgs{Recency: "off"})).To(BeFalse())
	g.Expect(recencyEnabled(QueryArgs{Recency: "OFF"})).To(BeFalse())
	// ...drive the query path over the synthetic pool (reuse buildSyntheticPool /
	// rankOf-style helper); assert the recent low-cosine chunk is present under
	// on, absent under off, and off ordering equals pure-cosine ordering.
}
```

- [ ] **Step 2: Run test to verify it fails** — `targ test-dev` (or the recency test by name). Expected: FAIL (`recencyEnabled` undefined / `Recency` field missing).
- [ ] **Step 3: Add the `Recency` field + `recencyEnabled` helper + gate the two seams** (query.go changes above). Add the `strings` import if absent.
- [ ] **Step 4: Run the test to verify it passes** — `targ test-dev`. Expected: PASS.
- [ ] **Step 5: Real-engram anchor probe (note 197)** — build the binary (`go install ./cmd/engram`), in an isolated `ENGRAM_CHUNKS_DIR` with a couple of recent chunks, confirm `engram query --phrase ... ` shows a `recent`-tagged item and `ENGRAM_RECENCY=off engram query --phrase ...` shows none. Record the two payloads' `recent`-item counts in the commit body.
- [ ] **Step 6: Run `targ check-full`** — Expected: no new lint/coverage failures (line length, DI, nil-guards per `.claude/rules/go.md`).
- [ ] **Step 7: Commit** — `feat(query): ENGRAM_RECENCY=off toggle disabling both recency paths`.

---

## 4. Unit B — the harness (Python, `dev/eval/traps/`)

### Task B1: the sandbox fixture + deterministic scorers

**Files:**
- Create: `dev/eval/traps/recency_value/fixtures/SPEC.md`, `orders.csv`, `validate_import.py`, `expected.json`
- Create: `dev/eval/traps/recency_value/score.py` (pure scorer functions)
- Test: `dev/eval/traps/recency_value/test_score.py`

**Interfaces — `score.py` produces:**
- `import_ok(db_path) -> bool` — every `amt == round(dollars*1000)` vs `orders.csv` (phase-1 gate; scorer-side, uses the fixture CSV not a workspace copy).
- `report_revenue_ok(stdout, expected_dollar_total) -> bool` — parses `total revenue: $X` from phase-2 stdout; True iff `abs(X - expected_dollar_total) < 0.005`.
- `narration_surfaced(query_payload_yaml) -> bool` — parses a captured `engram query` payload (recall's call) and returns True iff a phase-1 chunk mentioning the milli-dollar unit appears (in the `recent` channel OR the matched set). Mirrors `cumulative/recency_probe.py`'s `score_recency_hit` shape.

- [ ] **Step 1: Write failing scorer tests** in `test_score.py`:

```python
def test_report_revenue_ok_accepts_milli_dollar_math():
    assert report_revenue_ok("total revenue: $24.99\n", 24.99) is True
def test_report_revenue_ok_rejects_cents_misread():
    # amt read as cents => 100x too high
    assert report_revenue_ok("total revenue: $2499.00\n", 24.99) is False
def test_narration_surfaced_true_when_recent_chunk_mentions_unit():
    payload = "items:\n  - kind: chunk\n    provenances: [recent]\n    ...milli-dollar..."
    assert narration_surfaced(payload) is True
def test_narration_surfaced_false_when_absent():
    assert narration_surfaced("items: []\n") is False
```

- [ ] **Step 2: Run to verify fail** — `python3 -m pytest dev/eval/traps/recency_value/test_score.py -v`. Expected: FAIL (module/functions missing).
- [ ] **Step 3: Implement `score.py`** (the three pure functions) + author the four fixture files. `validate_import.py` fails the naive attempt with a symptom message (no formula).
- [ ] **Step 4: Run to verify pass** — `python3 -m pytest dev/eval/traps/recency_value/test_score.py -v`. Expected: PASS.
- [ ] **Step 5: Commit** — `test(eval): #646 orders-cli fixture + deterministic scorers`.

### Task B2: the two-phase per-arm trial runner

**Files:**
- Create: `dev/eval/traps/recency_value.py` (driver)

**Interfaces:**
- Consumes: `score.py`; the C5 isolation pattern (`c5.py:53-61`, `wrun.py:44-49`) for `CLAUDE_CONFIG_DIR`/`ENGRAM_VAULT_PATH`/`ENGRAM_CHUNKS_DIR`/`ENGRAM_TRANSCRIPT_DIR`; the `subprocess.run(["claude","-p",...,"--output-format","json"])` shape with the `(0,15,45,120)` degraded-build backoff (`c5.py:62-68`).
- Produces: `run_trial(arm, idx, model) -> dict` with `{arm, idx, correct, surfaced, phase2_cost, phase2_turns, phase2_dur_ms, phase1_cost, phase1_ok}`; a `main()` with `--arm on|off --trials N --model opus --out <path>`.

**`run_trial` sequence (each step isolated, per Global Constraints):**
1. Fresh `WS1`; copy fixtures in; set isolated engram env (empty vault, per-trial chunks dir, transcript dir, warm cfg with `/recall` + `/learn`).
2. Phase-1 `claude -p` (importer prompt). Record `phase1_cost`; gate `phase1_ok = import_ok(WS1/orders.db.json)` — a trial with `phase1_ok == False` is **excluded** (phase 1 must have captured the lesson for the phase-2 measurement to be meaningful; log the exclusion, do not silently drop — note the count).
3. `engram ingest --transcript <phase1 .jsonl> --chunks-dir <CHUNKS>`.
4. `WS2` = copy of `WS1` minus `orders.csv`.
5. Phase-2 `claude -p` (recall-then-report prompt) with `ENGRAM_RECENCY=off` iff `arm=="off"`. Capture the recall payload (see below), stdout, `total_cost_usd`, `num_turns`, `duration_ms`.
6. Score: `correct = report_revenue_ok(report_stdout, expected)`, `surfaced = narration_surfaced(payload)`.

**Capturing the recall payload (for the surfacing diagnostic):** set `ENGRAM_DEBUG_LOG` (query.go:main.go:19 reads it) OR wrap the phase-2 config's recall to tee the `engram query` YAML to a per-trial file; simplest robust path — run an out-of-band `engram query` with phase-2's plan phrases and the same `ENGRAM_RECENCY`/chunks env immediately after phase 2, purely to score surfacing (does not feed the agent, so it does not bypass the component under test — the agent's own recall already ran). Document which method is used.

- [ ] **Step 1: Write a failing smoke test** `test_run_trial_smoke` (marked `@pytest.mark.slow`, skipped in CI) that asserts `run_trial` returns a dict with the required keys. For fast unit coverage, factor the env-assembly into `build_trial_env(arm, trial_dir) -> dict` and unit-test it (asserts `ENGRAM_CHUNKS_DIR` set + per-trial, `ENGRAM_RECENCY=off` present iff `arm=="off"`, `ENGRAM_VAULT_PATH` set).
- [ ] **Step 2: Run to verify fail** — Expected: FAIL (functions missing).
- [ ] **Step 3: Implement `build_trial_env` + `run_trial` + `main`.**
- [ ] **Step 4: Run the env unit test to pass** — Expected: PASS.
- [ ] **Step 5: Commit** — `feat(eval): #646 two-phase recency ON/OFF trial runner`.

### Task B3: aggregation + metrics report

**Files:**
- Create: `dev/eval/traps/recency_value_agg.py` (or an `--aggregate` mode in the driver)
- Test: `dev/eval/traps/recency_value/test_agg.py`

**Interfaces — produces `aggregate(trials) -> dict`:**
- `n_valid` per arm (excludes `phase1_ok == False`).
- `correct_rate` per arm.
- `surfaced_rate` per arm (the note-83 diagnostic).
- `efficiency` — mean phase-2 cost/turns/dur **computed only over `correct == True` trials** (note 292); reported as `None` when an arm has 0 correct.
- A `verdict_inputs` block: (only-path? ON correct & OFF ~0), (cheaper/faster among correct?), sized against the same-arm trial spread.

- [ ] **Step 1: Write failing aggregation tests** — feed synthetic trial dicts; assert efficiency ignores incorrect trials, `surfaced_rate` counts correctly, `n_valid` excludes `phase1_ok=False`.
- [ ] **Step 2: Run to verify fail.**
- [ ] **Step 3: Implement `aggregate`.**
- [ ] **Step 4: Run to pass.**
- [ ] **Step 5: Commit** — `feat(eval): #646 metrics aggregation (correct/surfaced/efficiency-among-correct)`.

---

## 5. Pilot Milestone (P) — STOP for user go/no-go

**Purpose (note 196):** validate the contrast is non-vacuous AND capture the real per-trial $ before any batch. This is the zero-stub real-engram anchor (note 197).

### Commands
```bash
# 1 trial per arm, opus, real engram end-to-end
python3 dev/eval/traps/recency_value.py --arm on  --trials 1 --model opus --out /tmp/646-pilot-on.json
python3 dev/eval/traps/recency_value.py --arm off --trials 1 --model opus --out /tmp/646-pilot-off.json
python3 dev/eval/traps/recency_value_agg.py /tmp/646-pilot-on.json /tmp/646-pilot-off.json
```

### Pass/fail signals (pre-registered)
- **P1 — phase-1 capture works:** both trials `phase1_ok == True` and the transcript contains the milli-dollar narration (grep the phase-1 `.jsonl`). If phase 1 doesn't reliably discover+narrate → tune the validator message / prompt, re-pilot.
- **P2 — non-vacuous contrast (the critical gate):** ON `surfaced == True` AND OFF `surfaced == False` (or markedly lower rank). If OFF *also* surfaces the unit → the scenario is not recency-only (cosine leak or an un-neutralized escape) → redesign topical distance / withhold more before batching. **This is the note-195 vacuous-verdict guard.**
- **P3 — the mechanism can bite:** ON `correct == True` on at least the single pilot trial is encouraging but not required; OFF `correct == False` is the signal that the dead-end is real. (A single trial is anecdotal — the batch decides; P3 just confirms the machinery produces a live result, not an error.)
- **P4 — clean run:** no degraded-build retries exhausted, no isolation error (both arms' payloads confirm the per-trial `ENGRAM_CHUNKS_DIR` was used, not the global index).

### Cost/time capture
Record real `phase1_cost + phase2_cost` per trial → **per-trial $**. Multiply by `2 arms × N` for the batch estimate (note 101).

### STOP
Report to Joe: the two payloads, P1–P4 verdicts, per-trial $, and a proposed batch **N** (with the resulting $ estimate and the same-contrast noise-floor rationale). **Do not run the batch without go.**

---

## 6. Full Run (post-approval)

### Command
```bash
python3 dev/eval/traps/recency_value.py --arm on  --trials $N --model opus --out /tmp/646-on.json
python3 dev/eval/traps/recency_value.py --arm off --trials $N --model opus --out /tmp/646-off.json
python3 dev/eval/traps/recency_value_agg.py /tmp/646-on.json /tmp/646-off.json > /tmp/646-report.txt
```

### Metrics — presented as a labeled criteria table (memory-note: always a units-labeled table)
Columns: arm · n_valid · correct_rate · surfaced_rate · mean phase-2 $ (correct-only) · mean turns (correct-only) · mean dur ms (correct-only) · Δ vs OFF. Never narrate numbers in prose; never collapse to a bare %.

### Win-bar arithmetic (report explicitly)
- **Only-path win:** ON correct_rate high AND OFF correct_rate ≈ 0 → recency is the only path to unassisted correctness (note 99 capability case).
- **Cheaper/faster win:** both arms reach correctness but ON is cheaper/faster among correct trials, beyond the same-arm spread (note 292).
- Size every ON−OFF gap against the within-arm trial spread; a gap below it is underpowered, not a tie.

---

## 7. Analysis, LEDGER & Docs (Definition-of-Done)

### 7.1 Doc-surface enumeration grep — disposition (verified 2026-07-18)

| File:line | Disposition | Reason |
|---|---|---|
| `docs/FEATURES.md:159` ("the recency channel's delivery is not separately eval'd") | **UPDATE** | Replace with #646's verdict (proven / refuted / situational) + LEDGER anchor. The core claim this issue closes. |
| `dev/eval/LEDGER.md` | **ADD ROW** `#646-recency-e2e-value` | DoD ("updating the ledger is part of every eval's definition-of-done"). Verdict vocab: proven\|refuted\|unmeasured\|superseded, + figure + raw-data path. |
| `docs/ROADMAP.md:83` (#646 row) | **UPDATE** | Flip to the result; move #646 out of NOW rank 1. |
| `docs/ROADMAP.md:96` (#648 row) | **UPDATE** | Unblock #648 (its blocker #646 resolved). |
| `docs/GLOSSARY.md:308`, `:561` (recency channel / `--recent-fill`) | **UPDATE** | Document the new `ENGRAM_RECENCY=off` master toggle alongside `--recent-fill`. |
| `docs/architecture/c3-components.md:103` (K6 `query.go` row) | **UPDATE** | Note the `ENGRAM_RECENCY` toggle in the query-path description (it names `--recent-fill`/`ENGRAM_RECENT_FILL`). |
| `README.md:93` (`engram query` flag list) | **UPDATE** | Add `--recency on\|off` to the documented flag list. |
| `docs/ROADMAP.md:111,158,170,213`, `docs/architecture/c1-system-context.md:110`, `LEDGER.md#glance-fails-c5-delivery`, `#crowded-vault-capability-robustness` | **KEEP (N/A)** | Reference *other* recency levers (cluster centroid, C5-apply, channel-split), not #646's delivery value or the query toggle. Not stale. |

### 7.2 Tasks
- [ ] **Task C1:** Run the aggregator; write the results table + verdict.
- [ ] **Task C2:** Add the `dev/eval/LEDGER.md` row (verdict + vintage + n + raw-data path).
- [ ] **Task C3:** Update `docs/FEATURES.md:159`, `docs/ROADMAP.md` (#646 result + #648 unblock), and the `ENGRAM_RECENCY` doc sites per 7.1.
- [ ] **Task C4:** Deliver the results table to Joe (attach the report file — memory: attach deliverables).

---

## 8. Pre-Registered Acceptable Outcomes (honest null — write BEFORE the batch)

1. **Only-path win:** ON reaches unassisted correctness where OFF cannot (OFF correct_rate ≈ 0). → "recency recall is the only path to recovering an idiosyncratic self-captured decision after context loss." LEDGER: proven.
2. **Efficiency win:** both arms reach correctness but ON is cheaper/faster among correct trials beyond the spread. → proven (efficiency).
3. **Null — situational:** among correct runs, ON is neither the only path nor cheaper/faster, OR the ON−OFF correctness gap is below the within-arm spread. → "recency helps only situationally"; #648 tunes on surfacing evidence alone (the diagnostic surfaced_rate), not end-to-end value. LEDGER: unmeasured/refuted as an op-value win, with the surfacing figure recorded.
4. **Vacuous (pilot must prevent):** OFF also recovers the unit → contrast invalid; not a result — redesign, do not report as a null.

---

## 9. Cost Estimate

Per trial = phase-1 build + phase-2 build (two small opus `claude -p` builds, few turns each). From note 95 (opus 2-round CRUD ≈ few $/build), rough prior **~$3–8/trial**; the **pilot replaces this guess with the real number** before any batch (note 101). Pilot itself ≈ 2 trials ≈ **~$6–16**. A batch of N=10/arm ≈ **~$60–160** on opus (confirmed against pilot per-trial × 20 before launch). No interrupting spend cap; estimate-and-confirm, then let it finish.

## 10. Risks & Open Questions

- **R1 — vacuous contrast (highest):** phase-2's query cosine-surfaces the phase-1 note even OFF. Mitigation: the recency-only constraint (topical distance + withheld CSV) + the P2 pilot gate. If P2 fails, redesign before spend.
- **R2 — phase-1 non-capture:** the agent fixes the validator without narrating *why* (no unit mention in transcript). Mitigation: P1 gate; if it recurs, add a phase-1 prompt nudge ("note any non-obvious conventions you discover") — kept identical across arms so it doesn't bias.
- **R3 — recall doesn't fire in phase 2:** the agent skips `/recall`. Mitigation: the phase-2 prompt explicitly instructs recall-then-build (both arms); a trial where recall didn't fire (no `engram query` in the transcript) is excluded and counted, not scored.
- **R4 — over-narrow N:** if the pilot per-trial $ forces a small N, report the underpowered scope honestly (gap below spread = underpowered, not a tie) rather than over-claiming.
- **R5 — the toggle disturbs the phase timer:** Unit A must gate the *ranking* now, not the timer clock. Covered by the Unit-A test asserting timing still works with `--timings` under `Recency:off` (add that assertion in Step 1 if `--timings` is exercised).

## Self-Review

- **Spec coverage:** mechanism (§2), toggle (§3), harness (§4), pilot gate (§5), batch (§6), LEDGER/docs (§7), null (§8), cost (§9) — every design element from the approved design maps to a task. ✓
- **Placeholders:** none — every task has concrete files, code, commands, expected output.
- **Type consistency:** `run_trial` dict keys (`arm/idx/correct/surfaced/phase1_ok/phase2_cost/...`) are consumed identically by `aggregate`; `recencyEnabled`/`Recency` names consistent across §3.
- **Non-waivable grep:** done and dispositioned (§7.1).
- Open item deferred to pilot by design: exact batch **N** (set post-pilot from real $).
