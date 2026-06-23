# Design — cross-cluster linking pass for recall (minimal first slice)

> **Status: BUILT (2026-06-23).** Step 2.6 is live in `skills/recall/SKILL.md`; the end-to-end
> isolated-agent proof is `dev/eval/traps/cake.py` (+ `cake_fixtures.py`). Built via
> `superpowers:writing-skills` TDD. Grounded in `docs/research/2026-06-22-emergent-synthesis-case.md`.
> **The build corrected this spec's original premise — see §1.**

## 1. Problem & scope

**Original premise (FALSIFIED by the build).** The spec assumed real `/recall` over the 6-note cake
vault formed **only within-cluster links and zero cross-domain links** — that k-means cleanly split a
"requirements" domain from a "mechanisms" domain, structurally foreclosing the means-ends joins.

**What the cake harness actually measured (2026-06-23, real MiniLM embedder):**
- k-means does **not** split req vs mech. It groups by **shared property** — e.g. cluster 0 =
  {cake-needs-sweetness, cake-needs-texture, sugar-provides-sweetness}; cluster 1 = the rest. Means-ends
  pairs land in the same cluster as often as not.
- The current skill (no Step 2.6) **already forms cross-note links ad hoc** — but **imprecisely**:
  baseline **4/9 correct means-ends edges with 5 spurious "flood" edges** (e.g.
  `cake-needs-sweetness → flour-provides-texture` — a need linked to a property-*mismatched* provider).

So the real gap is **not absence of links — it is absence of a precision gate.** This matches the C6
lesson already in memory ("precision is the whole game; a loose gate makes the graph worse").

**This slice (as built):** add a **precision gate** (Step 2.6) governing which cross-note edges
persist — generate candidates loosely, then persist only on a directed relation type + a specific
shared key that pairs ~1:1 (default DROP). The gate governs Step 2.5's within-cluster linking too (its
old rule "link every cluster member" was itself a flood source). **The win is precision, not presence:
the flood went 5 → 0 while correct means-ends rose 4/9 → 8–9/9 — and unrelated clusters (cake+git) and
a tempting analogy both yield 0 links.**

**Explicitly deferred (NOT this slice):**
- **Multi-axis grouping / LLM-as-axis-selector / multi-lens embedding** — these are the *generation*
  layer's engine for finding *non-topical* candidates (research §7). The GENERATE step here works
  with plain cosine "what else is like this"; multi-axis improves **generator quality**, and the
  precision gate (JUSTIFY) is **independent of generator quality** — so the slice is correct and
  complete without it (a weak generator + sound gate still yields only-valid links). Deferred to
  raise recall later, not to make this slice work.
- **Graph-expanded retrieval** (traverse wikilinks to surface bridge notes cosine missed) — a
  *different* deferral: it fixes *retrieval misses*, and the cake's bridge is already retrieved, so
  it isn't needed here.
- **Synthesis-note Z creation** — this slice persists *edges* (the graph-growth primitive), not new
  notes.
- **Iterative retrieve→reason→retrieve loop** — transitive hops with no retrieved bridge.
- **Across-groups output reduce** — dropped per research (underperforms).

## 1b. The roadmap — why the deferrals are a sequence, not a punt

The deferrals above are **dependency-ordered slices**, not optional extras. Build order is forced:
slice 2 reads cross-cluster edges, but none exist today (the cake check formed zero) — slice 1 is
what *writes* them. You literally cannot build a later slice first.

| # | slice | what it does | solves | depends on | status |
|---|---|---|---|---|---|
| **1** | **cross-cluster linking** (this doc) | writes typed cross-cluster *edges*, precision-gated | **compositional join** (the cake — bridge already retrieved) | — | spec'd |
| 2 | graph-expanded retrieval | traverses those edges to surface bridge notes cosine missed | **transitive** (the "we need sugar" case, where a link exists) | slice 1 (edges must exist to traverse) | deferred |
| 3 | multi-axis generation (LLM-as-axis-selector) | finds *non-topical* candidate pairs for slice 1 to judge | raises *recall* of cross-cluster links | slice 1 (improves its generator) | deferred |
| 4 | iterative retrieve→reason→retrieve | names the bridge entity and re-queries for it | transitive hops with **no** pre-existing link | slices 1–2 (residual) | deferred |
| — | synthesis-note Z | persist a new integrative *note*, not just an edge | richer artifacts | decide after slice 1 | deferred |
| — | across-groups output reduce | GraphRAG global reduce | — | — | **killed** (evidence: worse), NOT deferred |

