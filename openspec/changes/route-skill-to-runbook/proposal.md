## Why

engram's shim-and-runbook architecture is meant to eventually replace every agent-instruction skill
(vault note 997). The archived `runbook-shim-follow-frame` change (2026-09-17) converted `recall`,
`learn`, `write-memory`, and `route` from `SKILL.md` bodies into runbook notes — but **only as eval
fixtures**, for its shim-only comparison harness (confirmed live: `agent-instructions/skills/{recall,
learn,write-memory,route}/SKILL.md` all still ship as skills; no production vault note exists for any
of the four). Promoting recall/learn/write-memory is tracked separately (#760) and is out of scope
here. `route`'s promotion is tracked by #757: a fully validated runbook body (`Route-R`, luhmann "1")
sits unpromoted in the eval's fixture tree, and route is the only one of the four whose D8 validation
run surfaced a confirmed, still-open defect (below) — so its promotion also needs that defect
addressed, not just a copy-over.

The promotion alone is not sufficient to close #757: the eval's own D8 acceptance bar (parity with
route's skill-form behavior) was never met for route across 6 rounds ($27.04 spent) — every round
converged on the same single remaining defect, detailed below.

## What Changes

- Promote the validated `Route-R` fixture body
  (`dev/eval/cumulative/runbook_vs_skill/phase2/encodings/taskRoute/Route-R/vault/1.2026-09-12.route-dispatch-tier-selection.md`)
  into the production vault as route's runbook, replacing `agent-instructions/skills/route/SKILL.md`
  as route's carried instructions. **BREAKING** for anything that currently invokes route as a
  Claude Code skill (`Skill route`) — after this change, route's guidance is delivered via the
  shim's runbook-matching path (`engram query` → matched `kind: runbook`), not a slash-invokable
  skill. `agent-instructions/skills/route/SKILL.md` is retired (removed or replaced with a stub
  pointing at the runbook) once the promotion is verified.
- Add exactly one new `red_flags` entry to the promoted runbook, naming the specific condition "about
  to write an aggregate-evidence field as plain prose instead of the `[[note-basename]]` wikilink
  bracket syntax the template above already shows" — the one lever not yet tried across the archived
  eval's 6 rounds (every round explicitly declined to re-word the aggregate-write templates, since
  three independent character-by-character re-reads confirmed the `[[wikilink]]` syntax already sits
  unambiguous and directly above the field being filled in). No other wording in the promoted body
  changes.
- Re-run the eval's route-only probe
  (`dev/eval/cumulative/runbook_vs_skill/phase2/probe_phase2.py --task route --model sonnet5 --n 3
  --arms R --shim-only --keep`, or the current equivalent invocation) against the `red_flags`-amended
  runbook to determine whether `followed_all`/`end_state` finally clear the D8 bar (within one trial
  of the skill row's 2/3). Record the outcome either way — this change does not assume the fix works.
- Retire `agent-instructions/skills/route/SKILL.md` only once the promoted runbook is confirmed live
  and retrievable (matches `guidance-runbook-follow-frame`'s existing matched-runbook contract).

## Capabilities

### New Capabilities

(none — this change applies the existing runbook/follow-frame capability to a new carrier; it does
not introduce new system behavior)

### Modified Capabilities

- `route-dispatch-evidence`: Requirement "Every route dispatch SHALL be recorded as a tagged fact
  note" normatively requires "the skill SHALL state [the write-memory-vs-direct-amend split]
  explicitly" — after promotion, route's instructions are carried by a runbook, not a skill; the
  requirement's carrier language updates to match (behavior itself is unchanged).
- `route-evidence-rubric`: the "Red flag on paid dispatch with a free option available" scenario
  normatively requires the condition be "a documented red flag in the skill's red-flags table" —
  the promoted runbook carries this as a structured `red_flags:` frontmatter entry, not a markdown
  table in a skill body; the requirement's carrier language updates to match (behavior itself is
  unchanged).

`guidance-runbook-follow-frame` is unchanged: its `red_flags` field and matching behavior already
support what this change needs; adding one new entry to one runbook's content is not a requirements
change to that spec.

## Impact

- **Removed/replaced**: `agent-instructions/skills/route/SKILL.md` (retired after promotion is
  verified).
- **Added**: route's promoted runbook note in the production vault (from `Route-R`, luhmann "1"),
  carrying an additional `red_flags` entry.
- **Deployed guidance**: any deployment path that currently packages `agent-instructions/skills/route/`
  (`engram update --with-guidance` and equivalent) stops shipping the route skill directory once
  retired.
- **Eval assets**: `dev/eval/cumulative/runbook_vs_skill/phase2/encodings/taskRoute/Route-R/` remains
  as the historical fixture; not deleted by this change.
- **GitHub issue #757**: this proposal is the change that closes it. Sibling issues #758 (please),
  #759 (curate), #760 (recall/learn/write-memory production promotion), and #761 (openspec skills →
  runbooks) are explicitly out of scope here — each is its own change.
