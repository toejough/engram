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

The agent has all cluster members from the query payload plus the representatives it saw/wrote during
Step 2.5 — all in context. Step 2.6 is a single reasoning pass over the cluster representatives (one
prompt that sees all of them; not per-pair calls):

**A. GENERATE (loose, high recall — persists nothing).** Scan all cluster representatives for
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

Tier for the build: enable **compositional + means-ends + causal/transitive** first (best grounded;
they cover the cake + transitive cases). Abstraction/contradiction second.

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

## 6. Build plan (separate effort — `superpowers:writing-skills` TDD)

The build edits `skills/recall/SKILL.md` Step 2.5/2.6. Per the repo rule, it runs writing-skills TDD:
- **RED (baseline):** run the cake check on the *current* skill → confirm 0 cross-domain links
  (already observed — this is the documented baseline).
- **GREEN:** add the Step 2.6 text (§3 + §4) → re-run → the 3 means-ends edges form.
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
