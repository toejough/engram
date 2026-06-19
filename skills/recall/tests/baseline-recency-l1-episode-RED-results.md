# RED results — recency tiebreaker on conflicting notes (current skill, pre-fix)

Run against the current SKILL.md with an agent that misapplies the recency rule by reading the
`created` dates correctly but failing to apply the **near** judgment on the candidate L2.

## Predicted RED failure mode

The current SKILL.md Step 2.5 C defines the coverage criteria clearly, but an agent applying
the rule without care for the recency-weighted view will:

1. See the candidate L2 covers "retry up to 3×" (Member A's stance).
2. Note that the cluster includes Member A (same stance) and Member B (conflicting stance).
3. FAIL to apply the recency weight first — treating both members as equally valid.
4. Mark the candidate **covered** (one member agrees) rather than **near** (the recency-weighted
   binding stance — Member B — is absent from the candidate).
5. Issue `engram amend --activate` (link-enrich only) rather than re-synthesizing content.

A secondary failure: if the agent does pick B correctly but still marks the candidate "covered",
that is also a fail — the candidate must be updated to reflect B's stance.

## What the RED run establishes

This is the failing baseline that authorizes the test scenario to exist. The scenario is
designed so that an agent which:
- Picks Member A (wrong recency pick) → fails the tiebreaker directly.
- Picks Member B but marks the candidate "covered" → fails the coverage judgment.

Either failure means the L2 is left stale with the superseded stance ("retry up to 3×") baked in.

## Capture

*To be filled in when a real RED run is performed against a skill version that produces this failure.*
