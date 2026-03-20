package memory

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
	Rationale       string   `toml:"rationale"`
	Confidence      string   `toml:"confidence"`
	CreatedAt       string   `toml:"created_at"`
	UpdatedAt       string   `toml:"updated_at"`

	// Tracking fields — feedback counters and surfacing metadata.
	SurfacedCount     int    `toml:"surfaced_count"`
	FollowedCount     int    `toml:"followed_count"`
	ContradictedCount int    `toml:"contradicted_count"`
	IgnoredCount      int    `toml:"ignored_count"`
	IrrelevantCount   int    `toml:"irrelevant_count"`
	LastSurfacedAt    string `toml:"last_surfaced_at"`
}
