# RED results — bootstrap create (agent-judged model, empty candidate_l2s)

Run against a skill version that interprets `candidate_l2s: []` as "no candidates → nothing to do"
rather than "no candidates → absent outcome → CREATE."

## Predicted RED failure

An agent following a misread of Step 2.5 A ("Run `engram show` on every entry in `candidate_l2s`")
may treat an empty list as a completed A step with no candidates to show — and skip directly past
the coverage judgment, leaving clusters 0 and 1 unwritten.

| Cluster | candidate_l2s | Expected | Misread agent | Result |
|---------|--------------|----------|---------------|--------|
| 0 | empty | CREATE | Skip — "no candidates to read" | FAIL |
| 1 | empty | CREATE | Skip — "no candidates to read" | FAIL |
| 2 | 0.97 (covered) | amend --activate | Depends on whether agent reads payload content | — |
| 3 | 0.85, 0.71, 0.58 | amend top candidate (covered) | Depends on show calls | — |

**Verbatim the agent might produce:**
> "Step 2.5 A: run `engram show` on every entry in `candidate_l2s`. Cluster 0: `candidate_l2s`
> is empty — no candidates to read. No coverage judgment possible. Cluster 1: same. Proceeding
> to clusters 2 and 3."

## What the RED run establishes

This is the failing baseline that authorizes the scenario. An empty `candidate_l2s` must trigger
the **absent** branch (no candidate addresses the situation → CREATE), not a skip. The RED outcome
is that a fresh vault never grows L2s because every bootstrap cluster has empty candidates and
each is silently skipped.

## Capture

*To be filled in when a real RED run is performed against a skill version that produces this failure.*
