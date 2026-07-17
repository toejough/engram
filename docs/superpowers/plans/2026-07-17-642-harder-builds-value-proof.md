# Harder-Builds Memory Value-Proof (#642) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. **This plan STOPS at the pilot gate (Milestone P) for a user go/no-go before any full-envelope spend.**

**Goal:** Run an adequately-powered (n>=8/arm) self-seeded warm-vs-cold memory value-proof on a *harder*, gotcha-concentrated, non-convergent, correctness-gated multi-round build regime, and rewrite `dev/eval/LEDGER.md#c1-c2-warm-op-negatives` with the new verdict + vintage.

**Architecture:** Reuse the existing cumulative-accumulation vehicle (`dev/eval/cumulative/matrix.py` orchestrator + `harness.py` per-op build loop + `aggregate.py` noise-floor analysis) — the same machinery that produced the current `c1-c2-warm-op-negatives` figure. Add a new **"hard" app-set**: three fictional-domain record-manager apps that share ~8 idiosyncratic, un-guessable **house-convention gotchas**. The gotchas live only in the behavioral checks (never in the build prompt's interface string), so a COLD agent must reverse-engineer them across feedback rounds (rebuild waste) while a WARM agent recalls them from app-1's self-learned notes. Cold vs `real.full` regimes give self-seeded warm-vs-cold; `n=8` trials give power; the re-measured recall+learn tax and warm-vs-warm noise floor fall out of the same run.

**Tech Stack:** Python 3 eval harness (`dev/eval/cumulative/`), Go build targets (the apps under test), `claude -p` headless with isolated `CLAUDE_CONFIG_DIR` clean-room cfg pool, real `/recall` + `/learn` skills, `engram` binary.

## Global Constraints

- **Vehicle is `dev/eval/cumulative/matrix.py` + `harness.py` + `aggregate.py`** — NOT the thinner `run-accumulation-chain.py`/`run-chain-stage.sh` wrapper (that has no correctness gate, no cost/time axes, no recall/learn-fired assertion, no noise floor). The done-condition's requirements (correctness-gated, n>=8/arm, warm-vs-warm noise floor, updates `c1-c2-warm-op-negatives`) are exactly what matrix/harness/aggregate already produce. `run-accumulation-chain.py` remains the conceptual self-seeding statement; this plan implements the measured form.
- **Do NOT modify the existing `notes`/`links`/`feeds` app-set or its specs** — it is the live easy-CRUD baseline. Add the hard set alongside it, selected by a new `--app-set` flag (default preserves current behavior).
- **Do NOT touch `archscore.py`'s 10 generic detectors** — they score all apps uniformly; changing them corrupts the existing baseline. The gotchas are behavioral, slotted via new spec files.
- **Gotcha rules never appear in `spec["interface"]`** (the only spec field `build_prompt()` injects, `harness.py:167-172`). They live only in the shared house-gotcha `checks` (symptom + steps), surfaced to cold via `feedback_prompt()` escalation and to warm via `/recall`.
- **Fictional domain, greenfield** — invented sigils/separators/exit-codes that cannot be guessed and cannot leak from `~/.claude/CLAUDE.md` (the ambient-conventions contamination that killed the original Run-1 calibration). This is the design's insurance and must be verified in the pilot's headless-clean check.
- **Three separate reporting axes, never collapsed** (memory-note 99): CAPABILITY (correctness on the gotchas) and OP-COST (time/$) are reported independently; a capability win can co-exist with a still-negative op-cost verdict.
- **Re-measure the fixed recall+learn tax** in-run (memory-note LEDGER caveat): the stale ~186 s / ~$1.80 figure is pre-payload-cut (predates the 2026-06-27 lazy-chunks/recent-fill cuts). Do not inherit that denominator; recompute it from this run's `recall_s`+`learn_s` and `recall_cost`+`learn.cost`.
- **Size every cold-vs-warm gap against a warm-vs-warm noise floor from the same contrast** (`aggregate.py:noise_floor`); a gap below the floor is *underpowered*, not a tie.
- **Pre-register the acceptable-outcomes list (Section 7) BEFORE the full run** so results can't be p-hacked.
- **No spend cap that interrupts a run** (memory-note): estimate + confirm cost up front, then let it finish. `matrix.py --budget 0` (the default) imposes no ceiling.

---

## 1. Goal & Done-Condition

**Surviving question (from the issue):** Does engram memory net *help* a real headless agent (lower cost/turns, higher correctness) end-to-end on a *harder, multi-round* build regime — not just on isolated single-convention traps?

**Done-condition (verbatim intent):** A clean, adequately-powered (n>=8/arm) self-seeded warm-vs-cold measurement on a harder multi-round build regime (non-convergent, real dead-ends, correctness-gated) that either (a) shows memory nets cheaper/faster/more-correct beyond the warm-vs-warm noise floor, or (b) refutes it with adequate power — and **updates the `c1-c2-warm-op-negatives` LEDGER row** (currently scoped "underpowered for harder multi-round builds") with the new verdict + vintage. Also flip `docs/ROADMAP.md` lines 97 & 208 (harder-builds eval) to "Measured".

---

## 2. Design of the Fictional Gotcha-Concentrated Regime

### 2.1 Domain

A fictional records-tooling family for an invented system (names are placeholders; keep them nonsensical so nothing can leak from real-world priors):

- **app1 `wardex`** — a "ward" registry
- **app2 `glyphex`** — a "glyph" registry
- **app3 `relayex`** — a "relay" registry

All three are command-line record managers with the SAME subcommand surface (`add`, `list`, `show`, `edit`, `rm`, `search`, `tag`, `export`, `import`, `history`) and — critically — the SAME idiosyncratic **house conventions**. This mirrors how the existing chain works: the transferable signal is the recurring conventions, learned on app1 and recalled on app2/app3; per-app native features are the non-transferable control.

### 2.2 The interface string (what the build prompt reveals)

Each spec's `interface` describes ONLY the subcommand surface and says *"follow this project's house conventions"* WITHOUT stating them — exactly as a new engineer joining a codebase is told conventions exist but must learn them. Warm learns them from recall; cold discovers them through review feedback. Example (wardex):

```
A command-line ward registry `wardex`. Subcommands: add <text> [--tag X ...], list [--tag X] [--search Q] [--deep] [--raw] [--json], show <sigil>, edit <sigil> <text>, rm <pos>, tag <sigil> <tag>, untag <sigil> <tag>, export <file>, import <file>, history <sigil>. Persist to a file. This project has established HOUSE CONVENTIONS for id format, persistence format, addressing, provenance, error protocol, tag normalization, search scoping, and audit journaling — follow them. (You will receive review feedback if your implementation violates a house convention.)
```

### 2.3 The ~8 shared house gotchas

Each is token-level, un-guessable, cross-app (recurs in all 3 specs via a shared `house_gotchas.json`), and encoded as a behavioral check (`{name, bucket:"house", symptom, steps:[{argv, assert}]}`). "Why cold rebuilds" = the obvious first implementation the cold agent writes is *wrong* and the symptom feedback forces rework. Three are **dead-ends** (the naive build must be torn out, not merely extended).

| # | Gotcha | House rule (token-level) | Naive cold guess | Why cold rebuilds | Dead-end? |
|---|--------|--------------------------|------------------|-------------------|-----------|
| G1 | **Sigil id format + no-reuse** | ids are `⟐NNNN` zero-padded width-4 from a monotonic counter that NEVER reuses after delete | `1,2,3` reused | add 3 / rm 2 / add 1 must yield `⟐0004`, not `⟐0002`; wrong counter → rework id allocation | yes |
| G2 | **Dual addressing** | `show`/`edit`/`tag` take a stable `⟐NNNN` sigil; `rm` takes a 1-based CURRENT-list POSITION; positions renumber, sigils don't | one id scheme for all | rm-by-position after building rm-by-id forces addressing rework across commands | yes |
| G3 | **Fenced persistence** | on disk: fields joined by `\x1f` (US), records by `\x1e` (RS) — NOT JSON/CSV/newline; a value with commas/newlines/quotes/tabs round-trips byte-identical; `export`/`import` use this fence format | JSON or CSV | add a value `a,b\n"c"\td`; `list --raw` returns it exact; `export` output must contain `\x1e` (a JSON exporter fails interop) → rip out serializer | yes |
| G4 | **Provenance-branch suffix** | `list` renders each record with a suffix: `+` added, `~` ever-edited, `<` imported, composing in fixed order (imported-then-edited = `<~`) | no suffix / ad-hoc | import→edit must show `<~`; fresh add shows `+` | no |
| G5 | **Not-found protocol** | unknown sigil → exit code `7`, stderr exactly `absent: <sigil>`, stdout empty | exit 1 / "not found" | `show ⟐9999` must exit 7 with `absent: ⟐9999` | no |
| G6 | **Canonical tags** | tags lowercased, deduped, sorted, joined with `·` (U+00B7) in storage + output; query matches any case | comma-join, case-preserving | tag Beta/alpha/BETA → `alpha·beta` | no |
| G7 | **Fenced search scoping** | `search <q>` matches TITLE only, case-insensitive substring, ignoring the sigil field; `search --deep <q>` also matches tags + body | match everything | body-only "zzq" found by `search --deep` but NOT `search` | no |
| G8 | **Ledger journaling** | every mutating command appends `<epoch>\x1f<cmd>\x1f<sigil>` to a sidecar `.ledger`; `history <sigil>` replays it in order | no journal | add→edit→`history ⟐0001` shows 2 lines in order | yes-ish |

Concentration (8 gotchas, ≥3 dead-ends) is the make-or-break property (design-req #1, memory-note 98): a generic "harder CRUD" would reproduce the prior null. The pilot's RED gate literally tests "cold accrues multi-round rebuild waste" on these.

### 2.4 Why this satisfies the hard constraints

- **Non-convergent / real dead-ends:** G1–G3, G8 require tearing out the obvious first implementation.
- **Un-guessable + contamination-proof:** invented separators/sigils/exit-codes/joiner cannot be guessed and cannot leak from ambient `~/.claude/CLAUDE.md`.
- **Cross-app transfer:** identical house gotchas in all 3 specs → app1-learn helps app2/app3 for warm; cold repeats full rediscovery per app.
- **Correctness-gated:** each gotcha is a behavioral assertion; `converged()` (`harness.py:339`) requires all feature buckets pass + `arch_pass>=8`.
- **Bounded, fair waste:** `feedback_prompt()` escalates symptom→concrete `run X expect Y` (via `_spec_check_detail`, `harness.py:216`) by round 3, so cold DOES converge — the waste is measured extra rounds/$, not failure. (Where even escalation can't rescue cold but recall rescues warm, that is a CAPABILITY win — memory-note 99 — which is the acceptable-outcome the design also captures.)

---

## 3. Harness Wiring — Concrete Changes

All paths absolute under `/Users/joe/repos/personal/engram/`.

### Task 1: `exit:N` assert kind in the behavioral scorer (needed for G5)

**Files:**
- Modify: `dev/eval/cumulative/behavioral.py:65-77` (`assert_on`)
- Test: `dev/eval/cumulative/test_behavioral_assert.py` (create)

- [ ] **Step 1 — failing test.** Create `test_behavioral_assert.py`:
```python
import behavioral
def test_exit_code_assert():
    assert behavioral.assert_on(7, "", "exit:7") is True
    assert behavioral.assert_on(1, "", "exit:7") is False
```
- [ ] **Step 2 — run, expect FAIL:** `cd dev/eval/cumulative && python3 -m pytest test_behavioral_assert.py -q` → fails (`exit:` unhandled → returns False for the 7 case).
- [ ] **Step 3 — implement.** In `assert_on`, before the final `return False`, add:
```python
    if kind.startswith("exit:"):
        try: return rc == int(kind.split(":", 1)[1])
        except Exception: return False
```
- [ ] **Step 4 — run, expect PASS.**
- [ ] **Step 5 — commit** (`feat(eval): exit:N assert kind for behavioral scorer`).

### Task 2: Shared house-gotcha checks + merge hook

**Files:**
- Create: `dev/eval/cumulative/house_gotchas.json` — `{"house_checks": [ ...G1..G8 as behavioral checks, bucket "house"... ]}`
- Modify: `dev/eval/cumulative/score.py:24-26` (merge house checks into `spec["checks"]` when `spec.get("house_checks_file")`)
- Test: `dev/eval/cumulative/test_house_merge.py` (create)

Rationale for a merge hook (vs duplicating checks in each spec): DRY across the 3 hard specs, and the gotcha block stays a single source of truth so all 3 apps share byte-identical conventions (required for transfer).

- [ ] **Step 1 — author `house_gotchas.json`.** Encode G1–G8 from Section 2.3 as `checks` entries. Use `\x1f`/`\x1e`/`⟐`/`·` as literal JSON string escapes; use `assert:"exit:7"` for G5. Each check carries a `symptom` (the user-visible surprise, NOT the rule) so `feedback_prompt` states it, and steps whose `argv`+`assert` encode the rule so `_spec_check_detail` can escalate to the concrete example by round 3.
- [ ] **Step 2 — failing test.** `test_house_merge.py`: a spec with `"house_checks_file": "house_gotchas.json"` and 1 native check → `score.score()`'s loaded spec must contain 9 checks (1 native + 8 house). Assert on the merged count via a helper `load_spec(path)`.
- [ ] **Step 3 — run, expect FAIL** (no merge yet).
- [ ] **Step 4 — implement** in `score.py`: extract a `load_spec(path)` that does `spec = json.load(open(path)); hcf = spec.get("house_checks_file"); if hcf: spec["checks"] = spec["checks"] + json.load(open(os.path.join(os.path.dirname(path), hcf)))["house_checks"]; return spec`. Call it wherever `score.py`, `behavioral.py`, and `harness.py` currently do `json.load(open(specpath))` (grep: `harness.py:682,741`, `score.py:25`, `behavioral.py:123`). Keep it one shared helper imported everywhere to avoid drift.
- [ ] **Step 5 — run, expect PASS. Commit** (`feat(eval): shared house-gotcha spec merge`).

### Task 3: The three fictional app specs

**Files:**
- Create: `dev/eval/cumulative/wardex_spec.json`, `glyphex_spec.json`, `relayex_spec.json`

Each: `{"app": "...", "interface": "<Section 2.2 surface, no rules>", "house_checks_file": "house_gotchas.json", "buckets": {...}, "checks": [ <2-3 app-specific native checks, bucket "native"> ]}`.

- [ ] **Step 1 — write the three specs** (differ only in app name + native checks; all pull the same house block).
- [ ] **Step 2 — validate they load + a reference impl converges.** Sanity: hand-write nothing; instead validate structurally — `python3 -c "import score; print(len(score.load_spec('wardex_spec.json')['checks']))"` prints 10–11. (Full correctness of the specs is proven by the pilot's cold builds converging under escalation; that IS the spec's acceptance test.)
- [ ] **Step 3 — commit** (`feat(eval): fictional hard-regime app specs (wardex/glyphex/relayex)`).

### Task 4: `--app-set` selector in the matrix orchestrator

**Files:**
- Modify: `dev/eval/cumulative/matrix.py:34` (`APP_SPEC`), `:110-143` (`real_cells_for` hardcoded `apps=[("notes",...),...]`), `:236-249` (argparse)
- Modify: `dev/eval/cumulative/aggregate.py` (the `notes`/`links`/`feeds` app-name assumptions in `chain_intervention_table:142`, `main:727`) OR pass the app-set through — see Task 6.

- [ ] **Step 1 — introduce `APP_SETS`** near line 34:
```python
APP_SETS = {
    "crud": [("notes", "app1"), ("links", "app2"), ("feeds", "app3")],   # existing baseline (default)
    "hard": [("wardex", "app1"), ("glyphex", "app2"), ("relayex", "app3")],
}
```
- [ ] **Step 2 — parametrize `real_cells_for(...)`** to take `apps` (replace the hardcoded list at `:117`) and build `--spec` from `f"{CUM}/{app}_spec.json"` (already generic at `:132`). Thread `apps` from `ops_for` → `main`.
- [ ] **Step 3 — add `--app-set` arg** (`default="crud"`, `choices=list(APP_SETS)`) and a `--max-apps N` arg (default 3; the pilot uses 2 to cap the chain). Slice `apps = APP_SETS[args.app_set][:args.max_apps]`.
- [ ] **Step 4 — record `app_set` in the manifest** (`write_manifest`) so `aggregate.py` can read it.
- [ ] **Step 5 — stub smoke:** `CUMMATRIX_ROOT=/tmp/hardsmoke python3 matrix.py --app-set hard --models sonnet --trials 1 --regimes cold --stub good --max-apps 2` → produces result JSONs with the fence/sigil checks scored deterministically (validates wiring/threading with zero LLM spend). Expect exit 0 and 2 result files.
- [ ] **Step 6 — commit** (`feat(eval): --app-set/--max-apps; hard regime wiring`).

---

## 4. Pilot Milestone (RED gate) — STOP for user go/no-go

Purpose: confirm (RED) COLD accrues genuine multi-round rebuild waste on the hard regime, (mechanism) the warm learn→recall→apply path transfers the gotchas, and (headless-clean) the run is clean and per-build cost is captured — BEFORE any full-envelope spend.

### Commands

```bash
cd /Users/joe/repos/personal/engram/dev/eval/cumulative
export CUMMATRIX_ROOT=/tmp/hard-pilot
# (a) COLD rebuild-waste RED signal: fictional app1+app2, n=3, cold only
python3 matrix.py --app-set hard --models sonnet --trials 1,2,3 --regimes cold --max-apps 2 --workers 3
# (b) WARM transfer mechanism check: one self-seeded pair app1(seed)->app2(recall), n=1
python3 matrix.py --app-set hard --models sonnet --trials 1 --regimes real.full --max-apps 2 --workers 2
```
Total ~8 headless builds (6 cold + 2 warm). Model = sonnet for pilot economy (the full run's model is a user decision — see Section 8/9). `--max-apps 2` includes app2 so transfer is actually exercised (deviation from the "1 app" guidance, with rationale — see Open Questions).

### Pass/fail signals (pre-registered)

Read from each `${CUMMATRIX_ROOT}/results/*-build.json` (`rounds_to_converge`, `round1_feature_fails`, `build_cost`, `recall_cost`, `learn.cost`, `recall_s`/`build_s`/`learn_s`, `wall_min`, `converged`, `recall_fired`, `learn.fired`, `rate_limited`, `tokens.cache_read`). Score the house-gotcha subset by intersecting `final_buckets`/round-1 fails with bucket `"house"`.

- **RED (make-or-break) — PASS iff:** cold median `rounds_to_converge` >= 3 AND cold median round-1 **house**-gotcha fails >= 5/8 AND cold median per-build $ materially above the easy-CRUD ~$2–4 anchor. → cold genuinely rediscovers across rounds. **FAIL** (cold clears gotchas in <=2 rounds) ⇒ regime too easy; redesign gotchas (add dead-ends / under-determine symptoms), do NOT proceed.
- **Mechanism — PASS iff:** on the warm pair, `recall_fired>=1` and `learn.fired` truthy on app1; app1's vault gained house-gotcha notes (`notes_written>0`, inspect the notes name the sigil/fence/exit-7 rules); warm app2 round-1 house-gotcha fails **< cold app2** round-1 house-gotcha fails. **FAIL** ⇒ learn/recall isn't transferring the gotchas; investigate (learn-capture, recall surfacing, apply) before spending on a full run.
- **Headless-clean — PASS iff:** no `rate_limited` cells; cfg pool is the clean-room template (`build_cfg_template`, warm carries only recall+learn skills); **leakage check** — cold's round-1 output does NOT already satisfy the gotchas (confirms they are un-guessable and did not leak from ambient `~/.claude/CLAUDE.md`); per-build wall-time + $ present in every result JSON.

### Cost/time capture

Tabulate the pilot's real per-build `build_cost`, `recall_cost`, `learn.cost`, `recall_s+learn_s` (the re-measured fixed tax, pilot-grade), and `rounds_to_converge`. This calibrates the full-run projection handed to the user.

### STOP

Report to the user a labeled criteria table (memory-note: always a labeled criteria table with units) of the three gates + calibrated full-run cost/wall projection, and request go/no-go. **Do not launch the full run without approval.**

---

## 5. Full Run (post-approval)

### Command

```bash
cd /Users/joe/repos/personal/engram/dev/eval/cumulative
export CUMMATRIX_ROOT=/tmp/hard-full
python3 matrix.py --app-set hard --models <opus|sonnet, user-decided> \
    --trials 1,2,3,4,5,6,7,8 --regimes cold,real.full --max-apps 3 --workers 4
```
- **Arms:** `cold` (no memory) vs `real.full` (self-seeded: each app recalls, builds, `/learn`s in-session; vault promoted app1→app2→app3). n=8 trials/arm.
- **Cells:** 8 trials × 2 regimes × 3 apps = 48 build ops (24 cold + 24 warm). Resumable (`op_done` skips valid results; re-runs rate-limited/timeout cells).
- **Model:** default to **opus** to match the `c1-c2-warm-op-negatives` vintage (n=8 opus) so the new verdict is comparable; sonnet is the cheaper alternative if the pilot shows sonnet already accrues clear waste (user decides at the gate).

### Metrics — three separate axes (memory-note 99), from `harness.py` output + `aggregate.py`

1. **CAPABILITY axis:** round-1 house-gotcha pass rate (warm app2/app3 vs cold), `converged` rate, `rounds_to_converge`, final correctness. This is where memory is expected to be a clean win on idiosyncratic content.
2. **OP-COST axis:** `axis_c1_recall_s`/`build_s`/`learn_s`; `axis_c2_cost_usd` (build) + `recall_cost` + `learn.cost`; chain-summed total op $ and wall. Cold vs warm.
3. **RE-MEASURED FIXED TAX (design-req #3):** current-vintage recall+learn tax = mean(`recall_s`+`learn_s`) and mean(`recall_cost`+`learn.cost`) across warm cells — replaces the stale ~186 s/$1.80. PLUS the per-round **cache-read premium** (design-req #2): warm build rounds re-read recalled context every turn; quantify from `tokens.cache_read` × price on warm build rounds.
4. **NOISE FLOOR (design-req #5):** `aggregate.py:noise_floor` = warm-vs-warm 95% CI half-width from the SAME contrast; `gap_label`/`axis_ci_table` tag each cold-vs-warm gap significant/underpowered. n=8 targets adequate power.

### Win-bar arithmetic (memory-note 95 — report explicitly, do not collapse)

For a net op-win, warm's avoided-rebuild saving on app2/app3 (cold op-cost − warm build-cost) must exceed the re-measured fixed tax + the cache-read premium. Report the subtraction with units and CIs; the sign of `(saving − tax − premium)` is the op-cost verdict.

---

## 6. Analysis & LEDGER Update

### Task 5: Run the aggregator + a shared-gotcha transfer slice

**Files:**
- Run: `python3 dev/eval/cumulative/aggregate.py --root /tmp/hard-full --out /tmp/hard-full/results-agg.md`
- Create (if the built-in tables don't cover the "house" bucket transfer): `dev/eval/cumulative/hard_analysis.py` — a small standalone that reads `results/*-build.json`, computes per-(regime,app) round-1 **house**-gotcha pass rate + the win-bar arithmetic, and emits a labeled table. (Prefer extending `aggregate.py`'s `axis_ci_table`/`chain_intervention_table` to the `house` bucket if it's a clean 1-function add; otherwise standalone to avoid destabilizing the live aggregator.)

- [ ] Produce: (1) axis CI table (times + $, cold vs warm, noise-floor tags); (2) house-gotcha capability table (round-1 pass rate + converged, cold vs warm, app1/app2/app3); (3) re-measured fixed-tax line (s + $) with the cache premium; (4) the win-bar subtraction with CIs.

### Task 6: Rewrite the LEDGER row + ROADMAP

**Files:**
- Modify: `dev/eval/LEDGER.md:39` (`c1-c2-warm-op-negatives`)
- Modify: `docs/ROADMAP.md:97` and `:208` (harder-builds eval → Measured)

Rewrite `c1-c2-warm-op-negatives` to KEEP the easy-CRUD refuted finding as historical scope and ADD the harder-regime verdict with fresh vintage (2026-07-17, post-payload-cut). New row content must carry: the three-axis result (capability / op-cost / re-measured tax + cache premium), noise-floor labels + n=8 power, the win-bar arithmetic, and the honest-null disposition if op-cost stays negative. Change the scope clause "underpowered for harder multi-round builds, not 'memory can't pay off'" to "**measured** on the gotcha-concentrated hard regime (n=8/arm): <verdict>." Cite `results-agg.md` + the result JSONs as raw data; keep the format identical to the existing rows (`| claim | verdict | figure (vintage) | superseded-by | raw data |`). Verdict token per Section 7 outcome. Flip ROADMAP lines 97/208 to "Measured" pointing at the new LEDGER row.

- [ ] Commit (`docs(ledger): harder-builds op-value verdict + vintage; ROADMAP measured`).

### 6.1 Doc-surface enumeration grep (non-waivable) — disposition list

Grep run 2026-07-17 over `docs/ dev/eval/LEDGER.md README.md CLAUDE.md` for `c1-c2-warm-op-negatives|warm-op-negatives|+182s|+$3.08|underpowered for harder|642`. The changing invariant is **the harder-builds eval verdict / #642 status**. Every echo + its disposition (Gate A's docs/diagrams reviewer verifies this AND runs its own independent scan):

| File:line | Current text | Disposition | Reason |
|-----------|--------------|-------------|--------|
| `dev/eval/LEDGER.md:39` | `c1-c2-warm-op-negatives` refuted, "underpowered for harder multi-round builds" | **rewrite** | Keep easy-CRUD scope, ADD harder-regime verdict + 2026-07-17 vintage + re-measured tax (Task 6) |
| `dev/eval/LEDGER.md:38` | `c1-c2-warm-op-mislabeled` superseded (historical) | **keep** | Historical accounting, unaffected by the new measurement |
| `docs/ROADMAP.md:97` | "harder-builds eval (measurement candidate; designed, unrun; no issue #)" | **rewrite** | It now IS #642 and is being run — flip to Measured, point at the new LEDGER row, drop "no issue #" |
| `docs/ROADMAP.md:208` | "Harder-builds eval baseline … designed but never run" (GATED band) | **rewrite** | Fill in the measured result; move out of the unrun-candidate framing |
| `docs/ROADMAP.md:84` | "#642 … self-seeding cold-vs-warm … value-proof spine" (NOW, rank 2) | **update** | Once measured, #642's status/ranking changes (spine delivered); reconcile with the closure in Step 6 |
| `docs/ROADMAP.md:96` | "#646 e2e recency value-proof … \| #642" | **verify/keep** | #646 is the distinct recency proof still blocked on #642's harness; keep unless closure changes the blocker |
| `docs/ROADMAP.md:102` | "#648 tune usefulness-activation … \| #642 AND #646" | **keep** | Dependency edge unchanged by this measurement |

Note: vault notes 95/98/99 also carry the stale verdict but are updated via the closing `/learn` (Step 7), NOT by this doc-scrub. `dev/eval/traps/RESULTS.md` and `cumulative/EXPERIMENT-LOG.md` did not match the grep but the docs reviewer should confirm they carry no stale echo.

---

## 7. Pre-Registered Acceptable Outcomes (honest null — write BEFORE the full run)

Per memory-note 98, all four are legitimate; none is a failure to be engineered away:

1. **Capability-win + op-cost-win** (beyond noise): memory nets cheaper/faster AND more-correct → `c1-c2-warm-op-negatives` verdict flips to **proven (harder regime)**; the hard build-eval becomes a valid value harness.
2. **Capability-win + op-cost neutral/negative:** memory is more-correct on the gotchas but avoided-rebuild < tax + cache-premium → op-cost stays **refuted**, row gains "**capability win on idiosyncratic content** (note 99)"; conclusion: memory's value is capability/behavioral, and op-value measurement points to **real long-session work, not cheap build evals**. (This is the pre-registered honest null.)
3. **No capability-win, gap below noise:** **underpowered / can't distinguish** (memory-note: gap below noise = underpowered, NOT a tie); report n and either recommend more power or accept the null with that label.
4. **Regime-invalid** (cold accrued no waste even here — should be caught at the pilot): conclude the build-eval regime cannot manufacture memory op-value → definitively redirect value measurement to long-session/behavioral work.

The verdict token written to the LEDGER MUST be the one matching the observed axes — chosen from this list, not narrated post-hoc.

---

## 8. Cost Estimate

**Anchors (cite, with vintage):**
- Easy 2-round build: **$2–4/build** (memory-note 95, measured).
- Easy-CRUD warm net op **+$3.08** over cold; recall+learn tax **~$1.80 / ~186 s** — PRE-PAYLOAD-CUT (2026-06-25), to be re-measured (LEDGER `c1-c2-warm-op-negatives`).

**Harder-build projection (UNVERIFIED — the pilot calibrates it):** harder builds target >=3–4 cold rounds vs ~1–2 easy, so per-build $ scales up. Rough projection: cold hard build ~**$4–8**, warm ~**$5–9** (tax + cache premium). Opus is materially pricier than sonnet — treat any pre-pilot number as a projection, replace with the pilot's measured per-build $ before the full-run go/no-go.

**Pilot:** ~8 builds (6 cold + 2 warm) × ~$3–6 ⇒ **~$25–50**, wall ~**1–2 h** (3–4 workers, chain dependency serializes the warm pair). *Projection; report the actual spent tally honestly.*

**Full run:** 48 builds × ~$4–8 ⇒ **~$190–380** (sonnet) or **~$300–600** (opus). Wall: 48 builds / 4 workers × ~15–30 min/build ⇒ **~3–6 h**. Present the pilot-calibrated version to the user at the gate; no interrupting spend cap (`--budget 0`), but confirm the estimate up front.

---

## 9. Risks & Open Questions

**Risks:**
- **Gotchas too easy / symptoms too informative** — cold clears them in one feedback round → reproduces the null. Mitigation: 8 concentrated gotchas + >=3 dead-ends + under-determined symptoms; the pilot RED gate is the check.
- **Escalation can't rescue cold on a gotcha** — `STALL_PATIENCE=3` halts and the cell is flagged `did_not_complete`, truncating its op-cost and muddying the op-cost axis (though it cleanly feeds the CAPABILITY axis). Mitigation: ensure every gotcha's `_spec_check_detail` escalation reveals a concrete example by round 3; consider raising `--max-rounds` for the hard regime; treat halted cells per the capability axis, not op-cost.
- **Transfer doesn't fire** — learn captures the gotchas but recall doesn't surface them or the build doesn't apply them. Mitigation: pilot mechanism gate; the learn-capture fix (commit e07bde3d) already improved requirement capture.
- **`aggregate.py` app-name coupling** — several tables hardcode `notes`/`links`/`feeds` (`chain_intervention_table:142`, `main:727`). Mitigation: drive capability analysis via `hard_analysis.py` (Task 5) rather than forcing the coupled tables; use `aggregate.py` only for the app-name-agnostic axis CI / noise-floor tables.
- **`claude -p` + `CLAUDE_CONFIG_DIR` redirection** — uncertainty whether the user-global `~/.claude/CLAUDE.md` still loads under the redirected cfg. The fictional gotchas are robust either way (they cannot leak), which is exactly why fictional was chosen; the pilot leakage check confirms it empirically.

**Open questions — RESOLVED by the orchestrator (2026-07-17):**
1. **Full-run model:** RESOLVED — **pilot on sonnet; full-run model is Joe's call at the post-pilot go/no-go** (his chosen scope was "pilot, report, then decide", so the model decision lands naturally at that report, informed by the pilot's measured per-build cost). Plan default for the full-run remains opus for c1-c2 comparability, pending that decision.
2. **Pilot scope deviation:** RESOLVED — **accept the 2-app pilot** (`--max-apps 2`, one app1→app2 warm pair). Catching a no-transfer failure for ~$25–50 now beats discovering it after 48 builds; the marginal cost is trivial.
3. **Design A vs B:** RESOLVED — **Design A** (gotchas as behavioral checks). Minimal, YAGNI, no `archscore`/`score.py`/`harness`-core risk. Revisit B only if a later need forces the native transfer tables to carry the signal.

---

## Self-Review

- **Spec coverage:** design (S2), harness wiring w/ real paths+line numbers (S3), pilot RED gate + STOP (S4), full run n=8 + 3 axes + re-measured tax + noise floor (S5), analysis + LEDGER/ROADMAP update (S6), pre-registered nulls (S7), cost anchors+projection (S8), risks+open-qs (S9) — all present.
- **No placeholders:** the one deliberately-deferred detail is the exact JSON of the 8 checks (Task 2 Step 1) — bounded by the fully-specified rules in the S2.3 table (id format, separators, exit code, suffix codes, joiner, search scoping, ledger format), which is the authoring contract.
- **Type/name consistency:** `--app-set`/`--max-apps`/`APP_SETS`/`house_checks_file`/`load_spec`/`bucket:"house"`/`exit:N` used consistently across S3–S6.
