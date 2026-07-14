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

## Current priorities (2026-07-08 triage)

A snapshot of what's actionable vs. blocked, from a full issue+roadmap triage. The tracks below hold the
detail; this is the ordering.

**Next:** unassigned — the payload-prune production build was **rejected 2026-07-08** (subprocess
complexity; revisit only on a viable subagent route — see Track B). Pick from Actionable.

**Just shipped:** **#655** — recall Step 3.5, the re-entry query (criterion 1): a lever conceived mid-synthesis gets its own lever-keyed `engram query` before it ships, with a forced `Re-entry:` line riding directly above the recommendation. Three iterations to a fired-path result: fire-rate 93% (14/15), honored-when-fired 14/14; the strict every-trial bar is explicitly unmet (asymptote — a prose+structure mechanism tops out below 100%), so Joe closed at 93% and filed the residual as a mechanical-enforcement follow-up (**#677** — closed won't-do 2026-07-10; 93% stands as the accepted operating point). Criterion 2 disposed SUPERSEDED on the same data (`dev/eval/LEDGER.md#c7-reentry-query-green`).

**Also recently shipped:** **`engram count`** (ADR-0018) — a read-only counting surface over the vault, separate from `query`'s similarity recall: `--group-by <attr>` (+ `--filter`) over frontmatter and `--backlinks-of <basename>` wikilink in-degree (`docs/FEATURES.md` — Count / backlinks aggregation).

**Also shipped 2026-07-10:** **#674** — route dispatch evidence, tags-based: each dispatch lands
as an ordinary recallable evidence note (`tags: [work-kind/<k>, tier/<t>, outcome/<o>]` via the
new repeatable `engram learn --tag` flag) plus an amended per-work-kind aggregate fact note
(`route-evidence-<work-kind>`, tier tallies + evidence wikilinks) surfaced by **plain recall**;
`engram count` is the tally/drowning **audit** surface only (ADR-0019). The scratch-vault
drowning gauge passed at 20 sibling evidence notes; if drowning is ever measured on the real
vault, the pre-registered remedies are (a) a "summarizes" ride-along edge or (b) demoting
evidence notes to the chunk-population ranking tier — chosen with the measured case in hand, per
the standing new-edge rule (ADR-0019 records both).

