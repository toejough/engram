## Why

`route`, `please`, `curate`, and `write-memory` are retired skills, live as runbooks. `learn` is mid-implementation (`learn-skill-to-runbook`). `recall` is the last skill in `agent-instructions/skills/` (GitHub #760's final slice) — once it converts, zero `SKILL.md` files remain.

`recall` is also the largest and structurally most unusual of the five: 344 lines, 27,327 bytes of body, a single unified 9-step procedure (19,262 bytes) annotated inline with `[glance: …]` notes marking where the cheap read-only rung differs from the full 10-phrase write-capable rung, and a 28-row red-flags table (4,206 bytes — the largest of any skill converted so far, more than please's 22 rows). Joe already decided (prior `/opsx:explore` session on #760) how to encode the glance/deep distinction: **two separate runbook notes**, `recall-glance` and `recall-deep`, not a single note with escalation logic in the body and not a new schema depth/tier field.

## What Changes

- Convert `agent-instructions/skills/recall/SKILL.md` into a runbook set with two triggered entry points, `recall-glance` and `recall-deep`, per Joe's decision. Because the skill's own procedure is written as ONE unified sequence (Steps 0–4) with inline `[glance: …]` annotations rather than two independent procedures, the entry points share underlying step content via wikilinked sub-runbooks rather than each duplicating the full ~19KB procedure text verbatim (see design.md D1 for the exact split, decided against measured byte counts, not assumed).
- `recall-glance`: the cheap, read-only-with-respect-to-vault-knowledge rung (Steps 0–3.5, ~3 phrases, keeps Step 2.7 activation, skips the write side). Its body states the C5 escalation rule verbatim (glance honors a recent-channel standard 0/5 vs deep's 4/5, #661) and wikilinks `recall-deep` for that case.
- `recall-deep`: the full rung (all 10 phrases, Steps 2.5C and Step 4 — coverage amend/learn and synthesis-persist). Its body explains when glance should have escalated to it, per Joe's decision.
- Both entry points wikilink write-memory's already-promoted real production runbook (`1053.2026-09-22.write-memory-compose-execute-verify`) directly at every write site (Step 2.5C, Step 4) — no fixture-placeholder detour, same approach `learn`'s conversion is taking, since write-memory is already live.
- Populate `red_flags` from the skill's 28-row table, prioritized under the 1200-byte cap per note (design D3) — by far the largest red-flags source converted so far, will spread across multiple notes.
- Build a `recall` eval task and skill-arm baseline from scratch in `dev/eval/cumulative/runbook_vs_skill/phase2/` (none exists; the eval-fixture runbook notes at `dev/eval/cumulative/runbook_vs_skill/phase2/encodings/shim/recall-learn/vault/` — `recall-glance-vs-deep-mode`, `recall-note-synthesis-and-coverage`, `recall-activation-and-closing-synthesis`, `recall-from-unified-memory` — are starting material, spot-checked faithful to the real skill in an earlier session pass, not a green field).
- Decide `triggers:` (or their absence) during design: unlike `please`/`curate`/`learn`, every session's first action already runs an implicit recall-shaped query (the shim's mandatory bootstrap query, per `guidance-runbook-follow-frame`), so `recall-glance`/`recall-deep` may surface primarily by similarity on that query rather than needing explicit lexical triggers — evaluated, not assumed (design D5).
- Retire `agent-instructions/skills/recall/SKILL.md` in this same change, gated on the same three-part bar every prior conversion used, applying the write-memory conversion's hash-check discipline before any paid re-run following a wording fix (that conversion cost $2.24 extra when this check was skipped once).
- Update every live reference to `recall` as a skill.

## Deferred (not built in this change)

- **Retiring `agent-instructions/guidance/recall.md`** and resolving its redundancy with the shim's four re-entry moments (flagged in #760's own comment thread; `guidance/recall.md`'s cues are recall-specific — `/recall glance` at the same four moments the shim's generic re-entry list already states, escalating to `/recall deep`). Once this change lands, `recall` itself is a real runbook, so retiring `guidance/recall.md` becomes a clean, immediately-buildable fast-follow — named explicitly here as the natural next issue, not bundled into this conversion.
- The same reasoning applies transitively to `guidance/delegate.md` and `guidance/learn.md`'s eventual retirement (both wait on their respective skills becoming runbooks; `learn`'s is mid-implementation in a sibling change).

## Capabilities

### New Capabilities

(none — applies the existing runbook/follow-frame/lexical-triggers capabilities to a new carrier)

### Modified Capabilities

- `recall-glance-deep-dial`: requirements naming "the recall skill"/"glance mode"/"deep mode" as skill-argument selection update the carrier to two runbooks (`recall-glance`, `recall-deep`) selected by which is discovered/triggered, not a CLI mode argument. Behavior (the C5 escalation rule, what glance keeps vs skips) is unchanged.
- `recall-two-channel-payload`, `recall-centroid-sampling`, `recall-matched-note-floor`, `recall-query-timings`: carrier-language updates wherever they name "the recall skill" normatively (verify each during design; behavior unchanged).
- `write-memory-worker`: the "parent (`recall`, still a skill; `learn`, a runbook)" language from the `learn` change's own delta updates again — `recall` becomes a runbook too, so both parents are now runbooks.
- `guidance-runbook-follow-frame`: no new requirement expected (recall's runbooks are query-matched/triggered, the existing case); confirm during design.

## Impact

- **Removed**: `agent-instructions/skills/recall/SKILL.md` (and its directory), after gates pass. This empties `agent-instructions/skills/` entirely — zero SKILL.md files remain once `learn`'s sibling change also lands.
- **Added**: `recall-glance`, `recall-deep`, and their shared sub-runbook(s) in the production vault; a `recall` eval task, skill-arm baseline, and conversion-fidelity report under `dev/eval/cumulative/runbook_vs_skill/phase2/`.
- **Docs**: `CLAUDE.md`, `README.md`, `docs/GLOSSARY.md`, `docs/architecture/{c1-system-context,c2-containers,c3-components}.md` wherever they list `recall` as a skill.
- **Deployment**: `engram update` stops shipping the `recall` skill directory and removes deployed copies in both harnesses.
- **Eval spend**: new fixture/baseline plus a shim-only probe run per entry point (glance and deep may need separate validation); estimate and confirm cost before running.
- **Issues**: progresses #760 (closes the final slice once `learn`'s sibling change also lands — this change alone completes `recall`'s conversion but #760 isn't fully closed until both are done).
