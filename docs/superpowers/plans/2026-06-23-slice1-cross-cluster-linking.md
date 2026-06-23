# Slice 1 — Cross-Cluster Linking Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:writing-skills to make the SKILL.md edit (the GREEN step is a skill change — RED→GREEN→pressure-test is non-negotiable per the repo rule). Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Teach the `recall` skill to write typed, precision-gated `[[wikilink]]` edges between notes that live in *different* clusters — the foundational graph-growth primitive the cake compositional-join case needs.

**Architecture:** Add a new **Step 2.6 — cross-cluster linking** to `skills/recall/SKILL.md`, inserted **after the Step 2.5 per-cluster coverage loop and before the activation paragraph** (which is relabeled Step 2.7 so activation runs last, matching design §2). It is one LLM reasoning pass (generate → justify → persist) over the cluster *representatives* — the note each cluster *settled on* during the 2.5 loop (Covered/Near note, or the note written for an Absent cluster), accumulated explicitly as the loop runs (NOT `candidate_l2s`, which are pre-judgment synthesis candidates). It persists an edge via the existing `engram amend --target <A> --relation "<B>|<typed rationale>"` primitive only when a candidate passes a reasoning-menu relation type + a named shared key (default DROP). No binary change. The proof is an end-to-end **isolated-agent harness** (`dev/eval/traps/cake.py`) that builds fixture vaults, runs the real warm `/recall` against each, and inspects the vault's wikilinks across domains.

**Tech Stack:** Python 3 harness (reuses `wrun.build_warm_cfg`, `run.MODELS`, `_slug`); `claude -p` subprocess with warm config + seeded vault; `engram learn fact` to build fixtures with correct `.vec.json` sidecars; markdown SKILL.md edit.

## Global Constraints

