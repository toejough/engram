package adapt

import "engram/internal/memory"

// CorpusSnapshot captures corpus-wide metrics at a point in time.
type CorpusSnapshot struct {
	FollowRate        float64
	IrrelevanceRatio  float64
	MeanEffectiveness float64
}

// ComputeCorpusSnapshot aggregates follow rate, irrelevance ratio, and mean
// effectiveness across all memories with feedback.
func ComputeCorpusSnapshot(memories []*memory.Stored) CorpusSnapshot {
	var totalFollowed, totalIrrelevant, totalFeedback int

	var effectivenessSum float64

	memoriesWithFeedback := 0

	for _, mem := range memories {
		feedback := mem.FollowedCount + mem.ContradictedCount + mem.IgnoredCount + mem.IrrelevantCount
		if feedback == 0 {
			continue
		}

		totalFollowed += mem.FollowedCount
		totalIrrelevant += mem.IrrelevantCount
		totalFeedback += feedback

		score := float64(mem.FollowedCount) / float64(feedback) * percentMultiplier
		effectivenessSum += score
		memoriesWithFeedback++
	}

	if totalFeedback == 0 {
		return CorpusSnapshot{}
	}

	return CorpusSnapshot{
		FollowRate:        float64(totalFollowed) / float64(totalFeedback),
		IrrelevanceRatio:  float64(totalIrrelevant) / float64(totalFeedback),
		MeanEffectiveness: effectivenessSum / float64(memoriesWithFeedback),
	}
}
