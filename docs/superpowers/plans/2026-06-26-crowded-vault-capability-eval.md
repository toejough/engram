# Crowded-vault capability eval — implementation plan (rev 2, post Gate A)

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development or executing-plans. Steps use `- [ ]` checkboxes.

**Goal:** Build + run an eval that tests whether memory's 4 verified wins (C3/C4i/C5/C6) survive a realistic crowded vault (variants of the real vault + links), using a free retrieval-precision sweep to guide where the expensive applied check spends.

**Architecture:** New `dev/eval/traps/` modules — `crowd.py` (variant generator from the real vault), `retrieval_probe.py` (Tier-1 surfaced/rank via the real multi-phrase `engram query`), a uniform `--crowd N` flag on the 4 warm harnesses, and `crowded_gate.py` (orchestrator). Reuse `gate_verdict`, `seed_c3`.

**Two vault models (Gate-A finding 1/2 — load-bearing):**
- **Cosine axes — C3, C4i, C6:** vault-of-notes; crowd = variant **notes** seeded via `engram learn`; the load-bearing note(s) compete on cosine, so Tier-1 (retrieval precision) is meaningful and crowding can bury them.
- **Recency axis — C5:** a **chunk index** whose target (`R-decision.md`) surfaces by being the *newest* chunk, not by cosine. Crowd = variant **chunks** seeded via `engram ingest --markdown` BEFORE R (R stays newest). **C5's Tier-1 is invariant** (R always surfaces) → skip the sweep for C5; its crowding effect is measured only in Tier-2 (honored rate) at a fixed heavy crowd.

## Global Constraints

- **NEVER seed into or mutate the real vault.** `crowd.load_real_notes` reads `$XDG_DATA_HOME/engram/vault` (or `ENGRAM_VAULT_PATH`) **read-only**; seeding goes only to temp dirs. A test asserts the source is unwritten (file-set + mtime unchanged).
- Tests via `pytest`, house style (plain `def test_*` + `assert`; `import pytest` only for `pytest.raises`).
- **Fail loud** on missing source vault / harness error / missing output JSON.
- **Tier 1 is free (local `engram query`/`ingest`); Tier 2 is the only LLM spend** (~$20–40).
- Determinism: `crowd.py` uses `random.Random(seed)` with **fixed `SEED = 7`** everywhere; a crowd is reproducible.
- New files importing `recency_probe` (in `dev/eval/cumulative/`) must add `sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "cumulative"))` as `c5.py` does (Gate-A finding 3).
- Confound guards (note 96): pair-before-pool; a gap below the **warm-vs-warm** noise floor is underpowered, not a tie; the C3 scorer greps the **code form** not note names; report contamination (degraded build / judge error) separately, never as a miss.
- `engram learn --relation` is lenient (no error on a not-yet-existing target, `learn.go:596`); seed all variants in one loop so cross-variant links resolve by the end (Gate-A finding 4 — acceptable).

---

### Task 1: `crowd.py` — real-vault variant generator

**Files:** Create `dev/eval/traps/crowd.py`; Test `dev/eval/traps/test_crowd.py`

**Concrete variant dict schema** (every variant is exactly this shape):
```python
{"slug": "crowd-<src_luhmann>-<i>", "src_slug": "<src basename>", "type": "fact"|"feedback",
 "situation": "<copied>", "fields": {"subject":..,"predicate":..,"object":..}      # fact
                          | {"behavior":..,"impact":..,"action":..},                # feedback
 "links": ["crowd-<otherluhmann>-<i>", ...], "newer": bool}
```