- Skill edits MUST go through `superpowers:writing-skills` TDD — RED (baseline) before any SKILL.md change, GREEN, then pressure tests. No exceptions (repo CLAUDE.md).
- Use `engram` installed binary for all vault ops; never hand-write `.vec.json` sidecars.
- Step 2.6 reuses existing primitives only — `engram amend --target … --relation "<B>|…"`. **No `--activate`** in 2.6 (that is 2.5's coverage marker; 2.6 writes an edge).
- Step 2.6 ranges over cluster **representatives only** (one note per cluster, the note 2.5 settled on) — never authors within-cluster edges.
- Analogy is a **generator, not a justifier** (note 69): an analogy-only candidate with no menu relation type + passing shared key is DROPPED.
- Default is **DROP**. Precision is the whole game (C6 lesson): a loose gate makes the graph worse.
- Commit trailer: `AI-Used: [claude]`.
- Harness data isolation: each fixture run uses its own `TRAPS_ROOT` subtree; never clobber other runs' data.

---

### Task 1: Cake-vault fixture builder + cross-cluster edge inspector (RED baseline)

**Files:**
- Create: `dev/eval/traps/cake_fixtures.py` (builders for the 4 fixture vaults)
- Create: `dev/eval/traps/cake.py` (warm runner + vault-inspection check)
- Reuse: `dev/eval/traps/wrun.py:build_warm_cfg`, `dev/eval/traps/run.py:MODELS`, `wrun._slug`

**Interfaces:**
- Produces: `cake_fixtures.build(kind, dst)` where `kind ∈ {"cake","control","analogy","transitive"}` → writes a seeded vault dir at `dst` (notes + `.vec.json` sidecars via `engram learn fact`). Returns the list of note basenames.
- Produces: `cake.inspect_edges(vault)` → `{note_basename: [linked_basenames]}` parsed from `[[wikilink]]` lines; `cake.classify_cross(vault)` → list of `(src, dst)` pairs whose endpoints are in different domains (domain = filename prefix before first `-`-group, e.g. `cake`/`sugar`).
- Produces: `cake.run_one(kind, model, cfg, idx)` → runs warm `/recall` over a fresh copy of the fixture vault, returns `{kind, idx, cross_edges, note_delta, recalled, cost, turns, wd}`.

- [ ] **Step 1: Write the cake fixture builder**

`cake_fixtures.py` — each note is created with `engram learn fact` so the means-ends shared key is a literal term shared between the need note and the provider note:

```python
"""Builders for the cake cross-cluster fixtures. Each note is a real engram fact note
(with a .vec.json sidecar) so `engram query` clusters them exactly as production would."""
import os, shutil, subprocess

# (slug, subject, predicate, object) — the predicate/object encode the means-ends shared key:
# a "needs <X>" note and a "provides <X>" note share the literal key X.
CAKE_REQ = [
    ("cake-needs-sweetness", "a cake", "needs", "sweetness"),
    ("cake-needs-texture",   "a cake", "needs", "texture"),
    ("cake-needs-fluffiness","a cake", "needs", "fluffiness"),
]
CAKE_MECH = [
    ("sugar-provides-sweetness",     "sugar",        "provides", "sweetness"),
    ("flour-provides-texture",       "flour",        "provides", "texture"),
    ("bakingsoda-provides-fluffiness","baking soda", "provides", "fluffiness"),
]
# Unrelated cluster for the precision control — git notes share no shared key with cake notes.
GIT_NOTES = [
    ("git-rebase-before-merge", "a feature branch", "must be rebased on", "main before merge"),
    ("git-ff-only-merges",      "merges",           "must be",            "fast-forward only"),
    ("git-never-push-unreviewed","worktree work",   "must not be",        "pushed before review"),
]

def _learn(vault, slug, subj, pred, obj):
    env = dict(os.environ); env["ENGRAM_VAULT_PATH"] = vault
    subprocess.run(
        ["engram", "learn", "fact", "--slug", slug, "--position", "top",
         "--source", f"cake fixture: {slug}",
         "--situation", f"{subj} {pred} {obj}",
         "--subject", subj, "--predicate", pred, "--object", obj],
        env=env, check=True, capture_output=True, text=True)

def build(kind, dst):
    if os.path.exists(dst):
        shutil.rmtree(dst)
    os.makedirs(dst)
    if kind == "cake":
        notes = CAKE_REQ + CAKE_MECH
    elif kind == "control":
        notes = CAKE_REQ + CAKE_MECH + GIT_NOTES   # cake + genuinely-unrelated git cluster
    elif kind == "analogy":
        # a tempting-but-invalid analogy pair: both "rise" but no shared provided property key
        notes = CAKE_MECH + [
            ("bread-dough-rises", "bread dough", "rises when", "yeast ferments"),
            ("stock-market-rises","the stock market", "rises when", "demand grows"),
        ]
    elif kind == "transitive":
        notes = [
            ("joe-wants-cake",        "Joe",   "wants",     "cake"),
            ("cake-needs-sweetness",  "a cake","needs",     "sweetness"),
            ("sugar-provides-sweetness","sugar","provides", "sweetness"),
        ]
    else:
        raise ValueError(kind)
    for slug, subj, pred, obj in notes:
        _learn(dst, slug, subj, pred, obj)
    # Embed-on-write can silently warn-and-skip the sidecar (learn.go autoEmbedNote). Without a
    # .vec.json, `engram query` cannot cluster the note and the RED/GREEN check passes vacuously
    # (0 clusters → 0 edges). Verify every note got a sidecar; fail loud if not.
    missing = [n for n in os.listdir(dst)
               if n.endswith(".md") and not os.path.exists(os.path.join(dst, n[:-3] + ".vec.json"))]
    if missing:
        raise RuntimeError(f"fixture {kind}: missing .vec.json sidecars for {missing} — "
                           f"run `engram embed apply --missing` or check the embedder")
    return sorted(n for n in os.listdir(dst) if n.endswith(".md"))
```

The returned basenames carry the Luhmann prefix (e.g. `7.2026-06-23.sugar-provides-sweetness.md`).
Step 2.6 must amend using **the basename exactly as it appears in the query payload** — `engram amend`
resolves `--relation` targets strictly against existing vault basenames (`relations.go`
`resolveRelationTargetsStrict`) and errors on an unresolved bare slug.

- [ ] **Step 2: Write the edge inspector + warm runner (`cake.py`)**

```python
"""Cake cross-cluster check. Build a fixture vault, run the real warm /recall over it
(skill Step 2.6 should write cross-cluster edges), then inspect the vault's [[wikilinks]].

Usage: python3 cake.py [--kind cake] [--model opus] [--n 3] [--workers 3]
"""
import argparse, glob, os, re, shutil, subprocess, sys, tempfile, time, json
import concurrent.futures as cf

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import cake_fixtures
from run import MODELS
from wrun import build_warm_cfg, _slug

ROOT = os.environ.get("TRAPS_ROOT", "/tmp/cake")
WIKILINK = re.compile(r"\[\[([^\]]+)\]\]")
PROMPT = (
    "Invoke your /recall skill for this situation, processing EVERY cluster it returns exactly as "
    "the skill directs — including any cross-cluster linking step. Situation: I am planning how to "
    "bake a cake and what each ingredient contributes.\n\nAfter recall, briefly state your plan — no code.")

def _domain(basename):
    return basename.split("-", 1)[0]            # cake / sugar / flour / git / joe …

def inspect_edges(vault):
    edges = {}
    for f in glob.glob(os.path.join(vault, "*.md")):
        base = os.path.basename(f)
        text = open(f, errors="ignore").read()
        targets = set()
        for m in WIKILINK.findall(text):
            t = m.split("|")[0].strip()
            if not t.endswith(".md"):
                t += ".md"
            targets.add(t)
        edges[base] = sorted(targets)
    return edges

def classify_cross(vault):
    edges = inspect_edges(vault)
    cross = []
    for src, dsts in edges.items():
        for d in dsts:
            if _domain(d) != _domain(src) and os.path.exists(os.path.join(vault, d)):
                cross.append((src, d))
    return sorted(set(cross))

def run_one(kind, model, cfg, idx):
    wd = tempfile.mkdtemp(prefix=f"{kind}-{idx}-", dir=os.path.join(ROOT, "ws"))
    vault = os.path.join(wd, "vault")
    cake_fixtures.build(kind, vault)
    before = len(glob.glob(os.path.join(vault, "*.md")))
    chunks = os.path.join(wd, "chunks"); os.makedirs(chunks, exist_ok=True)
    env = dict(os.environ)
    env["CLAUDE_CONFIG_DIR"] = cfg
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "32000"
    env["ENGRAM_VAULT_PATH"] = vault
    env["ENGRAM_CHUNKS_DIR"] = chunks
    env["ENGRAM_TRANSCRIPT_DIR"] = os.path.join(cfg, "projects", _slug(wd))
    args = ["claude", "-p", PROMPT, "--output-format", "json",
            "--model", MODELS[model], "--permission-mode", "bypassPermissions"]
    out = {}
    for backoff in (0, 15, 45, 120):
        if backoff:
            time.sleep(backoff)
        r = subprocess.run(args, cwd=wd, env=env, capture_output=True, text=True)
        try:
            out = json.loads(r.stdout)
        except Exception:
            out = {}
        if (out.get("is_error") or not out) and (out.get("total_cost_usd", 0) or 0) < 0.02:
            continue
        break
    after = len(glob.glob(os.path.join(vault, "*.md")))
    cross = classify_cross(vault)
    sid = out.get("session_id"); recalled = False
    if sid:
        for rt, _, fs in os.walk(os.path.join(cfg, "projects")):
            if f"{sid}.jsonl" in fs:
                tx = open(os.path.join(rt, f"{sid}.jsonl"), errors="ignore").read()
                recalled = "engram query" in tx
                break
    return {"kind": kind, "idx": idx, "cross_edges": cross, "note_delta": after - before,
            "recalled": recalled, "cost": out.get("total_cost_usd", 0) or 0,
            "turns": out.get("num_turns"), "wd": wd}

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--kind", default="cake")
    ap.add_argument("--model", default="opus")
    ap.add_argument("--n", type=int, default=3)
    ap.add_argument("--workers", type=int, default=3)
    a = ap.parse_args()
    os.makedirs(os.path.join(ROOT, "ws"), exist_ok=True)
    cfg = os.path.join(ROOT, "warm-cfg"); build_warm_cfg(cfg)
    print(f"cake check: kind={a.kind} × n={a.n} ({a.model})")
    results = []
    with cf.ThreadPoolExecutor(max_workers=a.workers) as ex:
        futs = {ex.submit(run_one, a.kind, a.model, cfg, i): i for i in range(a.n)}
        for fut in cf.as_completed(futs):
            r = fut.result(); results.append(r)
            print(f"  [{r['kind']} #{r['idx']}] cross_edges={len(r['cross_edges'])} "
                  f"note_delta={r['note_delta']} recall={r['recalled']} ${r['cost']:.2f}")
            for s, d in r["cross_edges"]:
                print(f"        {s} -> {d}")
    json.dump(results, open(os.path.join(ROOT, f"cake-{a.kind}-results.json"), "w"), indent=1)
    print(f"\ntotal spend: ${sum(r['cost'] for r in results):.2f}")

if __name__ == "__main__":
    main()
```

- [ ] **Step 3: Run the RED baseline against the CURRENT skill**

Run: `cd /Users/joe/repos/personal/engram/dev/eval/traps && python3 cake.py --kind cake --n 3`
Expected (RED): every run reports `cross_edges=0` — the current skill (no Step 2.6) forms only within-cluster links. This reproduces and commits the documented baseline.

- [ ] **Step 4: Commit the harness + RED baseline**

```bash
git add dev/eval/traps/cake_fixtures.py dev/eval/traps/cake.py
git commit -m "test(recall): cake cross-cluster harness + RED baseline (0 cross-domain links)"
```

---

### Task 2: GREEN — add Step 2.6 to the recall skill (writing-skills)

**Files:**
- Modify: `skills/recall/SKILL.md` (insert Step 2.6 after Step 2.5's activation block region, before Step 3; update the "Known gap" line; add a red-flag row)

**Interfaces:**
- Consumes: the cake harness from Task 1.
- Produces: the Step 2.6 procedure text matching design §3 (generate→justify→persist) + §4 (reasoning menu).

- [ ] **Step 1: Insert Step 2.6 into SKILL.md (and relabel activation as Step 2.7)**

Two structural edits, faithful to design §2's ordering (loop → cross-cluster link → activate):
1. In the Step 2.5 per-cluster loop's closing instructions, add the representative-tracking sentence:
   *"As you finish each cluster, record its **representative** — the basename you amended (Covered/Near)
   or created (Absent). Step 2.6 ranges over this one-note-per-cluster list, not over `candidate_l2s`
   (those are pre-judgment candidates)."*
2. Insert a new `### Step 2.6` block **after the per-cluster loop and before the activation paragraph**,
   and relabel the activation paragraph `### Step 2.7 — Activation` (so activation runs last). Step 2.6
   content (verbatim mechanism from design §3/§4):

```markdown
### Step 2.6 — Cross-cluster linking (graph growth, agent-judged)

Step 2.5 links *within* each cluster; the complementary edges that join clusters are never written
(cosine split the domains). After the per-cluster loop — and **before** Step 2.7 activation — run ONE
reasoning pass over the **cluster representatives** (the one-note-per-cluster list you recorded during
the 2.5 loop: each cluster's Covered/Near note, or the note written for an Absent cluster) to grow the
graph across clusters. Use the representative's **basename exactly as it appears in the payload**
(Luhmann-prefixed) as the `--target`/`<B>` — `engram amend` resolves relation targets strictly against
existing vault basenames and errors on a bare slug.

**A. GENERATE (loose, persists nothing).** Scan all representatives for *candidate* cross-cluster
relationships — use analogy and "what here relates to what" freely. This proposes; it never writes.
(Analogy generates, it does not justify.)

**B. JUSTIFY (strict — the precision gate).** Emit one audit line per candidate so the decision is
observable: `<A> ~ <B> | mode=<…> relation=<…> shared_key=<…> | PERSIST|DROP`. A candidate is
**PERSIST** only if ALL hold: (1) a relation TYPE from the menu below, (2) the SHARED KEY that passes
that relation's shared-key test, and (3) the key is a *specific property/entity/effect* — NOT a
domain/topic word or generic adjective shared by many notes ("both about baking", "both Go code").
Any missing → **DROP. Default is DROP.**

| relation (persist if…) | shared-key TEST | direction |
|---|---|---|
| **compositional / part-whole** — A,B are parts of a common whole | both name the same whole W each is part of | symmetric A↔B |
| **means-ends / requires-provides** — A needs X, B provides X | the need term in A is the provided effect in B (same X) | directed need→provider A→B |
| **causal / transitive** — A causes/depends-on B | A names a cause/dependency whose effect term is B's subject (bridge term in both) | directed cause→effect A→B (chains compose only if each hop passes) |
| **abstraction** — A,B are instances of one schema | name the schema S; both A,B are explicit instances (do NOT invent an S note) | symmetric A↔B |
| **contradiction** — A asserts X, B asserts ¬X | same subject+predicate, opposite/negated object | symmetric A↔B (flag conflict; resolution out of scope) |

**C. PERSIST.** For each surviving link: `engram amend --target <A> --relation "<B>|<TYPE>: <shared key> — <one-line>"`.
The rationale string encodes the relation TYPE so the edge is typed. **No `--activate`** (2.6 writes an
edge, it does not mark coverage). Write both directions for symmetric relations; one for directed.

**Bound:** AutoK gives k≤7 clusters → ≤21 pairs in ONE pass — no per-pair calls, no cost blowup.
```

- [ ] **Step 2: Update the "Known gap" line**

Replace the current line 152 (`**Known gap:** cross-cluster supersession …`) with:

```markdown
**Cross-cluster linking is handled by Step 2.6.** Cross-cluster *supersession* — reconciling a conflict
whose evidence did not cosine-cluster — remains deferred. Step 2.6 *links* across clusters and flags a
contradiction (the contradiction row) but does not *resolve* the supersession.
```

- [ ] **Step 3: Add a red-flag row** to the Step-table guarding against the analogy loophole and `--activate` cargo-culting:

```markdown
| You persisted a cross-cluster edge from an analogy with no named shared key | DROP it — analogy generates candidates; only a menu relation type + passing shared key persists |
| You passed `--activate` on a Step 2.6 amend | 2.6 writes an edge, not a coverage mark — `--activate` is Step 2.5's |
```

- [ ] **Step 4: Run the GREEN cake check**

Run: `cd dev/eval/traps && python3 cake.py --kind cake --n 3`
Expected (GREEN): the 3 means-ends edges form — `cake-needs-sweetness → sugar-provides-sweetness`, `cake-needs-texture → flour-provides-texture`, `cake-needs-fluffiness → bakingsoda-provides-fluffiness` — in a majority of runs, each persisted from a PERSIST audit line. `cross_edges ≥ 3` and every cross edge is one of those three (no flood).

- [ ] **Step 5: Gate B (design-fit) then commit**

After the SKILL.md change reads as part of the whole (Step 2.6 fits the 2.5/2.7 numbering and tone), commit:

```bash
git add skills/recall/SKILL.md
git commit -m "feat(recall): Step 2.6 cross-cluster linking (precision-gated edges)"
```

---

### Task 3: Pressure test (a) — control (precision holds under no relationship)

**Files:** none (uses `cake.py --kind control`)

- [ ] **Step 1: Run the control fixture**

Run: `cd dev/eval/traps && python3 cake.py --kind control --n 3`
Expected: the cake↔git cross-domain pairs form **0** edges (the means-ends edges *within* the cake half may still form; the assertion is **no cake↔git edge**). The default-DROP gate holds when there is no real relationship. If any cake↔git edge appears, the gate is too loose → tighten §B step 3 (non-topical guard) in SKILL.md and re-run.

- [ ] **Step 2: Record the control result** in the harness results JSON (already written by `cake.py`). No commit unless SKILL.md was tightened.

---

### Task 4: Pressure test (b) — analogy-drop (the audit line is falsifiable)

**Files:** none (uses `cake.py --kind analogy`)

- [ ] **Step 1: Run the analogy fixture**

Run: `cd dev/eval/traps && python3 cake.py --kind analogy --n 3`
Expected: `bread-dough-rises` and `stock-market-rises` share the surface word "rises" but no provided-property shared key — the audit line reads `… | DROP` and **no edge** is written between them. Verify by grepping the run's transcript for the audit line and the vault for the absence of the edge.

- [ ] **Step 2: Verify the audit line in the transcript**

Run: `grep -rl "DROP" /tmp/cake/warm-cfg/projects/ | head` then inspect — confirm a `bread … ~ stock … | … DROP` style line exists. The audit line is what makes the drop testable; if it's missing, strengthen the "emit one audit line per candidate" instruction in SKILL.md §B.

---

### Task 5: Pressure test (c) — edges-only (2.6 mutates links, not notes)

**Files:** none (uses `cake.py --kind cake`, the `note_delta` field)

- [ ] **Step 1: Assert note count unchanged**

From the Task 2 Step 4 GREEN run output: every cake run must report `note_delta=0` — Step 2.6 adds `[[wikilink]]` lines via `engram amend` but creates no new notes and rewrites no representative's content body. If `note_delta>0`, Step 2.6 is over-reaching (writing a synthesis note Z, which is deferred) → tighten SKILL.md to "persist edges only".

---

### Task 6: Pressure test (d) — transitive (reasons over post-2.5 state)

**Files:** none (uses `cake.py --kind transitive`)

- [ ] **Step 1: Run the transitive fixture**

Run: `cd dev/eval/traps && python3 cake.py --kind transitive --n 3`
Expected: with `joe-wants-cake`, `cake-needs-sweetness`, `sugar-provides-sweetness` in (up to) three clusters, the causal/means-ends hops that pass the shared-key test form edges (`cake-needs-sweetness → sugar-provides-sweetness` via means-ends; `joe-wants-cake → cake-needs-sweetness` via causal if the bridge term "cake" is present in both). This confirms 2.6 reasons over the post-2.5 vault state. **Note:** full transitive *retrieval* (surfacing sugar from a "Joe wants cake" query when the bridge was never retrieved) is **slice 2** and explicitly NOT asserted here — only that the edges between *retrieved* representatives form.

---

### Task 7: Document + close

**Files:**
- Modify: `docs/design/2026-06-23-cross-cluster-linking.md` (flip Status to "built", point §6 at the harness)
- Modify: `docs/architecture/c1-system-context.md` (the recall-flow sequence diagram — add a Step 2.6 cross-cluster-linking box after the per-cluster loop and before activation, so the diagram is not stale)
- Modify: `docs/superpowers/plans/2026-06-22-c4-c6-test-plan.md` (if its C6 section references the cross-cluster deferral, point it at the built slice-1 foundation)

- [ ] **Step 1: Update the design doc status + §6 + the c1 sequence diagram** to reflect the built skill and the committed harness path (`dev/eval/traps/cake.py`). The recall sequence in `docs/architecture/c1-system-context.md` currently ends Step 2.5 at the activation step; add the Step 2.6 cross-cluster-linking step (after the per-cluster loop, before activation) so the diagram matches the new flow. Gate C over every touched doc. (CLAUDE.md's Directory Structure does not enumerate `dev/eval/` harnesses, so no CLAUDE.md change is needed unless the structure list is extended.)

- [ ] **Step 2: Final commit** (gate D over commit prose):

```bash
git add -A
git commit -m "docs(recall): mark slice-1 cross-cluster linking built; wire harness"
```

## Self-Review

**1. Spec coverage:** design §3 generate/justify/persist → Task 2 Step 1; §4 menu → Task 2 Step 1 table; §5 cake acceptance (must-form + must-not-flood + control) → Tasks 2/3; §6 RED/GREEN/pressure (a/b/c/d) → Tasks 1–6; gap-line update → Task 2 Step 2. All covered.

**2. Placeholder scan:** no TBD/TODO; every code step has real code; every run step has exact command + expected output.

**3. Type consistency:** `build(kind, dst)`, `inspect_edges`, `classify_cross`, `run_one(kind,…)`, `_domain`, `_slug`, `MODELS`, `build_warm_cfg` consistent across Tasks 1–6.

**Known accepted risk:** the GREEN/pressure checks invoke a real model and are mildly non-deterministic — the bar is behavioral change (RED 0 → GREEN ≥3 means-ends) and gate-holding (control/analogy 0), assessed over n=3, not a single deterministic equality.
