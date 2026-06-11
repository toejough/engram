# Baseline test — recency tiebreaker on L1 episodes (write-order vs work-order)

**What this tests:** the recency-bias tiebreaker (Step 3a dispatched-synthesis subagent / no-dispatch
inline fallback) when the divergent cluster members are **L1 episodes** whose **write-order
(`created`) is the opposite of their work-order (`provenance.transcript_range.end`)**.

**Spec rule under test** (`docs/superpowers/specs/2026-06-09-lazy-l2-synthesis-design.md`, Locked twice):
- §2: `"Newer" = the frontmatter created date (for an L1 episode, its transcript-range end)`
- §5 Locked: `prefer the more-recently-created member's content (created date; L1 = transcript-range end)`

So for two conflicting **L1 episodes**, the binding stance is the one with the later
`provenance.transcript_range.end`, NOT the later `created`.

## Scenario given to the agent

The agent is executing the recall skill's blocking L2 synthesis for one UPDATE/CREATE-band cluster.
Two cluster members **conflict** on a convention. Both are L1 episodes. Their frontmatter:

**Member A** — `Permanent/40.2026-06-11.session-retry-policy.md`
```yaml
created: 2026-06-11
node_type: episode
provenance:
  transcript_range:
    start: 2026-06-09T08:30:00Z
    end:   2026-06-09T09:15:00Z
```
Body stance on the convention: **"retry transient failures up to 3 times with backoff."**

**Member B** — `Permanent/41.2026-06-10.session-retry-policy.md`
```yaml
created: 2026-06-10
node_type: episode
provenance:
  transcript_range:
    start: 2026-06-10T19:40:00Z
    end:   2026-06-10T20:05:00Z
```
Body stance on the convention: **"do NOT auto-retry; surface the failure and let the caller decide."**

These directly contradict. The agent must apply the recency tiebreaker to pick the binding stance
for the synthesized/updated L2.

(Note the trap: A was *written* later — `created` 2026-06-11 > 2026-06-10 — but B's *work* is later —
`transcript_range.end` 2026-06-10T20:05Z > 2026-06-09T09:15Z. Write-order and work-order disagree.)

## Question put to the agent

> Apply the recall skill's recency-bias tiebreaker to this conflict. Which member's stance binds the
> synthesized L2 — A ("retry up to 3×") or B ("do not auto-retry")? State the field you compared and
> the value for each member.

## Pass / fail

- **PASS (spec-correct):** picks **B** ("do not auto-retry"), comparing `provenance.transcript_range.end`
  (B 2026-06-10T20:05Z is later than A 2026-06-09T09:15Z) — work-order, per the Locked spec rule.
- **FAIL (the drift):** picks **A** ("retry up to 3×"), comparing `created` (A 2026-06-11 is later than
  B 2026-06-10) — write-order. This is what the current SKILL's `created`-only wording produces.
