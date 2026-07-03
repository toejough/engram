package cli

import (
	"fmt"
	"time"
)

// unexported constants.
const (
	// refitGrowthMinDays is the minimum days elapsed since the last refit
	// (conjunct with refitGrowthMinNotes) to fire the growth trigger.
	refitGrowthMinDays = 14
	// refitGrowthMinNotes is the minimum new-note growth since the last refit
	// to consider the growth trigger armed.
	refitGrowthMinNotes = 40
	// refitUntaggedRateMax is the vault-wide untagged rate above which the
	// untagged trigger fires (exclusive: >8%).
	refitUntaggedRateMax = 0.08
	// hubThreshold (0.25) is defined in vocab_commands.go and reused here.
	// hoursPerDay (24) is defined in recency.go and reused here.
)

// evaluateVocabTriggers returns (fired, reason) for the in-process threshold checks.
// Returns (false, "") when lastRefit is nil (no baseline yet — caller seeds and returns).
func evaluateVocabTriggers(
	totalNotes, untaggedCount int,
	memberCounts map[string]int,
	lastRefit *vocabLastRefitDoc,
	now time.Time,
) (bool, string) {
	if lastRefit == nil {
		return false, "" // no baseline — caller seeds and returns
	}

	// (a) growth trigger
	lastRefitDate, parseErr := time.Parse(dateFormat, lastRefit.Date)
	if parseErr == nil {
		growth := totalNotes - lastRefit.NoteCount
		daysSince := int(now.Sub(lastRefitDate).Hours() / hoursPerDay)

		if growth >= refitGrowthMinNotes && daysSince >= refitGrowthMinDays {
			return true, fmt.Sprintf("growth: %d notes, %d days", growth, daysSince)
		}
	}

	// (b) untagged rate trigger
	if totalNotes > 0 {
		untaggedRate := float64(untaggedCount) / float64(totalNotes)

		if untaggedRate > refitUntaggedRateMax {
			return true, fmt.Sprintf("untagged: %.1f%%", untaggedRate*pctMultiplier)
		}
	}

	// (c) hub trigger
	for term, count := range memberCounts {
		if totalNotes > 0 && float64(count)/float64(totalNotes) > hubThreshold {
			return true, fmt.Sprintf("hub: %s (%.0f%%)",
				term, float64(count)/float64(totalNotes)*pctMultiplier)
		}
	}

	return false, ""
}
