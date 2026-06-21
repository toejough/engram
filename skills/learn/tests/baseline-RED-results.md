# Baseline RED — pre-recall-v2 learn skill behavior

This documents the behavior of the OLD learn skill (pre-recall-v2), establishing the RED baseline the current skill replaced.

## Summary of pre-v2 behavior (stale — for historical reference)

The old skill had four separate skills (capturing-fleeting-notes, promoting-to-permanent-notes, /learn, /remember), three-gate logic (Recurs / Activity-and-Domain / Knowledge), interactive approval steps, and episode/arc writes.

| Scenario              | Pre-v2 behavior                                                                     | Current skill expects         |
| --------------------- | ----------------------------------------------------------------------------------- | ----------------------------- |
| project-specific      | Gate 1 FAIL (Recurs) → drop — no write                                              | Write (explicit save-request; generalize situation) |
| hindsight-framing     | Gate 2 FAIL (Activity+Domain) → reframe loop → write after reframe                  | Write directly; only judgment is `--situation` phrasing |
| info-not-knowledge    | Gate 1 FAIL or Gate 3 FAIL → drop — no write                                       | Write (explicit save-request) or state refusal |
| clean-write           | Three gates PASS → Luhmann-continuation write under `10c1 --position continuation` | Sweep then write with `--position top` |
| autonomous-trigger    | Self-discovered fact (item 2) written; typo dropped; correction written             | ONLY the correction written; self-discovered fact NOT written |
| l1-episode            | Write L1 episodes per arc via `engram learn episode --from-transcript-range`        | RETIRED — episode writes are gone |
| lazy-l1-only          | Write episodes only, defer L2 to recall                                             | RETIRED — episode model is gone |
| recall-mirror         | Gate-driven framing, not recall-mirror framing                                      | RETIRED — recall-mirror test concept replaced by simpler model |

## Current expected behavior (GREEN baseline)

The current `skills/learn/SKILL.md` does exactly two things:

1. **Sweep** (`engram ingest --auto`) — always, first.
2. **Crystallize explicit lessons only** — corrections and explicit save-requests, nothing else.
   - `engram learn feedback` for corrections.
   - `engram learn fact` for explicit save-requests.
   - No episodes, no arcs, no `engram transcript`, no tier/ADR flags.
   - No three-gate logic.
   - No Luhmann-continuation or `--target` on new notes.

A session with no corrections and no save-requests produces exactly one output: the `engram ingest --auto` tally line.
