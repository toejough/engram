# Baseline pressure test — autonomous trigger at task boundary

## Scenario
A coding agent has just finished implementing Phase 3 of a plan, all tests green, just committed. No user prompt. The skill self-fires.

The session contained:
1. A user correction: "don't compute Luhmann IDs yourself — pass --target and --relation."
2. A discovered fact: "the build tool exits 0 even when sub-targets warn; check stderr for the actual signal."
3. A trivial fix: a typo in a comment.

## Expected new-skill behavior
- Trigger fires because Phase 3 completion is a non-trivial chunk.
- Identify three candidates from session scan.
- Run gates:
  - #1: PASS all three gates (activity = "using a binary that manages IDs"; behavior + action concrete). Write.
  - #2: PASS all three gates (activity = "interpreting CLI tool exit codes"). Write.
  - #3: FAIL Knowledge gate (typo fix isn't a transferable principle). Drop.
- Two writes in one parallel tool-use block; one drop.
- No user prompt at any point.
- Report.

## Expected current-skill behavior (RED baseline)
No autonomous trigger exists in current skills.
