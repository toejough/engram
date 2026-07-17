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
    # a non-integer exit spec must NOT raise — the try/except int() returns False
    assert behavioral.assert_on(1, "", "exit:notanumber") is False
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

- [ ] **Step 1 — author `house_gotchas.json`.** Top-level `{"house_checks": [ ...G1..G8... ]}`, each entry `{name, bucket:"house", symptom, steps:[{argv, assert}]}`. **JSON encoding (D5/D13 — JSON has NO `\x` escape):** write the US separator as `\u001f` and RS as `\u001e`; write `⟐` and `·` as literal UTF-8 characters in the file (also fine inside a `contains:` assert string); use `"assert": "exit:7"` for G5. Each `symptom` states the user-visible SURPRISE, not the rule (so `feedback_prompt` states it and `_spec_check_detail` escalates to the concrete `run X expect Y` by round 3). **Authoring template — the COMPLETE G1 entry; copy this exact shape for G2–G8:**
```json
{
  "name": "sigil_no_reuse",
  "bucket": "house",
  "symptom": "The ids I get back look wrong and a deleted id gets handed out again — I expect stable, never-reused, zero-padded sigils.",
  "steps": [
    {"argv": ["add", "alpha"], "assert": "contains:⟐0001"},
    {"argv": ["add", "beta"],  "assert": "contains:⟐0002"},
    {"argv": ["add", "gamma"], "assert": "contains:⟐0003"},
    {"argv": ["rm", "2"],      "assert": "exit0"},
    {"argv": ["add", "delta"], "assert": "contains:⟐0004"}
  ]
}
```
  (`rm` takes the 1-based POSITION per G2; the `contains:⟐NNNN` asserts encode the sigil format + monotonic no-reuse rule that a naive `1,2,3`-reuse build fails. For G3/G8, the fence separators in `argv`/`assert` strings are `\u001f`/`\u001e`.)
