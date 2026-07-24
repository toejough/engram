# Engram — Implemented Capabilities

One entry per shipped, user-visible capability: what it does today, why it exists
(`docs/architecture/adr.md`), and what's proven about it (`dev/eval/LEDGER.md`). For
terminology, see `docs/GLOSSARY.md`; for install/CLI reference, see `README.md`; for
what's still in flight, see `docs/ROADMAP.md`.

## Matched-note floor

Recall's clustering reserves per-phrase note slots so a crystallized lesson competing
against a flood of raw chunks still surfaces — a relevant note is not drowned out by
noisier transcript fragments.

why: `docs/architecture/adr.md` — ADR-0004
validation: `dev/eval/LEDGER.md#matched-note-floor`

## Payload cuts (--lazy-chunks, --recent-fill)

Recall's underlying query defers a matched chunk's full text until the agent explicitly
asks for it, and trims how much raw recent activity rides along on every call — so a
recall pass carries less to read without losing reach; the agent fetches a chunk's real
content only when it turns out to matter.

why: `docs/architecture/adr.md` — ADR-0004
validation: `dev/eval/LEDGER.md#payload-cut-lazy-chunks`; `dev/eval/LEDGER.md#payload-cut-recent-fill`

## Query timing instrument (--timings)

`engram query --timings` emits a per-stage wall-clock timing block (scan / embed / cluster /
nominate / render) for the query in-flight phase, computed through the injected clock and gated
so the default recall payload stays byte-identical. It is a measurement-only diagnostic: breaking
the query's time down by stage showed that reading and decoding the on-disk chunk-index files
(the `scan` stage) — not running the embedding model (the `embed` stage) — is what makes a
recall query slow.

why: #691 — decompose the query in-flight span to find the dominant stage before optimizing
(measurement-first; DI clock, additive, default payload unchanged)
validation: `dev/eval/LEDGER.md#query-inflight-split`

## Glance/deep recall dial

Recall runs at two depths: a cheap, read-only rung for firing often at everyday decision
points, and a full rung that also crystallizes new lessons into the vault. The cheap rung
escalates to the full one automatically when the task requires actually applying a
convention that was recently corrected or superseded — retrieving that a newer convention
exists is not enough; the agent has to act on it, and testing showed the cheap rung
surfaces such conventions without acting on them.

