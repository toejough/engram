package memory

import "time"

// MemoryRecord is the canonical struct for reading and writing memory TOML files.
//
//nolint:revive // "memory.MemoryRecord" stutter is intentional for clarity. See #353.
// ALL code that touches memory TOML must use this struct to prevent field loss.
// See #353 for the bug caused by divergent struct definitions.
type MemoryRecord struct {
	Situation string `toml:"situation"`
	Behavior  string `toml:"behavior"`
	Impact    string `toml:"impact"`
	Action    string `toml:"action"`

	ProjectScoped bool   `toml:"project_scoped"`
	ProjectSlug   string `toml:"project_slug,omitempty"`

	CreatedAt string `toml:"created_at"`
	UpdatedAt string `toml:"updated_at"`

	SurfacedCount    int `toml:"surfaced_count"`
	FollowedCount    int `toml:"followed_count"`
	NotFollowedCount int `toml:"not_followed_count"`
	IrrelevantCount  int `toml:"irrelevant_count"`

	PendingEvaluations []PendingEvaluation `toml:"pending_evaluations,omitempty"`
}

// ToStored converts a MemoryRecord to a Stored for in-memory use.
func (r *MemoryRecord) ToStored(filePath string) *Stored {
	updatedAt, _ := time.Parse(time.RFC3339, r.UpdatedAt)

	return &Stored{
		Situation:        r.Situation,
		Behavior:         r.Behavior,
		Impact:           r.Impact,
		Action:           r.Action,
		ProjectScoped:    r.ProjectScoped,
		ProjectSlug:      r.ProjectSlug,
		SurfacedCount:    r.SurfacedCount,
		FollowedCount:    r.FollowedCount,
		NotFollowedCount: r.NotFollowedCount,
		IrrelevantCount:  r.IrrelevantCount,
		UpdatedAt:        updatedAt,
		FilePath:         filePath,
	}
}

// TotalEvaluations returns the sum of all evaluation counters.
func (r *MemoryRecord) TotalEvaluations() int {
	return r.FollowedCount + r.NotFollowedCount + r.IrrelevantCount
}

// PendingEvaluation records a surfacing event awaiting outcome feedback.
type PendingEvaluation struct {
	SurfacedAt  string `toml:"surfaced_at"`
	UserPrompt  string `toml:"user_prompt"`
	SessionID   string `toml:"session_id"`
	ProjectSlug string `toml:"project_slug"`
}
