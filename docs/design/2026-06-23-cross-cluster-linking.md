# Design — cross-cluster linking pass for recall (minimal first slice)

> **Status: design spec (spec-only).** Building this edits `skills/recall/SKILL.md`, which the repo
> rule says MUST go through `superpowers:writing-skills` TDD (RED→GREEN→pressure-test). That build is
> a separate effort; §6 specs its RED/GREEN/pressure cases. Grounded in
> `docs/research/2026-06-22-emergent-synthesis-case.md`.

## 1. Problem & scope

**Proven failure (cake check, 2026-06-23):** real `/recall` over a 6-note two-domain vault formed
**only within-cluster links** (req→req, mech→mech) and **zero cross-domain links** — even though
retrieval surfaced all 6 notes. Recall's Step 2.5 processes **"one write per cluster"** and never
looks across clusters, so the complementary edges (`cake-needs-sweetness` ↔ `sugar-provides-
sweetness`) that synthesis/graph-traversal need are never written.

**This slice:** add a **cross-cluster pass** to recall that, after the per-cluster loop, examines the
clusters *together*, proposes candidate cross-cluster relationships, validates each against a
reasoning menu, and **persists only precision-gated links — default no-link.** Grows the graph in the
one dimension the current write path can't.

**Explicitly deferred (NOT this slice):** multi-axis grouping / LLM-as-axis-selector / multi-lens
embedding (those address *retrieval misses*; the cake's bridge is already retrieved — different
problem); synthesis-note Z creation (this slice persists *edges*, the graph-growth primitive, not new
notes); the iterative retrieve→reason→retrieve loop (transitive hops with no retrieved bridge);
across-groups output reduce (dropped per research — underperforms).

## 2. Where it grafts

A new **Step 2.6 — cross-cluster linking**, after the Step 2.5 per-cluster loop completes (every
cluster's representative note exists) and **before** the activation call. It reuses existing
primitives only — no binary change:
- `engram amend --target <A> --relation "<B>|<typed rationale>"` to persist a cross-cluster edge
  (the graph-growth primitive). Bidirectional intent: link both directions where the relation is
  symmetric, one direction where directed (A requires B).

## 3. The pass: generate → justify → persist (one LLM reasoning pass, not O(n²) calls)

The agent already holds the whole payload (all clusters + their representatives from 2.5). Step 2.6
is a single reasoning pass over that set:

**A. GENERATE (loose, high recall — persists nothing).** Scan all cluster representatives for
*candidate* cross-cluster relationships. Use analogy / "what here relates to what" freely. This layer
proposes; it never writes. (Per note 69: analogy generates, it does not justify.)

**B. JUSTIFY (strict — the precision gate).** For each candidate, it is persisted **only if** the
agent can name ALL of:
1. a **relation type** from the menu (§4),
2. the **specific shared key / bridge** that joins the two notes (the property, entity, or effect —
   e.g. "sweetness"), and
3. a one-line statement of the relationship that is **not mere topical similarity** ("both about
   baking" is NOT a valid link).
If any of the three is missing → **no link.** **Default is no-link.** A candidate that only an
analogy suggested, with no rigorous relation type + shared key, is dropped.

**C. PERSIST.** For each surviving link: `engram amend --target <A> --relation "<B>|<TYPE>: <shared
key> — <one-line>"`. The rationale string encodes the relation TYPE so the edge is typed.

**Bound:** AutoK gives k∈[2,7] clusters → ≤21 pairs, examined in ONE pass (not per-pair calls). No
cost blowup.

## 4. The reasoning menu (justification layer) — from the research, tiered by grounding

A link persists only if it matches one of these (analogy is a *generator*, §3A — NOT in this menu):

| relation (persist if…) | mode | shared-key test | grounding |
|---|---|---|---|
| **compositional / part-whole** — A and B are parts of a common whole | (structural) | the whole they compose | strongest (3 traditions) |
| **means-ends / requires-provides** — A states a need keyed on X, B provides X (the cake) | abduction | the property/effect X (`goal ∩ effects`) | strong (planning + RST) |
| **causal / transitive** — A→B and B→C, or A causes B | deduction | the bridge term B | strong |
| **abstraction** — A,B are instances of one schema | induction | the schema | strong |
| **contradiction / supersession** — A asserts X, B asserts ¬X | (conflict) | the contested claim | moderate |

Tier note for the build: enable **compositional + means-ends + causal/transitive** first (best
grounded, and they cover the cake + the transitive case). Abstraction/contradiction second.

## 5. Acceptance test — the cake vault (the spec's pass/fail)

Re-run the cake check (`/tmp/cake-check`, 6 notes, 2 domains) with Step 2.6 active:
- **MUST form** the 3 means-ends edges: `cake-needs-sweetness ↔ sugar-provides-sweetness` (key:
  sweetness), texture↔flour, fluffiness↔soda — each typed `means-ends`.
- **MUST NOT flood:** no edge that fails the shared-key test (e.g. no "all baking-related" mass-link;
  no req↔req or mech↔mech cross-links beyond what 2.5 already made). Precision target: every persisted
  cross-cluster edge names a valid relation type + shared key.
- **Control (precision):** a vault of genuinely *unrelated* clusters (e.g. cake notes + git notes)
  MUST form **zero** cross-cluster links — the default-no-link gate holds under no-real-relationship.

## 6. Build plan (separate effort — `superpowers:writing-skills` TDD)

The build edits `skills/recall/SKILL.md` Step 2.5/2.6. Per the repo rule, it runs writing-skills TDD:
- **RED (baseline):** run the cake check on the *current* skill → confirm 0 cross-domain links
  (already observed — this is the documented baseline).
- **GREEN:** add the Step 2.6 text (§3 + §4) → re-run → the 3 means-ends edges form.
- **PRESSURE TESTS (close loopholes):** (a) the unrelated-clusters control forms 0 links (gate holds);
  (b) an analogy-only candidate with no shared key is dropped; (c) the pass doesn't rewrite or
  duplicate the per-cluster notes from 2.5 (edges only); (d) a transitive triple (A→B→C across
  clusters) forms the A–C-relevant edge. Each is a fresh-vault fixture + assertion.
- Update the SKILL's "Known gap: cross-cluster … not handled" line (it's being addressed).

## 7. Risks & open questions (carry into the build)

- **Precision is the whole game** (C6 lesson): if the gate is loose, cross-cluster linking makes the
  graph *worse*. The default-no-link + name-the-shared-key requirement is the defense; the
  unrelated-clusters control is the test. If precision is poor in the build, tighten the gate (require
  two independent confirmations) before widening the menu.
- **Edge vs synthesis-note:** this slice persists edges only. Whether a means-ends join should *also*
  crystallize a synthesis note Z is deferred — decide after the edge version proves out on the cake.
- **Directionality:** means-ends is directed (need→provider); compositional/contradiction are
  symmetric. The build must set link direction per relation type.

## Self-review checklist
- Grafts onto the real Step 2.5 (one LLM pass, existing `engram amend` primitive)?
- Generate/justify/persist separation explicit; analogy excluded from the persist menu (note 69)?
- Default-no-link gate concrete (name relation type + shared key + non-topical)?
- Cake acceptance test has BOTH a must-form and a must-not-flood/control case?
- Build correctly flagged as a separate writing-skills TDD effort with RED/GREEN/pressure cases?
- Deferrals named so scope doesn't creep (multi-axis, synthesis-notes, iterative loop, output reduce)?
