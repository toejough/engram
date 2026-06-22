# Opus-Trap Catalog & Memory-Validation Exercises — Plan

> **For agentic workers:** this is a research/catalog + spec deliverable (no production code this
> pass). Steps produce a committed markdown artifact: a catalog of opus correction-traps and
> minimal exercise specs to validate that memory prevents re-correction.

**Goal:** Catalog the conditions under which opus needed correction (or engram recall changed
course) across the user's session history, abstract each into a lesson, and spec a *minimal,
cheap* exercise per lesson that recreates the triggering condition — so memory's payoff (the
correction never has to be given twice) can be validated/invalidated quickly.

**North star (user, verbatim intent):** *"When I give a correction, I don't want to have to give
it again."* The success criterion for every exercise is therefore: it recreates the condition that
provoked the correction, cold opus predictably re-commits the error, and an agent carrying the
lesson (warm) does not. Whether the trap is a "genuine blind spot" or "repo-specific knowledge"
does not matter — both are in scope.

**Architecture:** Three correction sources, merged. Pre-abstracted lessons (vault + CLAUDE.md)
give the *lesson*; raw transcripts give the *triggering condition* and the *cold wrong-move*. The
output is a single catalog doc; exercises are SPECS, not built/run this pass.

**Tech stack:** Python over the JSONL session logs + markdown vault notes; engram binary for
recall context only. No new Go code.

## Global Constraints

- **Spec only this pass** — no exercises built, no opus runs. (User decision.)
- **Catalog comprehensively** — every correction-condition is in scope; no blind-spot filtering.
- **Exercises must be cheap to validate** — prefer a *deterministic* pass-check (grep/compile/test
  exit code) over an LLM judge, so cold-vs-warm validation is fast and unambiguous.
- **Honesty about corpus skew** — opus sessions are engram-dominated (21 engram, 14 synthetic
  please-tests, ~6 elsewhere). Report this; do not imply broad cross-project coverage we lack.
- **Don't lean on stale eval recommendations** found in logs (e.g. old L1/L2/L3 "opus + l2"
  conclusions) — they predate the current architecture. Mine logs for *what opus got wrong*, not
  for old recommendations. (Per vault lesson `feedback-stale-eval-results-dont-bind-new-architecture`.)

---

## Sources (exact)

- **A — engram vault feedback notes** (~23): `~/.local/share/engram/vault/*.md` with
  `type: feedback`. Pre-abstracted corrections with situation/behavior/impact/action fields.
- **B — global CLAUDE.md + auto-memory** : `/Users/joe/.claude/CLAUDE.md` ("Critical Warnings"
  / "Non-negotiable rules I've violated repeatedly") and
  `/Users/joe/.claude/projects/-Users-joe-repos-personal-engram/memory/feedback_*.md` (~22).
- **C — raw opus transcripts** (42 opus-tagged files, engram-heavy):
  `~/.claude/projects/*/*.jsonl` where `message.model` starts `claude-opus`, excluding
  `subagents/`, `cummatrix*`, `gate-lazy*`. Mine for USER corrections of an opus action and for
  recall-induced course changes.

## Catalog schema (one row per trap)

| field | meaning |
| --- | --- |
| `id` | short slug |
| `lesson` | the abstracted principle (the correction, generalized) |
| `trigger_condition` | the concrete situation that provoked it — **this is what an exercise must recreate** |
| `cold_wrong_move` | what opus did / defaults to absent the lesson |
| `correction` | what the user said to do instead |
| `source` | A/B/C + locator (note slug or `file#turn`) |
| `buildable` | can the condition be recreated as a small code/task exercise? (yes/reasoning-only) |
| `cold_falls_in` | likelihood cold opus re-commits it (high/med/low) — drives exercise value |

## Exercise spec format (one per catalogued trap that is `buildable` + `cold_falls_in: high|med`)

```
EX-<id>:
  setup:        minimal repo/task state that recreates trigger_condition
  prompt:       the instruction given to the model (no mention of the lesson)
  cold_predict: the specific wrong output cold opus is expected to produce
  pass_check:   DETERMINISTIC check (grep/compile/test/exit-code) — true iff lesson applied
  warm_input:   the memory note that should flip cold→pass
  cost:         est. tokens/$ to run one cold + one warm trial
```

---

## Tasks

### Task 1: Harvest pre-abstracted corrections (A + B)
- Read all vault `type: feedback` notes + auto-memory `feedback_*.md` + CLAUDE.md warnings.
- Emit one catalog row per distinct lesson (dedup A vs B overlaps), filling `lesson`, `correction`,
  `source`; leave `trigger_condition`/`cold_wrong_move` to enrich from C.
- Deliverable: partial catalog (lesson-side populated).

### Task 2: Mine raw opus transcripts (C) — fan out
- Split the 42 opus files into batches; dispatch one mining subagent per batch (route doctrine:
  cheap model, recall-first). Each returns structured records: USER-correction-of-opus-action and
  recall-induced course-changes, with `trigger_condition`, `cold_wrong_move`, `file#turn`.
- Merge into the catalog: attach trigger/cold-move to existing lesson rows; add new rows for
  uncrystallized corrections.
- Deliverable: full catalog, all fields populated.

### Task 3: Classify buildability + cold-falls-in
- For each row, judge `buildable` (code/task exercise vs reasoning-only) and `cold_falls_in`
  (does cold opus actually re-commit it? — the saturation check: a trap opus avoids unprompted is
  a weak exercise).
- Deliverable: catalog with classification.

### Task 4: Spec the exercises
- For each `buildable` + `cold_falls_in ∈ {high,med}` row, write the exercise spec (format above).
- Prioritize a deterministic `pass_check`. Flag any that can only be judged by an LLM (more
  expensive/noisier to validate) as lower-priority.
- Deliverable: exercise spec list, ranked by (cold_falls_in × cheapness).

### Task 5: Validation protocol (designed, not run)
- Describe the cold-vs-warm loop per exercise: cold opus → `pass_check` should FAIL; warm opus
  (lesson injected) → should PASS. Note how this plugs into the existing cumulative harness +
  saturation gate (task #39).
- Deliverable: protocol section + an explicit "what would invalidate the payoff" criterion
  (if cold opus passes `pass_check` unaided, the exercise is saturated — drop it).

### Task 6: Assemble + commit the doc
- Single doc: `dev/eval/cumulative/OPUS-TRAP-CATALOG.md` (catalog + specs + protocol). Link from
  `EXPERIMENT-LOG.md` and task #39.

## Self-review checklist
- Every correction source (A/B/C) represented? Corpus skew stated honestly?
- Each exercise spec has a deterministic pass_check or is flagged as LLM-judged?
- Each exercise recreates a real `trigger_condition`, not an invented one?
- `cold_falls_in` honestly assessed (no exercise that opus passes unaided)?
