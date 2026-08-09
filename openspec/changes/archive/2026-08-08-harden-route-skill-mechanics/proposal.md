## Why

Three follow-up findings against `agent-instructions/skills/route/SKILL.md` are each fully
specified (concrete fix, no open design question) but individually too small to justify their
own `writing-skills` TDD pass: a real cost incident where the roster table's named-model wording
caused 5+ paid dispatches while a free local model sat idle (#702); a silent-duplicate risk in the
aggregate-evidence lookup that the drowning gauge (ADR-0019) pre-registered but never hardened
against (#679); and three minor copyedits from #674's final review (#680). Bundling them into one
change pays the TDD ceremony cost (baseline RED, edit, GREEN, pressure test) once instead of
three times, since all three land in the same file with no interaction between the edits.

## What Changes

- **Roster wording (#702):** Reword the cold-start-priors roster line so "cheap tier" resolves
  against the current environment rather than reading as a named-model prescription: cheapest
  available agent in THIS environment, free local models preferred over paid API models. Add a
  red-flag row for dispatching cheap work to a paid model while a free local model was available.
- **Aggregate uniqueness + merge (#679):** Add a documented, executable uniqueness check to the
  aggregate-update procedure (a vault basename listing/glob, since aggregates are untagged and
  `engram count` cannot group by slug), run at the same trigger moments as the existing count
  audit. Add a documented merge procedure for the duplicate case: union evidence wikilinks,
  recompute the tally via the existing count commands, record the incident as ADR-0019
  drowning-remedy evidence.
- **Copyedits (#680):**
  1. State explicitly, in the aggregate-update branch, that the evidence note goes through
     write-memory while the aggregate amend-or-create is composed directly by the
     route-executing agent (structurally forced — write-memory has no amend form) — currently
     unstated, reads as an undocumented exception to the write-site doctrine.
  2. Trim the "kind=fact" 4x-in-5-sentences repetition in the evidence-note handoff spec.
  3. Add a half-line warning that `--group-by tier` groups the wrong thing (the pre-existing
     L1/L2/L3 note-tier frontmatter attribute, not the `tier/<cheap|mid|deep>` tag family) — a
     namespace-collision footgun in the count-audit section.
  - Out of scope: the issue's optional 4th item (reconciling `[[work-kind-definition]]`-style
    wikilinks with the file's "vault note N" citation style) is explicitly gated on #678; not
    touched here.

All edits ride `superpowers:writing-skills` TDD as one combined pass (baseline behavior test RED,
edit, GREEN, pressure test) since the three edits are independent and non-overlapping within the
same file. Trap gate smoke stays GREEN before and after.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `route-dispatch-evidence`: "Each work-kind SHALL have one aggregate fact note" gains a
  uniqueness-check-and-merge requirement (#679); "Every route dispatch SHALL be recorded as a
  tagged fact note" gains the write-site-attribution statement (#680.1); "Count SHALL audit
  aggregate drift using tags" gains the `--group-by tier` namespace-collision warning (#680.3).
- `route-evidence-rubric`: gains a new requirement that the cheap tier resolves to the cheapest
  available agent in the current environment, preferring free local models over paid API models
  (#702).

## Impact

- `agent-instructions/skills/route/SKILL.md`: cold-start-priors roster line + red-flags table
  (#702), aggregate-update procedure + count-audit section (#679, #680.3), evidence-note handoff
  spec (#680.1, #680.2).
- `agent-instructions/skills/route/tests/`: baseline behavior test(s) per `writing-skills`
  convention; trap gate smoke re-run before/after.
- No code changes — this is a skill-instructions-only change (no `internal/` or `cmd/` impact).
