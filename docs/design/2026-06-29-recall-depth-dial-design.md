# Recall as a depth dial — design + decomposition

> **Ask (Joe, 2026-06-29).** Is over-firing really a problem? Prefer firing *frequently* with a smaller /
> adjustable payload (quick-consider vs deep-think). Revive the "two part breakdown." Abstract recall into a
> **sliding scale of memory depth** — engram flags (`--n-turns-back/--n-notes/--n-chunks`), variable query
> count, and variable reasoning depth for crystallization. Decompose into a *small* set of roadmap
> items/investigations; move quickly.

**Verdict (TL;DR).** Lowering recall's per-fire cost is a *better* lever than scoping the cues — it attacks the
bigger term in `tax = over-fire × per-fire-cost` (note 109). The honest instantiation is a **2-rung dial via a
read-vs-write split**: a cheap `glance` (retrieve, recency-resolve, apply — no crystallization writes; the Step-2.7
activate is a metadata bump, not a content write) vs the full `deep`
(adds crystallization). It does **not** dissolve over-fire — it *relaxes the ceiling proportionally* to the
cost cut, and the **value** gate persists (memory is net-negative on non-idiosyncratic work even when cheap).
Decompose into 3 gated items — **measure glance-delivery → build the modes → relax the guidance's cost bar** —
with the load-bearing risk (does a shallow fire still *deliver*, and does the agent escalate correctly +
sparingly?) measured first, cheaply, before any build.

## The lever — relax the over-fire ceiling, don't pretend to dissolve it

`tax = over-fire × per-fire-cost` (note 109). The 2026-06-29 recall-at-decision-moments work shrank
**over-fire** (scoped the cues). This attacks **per-fire-cost**. Note 109 is precise, though: a fire *escapes
the ratio bound* only when its cost is **~0** (a deterministic hook / always-loaded doc). A `glance` recall is
**cheaper, not zero** — it still pages a (smaller) payload and synthesizes. So glance **raises the viable
firing rate proportionally to the cost cut; it does not remove the bound.** Per note 109's own discipline we
must pin glance's *measured* per-fire cost (Item 1b) and re-derive the viable firing ceiling at a chosen
fire-unit — not assert "fire freely."

**Back-of-envelope.** deep ≈ ~190s (notes 91/93; `docs/design/2026-06-25-recall-cost-isolation.md`). glance
keeps the binary (~3.5s) + a reduced Step-2 read + recency-resolve + apply, and drops the crystallization write
side — plausibly **~tens of seconds**, i.e. a several-fold cut. **Item 1b measures the real figure**; the point
here is order-of-magnitude, and the firing-ceiling claim is owed that number before it's made.

The "two part breakdown we were aiming for" is parked as **§P7 "two-speed quick-probe"** of the recall-trigger
analysis (`docs/design/2026-06-27-recall-trigger-patterns-and-proposals.md` §P7), which cites **issue #657**
(safe procedure-time cuts: O2 = binary inline-candidate, L2 = skill-side skip-when-empty; L3a is the separate
ingest-sweep dedupe) as its enabler. This ask **revives §P7 and generalizes** it — and
**refines P7's cut line**: P7 said "skip Step-2 paging"; we instead *keep* Step-2 matched-note retrieval (the
win-nucleus) and cut the **write** side, reducing Step-2 cost via fewer phrases rather than skipping it.

## Where recall's ~190s goes (measured — notes 91/93; `recall-cost-isolation.md`)

Steps refer to `skills/recall/SKILL.md`. Note: `--lazy-chunks` is **already recall's default for both rungs**,
so payload-laziness is *not* the glance/deep axis — phrase-count and the write-side are.

| Slice | ~share of ~190s | glance | knob |
|---|---|---|---|
| `engram query` binary | ~3.5s (≈2%) | keep | already cheap |
| Step 0/1 — plan + **10 phrases** | ~17% | **shrink** (fewer phrases) | query-count (skill) |
| Step 2 — page the matched payload | ~50% (~95s) | **shrink** (fewer phrases → smaller matched set; min `--recent-fill`) | phrase-count + `--recent-fill` |
| Step 2.5A/2.5B + 2.7 — read candidates, recency-resolve, activate | (within the rest) | **keep** (read-side win-nucleus) | — |
| Step 2.5C/2.6 + Step 4 — coverage writes, linking, synthesis-persist | ~9–33% | **drop** (the write side) | crystallize-or-not (skill) |
| Step 3 — synthesis output | ~8% | keep | read-side |

