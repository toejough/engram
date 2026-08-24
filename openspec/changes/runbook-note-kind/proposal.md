## Why

Kind-4 confirmed-approaches ("what worked, do it again") currently live awkwardly inside `feedback` notes' action/behavior/impact fields rather than as reusable, retrievable runbooks. SPL research shows procedural-strategy injection lifts weak models materially (OptiLLMBench +4%, AIME24 +7%, Arena-Hard +8.6%). Joe decided (2026-08-08) this warrants a new first-class content kind rather than a tag convention on `feedback`; Joe settled the kind's name and schema (2026-08-23, vault note 789): the kind is **`runbook`** — deliberately non-consonant with `fact`/`feedback`, the ops vibe is intended — and its schema answers exactly three questions: *when should you use this runbook, what are the steps (including which facts/feedback to read and consider), and what should be true when you're done.* A fuller calling convention (an `inputs` signature, per the #728 adapter experiment's provisional function schema) and a `task_type` classification field (SPL's fixed-taxonomy + pre-filter idea) were both considered and rejected (2026-08-24 design review) — SPL's cited benchmark numbers measure its whole bundled system (classification + strategy injection + refinement/pruning), not type-classification in isolation, so they don't specifically justify adding one. Related: #718 (efficacy tracking — `done_when` deliberately makes runbooks scorable for it later), #685 (task-kind glance cue), #727 (qa collapse — sequencing note in design.md), #728 (adapter/function parity experiment — independent), purview check #293.

## What Changes

- Add a new `runbook` note kind (frontmatter `type: runbook`), distinct from `fact | feedback | qa`, for "how to approach X" strategy notes.
- Wire `runbook` through all four existing touch points, mirroring how `qa` was added end-to-end:
  - `engram learn` — capture path accepts/produces `runbook` notes.
  - `write-memory` — composes and executes the vault-write command for `runbook` notes.
  - `recall` / `query` — retrieval surfaces `runbook` notes; they compete in the main matched set purely on situation-similarity, identical to `fact`/`feedback` (no new query flag, no exclusion treatment).
- Runbook schema = the three questions (design.md Decision 1): `situation` (when to use — the retrieval handle, embedded as the situation vector like fact/feedback), body of numbered steps that may `[[wikilink]]` fact/feedback notes to read and consider, and required `done_when` (ending expectations). No `inputs` signature, no `task_type` field.

## Capabilities

### New Capabilities
- `learn-runbook-capture`: `engram learn` capture path + frontmatter schema for the new `runbook` note kind, and the write-memory compose/execute wiring for it — mirrors `learn-qa-capture`'s role for `qa`.
- `recall-runbook-surfacing`: `runbook` notes retrieve and rank in `recall`/`query` purely by situation-similarity, symmetric with `fact`/`feedback` — no exclusion treatment, no task-type mechanism.

### Modified Capabilities
(none — no existing spec's requirements change; `qa`, `fact`, `feedback` handling is untouched)

## Impact

- `internal/cli` — frontmatter schema and `engram learn runbook` capture wiring.
- `agent-instructions/skills/write-memory/SKILL.md` — compose/execute support for `runbook` writes (per `superpowers:writing-skills` TDD).
- `agent-instructions/skills/learn/SKILL.md` — capture guidance (when a runbook vs a feedback note).
- `agent-instructions/skills/recall/SKILL.md` — retrieval/ranking guidance for `runbook` notes (situation-similarity only, same treatment as fact/feedback).
- `internal/cli` query path — new kind participates in `candidate_l2s`.
