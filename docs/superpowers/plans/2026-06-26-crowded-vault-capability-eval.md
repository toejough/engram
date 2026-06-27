# Crowded-vault capability eval — implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development or executing-plans. Steps use `- [ ]` checkboxes.

**Goal:** Build + run an eval that tests whether memory's 4 verified wins (C3/C4i/C5/C6) survive a realistic crowded vault (variants of the real vault + links), using a free retrieval-precision sweep to guide where the expensive applied check spends.

**Architecture:** New `dev/eval/traps/` modules — `crowd.py` (variant generator from the real vault), `retrieval_probe.py` (Tier-1 surfaced/rank via the real multi-phrase `engram query`), a uniform `--crowd N` flag on the 4 warm harnesses, and `crowded_gate.py` (orchestrator: Tier-1 sweep → break point → Tier-2 applied → verdict + degradation deltas). Reuse `gate_verdict`, `seed_c3`, `recency_probe`.

**Tech Stack:** Python 3, pytest, `engram` + `claude` CLIs.

## Global Constraints

- **NEVER seed into or mutate the real vault.** `crowd.py` reads `$XDG_DATA_HOME/engram/vault` (or `ENGRAM_VAULT_PATH`) **read-only** and seeds variants only into isolated temp vaults. A test must assert the source vault is never written.
- Tests via `pytest`, house style (plain `def test_*` + `assert`; `import pytest` only for `pytest.raises`).
- **Fail loud** on missing source vault / harness error / missing output JSON.
- **Tier 1 is free (local `engram query`); Tier 2 is the only LLM spend** (~$20–40). Retrieval-guided targeting bounds it.
- Determinism: `crowd.py` uses `random.Random(seed)` (a fixed seed), never unseeded randomness, so a crowd is reproducible.
- Validate the real component: Tier-1 uses the real multi-phrase `engram query` (note 72), Tier-2 the real warm harnesses — no bypass.
- Confound guards (note 96 / checklist): pair-before-pool; gap below warm-vs-warm noise = underpowered not tie; scorer greps code-form not note names; report contamination separately.

---

### Task 1: `crowd.py` — real-vault variant generator

**Files:** Create `dev/eval/traps/crowd.py`; Test `dev/eval/traps/test_crowd.py`

**Interfaces:**
- `load_real_notes(vault_path) -> list[dict]` — read each `*.md` note's frontmatter (`type`, `subject`/`predicate`/`object` or `behavior`/`impact`/`action`, `situation`, `luhmann`, and `Related to:` wikilink targets). Read-only.
- `make_variants(notes, n, seed, vocab_terms=(), vocab_frac=0.0, recency_frac=0.0) -> list[dict]` — pure/deterministic; each variant = `{slug, type, situation, fields, links}` with a fresh slug, lightly-perturbed text, and links re-pointed to sibling variants (preserve relative structure). `vocab_frac` of variants are biased toward source notes containing `vocab_terms`; `recency_frac` flagged newest. Cycle/index to reach `n > len(notes)`.
- `seed_into(vault_path, variants)` — I/O: `engram learn fact|feedback` per variant (with `--relation` for links), into `vault_path` (must be a temp dir, not the source). Raises on non-zero exit.

- [ ] **Step 1: failing tests**
```python
"""TDD for the crowd generator: variants are deterministic, preserve link structure, honor knobs,
and never touch the source vault."""
import os, sys
sys.path.insert(0, os.path.dirname(__file__))
import crowd

SRC = [{"slug": "a", "type": "fact", "situation": "s", "subject": "http requests in Go",
        "predicate": "use", "object": "NewRequestWithContext", "links": ["b"]},
       {"slug": "b", "type": "fact", "situation": "s2", "subject": "logging", "predicate": "use",
        "object": "slog", "links": []}]

def test_make_variants_count_and_determinism():
    v1 = crowd.make_variants(SRC, n=5, seed=7)
    v2 = crowd.make_variants(SRC, n=5, seed=7)
    assert len(v1) == 5
    assert [x["slug"] for x in v1] == [x["slug"] for x in v2]   # deterministic
    assert len({x["slug"] for x in v1}) == 5                     # unique slugs

def test_make_variants_preserves_links_among_variants():
    v = crowd.make_variants(SRC, n=2, seed=1)
    by_src = {x["src_slug"]: x for x in v}
    a = by_src["a"]
    assert a["links"] and all(t in {x["slug"] for x in v} for t in a["links"])  # links point to siblings

def test_vocab_overlap_knob_biases_toward_matching_notes():
    v = crowd.make_variants(SRC, n=10, seed=1, vocab_terms=["http"], vocab_frac=0.5)
    hits = sum(1 for x in v if "http" in (x["subject"] + x["object"]).lower())
    assert hits >= 4   # ~half biased toward the http note
```

