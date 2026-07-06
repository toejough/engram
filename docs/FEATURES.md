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

## Glance/deep recall dial

Recall runs at two depths: a cheap, read-only rung for firing often at everyday decision
points, and a full rung that also crystallizes new lessons into the vault. The cheap rung
escalates to the full one automatically when the decision turns on honoring a
recently-updated standard.

why: `docs/architecture/adr.md` — ADR-0004
validation: `dev/eval/LEDGER.md#glance-delivers-c3-c4i-c6` (delivery); `dev/eval/LEDGER.md#glance-fails-c5-delivery` (the escalation rule's reason); `dev/eval/LEDGER.md#glance-cost-realvault` (cost)

## Recall-at-decision-moments guidance (@import)

A standing instruction, imported into CLAUDE.md, nudges the agent to fire a cheap recall
pass at specific moments — before declaring work done, after a failure it can't explain,
and before committing to a new approach — instead of relying on recall firing only at
task start.

why: `docs/architecture/adr.md` — ADR-0001
validation: `dev/eval/LEDGER.md#recall-moments-opus48-remeasure` (current-model measurement; supersedes the earlier headless flip)

## Evidence-based route rubric

The `route` skill picks a subagent's model tier from memory, not a fixed table: every unit
of work starts at the cheapest tier, and only recalled evidence — or a failed review —
raises it. Each dispatch is recorded (work-kind, tier, concrete model, review-sourced
outcome); those records become recallable memory, so the effective rubric improves over
time without editing the skill. The memory tier discount — routing recallable-answer work
one tier cheaper — is its sole evidence-backed cold-start prior.

why: `docs/architecture/adr.md` — ADR-0017 (extends ADR-0014)
validation: `dev/eval/LEDGER.md#tier-routing-parity`

## Vocab lifecycle (term notes, dual-channel tagging, tag nomination, supersession ride-along, autonomous refit)

A controlled vocabulary of term-notes tags every written note on a shared axis, letting
recall nominate cross-cluster notes that share a tag with its top matches and letting a
superseding note ride along right behind the note it replaces. The vault also checks its
own tag health (growth, concentration, untagged rate) and prompts its own re-fit instead
of drifting stale.

why: `docs/architecture/adr.md` — ADR-0011
validation: `dev/eval/LEDGER.md#vocab-tag-nomination-l6xtag` (nomination); `dev/eval/LEDGER.md#vocab-refit-cost` (refit)

## Q&A memory round-1 (learn qa, D5′)

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
validation: unmeasured as a channel-split capability; the relevance channel's ranking is validated by `dev/eval/LEDGER.md#matched-note-floor`, the recency channel's delivery is not separately eval'd

## Ingest auto-sweep with non-persistent-workspace skip

`engram ingest --auto` keeps the chunk index current by mechanically sweeping every known
session and markdown source, while skipping session logs whose project path lives under
a throwaway filesystem root — so running evals or tests doesn't bloat the real vault's
index.

why: `docs/architecture/adr.md` — ADR-0010
validation: unmeasured as a capability — behavior is locked by `internal/cli` ingest/sweep unit tests, not an eval row

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
