package memory

import (
	"database/sql"
	"fmt"
)

// HookStat contains statistics for a hook.
type HookStat struct {
	HookName    string
	FireCount   int
	SuccessRate float64
	AvgDuration int
	LastFired   string
}

// GetHookStats retrieves statistics for all hooks.
func GetHookStats(db *sql.DB) ([]HookStat, error) {
	query := `
		SELECT
			hook_name,
			COUNT(*) as fire_count,
			SUM(CASE WHEN exit_code = 0 THEN 1 ELSE 0 END) * 1.0 / COUNT(*) as success_rate,
			CAST(AVG(duration_ms) AS INTEGER) as avg_duration,
			MAX(fired_at) as last_fired
		FROM hook_events
		GROUP BY hook_name
		ORDER BY hook_name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query hook stats: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var stats []HookStat

	for rows.Next() {
		var s HookStat

		err := rows.Scan(&s.HookName, &s.FireCount, &s.SuccessRate, &s.AvgDuration, &s.LastFired)
		if err != nil {
			return nil, fmt.Errorf("failed to scan hook stat: %w", err)
		}

		stats = append(stats, s)
	}

	return stats, rows.Err()
}

// RecordHookEvent records a hook execution event and prunes old events.
// Keeps last 1000 events per hook.
func RecordHookEvent(db *sql.DB, hookName string, exitCode int, durationMs int) error {
	// Insert event
	_, err := db.Exec(`
		INSERT INTO hook_events (hook_name, fired_at, exit_code, duration_ms)
		VALUES (?, datetime('now'), ?, ?)
	`, hookName, exitCode, durationMs)
	if err != nil {
		return fmt.Errorf("failed to insert hook event: %w", err)
	}

	// Prune to keep last 1000 per hook
	_, err = db.Exec(`
		DELETE FROM hook_events
		WHERE hook_name = ?
		AND id NOT IN (
			SELECT id FROM hook_events
			WHERE hook_name = ?
			ORDER BY id DESC
			LIMIT 1000
		)
	`, hookName, hookName)
	if err != nil {
		return fmt.Errorf("failed to prune hook events: %w", err)
	}

	return nil
}
