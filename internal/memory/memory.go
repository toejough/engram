// Package memory defines shared types for the engram memory pipeline.
package memory

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Stored represents a memory read back from a TOML file on disk (ARCH-9).
type Stored struct {
	Type      string
	Situation string
	Source    string
	Content   ContentFields
	UpdatedAt time.Time
	FilePath  string
}

// BuildIndex renders the type | name | situation index used for Haiku matching
// during recall and conflict detection.
func BuildIndex(memories []*Stored) string {
	var builder strings.Builder

	for _, mem := range memories {
		name := NameFromPath(mem.FilePath)
		fmt.Fprintf(&builder, "%s | %s | %s\n", mem.Type, name, mem.Situation)
	}

	return builder.String()
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
