package memory

import (
	"database/sql"
	"fmt"
)

// RecurringProblem represents a detected pattern of failures or issues.
type RecurringProblem struct {
	Source      string  // "hook" or "feedback"
	Name        string  // hook name or content snippet
	Count       int     // occurrence count
	Rate        float64 // failure rate (hooks) or 0 (feedback)
	Description string  // human-readable description
}

// RecurringProblemOpts holds options for recurring problem detection.
type RecurringProblemOpts struct {
	HookFailureRate  float64 // Min failure rate to flag (default 0.3)
	HookWindowDays   int     // Look-back window (default 7)
	FeedbackMinWrong int     // Min "wrong" feedback to flag (default 2)
}

// DetectRecurringProblems scans hook events and feedback for recurring problems.
func DetectRecurringProblems(db *sql.DB, opts RecurringProblemOpts) ([]RecurringProblem, error) {
	// Apply defaults
	if opts.HookFailureRate == 0 {
		opts.HookFailureRate = 0.3
	}
	if opts.HookWindowDays == 0 {
		opts.HookWindowDays = 7
	}
	if opts.FeedbackMinWrong == 0 {
		opts.FeedbackMinWrong = 2
	}

	var problems []RecurringProblem

	// 1. Detect hook failure patterns
	hookQuery := `
		SELECT hook_name, COUNT(*) as total,
			SUM(CASE WHEN exit_code != 0 THEN 1 ELSE 0 END) as failures
		FROM hook_events
		WHERE fired_at > datetime('now', '-' || ? || ' days')
		GROUP BY hook_name
		HAVING failures * 1.0 / total > ?
	`
	rows, err := db.Query(hookQuery, opts.HookWindowDays, opts.HookFailureRate)
	if err != nil {
		return nil, fmt.Errorf("failed to query hook failures: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var hookName string
		var total, failures int
		if err := rows.Scan(&hookName, &total, &failures); err != nil {
			return nil, fmt.Errorf("failed to scan hook failure: %w", err)
		}

		rate := float64(failures) / float64(total)
		description := fmt.Sprintf("Hook %s has %.1f%% failure rate (%d failures in last %d days)",
			hookName, rate*100, failures, opts.HookWindowDays)

		problems = append(problems, RecurringProblem{
			Source:      "hook",
			Name:        hookName,
			Count:       total,
			Rate:        rate,
			Description: description,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating hook failures: %w", err)
	}

	// 2. Detect feedback clusters
	feedbackQuery := `
		SELECT e.id, e.content, COUNT(*) as wrong_count
		FROM feedback f
		JOIN embeddings e ON f.embedding_id = e.id
		WHERE f.feedback_type = 'wrong'
		GROUP BY e.id
		HAVING wrong_count >= ?
	`
	rows2, err := db.Query(feedbackQuery, opts.FeedbackMinWrong)
	if err != nil {
		return nil, fmt.Errorf("failed to query feedback clusters: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var embeddingID int64
		var content string
		var wrongCount int
		if err := rows2.Scan(&embeddingID, &content, &wrongCount); err != nil {
			return nil, fmt.Errorf("failed to scan feedback cluster: %w", err)
		}

		// Use content snippet as name (truncate if needed)
		name := content
		if len(name) > 50 {
			name = name[:47] + "..."
		}

		description := fmt.Sprintf("Content has %d 'wrong' feedback responses", wrongCount)

		problems = append(problems, RecurringProblem{
			Source:      "feedback",
			Name:        name,
			Count:       wrongCount,
			Rate:        0, // Feedback doesn't have a rate
			Description: description,
		})
	}
	if err := rows2.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feedback clusters: %w", err)
	}

	return problems, nil
}

// ProblemsToProposals converts recurring problems to MaintenanceProposal format.
func ProblemsToProposals(problems []RecurringProblem) []MaintenanceProposal {
	proposals := make([]MaintenanceProposal, 0, len(problems))

	for _, p := range problems {
		var preview string
		if p.Source == "hook" {
			preview = fmt.Sprintf("Hook %s has %.1f%% failure rate (%d total events)",
				p.Name, p.Rate*100, p.Count)
		} else {
			preview = p.Name
		}

		proposals = append(proposals, MaintenanceProposal{
			Tier:    "meta",
			Action:  "surface",
			Target:  p.Name,
			Reason:  p.Description,
			Preview: preview,
		})
	}

	return proposals
}