*Shares are approximate and **not a clean partition that sums to 100%**: the write-side (2.5C/2.6/4) is
**conditional** — it fires only when there's something to crystallize — so it ranges ~9% (typical) up to ~33%
(heavy crystallization), and the read-side middle row absorbs the remainder. Item 1b measures the per-rung
split precisely.*

So glance's savings are **both** a smaller Step-2 read (fewer phrases) **and** dropping the write side — not
mostly one. **Dollars are not the lever** (notes 77/79/100): a −61% payload cap moved end-to-end time/$ by
~nothing (bytes are cheap cache-reads; the build loop is the $ sink). glance buys **wall-time per fire**.

## The design — a 2-rung dial via a read/write split

The non-arbitrary cut is **read-side vs write-side**:

- **`glance` (cheap default, for firing often):** run `skills/recall/SKILL.md` Steps 0–3 **with fewer phrases**,
  including **2.5A** (read surfaced candidates), **2.5B** (apply the recency weight so *this* decision uses the
  superseding lesson — a validated read-side capability, ROADMAP win-nucleus), and **2.7** (activate the notes
  it used, or recency-competition breaks — SKILL.md red-flag). **Drop only the write side:** 2.5C coverage
  amend/learn, 2.6 cross-cluster linking, Step 4 synthesis-persist. Surfaces matched notes and applies Step-3
  conventions-as-requirements.
- **`deep` (current full body):** all 10 phrases + the full write side (2.5C/2.6/4 crystallization, linking,
  synthesis). Fire when the decision is weighty/irreversible, or when `glance` flags an uncovered signal.

**Possible 3rd rung (`consider`), data-gated:** start at 2 rungs; **Item 1 decides** whether a middle rung (or
a finer/continuous grain) is warranted. We don't assert "a slider is over-engineering" up front — we let the
measurement say.

**On the axes — owning a design choice (vetoable).** Joe named five candidate *independent* dials —
turns-back, notes, chunks, query-count, crystallization-depth. We expose them as a single **read/write
diagonal** (glance↔deep), **not the full grid**, because cheap firing is inherently read-only, so the knobs
co-move along that seam. This is a deliberate simplification, not a given: **Item 1 surfaces whether any
off-diagonal rung** (e.g. many-queries + shallow-write, or few-queries + full-crystallize) **is actually
wanted**; if so, we add it. The three named flags map as: `--n-chunks/--n-notes` → payload
(`--content-budget`/`--lazy-chunks` ship; notes are never capped, so `--n-notes` has no analogue — Item 1 says
if it's wanted); **`--n-turns-back` → the recency-channel depth** (`--recent-fill`, ships — a loose analogue:
it bounds newest-by-ingest *chunks*, not literal turns), which glance minimizes.

## Corrections to the naive "add `--n-X` flags" version

1. **The cost is in the skill procedure, not the binary.** Payload `--n-X` flags barely move cost (see §Where
   recall's ~190s goes) and mostly already exist. The dial is primarily a **recall-skill restructure**
   (phrase-count + crystallize-or-not), with #657's safe cuts folded in (O2 inline-candidate content — binary;
   L2 skip-2.5C-when-empty — skill-side).
