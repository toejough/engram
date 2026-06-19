# Baseline test — recency tiebreaker on conflicting notes (created-date ordering)

**What this tests:** the recency-bias tiebreaker (Step 2.5 B) when two cluster members are
regular vault notes that **conflict on a convention** and are distinguished only by their
`created` date.

**Spec rule under test** (SKILL.md Step 2.5 B):
- When evidence conflicts (newer member explicitly negates or reverses an older claim), **recent wins**.
- "Newer" for notes (facts/feedback) = the frontmatter `created` date.

## Scenario given to the agent

The agent is executing recall's Step 2.5 (lazy L2 synthesis) for a cluster whose two members
directly contradict each other. Both are ordinary vault notes.

**Member A** — `Permanent/40.2026-06-11.session-retry-policy.md`
```yaml
created: 2026-06-09
```
Body stance: **"retry transient failures up to 3 times with exponential backoff."**

**Member B** — `Permanent/41.2026-06-14.session-retry-policy.md`
```yaml
created: 2026-06-14
```
Body stance: **"do NOT auto-retry; surface the failure immediately and let the caller decide."**

These directly contradict. The agent must apply the recency tiebreaker to pick the binding stance
when deciding the content for the synthesized/updated L2.

`candidate_l2s` for this cluster lists one candidate note. The agent has already run
`engram show` on that candidate and found it covers only the older "retry up to 3×" stance
(matching Member A, not Member B). Under the current SKILL.md, that candidate is **near**
(matches the superseded content only) — the recency-weighted view picks Member B.

## Question put to the agent

> Apply the recall skill's recency-bias tiebreaker to this conflict. Which member's stance
> binds the synthesized/updated L2 — A ("retry up to 3×") or B ("do not auto-retry")?
> State the field you compared and the value for each member. Then state the coverage
> judgment (covered/near/absent) for the candidate L2 and the action you would take.

## Pass / fail

- **PASS (correct):** picks **B** ("do not auto-retry"), comparing `created` (B's 2026-06-14
  is later than A's 2026-06-09); judges the candidate **near** (it matches the superseded
  content only); takes `engram amend --target <candidate> --subject/--predicate/--object ...`
  to re-synthesize content from the recency-weighted view.
- **FAIL (the drift):** picks **A** ("retry up to 3×"), ignoring or misapplying the
  recency rule; OR marks the candidate **covered** despite it reflecting only the older stance.
