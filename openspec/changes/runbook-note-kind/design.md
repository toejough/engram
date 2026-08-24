## Context

Three content kinds exist today: `fact | feedback | qa` (`qa` splits into `qa-question`/`qa-answer` at the frontmatter `type` level, via a bespoke standalone implementation — `internal/cli/qa.go` — that skips the shared capture pipeline entirely). `fact` and `feedback` are `runbook`'s actual structural analog: both are captured as named subcommands inside the same `learn` targ.Group, sharing one generic pipeline (`RunLearn` → Luhmann disposition via `nextLuhmannID` → lock → embed → vocab → `learnPath`'s `<luhmann-id>.<date>.<slug>.md` naming), differentiated only by frontmatter shape and required fields (`CommonLearnArgs` already carries the shared `--target`/`--position` disposition flags). `qa` is structurally different by design — it exists to be partially excluded from retrieval (`isQueryExcludedKind`) and it skips Luhmann disposition entirely — which is why it is NOT the template here. (An earlier draft of this change wrongly cited `qa.go` as the capture template and gave `runbook` a kind-prefixed flat filename with no Luhmann ID; both corrected 2026-08-24 design review — see Decision 1.)

`runbook` notes have no reason for asymmetric exclusion — unlike a `qa-question` (which exists only to be pointed at), a `runbook` note's content (a task's strategy) is itself the retrieval target. It SHALL compete in the main matched set like `fact`/`feedback` notes, and — like `fact`/`feedback` — it SHALL receive a Luhmann ID via the same disposition judgment, not a kind-prefixed flat filename (Decision 1). #727 (qa collapse, which removes `qa-question`/`qa-answer`/`internal/cli/qa.go`/`isQueryExcludedKind`) has no bearing on this change, since `runbook` depends on none of that machinery.

## Goals / Non-Goals

**Goals:**
- Add `type: runbook` as a fully wired first-class kind, symmetric with `fact`/`feedback` in both capture treatment (reuses the shared `learn` pipeline and Luhmann disposition, not a bespoke standalone implementation like `qa`) and retrieval treatment (not asymmetric like `qa`).
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
- `luhmann: <ID>` — runbooks participate in the Luhmann hierarchy exactly like `fact`/`feedback` (see filename/ID rationale below).
- `situation: <when should you use this runbook>` — the retrieval handle, phrased the way a future task would present; embedded as the situation vector exactly like `fact`/`feedback` (the sidecar already carries `situation_vector` + `body_vector`; no sidecar schema change).
- `done_when: <what should be true when you're done>` — required. The ending-expectations contract; kept deliberately (it also makes runbooks scorable for #718 later).
- `source: <provenance>`; optional `contributors: [<basename>, ...]`.
- Body: the numbered steps, which may `[[wikilink]]` fact/feedback notes to read and consider along the way.
- No `inputs` field, no `task_type` field (both rejected — see Non-Goals).

Filename/ID: `<luhmann-id>.<YYYY-MM-DD>.<slug>.md` — identical scheme to `fact`/`feedback`, distinguished only by the `type:` frontmatter field; the Luhmann ID is assigned via the same top/continuation/sibling disposition judgment (`--target`/`--position` flags, `nextLuhmannID`) fact/feedback capture already uses. **Corrects an earlier draft** (2026-08-24 design review) that gave `runbook` a kind-prefixed flat filename (`runbook.*`, no Luhmann ID, modeled on `qa.*`) on the reasoning that runbooks lack a natural parent idea to attach to. That reasoning doesn't hold: `position=top` mints a fresh top-level ID with no target required (`internal/cli/luhmann.go:126`), so "no clear relationship yet" was never a real blocker — every `fact`/`feedback` note already handles that exact case via `position=top`. And this change's own "Why" states kind-4 content currently lives *inside* `feedback` notes — a standing content relationship — meaning a captured runbook's disposition will often genuinely be `continuation`/`sibling` of a related feedback note, not `top`. Luhmann participation costs nothing and is strictly more expressive than opting out; `--reparent-luhmann` (`update-reparent-luhmann-batch` spec) covers runbook notes automatically, with no code change needed there, since it operates on Luhmann IDs generically regardless of kind.

**2. Retrieval treatment — compete, don't exclude:** `runbook` gets no exclusion treatment (no `isQueryExcludedKind` membership — a mechanism only `qa-question` uses). It participates in `candidate_l2s` like `fact`/`feedback`. Rationale: a `runbook` note's whole content (the strategy itself) is the thing worth retrieving — there's no paired "question" half to hide.

**3. Surfacing mechanism — situation-similarity only, no new mechanism:** Considered a `task_type` pre-filter fed by a new `engram query --task-type <slug>` flag (mirroring the matched-note-floor pattern already shipped for note-vs-chunk ranking, `recall-matched-note-floor` spec / `capWithNoteFloor`), and rejected (2026-08-24 design review). `runbook` notes rank purely by situation-similarity, identical to `fact`/`feedback` — no new query flag, no new ranking logic, no recall Step-1 changes. Rationale: the rejected mechanism's justification leaned on SPL's benchmark numbers, which measure SPL's whole bundled system and don't isolate type-classification's own contribution — insufficient grounds for a new ranking mechanism and its attendant regression risk to existing retrieval. If task-type-based ranking is wanted later, it re-enters as its own proposal with its own evidence.

## Risks / Trade-offs

None currently identified — with `qa.go` no longer the template (Decision 1/Context) and the task-type mechanism rejected (Decision 3), this change reuses the existing `fact`/`feedback` capture and retrieval pipeline in full, with no bespoke machinery of its own left to carry risk.

## Migration Plan

No data migration — this is purely additive (new kind, no changes to existing `fact`/`feedback`/`qa` handling). Rollback is a revert of the four touch-point changes; no vault content needs to be un-migrated since existing kind-4 content stays in `feedback` notes unless a future, separate change proposes migrating it.

## Open Questions

None — with the task-type mechanism rejected (Decision 3) and Luhmann participation restored to match `fact`/`feedback` (Decision 1), this change is now a straightforward, fully symmetric extension of the existing capture and retrieval pipeline, with no open mechanism questions left to resolve.
