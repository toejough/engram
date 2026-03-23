package signal

import (
	"time"

	"engram/internal/memory"
)

// TransferFields applies counter transfer rules from originals onto a base
// consolidated memory. Mutates base in place. Per spec: sum followed/contradicted,
// reset irrelevant/ignored/surfaced, set confidence B, clear project_slug,
// take max enforcement level.
func TransferFields(base *memory.MemoryRecord, originals []*memory.MemoryRecord, now time.Time) {
	var totalFollowed, totalContradicted int

	maxEnforcement := base.EnforcementLevel

	absorbed := make([]memory.AbsorbedRecord, 0, len(originals))

	for _, orig := range originals {
		totalFollowed += orig.FollowedCount
		totalContradicted += orig.ContradictedCount

		if enforcementRank(orig.EnforcementLevel) > enforcementRank(maxEnforcement) {
			maxEnforcement = orig.EnforcementLevel
		}

		absorbed = append(absorbed, memory.AbsorbedRecord{
			From:          orig.SourcePath,
			SurfacedCount: orig.SurfacedCount,
			Evaluations: memory.EvaluationCounters{
				Followed:     orig.FollowedCount,
				Contradicted: orig.ContradictedCount,
				Ignored:      orig.IgnoredCount,
			},
			ContentHash: orig.ContentHash,
			MergedAt:    now.Format(time.RFC3339),
		})
	}

	base.FollowedCount = totalFollowed
	base.ContradictedCount = totalContradicted
	base.IrrelevantCount = 0
	base.IgnoredCount = 0
	base.SurfacedCount = 0
	base.Confidence = consolidatedConfidence
	base.ProjectSlug = ""
	base.EnforcementLevel = maxEnforcement
	base.Absorbed = append(base.Absorbed, absorbed...)
}

// unexported constants.
const (
	consolidatedConfidence = "B"
)

func enforcementRank(level string) int {
	switch level {
	case "reminder":
		const reminderRank = 2
		return reminderRank
	case "emphasized_advisory":
		const emphasizedRank = 1
		return emphasizedRank
	default:
		return 0
	}
}