**The honest gap:** slice 1 alone solves the **cake** but NOT the **sugar** case. The cake is a
*write* problem (bridge retrieved, never linked); the sugar case is a *retrieval* problem (bridge
never retrieved), and slice 1 operates on what's already retrieved. Slice 1 fixes the sugar case only
*indirectly* — it writes the edges slice 2 then traverses. So the two motivating cases need **slice 1
(cake) + slice 2 (sugar)**, and slice 2 is impossible until slice 1 exists.

**Why deferring is safe:** slice 1 stands alone — it grows the graph and fixes compositional joins;
if we built only it and stopped, we'd have gained something real and broken nothing (it adds
cross-cluster linking on top of today's within-cluster linking). Each later slice *adds* capability;
none is required for the prior to deliver value.

## 2. Where it grafts

A new **Step 2.6 — cross-cluster linking**, after the Step 2.5 per-cluster loop completes and
**before** the activation call. It reuses existing primitives only — no binary change:
- `engram amend --target <A> --relation "<B>|<typed rationale>"` (verified: `amend.go:25`,
  pipe-split at `relations.go`) to persist a cross-cluster edge — the graph-growth primitive.
  Direction per relation type (§4). **No `--activate`** — 2.6 writes an *edge*, it is not marking
  coverage (Step 2.5 uses `--activate` for its covered-outcome amend; do not cargo-cult it here).

**"Representative" = the note Step 2.5 settled on for a cluster:** the existing note it judged
Covered/Near, or the new note it wrote for an Absent cluster. Step 2.6 reasons over these
representatives (one per cluster).

**Ordering is load-bearing (verified):** `engram amend` resolves `--relation` targets against
**existing vault basenames** (`relations.go` strict resolution) — it can only link notes that already
exist. Step 2.5 must complete first so every cluster's representative exists in the vault before 2.6
links them. 2.6 reasons over the **post-2.5 vault state** (including the within-cluster edges 2.5 just
wrote — relevant to the transitive case).

## 3. The pass: generate → justify → persist (one LLM reasoning pass, not O(n²) calls)

The agent has all cluster members from the query payload in context. Step 2.6 is a single reasoning
pass over the surfaced **note members across clusters** (one prompt that sees all of them; not per-pair
calls). **Correction from the build:** the spec originally said "over cluster *representatives* (one
per cluster)", but the precise means-ends pairs are between *member* notes — with one representative
per cluster you could form at most one edge, never the three the cake needs. The gate ranges over
members; the GENERATE step proposes only the property-sharing candidates, keeping the pass bounded.

**A. GENERATE (loose, high recall — persists nothing).** Scan all note members across clusters for
*candidate* cross-cluster relationships. Use analogy / "what here relates to what" freely. This layer
proposes; it never writes. (Per note 69: analogy generates, it does not justify.)

**B. JUSTIFY (strict — the precision gate). The agent must emit an audit line per candidate** (so the
drop is observable/testable): `<A> ~ <B> | mode=<…> relation=<…> shared_key=<…> | PERSIST|DROP`.
A candidate is **PERSIST** only if ALL hold:
1. a **relation type** from the menu (§4),
2. the **shared key** that passes that relation's shared-key test (§4 — e.g. means-ends: the need's
   property term appears in the other note's provided effect), and
3. it is **not mere topical similarity** — the shared key must be a *specific property/entity/effect*,
   NOT a domain/topic word or a generic adjective shared by many notes (reject "both about baking",
   "both Go code", "both mention errors").
