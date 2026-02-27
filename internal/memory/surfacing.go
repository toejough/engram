package memory

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// SurfacingEvent tracks a memory surfacing with filter results and post-interaction evaluation.
type SurfacingEvent struct {
	ID                  int64
	MemoryID            int64
	QueryText           string
	HookEvent           string
	Timestamp           time.Time
	SessionID           string
	HaikuRelevant       *bool
	HaikuTag            string
	HaikuRelevanceScore *float64
	ShouldSynthesize    *bool
	Faithfulness        *float64
	OutcomeSignal       string
	UserFeedback        string
	E5Similarity        float64
	ContextPrecision    float64
}

// FindLatestUnscoredSession returns the session_id of the most recent session
// that has surfacing events with haiku_relevant=true and faithfulness IS NULL.
// Returns "" if no such session exists.
func FindLatestUnscoredSession(db *sql.DB) (string, error) {
	var sessionID string

	err := db.QueryRow(`
		SELECT session_id FROM surfacing_events
		WHERE haiku_relevant = 1 AND faithfulness IS NULL AND session_id != ''
		ORDER BY timestamp DESC
		LIMIT 1`).Scan(&sessionID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("FindLatestUnscoredSession: %w", err)
	}

	return sessionID, nil
}

// GetMemorySurfacingHistory returns the most recent surfacing events for a memory, ordered by timestamp DESC.
func GetMemorySurfacingHistory(db *sql.DB, memoryID int64, limit int) ([]SurfacingEvent, error) {
	rows, err := db.Query(`
		SELECT id, memory_id, query_text, hook_event, timestamp, session_id,
		       haiku_relevant, haiku_tag, haiku_relevance_score, should_synthesize,
		       faithfulness, outcome_signal, user_feedback, e5_similarity, context_precision
		FROM surfacing_events
		WHERE memory_id = ?
		ORDER BY timestamp DESC
		LIMIT ?`, memoryID, limit)
	if err != nil {
		return nil, fmt.Errorf("GetMemorySurfacingHistory: %w", err)
	}

	defer func() { _ = rows.Close() }()

	return scanSurfacingEvents(rows)
}

// GetSessionSurfacingEvents returns all surfacing events for the given session, ordered by timestamp ASC.
func GetSessionSurfacingEvents(db *sql.DB, sessionID string) ([]SurfacingEvent, error) {
	rows, err := db.Query(`
		SELECT id, memory_id, query_text, hook_event, timestamp, session_id,
		       haiku_relevant, haiku_tag, haiku_relevance_score, should_synthesize,
		       faithfulness, outcome_signal, user_feedback, e5_similarity, context_precision
		FROM surfacing_events
		WHERE session_id = ?
		ORDER BY timestamp ASC`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("GetSessionSurfacingEvents: %w", err)
	}

	defer func() { _ = rows.Close() }()

	return scanSurfacingEvents(rows)
}

