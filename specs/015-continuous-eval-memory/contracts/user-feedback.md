# Contract: User Feedback Commands

**Spec**: FR-012 | **Priority**: P5

## Interface

```go
// RecordMemoryFeedback records explicit user feedback on the most recently surfaced memory.
RecordMemoryFeedback(db *sql.DB, sessionID string, feedbackType string) error
```

## CLI Command

```
projctl memory feedback --type=helpful|wrong|unclear --session-id=<session_id>
```

Invoked by `/memory helpful`, `/memory wrong`, `/memory unclear` slash commands.

## Behavior

1. Finds the most recent surfacing_event for the given session where `haiku_relevant=true`.
2. Updates `user_feedback` field with the feedback type.
3. Immediately adjusts `impact_score` for the memory:
   - "helpful": +0.1 (capped at 1.0)
   - "wrong": -0.2 (floored at -1.0)
   - "unclear": -0.05
4. One explicit signal is weighted equivalent to ~10 implicit signals.

## Error Handling

| Condition | Behavior |
|-----------|----------|
| No surfacing event for session | Return "No recent memory surfacing to rate" error |
| Invalid feedback type | Return "Valid types: helpful, wrong, unclear" error |
| Session ID missing | Attempt to derive from environment/stdin |

## Constraints

- Feedback applies to the most recent relevant surfacing event only.
- Multiple feedback commands in one session overwrite (last wins).
