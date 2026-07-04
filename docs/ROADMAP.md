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
**Track 0 (concurrency & write-safety) is foundational — it fixes live correctness bugs and blocks the
payload-prune production build, so it comes before both frontiers (split out + prioritized 2026-07-01).**

> **Full-system review (2026-07-01):** goals scorecard (4 ACHIEVED / 3 PARTIAL / 3 REFUTED /
> 3 UNMEASURED), external landscape, and the ranked exploration list live in
> `docs/design/2026-07-01-memory-system-review.md`.

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

# Track 0 — Concurrency & write-safety  ✅ SHIPPED 2026-07-01 (commit `f7f6b389`; #660 + #666 closed)

Split out + prioritized 2026-07-01, then **built and shipped the same day**. Correctness bugs, independent of
retrieval quality/cost, that **blocked** the payload-prune production build (the `engram recall` prune spawns
many parallel sub-recalls that write the vault + chunk index concurrently) and bit **today** — any two
concurrent `engram ingest`/`amend` runs corrupted state. **Fixed:** the existing **vault flock**
(`internal/cli/cli.go`, previously guarding only Luhmann-ID sequencing in `learn`) was extended to every
read-modify-write writer — manifest (`ingest`+`prune`, `.manifest.lock`) and vault notes/sidecars
(`amend`+`resituate`+`activate`, `.luhmann.lock`), acquired only at `Run*` entry points — plus **atomic
temp-rename** writes at all edges (one shared helper). `targ check-full` green; the bug inventory below is the
shipped fix list. **Payload-prune production is now unblocked.**

The complete RMW-writer surface (Step-2 code map + Gate-A code-alignment, which caught `prune` + `resituate`):

- **#660 — manifest lost-update + torn write (`ingest` AND `prune`).** Both `RunIngest`
  (`ingest.go:82`→`:108`) and `RunPrune` (`prune.go:31`→`:73`) read `chunks/manifest.json`, mutate it, and
  write it whole back via non-atomic `os.WriteFile`, with no lock. Two concurrent runs lose each other's
  entries; a torn write corrupts the file. FIX: flock the manifest RMW (`.manifest.lock`) + atomic temp-rename.