// LogSurfacingEvent inserts a surfacing event into the database and returns the new row ID.
func LogSurfacingEvent(db *sql.DB, event SurfacingEvent) (int64, error) {
	var haikuRelevant, shouldSynthesize any

	if event.HaikuRelevant != nil {
		if *event.HaikuRelevant {
			haikuRelevant = 1
		} else {
			haikuRelevant = 0
		}
	}

	if event.ShouldSynthesize != nil {
		if *event.ShouldSynthesize {
			shouldSynthesize = 1
		} else {
			shouldSynthesize = 0
		}
	}

	var haikuRelevanceScore, faithfulness any
	if event.HaikuRelevanceScore != nil {
		haikuRelevanceScore = *event.HaikuRelevanceScore
	}

	if event.Faithfulness != nil {
		faithfulness = *event.Faithfulness
	}

	result, err := db.Exec(
		`INSERT INTO surfacing_events
			(memory_id, query_text, hook_event, timestamp, session_id,
			 haiku_relevant, haiku_tag, haiku_relevance_score, should_synthesize,
			 faithfulness, outcome_signal, user_feedback, e5_similarity, context_precision)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.MemoryID, event.QueryText, event.HookEvent,
		event.Timestamp.Format(time.RFC3339), event.SessionID,
		haikuRelevant, event.HaikuTag, haikuRelevanceScore, shouldSynthesize,
		faithfulness, event.OutcomeSignal, event.UserFeedback,
		event.E5Similarity, event.ContextPrecision,
	)
	if err != nil {
		return 0, fmt.Errorf("LogSurfacingEvent: %w", err)
	}

	return result.LastInsertId()
}

// RecordMemoryFeedback adjusts the impact_score on the embedding for the most recently
// surfaced memory in the session. Adjustments: helpful +0.1, wrong -0.2, unclear -0.05.
// Score is capped to [-1.0, 1.0]. Returns an error for invalid feedback types.
func RecordMemoryFeedback(db *sql.DB, sessionID string, feedbackType string) error {
	var delta float64

	switch feedbackType {
	case "helpful":
		delta = 0.1
	case "wrong":
		delta = -0.2
	case "unclear":
		delta = -0.05
	default:
		return fmt.Errorf("RecordMemoryFeedback: invalid feedback type %q", feedbackType)
	}

	var eventID, memoryID int64

	err := db.QueryRow(`
		SELECT id, memory_id FROM surfacing_events
		WHERE haiku_relevant = 1 AND session_id = ?
		ORDER BY timestamp DESC
		LIMIT 1`, sessionID).Scan(&eventID, &memoryID)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("RecordMemoryFeedback: no relevant surfacing event found for session %q", sessionID)
	}

	if err != nil {
		return fmt.Errorf("RecordMemoryFeedback: %w", err)
	}

	_, err = db.Exec("UPDATE surfacing_events SET user_feedback = ? WHERE id = ?", feedbackType, eventID)
	if err != nil {
		return fmt.Errorf("RecordMemoryFeedback: update feedback: %w", err)
	}

	_, err = db.Exec(`
		UPDATE embeddings
		SET impact_score = MAX(-1.0, MIN(1.0, impact_score + ?))
		WHERE id = ?`, delta, memoryID)
	if err != nil {
		return fmt.Errorf("RecordMemoryFeedback: update impact_score: %w", err)
	}

	return nil
}

// UpdateSurfacingFeedback updates the user_feedback field on the most recent surfacing event
// where haiku_relevant=true for the given session. Returns an error if no such event exists.
func UpdateSurfacingFeedback(db *sql.DB, sessionID string, feedbackType string) error {
	var id int64

	err := db.QueryRow(`
		SELECT id FROM surfacing_events
		WHERE haiku_relevant = 1 AND session_id = ?
		ORDER BY timestamp DESC
		LIMIT 1`, sessionID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("UpdateSurfacingFeedback: no relevant surfacing event found for session %q", sessionID)
	}

	if err != nil {
		return fmt.Errorf("UpdateSurfacingFeedback: %w", err)
	}

	_, err = db.Exec("UPDATE surfacing_events SET user_feedback = ? WHERE id = ?", feedbackType, id)
	if err != nil {
		return fmt.Errorf("UpdateSurfacingFeedback: %w", err)
	}

	return nil
}

// scanSurfacingEvents scans rows from surfacing_events into SurfacingEvent structs.
func scanSurfacingEvents(rows *sql.Rows) ([]SurfacingEvent, error) {
	events := make([]SurfacingEvent, 0)

	for rows.Next() {
		var (
			e                                                                 SurfacingEvent
			tsStr                                                             string
			sessionID, haikuTag, outcomeSignal, userFeedback                  sql.NullString
			haikuRelevant, shouldSynthesize                                   sql.NullBool
			haikuRelevanceScore, faithfulness, e5Similarity, contextPrecision sql.NullFloat64
		)

		if err := rows.Scan(
			&e.ID, &e.MemoryID, &e.QueryText, &e.HookEvent, &tsStr,
			&sessionID, &haikuRelevant, &haikuTag, &haikuRelevanceScore,
			&shouldSynthesize, &faithfulness, &outcomeSignal, &userFeedback,
			&e5Similarity, &contextPrecision,
		); err != nil {
			return nil, fmt.Errorf("scanSurfacingEvents: %w", err)
		}

		ts, err := time.Parse(time.RFC3339, tsStr)
		if err != nil {
			return nil, fmt.Errorf("scanSurfacingEvents: parse timestamp %q: %w", tsStr, err)
		}

		e.Timestamp = ts

		if sessionID.Valid {
			e.SessionID = sessionID.String
		}

		if haikuRelevant.Valid {
			v := haikuRelevant.Bool
			e.HaikuRelevant = &v
		}

		if haikuTag.Valid {
			e.HaikuTag = haikuTag.String
		}

		if haikuRelevanceScore.Valid {
			v := haikuRelevanceScore.Float64
			e.HaikuRelevanceScore = &v
		}

		if shouldSynthesize.Valid {
			v := shouldSynthesize.Bool
			e.ShouldSynthesize = &v
		}

		if faithfulness.Valid {
			v := faithfulness.Float64
			e.Faithfulness = &v
		}

		if outcomeSignal.Valid {
			e.OutcomeSignal = outcomeSignal.String
		}

		if userFeedback.Valid {
			e.UserFeedback = userFeedback.String
		}

		if e5Similarity.Valid {
			e.E5Similarity = e5Similarity.Float64
		}

		if contextPrecision.Valid {
			e.ContextPrecision = contextPrecision.Float64
		}

		events = append(events, e)
	}

	return events, rows.Err()
}