2. **Shallow only wins if it still *delivers*** — measure by knowledge-delivery outcome (note 119), and measure
   **escalation** too ([[feedback_validate_lazy_retrieval_by_measuring_fetch_behavior]]): glance→deep beats a
   bulk dump only if the agent escalates **rarely** AND **correctly**. Cutting phrases to widen cheapness
   re-opens a **measured single-tier dead-end** (ROADMAP "Dead ends": "cutting the 10 query phrases — breadth
   surfaces the un-guessable notes"); the two-rung escalation model is the *only* thing that licenses
   re-opening it, so phrase-count is an **Item-1 measured parameter**, not an asserted 2–3.
3. **Keep the read-side win-nucleus** (note 100): glance retains Step-2 matched-note retrieval, **2.5B
   recency-resolution**, Step-2.7 activation, and Step-3 conventions-as-requirements; only the write side is
   safe to drop.

## Decomposition — 3 items, measure → build → ship (gated, not parallel)

**Item 1 — INVESTIGATION (measure-first, cheap, no build): "Does `glance` deliver, and does escalation work?"**
Approximate `glance` *today* (fewer phrases, drop the write-side steps) vs `deep` (full body), sweeping
phrase-count as a variable. Measure, per decision-moment class (declaring-done / unexplained-failure /
new-approach): (a) **knowledge-delivery** — 3-condition blind test (none / glance / deep): does the agent's
plan apply the lesson? (note 119); (b) **cost/wall-time** per rung via the `recall_cost` $METER (note 102),
then **re-derive the viable firing ceiling** at a pinned fire-unit (note 109); (c) **escalation** — when glance
is insufficient, does the agent escalate to deep, and does it *stay* glance otherwise?
**Pass-bar:** ship glance-as-default for a moment-class only if glance delivers **within ~1 case of deep** on
that class, escalation has **no false-negatives** (never misses escalating when glance was insufficient), and
glance **stays the rung in the majority** of fires (the lazy-retrieval break-even). **Gates Item 2's default.**
Reuses the $METER, the trap gate, and the recall-moments scenarios (`2026-06-29-recall-moments-revalidation-data/`).

**Item 2 — BUILD: "Recall glance/deep modes" (avoids the model-*tier* name).** Formalize `glance`/`deep` as a
recall-skill mode + land #657's safe cuts (O2 binary + L2 skill-side). Ship `glance`-as-default **only for the
moment-classes Item 1 validated**; others keep `deep`. **Pass-bar:** the C3/C4i/C5/C6 trap harness stays GREEN
— the read-side win-nucleus (incl. 2.5B recency-supersession) must not regress. On ship, scrub the downstream
docs (recall `SKILL.md`, the c1 recall sequence diagram + flowchart, ROADMAP) per the "doc-scrub is part of an
architectural change" rule.

**Item 3 — GUIDANCE: lower the decision-moment *cost* bar (not the value aim).** Depends on Item 2. Revise the
shipped `~/.claude/CLAUDE.md` recall guidance: fire a `glance` at the cues *more readily*, because the quick
rung is cheap. **But the cost-filter is also a value gate** — memory is net-negative on non-idiosyncratic work
even when cheap (commit `f0213f6d`: warm +182s/+$3.08 on easy builds, beyond noise; note 99). So glance still
fires only where memory is **plausibly a clean win** (idiosyncratic, unloaded content); cheapness *lowers the
bar proportionally*, it doesn't remove it. Escalate to `deep` when glance flags coverage or the call is weighty.
On ship, scrub the Track-A "✅ SHIPPED — recall at the decision moments" entry's cost-filter wording to match
the relaxed bar.
**Pass-bar:** the headless RED/GREEN ([[feedback_headless_not_subagents_for_insession_guidance_revalidation]])
shows the revised guidance still flips the control (glance fires at the cues) **and** the lowered bar does not
induce firing on clearly non-idiosyncratic decisions (a net-negative-regression check on easy-build scenarios).

Build order is **linear and gating** (measure → build → ship), not parallel: each item's result shapes the
next. Merging 1+2 builds before measuring (violates notes 109/119); merging 2+3 conflates the binary/skill
change with the global-guidance change (different artifacts, different gates).

## The load-bearing open question

Does `glance` deliver the decision-relevant lesson often enough — and does the agent escalate correctly and
sparingly — that frequent cheap firing beats occasional expensive firing, at a firing ceiling we can actually
derive from glance's measured cost? Item 1 answers it before any build. If glance under-delivers, or escalation
is unreliable, the over-fire scoping from 2026-06-29 was right and this dies cheaply. If it delivers (plausible
— the read-side, including recency-resolution, *is* the win-nucleus; only crystallization is write-side), the
dial is the better lever and the guidance's *cost* bar relaxes toward Joe's "fire frequently" — within the
value aim that still holds.
