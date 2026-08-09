## Context

`internal/cli/luhmann.go` implements three placements for a new note ID, already wired through
the CLI and (per the issue) through `write-memory`'s command composition:

- `--position top` → next unused top-level integer (`1`, `2`, `3`, ...)
- `--position continuation --target <id>` → next child of `<id>` (alternates digit/letter
  segments by depth: `1` → `1a`, `1a` → `1a1`)
- `--position sibling --target <id>` → next child of `<id>`'s parent (i.e., the next note at the
  same level as `<id>`)

Before commit f620bfaf (2026-06-12), the `learn` skill's Step 5 chose among these by inspecting
the note's relationship to recently-written/recalled notes. That call site was removed along with
an old episode-note workflow, and nothing replaced it — `learn` now always calls `--position top`
(implicit default), so all 447 existing vault notes are flat top-level IDs.

The canonical model this restores is Niklas Luhmann's paper-slip system, as documented in his own
description of the *Zettelkasten* ("Kommunikation mit Zettelkästen", 1981) and formalized for
digital practice by Sönke Ahrens (*How to Take Smart Notes*, 2017) and the broader
digital-zettelkasten community (e.g., Zettelkasten.de's "branches and links" writeups). The
consistent rule across these sources:

- A note's ID position encodes **where in the argument/thought tree** it sits, not chronology or
  topic-tagging (topic retrieval is already handled by tags/embeddings, per ADR-0025 — Luhmann
  himself resisted subject indexes for this reason, relying on ID-adjacency + links instead).
- **Child (continuation)**: when a note picks up *one specific point* raised inside another note
  and develops it further/deeper. The child inherits the parent's context implicitly; it would
  not stand alone as well without the parent.
- **Sibling**: when a note continues the *same line of thought* as another note at the same level
  — an alternative angle, a direct continuation, or the "next thing to say" about the same
  question — without being a sub-point of it.
- **Fresh top-level**: when the note starts an argument/thread with no direct developmental
  relationship to an existing note (a genuinely new topic entry point).

## Goals / Non-Goals

**Goals:**
- Restore a disposition decision in `learn` Step 2 that classifies each new note as
  top/continuation/sibling (+ target) before handing off to `write-memory`, using the rule above.
- Ground the rule in citable canonical practice, not reinvented ad hoc logic.
- Exercise `continuation`/`sibling` again on real captures.

**Non-Goals:**
- Changing `internal/cli/luhmann.go`, the CLI flags, or ID computation — the mechanism already
  works and is out of scope.
- Retroactively re-parenting the 447 existing top-level notes — this only changes disposition for
  *new* notes going forward.
- Introducing branching structure as a new retrieval channel — placement remains orthogonal to
  tag-based retrieval (ADR-0025) and supersession (ADR-0012); ADR-0007's rejection of
  wikilink-graph-for-retrieval is unaffected because this is not a retrieval mechanism.

## Decisions

**Decision 1 — Disposition runs against "recently-written or recalled notes this session" only.**
Luhmann's own process placed each new slip relative to the slip he was just working from — not a
global search of the whole archive. Scanning the *entire* vault for the best parent per note
would be expensive and outside what a human zettelkasten practice does. Alternative considered:
embedding-similarity search across the full vault for the nearest note — rejected as scope creep
(a retrieval-shaped operation, and ADR-0007 already found graph/similarity search doesn't help
retrieval; reusing it for placement risks the same null result and adds latency to every capture).

**Decision 2 — Three-way test, applied in this order:**
1. Does the new note develop **one specific sub-point** raised inside an existing note from this
   session (recently written or recalled)? → **continuation**, target = that note's ID.
2. Else, does it continue/extend the **same overall thought** as an existing note from this
   session at the same level (not a sub-point, but "more to say" on the same line)? →
   **sibling**, target = that note's ID.
3. Else → **top**.

This mirrors the literature's binary (branch deeper vs. continue same level) plus the necessary
default (no relation → new top-level thread), and matches the disposition step's original
placement in Step 2 (crystallization), right where the note's content and its relationship to
other in-session notes are already known.

**Decision 3 — Document the rule in `learn` Step 2, not as a new Step.** The pre-removal version
lived as old-SKILL.md §5; restoring it as a sub-step of the current Step 2 (rather than a new
top-level step) keeps it colocated with the point where write-memory is invoked, since disposition
is an input to that same handoff.

## Risks / Trade-offs

- **[Risk] Session-scope-only disposition may miss a good parent from a prior session.** →
  Mitigation: explicitly scope to this session per Decision 1; this matches Luhmann's own
  practice (relative to the slip in hand) and avoids an expensive full-vault search. Acceptable
  per the issue's framing — this restores mechanism, not optimal parenting.
- **[Risk] Ambiguity between continuation and sibling could make the step inconsistently applied.**
  → Mitigation: the ordered test in Decision 2 forces continuation to be checked first (does it
  answer/develop a sub-point?) before falling back to sibling, reducing judgment calls.
- **[Risk] No automated test can assert "correct" placement (it's a judgment call).** → Mitigation:
  the writing-skills TDD RED/GREEN cycle asserts *mechanism* is exercised (continuation/sibling
  IDs appear in captures with a clear branching relationship in a scripted scenario), not semantic
  correctness of every future judgment call.

## Open Questions

- Does `write-memory`'s current handoff contract already pass through `position`/`target` fields
  transparently, or does it need an explicit update to accept them from `learn`? (Verify against
  `agent-instructions/skills/write-memory/SKILL.md` during task execution.)
