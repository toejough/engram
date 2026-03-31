// Package memory defines shared types for the engram memory pipeline.
package memory

import (
	"strings"
	"time"
)

// Stored represents a memory read back from a TOML file on disk (ARCH-9).
type Stored struct {
	Situation        string
	Behavior         string
	Impact           string
	Action           string
	ProjectScoped    bool
	ProjectSlug      string
	SurfacedCount    int
	FollowedCount    int
	NotFollowedCount int
	IrrelevantCount  int
	UpdatedAt        time.Time
	FilePath         string
}

// SearchText returns a concatenation of all searchable fields for retrieval scoring.
func (s *Stored) SearchText() string {
	parts := make([]string, 0, searchTextCapacity)

	if s.Situation != "" {
		parts = append(parts, s.Situation)
	}

	if s.Behavior != "" {
		parts = append(parts, s.Behavior)
	}

	if s.Impact != "" {
		parts = append(parts, s.Impact)
	}

	if s.Action != "" {
		parts = append(parts, s.Action)
	}

	return strings.Join(parts, " ")
}

// TotalEvaluations returns the sum of all evaluation counters.
func (s *Stored) TotalEvaluations() int {
	return s.FollowedCount + s.NotFollowedCount + s.IrrelevantCount
}

// unexported constants.
const (
	searchTextCapacity = 4
)
