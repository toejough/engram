## Why

`route`'s cold-start priors table starts every work-kind at the cheapest tier absent recalled
evidence. Recorded dispatch evidence for `work-kind/go-package-implementation` now clears the
skill's own escalation bar — cheap-tier pass rate is 8/12 (67%, below the 70% floor) while
mid-tier pass rate is 10/10 (100%, above the 85% ceiling), with both tallies well past the
5-dispatch evidence floor. Leaving the cold-start default at cheap means every new session pays
the cost of a doomed cheap-tier attempt and a spec-retry before recall (or a failed dispatch)
teaches it what dispatch evidence already knows. Filed as toejough/engram#722.

## What Changes

- Add one row to `agent-instructions/skills/route/SKILL.md`'s Cold-start priors table:
  `work-kind/go-package-implementation` starts at `mid`, sourced from the 8/12 cheap vs 10/10 mid
  evidence tally (2026-08-09).
- Land the edit via `superpowers:writing-skills` TDD (RED: baseline test showing the cold-start
  default under-routes this work-kind to cheap absent recall; GREEN: add the priors row;
  REFACTOR: tidy if needed).

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
- `route-evidence-rubric`: the cold-start tier for `work-kind/go-package-implementation` changes
  from the default cheapest tier to `mid`, evidence-backed rather than the unproven default.

## Impact

- `agent-instructions/skills/route/SKILL.md` — Cold-start priors table gains one row.
- No production Go code, no CLI surface, no runtime behavior change — this is a skill-text-only
  edit governed by the route skill's own recall-first dispatch loop.
