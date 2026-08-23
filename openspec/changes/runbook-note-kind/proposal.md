## Why

Kind-4 confirmed-approaches ("what worked, do it again") currently live awkwardly inside `feedback` notes' action/behavior/impact fields rather than as reusable, retrievable, task-type-keyed runbooks. SPL research shows procedural-strategy injection lifts weak models materially (OptiLLMBench +4%, AIME24 +7%, Arena-Hard +8.6%). Joe decided (2026-08-08) this warrants a new first-class content kind rather than a tag convention on `feedback`; Joe settled the kind's name and schema (2026-08-23, vault note 789): the kind is **`runbook`** — deliberately non-consonant with `fact`/`feedback`, the ops vibe is intended — and its schema answers exactly three questions: *when should you use this runbook, what are the steps (including which facts/feedback to read and consider), and what should be true when you're done.* A fuller calling convention (an `inputs` signature, per the #728 adapter experiment's provisional function schema) was considered and rejected. Related: #718 (efficacy tracking — `done_when` deliberately makes runbooks scorable for it later), #685 (task-kind glance cue), #727 (qa collapse — sequencing note in design.md), #728 (adapter/function parity experiment — independent; reuses this change's task-type resolver), purview check #293.

## What Changes

- Add a new `runbook` note kind (frontmatter `type: runbook`), distinct from `fact | feedback | qa`, for task-type-keyed "how to approach X" strategy notes.
- Wire `runbook` through all four existing touch points, mirroring how `qa` was added end-to-end:
  - `engram learn` — capture path accepts/produces `runbook` notes.
  - `write-memory` — composes and executes the vault-write command for `runbook` notes.
  - `recall` / `query` — retrieval surfaces `runbook` notes; the surfacing mechanism is a `task_type` pre-filter fed by a new optional `engram query --task-type <slug>` flag (design.md Decision 3), gated on the validation evidence.
- Runbook schema = the three questions (design.md Decision 1): `situation` (when to use — the retrieval handle, embedded as the situation vector like fact/feedback), free-form `task_type` slug (the task-shape retrieval key), body of numbered steps that may `[[wikilink]]` fact/feedback notes to read and consider, and required `done_when` (ending expectations). No `inputs` signature.
- Introduce the task-type taxonomy as free-form, derived from the captured runbook's own subject rather than a fixed enum (SPL's 16-type set is benchmark-tuned, not engram-native). Anchor skill docs with 2-3 concrete engram examples (e.g. "running eval harnesses", "releasing Go modules", "recall-moment discovery").
- Validation gate before the surfacing mechanism ships (documented in design.md, executed as tasks, with pre-registered bars): (1) vault supply check — how much existing kind-4 content is already runbook-shaped, (2) A/B recall test on a recurring task class run to house eval-harness standards (tasks.md 3.2), (3) cross-check against SPL's own eval methodology.

## Capabilities

### New Capabilities
- `learn-runbook-capture`: `engram learn` capture path + frontmatter schema for the new `runbook` note kind, and the write-memory compose/execute wiring for it — mirrors `learn-qa-capture`'s role for `qa`.
- `recall-runbook-surfacing`: how `runbook` notes are retrieved and ranked in `recall`/`query` against situation-similarity, including the task-type taxonomy, the `engram query --task-type` interface, and the pre-filter surfacing mechanism.

### Modified Capabilities
(none — no existing spec's requirements change; `qa`, `fact`, `feedback` handling is untouched)

## Impact

- `internal/cli` — frontmatter schema, `engram learn runbook` capture wiring, and the `engram query --task-type` flag.
- `agent-instructions/skills/write-memory/SKILL.md` — compose/execute support for `runbook` writes (per `superpowers:writing-skills` TDD).
- `agent-instructions/skills/learn/SKILL.md` — capture guidance (when a runbook vs a feedback note) and task-type taxonomy examples.
- `agent-instructions/skills/recall/SKILL.md` — task-type inference + `--task-type` pass-through, and retrieval/ranking guidance for `runbook` notes.
- `internal/cli` query path — new kind participates in `candidate_l2s` plus the task-type pre-filter.
- Vault content — a supply-check mining pass over existing kind-4 feedback notes to validate real demand before the surfacing mechanism ships.
