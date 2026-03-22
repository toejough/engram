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

// LinkRecord represents a directed relationship between two memory files.
type LinkRecord struct {
	Target           string  `toml:"target"`
	Weight           float64 `toml:"weight"`
	Basis            string  `toml:"basis"`
	CoSurfacingCount int     `toml:"co_surfacing_count,omitempty"`
}

// MemoryRecord is the canonical struct for reading and writing memory TOML files.
//
//nolint:revive // "memory.MemoryRecord" stutter is intentional for clarity. See #353.
// ALL code that touches memory TOML must use this struct to prevent field loss.
// See #353 for the bug caused by divergent struct definitions.
type MemoryRecord struct {
	// Content fields.
	Title           string   `toml:"title"`
	Content         string   `toml:"content"`
	ObservationType string   `toml:"observation_type"`
	Concepts        []string `toml:"concepts"`
	Keywords        []string `toml:"keywords"`
	Principle       string   `toml:"principle"`
	AntiPattern     string   `toml:"anti_pattern"`
	Rationale        string `toml:"rationale"`
	ProjectSlug      string `toml:"project_slug,omitempty"`
	Generalizability int    `toml:"generalizability,omitempty"`
	Confidence       string `toml:"confidence"`
	CreatedAt       string   `toml:"created_at"`
	UpdatedAt       string   `toml:"updated_at"`

	// Tracking fields — feedback counters and surfacing metadata.
	SurfacedCount     int    `toml:"surfaced_count"`
	FollowedCount     int    `toml:"followed_count"`
	ContradictedCount int    `toml:"contradicted_count"`
	IgnoredCount      int    `toml:"ignored_count"`
	IrrelevantCount   int    `toml:"irrelevant_count"`
	LastSurfacedAt    string `toml:"last_surfaced_at"`

	// Provenance.
	SourceType  string `toml:"source_type,omitempty"`
	SourcePath  string `toml:"source_path,omitempty"`
	ContentHash string `toml:"content_hash,omitempty"`

	// Enforcement escalation.
	EnforcementLevel string             `toml:"enforcement_level,omitempty"`
	Transitions      []TransitionRecord `toml:"transitions,omitempty"`

	// Relationships.
	Links    []LinkRecord     `toml:"links,omitempty"`
	Absorbed []AbsorbedRecord `toml:"absorbed,omitempty"`
}

// TransitionRecord records an enforcement level change.
type TransitionRecord struct {
	From   string `toml:"from"`
	To     string `toml:"to"`
	At     string `toml:"at"`
	Reason string `toml:"reason"`
}
