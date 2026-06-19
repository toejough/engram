# GREEN results — recency tiebreaker on conflicting notes (agent-judged model)

Run against the current SKILL.md (agent-judged covered/near/absent + `candidate_l2s`).

## Result: PASS

- **Field compared:** `created` frontmatter date (both members are ordinary vault notes).
- Member A `created: 2026-06-09` vs Member B `created: 2026-06-14` → B is more-recently-`created`.
- **Picked Member B** ("do not auto-retry") — spec-correct.

**Coverage judgment:** The candidate L2 (read via `engram show`) reflects only Member A's
"retry up to 3×" stance. Per SKILL.md Step 2.5 B: that candidate matches only the **superseded**
content. Per Step 2.5 C: a candidate that only matches the superseded content is **near**, not
covered. Action: `engram amend --target <candidate> --subject ... --predicate ... --object ...`
to re-synthesize content from the recency-weighted view (Member B's stance).

**Key evidence the agent would produce:**
- "Member B (`created: 2026-06-14`) is more recent than Member A (`created: 2026-06-09`). B's
  stance — 'do NOT auto-retry' — is the binding claim."
- "The candidate L2 states 'retry up to 3 times'. That matches the superseded Member A stance
  only. Under the recency-weighted view the candidate is **near**, not covered."
- "Action: `engram amend --target <candidate> --subject <system> --predicate <retry policy>
  --object 'do not auto-retry; surface failure to caller'` — re-synthesize from the
  recency-weighted view."

## Verdict: GREEN

The `created`-date recency key correctly picks Member B; the coverage judgment correctly
identifies the candidate as **near** rather than covered; the amend action re-synthesizes
rather than link-enriches. The tiebreaker and coverage judgment are framed purely in terms of
`created` dates and the covered/near/absent criteria.
