# Baseline pressure test — autonomous learn: only crystallize explicit moments

## Scenario

A coding agent has just finished implementing Phase 3 of a plan, all tests green, just committed. No user prompt. The learn skill self-fires.

The session contained:
1. A user correction: "don't compute Luhmann IDs yourself — pass --target and --relation."
2. A discovered fact (not requested by user): "the build tool exits 0 even when sub-targets warn; check stderr for the actual signal."
3. A trivial fix: a typo in a comment.

## Expected current-skill behavior (PASS)

- **Step 1 — sweep:** `engram ingest --auto` runs first.
- **Step 2 — scan for explicit moments only:**
  - Item 1 (user correction) → WRITE as `engram learn feedback`. This is an explicit correction.
  - Item 2 (self-discovered fact) → DO NOT WRITE. The current skill crystallizes ONLY corrections and explicit save-requests. Self-discovered facts are not in scope — raw chunks already capture this via Step 1's ingest. Writing it would violate the "no moments of either kind → write nothing" rule.
  - Item 3 (typo fix) → DO NOT WRITE. Not a correction of the agent's behavior; one-time event.
- **Result:** 1 write (the correction), 0 for the self-discovered fact, 0 for the typo.
- No `engram transcript` calls. No `engram learn episode` calls. No session summarizing.
- Report: "1 explicit correction crystallized. Self-discovered facts and routine work left to chunk index (Step 1)."

## Failure modes that must FAIL this test

- Writing the self-discovered fact (item 2) as `engram learn fact` — that is out of scope for the current skill.
- Writing any episode/arc notes.
- Running `engram transcript` anything.
- Skipping item 1 (the genuine correction).
- Applying Gate/Recurs/Activity-Domain/Knowledge three-gate logic explicitly (those concepts are removed from the current skill).

## What changed from the pre-v2 expected behavior

The pre-v2 fixture expected the agent to PASS all three gates and write BOTH the correction AND the self-discovered fact. The current skill does NOT write self-discovered facts. An agent that writes item 2 should FAIL this test, not PASS.
