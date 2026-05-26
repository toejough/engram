# v3 research — field-specific query

Date: 2026-05-26. Captured per Joe's inline note in `/recall`
SKILL.md questioning whether the "query by task, not by fear"
guidance still applies once queries can name specific note
fields. Deferred from v2; this doc records the question and the
shape a future spec would take.

## The question

Today `engram query <string>` embeds the full string and ranks
notes by whole-note cosine. The note's frontmatter has structured
fields:

- `situation` — when the principle applies
- `subject` / `predicate` / `object` (fact) — the principle's grammar
- `behavior` / `impact` / `action` (feedback) — the corrective shape
- `boundary_rationale` (L1 episode, after `2026-05-26-l1-episode-fix-spec.md`)

Joe's note: callers may want to query specific fields directly —
"what's the subject?", "what behavior am I about to embark on?",
"is there a time period involved?". Today they have to embed
that intent into the query string and hope the embedder picks it
up.

## What a field query would look like

Hypothetical CLI:

```
engram query "<text>" \
  --field situation="<phrase>" \
  --field subject="<phrase>" \
  --field behavior="<phrase>" \
  --field time="<range>"
```

Each `--field` clause:

- For text fields (`situation`, `subject`, `predicate`, `object`,
  `behavior`, `impact`, `action`, `boundary_rationale`):
  embed the clause separately; score the note's matching field
  via cosine; combine with whole-note score.
- For `time`: filter by `created` field or
  `provenance.transcript_range` for episodes.

## Why this is bigger than it looks

To do field cosine, every note's *field* must have its own
embedding. Today's sidecar stores one vector per note (the
whole body). Switching to field embeddings means:

- **Sidecar shape change.** Either per-field vectors in one
  sidecar, or one sidecar per field. Backward-compat with the
  current shape needs an explicit migration.
- **Embedding cost ~Nx.** Each note today produces one
  vector; field embeddings produce N vectors. Storage and
  embed-time both scale.
- **F9.2 lives here.** The deferred F9.2 (block-level /
  field-level granularity) is exactly this work. Promoting
  field queries means picking up F9.2.

## Connection to F9.2 (deferred)

F9.2 from the research log:
> Longer notes (especially episodes — narrative, multi-paragraph)
> may benefit from block-level vectors. Field-level is a special
> case of block-level where the "block" is a single frontmatter
> field's content.

If F9.2 ships, field queries become natural — fields are just a
specific kind of block. If field queries are the only motivation,
F9.2 is over-built; a simpler "embed-each-field" pass would do.

## Open design questions (for a future spec)

1. **Storage shape:** per-field vectors in one sidecar (map of
   field-name → vector) vs. per-field sidecars (e.g.,
   `<note>.situation.vec.json`). Per-sidecar wastes filesystem
   inodes; per-vector-map needs schema for the map.
2. **Combination rule:** how does a multi-field query combine
   into one item ranking? Geometric mean? Min? Weighted sum?
3. **Backward compatibility:** sidecars written under the
   single-vector shape need re-embedding. Bulk operation
   (`engram embed apply --field`?) or full reset?
4. **What about `time` queries?** Time isn't an embedding —
   it's a filter. Does `--field time=` route to the date-range
   filter instead of cosine?
5. **Query DSL:** is `--field name="phrase"` the right shape, or
   a single query expression like `situation:"X" subject:"Y"`?
6. **Episode body queries:** the L1 episode body is a transcript
   chunk, not a structured field. Does block-level embedding
   apply (split body into blocks) or stays whole-body?

## What to do for v3

Open a v3 spec when there's a concrete use case driving it —
e.g., "we want to answer 'what behavior was I doing last week'
and the whole-note vector isn't picking it up." Without a forcing
function, the cost (sidecar migration + ~8x embedding cost) is
hard to justify.

The v2 `/recall` SKILL.md notes this as a known limitation in
its "What this skill is not for" section so future-me sees the
limit and can decide to invest.

## Status

**Deferred to v3.** No v3 roadmap yet; this doc is a placeholder
to surface the question when the v3 roadmap gets drafted.
Tracked as GitHub issue
[#637](https://github.com/toejough/engram/issues/637). See also
issue [#636](https://github.com/toejough/engram/issues/636)
(project/issue metadata on vault notes) — an adjacent
vault-shape change deferred from the same /please pass.
