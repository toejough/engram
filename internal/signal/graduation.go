package signal

import (
	"errors"
	"time"
)

// Exported variables.
var (
	ErrGraduationNotFound = errors.New("graduation entry not found")
)

// GraduationEntry persists one graduated memory's lifecycle.
//
//nolint:tagliatelle // spec requires snake_case JSON field names.
type GraduationEntry struct {
	ID             string    `json:"id"`
	MemoryPath     string    `json:"memory_path"`
	Recommendation string    `json:"recommendation"` // "settings.json", ".claude/rules/", "skill", "CLAUDE.md"
	Status         string    `json:"status"`         // "pending", "accepted", "dismissed"
	DetectedAt     time.Time `json:"detected_at"`
	ResolvedAt     string    `json:"resolved_at"` // RFC3339 or empty
	IssueURL       string    `json:"issue_url"`   // empty until accepted
}

// TimestampNow returns the current time as an RFC3339 string.
func TimestampNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}