why: `docs/architecture/adr.md` — ADR-0004
validation: `dev/eval/LEDGER.md#glance-delivers-c3-c4i-c6` (delivery); `dev/eval/LEDGER.md#glance-fails-c5-delivery` (the escalation rule's reason); `dev/eval/LEDGER.md#glance-cost-realvault` (cost)

## Recall-at-decision-moments guidance (@import)

A standing instruction, imported into CLAUDE.md, nudges the agent to fire a cheap recall
pass at specific moments — when weighing a proposed approach in conversation, before
declaring work done, after a failure it can't explain, and before committing to a new
approach — instead of relying on recall firing only at task start.

why: `docs/architecture/adr.md` — ADR-0001
validation: `dev/eval/LEDGER.md#recall-moments-opus48-remeasure` (current-model measurement; supersedes the earlier headless flip); `dev/eval/LEDGER.md#underload-firing-wording-fix` (the conversational-endorse cue lifts under-load firing 50%→100%, deployed 2026-07-15)

## Delegate-object-level-work guidance (@import)

A standing instruction, imported into CLAUDE.md, fires the "you are an orchestrator" reflex at the
delegation decision moment — plan the unit, hand it to a subagent, review what returns, report the
outcome — instead of doing object-level work solo out of habit. Sibling to the recall-firing
guidance; it points at the `route` skill (the *how* of one dispatch) and `/please` (the full gated
procedure) rather than restating them. The floor is evidence, not a guess: go inline only when
recalled memory shows a task-kind runs reliably below the routing overhead — never on an
in-the-moment "it's trivial" guess.

why: mirrors the recall-firing guidance (@import) pattern; doctrine in `agent-instructions/guidance/delegate.md`, `agent-instructions/skills/route/SKILL.md`, `agent-instructions/skills/please/SKILL.md`
validation: `dev/eval/LEDGER.md#delegate-guidance-flip` (headless RED→GREEN: solo→subagent-dispatch, including the trivial-rename case)

## Please Step 3 doc-surface enumeration grep + Gate A independent-pass verification

`please`'s Step 3 (Plan) carries a non-waivable doc-surface enumeration grep: when a plan alters
a repeated invariant (a payload shape, a cadence, a naming convention echoed across docs,
diagrams, or skills), the author runs a concept-variant grep over the repo and pastes a per-file
disposition list into the plan itself. Before that plan can proceed, `please`'s Gate A review
independently checks that list against the repo and always runs its own independent
discovery pass too — the author's list is never the reviewer's source, and its presence
never narrows the reviewer's scan.

why: issue #685 (Change #1) — gate reviewers kept catching doc-scrub the plan author missed across
4+ cycles
validation: `dev/eval/LEDGER.md#685-doc-enumeration-grep`; new headless probe harness
`dev/eval/cumulative/please_step3_probe/`

## Evidence-based route rubric

The `route` skill picks a subagent's model tier from memory, not a fixed table: every unit
of work starts at the cheapest tier, and only recalled evidence — or a failed review —
raises it. Each dispatch is recorded (work-kind, tier, concrete model, review-sourced
outcome); those records become recallable memory, so the effective rubric improves over
time without editing the skill. The memory tier discount — routing recallable-answer work
one tier cheaper — is its sole evidence-backed cold-start prior.

why: `docs/architecture/adr.md` — ADR-0017 (extends ADR-0014)
validation: `dev/eval/LEDGER.md#tier-routing-parity`

## Vocab lifecycle (definition notes, tags-based term assignment, tag nomination, supersession ride-along, autonomous refit)

A controlled vocabulary of bare-`vocab`-tagged definition notes (shipped as ordinary tags-based fact
notes 2026-07-10, #678, superseding the earlier dual-channel `vocab.<term>.md` term-note form) tags
every written note on a shared axis via `tags: [vocab/<term>]`, letting recall nominate cross-cluster
notes that share a tag with its top matches, and — when a newer note supersedes an older one — surfacing
the newer note alongside the older one it replaces, instead of leaving them to be found separately. The
vault also checks its own tag health (growth, concentration, untagged rate) and prompts its own re-fit
instead of drifting stale.

why: `docs/architecture/adr.md` — ADR-0011
validation: `dev/eval/LEDGER.md#vocab-tag-nomination-l6xtag` (nomination); `dev/eval/LEDGER.md#vocab-refit-cost` (refit)

## Q&A memory round-1 (learn qa)

`engram learn qa` captures a question and its answer as a linked pair. The answer
competes for retrieval like any other note, while the question stays out of the main
search space — its wording measurably hurts retrieval — and is instead reachable through
the answer it belongs to.

why: `docs/architecture/adr.md` — ADR-0012
validation: `dev/eval/LEDGER.md#qa-arm-v-borderline` (the round-3 premise check; round-1 capture itself shipped without a dedicated eval row)

## Concurrency-safe vault writes

Every vault writer — learn, amend, resituate, activate, ingest, prune — is serialized
under a file lock with atomic rename, so two agents working the same vault at the same
time no longer corrupt notes, sidecars, or the chunk-index manifest.

why: `docs/architecture/adr.md` — ADR-0013
validation: concurrent-writers regression test + `targ check-full` (commit `f7f6b389`) — no eval-ledger row; correctness is test-locked, not eval-measured

## Write-memory worker + capture guards (reversals, lessons audit, escalation provenance)

A dedicated worker skill executes the actual vault-write commands on behalf of recall and
learn, so the judgment of what to capture stays separate from the mechanics of writing
it. Learn also captures self-discovered reversals and confirmed approaches (positive
reinforcement, user-praised or self-validated) as their own lesson kinds, and `please`
audits each cycle's mechanical corpus — failed gates, corrections, escalations — against
the vault to catch lessons that went uncaptured.

why: `docs/architecture/adr.md` — ADR-0001
validation: `dev/eval/LEDGER.md#write-memory-worker-fire-rates`

## Embed-on-write + dual-vector sidecars

Every note gets a sibling embedding file the moment it's written — one vector for its
situation, one for its body — so the vault stays self-contained (no separate index to
build or fall out of sync), and retrieval can match a query against whichever angle of a
note fits best.

why: `docs/architecture/adr.md` — ADR-0003
validation: unmeasured as a capability — correctness rests on the embed/sidecar invariants (`docs/architecture/memory-invariants.md`) and unit tests, not an eval row

## Unified two-channel recall payload

One retrieval call returns two kinds of context: a clustered relevance channel
(crystallized notes and raw transcript fragments ranked and grouped together) and a
separate, un-clustered recency channel (the newest raw activity). An agent re-orienting
after context loss gets both "what's relevant" and "what just happened" in a single pass.

why: `docs/architecture/adr.md` — ADR-0004
validation: the relevance channel's ranking is validated by `dev/eval/LEDGER.md#matched-note-floor`. The recency channel decomposes into two mechanisms, both proven in the same eval
(`dev/eval/LEDGER.md#crowded-vault-capability-robustness`, test conditions C4i and C5 there). Its
**re-rank** (recent-relevant outranks old-relevant — the day-to-day continuity lever) is proven under
that eval's C4i condition, and its delivery of *deliberately-unrelated* recent items is proven under
its C5 condition. Its **recent-fill** delivery of *self-captured work the agent needs* was found to add
nothing over plain cosine — a needed decision is topically related to the task, so `/recall`'s broad
query already surfaces it directly, making the recent-fill channel redundant there and structurally
untestable via self-capture (`dev/eval/LEDGER.md#646-recency-recent-fill-selfcapture`, issue #646 closed
2026-07-19)

## Ingest auto-sweep with non-persistent-workspace skip

`engram ingest --auto` keeps the chunk index current by mechanically sweeping every known
session and markdown source, while skipping session logs whose project path lives under
a throwaway filesystem root — so running evals or tests doesn't bloat the real vault's
index. A source that yields zero chunk records no longer leaves a 0-byte `.jsonl`
chunk-index file behind — the read path previously had to open and enumerate one such file
every query; `engram prune --empty` (+ `--dry-run`) additionally sweeps up any pre-existing
empties, ranking-neutral (empty files hold zero records).

why: `docs/architecture/adr.md` — ADR-0010
validation: unmeasured as a capability — behavior is locked by `internal/cli` ingest/sweep unit tests, not an eval row; the 0-byte guard is `TestRunIngestSkipsEmptyIndexFile` (`internal/cli/ingest_test.go`), `prune --empty` is `TestRunPruneEmptyRemovesOnlyEmptyFiles`/`TestRunPruneEmptyDryRunDeletesNothing` (`internal/cli/prune_test.go`); real-scale copy-verified scan recovery: `dev/eval/LEDGER.md#chunk-empty-file-prune`

## Prune preserves memory (detach on source deletion)

`engram prune` no longer garbage-collects chunks whose source file is gone — the embedded chunk (with
its vector) is the asset; the source `.jsonl` is only provenance. Prune now **detaches**: it drops the
stale manifest entry but keeps the per-source index file, which chunk search discovers by directory scan
(never via the manifest), so the memory stays fully searchable. Deleting ingested source files (e.g. a
restored-transcript dir) reclaims disk without losing the recovered memory.

why: `docs/GLOSSARY.md` — `engram prune` (subcommand); issue #659
validation: `internal/cli/prune_test.go` (`TestPruneDetachesDeadSources`) + `prune_integration_test.go` (real-fs detach); verified end-to-end (ingest → delete source → prune → query still finds the chunk)

## Count / backlinks aggregation (engram count)

A read-only counting surface over the vault, separate from `query`'s similarity recall.
`--group-by <attr>` counts distinct note membership per frontmatter-attribute value (a list attr
counts one per distinct element; a value listed twice in one note still counts once), optionally
restricted by repeatable AND-ed `--filter attr=value` predicates. `--backlinks-of <basename>`
prints a vault-graph node's wikilink in-degree plus its sorted linkers. The two modes are each
independently checkable by hand in Obsidian against their own view — group-by against a
property/tag filter, backlinks-of against the backlinks panel — but they are **not** equal to each
other: backlinks-of counts every linker while group-by counts only frontmatter members, so the two
diverge by the number of non-member linkers (e.g. a hand-authored MOC/hub page that links every note
on a topic without itself carrying that topic in frontmatter).
Count is also the audit surface for route's dispatch evidence (see "Route dispatch evidence +
aggregates" below): `--group-by tags --filter tags=tier/<t> [--filter tags=outcome/pass]` recomputes true
tier×work-kind tallies from evidence-note tags to verify/repair the LLM-maintained aggregate
notes — never on the routing read path (plain recall reads the aggregates).

why: `docs/architecture/adr.md` — ADR-0018
validation: `internal/cli/count_test.go` (`TestRunCount_GroupByBacklinksAgreement` — clean-case
agreement; `TestRunCount_BacklinksExceedGroupByForNonMemberLinkers` — divergence) + real-binary
vocab-stats parity on the live vault, historical (pre-#678, measured 2026-07-08): `--group-by vocab`
counted 33 for `retrieval-design`; `--backlinks-of vocab.retrieval-design` reported in-degree 34, the
+1 being `vocab.index.md`. The vocab wikilink channel and `vocab.index.md` were retired 2026-07-10
under #678, so `--backlinks-of vocab.<term>` now reads 0 for every term — the group-by/backlinks-of
divergence property itself is unaffected (proven by the two unit tests above, not by this stale
worked example)

## Route dispatch evidence + aggregates (tags-based)

Every route dispatch is recorded as an ordinary recallable fact note tagged with three
categoricals (`work-kind/<k>`, `tier/<t>`, `outcome/<o>` in frontmatter `tags:`, written by the
repeatable `engram learn --tag` flag), and each work-kind keeps one aggregate fact note
(`route-evidence-<work-kind>`) whose object text holds the running tier tallies plus wikilinks to
every evidence note. Route reads evidence by plain recall — aggregates surface as normal
memories; `engram count` recomputes tallies from tags as the drift audit. Family definition notes
(bare-tag convention) document the three tag families in the vault itself.

why: `docs/architecture/adr.md` — ADR-0019 (the 2026-07-10 decision on #669); issue #674
validation: `internal/cli/learn_test.go` (`TestLearnFact_Tags_WrittenToFrontmatter`,
`TestLearnFact_InvalidTag_RejectedBeforeWrite`, `TestRenderFactFrontmatter_TagsRoundtripFidelity`)
+ `internal/cli/amend_test.go` (`TestRunAmend_PreservesTagsFrontmatter`); scratch-vault drowning
gauge PASS at 20 sibling evidence notes + count recompute parity (2026-07-10 — this plan's
execution log, retired to git history at cycle close)

## Validated goals (mission rollup — not a capability)

This closing section is a cross-cutting summary, not a shipped capability: it records which
founding-mission goals an adversarial review found fully achieved, drawing on the capability
entries above and their `dev/eval/LEDGER.md` rows (the review's source document is retired; the
still-open goals live in `docs/ROADMAP.md`). Engram's mission: a correction given once should be
applied thereafter, without the user repeating it or the agent re-deriving it.

- **Say-once, capability half** — memory carries conventions and facts a cold model
  cannot derive on its own, and this holds even under a large, realistic mix of unrelated
  notes.
- **Persistent substrate + retrieval** — the vault reliably surfaces the right note for a
  query, at the practical ceiling of what the embedder can distinguish.
- **Retrieval ranking** — crystallized notes are not lost among raw transcript noise, and
  ranking quality holds across every subsequent recall change under a standing regression
  gate.
- **Tier democratization** — a cheaper model applying recalled memory matches a pricier
  model on the same memory-backed work, on most tested axes.

validation: `dev/eval/LEDGER.md` (draws on the matched-note-floor, tier-routing, and
capability-trap rows above)