Any missing → **DROP**. **Default is DROP.** An analogy-suggested candidate with no rigorous relation
type + passing shared key is DROPPED (this is what the §6 pressure test (b) asserts on the audit lines).

**C. PERSIST.** For each surviving link: `engram amend --target <A> --relation "<B>|<TYPE>: <shared
key> — <one-line>"`. The rationale string encodes the relation TYPE so the edge is typed.

**Bound:** AutoK gives k∈[2,7] clusters → ≤21 pairs, examined in ONE pass (not per-pair calls). No
cost blowup.

## 4. The reasoning menu (justification layer) — from the research, tiered by grounding

A link persists only if it matches one of these (analogy is a *generator*, §3A — NOT in this menu).
The **shared-key test** is the concrete pass/fail the JUSTIFY gate checks; **direction** sets the
edge(s) to write:

| relation (persist if…) | mode | shared-key TEST (concrete) | direction (edges to write) | grounding |
|---|---|---|---|---|
| **compositional / part-whole** — A and B are parts of a common whole | (structural) | both notes name the *same whole* W that each is a part/component of | **symmetric**: A↔B (both) | strongest (3 traditions) |
| **means-ends / requires-provides** — A needs property X, B provides X (the cake) | abduction | the **need term in A is the provided effect in B** (A: "needs/requires X"; B: "provides/produces X"; same X) | **directed need→provider**: A→B | strong (planning + RST) |
| **causal / transitive** — A causes B, or A→B→C | deduction | A names an explicit cause/dependency whose **effect/target term is the subject of B** (the bridge term is literally present in both) | **directed cause→effect**: A→B (chains compose A→C only if A→B and B→C both pass) | strong |
| **abstraction** — A,B are instances of one schema | induction | the agent **names the schema S** and both A,B are explicit instances of it | **symmetric A↔B** (do NOT invent an S note this slice) | strong |
| **contradiction / supersession** — A asserts X, B asserts ¬X | (conflict) | A and B share the **same subject+predicate** with **opposite/negated object** | **symmetric A↔B** (flag conflict; supersession resolution is out of scope) | moderate |

**Non-topical guard (applies to every row):** the shared key must be a specific property/entity/
effect that the test above pins to *both* notes. A key that is a domain/topic word ("baking", "Go",
"errors") or a generic adjective shared by many notes FAILS — DROP.

**Tier as BUILT (corrected during the build):** enable **means-ends + causal/transitive +
contradiction** — the **directed** relations plus contradiction. The **symmetric similarity** relations
(**compositional/part-whole** and **abstraction**) are **disabled**: the build proved they are the
flood vector — their shared key ("the cake", a schema) is almost always a **hub** that meshes many
notes at once. They are gated behind a future tier, not enabled in slice 1. This is the
"directed-only menu" the pressure tests required to reach 0 flood.

**The non-topical guard, as built, includes the hub test:** a valid shared key pairs notes ~**1:1**;
a key that would link one note to 3+ others is a hub → DROP. This is what stops the within-topic flood
that pure "specific vs generic word" framing missed.

## 5. Acceptance test — the cake vault (the spec's pass/fail)

**Fixture:** the same 6-note two-domain cake vault used in the 2026-06-23 check
(`cake-needs-{sweetness,texture,fluffiness}` + `{sugar,flour,bakingsoda}-provides-…`). **Edge count =
vault inspection after the run:** grep each note's `[[wikilink]]` lines and classify endpoints by
domain (req / mech), exactly as the cake check already does. With Step 2.6 active:
- **MUST form** the 3 means-ends edges (directed need→provider per §4): `cake-needs-sweetness →
  sugar-provides-sweetness` (key: sweetness), texture→flour, fluffiness→soda — each rationale typed
  `means-ends`.
