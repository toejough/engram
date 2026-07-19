# Engram Roadmap

Shipped work: `docs/FEATURES.md` · results: `dev/eval/LEDGER.md` · decisions: `docs/architecture/adr.md`

## How we prioritize

> The ranks below are a **proposal** — Joe's call at real decision points, not a settled decision
> (an assistant-maintained priority flag is a standing proposal, note 192).
> **Bands are readiness-slices of ONE order, not independent workstreams** — the opposite of the
> old A/B/C tracks. A single top-to-bottom priority; the band says how ready an item is to start.

A lever counts only if it moves a real axis: quality by the retrieval probe + value test + trap
gate; cost by actual tokens/dollars/wall-time — never by relocating work off the perceived path
(vault note 100). Do one at a time, ship each gated, measure, then take the next.

Score every item on five things, then place it:

1. **Axis it moves** (lean-ordered per Joe):

   | Axis | What it is | Maps to |
   |---|---|---|
   | **Dev-velocity & safety** (1st) | makes future cycles faster / less error-prone (process, hardening, staleness detection) | program axis (new) |
   | **Measurement / value-proof** (2nd) | lets us measure whether memory helps — the foundation that says if other work is worth doing | program axis (new) |
   | **Correctness / capability** (3rd) | memory quality, coverage/reach, ranking fidelity | C3–C6 |
   | **Cost / latency** (4th) | recall/learn speed, tokens/$ | C1–C2 |

2. **Impact type** (note 190): **mover** (moves a real axis) > **enabler** (unblocks; ranks via
   what it unblocks); **hygiene/defect** ranks on the HARM it stops, NOT its op-share % (note 271).
3. **Value magnitude** — High / Med / Low, by this test:
   - **High** = retro-verified or measured need, OR unblocks/gates other work.
   - **Med** = design-sound with a clear beneficiary, not yet evidenced.
   - **Low** = suspected/nice-to-have, "not a live gap."

   Value breaks ties WITHIN the axis lean; a High measurement item may cross above a Low dev item —
   annotate the crossover when it happens.
4. **Cost/risk to build** — a NEW SUBSYSTEM whose whole-op share is small leans DEFER (note 270 —
   e.g. the vector-read/decode-cache item deferred at the Ratification Gate, see Provenance below);
   a small change reusing existing code is cheap regardless of op-share.
5. **Dependency position** (hard constraint) — an item never ranks above its own blocker.

**To place:** dependencies set the BAND (no blocker → NOW; one NOW blocker → NEXT; ≥2 / transitive
/ prior-work → LATER; external trigger → GATED; Joe-rejected → DEFERRED); axis + value set the RANK
within the band.

**Down-rank within its band** (score normally, then slide it down; these need no vault context to
apply): (a) reasoning-scaffold mechanisms that add priming/structure without moving a core axis;
(b) enforcement/mechanization on an already-accepted operating point; (c) work relocated to
async/background and framed as savings (a lateral move, not a cut). *(Historical context: notes 73,
201, 108.)*

**Band definitions (mutually exclusive):**
- **NOW** — unblocked, actionable today; ranked.
- **NEXT** — blocked by exactly ONE actionable (NOW) item.
- **LATER** — blocked by ≥2 items, or transitively (blocked by a NEXT item), or on prior WORK we
  must build first.
- **GATED** — blocked on an EXTERNAL fact/date/validation-result outside our immediate control
  (a date, ≥N captured pairs, an upstream feature, an eval PASS). Auto-actionable when the trigger
  fires.
- **DEFERRED (Joe)** — Joe explicitly rejected/parked; revives ONLY when Joe weighs a NEW fact.
- **PARKED backlog** — speculative, no active trigger; kept for when a problem actually shows.
- **Provenance** — shipped / refuted / dead-end; do not relitigate.

**Keep-current rule (the ongoing process — the point of this restructure):**
1. **On a new feature/bug/task:** score it on 1–5 and insert it into `## The roadmap` at its band.
   No new parallel track — ever.
2. **On new information** (a blocker resolves, evidence lands, a mover proves to be an enabler, a
   measurement comes back): **re-score the affected EXISTING items and move them between bands** —
   the roadmap is re-placed, not just appended to.
   *(Optional future mechanization — a filing-time hook prompting "score + place per the rubric" —
   is a candidate follow-up, not built in this cycle.)*

> Shorthand codes used below — the capability axes **C1–C7** and the capture guards **G1–G6** — are
> defined in `docs/GLOSSARY.md` (see "capability axes" and "capture guards").

## The roadmap

> Ranks are a proposal for Joe's ratification (note 192). NOW/NEXT/LATER are rank-ordered;
> GATED/DEFERRED/PARKED are not-yet-actionable (ordered by trigger-proximity, not priority).

