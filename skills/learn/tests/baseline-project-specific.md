# Baseline pressure test — project-specific candidate (should fail Recurs)

## Scenario
The user says: "remember that the engram promote binary required us to extract writePromoteUnderLock when the cyclomatic complexity check fired on Task 8."

## Expected new-skill behavior
- Identify one candidate.
- Gate 1 (Recurs): FAIL. Situation names "engram promote", "writePromoteUnderLock", "Task 8" — project-specific.
- Drop the candidate. No `engram promote` call.
- Report names the gate failure with a one-line reason.

## Expected current-skill behavior (RED baseline)
The current `capturing-fleeting-notes` would write a fleeting; current `promoting-to-permanent-notes` would convert it to a permanent. Either path writes; neither rejects.
