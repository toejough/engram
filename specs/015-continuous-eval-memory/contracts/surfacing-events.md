# Contract: Surfacing Events

**Spec**: FR-002, FR-004, FR-009, FR-018, FR-019 | **Priority**: P1, P2

## Interface

```go
// LogSurfacingEvent records a memory surfacing with filter results.
LogSurfacingEvent(db *sql.DB, event SurfacingEvent) error

// GetSessionSurfacingEvents retrieves all surfacing events for a session.
GetSessionSurfacingEvents(db *sql.DB, sessionID string) ([]SurfacingEvent, error)

// UpdateSurfacingOutcome fills in post-interaction evaluation fields.
UpdateSurfacingOutcome(db *sql.DB, eventID int64, faithfulness float64, outcome string) error

// UpdateSurfacingFeedback records explicit user feedback on the most recent surfacing.
UpdateSurfacingFeedback(db *sql.DB, sessionID string, feedback string) error

// GetMemorySurfacingHistory retrieves surfacing events for a specific memory.
GetMemorySurfacingHistory(db *sql.DB, memoryID int64, limit int) ([]SurfacingEvent, error)
```

## Data Type: SurfacingEvent

| Field | Type | Description |
|-------|------|-------------|
| ID | int64 | Auto-generated |
| MemoryID | int64 | FK to embeddings |
| QueryText | string | Triggering query |
| HookEvent | string | Hook type name |
| Timestamp | time.Time | When surfaced |
| SessionID | string | Claude Code session |
| HaikuRelevant | *bool | Filter decision (nil = not filtered) |
| HaikuTag | string | Classification tag |
| HaikuRelevanceScore | *float64 | Filter confidence |
| ShouldSynthesize | *bool | Synthesis recommendation |
| Faithfulness | *float64 | Post-eval score |
| OutcomeSignal | string | "positive", "negative", "" |
| UserFeedback | string | "helpful", "wrong", "unclear", "" |
| E5Similarity | float64 | Original similarity score |
| ContextPrecision | float64 | Kept/total ratio |

## Behavior

### LogSurfacingEvent
- Inserts a single row per memory per query.
- Multiple memories from one query share the same `context_precision` value.
- Called by the hook pipeline after `Filter()` returns — logs both kept and filtered memories.

### GetSessionSurfacingEvents
- Returns all events for a session, ordered by timestamp.
- Used by end-of-session scoring to batch-process evaluations.

### UpdateSurfacingOutcome
- Updates `faithfulness` and `outcome_signal` on an existing event.
- Called during end-of-session scoring after Haiku post-evaluation.

### UpdateSurfacingFeedback
- Finds the most recent surfacing_event for the given session where `haiku_relevant=true` and updates `user_feedback`.
- Called by `/memory helpful|wrong|unclear` commands.
- If no relevant surfacing event exists for the session, returns a descriptive error.

### GetMemorySurfacingHistory
- Returns the N most recent surfacing events for a memory, ordered by timestamp desc.
- Used by leech diagnosis and quadrant scoring.

## Error Handling

| Condition | Behavior |
|-----------|----------|
| Invalid memory_id (FK violation) | Return error |
| No events for session | Return empty slice |
| No recent surfacing for feedback | Return descriptive error |

## Constraints

- All writes use the existing WAL mode + busy timeout settings.
- No batched inserts needed (one event per memory per query; typical batch is 5-10 events).
