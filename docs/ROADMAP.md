# Engram roadmap — recall efficiency

Driving recall efficiency for **real resource reduction** — actual tokens, dollars, and wall-time. A
lever counts only if it moves one of those axes. Relocating work off the *perceived* critical path (e.g.
async `learn`) hides cost without reducing it — **explicitly out of scope**. "Call recall more often" is
description-gated, not body-gated (note 100), so a lighter body is not a usage lever either. Each lever
below is tagged with the axis it actually moves; **do one at a time**, ship each gated, measure, then
take the next.

## Where we are

- **Retrieval quality was the real bug (fixed 2026-06-28).** A probe found engram's embedder is fine
  (nuanced note recall@5 0.81 in isolation) but the unified ranking **drowned notes under chunks**
  (real-path 0.19). The **matched-note floor** (`capWithNoteFloor`, commit `33821e64`) closed the gap to
  **0.83** (the embedder's ceiling), trap gate GREEN. See `docs/design/2026-06-28-retrieval-probe-results.md`.
- Memory's value is **validated and generalizes**: the 4 capability wins (apply-conventions,
  recency-supersession, honor-standard, abduction) hold with zero degradation under a realistic
  200-note crowded vault (2026-06-26). Value is real on idiosyncratic content; cost/speed is the tax.
- **Recall is the tax** (measured): ~150–190s/op, of which **~half (~80–120s at the original ~200 KB) is the agent paging the
  `engram query` payload** (~8 reads; trimmed to ~97 KB by the shipped payload cuts, so paging cost scales down); the binary itself is only ~3s. Recall is ~20% of op $
  and ~25% of op time. **Decompose by axis (notes 100/92):** the TIME tax is the procedure + paging; the
  DOLLAR tax is *carrying* the payload in build context across turns (not its size — bytes are cheap
  cache_read), so the only verified $ lever is pruning it after Step 3.

## Standing constraint (non-negotiable)

Every recall/learn skill change ships **gated by the trap regression harness**
(`dev/eval/traps/gate.py`, run before+after) and **measured by the `recall_cost` `$METER`** (cumulative
harness, schema v5). **Never touch the win-nucleus:** Step-3 conventions-as-requirements directive,
Step-2.5B recency-weight, Step-2 matched-note retrieval, the frontmatter `description` field. (The 2026-06-28
matched-note floor is a *deliberate, gated* change to matched-note retrieval — it RESTORES the nucleus the
drowning was eroding, trap gate GREEN; see the exception rationale in
`docs/superpowers/plans/2026-06-28-note-vs-chunk-ranking.md`.)

## Efficiency levers (ranked by the real axis they move; one at a time)

Each lever is tagged with the axis it **actually** moves. Per note 100: payload **size** is cache_read-cheap
(it moves TIME/paging, not dollars); the only verified **dollar** lever is pruning the payload out of build
context after Step 3; the **token+time** lever is shrinking the procedure itself.

### ← NEXT — payload-prune-after-Step-3  [DOLLARS — the only verified $ lever, ~$1/op]
Drop the raw ~97 KB query payload out of the build's *ongoing* context once Step 3 has synthesized the
requirements list. The real warm-over-cold dollar premium is *carrying* the payload across every
subsequent build turn — not its size (the bytes are cheap to cache-read once — note 100). The synthesized
requirements survive in context; only the raw payload is dropped. Measure with the `recall_cost` USD-meter
(unbundles recall $ from build $). Lowest-risk real-dollar win.

### Shrink the recall procedure  [TOKENS + WALL-TIME, ~186s tax]
Recall is one heavy ~287-line procedure run in full every time. Cut steps and/or route the mechanical
sub-steps (per-cluster reads, linking) to a cheaper tier to reduce the agent's actual token-spend AND the
~186s procedure tax. A "two-speed" split — a minimal quick-recall vs the full synthesis/linking machinery —
is one form. **Honest caveats:** does NOT increase usage (firing is decided from the frontmatter
*description*, not the body — note 100); and recall wall-time structurally exceeds a cold build (note 92),
so the win is shaving the tax, not beating baseline. Architectural — brainstorm the split first. Gate hard:
the body holds the win-nucleus.