### NOW — unblocked, actionable
| Rank | Item | Axis / type / value | Why here | Deps |
|---|---|---|---|---|
| 1 | **#687** run-level self-evaluation (surprise-mining retrospective) | Dev / mover / Med | Generates follow-up issues mechanically; meta-leverage | none |
| 2 | **#658** unbundle recall's $ from build_cost | Meas / enabler / Med | Separable recall_cost unblocks the warm-vs-cold $ question + L4-at-n | none |
| 3 | **#644** OpenCode SQLite session ingest | Corr(reach) / mover / Med | Extends recall coverage to OpenCode history (currently invisible); needs restoring a removed backend | none |
| 4 | **#696** Pi session-file ingest | Corr(reach) / mover / Med | Extends recall coverage to Pi coding-agent sessions (JSONL, USER:/ASSISTANT:, RFC3339 timestamps) through the same transcript pipeline; sibling of #644 | none |
| 5 | **#670** route rubric-refit (fold accrued tier evidence into cold-start priors) | Corr / mover / Med | Unblocked — #674 already shipped the evidence notes; defining the evidence-threshold trigger is the first STEP of the work, not an external blocker | none |
| 6 | **#672** route price table + non-CC cost source | Cost / mover / Low-Med | Real $ in the route mini-report; CC capture already done | none |
| 7 | **Batchable hardening** (one writing pass): **#683** vocab byte-oracle · **#688** recall_time.py hardening · **#679** route dup-aggregate guard · **#680** route copyedit bundle | Dev / hygiene / Low | Real but low-value; each "not a live gap." Batched to amortize cycle overhead; explicitly NOT floated above the higher-value movers despite being axis-1 (value-crossover: Low value sinks them below the Cost item #672) | none |
| 8 | **#695** validate under-load recall-firing fix (placebo control · real-vault retrieval precision · attention-displacement) | Meas / enabler / High | Three caveats the adversarial verifier flagged on the engagement-led `recall.md` cue deployed 2026-07-15 (`LEDGER.md#underload-firing-wording-fix`): isolate engagement-framing from any leading-cue, validate fire→surface precision beyond 5-note fixtures, and rule out the cue displacing the other three decision-moment cues; gates the Parked → Decision-moment recall hook revive trigger. Also carries #686's folded goal — whether the task-kind cue actually fires and doesn't displace the existing decision-moment cues | none |
| 9 | **#648** tune recency constants | Meas / mover / Low-Med | Unblocked — #646 closed 2026-07-19. **Scope shifted by #646's finding:** the recent-fill CHANNEL is redundant for self-captured needs, so #648's lever is the RE-RANK bias (the proven C4i continuity mechanism — recent-relevant outranks old-relevant), i.e. tune the recency half-life / tail-weight, NOT the recent-fill count. Low-Med until a concrete drowning/staleness gauge motivates a sweep | none |

