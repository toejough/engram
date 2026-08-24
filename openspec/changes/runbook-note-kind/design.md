## Context

Four content kinds exist today: `fact | feedback | qa` (`qa` splits into `qa-question`/`qa-answer` at the frontmatter `type` level). Each kind is wired through the same four touch points: `engram learn` (capture), `write-memory` (compose/execute the write), `recall`/`query` (retrieval), and skill docs. `qa` is the most recent addition (`openspec/specs/learn-qa-capture/spec.md`) and is the closest structural analog — its asymmetric question/answer split shows the pattern for adding a kind cleanly, including `isQueryExcludedKind` (`internal/cli/qa.go`) for kinds that need partial retrieval exclusion.

**Sequencing note (#727):** the qa references above are point-in-time. #727 (qa collapse) removes `qa-question`/`qa-answer`, `internal/cli/qa.go`'s pair machinery, and `isQueryExcludedKind` entirely. If #727 lands before this change, use the `fact`/`feedback` capture path as the implementation template instead, and read "NOT added to `isQueryExcludedKind`" as "no exclusion mechanism exists or is added" — Decision 2 (compete, don't exclude) is unaffected either way; it simply becomes the only behavior.

`runbook` notes have no reason for asymmetric exclusion — unlike a `qa-question` (which exists only to be pointed at), a `runbook` note's content (a task's strategy) is itself the retrieval target. It SHALL compete in the main matched set like `fact`/`feedback` notes.

## Goals / Non-Goals

**Goals:**
- Add `type: runbook` as a fully wired first-class kind, symmetric with `fact`/`feedback` in retrieval treatment (not asymmetric like `qa`).
- Schema that answers Joe's three questions (2026-08-23, vault note 789): when should you use this runbook; what are the steps (including which facts/feedback to read and consider); what should be true when you're done.
- Retrieval purely via situation-similarity, identical to how `fact`/`feedback` notes rank today — no new taxonomy, no new query flag, no new ranking mechanism.

**Non-Goals:**
- Per-note efficacy/outcome tracking (success_count/total_attempts, SPL-style ranking formula) — that's #718, explicitly deferred. (`done_when` is kept partly because it makes runbooks scorable for #718 later, but no scoring ships here.)
- An `inputs` signature / full calling convention — considered during the #728 adapter-experiment debate and rejected by Joe (2026-08-23); #728 tests its provisional `function` schema in isolated fixtures only, independent of this change.
- Any `task_type` classification field or taxonomy (fixed or free-form) — considered and rejected (2026-08-24 design review). The SPL benchmark numbers that motivated it measure SPL's whole bundled system (classification + strategy injection + refinement/pruning), not type-classification in isolation — no ablation in the source material isolates that lever, so it doesn't specifically justify adding one. Retrieval relies on situation-similarity alone, same as `fact`/`feedback`.
- Retrofitting existing `feedback` notes into `runbook` notes — this change adds the new kind; migrating the vault's existing kind-4 content is a follow-up, not part of this change's acceptance criteria.

## Decisions

**1. Kind name and frontmatter shape:** `type: runbook` (Joe, 2026-08-23, vault note 789 — chosen over `procedure`, `function`, and `strategy`; deliberately non-consonant with `fact`/`feedback`, the ops vibe is intended; `procedure` also collides with existing "recall procedure" prose and `function` with tool-calling vocabulary). Frontmatter:

- `type: runbook`
- `date: <YYYY-MM-DD>`
- `situation: <when should you use this runbook>` — the retrieval handle, phrased the way a future task would present; embedded as the situation vector exactly like `fact`/`feedback` (the sidecar already carries `situation_vector` + `body_vector`; no sidecar schema change).
- `done_when: <what should be true when you're done>` — required. The ending-expectations contract; kept deliberately (it also makes runbooks scorable for #718 later).
- `source: <provenance>`; optional `contributors: [<basename>, ...]`.
- Body: the numbered steps, which may `[[wikilink]]` fact/feedback notes to read and consider along the way.
- No `inputs` field, no `task_type` field (both rejected — see Non-Goals).

Filename: `runbook.<YYYY-MM-DD>.<slug>.md` — a kind-prefixed name like `qa.*`, NOT a Luhmann-numbered name. Deliberate: runbooks aren't part of the Luhmann hierarchy's zettelkasten branching structure, and the prefix keeps them out of `--reparent-luhmann` scope. (Retrieval symmetry with fact/feedback — embed + vocab + `candidate_l2s` — is unaffected by the filename.)

**2. Retrieval treatment — compete, don't exclude:** `runbook` gets no exclusion treatment (no `isQueryExcludedKind` membership while that mechanism exists — see the #727 sequencing note). It participates in `candidate_l2s` like `fact`/`feedback`. Rationale: a `runbook` note's whole content (the strategy itself) is the thing worth retrieving — there's no paired "question" half to hide.

**3. Surfacing mechanism — situation-similarity only, no new mechanism:** Considered a `task_type` pre-filter fed by a new `engram query --task-type <slug>` flag (mirroring the matched-note-floor pattern already shipped for note-vs-chunk ranking, `recall-matched-note-floor` spec / `capWithNoteFloor`), and rejected (2026-08-24 design review). `runbook` notes rank purely by situation-similarity, identical to `fact`/`feedback` — no new query flag, no new ranking logic, no recall Step-1 changes. Rationale: the rejected mechanism's justification leaned on SPL's benchmark numbers, which measure SPL's whole bundled system and don't isolate type-classification's own contribution — insufficient grounds for a new ranking mechanism and its attendant regression risk to existing retrieval. If task-type-based ranking is wanted later, it re-enters as its own proposal with its own evidence.

## Risks / Trade-offs

- **[Risk] #727 (qa collapse) lands first and deletes the qa template files this design cites** (`internal/cli/qa.go`, `isQueryExcludedKind`) → **Mitigation**: the Context sequencing note pins the fallback — template from the `fact`/`feedback` capture path; Decision 2 is unaffected.

## Migration Plan

No data migration — this is purely additive (new kind, no changes to existing `fact`/`feedback`/`qa` handling). Rollback is a revert of the four touch-point changes; no vault content needs to be un-migrated since existing kind-4 content stays in `feedback` notes unless a future, separate change proposes migrating it.

## Open Questions

None — with the task-type mechanism rejected (Decision 3), this change is a straightforward symmetric addition with no open ranking-mechanism questions left to resolve.
