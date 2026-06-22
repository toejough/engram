# Opus-Trap Catalog & Memory-Validation Exercises — Plan

> **For agentic workers:** this is a research/catalog + spec deliverable (no production code this
> pass). Steps produce a committed markdown artifact: a catalog of opus correction-traps and
> minimal exercise specs to validate that memory prevents re-correction.

**Goal:** Catalog the conditions under which opus needed correction (or engram recall changed
course) across the user's session history, abstract each into a lesson, and spec a *cheap*
exercise per lesson that recreates the triggering condition — so memory's payoff (the correction
never has to be given twice) can be validated/invalidated quickly.

**North star (user, verbatim intent):** *"When I give a correction, I don't want to have to give
it again."* The success criterion for every exercise: it recreates the condition that provoked the
correction, cold opus predictably re-commits the error, and an agent carrying the lesson (warm)
does not. Whether the trap is a "genuine blind spot" or "repo-specific knowledge" is out of scope
as a filter — both are catalogued.

**Architecture:** Three correction sources, merged. Pre-abstracted lessons (vault + CLAUDE.md)
give the *lesson*; raw transcripts give the *triggering condition* and the *cold wrong-move*. The
output is a single catalog doc; exercises are SPECS, not built/run this pass.

**Tech stack:** Python over the JSONL session logs + markdown vault notes; cheap-model mining
subagents. Recall context only from the engram binary. No new Go code, **no opus runs.**

## Global Constraints

- **Spec only this pass** — no exercises built, no opus runs. (User decision.)
- **Catalog comprehensively** — every correction-condition is in scope; no blind-spot filtering.
- **Mining uses a CHEAP model** (haiku/sonnet), not opus — this is text analysis over transcripts,
  not a capability test. Opus is the *subject* of the catalog, never a miner.
- **Exercises must be cheap to validate** — prefer a *deterministic* pass-check (grep/compile/test
  exit code) over an LLM judge, so cold-vs-warm validation is fast and unambiguous.
- **Corpus skew is real and stated** (verified counts below) — opus sessions are engram-dominated.
  Do not imply broad cross-project coverage we lack.
- **Don't lean on stale eval recommendations** found in logs (e.g. old L1/L2/L3 "opus + l2"
  conclusions) — they predate the current architecture. Mine logs for *what opus got wrong*, not
  for old recommendations. (Per vault lesson `feedback-stale-eval-results-dont-bind-new-architecture`.)

---

## Sources (exact, verified counts)

- **A — engram vault feedback notes (23):** `~/.local/share/engram/vault/*.md` with
  `type: feedback`. Pre-abstracted corrections with `situation`/`behavior`/`impact`/`action`
  frontmatter fields. **Example shape** (the structure miners synthesize toward):
  `situation: "<when this applies>"`, `behavior: "<what was done wrong>"`,
  `impact: "<why it cost>"`, `action: "<what to do instead>"`.
- **B — global CLAUDE.md + auto-memory (22):** `/Users/joe/.claude/CLAUDE.md` § "Critical
  Warnings" ("Non-negotiable rules I've violated repeatedly") and
  `/Users/joe/.claude/projects/-Users-joe-repos-personal-engram/memory/feedback_*.md`.
- **C — raw opus transcripts (38 files, verified):** `~/.claude/projects/*/*.jsonl` where
  `message.model` starts `claude-opus`, **excluding `subagents/` paths** (the only exclusion that
  removes anything — 271 opus subagent sessions; cummatrix/gate-lazy dirs hold zero opus files).
  **Verified corpus skew:** 20 engram, 14 synthetic `please-tdd` pressure-tests, 4 elsewhere
  (toejough-github-io ×2, dotfiles ×2). State this skew in the catalog; the cross-project signal
  is thin by nature, not by under-mining.

**Large-file handling:** two opus files are 34 MB / 24 MB (single long engram sessions). Mining
MUST stream/chunk: extract only USER turns + the opus ASSISTANT turn immediately preceding each,
via a Python pre-pass that emits a compact `(file#turn, user_text, prior_assistant_action)` list —
never feed a 34 MB JSONL into a subagent whole. Subagents mine the compact extract, not raw logs.

## Catalog schema (one row per trap)

| field | meaning |
| --- | --- |
| `id` | short slug |
| `source_type` | **`user_correction`** (user intervened after an opus action) or **`engram_course_change`** (recall changed course before failure). Distinct because they imply different cold-baseline confidence. |
| `lesson` | the abstracted principle (the correction, generalized) |
| `trigger_condition` | the concrete situation that provoked it — **this is what an exercise must recreate** |
| `cold_wrong_move` | what opus did / defaults to absent the lesson |
| `correction` | what the user said to do instead |
| `source` | A/B/C + locator (note slug or `file#turn`) |
| `crystallized` | is this lesson already represented by a source-A or -B note? (yes → enrich; no → net-new) |
| `buildable` | `code` (recreatable as a small build/task) or `reasoning_only` |
| `cold_falls_in` | **pre-empirical estimate** (confirmed only by a cold trial): `high`/`med`/`low` per the saturation thresholds below |

**What counts as a `user_correction` in a transcript (the mining marker):** a USER turn that
edits, reverts, or redirects opus's immediately-prior action — recognized by (a) imperative
negation/redirection ("don't", "no,", "instead", "use X not Y", "that's not how we…", "stop"),
or (b) the user re-stating a requirement opus violated, or (c) the user reverting/replacing opus's
output. A `engram_course_change` is a turn where a `/recall` result visibly altered the stated plan.

