## Why

`route`, `please`, `curate`, and `write-memory` are now retired skills, live in production as runbooks. `learn` and `recall` are the last two (GitHub #760). `learn` is structurally the easier of the two: it is one procedure with branching (unlike `recall`'s glance/deep two-mode split, already decided as two separate future runbooks but not built here), and it already hands off its writes to write-memory's promoted runbook (`1053.2026-09-22.write-memory-compose-execute-verify`) at six call sites repointed in the prior change. Converting `learn` next keeps momentum on #760 and, unlike `write-memory`, tests the please/curate pattern (a directly user/shim-triggered procedure) rather than the worker pattern.

No `learn` eval exists yet — please, curate, and write-memory each needed one built from scratch, and `learn` is no exception.

## What Changes

- Convert `agent-instructions/skills/learn/SKILL.md` (307 lines, 21,233 bytes) into a runbook note set. Measured section sizes: Step 1 Sweep (1,250 B), Step 1.5 Vocab liveness (2,067 B), Step 2 Crystallize (10,167 B — the four lesson kinds: corrections, save-requests, reversals, confirmed approaches), Step 2.5 Ad-hoc QA capture (2,014 B, already carrying this session's self-referential duplicate-guard fix — carried forward verbatim, not regressed), Batch mode `--reparent-luhmann` (2,283 B, a deliberately-separate standalone mode per the skill's own text), Red flags (10 rows, 1,372 B — over the 1200-byte cap on its own, needs prioritization like `please`'s 22-row table did).
- Given Step 2 alone (10,167 B) plus a prioritized red_flags set approaches or exceeds a single query item's practical size, split into a top runbook (situation, the four-kind scan, Luhmann placement test, `done_when`, prioritized `red_flags`) plus wikilinked sub-runbooks for Step 1/1.5 (the sweep-and-vocab entry sequence), Step 2.5 (QA capture), and the batch mode (a clearly-marked alternate-entry procedure, not part of the normal Step 1→2.5 sequence). Exact split boundaries are a design decision (see design.md D1), following `please`'s three-sub-runbook precedent.
- The runbook body wikilinks write-memory's real production basename directly (`[[1053.2026-09-22.write-memory-compose-execute-verify]]`) at every write site — no fixture-placeholder intermediate step, since write-memory is already promoted (unlike write-memory's own conversion, which had to invent and then repoint a temporary basename).
- Populate `red_flags` from the skill's 10-row table (episode-workflow reconstruction, over-capture of routine successes, skipping the sweep, `--tier`/L3 writing, record-correction-isn't-capture, the QA self-referential duplicate guard, plus others) — prioritized under the 1200-byte cap; overflow moves to sub-runbook `red_flags`/body, mirroring `please`'s D3.
- Set `triggers:` from `learn`'s actual firing language and `guidance/learn.md`'s cues: `/learn`, "remember this", "note for next time", explicit save-request phrasing — whole-word matched (shipped this session), so common words in this space (e.g. "remember") need the same over-fire scrutiny `please`/`curate` went through before being accepted or excluded.
- Build a `learn` eval task and skill-arm baseline in `dev/eval/cumulative/runbook_vs_skill/phase2/`, mirroring please/curate: explicit-ask baseline first (paid, cost confirmed before running), then the shim-only arm, D8 bar = within one trial of the skill row, with a validity gate proving `learn`'s actual steps ran (placement decided correctly, the right lesson kind chosen, write-memory's runbook actually fetched and followed) rather than a plausible-looking outcome produced by guessing.
- Retire `agent-instructions/skills/learn/SKILL.md` in this same change, gated on the same three-part bar every prior conversion used: D8 met, retrieval verified against the real vault, `shim.md` confirmed imported (already true).
- Update every live reference to `learn` as a skill.

## Deferred (not built in this change)

- **`recall`-to-runbook** — its own future change, gated on the glance/deep design already decided (two separate runbook notes, `recall-glance` and `recall-deep`) but not built. `recall` (344 lines, 28-row red-flags table) is larger than `learn` and will need more sub-runbooks.
- **Retiring `agent-instructions/guidance/learn.md`** and resolving its redundancy with `shim.md`'s four re-entry moments (flagged in #760's own comment thread) — deliberately out of scope here; only sound once `learn` itself is a real production runbook, and even then is its own fast-follow change, not bundled into this conversion.
- **The `shim.md`/`recall.md` firing-cue redundancy** — same reasoning, waits on `recall`'s own conversion.

## Capabilities

### New Capabilities

(none — applies the existing runbook/follow-frame/lexical-triggers capabilities to a new carrier)

### Modified Capabilities

- `learn-runbook-capture`: requirements naming "the learn skill" as the carrier update to name the learn runbook (and its sub-runbooks where the requirement maps to a specific step). Behavior (four-kind capture, Luhmann placement, write-memory handoff) is unchanged.
- `learn-adhoc-qa-capture`: the just-shipped Step 2.5 duplicate-guard spec's carrier language updates from "learn's Step 2.5" (a skill section) to the QA-capture sub-runbook. Behavior (the self-referential duplicate guard) is unchanged.
- `learn-branching-disposition`: the Luhmann-placement-test requirement's carrier updates the same way.
- `guidance-runbook-follow-frame`: no new requirement expected (the "skill names a runbook by basename" requirement already shipped for write-memory covers any remaining skill→runbook references; `learn` itself becoming a runbook is a query-matched/triggered runbook, the existing case). Confirm during design whether anything new is needed.
- `write-memory-worker`: no change expected (its "parent skill" language is carrier-agnostic; confirm during design).

## Impact

- **Removed**: `agent-instructions/skills/learn/SKILL.md` (and its directory), after gates pass.
- **Added**: `learn`'s promoted runbook (+ sub-runbooks) in the production vault; a `learn` eval task, skill-arm baseline, and conversion-fidelity report under `dev/eval/cumulative/runbook_vs_skill/phase2/`.
- **Docs**: `CLAUDE.md`, `README.md`, `docs/GLOSSARY.md`, `docs/architecture/{c1-system-context,c2-containers,c3-components}.md` wherever they list `learn` as a skill (skill count drops to one: `recall`).
- **Deployment**: `engram update` stops shipping the `learn` skill directory and removes deployed copies in both Claude Code and Pi harnesses.
- **Eval spend**: new fixture/baseline plus a shim-only probe run; estimate and confirm cost before running (please/curate/write-memory each ran roughly $1–5 per stage, write-memory's total across three attempts $4.73 due to a fixture-vs-live-file mixup — this change's tasks should explicitly verify any wording fix reaches the live skill file, not just a frozen fixture copy, before spending on a re-run).
- **Issues**: progresses #760 (closes the `learn` slice; `recall` remains open).
