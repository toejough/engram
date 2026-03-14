// Package effectiveness computes memory effectiveness scores from embedded TOML metrics.
package effectiveness

import "engram/internal/memory"

// Stat holds aggregated effectiveness counts for a single memory.
type Stat struct {
	FollowedCount      int
	ContradictedCount  int
	IgnoredCount       int
	EffectivenessScore float64 // followed / (followed + contradicted + ignored) * 100
}

// FromMemories builds per-path effectiveness stats from in-memory Stored objects.
// Scores are computed from the evaluation counters embedded in each memory TOML file.
func FromMemories(memories []*memory.Stored) map[string]Stat {
	const percentMultiplier = 100.0

	result := make(map[string]Stat, len(memories))

	for _, mem := range memories {
		total := mem.FollowedCount + mem.ContradictedCount + mem.IgnoredCount

		var score float64
		if total > 0 {
			score = float64(mem.FollowedCount) / float64(total) * percentMultiplier
		}

		result[mem.FilePath] = Stat{
			FollowedCount:      mem.FollowedCount,
			ContradictedCount:  mem.ContradictedCount,
			IgnoredCount:       mem.IgnoredCount,
			EffectivenessScore: score,
		}
	}

	return result
}
