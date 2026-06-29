# Engram roadmap — retrieval quality & cost

**Retrieval *ranking* quality is largely solved (2026-06-28).** The note-floor fixed the drowning (notes were
nearly invisible in real recall; they now surface at the embedder's ceiling), and the follow-on probes found
diagnostic surfacing healthy (recall@5 0.99) and crystallization-quality a smaller lever than the audit
implied. So the *"is the right note retrievable"* question is answered — yes. The **two live frontiers** are
now: **Track A — recall *timing / coverage*** (fire recall at the *right moments*, so the surfaced knowledge is
actually present when a decision is made) and **Track B — cost** (run memory-backed reasoning on *cheaper
tiers* — the biggest $ lever found). A lever counts only if it moves a real axis (quality by the retrieval
probe + value test + trap gate; cost by actual tokens/dollars/wall-time — relocating work off the *perceived*
path is not a reduction, note 100). **Do one at a time, ship each gated, measure, then take the next.**

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

# Track A — Recall timing / coverage (is the knowledge present when the decision is made?)

Ranking quality is settled (the floor surfaces the right note; diagnostic surfacing 0.99). The open lever is
*timing* — recall fires at coarse moments (task-init, subagent recall-first, the parent brief), but failures
cluster at mid-task decision cues recall never reaches. Fire recall at the *right moment* so
the knowledge is present when it's needed.

### ✅ SHIPPED — recall at the decision moments (CLAUDE.md guidance, not hooks)
The failure-mining (`docs/design/2026-06-28-failure-eval-material.md`) found **77% of failures at mid-task
moments current recall never reaches** (top: before-declaring-done ~26%, after-tool-failure-before-retry,
before-writing-code/first-edit on a new approach). Addressed **2026-06-29 via global CLAUDE.md guidance** —
*not* hooks (Joe: hooks are harness-specific + a mechanical "recall before X" over-fires ~147×–380×, fatal at
~190s/fire; guidance lets the agent choose contextually, harness-agnostic). Three cues — **before declaring
done**, **after a failure you can't explain** (once, before guessing), **before building a new approach** —
each gated by a cost-filter ("fire only when you expect a vault-specific gotcha"), scoping firing to
idiosyncratic unloaded content (the one regime where memory is a clean win — note 99). Key wording: *recalling
is the action, not a substitute self-check*.

**Gate-A cost/over-fire review hardened it.** The first draft also carried "before a final verdict" (**cut** —
double-recalls with the please/route reviewers that already recall-first at task-init) and an unscoped "after a
tool fails / before retrying" (**tightened** to a failure you *can't explain*, *once* — a debug loop would
otherwise re-fire it per retry). The guidance knowingly trades the mechanical-hook over-fire for an
agent-judgment bet; the "use a free Stop-hook instead" alternative is out of scope (Joe: no hooks).