### NEXT — blocked by exactly one NOW item
| Item | Axis / type / value | Why here | Blocked on |
|---|---|---|---|
| *(none — #648 moved to NOW after #646 closed 2026-07-19; see NOW rank 9)* | — | — | — |

### LATER — blocked deeper, or on prior work we must build
| Item | Axis / type / value | Why here | Blocked on |
|---|---|---|---|
| **#656** gate analytical recommendations as outcome-keyed artifacts | Dev/Corr / mover / Med | Needs a `/please`-orchestration harness variant of #654 built first (our work, not an external gate) | build the harness variant |

### GATED — external / date / validation trigger
| Item | Axis | Trigger |
|---|---|---|
| **Q&A memory round-2** (P2′ attribution · P3′ spread · Arm V @ larger-n) | Corr | ≥20 captured pairs **or** 2026-07-17 — **date trigger FIRED, but only 7/20 pairs captured as of 2026-07-18: a run now hits P3′ INFORMATIVE-NULL (underpowered). Decision pending (Joe): wait for ≥20 pairs, or run accepting the INFORMATIVE-NULL floor** |
| **#675** `engram usage report` (round-3) | Corr | round-2 P3′ = PASS |
| Dedicated Q-channel + `answered_by` (round-3) | Corr | round-2 Arm V = PASS |
| Ranking A/B falsifier (round-3 sketch; pre-registered bar only) | Corr | a future ranking ambition invokes its bar |
| **#667** deploy recall/delegate guidance to OpenCode | Corr(reach) | OpenCode `AGENTS.md @import` validated |
| **#652** recency-weighted cluster centroid | Corr | an over-surfacing eval exists |
| **#685 Change #2** task-kind glance cue (`guidance/recall.md`) | Corr | #695's attention-displacement finding (does the shipped engagement-led cue already generalize to task-kind moments, or does a dedicated cue risk displacing the other three decision-moment cues?). Also the fold target for #686 — its doc-staleness goal, reframed from a detector (out of purview) to this recall cue (see Provenance) |

> **Q&A memory round-2 gate criteria (pre-registered, not yet run; date trigger 2026-07-17 has now
> PASSED, but only 7/20 pairs are captured as of 2026-07-18 → P3′ pre-registers INFORMATIVE-NULL below
> 20 exchanges, so a run now is underpowered; awaiting Joe's call — wait for ≥20 pairs or run accepting
> the null):** two checks re-run on real data, same branch sets as round-1's probes (source:
> `docs/superpowers/plans/2026-07-03-qa-memory-exploration.md` and
> `docs/design/2026-07-03-qa-memory-proposals.md`, both deleted 2026-07, git log).
> - **P2′ — attribution fidelity** (does cite-derived attribution — the `[[basename]]` links
>   actually written in an answer — beat free-listed attribution?): PASS if cite-derived
>   confabulation rate <20% AND free-list confabulation rate >30% AND separation ≥15pp →
>   cite-derived is the channel. BOTH >30% → even cite-derived confabulates, revise the capture bar
>   before further build. BOTH <20% → cite-derived is validated-accurate, adopt it; free-list is not
>   refuted this run (inconclusive/tier-specific). ANY OTHER RESULT → BORDERLINE, report both rates,
>   adopt cite-derived only if its rate is <20% with the caveat recorded. RECALL-BORDERLINE (separate
>   axis): cite-derived recall (coverage of actually-used notes) <50% → the "cites ≥1 vault note"
>   capture bar misses too many contributors, consider an enrichment step.
> - **P3′ — usage-distribution spread** (would the contribution in-degree signal discriminate for
>   triage?): PASS if top-10% of notes by in-degree receive ≥3× the median → build the
>   retention/triage consumer (the usage-report item above). FAIL if distribution is flat (CV <0.5)
>   → the signal is uninformative, defer the consumer. INFORMATIVE-NULL if fewer than 20 Q&A-eligible
>   exchanges found → underpowered, note the floor and defer.
> - **Arm V (q-space channel premise) at larger n** (≥30 paraphrases, settling the round-1
>   BORDERLINE result — see Provenance below): same proportions as round 1 — PASS ≥80%,
>   BORDERLINE 60–70%, FAIL <60%.
>
> **Round-3 scope (gated on round-2 licensing):** the usage-report item above builds only if P3′
> shows spread; the dedicated Q-channel + `answered_by` ride-along builds only if Arm V's larger-n
> check reaches its PASS bar; the ranking A/B falsifier (arms = warm recall vs warm recall +
> usage-count boost in ranking; population = the 48-case miss set + trap suites; metric = knowledge
> delivery, not item rank; falsified if delivery does not improve ≥2σ while collateral stays 0)
> exists so a future ranking ambition has a pre-registered bar, not a fresh one.

### DEFERRED (Joe) — do not relitigate without NEW evidence
| Item | Axis | Reason | Revive trigger |
|---|---|---|---|
| **#693** vector-read/decode cache | Cost | ~87% of scan but only ~7% of the ~52 s op; new subsystem not worth it (note 270) | decode grows to dominate a larger share of the op |
| **payload-prune production build** | Cost | rejected 2026-07-08 (subprocess complexity) | a viable subagent route (no engram-launched `claude -p`) |
| **between-session consolidation pass** | Cost | Joe rejected async-as-relocation (note 108) | proof it *reduces total spend* (not relocates it) |
| **#671** parallel-builders | Dev | ADR-0017 chose the escalate-ladder | cheapest-first ladder shown insufficient |
| **#637** `--field` query flags | Corr | large surface, no forcing function | a concrete forcing function |

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

## Parked backlog (revisit-on-trigger)

| Group | Item | Revive trigger |
|---|---|---|
| Recall timing/apply residuals | C5 recency-apply lift | a chunk/note-quality gauge shows a real drowning/miss case |
| Recall timing/apply residuals | Ranking follow-ups (z/rank-normalize, notes/chunks split, chunk-down-weight, write-time importance) | a chunk/note-quality gauge shows a real drowning case (chunk-down-weight carries its own downside — a dispatch prompt is sometimes the right recall, no benefit gauge yet, note 121) |
| Capture-quality residuals | Two-track note schema | note triage shows the two purposes actively fighting each other in one schema |
| Capture-quality residuals | Write-time overlap detection | duplicate/near-duplicate notes are shown to cost retrieval quality or triage effort |
| Capture-quality residuals | Post-write discoverability check | a shipped guidance change is later found to have silently stopped surfacing |
| Capture-quality residuals | `engram refresh` staleness sweep | vault scale or note age makes content staleness a visible problem |
| Capture-quality residuals | Scratch-artifact learn pattern | a future design reintroduces subagent fan-out into capture |
| Capture-quality residuals | Success-phrase auto-capture | the existing explicit/step-gated capture moments are shown to miss a meaningful share of substantive answers |
| Capture-quality residuals | Isolation-rate/L7 tier | isolated notes are suspected to indicate a crystallization or linking-quality problem |
| Capture-quality residuals | Periodic linking-review | tag nomination's write-time-only linking is shown to miss links recoverable in hindsight |
| Capture-quality residuals | Note-splitting/atomicity | composite notes are shown to rank worse than atomic ones |
| Capture-quality residuals | Periodic MOC generation | tag nomination is shown insufficient as the bridging mechanism |
| Deeper-arc extensions | Write-time link enrichment | vocab tags are shown to go stale between refit cycles specifically because of new neighboring notes |
| Deeper-arc extensions | Bi-temporal supersession edges | a query anchored to a past time needs the superseded note surfaced on purpose |
| Deeper-arc extensions | Co-retrieval association-scorer | tag nomination is shown to miss notes that co-occur in practice but share no vocabulary term |
| Deeper-arc extensions | GRAFT edge repair | a systematic (not one-off) missed-edge pattern is diagnosed |
| Pre-registered guard upgrades | G6→G5 (escalation gating) | a future escalation ships a measured claim with an absent or dishonest validity line |
| Pre-registered guard upgrades | G2→G3 (fresh-context lessons reviewer) | a future capture-blindspot audit's "no lesson" mapping is shown wrong |
| Pre-registered guard upgrades | G4 (crystallize-on-discovery) | no trigger defined; revisit only if a future capture-blindspot review calls for real-time write-on-discovery over the current end-of-cycle audit |
| Decision-moment recall hook | narrower before-done / after-failure deterministic hook | guidance enables firing under load: an engagement-led `recall.md` cue lifted under-load firing +50pp (50%→100%) on 5-note fixtures and is deployed (`LEDGER.md#underload-firing-wording-fix`, 2026-07-15); revisit a hook only if real-vault retrieval precision or endorse-moment firing prove insufficient at scale — see **#695** (NOW band above) |

