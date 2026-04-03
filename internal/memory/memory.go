// Package memory defines shared types for the engram memory pipeline.
package memory

import (
	"path/filepath"
	"strings"
	"time"
)

// Stored represents a memory read back from a TOML file on disk (ARCH-9).
type Stored struct {
	Type               string
	Situation          string
	Content            ContentFields
	Core               bool
	InitialConfidence  float64
	ProjectScoped      bool
	ProjectSlug        string
	SurfacedCount      int
	FollowedCount      int
	NotFollowedCount   int
	IrrelevantCount    int
	CreatedAt          time.Time
	UpdatedAt          time.Time
	FilePath           string
	PendingEvaluations []PendingEvaluation
}

// SearchText returns a concatenation of all searchable fields for retrieval scoring.
func (s *Stored) SearchText() string {
	parts := make([]string, 0, searchTextCapacity)

	if s.Situation != "" {
		parts = append(parts, s.Situation)
	}

	parts = appendContentFields(parts, s.Type, s.Content)

	return strings.Join(parts, " ")
}

// TotalEvaluations returns the sum of all evaluation counters.
func (s *Stored) TotalEvaluations() int {
	return s.FollowedCount + s.NotFollowedCount + s.IrrelevantCount
}

// FactsDir returns the directory for fact memory files.
func FactsDir(dataDir string) string {
	return filepath.Join(dataDir, "memory", "facts")
}

// FeedbackDir returns the directory for feedback memory files.
func FeedbackDir(dataDir string) string {
	return filepath.Join(dataDir, "memory", "feedback")
}

// MemoriesDir returns the path to the memories subdirectory within a data directory.
func MemoriesDir(dataDir string) string {
	return filepath.Join(dataDir, "memories")
}

// NameFromPath extracts the base filename without extension from a memory path.
func NameFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// ResolveMemoryPath finds the TOML file for a slug, checking new layout directories
// (memory/feedback/, memory/facts/) first, then falling back to legacy memories/.
// The fileExists function is injected for testability.
func ResolveMemoryPath(dataDir, slug string, fileExists func(string) bool) string {
	filename := slug + ".toml"

	candidates := []string{
		filepath.Join(FeedbackDir(dataDir), filename),
		filepath.Join(FactsDir(dataDir), filename),
		filepath.Join(MemoriesDir(dataDir), filename),
	}

	for _, path := range candidates {
		if fileExists(path) {
			return path
		}
	}

	// Fall back to legacy path even if it doesn't exist, so the caller
	// gets a meaningful "file not found" error.
	return filepath.Join(MemoriesDir(dataDir), filename)
}

// unexported constants.
const (
	searchTextCapacity = 4
)

// appendContentFields adds the relevant content fields based on memory type.
func appendContentFields(parts []string, memType string, content ContentFields) []string {
	if memType == "fact" {
		return appendNonEmpty(parts, content.Subject, content.Predicate, content.Object)
	}

	return appendNonEmpty(parts, content.Behavior, content.Impact, content.Action)
}

// appendNonEmpty appends non-empty strings to the slice.
func appendNonEmpty(parts []string, values ...string) []string {
	for _, val := range values {
		if val != "" {
			parts = append(parts, val)
		}
	}

	return parts
}