**Re-validation: clean RED 0/5 → GREEN 4/5** (headless `claude -p` — fresh process, fictional domains; the
in-session *subagent* method was invalid, control inherited the treatment — see `…revalidation-data/results.md`).
The single GREEN non-recall is the cost-filter correctly staying silent on an obvious-infra failure
(connection-refused → env check) while the *puzzling* failure recalled — the after-failure cue **discriminates,
not over-fires**. Data: `docs/design/2026-06-29-recall-moments-revalidation-data/`. Bound: small proxy; the
failures split ~56% *application*-class (lesson present, unapplied — the cue's target) and, on a separate cut,
~60% *behavioral* (needs a rich-context harness, out of reach of cheap evals). Direction note:
`docs/design/2026-06-28-failure-eval-material.md`.

### Residual — crystallize question-shaped notes  [QUALITY — deflated by the first wave]
The **crystallization audit** (`docs/design/2026-06-28-crystallization-audit.md`) found ~half of
**cluster-driven** notes (recall Step 2.5) are not question-useful (40% vs 79%). **But the first wave
(2026-06-28) deflated this lever** (`…question-shaped-crystallization-proposals.md` §First-wave results):
diagnostic surfacing is healthy (recall@5 0.99, **#4 dropped**); the prose-rule RED baseline *passed* (fresh
agents already write question-shaped handles when focused), so the audit's gap is a session-load/old-notes
artifact. **If pursued:** the deterministic retroactive `engram resituate` to clean up the existing
topic-shaped handles, then re-audit. Low priority — the first wave's real win (#7, weaker-model reuse) is now
Track B's NEXT.

### Ranking follow-ups — only if the floor proves too blunt  [QUALITY]
The note-floor (shipped, see Done) reserves up to `noteFloorK=5` per-phrase slots. If it proves blunt (caps
relevant notes, or promotes a marginal one), the principled successors are **per-population score
normalization** (z/rank-normalize notes vs chunks before merge) and a **two-channel** split (notes and chunks
get separate budgets, never compete). **Parked — chunk-down-weight:** down-weighting low-density chunk types
(turn-1 dispatch prompts) was the original "damp the noise" lever; the floor made its *drowning* rationale
moot, and it carries a real downside (a dispatch prompt is sometimes the right recall) with no gauge for its
intended benefit — needs its own chunk-quality gauge before shipping (vault note 121).

### Deeper arc — relational synthesis (note 68)
Engram does *aggregation* (cosine similarity), not *emergent synthesis* (compositional-join / transitive-chain
/ analogical-transfer). The substrate for the real thing is the unused `internal/vaultgraph` wikilink graph,
via graph-expanded retrieval (spreading activation / GraphRAG local search). Long arc, not next.

# Track B — Retrieval cost (the token/dollar/wall-time tax)

The original efficiency work. Per note 100: payload **size** is cache_read-cheap (it moves TIME/paging, not
dollars); the only verified **dollar** lever is pruning the payload out of build context after Step 3; the
**token+time** lever is shrinking the procedure itself.

### ✅ SHIPPED — tier-routing: memory discounts the model tier  [DOLLARS — the biggest $ lever found]
**Validated + shipped 2026-06-28** (route skill, commit `2bf959f4`; vault note 135;
`docs/design/2026-06-28-question-shaped-crystallization-proposals.md`). The finding — *memory democratizes
reasoning across model tiers*: sonnet+memory fully matched opus+memory across C3 (15/15), C4i (3/3), C6 (6/6)
while sonnet *cold* failed — is wired into `route/SKILL.md` **model-agnostically**: route by *tier* (not model
name; the roster re-fills the tiers), and **drop one tier for memory-backed units** (the model applies recalled
knowledge vs derives it). RED/GREEN: the router over-provisioned 4/6 memory-backed units to mid; the rule
discounts. Bound: measured at the deep→mid boundary; other boundaries inferred (the upgrade-if-cheaper-fails
rule is the safety net); C5 axis flaked (re-run). Whole-task downgrade — far bigger than the payload-$ lever.

### ← NEXT (cost) — payload-prune-after-Step-3  [DOLLARS — verified $ lever, ~$1/op]
Drop the raw ~97 KB query payload out of the build's *ongoing* context once Step 3 has synthesized the
requirements list. The real warm-over-cold dollar premium is *carrying* the payload across every
subsequent build turn — not its size (the bytes are cheap to cache-read once — note 100). The synthesized
requirements survive in context; only the raw payload is dropped. Measure with the `recall_cost` USD-meter
(unbundles recall $ from build $). Lowest-risk real-dollar win.

### Recall depth dial (was: shrink the recall procedure)  [WALL-TIME tax]  ← #661 DONE · #662 greenlit (real-vault de-risk: glance 2.2× faster / 46% cheaper per fire)
The "two-speed" split is now designed: **`docs/design/2026-06-29-recall-depth-dial-design.md`** — a 2-rung
**glance/deep** dial via a read-vs-write split (glance = retrieve + recency-resolve + apply, no crystallization writes;
deep = adds crystallization). It attacks **per-fire-cost** (note 109), so frequent firing becomes affordable —
*relaxing* the over-fire ceiling, not dissolving it (cheap ≠ free), with the **value** gate still holding
(memory net-negative on non-idiosyncratic work — note 99 / commit f0213f6d). **3 gated items, measure → build → ship:** (1, #661 ✅ DONE) `glance` DELIVERS C3/C4i/C6 at the verified
bars but FAILS C5 (it surfaces the recency item but applies it 0/5 vs deep 4/5 — retrieval ≠ delivery); cost
de-risked on a **real-scale vault** (glance **2.23× faster / 46% cheaper** per fire; #661's tiny-vault 1.2× was
a misleading artifact — `2026-06-29-realvault-glance-cost-662.md`). (2, #662 ✅ greenlit) build the glance/deep
modes + #657's safe cuts (O2 binary, L2 skill-side) + **route C5-type recency cues to deep**, trap-gated.
(3, #663) lower the decision-moment guidance's *cost* bar (not its value aim). **Honest caveats:** the win
is shaving the per-fire tax, not beating a cold build; and the skill's *auto-trigger* rate stays
description-driven and unchanged (note 100) — the deliberate rise in *cue-firing* is Item 3's guidance change,
affordable because each `glance` fire is cheap. Gate hard: the read-side win-nucleus (incl. Step-2.5B
recency-resolution) must not regress.

**Trigger analysis (2026-06-27) — when should recall fire, cheaply?** See
`docs/design/2026-06-27-recall-trigger-patterns-and-proposals.md`. Verdict: **not** "recall before tool
calls" (~147× over-fire) — the wins are a narrow task-type trigger + a **two-speed quick-probe** (the
execution-cost half of this lever), a free note-negation **re-rank** (#655), and a please **reconcile gate**
(#656); ~28% of corrections are a write-side/capture ceiling no trigger
reaches. (The analysis also proposed deterministic hooks — since dropped; Joe chose CLAUDE.md guidance over
harness-specific hooks.) Proposals to evaluate (corpus is engram-only — does not auto-generalize).

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

> **Note:** the recent-fill cut was the *safe biggest single* payload reducer, done first. It does NOT close
> the **matched-set clusters-first / lazy-content payload restructure** — a remaining structural *time/paging*
> cost win (~40-80s) if the −28% slice isn't enough. Smaller than the tier-routing $ lever above; pursue only
> if paging time becomes the complaint.

## Dead ends (measured — do not revisit)
Payload-size cap *for dollars* (payload is cheap cache_read); whole-op or split **haiku** (−14%, broke
the build half, rolled back); cutting the 10 query phrases (breadth surfaces the un-guessable notes);
lightening the skill *body* to increase firing (firing is set by the `description`, not the body).

## Done
- **Matched-note floor** (2026-06-28) [QUALITY] — fixed note-vs-chunk drowning: real-path note recall@5
  0.22→0.83 (the embedder's isolation ceiling), trap gate GREEN. `capWithNoteFloor` reserves up to
  `noteFloorK=5` per-phrase slots for floor-qualified notes. Probe + value test:
  `docs/design/2026-06-28-retrieval-probe-results.md` (the probe `score_probe.py` is now a reusable
  retrieval-regression harness).
- **Crowded-vault capability eval** (2026-06-26) — the 4 wins generalize to a realistic crowded vault
  (zero degradation @ 200 notes). Bound: *same-domain competing* notes still untested. See
  `dev/eval/traps/{RESULTS.md, README.md}`.
- **Instruments** (2026-06-26) — the `recall_cost` `$METER` (schema v5) + the C3/C4i/C5/C6 trap
  regression gate. These make every lever above safe (regression-caught) and measurable.

## Infrastructure — prune must preserve memory across source deletion (#659)
`engram prune` currently orphan-deletes chunks whose **source file is gone** — but the embedded chunk is
the asset, not the source `.jsonl`. This blocks reclaiming the ~1.3 GiB of restored cross-repo transcripts
in `~/restic-restore-claude/` (deleting them would lose the recovered imptest/glowsync/targ/traced memory).
Brainstorm a prune that **decouples chunk lifetime from source-file existence** — never GC valuable chunks
just because the source vanished (detach/archive vs delete; explicit-purge-only). See **#659**. Once
fixed, delete the restore dir to reclaim the space.
