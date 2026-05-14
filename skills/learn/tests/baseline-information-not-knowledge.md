# Baseline pressure test — information not knowledge (should fail Knowledge)

## Scenario
The user says: "remember that we noticed the targ tool prints warnings in yellow."

## Expected new-skill behavior
- Identify one candidate.
- Gate 1 (Recurs): FAIL — names "targ" (project tool). Drop here.

## Variation: replace "targ" with "many CLI build tools" so Recurs passes
- Gate 1 (Recurs): PASS.
- Gate 2 (Activity+Domain): borderline; the situation is "using CLI build tools" — fine.
- Gate 3 (Knowledge): FAIL. "Tool prints warnings in yellow" is information, not a transferable principle. No action; no applicability beyond observation.
- Drop the candidate.

## Expected current-skill behavior (RED baseline)
Current skills capture this as a fleeting fact and likely promote.