**Interfaces:**
- `real_vault() -> str` — `os.environ.get("ENGRAM_VAULT_PATH") or <XDG default>/engram/vault`.
- `load_real_notes(vault_path) -> list[dict]` — read each `*.md`: parse YAML frontmatter (`type`, `situation`, `luhmann`, fact `subject/predicate/object` or feedback `behavior/impact/action`) and the `Related to:` body bullets (`- [[basename]] — …`) into `links` (source basenames). Read-only.
- `make_variants(notes, n, seed=7, vocab_terms=(), vocab_frac=0.0, recency_frac=0.0) -> list[dict]` — **pure/deterministic**. Algorithm:
  1. `rng = random.Random(seed)`. If `vocab_terms`, partition `notes` into `match` (any term in `subject+object`/`behavior+action`, case-insensitive) and `rest`.
  2. Build an ordered source list of length `n`: take `ceil(vocab_frac*n)` items by cycling `match` (if any), the remainder by cycling `rest` then all `notes`; `rng.shuffle` the result.
  3. For the i-th source note `s`, emit a variant: `slug=f"crowd-{s['luhmann']}-{i}"`, `src_slug=s basename`, copy `type/situation/fields` **verbatim** (the "variant" is a re-slugged real note — realistic by construction; NO text paraphrase, which can't be done deterministically without an LLM), `newer = (i < recency_frac*n)`.
  4. **Link re-pointing:** after all variants exist, for each variant of source `s`, for each `s.link → target_basename`: if any variant has `src_slug == target_basename`, point to the variant **at the same index i** (`crowd-{target_luhmann}-{i}`); if none at index i, wrap to `i % count_of_target_variants`; if the target isn't in the crowd at all, drop that link.
- `seed_into(vault_path, variants)` — **vault axes**: assert `os.path.realpath(vault_path) != os.path.realpath(real_vault())` (raise `RuntimeError` otherwise); `engram learn fact|feedback` per variant (`--slug`, `--position sibling`, `--source "crowd"`, `--situation`, the type's three fields, one `--relation "<link>|crowd"` per link), `ENGRAM_VAULT_PATH=vault_path`; raise on non-zero exit.
- `seed_into_chunks(chunks_dir, variants)` — **C5 only** (Gate-A finding 1): write each variant to a temp `.md` file (title + the field text) and `engram ingest --markdown <file> --chunks-dir <chunks_dir>`; raise on non-zero. (Caller seeds these BEFORE R so R stays newest.)

- [ ] **Step 1: failing tests**
```python
"""TDD for the crowd generator: deterministic variants, link re-pointing, knobs, read-only source."""
import os, sys
sys.path.insert(0, os.path.dirname(__file__))
import crowd

SRC = [{"slug": "77.x", "luhmann": "77", "type": "fact", "situation": "s",
        "fields": {"subject": "http requests in Go", "predicate": "use", "object": "NewRequestWithContext"},
        "links": ["91.y"]},
       {"slug": "91.y", "luhmann": "91", "type": "fact", "situation": "s2",
        "fields": {"subject": "logging", "predicate": "use", "object": "slog"}, "links": []}]

def test_make_variants_count_unique_deterministic():
    a = crowd.make_variants(SRC, n=5, seed=7); b = crowd.make_variants(SRC, n=5, seed=7)
    assert len(a) == 5 and len({v["slug"] for v in a}) == 5
    assert [v["slug"] for v in a] == [v["slug"] for v in b]

def test_links_repoint_to_sibling_variants_or_drop():
    v = crowd.make_variants(SRC, n=4, seed=7)
    slugs = {x["slug"] for x in v}
    for x in v:
        for L in x["links"]:
            assert L in slugs                      # never dangling-outside-crowd

def test_vocab_knob_biases_toward_matching_notes():
    v = crowd.make_variants(SRC, n=10, seed=7, vocab_terms=["http"], vocab_frac=0.5)
    hits = sum(1 for x in v if "http" in x["fields"].get("subject", "").lower())
    assert hits >= 4

def test_recency_knob_marks_some_newer():
    v = crowd.make_variants(SRC, n=10, seed=7, recency_frac=0.5)
    assert 1 <= sum(1 for x in v if x["newer"]) <= 9

def test_seed_into_refuses_real_vault(tmp_path):
    import pytest
    with pytest.raises(RuntimeError):
        crowd.seed_into(crowd.real_vault(), [])
```
- [ ] **Step 2: run → FAIL** — `cd dev/eval/traps && python3 -m pytest test_crowd.py -v`
- [ ] **Step 3: implement** per the interfaces above.
- [ ] **Step 4: run → PASS.**
- [ ] **Step 5: real read-only check** — snapshot `sorted(os.listdir(real_vault()))` + total size; run `load_real_notes(real_vault())` (expect ~100); assert the snapshot is unchanged. Commit `feat(eval): crowd.py — real-vault variant generator (notes + chunks, link-preserving)`

### Task 2: `retrieval_probe.py` — Tier-1 surfaced/rank (free, multi-target)

**Files:** Create `dev/eval/traps/retrieval_probe.py`; Test `dev/eval/traps/test_crowd.py` (extend)

**Concrete `AXIS_PHRASES`** (the phrases `/recall` would generate for each axis's task — Tier-1 runs the real multi-phrase `engram query`, the retrieval call `/recall` makes; not the full skill):
```python
AXIS_PHRASES = {
  "C3": ["building a command-line app in Go", "making an HTTP request in Go", "wrapping errors in Go",
         "writing parallel Go tests", "terminal color output and NO_COLOR", "guarding a slice index in Go",
         "idiomatic Go error handling", "Go CLI architecture conventions", "Go context cancellation",
         "Go code quality standards"],
  "C4i": ["error marker convention in cfgload", "prefixing returned errors in Go", "cfgload codebase conventions",
          "error wrapping standard update", "ERR-CFG error prefix", "fmt.Errorf marker token",
          "exported function error format", "Go error return convention", "config loader error handling",
          "superseded error-marker convention"],
  "C6": ["<case A framing phrases>", "<case B framing phrases>"],   # from reasoning_recall_eval.CASES tasks
}
AXIS_TARGETS = {  # the load-bearing note basename(s) per axis (multi-target for C3, C6)
  "C3": ["<5 seed_c3 convention-note basenames>"], "C4i": ["<errcfg-supersedes-e7 basename>"],
  "C6": ["<both premise-note basenames per case>"],   # C5 omitted — Tier-1-invariant
}
```
(Task implementer: populate `AXIS_TARGETS` from `seed_c3.C3_NOTES` slugs, `c4_idio` `errcfg-supersedes-e7`, and `reasoning_recall_eval.CASES[*]["notes"]` slugs; derive C6 phrases from each case's `task`.)

**Interfaces:**
- `rank_in_payload(payload, target_basename) -> {"surfaced": bool, "rank": int|None}` — pure: 1-based index of the first `items[i].path` whose basename matches.
- `probe(vault_path, axis) -> {"targets": {basename: {surfaced,rank}}, "all_surfaced": bool, "worst_rank": int|None}` — run `engram query` with `AXIS_PHRASES[axis]` (10 `--phrase`) against `vault_path`, parse `items[]`, call `rank_in_payload` for each `AXIS_TARGETS[axis]`. `all_surfaced` = every target surfaced; `worst_rank` = max rank (None if any absent). **For C3/C6 a break fires if ANY target drops (Gate-A finding A).**

- [ ] **Step 1: failing tests**
```python
import retrieval_probe as rp
def test_rank_in_payload_found_and_absent():
    p = {"items": [{"path": "v/other.md"}, {"path": "v/target.md"}]}
    assert rp.rank_in_payload(p, "target") == {"surfaced": True, "rank": 2}
    assert rp.rank_in_payload({"items": [{"path": "v/x.md"}]}, "target") == {"surfaced": False, "rank": None}
```
- [ ] **Step 2 → FAIL. Step 3: implement** (`probe` shells `engram query`, parses YAML/JSON via the same loader the gate uses). **Step 4 → PASS. Step 5: commit** `feat(eval): retrieval_probe.py — multi-target Tier-1 surfaced/rank`

### Task 3: uniform `--crowd N` injection on the 4 warm harnesses

**Files:** Modify `dev/eval/traps/{wrun.py,c4_idio.py,c5.py,seed_c5.py,c6_clean.py}`

- [ ] **Step 1:** Add `--crowd N` (default 0). After each harness's normal seed, if `N>0`, build `variants = crowd.make_variants(crowd.load_real_notes(crowd.real_vault()), N, seed=7, vocab_terms=<axis terms>, recency_frac=0.3)` and inject:
  - **C3** (`wrun.py`): after the caller seeds `seed_c3` into `--vault`, `crowd.seed_into(args.vault, variants)`. vocab_terms=`["http","error","test","color","Go"]`.
  - **C4i** (`c4_idio.py`): in `main()` after `seed_vaults()`, `crowd.seed_into(VAULTS["warm-XXp"], variants)`. vocab_terms=`["error","cfgload","marker","Go"]`.
  - **C5** (`seed_c5.py`): add `--crowd N` — generate N variant `.md` files and `engram ingest --markdown … --chunks-dir SEED_CHUNKS` for each **before** ingesting R (R stays newest). Use `crowd.seed_into_chunks`. (`c5.py` passes `--crowd` through to its `seed_c5` invocation, or the gate seeds first.)
  - **C6** (`c6_clean.py`): give `warm_one(case, cfg, judge_cfg, idx, model="opus", crowd_variants=None)` the new param; after `for n in spec["notes"]: rr._learn(vault, *n)`, if `crowd_variants`: `crowd.seed_into(vault, crowd_variants)`. Thread it: `ex.submit(warm_one, c, cfg, judge_cfg, i, a.model, crowd_variants)` (Gate-A finding 5). vocab_terms=`["error","reasoning","memory"]`.
- [ ] **Step 2: verify (no real spend).** The harnesses have **no `--stub`** (Gate-A/docs finding) — so validate the injection by **unit/AST** only here: `python3 -c "import ast; [ast.parse(open(f).read()) for f in ['wrun.py','c4_idio.py','c5.py','seed_c5.py','c6_clean.py']]"`. The real injection is exercised for free in Task 5 Step 1 (Tier-1 builds crowded vaults end-to-end). Do NOT run the harnesses for real here.
- [ ] **Step 3: commit** `feat(eval): --crowd N injecting real-vault variants into each warm harness`

### Task 4: `crowded_gate.py` — orchestrator

**Files:** Create `dev/eval/traps/crowded_gate.py`; Test `dev/eval/traps/test_crowd.py` (extend)

**Interfaces:**
- `RANK_THRESHOLD = 10` (a target ranked worse than top-10 counts as "buried"). `LEVELS = [0,10,30,50,100,200,400]`. `HEAVY_FALLBACK = 200`.
- `break_point(sweep, rank_threshold=RANK_THRESHOLD) -> int|None` — pure: smallest `n` in `sweep` where `all_surfaced` is False OR `worst_rank > rank_threshold`; `None` if never (→ caller uses `HEAVY_FALLBACK`).
- `degradation(crowded_axis, toy_axis) -> {"delta": int, "note": str}` — pure: `passed/valid` delta; if `abs(delta)` ≤ 1 (n=5 warm-vs-warm noise) note "within noise — underpowered".
- `tier1_sweep(axis) -> list[{n,all_surfaced,worst_rank}]` — for each `n` in LEVELS (skip C5): mkdtemp; seed the axis base note(s) (`seed_c3`/`c4_idio.seed_vaults`/`c6` case notes) into it; `crowd.seed_into(temp, make_variants(...,n))`; `retrieval_probe.probe(temp, axis)`; teardown. (C5 returns a fixed `[{n:HEAVY_FALLBACK, note:"recency-invariant"}]`.)
- `main(--tier1-only?)` — Tier-1: `tier1_sweep` each cosine axis → `break_point` → chosen `B` (or HEAVY_FALLBACK). If `--tier1-only`, print the curves + chosen B and exit (free). Tier-2: run each axis warm harness with `--crowd B` and `--crowd <heavier=min(2*B,400)>` (C5: `--crowd 200` only), `gate_verdict.normalize`+`axis_verdict`, compare crowded-warm vs the toy baseline (`gate.py --tier smoke` numbers or re-run crowd=0) and report `beats_cold` (cold pass≈0 for these traps). Emit a labeled table (`axis | break_n | crowd | crowded_pass | toy_pass | delta | verdict`) + `crowded-verdict.json`.

- [ ] **Step 1: failing tests**
```python
import crowded_gate as cg
def test_break_point_first_buried():
    s = [{"n":0,"all_surfaced":True,"worst_rank":1},{"n":50,"all_surfaced":True,"worst_rank":4},
         {"n":200,"all_surfaced":False,"worst_rank":None}]
    assert cg.break_point(s) == 200
def test_break_point_rank_threshold():
    s = [{"n":0,"all_surfaced":True,"worst_rank":2},{"n":100,"all_surfaced":True,"worst_rank":14}]
    assert cg.break_point(s) == 100               # worst_rank 14 > 10
def test_break_point_none_when_robust():
    assert cg.break_point([{"n":0,"all_surfaced":True,"worst_rank":1},
                           {"n":400,"all_surfaced":True,"worst_rank":3}]) is None
def test_degradation_within_noise_flagged():
    d = cg.degradation({"passed":5,"valid":5}, {"passed":5,"valid":5}); assert d["delta"] == 0
    d2 = cg.degradation({"passed":4,"valid":5}, {"passed":5,"valid":5}); assert "noise" in d2["note"]
```
- [ ] **Step 2 → FAIL. Step 3: implement** (fail loud; add `crowded-verdict.json` to `.gitignore`). **Step 4 → PASS. Step 5: commit** `feat(eval): crowded_gate.py — Tier-1 sweep + retrieval-guided Tier-2 applied check`

### Task 5: real Tier-1 sweep (free) → realism gate → Tier-2 run (LLM)

- [ ] **Step 1: Tier-1 sweep (free, no LLM).** `python3 crowded_gate.py --tier1-only`. Read per-axis surfaced/rank curves + chosen B. **Realism gate (define the threshold up front):** the crowd is a valid competitor iff, at n≥50, **≥2 crowd variants rank in the top 5** of an axis's query (i.e. the crowd is actually competing with the target, not off-topic noise). Inspect the `engram query` items for a crowd-slug presence in the top 5; if not met, re-run Task-5 Step 1 with higher `vocab_frac` (0.5→0.8) before any spend.
- [ ] **Step 2: Tier-2 (real LLM).** Print the estimated cost (~$0.45/trial × trial-count ≈ $20–40) — **do NOT gate; scope is pre-approved** (Gate-A finding B) — and immediately run `python3 crowded_gate.py`.
- [ ] **Step 3: Verify + report.** Confirm real harnesses ran (non-zero costs), table + `crowded-verdict.json` produced. Report per axis: does crowded-warm still pass + beat cold, and the toy-vs-crowded delta — applying the confound guards (pair-before-pool; delta ≤1 of 5 = within noise/underpowered, not a tie; contamination separate) before any "holds"/"breaks" claim.

---

## Self-Review (vs Gate-A findings)
- Clarity: perturbation = re-slug copies (no LLM paraphrase) ✓; link-repoint algorithm w/ index-wrap ✓; concrete variant schema ✓; AXIS_PHRASES populated (C6 from CASES) ✓; SEED=7 fixed ✓; read-only assertion test ✓; recency test ✓; injection points exact ✓; probe vs rank_in_payload split ✓.
- ask: C6 multi-target (AXIS_TARGETS list, break if ANY drops) ✓; Tier-1 temp-vault seeding loop described (tier1_sweep) ✓; "print don't gate" cost ✓; roadmap/spec wording aligned (Tier-1 = the real multi-phrase query /recall makes) ✓.
- docs: no --stub → Task 3 verify is AST-only, real injection via free Tier-1 ✓; realism threshold defined (≥2 crowd variants in top-5 @ n≥50) ✓; recency unit test ✓.
- code: C5 chunk-based `seed_into_chunks` + excluded from Tier-1 (recency-invariant), Tier-2 @ crowd=200 ✓; recency_probe sys.path noted (only if used — C5 Tier-1 dropped, so not needed) ✓; c6 warm_one crowd_variants threaded ✓; lenient --relation acceptable (seed-all-then-resolve) ✓; gate_verdict.normalize unaffected ✓.