- **MUST NOT flood:** zero cross-cluster edge whose audit line failed the shared-key test. Note the
  must-not-flood guarantee is **behavioral** (the precision gate), reinforced **structurally** by 2.6
  only ranging over cluster *representatives* across clusters — it does not author within-cluster
  edges (that is 2.5's job). Assertion: every persisted cross-cluster edge has a PERSIST audit line
  naming a valid relation type + passing shared key.
- **Control (precision, the core risk):** a vault of genuinely *unrelated* clusters (cake notes + git
  notes) MUST form **zero** cross-cluster links — the default-DROP gate holds with no real relationship.

## 6. Build plan — COMPLETED (`superpowers:writing-skills` TDD, 2026-06-23)

Built. Harness: `dev/eval/traps/cake.py` + `cake_fixtures.py` (4 fixtures: cake/control/analogy/
transitive). The precision metric splits cross-note edges into property-matched means-ends links vs
spurious flood. Result across the cycle:
- **RED (current skill):** 4/9 correct means-ends, **5 spurious flood** edges (the gap was imprecision,
  not absence — see §1).
- **GREEN + 3 refactors** (gate Step 2.5's "link every member"; the hub test; directed-only menu):
  **8–9/9 correct means-ends, 0 flood.**
- **Control** (cake+git, unrelated clusters): **0** cross-links, all runs (n=4).
- **Analogy** (tempting bread/stock "rises" pair): **0** links, all runs — analogy DROPped.

(The spec's *original* RED/GREEN plan — "confirm 0 cross-domain links, then make 3 form" — was
discarded at the first baseline check, which revealed the premise was wrong: links already formed, just
imprecisely. See §1.)
- **PRESSURE TESTS (close loopholes); each is a fresh-vault fixture + a vault-inspection assertion:**
  (a) **control** — cake+git vault → 0 cross-cluster links (gate holds under no relationship);
  (b) **analogy-drop** — a vault with a tempting-but-invalid analogy pair → its audit line reads DROP
      and no edge is written (the audit line is what makes this falsifiable);
  (c) **edges-only** — 2.6 writes no new notes and does not rewrite the 2.5 representatives (note count
      unchanged; only `[[wikilink]]` lines added);
  (d) **transitive** — three notes in three clusters, A→B and B→C each passing causal, → the A→C-relevant
      chain edges form (2.6 reasons over the post-2.5 state).
- **Update the SKILL gap line precisely:** the current line is about cross-cluster *supersession*
  specifically. Replace with: "Cross-cluster *linking* is handled (Step 2.6). Cross-cluster
  *supersession resolution* — reconciling a conflict whose evidence did not cosine-cluster — remains
  deferred (2.6 flags the contradiction but does not resolve it)."

## 7. Risks & open questions (carry into the build)

- **Precision is the whole game** (C6 lesson): if the gate is loose, cross-cluster linking makes the
  graph *worse*. The default-no-link + name-the-shared-key requirement is the defense; the
  unrelated-clusters control is the test. If precision is poor in the build, tighten the gate (require
  two independent confirmations) before widening the menu.
- **Edge vs synthesis-note:** this slice persists edges only. Whether a means-ends join should *also*
  crystallize a synthesis note Z is deferred — decide after the edge version proves out on the cake.
- **Directionality** is now specified per relation in the §4 menu (directed for means-ends/causal,
  symmetric for compositional/abstraction/contradiction); the build follows that column.
- **Generator quality:** the GENERATE layer uses plain cosine this slice, which finds topical not
  structural candidates (research §7) — so recall of valid cross-domain links may be low at first.
  That is acceptable: the gate guarantees *precision*, and multi-axis grouping (deferred) is the lever
  to raise *recall* later. Do not conflate "few links formed" with "gate broken."

## Self-review checklist
- Grafts onto the real Step 2.5 (one LLM pass, existing `engram amend` primitive)?
- Generate/justify/persist separation explicit; analogy excluded from the persist menu (note 69)?
- Default-no-link gate concrete (name relation type + shared key + non-topical)?
- Cake acceptance test has BOTH a must-form and a must-not-flood/control case?
- Build correctly flagged as a separate writing-skills TDD effort with RED/GREEN/pressure cases?
- Deferrals named so scope doesn't creep (multi-axis, synthesis-notes, iterative loop, output reduce)?