- **Vault-note lost-update (`amend` AND `resituate`, #666).** `RunAmend` (`amend.go:80/95/100`) and
  `RunResituate` (`resituate.go:55/65/70`) do an unlocked note read-modify-write; two concurrent writers on the
  same note lose one. FIX: extend the vault flock (`.luhmann.lock`) to both.
- **`activate` sidecar vector-clobber + torn write (#666).** `bumpLastUsed` (`activate.go:66-67`) rewrites
  the WHOLE `.vec.json` sidecar unlocked (assuming `os.WriteFile` is atomic — it is not), so it can clobber a
  concurrent amend/resituate re-embed's vectors with stale ones. FIX: flock `RunActivate` on the vault lock +
  atomic temp-rename.
- `learn` is **already** flock-safe (`writeLearnUnderLock`, `learn.go:571`) — the precedent to follow.
- **Deadlock-avoidance:** flock only at `Run*` entry points; shared write helpers (`bumpLastUsed`,
  `writeManifestFile`, `reEmbedAndActivate`) stay lock-free (amend already holds the lock when it calls them).

Plan: `docs/superpowers/plans/2026-07-01-concurrency-write-safety.md` (this `/please`). Gated by `targ check-full`
+ a concurrent-writers regression test.

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
each *originally* gated by a cost-filter ("fire only when you expect a vault-specific gotcha"), scoping firing to
idiosyncratic unloaded content (note 99) — **superseded by the #663 update below** (cues now fire the cheap
glance rung + encourage firing; the value gate was measured not to hold on opus). Key wording: *recalling
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

**#663 update (2026-06-30) — cues fire the cheap `/recall glance` rung; encourage-firing reframe.** The three
cues now invoke `/recall glance` (the read-only depth-dial rung, #662), not bare `/recall`. Glance is cheap, so
the guidance *encourages* firing — **under-recalling is the bigger risk; over-firing is fine and cheap**
(Joe's framing call). The default `/recall` stays `deep` and **still crystallizes**; only the decision-moment
cues use glance. **This item was re-measured on opus-4.8[1m] — the real model that runs this guidance; the
"0/5→4/5" above was a *different* model/run, which is why the numbers differ.** Result: cue-firing 2-3/5 (≥ the
un-guided 2/5, not regressed), all cue-fires use glance. Two honest findings: (1) the **after-failure cue never
fired** (0/2 on its two scenarios CF3/CF4, across every variant — opus reaches for direct diagnostics there; a
cue-*framing* lever, **rejected 2026-06-30** — plausibly-correct infra-discrimination, not a miss); (2) the old cost-filter's **value gate does NOT hold on opus** — it fires
recall on routine work too (3-5/5 regardless of wording: opus over-classifies trivial work as idiosyncratic, and
naming routine examples in the guidance made it *worse*, not better). Accepted because over-firing the cheap
glance rung is low-cost; the deeper value-gate problem is tracked as **#665**. Depth-dial arc (#661→#662→#663)
complete.

### Residual — question-anchored distillation  [QUALITY — ⛔ PARKED 2026-07-01: no delivery benefit + clear retrieval loss]
The **crystallization audit** (`docs/design/2026-06-28-crystallization-audit.md`) found ~half of
**cluster-driven** notes (recall Step 2.5) are not question-useful (40% vs 79%). That is a **note-wording audit**
(haiku judging whether a note *reads* question-shaped), NOT a delivery test. Its prescriptive tail — *"derive the
handle from the question, not the cluster topic"* — was the un-measured lever: recall clusters by content centroid
and discards the query phrases at the union, so notes distill the *topic*, not the *question* investigated. Designed
+ prototyped + **eval'd** 2026-07-01 (`docs/design/2026-07-01-question-anchored-distillation.md`).
**Verdict: PARK** — question-anchoring beats topic-anchoring on **neither** channel. **Application** (note
injected) topic-anchored 62% vs question-anchored 52% — B−A = −10pp, **inside the 2σ floor (±22pp) → no detectable
benefit** (a gap below noise is "can't distinguish," not a win). **Retrieval** (cosine of a concrete future
question to each note) topic-anchored won **10/10** (mean 0.52 vs 0.35) — **a clear loss** for question-anchoring.
Root cause: the **concrete idiosyncratic token is load-bearing on both channels** — keeping it
(topic-anchoring) embeds nearer a concrete future question *and* lets a downstream agent confidently apply it to the
named system; question-abstraction strips the token and loses both ways. The 40-vs-79 wording gap is real but
**delivery-inert** (note 119's "proxy moves, outcome doesn't"). The handle-**wording** prose rule stays
settled-rejected; binary re-clustering **not built** (the phrase-provenance plumbing prototype is reverted, backed
up at `docs/design/artifacts/2026-07-01-phrase-provenance-plumbing.patch`). Harness: `dev/eval/traps/qanchor_*.py`.
The first wave's real win (#7, weaker-model reuse) shipped as Track B tier-routing.

**Untested sub-lever (recorded, NOT to act on yet):** the eval hinted anchoring interacts with lesson **type** —
question-anchoring/abstraction *helped* transferable-**pattern** lessons (B 83% vs A 58%) but *hurt* concrete-**API**
lessons (A 64% vs B 39%). So a narrower lever — *question-anchor pattern-type lessons only, keep the token for
concrete-API lessons* — is untested as an isolated intervention. **Honest bound: n=3 pattern pairs, within the same
eval, not independently validated** — a hint, not a result. Parked; revisit only if crystallization quality resurfaces
as a bottleneck.

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
**Link-value exploration RESOLVED the retrieval half of this arc (2026-07-02,
`docs/design/2026-07-02-link-value-exploration.md`):** spreading activation/PPR is ⛔ KILLED on this vault
(drops non-activated baseline notes; 32–36 collateral regressions); the WINNER is **controlled-vocab tag
NOMINATION (L6×TAG)** — 54% retrieval recovery of verified misses, delivery +17.3pp overall / +50pp on
cross-domain bridges, both above 2σ, zero collateral. Supersession edges (L5×T5) proven as mechanism,
underpowered on delivery. **✅ WRITE-SIDE SHIPPED 2026-07-03:** vocab term-notes (25-term set, `vocab.<term>.md`,
dual-channel assignment at every `learn`/`amend` write), tag-match nomination in `candidate_l2s`, supersession
ride-along, typed `--supersedes` flag, live vault migrated (vocab bootstrap + 6 supersessions classified; retired
relation edges archived in `docs/design/artifacts/2026-07-02-retired-relation-rationales.md`).
See `docs/design/2026-07-03-vocab-notes-build-results.md`. Obsidian acceptance (graph hub visibility): ✅ signed off by Joe 2026-07-03.
**✅ REFIT LIFECYCLE LIVE 2026-07-03:** in-process trigger check at all three write sites (learn, amend, resituate) → `refit_pending` in `vocab.centroids.json` → `engram vocab stats` verdict line (`verdict: OK` / `verdict: REFIT_PENDING (<reason>)`) + query payload flag → learn skill Step 1.5 autonomous refit; recalibrated triggers (independent — ANY one trips the flag): growth ≥40 notes AND ≥14d since last refit; vault-wide untagged >8%; any term >25% of vault. Measured per-refit cost ≈$0.09 (2026-07-03 validation run).

### Deeper arc — rebuild the skills from behavioral atoms  [ARCHITECTURE — Joe 2026-06-29]
The skills (recall, learn, please, route) overlap; the underlying *atoms* are distinct behaviors —
**read-memory, write-memory, route-a-task, orchestrate-a-workflow (reason + adversarial-check)**. Decompose the
skills into atoms dedicated to each behavior and recompose, **without ending up with N skills that almost all do
the same thing** (Joe's explicit constraint). The glance/deep read-vs-write split (#662) is a first, small
instance of the seam this would generalize. Scope/sequence TBD — brainstorm before any build.

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

### payload-prune-after-Step-3  [DOLLARS — verified $ lever] · premise ✅ SMOKE-VALIDATED 2026-06-30 · production build ← NEXT (Track 0 shipped — unblocked 2026-07-01)
Drop the raw ~97 KB query payload out of the build's *ongoing* context once Step 3 has synthesized the
requirements list. The real warm-over-cold dollar premium is *carrying* the payload across every
subsequent build turn — not its size (the bytes are cheap to cache-read once — note 100). The synthesized
requirements survive in context; only the raw payload is dropped.

**Smoke (synthesis-injection proxy, `dev/eval/cumulative/smoke_prune.py`, `claude-opus-4-8`, n=3 apps —
`docs/superpowers/specs/2026-06-30-payload-prune-mechanism-design.md`):** carrying **only the synthesis** cut
**build_cost ~40% (~$1.6/app; feeds −45%, links −23%, notes −51%)** with **zero capability loss** — identical
rounds (2/2/2), success (3/3), final convergence + arch 10/10 on every app. The saving shows in *every* build
round (mechanistic — the payload re-reading as `cache_read`), so the ~$1/op premise (note 95) held, if anything
an underestimate. **Honest bound:** n=1/app, no same-arm noise floor measured → large-consistent-mechanism, not
noise-floor-proven; a replicate would make it conclusive (not required to proceed).

**⛔ Production mechanism — DEFERRED behind Track 0.** The smoke validated the *isolation premise* via a proxy;
it did **not** ship a product. Design captured 2026-07-01 in
`docs/superpowers/specs/2026-07-01-engram-recall-subprocess-design.md`: a new **`engram recall`** Go command
that shells to **`claude -p`** — the caller generates the queries in-context and passes only those; the isolated
sub-recall runs `engram query` + cluster-judgment + crystallization + Step-3 synthesis behind the subprocess
boundary and returns **only the synthesis**, so the ~97 KB payload never enters the caller's context, at any
nesting depth (an Agent-tool subagent can't reach a leaf's own first-step recall — hence the subprocess). It is
**blocked by Track 0**: the sub-recalls write the vault/manifest in parallel, so concurrency-safety must land
first. Open forks recorded in the spec (glance inline vs subprocess; sub-recall model/tier; return-path
fidelity). It also touches recall's inline crystallization (Steps 2.5C/2.6/Step-4).

### Recall depth dial (was: shrink the recall procedure)  [WALL-TIME tax]  ← #661 DONE · #662 ✅ SHIPPED 2026-06-29 (glance/deep modes; 2.23× faster per fire; deep default; C5→deep escalation; trap gate GREEN)
The "two-speed" split is now designed: **`docs/design/2026-06-29-recall-depth-dial-design.md`** — a 2-rung
**glance/deep** dial via a read-vs-write split (glance = retrieve + recency-resolve + apply, no crystallization writes;
deep = adds crystallization). It attacks **per-fire-cost** (note 109), so frequent firing becomes affordable —
*relaxing* the over-fire ceiling, not dissolving it (cheap ≠ free), with the **value** gate still holding
(memory net-negative on non-idiosyncratic work — note 99 / commit f0213f6d). **3 gated items, measure → build → ship:** (1, #661 ✅ DONE) `glance` DELIVERS C3/C4i/C6 at the verified
bars but FAILS C5 (it surfaces the recency item but applies it 0/5 vs deep 4/5 — retrieval ≠ delivery); cost
de-risked on a **real-scale vault** (glance **2.23× faster / 46% cheaper** per fire; #661's tiny-vault 1.2× was
a misleading artifact — `2026-06-29-realvault-glance-cost-662.md`). (2, #662 ✅ SHIPPED 2026-06-29) glance/deep
modes built (commit `bdb8b0dc`; **deep stays default**, glance is opt-in/read-only/~3-phrases, no crystallization
writes; #657 O2/L2 confirmed already landed; **C5-type recency cues escalate to deep**) — smoke trap gate GREEN
(C3/C4i/C5/C6). The deeper C5 recency-*apply* fix (lift both rungs above deep's 4/5) is a separate follow-up.
(3, #663) ✅ SHIPPED 2026-06-30 — cues fire the cheap `/recall glance` rung; **encourage-firing reframe**
(under-firing is the bigger risk); deep default still crystallizes. The cost-bar/value-gate premise was
falsified on re-measurement (→ #665); details in the Track-A #663 update. **Honest caveats:** the win
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

### ✅ SHIPPED — inline `candidate_l2` content (O2, #657)  [latency/clarity, not a $ lever]
Landed 2026-06-29 (commit `e79d8b37`): `candidate_l2s` carry `content` inline so recall Step 2.5 needs no
per-candidate `engram show`. **Honest scope:** a behavioral check showed the well-behaved agent already
cross-referenced `items[]` content (no redundant shows), so the real win is **clarity/robustness** — it removes
the skill's contradictory "show every candidate" instruction + the cross-reference burden + a latent loophole —
not a measured round-trip cut. Bytes are cache_read-cheap → no $ win (note 100). #657's L2 was already done;
**L3a (batch ingest sweep — overlaps the "dedupe the double ingest sweep" item below) + O1 (chunk content-budget) remain open under #657.**

### Removed — async / non-blocking `learn`  [relocation, not a reduction]
Detaching the closing `/learn` (~61s) would move it off the *perceived* path but spends the same tokens,
dollars, and total wall-time — it hides cost, it does not cut it. Does not move any real axis. Dropped
2026-06-27 (Joe).

# Track C — Q&A memory (capture + retrieval)

Structured capture and retrieval of question-and-answer pairs. Round 1 ships the capture path
and D5′ asymmetric participation (A-notes compete; Q-notes excluded). Later rounds gate on
measured validation over accumulated pairs.

- **[SHIPPED 2026-07-03] Q&A memory round-1 (capture):** `engram learn qa`, D5′ exclusion
  (`isQueryExcludedKind` at all four query-pipeline seam points), `stripMachineLines` QA markers,
  `qa pairs:` / `qa round-2 gate:` lines in `engram vocab stats`; recall Step 4 + learn Step 2.5
  QA capture extensions. Round-2 gate: ≥20 pairs or ~2026-07-17 (whichever first).

- **[DEFERRED — round 3, gated on round-2 validation]** The dedicated Q-channel (incoming ask
  matched against Q-note embeddings in q-space) and the `answered_by` ride-along (a surfaced Q
  delivers its paired A). Gated on the Arm V large-n eval (the q-space channel premise check; see
  `docs/design/2026-07-03-qa-memory-proposals.md`) reaching PASS (≥80% — its pre-registered
  bands; BORDERLINE does not license the build) and on P2′/P3′ post-ship validation over ≥20
  real pairs. Arm V large-n came in BORDERLINE 63% (19/30) — round 3 remains unlicensed pending
  a further check. NOT built in round 1.

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
