# Engram Roadmap

Shipped work: `docs/FEATURES.md` · results: `dev/eval/LEDGER.md` · decisions: `docs/architecture/adr.md`

## Where we are

Retrieval ranking quality is settled — the matched-note floor closed the note-vs-chunk drowning gap
(`docs/FEATURES.md` — Matched-note floor) and diagnostic symptom→cause surfacing checks out separately
healthy (`dev/eval/LEDGER.md#diagnostic-surfacing-probe`). Two live frontiers remain: **Track A** — recall
fires at the right moments, and what surfaces actually gets applied — and **Track B** — the token/dollar/
wall-time tax memory costs. **Track C** (Q&A memory) is a newer, separate capture axis with its own gated
rounds. A lever counts only if it moves a real axis — quality by the retrieval probe + value test + trap
gate; cost by actual tokens/dollars/wall-time, never by relocating work off the perceived path (vault note
100). Do one at a time, ship each gated, measure, then take the next.

> Shorthand codes used below — the capability axes **C1–C7** and the capture guards **G1–G6** — are
> defined in `docs/GLOSSARY.md` (see "capability axes" and "capture guards").

## Standing constraint (non-negotiable)

Every recall/learn skill change ships gated by the trap regression harness (`dev/eval/traps/gate.py`, run
before+after) and measured by the `recall_cost` `$METER` (cumulative harness, schema v5). Never touch the
win-nucleus: the Step-3 conventions-as-requirements directive, Step-2.5B recency-weight, Step-2 matched-note
retrieval, and the frontmatter `description` field. The 2026-06-28 matched-note floor is the one deliberate,
gated exception — it restores the nucleus the drowning was eroding (`dev/eval/LEDGER.md#matched-note-floor`;
exception rationale: `docs/superpowers/plans/2026-06-28-note-vs-chunk-ranking.md`, deleted 2026-07, git log).
A new edge type (typed links, graph expansion) must clear the same bar the shipped supersession-only design
already meets: don't add one without demonstrating its own retrieval value first — currently an informal
practice, now a standing rule.

## Track A — Recall timing & coverage

Ranking quality is settled; the open lever is *timing and application* — does the right knowledge surface
at the right moment, and does it get acted on.

### Residuals

- **#665 — value-gate over-fire (closed, not-planned 2026-06-30).** The wording-based value gate meant to
  scope decision-moment recall firing to idiosyncratic content does not hold on opus — it fires on routine
  work regardless of phrasing. Joe closed this as an accepted cost, not a problem: over-firing the cheap
  `glance` rung is fine; under-firing is the real risk. **Reopen condition:** only if cheap over-fire is
  later shown to be a measured problem (e.g. `glance` itself stops being cheap, or over-fire is shown to
  displace something load-bearing).
- **C5 recency-apply follow-up.** `/recall glance` surfaces a recently-updated standard but does not
  reliably apply it (`dev/eval/LEDGER.md#glance-fails-c5-delivery`); C5-type cues currently escalate to
  `deep` as the mitigation. Lifting both rungs' apply rate above deep's own is a separate, unscheduled
  follow-up.