**`cold_falls_in` thresholds (estimate now, confirm with one cold trial later):**
`high` = expected cold opus re-commits in ≥50% of trials; `med` = 20–50%; `low` = <20% (opus
mostly avoids it unprompted → weak exercise, deprioritize). These are judgments this pass; the
validation protocol (Task 5) confirms them empirically before any exercise is built.

## Exercise spec format (one per `buildable: code` row with `cold_falls_in ∈ {high, med}`)

```
EX-<id>:
  complexity_tier: simple-check | customization | app-build   (+ one line: why this is the
                   MINIMUM-viable form that recreates the trigger — no over-building)
  setup:           minimal repo/task state that recreates trigger_condition
  prompt:          the instruction given to the model (NO mention of the lesson)
  cold_predict:    the specific wrong output cold opus is expected to produce
  pass_check_type: deterministic | hybrid | heuristic        (deterministic preferred; hybrid =
                   deterministic gate + content check; heuristic = LLM-judged, lowest priority)
  pass_check:      the concrete check — true IFF the lesson was applied (e.g. `grep -q PATTERN`,
                   `go build`, a test exit code)
  warm_input:      the memory note (source A/B/C row) that should flip cold→pass
  lesson_delivery: how warm receives it — via `engram recall` surfacing the note into the prompt
                   (the real mechanism), NOT a hand-pasted hint
  cost:            est. USD for one cold + one warm trial (price one chain at the model under test)
```

`cheapness` for ranking = **estimated USD of one cold + one warm trial** (lower = cheaper).
Final exercise ranking = `cold_falls_in` tier first (high > med), then cheapest within tier.

---

## Tasks

### Task 1: Harvest pre-abstracted corrections (A + B)
- Read all vault `type: feedback` notes + auto-memory `feedback_*.md` + CLAUDE.md § Critical
  Warnings. Emit one catalog row per distinct lesson (dedup A vs B overlaps), filling
  `source_type` (these are nearly all `user_correction`), `lesson`, `correction`, `source`,
  `crystallized: yes`; leave `trigger_condition`/`cold_wrong_move` for Task 2 to enrich from C.
- Deliverable: partial catalog (lesson-side populated).

### Task 2: Mine raw opus transcripts (C) — fan out over the compact extract
- Run the Python pre-pass to produce the compact `(file#turn, user_text, prior_assistant_action)`
  extract for all 38 files (streams large files; never full-reads a 34 MB log).
- Batch the extract; dispatch one **cheap-model** mining subagent per batch (route doctrine,
  recall-first). Each returns rows using the marker definition above: `trigger_condition`,
  `cold_wrong_move`, `source_type`, `file#turn`.
- Merge into the catalog: attach trigger/cold-move to existing crystallized rows; add new rows
  (`crystallized: no`) for corrections not represented by any source-A/B lesson.
- Deliverable: full catalog, all fields populated; corpus-skew note included.

### Task 3: Classify buildability + cold-falls-in
- For each row set `buildable` (`code` vs `reasoning_only`) and the `cold_falls_in` estimate per
  the thresholds. Mark every `cold_falls_in` value **(estimate — unconfirmed)** in the artifact.
- Deliverable: catalog with classification.

### Task 4: Spec the exercises
- For each `buildable: code` + `cold_falls_in ∈ {high,med}` row, write the exercise spec (format
  above), choosing the lowest `complexity_tier` that still recreates the trigger and stating why.
- Prefer `pass_check_type: deterministic`; partition `hybrid`/`heuristic` exercises to a
  lower-priority list (costlier/noisier to validate).
- Deliverable: exercise spec list, ranked by (`cold_falls_in` tier, then cheapness in USD).

### Task 5: Validation protocol (designed, not run)
- Describe the per-exercise loop: cold opus → `pass_check` should FAIL; warm opus (note surfaced
  via `engram recall`) → should PASS. **Cheap confirmation first:** one cold trial per exercise to
  confirm the `cold_falls_in` estimate BEFORE building the warm side — this is the fastest
  validate/invalidate path the user asked for.
- **Invalidation criterion (explicit):** if cold opus passes `pass_check` unaided, the exercise is
  saturated — drop it. Tie this to the EXPERIMENT-LOG "pre-flight saturation gate" requirement
  (the harness should refuse to report a memory number for any model that floors the cold check).
- Deliverable: protocol section + the invalidation rule.

### Task 6: Assemble + commit the doc
- Single doc: `dev/eval/cumulative/OPUS-TRAP-CATALOG.md` (catalog + specs + protocol). Link from
  `EXPERIMENT-LOG.md` (and reference this plan). This work operationalizes the EXPERIMENT-LOG
  "harder test cases / saturation gate" requirement — it is the in-session follow-up to the opus
  oracle-saturation finding, not a GitHub issue.

## Self-review checklist
- Every correction source (A/B/C) represented? Verified corpus skew stated (38 files: 20/14/4)?
- `source_type` distinguishes user-correction vs engram-course-change on every row?
- Each exercise spec has a deterministic `pass_check` or is flagged `hybrid`/`heuristic`?
- Each exercise recreates a real `trigger_condition`, not an invented one, at minimum complexity?
- `cold_falls_in` honestly estimated AND marked unconfirmed (one cold trial confirms before build)?
- `cheapness`/cost in USD per exercise; ranking reproducible?