**Also shipped 2026-07-10:** **#678** — vocab→tags migration: the vocab layer moved from
`vocab.<term>.md` term-note files + `Vocab:` body-line/`vocab:` frontmatter dual-channel tagging +
a maintained `vocab.index.md` to the #674 tags convention — member notes carry
`tags: [vocab/<term>]`, term/family definitions are bare-`vocab`-tagged recallable fact notes
(`vocab-<term>-definition`, `vocab-definition`), and the index is emergent
(`engram count --group-by type --filter tags=vocab`). Assignment was preserved (not re-scored) via
the one-shot idempotent `engram vocab migrate-tags` subcommand (retired 2026-07-11, #681); the
real-vault migration verified all eleven mechanical bars exact, zero re-embeds beyond the 27
minted definitions, and trap-gate GREEN both before and after (`docs/architecture/adr.md`
ADR-0011/ADR-0019). Accepted consequence:
`count --backlinks-of vocab.<term>` now reads 0 (the vocab wikilink channel is retired) — the
ADR-0018 divergence example is annotated historical.

**Actionable now (unblocked, fleshed out):**

- **#693** (Track B — the open front, **re-scoped 2026-07-14**) — cut recall's chunk-index **scan** cost. #693 measured the within-`scan` attribution (`dev/eval/LEDGER.md#chunk-scan-vector-read-attribution`, n=3): the ~4.8 s scan is **~83% reading+decoding the 6,355 non-empty vector files (364 MB)**, only ~16% the 35k empty-file opens, ~1% vault+sidecar. So the lever is the **vector read** — consolidate files / binary-encode vectors (JSON float-array decode of ~2.4M floats dominates) / cache parsed vectors — measured-first before building. The candidate "skip the 85%-empty files" was measured and **rejected** (16% cut < the pre-registered 40% bar; the empties are a *disk-hygiene* issue, filed separately as #694 — ingest-guard + prune). (Distinct quantities: **scan ~67%** of binary wall; **within-scan vector-read ~83%**; **empty-file count ~85%** of *files* — NOT time.)
- **#658** (L) — unbundle recall's $ from `build_cost` (per-phase $ metering).
- **#644** (M) — OpenCode SQLite session ingest (restore + rewire the removed backend).
- **#672** (M) — route price table + one non-Claude-Code harness cost source (residual after the Claude Code capture).

**Eval-value chain:** #642 (cold-vs-warm harness) → #646 (e2e recency value-proof) → #648 (tune activation constants). The headless-learn blocker (#643) that fronted this chain is **resolved/closed** — learn Step 1 is now non-interactive `engram ingest --auto` (empty-vault headless run verified 2026-07-07); #642 is the real front.

**Gated (data/date/validation):** Track C round-2 opens at ≥20 pairs or 2026-07-17 · #667 deploy guidance to OpenCode (now includes `delegate.md`; gated on AGENTS.md `@import` validation) · #656 (its stated blocker — #654's harness — is resolved: the harness now exists with a RED baseline established 2026-07-08, `dev/eval/LEDGER.md#c7-lever-recheck-red-baseline`; still gated, now on a narrower gap — #656's AC calls for verification against a `/please`-orchestration variant of the harness, which this cycle's recall-only trials didn't build) · #652 recency centroid (gated on an over-surfacing eval) · #675 (Track C round-3 usage report; gated on P3′ spread PASS).

**Parked (revisit on trigger):** #671 parallel-builders (ADR-0017 chose the escalate-ladder) · #670 rubric-refit (needs accrued evidence — #674 shipped the evidence/aggregate notes
  2026-07-10; #669 closed subsumed) · #637 `--field` query flags (awaiting a forcing function) · payload-prune production build (rejected 2026-07-08 — revisit only on a viable subagent route; see Track B) · the "capture-quality residuals" and "deeper-arc synthesis" lists below.

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
- **Artifacts/file-keyed retrieval angle — PARKED (no headroom, 2026-07-13).** A free warm-vs-warm
  probe of adding a file-keyed angle to recall's 10 topic angles refuted it: ≤2/6 fixtures gained an
  actionable note the topic angles missed, below the ≥4/6 bar (`dev/eval/LEDGER.md#b-artifacts-angle-headroom`).
  Lessons are keyed by situation/principle, not filename, so a file-keyed angle re-finds what the topic
  angles already surface (corroborates note 73). **Reopen condition:** a real recall miss where a
  file-keyed angle would have surfaced a note the topic angles missed. Companion: the issue-briefing
  convention gained an independent systems/artifacts split + a prior-work/failure element the same
  cycle (memory-only; no repo change).
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
- **Shipped — recall-before-recommend re-entry (#654 RED harness; #655 criterion 1, Step 3.5).** Recall
  fired once, keyed to the incoming ask; a lever invented mid-synthesis was never re-checked against the
  vault, so a previously-killed direction could resurface as if new. #654's C7 "lever-recheck" anti-amnesia
  harness shipped 2026-07-08 with a RED baseline established — 4/5 fixtures reproduce the miss, 0/15
  re-query either turn (`dev/eval/LEDGER.md#c7-lever-recheck-red-baseline`). #655 closed criterion 1 in
  three iterations (worded honor-rule → forced output contract → contract coupled to the RECOMMENDATION
  line's adjacency): the v3 GREEN batch measures fire-rate 93% (14/15), honored-when-fired 14/14
  (`dev/eval/LEDGER.md#c7-reentry-query-green`). The strict pre-registered bar (every valid arm-A trial
  RECONCILED, all 4 fixtures at 1.0) is **not met** — one stochastic non-fire (fixture4 t1) — and an
  asymptote analysis shows a prose+structure mechanism can't reliably clear an every-trial bar (P(12/12
  fires) ≈ 42% at 93% per-trial). Joe closed #655 at 93% and filed the residual as a mechanical-enforcement
  follow-up: **#677** (recall: mechanical enforcement layer for the Step 3.5 Re-entry contract, 93%→100%)
  — **closed won't-do 2026-07-10 (Joe)**: 93% is the accepted operating point; no enforcement layer.
  Reopen only if a missed re-entry ships a closed-lever recommendation in real work (not the harness).
  Criterion 2's premise (negation-carrying notes outranked by chunks) is disposed **superseded** — it
  predates the matched-note floor, and 0 surfaced-but-ignored cases appear across 24 honest-instrument
  fired trials.

## Track B — Retrieval cost

The token/dollar/wall-time tax memory costs. Operating premise (from the 2026-06 cost-axis analysis;
see `dev/eval/LEDGER.md#payload-prune-smoke` and git history for the cost-reanchor docs): payload *size*
is cache_read-cheap (it moves time/paging, not dollars); the dollar lever is pruning the payload out of
build context after Step 3 — smoke-validated, but the production build was rejected 2026-07-08 (below).

### Payload-prune production build — ⛔ REJECTED 2026-07-08 (revisit only on a viable subagent route)

Rejected by Joe 2026-07-08: the production design got too complicated, and isolation at all nesting
depths forces launching the agent **from engram** (`engram recall` shelling to `claude -p`) rather than
running as a plain Agent-tool subagent — a subagent can't reach a leaf's own first-step recall, the
recursion identified 2026-07-01, which is what forced the subprocess form. The isolation *premise* stays
smoke-validated and is not relitigated (`dev/eval/LEDGER.md#payload-prune-smoke`: build_cost −40%, n=3,
synthesis-injection proxy; honest bounds in the row). **Reopen condition:** a viable subagent route —
isolation achieved via normal subagent dispatch (or an accepted depth-limited form) without engram
launching `claude -p`. Design record: `docs/design/2026-07-01-engram-recall-subprocess-design.md`
(status header marks the rejection). Note: the flag history on this item is a lesson — its "← NEXT" flag
was assistant-maintained and never user-ratified; this rejection is the first recorded user decision on
the build.

### #657 remaining cuts (L3a/O1) — outcome record (2026-07-12; CLOSED by Joe, time axis continues at #684)

All four of #657's procedure-time cuts are now disposed. O2 (inline `candidate_l2s` content, commit
`e79d8b37`) and L2 (empty-cluster skip, already in the skill) shipped earlier per #657's comment
thread. This cycle closed out the rest:

