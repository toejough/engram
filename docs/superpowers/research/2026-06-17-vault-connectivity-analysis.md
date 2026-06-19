# Vault connectivity analysis — why the web is empty, and what (if anything) to do

**Date:** 2026-06-17
**Status:** recommendation — no code changes; decision pending
**Ask:** "Why aren't there many connections in our vault? Should we expect a better-connected web by now? If not, by when?"

---

## TL;DR

The vault isn't *sparsely* connected — it has **zero connections**. 34 notes, 0 resolved
wikilink edges, every note an orphan, 34 single-note components. This is **mechanism-gated** — a
limit imposed by code, not by data or elapsed time: the only link-writing path reachable from
our skills is `engram learn --relation`, and **no skill ever invokes it**. So:

1. **Why so few?** Every note is born an orphan. Neither `learn` nor recall's Step 2.5
   crystallization passes `--relation`, so notes emerge unlinked — they get only a `--position`
   placement, which allocates a Luhmann *ID*, not an edge.
2. **Should we expect better by now?** No. Connectivity does not accrue with time or use under
   the current build. Waiting longer produces zero edges, same as today. The premise that a web
   should be *emerging* assumes a process that does not exist.
3. **By when?** Not a date — a **dependency**. Edges start accruing the first recall *after* an
   auto-linking change ships. That change is small: the binary already supports `--relation` and
   the design is specced (only "pending review"), so wiring it into recall is ~a day's work, and
   active use would fill a connected backbone within weeks. Whether to ship it is itself a value
   question — does the note graph actually improve recall? — which needs its own value-proof
   (currently untracked; *not* #646, which covers recency recall only). Treat that as the gate on
   the *decision*, not a blocker on the *estimate*.

---

## 1. Measured state (the actual graph)

Vault: `/Users/joe/.local/share/engram/vault` — flat, 34 `*.md` notes (each with a `.vec.json`
sidecar).

| Metric | Value |
| --- | --- |
| Notes | 34 |
| Raw `[[…]]` occurrences | 2 (both in ONE note, identical) |
| Distinct authored edges | 1 |
| **Resolved edges** (target is a real vault note) | **0** |
| Notes with an outgoing resolved link | 0 |
| Notes with an incoming link | 0 |
| Orphans (degree 0) | **34 (all)** |
| Connected components | **34** (largest = 1) |
| Degree distribution (min/median/max) | 0 / 0 / 0 |
| In-degree hubs | none |

Cross-checked against the binary's own invariant checker: `engram check` reports
`WARN G3 dangling: 1 authored links target no note` — the one link
(`33.…enforce-discipline-not-teach → [[feedback_skill_edits_writing_skills]]`) points at a bare
slug matching a Claude auto-memory filename, **not** a Luhmann-keyed vault note, so it resolves
to nothing and is dropped from the graph.

Corroborated live by recall's own `budget` block (`total_notes: 33` at query time),
`hops_traversed: 1`, `hubs_returned: 0` — the 3-hop wikilink BFS died after one hop and found
zero hubs. Engram's own traversal confirms the graph is edgeless.

The count drifts upward as recall crystallizes notes: this very review added one (note 55, via a
gate reviewer's recall) — itself an orphan, because Step 2.5 wrote it with no `--relation`. The
thesis demonstrating itself.

## 2. Why — the mechanism (root cause)

A note↔note `[[wikilink]]` is written through **one code path reachable from our skills**:

- `engram learn --relation "<target>|<rationale>"` → `renderRelatedSection`
  (`internal/cli/learn.go:926-943`) emits `- [[target]] — rationale.` into the **new note's**
  body. One-directional (no back-link is written into the target).

(A second writer exists in code — `episodeRelatedBullets` (`learn.go:591-622`), which emits
`[[links]]` for `engram learn episode` — but the `episode` subcommand is retired and no skill
invokes it, so it never runs.)

Everything else that looks structural is not an edge:

- `--target` / `--position {top,continuation,sibling}` only allocate a **Luhmann ID** for the
  filename (`internal/cli/luhmann.go`). `continuation`/`sibling` produce a child/sibling *ID*,
  never a wikilink.
- `internal/vaultgraph` builds edges **purely** from parsed `[[wikilinks]]`
  (`scanner.go:76`, `graph.go:73`); `LuhmannID` is never used to build edges (it's read only for
  start-point tie-breaking in `selector.go`). So ID hierarchy contributes no connectivity — and
  the 2 compound-ID `continuation` children in the vault are *not* linked to their ID parents.

And the two skills that actually create notes never use `--relation`:

- **`skills/learn/SKILL.md`** writes only `--position top` leaves (corrections + save-requests).
- **`skills/recall/SKILL.md` Step 2.5** writes `--position top` (CREATE band) or
  `--target … --position continuation` (UPDATE band) — placement, not linkage. No `--relation`.

Net: **every note is born an orphan, and stays one.** The lone `[[…]]` in the vault isn't from a
linking mechanism at all — it's incidental prose typed into note 33's `action:` text ("See
`[[feedback_skill_edits_writing_skills]]`"), referencing a Claude auto-memory slug.
`ParseWikilinks` scans all body text so it counts as an edge, but it dangles because the slug
isn't a Luhmann basename.

## 3. Should we expect a better-connected web "by now"?

No — and the framing is the thing to correct. "By now" implies edges *accrue* with vault age or
usage, the way a human zettelkasten thickens as you add cross-references. Under the current
build there is no such accrual: connectivity is a step function of **whether a linking mechanism
runs**, and none does. 34 notes over ~a week produced 0 edges; 340 notes over a year would
produce 0 edges. Age and volume are not the variables.

## 4. "By when?" — a dependency, not a date

Edges begin accruing on the **first recall after** an auto-linking change ships. Components:

- **Binary half: already shipped.** `--relation` works (`learn.go:926-943`); `--synthesize-l2`
  / `nearest_l2` cluster output works (`query.go`).
- **Linking half: designed, not shipped.** `docs/superpowers/specs/2026-06-09-lazy-l2-synthesis-design.md`
  (status: *"design, pending review"*) specifies new L2s carrying `--relation` links "down to
  the cluster's L1s and L2s." That instruction never reached `recall/SKILL.md`.

The cheapest high-value increment: make Step 2.5's **UPDATE band** (cosine 0.80–0.95) pass
`--relation <nearest_l2>` in addition to its current `--target … --position continuation`. That
turns an existing, frequently-hit placement into a real edge. Many of this session's recall
note-clusters had `nearest_l2.cosine` in 0.79–0.92 — i.e. the UPDATE band fires often — so
edges would accrue at roughly the crystallization rate: a connected backbone within a few weeks
of active use. **(Projection, not a guarantee — contingent on §5.)**

## 5. The real question: do links earn their keep?

Before building auto-linking, challenge the goal. The case for caution:

- **Chunk-count dominance, not a note-ranking problem.** In this session's recall, 17 of 20
  surfaced items were raw chunks; only 3 were notes. With ~34 notes against a large chunk
  corpus, the bulk of retrieval hits come from the chunk space simply by count. Recency is *not*
  the differentiator here — per `c1-system-context.md` it applies to **both** chunks and notes
  (keyed on `LastUsed`, a deliberate evolution from the v1 chunks-only scope); this is a count
  ratio, not a mechanism gap. As the vault grows, notes will surface more, so the imbalance is
  partly transient.
- **The dead part is the note *graph*, not note ranking.** Notes still rank fine via cosine +
  recency. What 0 edges kills is the graph-only machinery: 3-hop BFS expansion (pulls in
  related-but-not-directly-matched notes) and in-degree top-5 hubs (anchor concepts). Both are
  inert today (`hops_traversed: 1`, `hubs_returned: 0`). Links would switch them back on —
  whether *that* beats the current ranking is unproven — and needs its own value-proof (untracked;
  distinct from #646, which is the recency-recall proof).

So the decision isn't "links good, build them." It's: **does graph-based recall (BFS expansion +
hubs) measurably beat the cosine+recency ranking over chunks and notes we already have?** If
yes, ship the cheap UPDATE-band `--relation` change and the web fills in. If we can't show it
helps, an empty graph is the *correct* state for a recall path that doesn't traverse it.

## 6. Recommendation

1. **Reframe the expectation:** the vault is edgeless by mechanism, not by youth. Nothing to
   "wait for."
2. **One concrete bug surfaced (your call whether to act):** the lone `[[…]]` dangles — note
   33's `action:` text references a Claude auto-memory slug (`feedback_skill_edits_writing_skills`)
   that isn't a vault basename. Worth fixing whenever links start to matter: either correct/remove
   it, or have the link-writing path validate targets so a malformed link fails loud instead of
   silently dangling. Could be filed as an issue.
3. **Gate the web on value, not aesthetics:** decide via a dedicated graph value-proof (untracked —
   *not* #646, which covers recency recall) whether BFS-expansion + hubs beat the current ranking.
   If yes, the minimal build is the in-place `amend` capability + `--relation` in recall Step 2.5's
   bands (see the lazy-L2 design). If no, consciously accept an empty graph.

**No build is committed by this analysis.** Next step is the user's call on whether to pursue the
value-proof / the UPDATE-band linking, or to leave the graph empty by design.
