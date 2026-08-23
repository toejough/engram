## Context

Four content kinds exist today: `fact | feedback | qa` (`qa` splits into `qa-question`/`qa-answer` at the frontmatter `type` level). Each kind is wired through the same four touch points: `engram learn` (capture), `write-memory` (compose/execute the write), `recall`/`query` (retrieval), and skill docs. `qa` is the most recent addition (`openspec/specs/learn-qa-capture/spec.md`) and is the closest structural analog — its asymmetric question/answer split shows the pattern for adding a kind cleanly, including `isQueryExcludedKind` (`internal/cli/qa.go`) for kinds that need partial retrieval exclusion.

**Sequencing note (#727):** the qa references above are point-in-time. #727 (qa collapse) removes `qa-question`/`qa-answer`, `internal/cli/qa.go`'s pair machinery, and `isQueryExcludedKind` entirely. If #727 lands before this change, use the `fact`/`feedback` capture path as the implementation template instead, and read "NOT added to `isQueryExcludedKind`" as "no exclusion mechanism exists or is added" — Decision 2 (compete, don't exclude) is unaffected either way; it simply becomes the only behavior.

`runbook` notes have no reason for asymmetric exclusion — unlike a `qa-question` (which exists only to be pointed at), a `runbook` note's content (a task-type + strategy) is itself the retrieval target. It SHALL compete in the main matched set like `fact`/`feedback` notes.

## Goals / Non-Goals

**Goals:**
- Add `type: runbook` as a fully wired first-class kind, symmetric with `fact`/`feedback` in retrieval treatment (not asymmetric like `qa`).
- Schema that answers Joe's three questions (2026-08-23, vault note 789): when should you use this runbook; what are the steps (including which facts/feedback to read and consider); what should be true when you're done.
- Free-form task-type taxonomy, seeded by 2-3 engram-native examples, not a fixed enum.
- A documented, evidence-gated surfacing mechanism (the `task_type` pre-filter), with a pinned CLI interface.
- A pre-ship validation gate with pre-registered bars: vault supply check, A/B recall test to house harness standards, SPL methodology cross-check.

**Non-Goals:**
- Per-note efficacy/outcome tracking (success_count/total_attempts, SPL-style ranking formula) — that's #718, explicitly deferred. (`done_when` is kept partly because it makes runbooks scorable for #718 later, but no scoring ships here.)
- An `inputs` signature / full calling convention — considered during the #728 adapter-experiment debate and rejected by Joe (2026-08-23); #728 tests its provisional `function` schema in isolated fixtures only, independent of this change.
- A fixed 16-type taxonomy — SPL's set is benchmark-tuned, not adopted.
- Retrofitting existing `feedback` notes into `runbook` notes — this change adds the new kind; migrating the vault's existing kind-4 content is a follow-up informed by the supply-check task, not part of this change's acceptance criteria.

## Decisions

**1. Kind name and frontmatter shape:** `type: runbook` (Joe, 2026-08-23, vault note 789 — chosen over `procedure`, `function`, and `strategy`; deliberately non-consonant with `fact`/`feedback`, the ops vibe is intended; `procedure` also collides with existing "recall procedure" prose and `function` with tool-calling vocabulary). Frontmatter:

- `type: runbook`
- `date: <YYYY-MM-DD>`
- `situation: <when should you use this runbook>` — the retrieval handle, phrased the way a future task would present; embedded as the situation vector exactly like `fact`/`feedback` (the sidecar already carries `situation_vector` + `body_vector`; no sidecar schema change).
- `task_type: <free-form-slug>` (e.g. `running-eval-harnesses`, `releasing-go-modules`, `recall-moment-discovery`), derived from the captured runbook's own subject at capture time — not selected from a hardcoded enum. Rationale: SPL's fixed taxonomy was tuned to their own benchmark suite; engram's task surface doesn't map onto it, and a free-form field lets the taxonomy emerge from actual usage (validated by the supply-check task) instead of being guessed up front.
- `done_when: <what should be true when you're done>` — required. The ending-expectations contract; kept deliberately (it also makes runbooks scorable for #718 later).
- `source: <provenance>`; optional `contributors: [<basename>, ...]`.
- Body: the numbered steps, which may `[[wikilink]]` fact/feedback notes to read and consider along the way.
- No `inputs` field (rejected — see Non-Goals).

Filename: `runbook.<YYYY-MM-DD>.<slug>.md` — a kind-prefixed name like `qa.*`, NOT a Luhmann-numbered name. Deliberate: runbooks are organized by `task_type`, not by position in the Luhmann hierarchy, and the prefix keeps them out of `--reparent-luhmann` scope. (Retrieval symmetry with fact/feedback — embed + vocab + `candidate_l2s` — is unaffected by the filename.)

**2. Retrieval treatment — compete, don't exclude:** `runbook` gets no exclusion treatment (no `isQueryExcludedKind` membership while that mechanism exists — see the #727 sequencing note). It participates in `candidate_l2s` like `fact`/`feedback`. Rationale: a `runbook` note's whole content (the task-type-keyed strategy) is the thing worth retrieving — there's no paired "question" half to hide.

**3. Surfacing mechanism — pre-filter on `task_type`, fed by `engram query --task-type <slug>`:** The query CLI grows one optional flag, `--task-type <free-form-slug>`. The recall skill infers the session's current task type during Step 1 and passes it; harness callers may pass it directly. When present, `runbook` notes whose `task_type` matches (exact or embedding-similar) are boosted/pre-filtered ahead of pure situation-similarity ranking, mirroring the matched-note floor pattern already shipped for note-vs-chunk ranking (`recall-matched-note-floor` spec, `capWithNoteFloor`). When absent, runbooks rank purely by situation-similarity like any other note. Rationale: adding a brand-new recall Step-1 phrase would touch the Step-1 sequencing depended on by other kinds (higher blast radius); a pre-filter/floor is additive and isolated to `runbook` notes. This choice and its rationale MUST be restated in the shipping commit/PR per the proposal's acceptance criteria.

**4. Validation gate is sequenced before the recall-ranking task, not after — and runs to house harness standards:** The supply-check (mining existing kind-4 feedback content) and the A/B recall test both need the capture path to exist first (to know what a real `runbook` note looks like) but must complete BEFORE the pre-filter ranking logic (Decision 3) ships, since the A/B test's outcome could revise the surfacing mechanism decision. The A/B test is an eval, and evals in this repo follow the measured harness rules (see tasks.md 3.2): headless `claude -p` fresh processes (subagent controls inherit session context), the real installed `engram` binary with per-trial `ENGRAM_VAULT_PATH` fixture vaults carrying real sidecars (no stubs — stub rank order has invalidated a batch before), a per-arm delivery gate verified from transcripts before any scoring, a noise floor sized from same-contrast repeats, and pass bars pre-registered before the first scored run. Tasks.md sequences: capture/schema → supply check → A/B test → (confirm or revise Decision 3) → ranking implementation → skill docs.

## Risks / Trade-offs

- **[Risk] Free-form `task_type` fragments into near-duplicate slugs** (e.g. `run-eval-harness` vs `running-eval-harnesses`), hurting the pre-filter's match rate → **Mitigation**: pre-filter matches by embedding similarity on `task_type`, not exact string equality, so near-duplicates still cluster; the supply-check task also surfaces early whether normalization guidance is needed in the `learn` skill docs.
- **[Risk] Pre-filter boost over-fires and buries good situation-similarity matches for non-matching task types** → **Mitigation**: the A/B recall test (acceptance criterion 3) is a hard gate before the ranking logic ships, not an optional nice-to-have; if it shows no lift or regression against its pre-registered bar, Decision 3 is revisited before implementation continues.
- **[Risk] Validation gate findings (supply check shows too little existing runbook-shaped content, or A/B test is flat) could invalidate the whole change post-capture-implementation** → **Mitigation**: this is intentional — the proposal explicitly makes the validation gate a pre-ship checkpoint, not a post-hoc formality; tasks.md must not implement the ranking/surfacing step until the gate passes, and if the gate fails, tasks.md's remaining ranking/doc tasks become the record of what to reconsider rather than being silently skipped.
- **[Risk] #727 (qa collapse) lands first and deletes the qa template files this design cites** (`internal/cli/qa.go`, `isQueryExcludedKind`) → **Mitigation**: the Context sequencing note pins the fallback — template from the `fact`/`feedback` capture path; Decision 2 is unaffected.

## Migration Plan

No data migration — this is purely additive (new kind, no changes to existing `fact`/`feedback`/`qa` handling). Rollback is a revert of the four touch-point changes; no vault content needs to be un-migrated since existing kind-4 content stays in `feedback` notes unless a future, separate change proposes migrating it.

## Open Questions

- Exact `task_type` embedding-similarity threshold for the pre-filter — left to implementation/eval, not fixed here.
- Whether the supply-check finds enough existing runbook-shaped content to bootstrap a meaningful A/B test population, or whether synthetic/seeded runbook notes are needed first — resolved by the supply-check task itself.