- **L3a — SHIPPED** (`35ba791c`, both skills deployed; trap gate GREEN before+after): the ingest
  sweep runs once per session — recall Step 0.5 and non-closing learns skip the sweep when one
  already ran this session; the closing learn always sweeps (carve-out), with an explicit
  counter-clause against the "something might have changed outside this session" re-sweep
  rationalization (a headless pressure probe caved on exactly that; held 3/3 after the fix).
  Honest cadence: a full please-cycle now sweeps **twice** (the opening learn + the closing learn; recall's Step 0.5 skips when a sweep already ran), down
  from 2–4× before. This also retires the former "Dedupe the double ingest sweep" Track B item —
  **SHIPPED (L3a, #657)**; L3a is that collapse.
- **O1 — DISPOSED SUBSUMED**: tightening the chunk content-budget is moot — under `--lazy-chunks`
  (recall's default invocation) `internal/cli/query.go:1430` bypasses the chunk-content path
  (`capChunkContent`/`ContentBudget`) entirely; and the prior measurement (vault note 79) had
  already shown the content budget is not the cost bottleneck.
- **Re-measure**: recall-only wall-time is now median 51.9 s, range 39.3–63.6 s (real-vault copy,
  n=3, directional — the ~73% drop vs the ~190 s pre-cuts prior is directional, not a controlled
  A/B; the trial prompt differs from the prior's task), small-fixture floor 39.7 s
  (`dev/eval/LEDGER.md#recall-time-remeasure`). Band-edge conservatism (the range straddles the
  60 s boundary) lands the disposition in the 60–120 s band: the pre-registered mapping
  **recommends closing #657**, with the parked clusters-first/lazy-content restructure (next
  section) named as the remaining lever — its revisit-condition stands unchanged. Recommended
  disposition only; final recorded after Joe's ack.

### #684 payload restructure — BUILT, MEASURED, REFUTED, REVERTED (outcome record, 2026-07-13)

Un-parked by Joe 2026-07-12 at #657's close ("the time reduction was the whole point. Let's try this
out."), then closed by Joe 2026-07-13 after two adverse re-measurements. Full arc:

- **Measure first (Task 1):** segmented the ~52 s baseline (`dev/eval/LEDGER.md#recall-time-split`)
  into pre-query (~17.5 s median), query in-flight (~12.2 s median), payload consumption — the
  addressable slice — (24.6 s median [24.4–30.0]), and remainder (~0 s). Census: 125.3 KB median
  payload, 12.3 KB duplicated candidate-note content, ~58 KB of `items[]` ahead of the first
  candidate. Cleared the pre-registered ≥15 s build bar.
- **Checkpoint 1 (Joe, 2026-07-12):** chose to **build Variant B** — true-lazy matched-set (all
  matched-note content in `items[]` goes path-only; content fetched on demand via `engram show`) —
  over the cheaper dedupe-only Variant A or closing measured-out (Variant A was later probed
  separately as #689 and refuted — see below).
- **Built (Tasks 2–3):** clusters-first payload ordering + withheld matched-note content shipped in
  the binary and the recall skill's consumption contract; trap gate GREEN before and after.
- **Round-1 re-measure (Task 4):** contradicted the bet. Bytes down (payload total −23.3%, disjoint
  ranges) but phase-c **up** 24.6→35.3 s [30.0–49.4] and total span up (overlapping ranges).
  Mechanism: 2 of 3 after-trials ran `engram query` unpiped, colliding with Claude Code's ~90 KB
  persisted-output truncation, forcing the agent to page the payload back out of its own session
  transcript via grep/sed/Read.
- **Checkpoint 2 (Joe, 2026-07-12):** chose to **keep Variant B and fix the skill** (iterate) rather
  than revert immediately — accepted one more round's risk.
- **Task 3b:** added a mandatory single-capture query discipline to the recall skill — redirect
  `engram query` output to a session-tmp file (never unpiped through Bash stdout), then sliced Reads.
- **Round-2 re-measure (Task 4b), the deciding run:** phase-c got **worse**, not better — median
  **58.0 s [37.5–62.0]**, the entire range above the pre-registered 26 s keep/revert bar. Mechanism:
  the capture discipline converts what was in-context reading into a 5–10-round-trip grep+Read
  fetch loop over the redirected capture file (~5–9 s/round-trip); pre-build's inline items-first
  payload let the agent read and judge directly from the query tool_result with zero fetch turns —
  the −21% byte cut (true census) could not buy back that round-trip latency. No `engram show`
  fetch ever fired in either round — nothing was missing, so the regression is pure round-trip
  overhead, not fetch cost.
- **Checkpoint 3 (Joe, 2026-07-13):** **revert**, per the pre-registered rule (phase-c median > 26 s).
- **Reverted:** commit `0ae98779` restores the pre-build payload shape across `internal/cli`,
  `skills/recall/SKILL.md`, and the three downstream eval consumers touched by the build —
  diff-empty vs the last pre-build commit (`055a07f5`); trap gate GREEN.

**Standing conclusion (scoped to what was measured):** fetch-mediated and file-mediated payload
reading (Variant B — withheld matched-note content, fetch-on-demand, capture-then-read) is a dead
lever at current scale — inline in-context reading is the measured fast path, and byte cuts do not
buy back API round-trips (`dev/eval/LEDGER.md#payload-restructure-refuted`; revisiting THAT lever
needs a new fact, not a retry). **Variant A** (clusters-first + dedupe-only, everything stays
inline — zero new round-trips), skipped at checkpoint 1, was subsequently probed as **#689** and
**REFUTED**: dedup worked (duplicated candidate-note content 10.1KB→0, total payload ~10% lighter)
but bought no consumption time (24.70s after-measure vs 21.75s baseline — the pre-registered
class-closing bar, ≥3.0s improvement AND after_max<baseline_min, failed both conditions),
confirming recall consumption is round-trip-dominated, not byte-dominated
(`dev/eval/LEDGER.md#variant-a-probe`). The **payload-shape lever class is now CLOSED** — Variant B
refuted as #684, Variant A refuted as #689 — both byte-shape levers dead. The segmented baseline
(`dev/eval/LEDGER.md#recall-time-split`) is the reference measurement for any recall-time lever.
**#690 (pre-query, refined to ~18.6 s (n=8) from the ~15–21 s n=3 estimate — the largest single
phase) is now measured and CLOSED measured-no-cut** (`dev/eval/LEDGER.md#prequery-composition`,
2026-07-13): the inner split is ~73% irreducible model reasoning (skill-read+Step-0 7.15 s +
phrase-composition 6.5 s), the one clean mechanical slice (the `engram ingest` sweep) is only
0.7 s — far below the 3.0 s bar — and Gate A found the recall skill is ALREADY a `--phrase`
template, so the compose lever's delta was near-empty; Joe chose to close it rather than spend on
the experiment. No recall behavior change; the pre-query span is
model-reasoning-bound, not mechanically cuttable — the same conclusion as consumption. **#691 (query
in-flight) is now measured (`dev/eval/LEDGER.md#query-inflight-split`, 2026-07-13) and BREAKS the
pattern:** the chunk-index I/O load (scan) dominates at ~67% of the ~7.1 s binary wall — NOT the
embedder — a real mechanically-cuttable lever. **#693 then measured the within-`scan` attribution
(`dev/eval/LEDGER.md#chunk-scan-vector-read-attribution`, 2026-07-14):** the ~4.8 s scan is **~83%
reading+decoding the 6,355 non-empty vector files (364 MB)**, only ~16% the 35k empty-file opens, ~1%
vault+sidecar — so the lever is the **vector read** (consolidate files / binary-encode vectors / cache
parsed vectors), not the empty-file count. The "skip the 85%-empty files" candidate was measured and
**rejected** (16% < the pre-registered 40% bar); the empties are a *disk-hygiene* issue (#694).
**Track B is therefore NOT closed;** the vector-read cut is the open front, **#693 re-scoped** (measure
the ceiling before building, per the #684/#690 moral; embed's 41k-vector match → ANN remains a separate lever).
#691 shipped the `engram query --timings` instrument (measurement only; DI clock, default payload
unchanged). Note: the 7.1 s binary wall is less than the 12.2 s in-session tool-span (`recall-time-split`)
— the ~5 s gap is Claude-Code/Bash harness overhead outside the binary.

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

- **`engram usage report`** (**#675** — sorted per-note contribution in-degree, for retention/triage;
  the `count --backlinks-of` primitive now ships the in-degree building block) builds only if
  P3′ shows spread (PASS above).
- **The dedicated Q-channel + `answered_by` ride-along** builds only if Arm V's larger-n check reaches its
  PASS bar.
- **Ranking A/B falsifier (deferred, sketch only, not scheduled):** arms = warm recall vs warm recall +
  usage-count boost in ranking; population = the 48-case miss set + trap suites; metric = knowledge
  delivery, not item rank; falsified if delivery does not improve ≥2σ while collateral stays 0. Exists so a
  future ranking ambition has a pre-registered bar, not a fresh one.

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
