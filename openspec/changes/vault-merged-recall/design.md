## Context

See proposal.md - Why for motivation. Two relevant existing shapes:

- `engram query` today has exactly two modes, dispatched in `targets.go`:
  `ENGRAM_SERVER` set → `fetchQuery` (HTTP client, `fetchAndCopy` pipes the
  served response's raw bytes straight to stdout, never parsed into a Go
  value); unset → `RunQuery` (the full local pipeline: matched-set → cluster
  → nominate → render). `fetchQuery` never needed to parse the response
  because exclusive mode has nothing to combine it with.
- `vault-serve-api`'s `/query` route already returns the same payload shape
  a local invocation produces, `model_id` included (`vault-query-model-
  provenance`) — this capability consumes that route unmodified; no
  `internal/cli/serve.go` change is needed.
- `show`/`show-chunk` refs are resolved purely against the local vault/chunk
  index today, same exclusive-switch dispatch as `query`. Ref formats:
  notes take "full basename | [[wikilink]] | trailing .md | bare Luhmann
  id"; chunks take "source#anchor". `nextLuhmannID` (`internal/cli/
  luhmann.go`) mints ids purely from each vault's own existing id set —
  no cross-vault coordination exists or is planned.
- `--limit`'s resolved value (`args.Limit`, default 20 via
  `defaultQueryLimit`) is threaded all the way through `runQuery` into
  `aggregatedSummary.limit`, but the only place it lands is
  `Budget.Limit` — reported metadata. Nothing truncates `Items`; actual
  item count is governed by `matchPhraseLimit`=30 (per phrase),
  `matchSetCap`=300 (total), and `candidateNoteK`=5 (per cluster).
  Discovered while implementing the merge (see Decision 9).

## Goals / Non-Goals

**Goals:**
- One config knob, `ENGRAM_PARENT`, a single URL.
- Merged mode on `engram query` only — matches the issue's own scoping and
  the one identified consumer (phone-llm#31).
- Merge = score-sort interleave of two already-complete payloads (Option A
  from the engram#729 exploration).
- A deterministic way to follow up on a merged result: a per-item origin
  tag on `query`'s merged payload, and a `--parent` routing flag on
  `show`/`show-chunk` — not an implicit local-then-fallback probe (see
  Decision 8).
- `--limit` becomes a real, enforced cap on `items[]` count in every mode,
  not merge-only (Decision 9) — a deliberate scope expansion beyond the
  original issue, chosen over an asymmetric merge-only cap.

**Non-Goals:**
- N-remote fan-out — the topology is single-parent by design (proposal.md).
- Re-clustering across nodes — that's Option B, filed separately as
  engram#731, an experiment to run after this ships and compare against.
- Any `engram serve` (server-side) change — the parent is queried through
  the existing, unmodified `/query`, `/show`, and `/show-chunk` routes.
- Merged (fusion) mode for `query-chunks` and `activate` — no consumer need
  identified yet. `show`/`show-chunk` are in scope, but only for single-
  target `--parent` routing (Decision 8), not fusion — there's nothing to
  fuse when a lookup names one specific ref.
- Cross-model score calibration/normalization (see Decisions).

## Decisions

1. **Payload-level fusion (Option A), not matched-set-level fusion (Option
   B).** Fusing before clustering would need the parent to expose pre-
   cluster scored candidates, which `/query`'s byte-identical-output
   contract can't do without either a new served route (growing the fixed
   served command set) or breaking that contract. Fusing two complete,
   already-clustered payloads needs neither. Option B is filed as
   engram#731 to get a real Option A baseline to compare fidelity against,
   not blocking this work.

2. **Score-sort interleave, no score normalization.** Considered and
   rejected: rescaling each source's top-N to a 0–100 range before
   interleaving. That's a within-batch relative rescale, not a fix for
   cross-model comparability — a source with nothing relevant for a query
   still produces a "100"-scaled top item (the "no true negative" failure),
   making a weak match look as strong as a genuinely good one from another
   source. Two different embedding models occupy different geometric
   spaces; no linear rescale fixes that, only real calibration against a
   shared reference corpus would, which is out of scope. Plain descending
   sort on raw score is used instead.

3. **Merge is unconditional on `model_id` — no refuse, no fallback.**
   Interleaving by score is itself a comparison, so a mismatched embedding
   model is a real skew risk. But refusing or falling back to local-only on
   mismatch would make merged recall break across a fleet doing rolling
   model upgrades — an explicit goal is letting individual envs adopt
   whatever embedding model suits them, ad hoc, without breaking merged
   recall fleet-wide. Per-item `model_id` rides along as a visible,
   non-gating advisory signal instead, consistent with the existing
   non-fatal `warnModelMismatch` pattern for stale local sidecars.

4. **Parent-unreachable degrades to local-only + warning, not a hard
   failure.** New call made in this design (not previously discussed) but
   symmetric with Decision 3's posture: a node's local recall shouldn't
   become dependent on its parent's network availability, or a transient
   host outage takes down every env's local recall along with it. Flagging
   this explicitly for review since it wasn't settled in the prior
   exploration.

5. **`ENGRAM_SERVER` takes precedence when both are set.** `ENGRAM_SERVER`'s
   existing contract already fully delegates command handling to the
   remote ("instead of touching local files") — there's no local pipeline
   left to merge against in that mode. Rather than defining three-node
   query chaining, `ENGRAM_PARENT` is simply inert whenever `ENGRAM_SERVER`
   is active. Keeps the two mechanisms orthogonal.

6. **New serve-client decode path required.** `fetchQuery`'s `fetchAndCopy`
   writes the parent's response bytes straight to stdout — sufficient for
   exclusive mode, insufficient for merging. This needs a variant that
   decodes the parent's response into the same payload type the local
   pipeline already produces, so the two can be interleaved before one
   render-to-stdout call. The parsed type isn't new (it's the existing
   query payload shape, now also producible from an HTTP response); only
   the decode path is.

7. **`content-budget`, `recent-fill`, and `limit` all apply post-merge, via
   one shared mechanism.** Naively letting each source apply its own budget
   independently and concatenating the results would let the merged total
   run up to 2x the requested cap (e.g. 15 full-content chunks from local
   plus 15 from parent, instead of 15 total) — exactly the payload-size
   problem these flags exist to prevent, and now (Decision 9) `limit` is a
   real cap too, with the same risk. The fix for all three is the same
   shape: each source is fetched with these three knobs set to effectively
   unbounded values (content-budget disabled via `<= 0`, per
   `capChunkContent`'s existing guard; recent-fill and limit set to a large
   internal sentinel — recent-fill has no "unlimited" value of its own,
   negative means the channel is *off*, so a large explicit number is used
   instead), so neither source pre-truncates candidates the merge might
   need. The user's real requested values are applied exactly once, as a
   single final pass over the fully-merged, re-ranked set.

   One known limitation: the recency channel's "newest first" ordering is
   positional in each source's own rendered payload — `queryItem` carries
   no timestamp field, so there is no way to interleave two sources'
   recency items into one true chronological order after rendering.
   Fetched-unbounded items from each source are concatenated in
   alternating (local, parent, local, parent...) order and capped to the
   requested `recent-fill` count — bounding total size correctly (the
   property that actually matters — recall's own spec already marks this
   channel "non-load-bearing") without claiming an exact global
   chronological order the wire format can't support. Adding a timestamp
   field would fix this properly but is out of scope here.

8. **Per-item origin tag + explicit `--parent` flag, not implicit
   try-local-then-fallback.** Considered and rejected: on a `show`/
   `show-chunk` miss, silently falling back to the parent. Chunk ids
   (`source#anchor`, derived from each node's own local file paths) are
   safe from cross-node collision but still need an explicit signal that a
   lookup should go to the parent at all — nothing to fall back *from*
   without one. Note refs (bare Luhmann ids) are genuinely unsafe:
   `nextLuhmannID` mints ids purely from each vault's own local id set, no
   cross-vault coordination, so two independently-run vaults routinely
   produce the same id for unrelated notes. A silent fallback on
   not-found is safe by construction (it only fires when nothing local
   matched); the actual risk is the *unprompted* case — an agent guessing
   local-first without knowing the item came from the parent, where a
   coincidental local id collision returns the wrong note's content with
   no error at all. Explicit per-item `from_parent` tagging plus an
   explicit `--parent` flag removes the guess entirely: the agent always
   knows an item's origin from the query result and states where to look.