- **Ranking follow-ups — only if the matched-note floor proves too blunt.** If the shipped floor
  (`docs/FEATURES.md` — Matched-note floor) later caps a relevant note or promotes a marginal one, the
  principled successors are per-population score normalization (z/rank-normalize notes vs chunks before
  merge) and a two-channel notes/chunks split (separate budgets, never compete). **Chunk-down-weight**
  (damping low-density chunk types like turn-1 dispatch prompts) is separately parked — the floor made its
  original drowning rationale moot, and it has a real downside (a dispatch prompt is sometimes the right
  recall) with no gauge for its intended benefit yet (vault note 121). **Write-time importance scoring**
  (rate a note's likely future value at write time, Generative-Agents-style) is parked on the same trigger.
  **Revisit condition (all three):** only once a chunk/note-quality gauge exists and shows a real drowning
  case — none of the three acts until then.
- **Question-anchored crystallization — parked (no delivery benefit, clear retrieval loss).** Anchoring
  notes to the prompting question instead of the topic was evaluated and parked
  (`dev/eval/LEDGER.md#qanchor-park`). One untested sub-lever survives as a hint, not a result:
  question-anchoring may help transferable-**pattern** lessons specifically while hurting concrete-API
  lessons (`dev/eval/LEDGER.md#qanchor-pattern-type-sublever`, n=3, not independently validated).
  **Revisit condition:** only if crystallization quality resurfaces as a bottleneck — every crystallization
  lever tried so far (handle-wording, question-anchoring, synthesis persistence, graph expansion) has nulled
  on delivery.

### Atoms-arc residual triggers

The skills-from-atoms decomposition (write-memory worker, capture guards G1/G2/G6) shipped
(`docs/FEATURES.md` — Write-memory worker + capture guards; the stop-at-the-write-seam disposition is
`docs/architecture/adr.md` ADR-0015). Three pre-registered upgrade triggers remain live:

- **G6→G5** (enforced escalation gating) fires if any future escalation ships a measured claim with an
  absent or dishonest validity line.
- **G2→G3** (a fresh-context lessons reviewer) fires if a future capture-blindspot audit's "no lesson"
  mapping is shown wrong.
- **G4** (crystallize-on-discovery) stays parked — no trigger defined; revisit only if a future
  capture-blindspot review calls for real-time write-on-discovery over the current end-of-cycle audit.

### Capture-quality residuals (unexplored)

None of these are scheduled; each needs its own shown problem, not a hypothetical one, before it's worth a
design pass (source: a 2026-07-02 research survey diffed against this roadmap — `docs/design/2026-07-02-research-followups.md`,
deleted 2026-07, git log):

- **Two-track note structure** — split bug-track (what-didn't-work + prevention) from knowledge-track
  (when-to-apply + examples) note schemas, instead of one undifferentiated format. Revisit if note triage
  shows the two purposes actively fighting each other in one schema.
- **Write-time overlap detection** — a dedup check at `engram learn` write time (problem/root-cause/
  approach/files/prevention) to prompt update-vs-create; tag nomination only helps at retrieval time today.
  Revisit if duplicate/near-duplicate notes are shown to cost retrieval quality or triage effort.
- **Post-write discoverability check** — after a note write, verify CLAUDE.md/AGENTS.md still surfaces the
  relevant guidance. Revisit if a shipped guidance change is later found to have silently stopped
  surfacing.
- **Structured staleness sweep (`engram refresh`)** — a periodic Keep/Update/Consolidate/Replace/Delete pass
  over aging note *content*, distinct from the shipped vocab-refit lifecycle (which re-fits tags/centroids
  on drift triggers, not per-note content staleness). Revisit once vault scale or note age makes content
  staleness a visible problem.
- **Scratch-artifact pattern for learn** (a subagent writes full output to a scratch file; the orchestrator
  reads path-only, to prevent summary-collapse) — likely moot: recall/learn already moved to inline
  single-skill judgment with a write-memory worker, not fan-out synthesis subagents. Revisit only if a
  future design reintroduces subagent fan-out into capture.
- **Success-phrase implicit auto-capture** — fire a capture automatically on conversational success-phrases
  ("that worked", "fixed"), distinct from `please`'s fixed-step learn call and `learn`'s explicit trigger.
  Revisit if the existing explicit/step-gated capture moments are shown to miss a meaningful share of
  substantive answers.
- **Vault isolation-rate benchmark + an "L7" episode/provenance tier** — a Luhmann-style isolated-notes
  health floor; no such check exists today. Revisit if isolated notes are suspected to indicate a
  crystallization or linking-quality problem.
- **Deferred/periodic linking-review pass** — a batch review linking recent notes to older content,
  distinct from write-time-only tagging. Revisit if tag nomination's write-time-only linking is shown to
  miss links recoverable in hindsight.
- **Note-splitting/atomicity pass** — split composite multi-lesson notes so each ranks independently;
  write-memory already writes one representative note per cluster but never audits/splits existing
  composite notes. Revisit if composite notes are shown to rank worse than atomic ones.
- **Periodic MOC-generation pass** — an LLM pass generating curated hub notes for recurring topics, the
  practitioner-validated alternative to the shipped automated tag-hub (vocab tag nomination). Revisit if
  tag nomination is shown insufficient as the bridging mechanism.

### Deeper arc — relational synthesis (unexplored extensions)

The retrieval half of this arc resolved 2026-07-02 — controlled-vocab tag nomination beat graph traversal
outright (`docs/architecture/adr.md` ADR-0011). These are further, unbuilt extensions the same 2026-07-02
research survey raised (`docs/design/2026-07-02-research-followups.md`, deleted 2026-07, git log):

- **Write-time link enrichment** (A-Mem-style "memory evolution") — retroactively update *neighboring*
  existing notes' tags/keywords when a new related note is written; the shipped vocab system tags only the
  new note. Revisit if vocab tags are shown to go stale between refit cycles specifically because of new
  neighboring notes (distinct from the refit lifecycle's global drift trigger).
- **Bi-temporal supersession edges** (4-field: t_valid/t_invalid/created_at/expired_at) — the shipped
  `--supersedes` is binary (type + claim), not time-windowed. Revisit if a query anchored to a specific past
  time is shown to need the superseded note surfaced on purpose.
- **Co-retrieval logging / association-scorer (AAR)** — log which notes get retrieved together for the same
  task and approximate an association scorer, a zero-curation alternative to tag nomination. Revisit if tag
  nomination is shown to miss notes that co-occur in practice but share no vocabulary term.
- **GRAFT-style targeted edge repair** — diagnose *why* a specific retrieval missed a note (no edge / hub
  dilution / incomplete extraction) and surgically repair just that edge, instead of tag nomination's
  blanket mechanism. Revisit if a systematic (not one-off) missed-edge pattern is diagnosed.

### From the 2026-07-01 system review — recall-timing items

The system review (`docs/design/2026-07-01-memory-system-review.md`, deleted 2026-07, git log) ranked
several still-open explorations; its fully-resolved items are omitted here (they already landed in
`docs/FEATURES.md`/`docs/architecture/adr.md`).

- **Open question — decision-moment recall as a deterministic hook.** Distinct from the already-rejected
  blanket "recall before every tool call" hook, refuted as a fatal over-firer
  (`dev/eval/LEDGER.md#recall-overfire-hook-rejected`): a *narrower* hook scoped to the two highest-value
  moments (before-declaring-done, after-a-tool-failure) was flagged as unfiled and unexplored by the review.
  Not yet decided by Joe — the shipped CLAUDE.md **recall-firing** guidance (`recall.md`) already covers these moments non-mechanically.
  Revisit only if the guidance-based cues are shown insufficient at scale.
- **Filed — recall-before-recommend re-entry (#654, #655).** Recall fires once, keyed to the incoming ask;
  a lever invented mid-synthesis is never re-checked against the vault, so a previously-killed direction can
  resurface as if new. The fix (#655: a second, lever-keyed `engram query` mid-synthesis before shipping a
  recommendation) and its regression harness (#654: a C7 "lever-recheck" anti-amnesia eval) are both filed
  and open — schedule the filed work, don't re-derive the design.

## Track B — Retrieval cost

The token/dollar/wall-time tax memory costs. Operating premise (from the 2026-06 cost-axis analysis;
see `dev/eval/LEDGER.md#payload-prune-smoke` and git history for the cost-reanchor docs): payload *size*
is cache_read-cheap (it moves time/paging, not dollars); the dollar lever is pruning the payload out of
build context after Step 3, smoke-validated but not yet productionized.

### Payload-prune production build ← NEXT

The smoke-validated isolation premise (`dev/eval/LEDGER.md#payload-prune-smoke`) is unblocked — concurrency
and write-safety shipped (`docs/architecture/adr.md` ADR-0013) — but not yet built as a product. Design:
`docs/design/2026-07-01-engram-recall-subprocess-design.md` — a new `engram recall` command that shells to
`claude -p` so the raw query payload never enters the caller's context at any nesting depth (an Agent-tool
subagent can't reach a leaf's own first-step recall, hence the subprocess). Open forks recorded in the spec:
glance-inline vs subprocess, sub-recall model/tier, return-path fidelity. Also touches recall's inline
crystallization (Steps 2.5C/2.6/Step-4).

### #657 remaining cuts (L3a/O1)

Two of #657's four procedure-time cuts remain open — O2 (inline `candidate_l2s` content, commit
`e79d8b37`) and L2 (empty-cluster skip, already in the skill) shipped per #657's comment thread. **L3a**
(batch the learn-ingest sweep once per session, without deferring crystallization) and **O1** (tighten the
chunk content-budget without starving Step 2.5's full-content read) are still open, each C7-gated. Blocked
by #654 (the C7 harness itself).

### Parked — matched-set clusters-first / lazy-content payload restructure

The recent-fill and lazy-chunks cuts were the safe payload reducers; they do NOT close the structural
clusters-first / lazy-content restructure of the matched set — an estimated ~40–80s further time/paging
win (estimate, never measured; smaller than the tier-discount lever). **Revisit condition:** only if
recall paging time becomes the complaint after the shipped cuts.

### Dedupe the double ingest sweep

Recall and learn each run `engram ingest --auto`; collapse the redundant pass. Mechanical, unscheduled.

### From the 2026-07-01 system review — cost items

- **Harder-builds eval.** The easy-build regime measured memory as a net tax
  (`dev/eval/LEDGER.md#c1-c2-warm-op-negatives`); the regime where multi-round convergence should dominate
  the tax was designed but never run (Joe specified a cross-repo corpus — spaced-repetition, file-sync,
  spec-review histories — 2026-06-25). Open candidate: 2–3 hard multi-round builds, warm vs cold, metered.
- **Between-session consolidation pass (gated).** A batched dedup/contradiction-sweep/decay pass over the
  vault, sleep-time-compute-shaped — a field-wide pattern this system doesn't have. Joe rejected async-learn
  as relocation-not-reduction (moving cost off the perceived path isn't cutting it); this pass is the same
  trap unless it clears a stricter bar: it must *reduce* total spend (e.g. replace N per-session
  crystallizations with one cheaper batch) or measurably shrink recall payloads, gated by the trap
  regression + `$METER`. Treat as reopening a settled park — requires demonstrating that new fact first, not
  just proposing the pass.

## Track C — Q&A memory, rounds 2/3

Round 1 (capture) shipped (`docs/FEATURES.md` — Q&A memory round-1); round 2/3 gate on real accumulated
pairs, per `docs/architecture/adr.md` ADR-0012.

### Round-2 gate — validate the capture instrument

Gates open at ≥20 captured pairs or ~2026-07-17, whichever comes first. Two pre-registered checks re-run on
real data, same branch sets as round 1's probes (source: `docs/superpowers/plans/2026-07-03-qa-memory-exploration.md`
and `docs/design/2026-07-03-qa-memory-proposals.md`, both deleted 2026-07, git log):

**P2′ — attribution fidelity** (does cite-derived attribution — the `[[basename]]` links actually written
in an answer — beat free-listed attribution?):

- PASS: cite-derived confabulation rate < 20% AND free-list confabulation rate > 30% AND separation ≥ 15pp
  → cite-derived is the channel.
- BOTH > 30%: even cite-derived confabulates → revise the capture bar before any further build.
- BOTH < 20%: cite-derived is validated-accurate → adopt it; free-list is not refuted this run, label
  inconclusive/tier-specific.
- ANY OTHER RESULT: BORDERLINE — report both rates; adopt cite-derived only if its rate is < 20%, with the
  caveat recorded (weak separation, middle-band free-list, or inverted ordering as applicable).
- RECALL-BORDERLINE (separate axis): cite-derived recall (coverage of actually-used notes) < 50% → the
  "cites ≥1 vault note" capture bar misses too many contributors; consider an enrichment step.

**P3′ — usage-distribution spread** (would the contribution in-degree signal discriminate for triage?):

- PASS: top-10% of notes by in-degree receive ≥3× the median → the signal has spread; build the
  retention/triage consumer.
- FAIL: distribution is flat (CV < 0.5) → the signal is uninformative; defer the consumer until more Q&A
  nodes exist naturally.
- INFORMATIVE-NULL: fewer than 20 Q&A-eligible exchanges found → underpowered; note the floor and defer.

**Arm V (q-space channel premise) at larger n** (≥30 paraphrases, to settle the round-1 BORDERLINE result —
`dev/eval/LEDGER.md#qa-arm-v-borderline`). Original pre-registered bands (n=10): **PASS ≥8, BORDERLINE 6–7,
FAIL <6** (paraphrases ranking their own Q-note first among Q-notes and above every content note); the
larger-n gate applies the same proportions — PASS ≥80%, BORDERLINE 60–70%, FAIL <60%.

### Round-3 scope — gated on round-2 licensing

- **`engram usage report`** (sorted per-note contribution in-degree, for retention/triage) builds only if
  P3′ shows spread (PASS above).
- **The dedicated Q-channel + `answered_by` ride-along** builds only if Arm V's larger-n check reaches its
  PASS bar.
- **Ranking A/B falsifier (deferred, sketch only, not scheduled):** arms = warm recall vs warm recall +
  usage-count boost in ranking; population = the 48-case miss set + trap suites; metric = knowledge
  delivery, not item rank; falsified if delivery does not improve ≥2σ while collateral stays 0. Exists so a
  future ranking ambition has a pre-registered bar, not a fresh one.

## Infrastructure — prune must preserve memory across source deletion (#659)

`engram prune` currently orphan-deletes chunks whose source file is gone — but the embedded chunk is the
asset, not the source `.jsonl`. This blocks reclaiming ~1.3 GiB of restored cross-repo transcripts in
`~/restic-restore-claude/` (deleting them would lose the recovered imptest/glowsync/targ/traced memory).
Open: decouple chunk lifetime from source-file existence — never GC valuable chunks just because the source
vanished (detach/archive vs delete; explicit-purge-only). Once fixed, delete the restore dir to reclaim the
space.

## Dead ends / not pursuing (measured or pre-registered — do not relitigate)

- Whole-op or split **haiku** recall+build: `dev/eval/LEDGER.md#haiku-whole-op-dead-end`,
  `dev/eval/LEDGER.md#haiku-split-dead-end`.
- Payload-size caps *for dollars*, cutting the 10 query phrases, and lightening the skill *body* to raise
  firing rate — all settled (2026-06/07 cost-axis + recall-trigger analysis, in git history): bytes are
  cache_read-cheap (the dollar tax is carrying the payload, not its size — same premise as Track B above,
  `dev/eval/LEDGER.md#payload-prune-smoke`); breadth in the query phrases surfaces the un-guessable notes;
  firing rate is set by the skill `description`, not its body.
- **Link-prediction (TransE/RotatE)** for the vault graph — an explicit pre-registered guard from the
  2026-07-02 research survey (`docs/design/2026-07-02-research-followups.md`, deleted 2026-07, git log): do
  not invest in an LP-predicted edge-fabric variant without a downstream retrieval benchmark demonstrating
  headroom first. No such benchmark exists; nothing to revisit until one does.
