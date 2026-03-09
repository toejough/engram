package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// BackfillConfig holds injected readers for the backfill operation.
type BackfillConfig struct {
	SurfacingLog SurfacingLogReader
	CreationLog  CreationLogReader
	Evaluations  EvaluationsReader
	Scanner      MemoryScanner
	Now          time.Time
}

// CreationLogReader provides registration timestamps per memory path.
type CreationLogReader interface {
	CreationTimes() (map[string]time.Time, error)
}

// EvaluationsReader aggregates evaluation outcomes per memory path.
type EvaluationsReader interface {
	AggregateEvaluations() (map[string]EvaluationCounters, error)
}

// MemoryScanner lists memory files with metadata.
type MemoryScanner interface {
	ScanMemories() ([]ScannedMemory, error)
}

// ScannedMemory holds metadata from a memory TOML file.
type ScannedMemory struct {
	FilePath  string
	Title     string
	Content   string
	RetiredBy string
	UpdatedAt time.Time
}

// SurfacingData holds aggregated surfacing info for one memory.
type SurfacingData struct {
	Count        int
	LastSurfaced *time.Time
}

// SurfacingLogReader aggregates surfacing data per memory path.
type SurfacingLogReader interface {
	AggregateSurfacing() (map[string]SurfacingData, error)
}

// Backfill creates registry entries from existing memory data.
// Retired memories have their counters absorbed into the covering instruction.
func Backfill(config BackfillConfig) ([]InstructionEntry, error) {
	memories, err := config.Scanner.ScanMemories()
	if err != nil {
		return nil, fmt.Errorf("scanning memories: %w", err)
	}

	surfacing, err := config.SurfacingLog.AggregateSurfacing()
	if err != nil {
		return nil, fmt.Errorf("reading surfacing log: %w", err)
	}

	creationTimes, err := config.CreationLog.CreationTimes()
	if err != nil {
		return nil, fmt.Errorf("reading creation log: %w", err)
	}

	evaluations, err := config.Evaluations.AggregateEvaluations()
	if err != nil {
		return nil, fmt.Errorf("reading evaluations: %w", err)
	}

	// First pass: build entries for active (non-retired) memories.
	entries := make(map[string]*InstructionEntry, len(memories))
	retired := make([]ScannedMemory, 0)

	for _, mem := range memories {
		if mem.RetiredBy != "" {
			retired = append(retired, mem)

			continue
		}

		entry := buildEntry(mem, surfacing, creationTimes, evaluations, config.Now)
		entries[mem.FilePath] = &entry
	}

	// Second pass: absorb retired memory counters into covering instruction.
	for _, mem := range retired {
		absorbRetired(mem, entries, surfacing, evaluations)
	}

	result := make([]InstructionEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, *entry)
	}

	return result, nil
}

func absorbRetired(
	mem ScannedMemory,
	entries map[string]*InstructionEntry,
	surfacing map[string]SurfacingData,
	evaluations map[string]EvaluationCounters,
) {
	target, ok := entries[mem.RetiredBy]
	if !ok {
		return
	}

	surf := surfacing[mem.FilePath]
	evals := evaluations[mem.FilePath]

	record := AbsorbedRecord{
		From:          mem.FilePath,
		SurfacedCount: surf.Count,
		Evaluations:   evals,
		ContentHash:   contentHash(mem.Content),
		MergedAt:      mem.UpdatedAt,
	}

	target.Absorbed = append(target.Absorbed, record)
	target.SurfacedCount += surf.Count
	target.Evaluations.Followed += evals.Followed
	target.Evaluations.Contradicted += evals.Contradicted
	target.Evaluations.Ignored += evals.Ignored
}

func buildEntry(
	mem ScannedMemory,
	surfacing map[string]SurfacingData,
	creationTimes map[string]time.Time,
	evaluations map[string]EvaluationCounters,
	now time.Time,
) InstructionEntry {
	entry := InstructionEntry{
		ID:          mem.FilePath,
		SourceType:  "memory",
		SourcePath:  mem.FilePath,
		Title:       mem.Title,
		ContentHash: contentHash(mem.Content),
		UpdatedAt:   mem.UpdatedAt,
	}

	if created, ok := creationTimes[mem.FilePath]; ok {
		entry.RegisteredAt = created
	} else {
		entry.RegisteredAt = now
	}

	if surf, ok := surfacing[mem.FilePath]; ok {
		entry.SurfacedCount = surf.Count
		entry.LastSurfaced = surf.LastSurfaced
	}

	if evals, ok := evaluations[mem.FilePath]; ok {
		entry.Evaluations = evals
	}

	return entry
}

func contentHash(content string) string {
	hash := sha256.Sum256([]byte(content))

	return hex.EncodeToString(hash[:])
}
