package memory

// AbsorbedRecord records a memory that was merged into this one.
type AbsorbedRecord struct {
	From          string             `toml:"from"`
	SurfacedCount int                `toml:"surfaced_count"`
	Evaluations   EvaluationCounters `toml:"evaluations"`
	ContentHash   string             `toml:"content_hash"`
	MergedAt      string             `toml:"merged_at"`
}

// EvaluationCounters holds feedback outcome counts for an absorbed memory.
type EvaluationCounters struct {
	Followed     int `toml:"followed"`
	Contradicted int `toml:"contradicted"`
	Ignored      int `toml:"ignored"`
}

// MaintenanceAction records a single maintenance action applied to this memory
// and its before/after effectiveness for outcome tracking (#387).
type MaintenanceAction struct {
	Action              string  `toml:"action"`
	AppliedAt           string  `toml:"applied_at"`
	EffectivenessBefore float64 `toml:"effectiveness_before"`
	SurfacedCountBefore int     `toml:"surfaced_count_before"`
	FeedbackCountBefore int     `toml:"feedback_count_before"`
	EffectivenessAfter  float64 `toml:"effectiveness_after"`
	SurfacedCountAfter  int     `toml:"surfaced_count_after"`
	Measured            bool    `toml:"measured"`
}

// MemoryRecord is the canonical struct for reading and writing memory TOML files.
//
//nolint:revive // "memory.MemoryRecord" stutter is intentional for clarity. See #353.
// ALL code that touches memory TOML must use this struct to prevent field loss.
// See #353 for the bug caused by divergent struct definitions.
type MemoryRecord struct {
	// Content fields.
	Title            string   `toml:"title"`
	Content          string   `toml:"content"`
	ObservationType  string   `toml:"observation_type"`
	Concepts         []string `toml:"concepts"`
	Keywords         []string `toml:"keywords"`
	Principle        string   `toml:"principle"`
	AntiPattern      string   `toml:"anti_pattern"`
	Rationale        string   `toml:"rationale"`
	ProjectSlug      string   `toml:"project_slug,omitempty"`
	Generalizability int      `toml:"generalizability,omitempty"`
	Confidence       string   `toml:"confidence"`
	CreatedAt        string   `toml:"created_at"`
	UpdatedAt        string   `toml:"updated_at"`

	// Tracking fields — feedback counters and surfacing metadata.
	SurfacedCount     int      `toml:"surfaced_count"`
	FollowedCount     int      `toml:"followed_count"`
	ContradictedCount int      `toml:"contradicted_count"`
	IgnoredCount      int      `toml:"ignored_count"`
	IrrelevantCount   int      `toml:"irrelevant_count"`
	IrrelevantQueries []string `toml:"irrelevant_queries,omitempty"`
	LastSurfacedAt    string   `toml:"last_surfaced_at"`

	// Provenance.
	SourceType  string `toml:"source_type,omitempty"`
	SourcePath  string `toml:"source_path,omitempty"`
	ContentHash string `toml:"content_hash,omitempty"`

	// Relationships.
	Absorbed []AbsorbedRecord `toml:"absorbed,omitempty"`

	// Maintenance history — tracks action outcomes for adaptive policy (#387).
	MaintenanceHistory []MaintenanceAction `toml:"maintenance_history,omitempty"`
}

// TotalFeedback returns the sum of all evaluation counters.
func (r *MemoryRecord) TotalFeedback() int {
	return r.FollowedCount + r.ContradictedCount + r.IgnoredCount + r.IrrelevantCount
}