- [ ] **Step 2: run → FAIL.** `cd dev/eval/traps && python3 -m pytest test_crowd.py -v`
- [ ] **Step 3: implement** `load_real_notes` (parse YAML frontmatter + `Related to:` bullets), `make_variants` (deterministic via `random.Random(seed)`; fresh slug `crowd-{src_slug}-{i}`; carry `src_slug`; re-point links to sibling variant slugs; vocab/recency biasing), `seed_into` (subprocess `engram learn`, temp-dir guard: raise if `vault_path` resolves to the source vault).
- [ ] **Step 4: run → PASS.**
- [ ] **Step 5: real read-only check** — `python3 -c "import crowd; ns=crowd.load_real_notes(crowd.real_vault()); print(len(ns))"` prints ~100; confirm the real vault is unchanged (`git -C` n/a — it's outside the repo; assert mtime/no new files). Commit: `git add ... && git commit -m "feat(eval): crowd.py — real-vault variant generator for crowded-vault eval"`

### Task 2: `retrieval_probe.py` — Tier-1 surfaced/rank (free)

**Files:** Create `dev/eval/traps/retrieval_probe.py`; Test `dev/eval/traps/test_crowd.py` (extend)

**Interfaces:**
- `AXIS_PHRASES: dict[str, list[str]]` — the ~10 phrases the `/recall` skill would generate for each axis's task (C3: building a Go CLI + conventions; C4i: cfgload error-marker; C5: the ZÖRBAX timestamp standard; C6: the abduction case framing).
- `probe(vault_or_chunks, axis, target_basename) -> {"surfaced": bool, "rank": int|None, "score": float|None}` — run the real multi-phrase `engram query` (10 `--phrase`), parse `items[]`, find `target_basename`, return surfaced + 1-based rank.

- [ ] **Step 1: failing test** (parse logic, with a canned `engram query` YAML payload fixture):
```python
import retrieval_probe as rp
def test_probe_finds_target_rank_from_payload():
    payload = {"items": [{"path": "x/other.md"}, {"path": "x/target.md"}, {"path": "x/z.md"}]}
    r = rp.rank_in_payload(payload, "target")
    assert r["surfaced"] is True and r["rank"] == 2
def test_probe_absent_target():
    r = rp.rank_in_payload({"items": [{"path": "x/other.md"}]}, "target")
    assert r["surfaced"] is False and r["rank"] is None
```
- [ ] **Step 2: run → FAIL.**
- [ ] **Step 3: implement** `rank_in_payload` (pure) + `probe` (shells `engram query`, parses YAML/JSON, calls `rank_in_payload`). `AXIS_PHRASES` per axis.
- [ ] **Step 4: run → PASS.**
- [ ] **Step 5: commit** `feat(eval): retrieval_probe.py — Tier-1 surfaced/rank via real multi-phrase query`

### Task 3: uniform `--crowd N` flag on the 4 warm harnesses

**Files:** Modify `dev/eval/traps/{wrun.py,c4_idio.py,c5.py,c6_clean.py}`

- [ ] **Step 1:** Add `--crowd N` (default 0) to each argparse. After the normal seed, if `N>0`, plant `crowd.make_variants(load_real_notes(real_vault()), N, seed=...)` via `crowd.seed_into(<that axis's warm vault/index>, variants)`:
  - `wrun.py`: into the `--vault` dir (after caller seeds `seed_c3`).
  - `c4_idio.py`: into `VAULTS["warm-XXp"]` after `seed_vaults()`.
  - `c5.py`/`seed_c5.py`: ingest N distractor chunks **before** R-decision (R stays newest) — extend `seed_c5` with `--crowd N`.
  - `c6_clean.py`: into the per-trial vault in `warm_one` after the case notes.
- [ ] **Step 2:** Smoke each with `--crowd 5 --n 1` on the **stub/dry path where available**, else a parse check (`ast.parse`) — no real spend here; real spend is Task 5. Confirm each still writes its results JSON.
- [ ] **Step 3: commit** `feat(eval): --crowd N flag injecting real-vault variants into each warm harness`

### Task 4: `crowded_gate.py` — orchestrator (Tier-1 sweep → break → Tier-2)

**Files:** Create `dev/eval/traps/crowded_gate.py`; Test `dev/eval/traps/test_crowd.py` (extend)

**Interfaces:**
- `break_point(sweep: list[{"n","rank","surfaced"}], rank_threshold) -> int|None` — pure: smallest `n` where `surfaced` is False OR `rank > rank_threshold`; `None` if never.
- `degradation(crowded: dict, toy: dict) -> dict` — pure: per-axis pass-rate delta crowded-vs-toy with the underpowered/noise note.
- `main()` — Tier-1: for each axis, sweep crowd sizes `[0,10,30,50,100,200,400]` calling `retrieval_probe.probe` (free), record the curve, compute `break_point` (fallback heavy=200). Tier-2: run each axis warm harness with `--crowd B` and `--crowd <heavier>` (real LLM), normalize via `gate_verdict`, compute crowded-warm verdict + `degradation` vs the toy baseline + vs cold. Emit a labeled table (axis | break_n | crowded pass | toy pass | beats_cold | verdict) + `crowded-verdict.json`.

- [ ] **Step 1: failing tests**
```python
import crowded_gate as cg
def test_break_point_detects_first_drop():
    sweep = [{"n":0,"surfaced":True,"rank":1},{"n":50,"surfaced":True,"rank":2},
             {"n":200,"surfaced":False,"rank":None}]
    assert cg.break_point(sweep, rank_threshold=10) == 200
def test_break_point_none_when_robust():
    sweep = [{"n":0,"surfaced":True,"rank":1},{"n":400,"surfaced":True,"rank":3}]
    assert cg.break_point(sweep, rank_threshold=10) is None
def test_degradation_flags_below_noise():
    d = cg.degradation({"C3":{"passed":5,"valid":5}}, {"C3":{"passed":5,"valid":5}})
    assert d["C3"]["delta"] == 0
```
- [ ] **Step 2: run → FAIL.**
- [ ] **Step 3: implement** the pure helpers + `main()` (fail loud on harness errors / missing JSON; `--gitignore`'d `crowded-verdict.json`).
- [ ] **Step 4: run → PASS** (unit). Add `crowded-verdict.json` to `dev/eval/traps/.gitignore`.
- [ ] **Step 5: commit** `feat(eval): crowded_gate.py — Tier-1 sweep + retrieval-guided Tier-2 applied check`

### Task 5: Tier-1 real sweep (free) + Tier-2 real run (LLM)

- [ ] **Step 1: Tier-1 sweep (free).** Run the retrieval sweep alone first (`crowded_gate.py --tier1-only` or equivalent). Read the per-axis surfaced/rank curves + break points. **Spot-check distractor realism** (note: validity crux): confirm some crowd variants rank *near* the load-bearing note (else the crowd isn't competing — fix the vocab knob before spending). No LLM spend.
- [ ] **Step 2: Announce Tier-2 cost** (estimate from the gate-smoke per-trial ~$0.45 × trial count; likely ~$20–40). Run the full `crowded_gate.py`.
- [ ] **Step 3: Verify** the run invoked real harnesses (non-zero costs), produced the table + `crowded-verdict.json`, and report: do the wins hold under crowding (crowded-warm still beats cold + within noise of toy-warm), or where do they break? Apply the confound guards before any favorable claim (note 96).

---

## Self-Review
- Spec coverage: crowd from real-vault variants + links (T1) ✓; read-only-source guard (T1 global + test) ✓; Tier-1 free retrieval probe via real multi-phrase query (T2) ✓; uniform --crowd injection (T3) ✓; retrieval-guided orchestrator + deltas (T4) ✓; real Tier-1 then Tier-2 with realism spot-check + confound guards (T5) ✓.
- Placeholder scan: pure-logic tasks have real test code; harness-mod + real-run steps reference exact files/flags.
- Type consistency: variant dict shape (slug/src_slug/links/fields) consistent across crowd/seed_into; sweep dict (n/rank/surfaced) consistent across probe/break_point.
