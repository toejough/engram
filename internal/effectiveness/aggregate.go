// Package effectiveness computes memory effectiveness scores from evaluation logs.
package effectiveness

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Computer aggregates evaluation logs into per-memory effectiveness stats.
type Computer struct {
	evalDir  string
	readDir  func(string) ([]os.DirEntry, error)
	readFile func(string) ([]byte, error)
}

// New creates a Computer that reads .jsonl files from evalDir.
func New(evalDir string, opts ...Option) *Computer {
	computer := &Computer{
		evalDir:  evalDir,
		readDir:  os.ReadDir,
		readFile: os.ReadFile,
	}
	for _, opt := range opts {
		opt(computer)
	}

	return computer
}

// Aggregate reads all .jsonl files in evalDir and returns per-memory stats.
// Missing directory returns empty map with no error. Malformed lines are skipped.
func (c *Computer) Aggregate() (map[string]Stat, error) {
	entries, err := c.readDir(c.evalDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make(map[string]Stat), nil
		}

		return nil, fmt.Errorf("reading evaluations directory: %w", err)
	}

	counts := make(map[string]*Stat)

	for _, dirEntry := range entries {
		if dirEntry.IsDir() || !strings.HasSuffix(dirEntry.Name(), ".jsonl") {
			continue
		}

		c.processFile(filepath.Join(c.evalDir, dirEntry.Name()), counts)
	}

	return buildResult(counts), nil
}

// processFile reads a single .jsonl file and accumulates outcome counts into counts.
// Unreadable files and malformed lines are silently skipped.
func (c *Computer) processFile(filePath string, counts map[string]*Stat) {
	data, err := c.readFile(filePath)
	if err != nil {
		return
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var logEntry evalEntry

		jsonErr := json.Unmarshal([]byte(line), &logEntry)
		if jsonErr != nil {
			continue // skip malformed lines
		}

		if logEntry.MemoryPath == "" {
			continue
		}

		applyOutcome(counts, logEntry.MemoryPath, logEntry.Outcome)
	}
}

// Option configures a Computer.
type Option func(*Computer)

// Stat holds aggregated effectiveness counts for a single memory.
type Stat struct {
	FollowedCount      int
	ContradictedCount  int
	IgnoredCount       int
	EffectivenessScore float64 // followed / (followed + contradicted + ignored) * 100
}

// WithReadDir injects a directory reader for testing.
func WithReadDir(fn func(string) ([]os.DirEntry, error)) Option {
	return func(c *Computer) {
		c.readDir = fn
	}
}

// WithReadFile injects a file reader for testing.
func WithReadFile(fn func(string) ([]byte, error)) Option {
	return func(c *Computer) {
		c.readFile = fn
	}
}

// evalEntry is the JSON structure of a single evaluation log line.
//
//nolint:tagliatelle // external log format uses snake_case field names
type evalEntry struct {
	MemoryPath string `json:"memory_path"`
	Outcome    string `json:"outcome"`
}

// applyOutcome increments the appropriate counter for a memory path.
func applyOutcome(counts map[string]*Stat, memoryPath, outcome string) {
	stat, ok := counts[memoryPath]
	if !ok {
		stat = &Stat{}
		counts[memoryPath] = stat
	}

	switch outcome {
	case "followed":
		stat.FollowedCount++
	case "contradicted":
		stat.ContradictedCount++
	case "ignored":
		stat.IgnoredCount++
	}
}

// buildResult converts raw counts to final Stat values with computed scores.
func buildResult(counts map[string]*Stat) map[string]Stat {
	const percentMultiplier = 100.0

	result := make(map[string]Stat, len(counts))

	for memoryPath, stat := range counts {
		total := stat.FollowedCount + stat.ContradictedCount + stat.IgnoredCount
		if total > 0 {
			stat.EffectivenessScore = float64(
				stat.FollowedCount,
			) / float64(
				total,
			) * percentMultiplier
		}

		result[memoryPath] = *stat
	}

	return result
}