9. **`--limit` becomes a real cap everywhere, not merge-only.** Presented
   three options when this was discovered: (a) leave item count ungoverned
   by `--limit` in merged mode too, same as local today; (b) fix `--limit`
   everywhere — local, `ENGRAM_SERVER`-exclusive, and merged; (c) a new,
   merge-only cap, leaving local-only behavior untouched (so `--limit`
   would mean two different things depending on mode). Chose (b), by
   explicit direction: merging two sources' independently-nominated
   candidate sets needs a real bound regardless, and a merge-only cap
   would make `--limit`'s meaning mode-dependent for no good reason. This
   is marked **BREAKING** in proposal.md — existing local/served callers
   will see fewer items by default than they do today. See Migration Plan.

## Migration Plan

`--limit`'s new enforcement (Decision 9) changes already-shipped behavior
for every existing `engram query` caller, not just this change's new
merged mode. Two things should happen before/around rollout, not silently
absorbed:

- `dev/eval/LEDGER.md`'s recall-quality gates that measured payload size or
  item survival against the current *uncapped* item count — at minimum
  `#matched-note-floor`, `#payload-cut-lazy-chunks`, `#payload-cut-
  recent-fill`, and `#crowded-vault-capability-robustness` (all cited in
  `recall-payload-cuts`/`recall-two-channel-payload`/`recall-matched-note-
  floor`) — should be re-run against the new capped behavior to confirm
  the default `--limit`=20 doesn't regress recall quality. `defaultQuery
  Limit`=20 was chosen and reported as metadata only; it was never
  validated as an actual coverage floor, because until now nothing
  enforced it.
- If those evals regress, the fix is raising the default, not reverting
  the cap — the cap itself is what makes merged mode's size bound (and
  `content-budget`/`recent-fill`'s post-merge reapplication, Decision 7)
  actually work.

No code rollback plan beyond a normal revert — this is a behavior change
to an existing flag's enforcement, not a data migration.

## Risks / Trade-offs

- [Risk] Mismatched-model interleave produces a skewed ranking with no
  correctness signal beyond the advisory tag → [Mitigation] `model_id` is
  visible per item; engram#731 remains available as a higher-fidelity
  alternative if this proves inadequate in practice.
- [Risk] A stderr warning on parent-unavailability is easy for an agent
  consumer to miss → [Mitigation] reuse the existing `LogWarning`
  convention agents already handle for other non-fatal advisories
  (`warnModelMismatch`, `warnOldSchema`) rather than inventing a new signal
  shape.
- [Risk] The new serve-client decode path duplicates some of `fetchQuery`'s
  request-building → [Mitigation] share the existing factored-out helpers
  (`buildURL`, `encodeQuery`, `setIntParam`, etc.) in `serve_client.go`;
  only the response-handling tail differs.
- [Risk] An agent ignores a merged item's `from_parent` tag and runs
  `show`/`show-chunk` without `--parent`, hitting a coincidental local id
  collision (notes) or a plain not-found (chunks) instead of the intended
  content → [Mitigation] this is a correctness footgun inherent to any
  design that requires the caller to pass routing state along, not fully
  closable client-side; the query payload surfaces `from_parent` right
  next to each item specifically to make it hard to miss.
- [Risk] Enforcing `--limit`=20 by default may cut recall quality for
  existing callers who never hit a real cap before → [Mitigation] see
  Migration Plan — re-run the cited eval gates and raise the default if
  they regress, rather than discovering this in production.
