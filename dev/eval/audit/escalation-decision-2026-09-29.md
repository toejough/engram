# Escalation decision (learn-rate-skill-only, task 3.4)

Task 3.4 of `learn-rate-skill-only` asks for the design D-F escalation decision after the pre-registered re-measure. Its done-when is a joint review with Joe. Joe gave the decision on 2026-09-29; it was recorded, and 3.4 ticked, on 2026-09-30, when Joe asked for `learn-rate-skill-only` to be archived.

## Joe's instruction (2026-09-29)

> "update the guidance to lean into the agents' tendency to [learn] midstream … after sub agents return, in particular."

## Inputs

| Report | Metric | Unit | Value |
|---|---|---|---|
| `REPORT-re-measure-2026-09-28.md` (task 3.3) | W2: learn fired, of worth-learning moments | moments | 3/66 = 4.5% [0.0, 10.6]% vs baseline 4/71 = 5.6% [1.4, 11.3]% — not distinguishable from baseline |
| same, caveat 3a | the instrument can't see v1's parent-side capture path | — | W2 undercounts v1 capture by construction |
| `REPORT-parent-capture-2026-09-29.md` | C0: a closing `learn` after the session's last LESSONS receipt | sessions | 0/2 |
| same | C1: captured or covered, of worth-capturing items | items | 93/114 = 81.6% [74.6, 87.7]% (upper bound) |
| same, post hoc | C1 without route-dispatch/route-evidence note matches | items | 80/114 = 70.2% [62.3, 78.1]% |

Task 3.3's verdict: W2 did not rise detectably. The parent-side check shows capture does happen, but midstream (parents writing notes while work runs), not at the closing learn v1 routes it to.

## Decision

- **Escalate:** none of D-F's three layers (watcher/metacognition layer, mechanical LESSONS validator, hooks/harness push).
- **Park further:** no. The layers stay parked (not parked further), with D-F's revisit condition unchanged; the next lever is this change.
- **Next lever:** the md-only change `learn-at-subagent-return` (`openspec/changes/learn-at-subagent-return/`), which moves capture to the subagent-return moment and keeps the closing learn as a backstop.