**Trigger analysis (2026-06-27) — when should recall fire, cheaply?** See
`docs/design/2026-06-27-recall-trigger-patterns-and-proposals.md`. Verdict: **not** "recall before tool
calls" (~147× over-fire) — the wins are a narrow task-type trigger + a **two-speed quick-probe** (the
execution-cost half of this lever), a free note-negation **re-rank** (#655), a please **reconcile gate**
(#656), and deterministic **hooks**; ~28% of corrections are a write-side/capture ceiling no trigger
reaches. 10 proposals to evaluate (corpus is engram-only — does not auto-generalize).

### dedupe the double ingest sweep  [small compute/time]
Recall and learn each run `engram ingest --auto`; collapse the redundant pass. Mechanical.

### Parked — inline `candidate_l2` content  [NOT a cost lever]
Shipping candidate content inline would cut ~3–8 blocking `engram show` round-trips — a *latency* nicety
only. The bytes are cache_read-cheap and it ships content the agent may not read, so it is
~token-neutral-or-worse with **no dollar win** (note 100: payload size ≠ dollars). Pursue only if
round-trip latency itself becomes the complaint.

### Removed — async / non-blocking `learn`  [relocation, not a reduction]
Detaching the closing `/learn` (~61s) would move it off the *perceived* path but spends the same tokens,
dollars, and total wall-time — it hides cost, it does not cut it. Does not move any real axis. Dropped
2026-06-27 (Joe).

## Shipped — payload-size cuts  [TIME/paging wins; cache_read-cheap, so NOT dollar wins]
- ✅ **Lazy-chunk content — 2026-06-27** (`--lazy-chunks` + `show-chunk`): payload **−33.7%** (146→97 KB),
  trap gate GREEN; validated **0** chunk fetches across 13 realistic uninstructed recalls + **2/2**
  sole-source capability (no evidence drop). Agent fetches deferred chunk text on demand via `show-chunk`.
- ✅ **Recent-fill cut — 2026-06-27** (`--recent-fill`, 200→25): payload **−28%** (230→165 KB), trap gate
  GREEN, `targ check-full` clean. Cumulative with lazy-chunks: ~230→97 KB (**~−58%**).

> **Note:** the recent-fill cut was the *safe biggest single* payload reducer, done first. It does
> NOT close **#1** (the matched-set clusters-first/lazy-content restructure) — that remains the next
> structural win (~40-80s) once we decide the −28% slice isn't enough.

## Dead ends (measured — do not revisit)
Payload-size cap *for dollars* (payload is cheap cache_read); whole-op or split **haiku** (−14%, broke
the build half, rolled back); cutting the 10 query phrases (breadth surfaces the un-guessable notes);
lightening the skill *body* to increase firing (firing is set by the `description`, not the body).

## Done
- **Matched-note floor** (2026-06-28) — fixed note-vs-chunk drowning: real-path note recall@5 0.22→0.83
  (the embedder's isolation ceiling), trap gate GREEN. `capWithNoteFloor` reserves up to `noteFloorK=5`
  per-phrase slots for floor-qualified notes. Probe + value test:
  `docs/design/2026-06-28-retrieval-probe-results.md` (the probe `score_probe.py` is now a reusable
  retrieval-regression harness).
- **Crowded-vault capability eval** (2026-06-26) — the 4 wins generalize to a realistic crowded vault
  (zero degradation @ 200 notes). Bound: *same-domain competing* notes still untested. See
  `dev/eval/traps/{RESULTS.md, README.md}`.
- **Instruments** (2026-06-26) — the `recall_cost` `$METER` (schema v5) + the C3/C4i/C5/C6 trap
  regression gate. These make every lever above safe (regression-caught) and measurable.

## Adjacent direction — crystallize question-shaped notes (NEXT; audit DONE 2026-06-28)
The floor surfaces a good note — but the **crystallization audit** (`2026-06-28-crystallization-audit.md`)
found ~half of **cluster-driven** notes (recall Step 2.5) are not question-useful (40% vs 79% for
correction-driven), and real failure situations are 68% uncovered / 30% partial. **Next lever:** derive a
note's `situation` handle from the **question/failure it answers**, not the cluster topic (route cluster-driven
candidates through the learn path's question-shaping). Deeper arc: the vaultgraph relational substrate (note
68 — engram does aggregation, not synthesis). **Also parked:** the chunk-down-weight (drowning-rationale moot
after the floor; needs its own chunk-quality gauge before shipping); two-channel + per-population normalization
(ranked ranking follow-ups if the floor proves too blunt).

## Adjacent direction — learn from failures, not just corrections (ANALYSIS DONE 2026-06-28)
Mined **failure moments** from a 40-transcript stratified sample (main + subagent, 5 repos) with a
semantic adversarial-auditor detector (haiku; validated == sonnet at single-read size). Result:
`docs/design/2026-06-28-failure-eval-material.md` (data trail `…-failure-eval-data/`). **137 confirmed
failures; the shape: 77% UNCOVERED (a decision cue current recall doesn't reach) × 56% APPLICATION
(rule present, not applied) × 68% SUBTLE (no signal word — word-match would miss them).** The headline
is the **candidate new recall moments**, not new lessons: the highest-value process change is a
**before-declaring-done** recall checkpoint (~26% of the uncovered set) + a fully-deterministic
**after-tool-failure-before-retry** PostToolUse hook. ~40% of the corpus is cheaply evalable (tactical
C3/C5/C6 + a new C7 "source-grounding" axis); ~60% is behavioral (needs a rich-context harness).
**Next:** pick a candidate new moment to prototype (the two hooks above are the cheapest, highest-reach),
gated by the trap regression harness. Original direction note: `2026-06-27-mine-failures-as-eval-material.md`.

## Adjacent direction — prune must preserve memory across source deletion (#659)
`engram prune` currently orphan-deletes chunks whose **source file is gone** — but the embedded chunk is
the asset, not the source `.jsonl`. This blocks reclaiming the ~1.3 GiB of restored cross-repo transcripts
in `~/restic-restore-claude/` (deleting them would lose the recovered imptest/glowsync/targ/traced memory).
Brainstorm a prune that **decouples chunk lifetime from source-file existence** — never GC valuable chunks
just because the source vanished (detach/archive vs delete; explicit-purge-only). See **#659**. Once
fixed, delete the restore dir to reclaim the space.