## Provenance — shipped / refuted / dead ends (do not relitigate)

| Item | Type | Note | Relitigation bar |
|---|---|---|---|
| #655 · `engram count` · #674 · #678 · #694 · #657 (L3a) · #691 | Shipped | **#655** (recall Step 3.5, re-entry query) — closed at 93% fire-rate; see the #654/#655/#677 outcome record below.<br>**`engram count`** (ADR-0018) — read-only counting surface over the vault (`--group-by`/`--filter` over frontmatter, `--backlinks-of` wikilink in-degree); `docs/FEATURES.md` — Count / backlinks aggregation.<br>**#674** (route dispatch evidence, tags-based) — each dispatch lands as a recallable evidence note (`tags: [work-kind/<k>, tier/<t>, outcome/<o>]` via `engram learn --tag`) plus an amended per-work-kind aggregate fact note, surfaced by plain recall; `engram count` is the tally/drowning audit surface (ADR-0019); scratch-vault drowning gauge passed at 20 sibling evidence notes; pre-registered remedies on a real-vault drowning measurement: a "summarizes" ride-along edge, or demoting evidence notes to the chunk-population ranking tier.<br>**#678** (vocab→tags migration) — vocab layer moved from `vocab.<term>.md` term-note files + dual-channel tagging + a maintained index to the #674 tags convention; migrated via the one-shot idempotent `engram vocab migrate-tags` subcommand (retired 2026-07-11, #681); real-vault migration verified all eleven mechanical bars exact, zero re-embeds beyond the 27 minted definitions, trap-gate GREEN before and after (ADR-0011/ADR-0019). Accepted consequence: `count --backlinks-of vocab.<term>` now reads 0 (vocab wikilink channel retired) — the ADR-0018 divergence example is annotated historical.<br>**#694** (empty-file hygiene) — `rebuildIndex` guard (no more 0-byte writes) + `engram prune --empty` (+ `--dry-run`); real `~/.local/share/engram/chunks` pruned 2026-07-14: 54,618→6,628 files, 47,990 empties removed, 0 remaining; scan_ms 5149→4061 (−21% of the scan stage), ranking-neutral (`LEDGER.md#chunk-empty-file-prune`); the empty count had grown +12,711 since the prior measurement, confirming the guard's unbounded-growth-cap value.<br>**#657 (L3a)** — the ingest sweep runs once per session (recall Step 0.5 and non-closing learns skip when one already ran; the closing learn always sweeps), shipped `35ba791c`, both skills, trap gate GREEN before+after; honest cadence a full please-cycle now sweeps twice, down from 2–4× before; retires the former "dedupe the double ingest sweep" item. #657's O1 (chunk content-budget tightening) disposed SUBSUMED — moot under `--lazy-chunks` (bypasses the content-budget path entirely) and the content budget was already shown not to be the cost bottleneck (vault note 79). Re-measure: recall-only wall-time median 51.9s [39.3–63.6] (`LEDGER.md#recall-time-remeasure`), the ~73% drop vs the ~190s pre-cuts prior is directional not controlled A/B; band-edge conservatism (range straddles 60s) recommended closing #657, with the clusters-first/lazy-content restructure named as the remaining lever (pursued next as #684).<br>**#691** (query in-flight split + `--timings` instrument) — measured and BREAKS the pattern of the other cuts: chunk-index I/O (scan) dominates ~67% of the ~7.1s binary wall, not the embedder (`LEDGER.md#query-inflight-split`); shipped the `engram query --timings` instrument (measurement only, DI clock, default payload unchanged). Note: the 7.1s binary wall is less than the 12.2s in-session tool-span (`LEDGER.md#recall-time-split`) — the ~5s gap is Claude-Code/Bash harness overhead outside the binary. | decided — none |
| #685 (Change #1) | Shipped (staged, not deployed) | **Plan-write doc-surface enumeration grep + Gate A independent-pass clause** — Step 3 now mandates a concept-variant grep + pasted per-file disposition list when a plan alters a repeated invariant; Gate A's docs/diagrams-alignment reviewer independently verifies that list rather than deferring to it. Validated via a new headless probe harness (`please_step3_probe`): a clean single-task probe could NOT reproduce #685's failure (6/6 full enumeration, un-pressured and under pressure — salience-under-load, not a capability gap); a loaded buried-subtask repro DID reproduce it and validated the fix — RED (old text) 5/5 UNDER-enumerated (consistently dropped one off-radar doc) → GREEN (Edit A) 5/5 fully enumerated, +100pp, Fisher exact p≈0.008 (`LEDGER.md#685-doc-enumeration-grep`). Staged to `skills/please/SKILL.md`; `engram update` deploy and commit held for Joe. Change #2 (task-kind glance cue) deferred — see GATED band → #685 Change #2. | decided — none |
| #643 (headless-learn blocker) | Resolved / closed | learn Step 1 is now non-interactive `engram ingest --auto` (empty-vault headless run verified 2026-07-07) — unblocked the eval-value chain's front link (see NOW band above). | decided — none |
| #669 | Closed (subsumed) | closed, subsumed by the route rubric-refit work (see NOW band above). | decided — none |
| **#686** old-text echo detector (doc staleness at the creating commit) | Closed (out of purview) | literal echo-detector built + PARKED 2026-07-16 (git `stash`; note 285) — false-positive-swamped on this mirror-heavy repo (docs/skills quote each other, so a removed string still echoes wherever its legit copies live — up to 39 false hits/cycle); the semantic+LLM variant rejected as outside engram's MEMORY purview — a pre-merge doc-staleness detector, literal or LLM, is general dev-tooling (note 293). Goal reframed as a memory/recall problem ("remember the right docs exist when changing a shared thing") and folded into **#685 Change #2** (task-kind glance cue, retrieval-proven 6/6, `LEDGER.md#685-task-kind-cue-headroom`) + **#695** (does it fire / does it displace the existing decision-moment cues). The file-keyed retrieval variant is refuted (`LEDGER.md#b-artifacts-angle-headroom`, ≤2/6 — docs are keyed by situation/principle, not filename). | a recall-firing mechanism (the task-kind cue) shown insufficient at real scale |
| **#646** e2e recency value-proof (recent-fill self-capture) | Closed (structurally untestable) | Four clean opus pilots showed the recency **recent-fill** channel redundant with cosine for self-captured needs: a decision the agent needs is topically related to its task, so `/recall`'s broad query cosine-matches it (`direct` 0.48–0.60, zero `recent`) and the deduped recent-fill channel adds nothing — **needed ⟹ queried** (`LEDGER.md#646-recency-recent-fill-selfcapture`; notes 294/295). Recency's day-to-day **continuity value stands on the RE-RANK bias** (C4i 5/5) + recent-fill for deliberately-unrelated items (C5 5/5) — neither of which #646's OFF arm (`ENGRAM_RECENT_FILL=-1`) disabled. Byproduct (eval-isolation gotcha, not a production bug): recall's `ingest --auto` sweeps ancestor `.claude` dirs, so an eval cwd under `~/.claude` pulls operator-global memory into the isolated index; the harness handles it (clean-`/tmp` cwd). Reusable harness at `dev/eval/traps/recency_value/`. Closed 2026-07-19 (Joe): continuity value stands on C4i/C5, no new proof needed. | do not re-attempt a self-capture recent-fill proof — needed⟹queried makes it structurally vacuous |
| Retrieval ranking quality (matched-note floor) | Shipped / Settled | closed the note-vs-chunk drowning gap (`docs/FEATURES.md` — Matched-note floor; also the gated exception recorded in Standing constraint above, `LEDGER.md#matched-note-floor`); diagnostic symptom→cause surfacing checks out separately healthy (`LEDGER.md#diagnostic-surfacing-probe`). | decided — none |
| Skills-from-atoms decomposition (write-memory worker, capture guards G1/G2/G6) | Shipped | `docs/FEATURES.md` — Write-memory worker + capture guards; stop-at-the-write-seam disposition `docs/architecture/adr.md` ADR-0015. Three pre-registered upgrade triggers remain live in Parked backlog → Pre-registered guard upgrades (G6→G5, G2→G3, G4). | decided — none |
| Deeper-arc retrieval (tag nomination vs graph traversal) | Shipped / Settled | resolved 2026-07-02 — controlled-vocab tag nomination beat graph traversal outright (`docs/architecture/adr.md` ADR-0011); further unbuilt extensions collapsed into Parked backlog → Deeper-arc extensions. | decided — none |
| Q&A memory round-1 (capture) · Arm V (q-space channel premise) | Shipped / Measured (borderline) | round-1 capture shipped (`docs/FEATURES.md` — Q&A memory round-1); round-2/3 gate on real accumulated pairs per `docs/architecture/adr.md` ADR-0012 (see GATED band above). Arm V's round-1 result was BORDERLINE at n=10 (`LEDGER.md#qa-arm-v-borderline`); pre-registered bands PASS≥8, BORDERLINE 6–7, FAIL<6 (paraphrases ranking their own Q-note first among Q-notes and above every content note) — the larger-n round-2 gate applies the same proportions (PASS≥80%, BORDERLINE 60–70%, FAIL<60%). | settled at n=10 — a NEW larger-n result, not a retry |
| #684 Variant B · #689 Variant A | Refuted | **#684 (Variant B — true-lazy matched-set, withheld matched-note content fetched on demand)** — un-parked by Joe 2026-07-12 at #657's close ("the time reduction was the whole point"), closed by Joe 2026-07-13 after two adverse re-measurements.<br>Measure-first: segmented the ~52s baseline (`LEDGER.md#recall-time-split`) into pre-query (~17.5s median), query in-flight (~12.2s median), payload consumption (24.6s median [24.4–30.0], the addressable slice), remainder (~0s); census 125.3 KB median payload, 12.3 KB duplicated candidate-note content, ~58 KB of `items[]` ahead of the first candidate.<br>Checkpoint 1 (Joe, 2026-07-12): built Variant B over the cheaper dedupe-only Variant A.<br>Built: clusters-first payload ordering + withheld matched-note content (`engram show` fetch-on-demand) shipped in the binary + recall skill's consumption contract; trap gate GREEN before/after.<br>Round-1 re-measure: contradicted the bet — bytes down (payload total −23.3%) but phase-c UP 24.6→35.3s [30.0–49.4] and total span up; mechanism: 2 of 3 after-trials ran `engram query` unpiped, colliding with Claude Code's ~90 KB persisted-output truncation, forcing the agent to page the payload back out of its own session transcript via grep/sed/Read.<br>Checkpoint 2 (Joe, 2026-07-12): kept Variant B, fix the skill (Task 3b: mandatory single-capture query discipline — redirect `engram query` to a session-tmp file, never unpiped, then sliced Reads) rather than revert immediately.<br>Round-2 re-measure (deciding run): phase-c got WORSE, not better — median 58.0s [37.5–62.0], the entire range above the pre-registered 26s keep/revert bar; mechanism: the capture discipline converts in-context reading into a 5–10-round-trip grep+Read fetch loop over the redirected capture file (~5–9s/round-trip); pre-build's inline items-first payload let the agent read/judge directly from the query tool_result with zero fetch turns — the −21% byte cut (true census) could not buy back that round-trip latency; no `engram show` fetch ever fired in either round (nothing was missing — the regression is pure round-trip overhead, not fetch cost).<br>Checkpoint 3 (Joe, 2026-07-13): revert, per the pre-registered rule (phase-c median > 26s). Reverted: commit `0ae98779` restores the pre-build payload shape across `internal/cli`, `skills/recall/SKILL.md`, and the three downstream eval consumers touched by the build — diff-empty vs the last pre-build commit `055a07f5`; trap gate GREEN.<br>Standing conclusion (scoped to what was measured): fetch-mediated and file-mediated payload reading (Variant B) is a dead lever at current scale — inline in-context reading is the measured fast path, byte cuts do not buy back API round-trips (`LEDGER.md#payload-restructure-refuted`).<br>**#689 (Variant A — clusters-first + dedupe-only, everything inline, zero new round-trips)** — probed separately, subsequently REFUTED: dedup worked (duplicated candidate-note content 10.1 KB→0, total payload ~10% lighter) but bought no consumption time (24.70s after-measure vs 21.75s baseline — the pre-registered class-closing bar, ≥3.0s improvement AND after_max<baseline_min, failed both conditions), confirming recall consumption is round-trip-dominated, not byte-dominated (`LEDGER.md#variant-a-probe`).<br>The payload-shape lever class is now CLOSED — Variant B refuted as #684, Variant A refuted as #689, both byte-shape levers dead. The segmented baseline (`LEDGER.md#recall-time-split`) is the reference measurement for any future recall-time lever. | a NEW fact, not a retry |
| #690 (pre-query) | Refuted | measured-no-cut: refined to ~18.6s (n=8) from the ~15–21s n=3 estimate — the largest single phase; inner split ~73% irreducible model reasoning (skill-read+Step-0 7.15s + phrase-composition 6.5s); the one clean mechanical slice (the `engram ingest` sweep) is only 0.7s — far below the 3.0s bar; Gate A found the recall skill is ALREADY a `--phrase` template, so the compose lever's delta was near-empty. Joe chose to close it rather than spend on the experiment; no recall behavior change (`LEDGER.md#prequery-composition`). The pre-query span is model-reasoning-bound, not mechanically cuttable — same conclusion as consumption. | decode/model split changes |
| Vector-read / decode-cache measurements (see DEFERRED band above) | Measured | within-`scan` attribution: the ~4.8s scan is ~83% reading+decoding the 6,355 non-empty vector files (364 MB), only ~16% the 35k empty-file opens, ~1% vault+sidecar (`LEDGER.md#chunk-scan-vector-read-attribution`) — the lever is the vector read (consolidate files / binary-encode vectors / cache parsed vectors), not the empty-file count; the "skip the 85%-empty files" candidate was measured and rejected (16% < the pre-registered 40% bar) — the empties are a disk-hygiene issue, addressed separately (see the empty-file-hygiene ship above). Finer split: the ~4s scan is ~95% JSON float-decode (consolidation refuted at ~3%); a binary parsed-record cache ceiling is ~87%/~3.5s but only ~7% of the ~52s recall op (`LEDGER.md#chunk-scan-decode-cache-deferred`) — Joe DEFERRED the cache subsystem at the Ratification Gate (a 4-angle Gate-A'd plan existed); embed's 41k-vector match→ANN stays a separate, undiscussed lever. | decode grows to dominate a larger share of the op |
| Payload-prune production build (see DEFERRED band above) | Rejected (revisit on trigger) | operating premise (2026-06 cost-axis analysis): payload size is cache_read-cheap (moves time/paging, not dollars); the dollar lever is pruning the payload out of build context after Step 3 — smoke-validated (`LEDGER.md#payload-prune-smoke`: build_cost −40%, n=3, synthesis-injection proxy, honest bounds in the row), not relitigated at the smoke stage. Production build rejected by Joe 2026-07-08: the design got too complicated — isolation at all nesting depths forces launching the agent FROM engram (`engram recall` shelling to `claude -p`) rather than a plain Agent-tool subagent, because a subagent can't reach a leaf's own first-step recall (the recursion identified 2026-07-01), which is what forced the subprocess form. Design record: `docs/design/2026-07-01-engram-recall-subprocess-design.md` (status header marks the rejection). Note: the flag history here is a lesson — its "← NEXT" flag was assistant-maintained and never user-ratified; this rejection is the first recorded user decision on the build. | a viable subagent route — isolation via normal subagent dispatch (or an accepted depth-limited form) without engram launching `claude -p` |
| Between-session consolidation pass (see DEFERRED band above) | Rejected (revisit on trigger) | a batched dedup/contradiction-sweep/decay pass over the vault, sleep-time-compute-shaped — a field-wide pattern this system doesn't have. Joe rejected async-learn as relocation-not-reduction (moving cost off the perceived path isn't cutting it); this pass is the same trap unless it clears a stricter bar: it must reduce total spend (e.g. replace N per-session crystallizations with one cheaper batch) or measurably shrink recall payloads, gated by the trap regression + `$METER`. Treat as reopening a settled park — requires demonstrating that new fact first, not just proposing the pass. | proof it reduces total spend (not relocates it) |
| Harder-builds eval baseline + re-test (dispositioned #642) | Measured | the easy-build regime measured memory as a net tax (`LEDGER.md#c1-c2-warm-op-negatives`); the harder multi-round regime was ATTEMPTED 2026-07-17 under #642 and DISPOSITIONED — op-cost is NOT cleanly measurable via a self-seeding build eval (no Goldilocks window: easy → cold converges too → the fixed tax dominates; hard → the seeder can't converge either → `/learn` captures nothing). A contamination bug (warm arm read the operator's REAL global chunk index = the answer key; warm app1 scored 8/8 from an empty vault) was found + FIXED (per-cell chunk isolation, commit `d943ea93`; post-fix warm app1 → round-1 0/8) — it would have silently poisoned ANY future memory eval. The capability win on idiosyncratic content stands (note 99; ~3–4/8 without notes vs 8/8 with); op-value measurement redirected to real long-session work (note 98) (`LEDGER.md#harder-regime-op-cost-unmeasurable`). | a non-self-seeding long-session op-value measurement (note 98), not a retry of the self-seeding build eval |
| #665 · #677 | Closed (won't-do) | **#665 — value-gate over-fire, closed not-planned 2026-06-30.** The wording-based value gate meant to scope decision-moment recall firing to idiosyncratic content does not hold on opus — it fires on routine work regardless of phrasing. Joe closed this as an accepted cost, not a problem: over-firing the cheap `glance` rung is fine, under-firing is the real risk. Reopen condition: only if cheap over-fire is later shown to be a measured problem (e.g. `glance` itself stops being cheap, or over-fire is shown to displace something load-bearing).<br>**Recall-before-recommend re-entry (the harness + criteria that produced #677).** A lever invented mid-synthesis was never re-checked against the vault, so a previously-killed direction could resurface as if new. The anti-amnesia harness ("lever-recheck") shipped 2026-07-08 with a RED baseline established — 4/5 fixtures reproduce the miss, 0/15 re-query either turn (`LEDGER.md#c7-lever-recheck-red-baseline`). Criterion 1 (Step 3.5, forced `Re-entry:` line riding directly above the recommendation) closed in three iterations (worded honor-rule → forced output contract → contract coupled to the RECOMMENDATION line's adjacency): the v3 GREEN batch measures fire-rate 93% (14/15), honored-when-fired 14/14 (`LEDGER.md#c7-reentry-query-green`). The strict pre-registered bar (every valid arm-A trial RECONCILED, all 4 fixtures at 1.0) is NOT met — one stochastic non-fire (fixture4 t1) — and an asymptote analysis shows a prose+structure mechanism can't reliably clear an every-trial bar (P(12/12 fires) ≈ 42% at 93% per-trial). Joe closed the re-entry work at 93% and filed the residual as a mechanical-enforcement follow-up: **#677** (mechanical enforcement layer for the Step 3.5 Re-entry contract, 93%→100%) — closed won't-do 2026-07-10 (Joe): 93% is the accepted operating point, no enforcement layer. Reopen only if a missed re-entry ships a closed-lever recommendation in real work (not the harness). Criterion 2's premise (negation-carrying notes outranked by chunks) is disposed SUPERSEDED — it predates the matched-note floor, and 0 surfaced-but-ignored cases appear across 24 honest-instrument fired trials.<br>The harness itself remains live infrastructure — a `/please`-orchestration variant of it is the blocker for the LATER-band gate-analytical-recommendations item above. | a missed re-entry ships a closed lever in real work |
| haiku whole-op/split · payload-size-for-$ / phrase-cut / skill-body-lighten · link-prediction (TransE/RotatE) | Dead end | Whole-op or split haiku recall+build: pre-registered guards, both measured dead (`LEDGER.md#haiku-whole-op-dead-end`, `LEDGER.md#haiku-split-dead-end`).<br>Payload-size caps for dollars, cutting the 10 query phrases, and lightening the skill body to raise firing rate — all settled (2026-06/07 cost-axis + recall-trigger analysis, in git history): bytes are cache_read-cheap (the dollar tax is carrying the payload, not its size — same premise as the payload-prune item above, `LEDGER.md#payload-prune-smoke`); breadth in the query phrases surfaces the un-guessable notes; firing rate is set by the skill `description`, not its body.<br>Link-prediction (TransE/RotatE) for the vault graph — an explicit pre-registered guard from the 2026-07-02 research survey (`docs/design/2026-07-02-research-followups.md`, deleted 2026-07, git log): do not invest in an LP-predicted edge-fabric variant without a downstream retrieval benchmark demonstrating headroom first. No such benchmark exists; nothing to revisit until one does. | a downstream retrieval benchmark showing headroom |
| Blanket "recall before every tool call" hook | Dead end | refuted as a fatal over-firer (`LEDGER.md#recall-overfire-hook-rejected`). Distinct from the still-open narrower hook (before-declaring-done / after-a-tool-failure only) in Parked backlog → Decision-moment recall hook. | a downstream case showing the blanket form is actually safe (none proposed) |
| C5 recency-apply gap | Measured (mitigated, unscheduled follow-up) | `/recall glance` surfaces a recently-updated standard but does not reliably apply it (`LEDGER.md#glance-fails-c5-delivery`); C5-type cues currently escalate to `deep` as the mitigation. Lifting both rungs' apply rate above `deep`'s own is a separate, unscheduled follow-up — see Parked backlog → Recall timing/apply residuals. | a chunk/note-quality gauge shows a real drowning/miss case (same as Parked trigger) |
| Artifacts/file-keyed retrieval angle | Refuted (parked) | a free warm-vs-warm probe of adding a file-keyed angle to recall's 10 topic angles refuted it: ≤2/6 fixtures gained an actionable note the topic angles missed, below the ≥4/6 bar (`LEDGER.md#b-artifacts-angle-headroom`). Lessons are keyed by situation/principle, not filename, so a file-keyed angle re-finds what the topic angles already surface (corroborates vault note 73). Companion: the issue-briefing convention gained an independent systems/artifacts split + a prior-work/failure element the same cycle (memory-only; no repo change). | a real recall miss where a file-keyed angle would have surfaced a note the topic angles missed |
| Question-anchored crystallization | Refuted (parked) | anchoring notes to the prompting question instead of the topic was evaluated and parked — no delivery benefit, clear retrieval loss (`LEDGER.md#qanchor-park`). One untested sub-lever survives as a hint, not a result: question-anchoring may help transferable-pattern lessons specifically while hurting concrete-API lessons (`LEDGER.md#qanchor-pattern-type-sublever`, n=3, not independently validated). | crystallization quality resurfaces as a bottleneck — every lever tried so far (handle-wording, question-anchoring, synthesis persistence, graph expansion) has nulled on delivery |