- [ ] **Step 2 — failing test.** `test_house_merge.py` (`import score`), two cases against `score.load_spec(path)` using `tmp_path` fixtures (no shared mutable state): (a) a spec with `"house_checks_file": "house_gotchas.json"` + 1 native check → `score.load_spec(...)["checks"]` has 9 entries (1 native + 8 house); (b) a spec with NO `checks` key and no `house_checks_file` → `score.load_spec(...)["checks"] == []` (the missing-`checks` guard). Add `t.Parallel()`-equivalent isolation via distinct `tmp_path`.
- [ ] **Step 3 — run, expect FAIL** (no merge yet).
- [ ] **Step 4 — implement `load_spec` in `score.py`** (the reviewer's chosen home). `score.py` currently imports only `sys, json` — **add `import os`** (needed for `os.path`). Define the single source of truth with `with`-managed handles (no leaked descriptors) and a missing-`checks` guard:
```python
def load_spec(path):
    with open(path) as f:
        spec = json.load(f)
    spec.setdefault("checks", [])
    hcf = spec.get("house_checks_file")
    if hcf:
        with open(os.path.join(os.path.dirname(path), hcf)) as f:
            spec["checks"] = spec["checks"] + json.load(f).get("house_checks", [])
    return spec
```
  Replace the four raw `json.load(open(...))` spec loads (exact old→new per site):
  - `score.py:25` — `spec = json.load(open(specpath))` → `spec = load_spec(specpath)` (same module; no import).
  - `harness.py:682` — `interface = json.load(open(args.spec))["interface"]` → `interface = scoremod.load_spec(args.spec)["interface"]` (`harness.py` already `import score as scoremod`).
  - `harness.py:741` — `spec = json.load(open(args.spec))` → `spec = scoremod.load_spec(args.spec)` (**LOAD-BEARING**: this is the spec `feedback_prompt()`/`_spec_check_detail()` escalate from — it MUST carry the merged house checks, or cold never gets house-gotcha feedback and can't converge).
  - `behavioral.py:123` (`__main__`) — `spec = json.load(open(sys.argv[2]))` → **use a LOCAL import to avoid the `score`↔`behavioral` cycle** (`score.py` does `import behavioral` at module top). Inside `if __name__ == "__main__":` write `import score; spec = score.load_spec(sys.argv[2])` — safe because `score` is fully imported before that block runs. (This only affects a DIRECT `python3 behavioral.py <workdir> <spec>` run — `score.score()` passes an already-loaded spec into `behavioral.score()`, so the leakage check's `python3 score.py … wardex_spec.json` merges via `score.py:25`, not here; keep this consistent so a standalone behavioral run also merges house checks.)
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
- [ ] **Step 5 — stub smoke (WIRING/THREADING ONLY, not gotcha-scoring):** `CUMMATRIX_ROOT=/tmp/hardsmoke python3 matrix.py --app-set hard --models sonnet --trials 1 --regimes cold --stub good --max-apps 2`. NOTE: `_stub_build()` (`harness.py:482-497`) copies the `testdata/good/main.go` notes fixture regardless of app, so it implements NONE of G1–G8 and the house checks deterministically FAIL — that is EXPECTED. This step only proves the flags thread through and result JSONs are produced (the stub path skips the feedback loop and writes a result regardless), NOT that the gotchas score correctly. Expect exit 0 and 2 result files under `/tmp/hardsmoke/results/`.
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
Total ~8 headless builds (6 cold + 2 warm). Model = sonnet for pilot economy (the full run's model is a user decision — see Section 8/9). `--max-apps 2` includes app2 so transfer is actually exercised (deviation from the "1 app" guidance, with rationale — see Open Questions). **Workers (pilot 3 vs full 4, deliberate):** the pilot uses `--workers 3` — fewer lanes keep vault-clone I/O contention low on the short 2-app chain; the full run uses `--workers 4` for the 24-cell 3-app matrix where the extra lane pays off. Not arbitrary.

### Pass/fail signals (pre-registered)

Read from each `${CUMMATRIX_ROOT}/results/*-build.json`: `rounds_to_converge`, `build_cost`, `recall_cost`, `learn.cost`, `build_turns` (`harness.py:882`), `learn.turns` (`harness.py:817`), `recall_s`/`build_s`/`learn_s`, `wall_min`, `converged`, `recall_fired`, `learn.fired`, `rate_limited`, `tokens.cache_read`. **Isolate the house-gotcha count** via `rounds[0]["feat_buckets"]["house"]` — a `"N/8"` string populated at `harness.py:651-655` (`_round_rec`), the same key `beta_table` reads at `aggregate.py:180-182`; parse the numerator. Do NOT use `round1_feature_fails` (`harness.py:869`), a BLENDED native+house count that cannot isolate the gotchas.

- **RED (make-or-break) — PASS iff ALL THREE:** (a) cold median `rounds_to_converge` >= 3; (b) cold median round-1 house fails (numerator of `rounds[0]["feat_buckets"]["house"]`) >= 5/8; (c) cold median per-build `build_cost` >= **$5** (a concrete bar, above the easy-CRUD ~$2–4 anchor). → cold genuinely rediscovers across rounds. **FAIL** (any bar unmet — e.g. cold clears gotchas in <=2 rounds, or median `build_cost` < $5) ⇒ regime too easy; redesign gotchas (add dead-ends / under-determine symptoms), do NOT proceed.
- **Mechanism — PASS iff:** on the warm pair, `recall_fired>=1` and `learn.fired` truthy on app1; app1's vault gained house-gotcha notes (`notes_written>0`, inspect the notes name the sigil/fence/exit-7 rules); AND warm app2 round-1 house fails <= **cold app2 round-1 house fails − 2** (an explicit margin — an undifferentiated 1-note drop does NOT pass). **FAIL** ⇒ learn/recall isn't transferring the gotchas; investigate (learn-capture, recall surfacing, apply) before spending on a full run.
- **Headless-clean — PASS iff:** no `rate_limited` cells; cfg pool is the clean-room template (`build_cfg_template`, warm carries only recall+learn skills); per-build wall-time + $ present in every result JSON; AND the **executable leakage check** passes: on a cold app1 round-1 workdir (under `${CUMMATRIX_ROOT}/ws/*-app1-cold`, or a snapshot taken before the feedback loop mutates it), if the round-1 build compiles, run `python3 score.py <cold-app1-round1-workdir> wardex_spec.json` and read `feat_buckets["house"]` — it MUST be `0/8` (every house gotcha failing in round 1). If it is anything above `0/8`, the regime leaked (ambient `~/.claude/CLAUDE.md`) or is too easy — halt & redesign, do NOT proceed.

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
2. **OP-COST axis:** `axis_c1_recall_s`/`build_s`/`learn_s` (time); `axis_c2_cost_usd` (build) + `recall_cost` + `learn.cost` ($); **`build_turns` (`harness.py:882`) + `learn.turns` (`harness.py:817`) (TURNS — the ask says "lower cost/TURNS", so turns is a first-class reported metric, not folded into time/$)**; chain-summed total op $, wall, and turns. Cold vs warm.
3. **RE-MEASURED FIXED TAX (design-req #3):** current-vintage recall+learn tax = mean(`recall_s`+`learn_s`) and mean(`recall_cost`+`learn.cost`) across warm cells — replaces the stale ~186 s/$1.80. PLUS the per-round **cache-read premium** (design-req #2): warm build rounds re-read recalled context every turn; quantify from `tokens.cache_read` × price on warm build rounds.
4. **NOISE FLOOR (design-req #5):** `aggregate.py:noise_floor` = warm-vs-warm 95% CI half-width from the SAME contrast; `gap_label`/`axis_ci_table` tag each cold-vs-warm gap significant/underpowered. n=8 targets adequate power.

### Win-bar arithmetic (memory-note 95 — report explicitly, do not collapse)

For a net op-win, warm's avoided-rebuild saving on app2/app3 (cold op-cost − warm build-cost) must exceed the re-measured fixed tax + the cache-read premium. Report the subtraction with units and CIs; the sign of `(saving − tax − premium)` is the op-cost verdict.

---

## 6. Analysis & LEDGER Update

### Task 5: Run the aggregator + a shared-gotcha transfer slice

**Files:**
- Run (APP-AGNOSTIC SECTIONS ONLY): `python3 dev/eval/cumulative/aggregate.py --root /tmp/hard-full --out /tmp/hard-full/results-agg.md`. **WARNING (verified):** `aggregate.py` hardcodes `notes`/`links`/`feeds` across MANY sections — `chain_intervention_table:142`, `main:727`, `chain_rows:221-223`, `SEED_APPS`/`PAYBACK_APPS:277-278`, `beta_table:178`, `native_control_table:444`, `full_matrix_tables:600-603`, `escalation_table:680`, `cost_time_table:368`. For the hard app-set these do NOT throw — they silently emit empty/zero rows, so `results-agg.md` LOOKS complete but ~8 of ~15 sections are blank/zero. **Consume ONLY the verified app-agnostic sections:** `axis_ci_table` (`:75-121` — C1/C2 times+$ CI table with noise-floor tags), `noise_floor` (`:60-65`), `token_io_table` (`:497-548`), `notes_written_table` (`:463-494`), `cost_calibration` (`:688-706`). Explicitly DISREGARD every coupled section named above.
- Create: `dev/eval/cumulative/hard_analysis.py` — a STANDALONE (does not touch the live aggregator) that reads `results/*-build.json`, parses `rounds[0]["feat_buckets"]["house"]` per (regime, app), and emits the CAPABILITY + house-bucket + win-bar tables. ALL capability/house/turns analysis flows through this, never the coupled `aggregate.py` sections.

- [ ] Produce ONE labeled table (columns with UNITS, cold vs warm side-by-side + Δ per app1/app2/app3): (1) **CAPABILITY** — round-1 house pass rate (`N/8`) + `converged` rate + `rounds_to_converge` (from `hard_analysis.py`); (2) **OP-COST** — `build_cost`+`recall_cost`+`learn.cost` ($), `build_s`+`recall_s`+`learn_s` (s), and **`build_turns` + `learn.turns` (turns — the ask's "lower TURNS" half)**, cold vs warm (axis CI table + `hard_analysis.py`); (3) re-measured fixed-tax line (s + $) with the per-round cache premium; (4) the win-bar subtraction with CIs.

### Task 6: Rewrite the LEDGER row + ROADMAP

**Files:**
- Modify: `dev/eval/LEDGER.md:39` (`c1-c2-warm-op-negatives`)
- Modify: `docs/ROADMAP.md:97` and `:208` (harder-builds eval → Measured)

**GATE (D9 — the LEDGER/ROADMAP write is a user-facing state mutation, NOT auto-applied):** first present the measured three-axis verdict + the chosen Section-7 verdict token to the user and AWAIT approval; only then edit the files. (The full run is user-gated at the pilot; this is a SECOND, separate gate on committing the verdict to the durable record.)

Rewrite `c1-c2-warm-op-negatives` to KEEP the easy-CRUD refuted finding as historical scope and ADD the harder-regime verdict with fresh vintage (2026-07-17, post-payload-cut). New row content must carry: the three-axis result (capability / op-cost / re-measured tax + cache premium), noise-floor labels + n=8 power, the win-bar arithmetic, and the honest-null disposition if op-cost stays negative. Change the scope clause "underpowered for harder multi-round builds, not 'memory can't pay off'" to "**measured** on the gotcha-concentrated hard regime (n=8/arm): <verdict>." **Cite ONLY the app-agnostic `results-agg.md` sections + `hard_analysis.py`'s tables + the result JSONs as raw data — NEVER the coupled/blank `aggregate.py` sections (Task 5), which emit zeros for the hard app-set and would be false evidence.** Keep the format identical to the existing rows (`| claim | verdict | figure (vintage) | superseded-by | raw data |`). Verdict token per Section 7 (`proven|refuted|unmeasured`). Flip ROADMAP lines 97/208 to "Measured" pointing at the new LEDGER row.

- [ ] Commit (`docs(ledger): harder-builds op-value verdict + vintage; ROADMAP measured`).

### 6.1 Doc-surface enumeration grep (non-waivable) — disposition list

Grep run 2026-07-17 over `docs/ dev/eval/LEDGER.md README.md CLAUDE.md` for `c1-c2-warm-op-negatives|warm-op-negatives|+182s|+$3.08|underpowered for harder|642`. The changing invariant is **the harder-builds eval verdict / #642 status**. Every echo + its disposition (Gate A's docs/diagrams reviewer verifies this AND runs its own independent scan):

| File:line | Current text | Disposition | Reason |
|-----------|--------------|-------------|--------|
| `dev/eval/LEDGER.md:39` | `c1-c2-warm-op-negatives` refuted, "underpowered for harder multi-round builds" | **rewrite** | Keep easy-CRUD scope, ADD harder-regime verdict + 2026-07-17 vintage + re-measured tax (Task 6) |
| `dev/eval/LEDGER.md:38` | `c1-c2-warm-op-mislabeled` superseded (historical) | **keep** | Historical accounting, unaffected by the new measurement |
| `docs/ROADMAP.md:97` | "harder-builds eval (measurement candidate; designed, unrun; no issue #)" | **rewrite** | It now IS #642 and is being run — flip to Measured, point at the new LEDGER row, drop "no issue #" |
| `docs/ROADMAP.md:208` | "Harder-builds eval baseline … designed but never run" (**PROVENANCE — shipped/refuted/dead-ends section**; a backward reference reading "see NEXT band above", NOT the GATED band) | **rewrite** | Update the provenance note to record where #642 landed (e.g. "measured 2026-07-17, verdict → `LEDGER#c1-c2-warm-op-negatives`"); it is already in Provenance, so this is a status update, not a band move |
| `docs/ROADMAP.md:84` | "#642 … self-seeding cold-vs-warm … value-proof spine" (NOW, rank 2) | **update** | Once measured, #642's status/ranking changes (spine delivered); reconcile with the closure in Step 6 |
| `docs/ROADMAP.md:96` | "#646 e2e recency value-proof … \| #642" | **verify/keep** | #646 is the distinct recency proof still blocked on #642's harness; keep unless closure changes the blocker |
| `docs/ROADMAP.md:102` | "#648 tune usefulness-activation … \| #642 AND #646" | **keep** | Dependency edge unchanged by this measurement |

Note: vault notes 95/98/99 also carry the stale verdict but are updated via the closing `/learn` (Step 7), NOT by this doc-scrub. Gate A's docs/diagrams reviewer confirmed `dev/eval/traps/RESULTS.md`, the architecture docs, `FEATURES`, `GLOSSARY`, and `README` carry no stale echo of this verdict. (There is no `cumulative/EXPERIMENT-LOG.md` — that file does not exist.)

---

## 7. Pre-Registered Acceptable Outcomes (honest null — write BEFORE the full run)

Per memory-note 98, all four are legitimate; none is a failure to be engineered away:

1. **Capability-win + op-cost-win** (beyond noise): memory nets cheaper/faster AND more-correct. **→ LEDGER verdict token: `proven`** (scoped "harder regime"); the hard build-eval becomes a valid value harness.
2. **Capability-win + op-cost neutral/negative:** memory is more-correct on the gotchas but avoided-rebuild < tax + cache-premium. **→ LEDGER verdict token: `refuted`** (op-cost dominates the row's verdict) **+ append the clause "capability win on idiosyncratic content (note 99)"**; conclusion: memory's value is capability/behavioral, and op-value measurement points to **real long-session work, not cheap build evals**. (The pre-registered honest null.)
3. **No capability-win, gap below noise:** can't distinguish (memory-note: gap below noise = underpowered, NOT a tie). **→ LEDGER verdict token: `unmeasured`** (labeled underpowered at this n); report n and either recommend more power or accept with that label.
4. **Regime-invalid** (cold accrued no waste even here — should be caught at the pilot): the build-eval regime cannot manufacture memory op-value → redirect value measurement to long-session/behavioral work. **→ LEDGER verdict token: `refuted`** if the regime ran but memory still didn't pay; **`unmeasured`** if the regime premise itself failed (cold didn't accrue waste, so the contrast never tested memory).

The verdict token written to the LEDGER MUST be SELECTED from this list to match the observed axes — chosen, not narrated post-hoc. Valid tokens: **`proven` | `refuted` | `unmeasured`**.

---

## 8. Cost Estimate

**Anchors (cite, with vintage):**
- Easy 2-round build: **$2–4/build** (memory-note 95, measured).
- Easy-CRUD warm net op **+$3.08** over cold; recall+learn tax **~$1.80 / ~186 s** — PRE-PAYLOAD-CUT (2026-06-25), to be re-measured (LEDGER `c1-c2-warm-op-negatives`).

**Harder-build projection (UNVERIFIED — the pilot calibrates it):** harder builds target >=3–4 cold rounds vs ~1–2 easy, so per-build $ scales up. Rough projection: cold hard build ~**$4–8**, warm ~**$5–9** (tax + cache premium). Opus is materially pricier than sonnet — treat any pre-pilot number as a projection, replace with the pilot's measured per-build $ before the full-run go/no-go.

**Pilot:** ~8 builds (6 cold + 2 warm) × ~$3–6 ⇒ **~$25–50**, wall ~**1–2 h** (3 pilot workers; chain dependency serializes the warm pair). NOTE: the Task-4 stub-smoke is ZERO-LLM-spend and CANNOT anchor $ — the **first 1–2 real cold pilot builds are the cost anchor**; watch `matrix.py`'s running `spent $` tally and update the full-run projection from them. *Projection; report the actual spent tally honestly.*

**Full run [PILOT-CALIBRATED: TBD after the recall+learn re-measure]:** 48 builds × ~$4–8 ⇒ **~$190–380** (sonnet) or **~$300–600** (opus) — UNVERIFIED projections extrapolated off the STALE $1.80/186 s tax vintage; do NOT treat as firm. Wall: 48 builds / 4 workers × ~15–30 min/build ⇒ **~3–6 h**. **Replace the $ range with the pilot's measured per-build cost + re-measured tax before presenting the go/no-go.** No interrupting spend cap (`--budget 0`), but confirm the estimate up front.

---

## 9. Risks & Open Questions

**Risks:**
- **Gotchas too easy / symptoms too informative** — cold clears them in one feedback round → reproduces the null. Mitigation: 8 concentrated gotchas + >=3 dead-ends + under-determined symptoms; the pilot RED gate is the check.
- **Escalation can't rescue cold on a gotcha** — `STALL_PATIENCE=3` halts and the cell is flagged `did_not_complete`, truncating its op-cost and muddying the op-cost axis (though it cleanly feeds the CAPABILITY axis). Mitigation: ensure every gotcha's `_spec_check_detail` escalation reveals a concrete example by round 3; consider raising `--max-rounds` for the hard regime; treat halted cells per the capability axis, not op-cost.
- **Transfer doesn't fire** — learn captures the gotchas but recall doesn't surface them or the build doesn't apply them. Mitigation: pilot mechanism gate; the learn-capture fix (commit e07bde3d) already improved requirement capture.
- **`aggregate.py` app-name coupling (worse than a one-liner implies)** — the `notes`/`links`/`feeds` names are hardcoded across MANY sections: `chain_intervention_table:142`, `main:727`, `chain_rows:221-223`, `SEED_APPS`/`PAYBACK_APPS:277-278`, `beta_table:178`, `native_control_table:444`, `full_matrix_tables:600-603`, `escalation_table:680`, `cost_time_table:368`. None THROW for the hard app-set — they silently emit empty/zero rows, so `results-agg.md` looks complete but ~8/15 sections are blank. Mitigation: consume ONLY the verified app-agnostic sections (`axis_ci_table`, `noise_floor`, `token_io_table`, `notes_written_table`, `cost_calibration`) and drive ALL capability/house/turns analysis through `hard_analysis.py` (Task 5); Task 6 must never cite a coupled/blank section as evidence.
- **`claude -p` + `CLAUDE_CONFIG_DIR` redirection** — uncertainty whether the user-global `~/.claude/CLAUDE.md` still loads under the redirected cfg. The fictional gotchas are robust either way (they cannot leak), which is exactly why fictional was chosen; the pilot leakage check confirms it empirically.

**Open questions — RESOLVED by the orchestrator (2026-07-17):**
1. **Full-run model:** RESOLVED — **pilot on sonnet; full-run model is Joe's call at the post-pilot go/no-go** (his chosen scope was "pilot, report, then decide", so the model decision lands naturally at that report, informed by the pilot's measured per-build cost). Plan default for the full-run remains opus for c1-c2 comparability, pending that decision.
2. **Pilot scope deviation:** RESOLVED — **accept the 2-app pilot** (`--max-apps 2`, one app1→app2 warm pair). Catching a no-transfer failure for ~$25–50 now beats discovering it after 48 builds; the marginal cost is trivial.
3. **Design A vs B:** RESOLVED — **Design A** (gotchas as behavioral checks). Minimal, YAGNI, no `archscore`/`score.py`/`harness`-core risk. Revisit B only if a later need forces the native transfer tables to carry the signal.

---

## Self-Review

- **Spec coverage:** design (S2), harness wiring w/ real paths+line numbers (S3), pilot RED gate + STOP (S4), full run n=8 + 3 axes + re-measured tax + noise floor (S5), analysis + LEDGER/ROADMAP update (S6), pre-registered nulls (S7), cost anchors+projection (S8), risks+open-qs (S9) — all present.
- **No placeholders:** Task 2 Step 1 now carries a COMPLETE real-JSON authoring template (the full G1 `sigil_no_reuse` entry) plus the exact JSON encoding rules (`\u001f`/`\u001e`, literal `⟐`/`·`); G2–G8 copy that shape from the fully-specified S2.3 rule table (id format, separators, exit code, suffix codes, joiner, search scoping, ledger format).
- **Type/name consistency (self-checked across §4/§5/§6/§7):** the house count is read as `rounds[0]["feat_buckets"]["house"]` (`"N/8"`) EVERYWHERE (§4 RED bar, §4 leakage check, §5 capability, §6 Task 5) — never `round1_feature_fails`; `build_turns` + `learn.turns` reported in §4 field-list, §5 OP-COST, §6 Task 5; the app-agnostic aggregate section set (`axis_ci_table`/`noise_floor`/`token_io_table`/`notes_written_table`/`cost_calibration`) named identically in §6 Task 5 and §9; the verdict tokens `proven|refuted|unmeasured` match across §6 Task 6 and §7; `load_spec` is defined once in `score.py` and referenced consistently (§Task 2/3, leakage check); `--app-set`/`--max-apps`/`APP_SETS`/`house_checks_file`/`bucket:"house"`/`exit:N` consistent.
