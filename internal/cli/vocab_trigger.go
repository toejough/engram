package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"go.yaml.in/yaml/v3"
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

// checkAndPersistVocabRefitTrigger evaluates vault trigger state and updates
// vocab.centroids.json. On first call (no last_refit): seeds the baseline and
// returns without firing. On subsequent calls: evaluates and persists when triggered.
// Silent no-op when any dep is nil or when centroids file is absent.
func checkAndPersistVocabRefitTrigger(
	vault string,
	listMD func(string) ([]string, error),
	readFile func(string) ([]byte, error),
	writeFile func(string, []byte) error,
	logWarn func(string, ...any),
	now time.Time,
) {
	if listMD == nil || readFile == nil || writeFile == nil {
		return
	}

	doc, _ := readCentroidsDoc(vault, readFile) // zero-value on missing file

	if doc.RefitPending {
		return // already flagged — idempotent; no vault scan needed
	}

	totalNotes, untaggedCount, memberCounts := collectTriggerVaultStats(vault, listMD, readFile)

	if doc.LastRefit == nil {
		// Seed baseline — no trigger fires this call.
		doc.LastRefit = &vocabLastRefitDoc{
			NoteCount: totalNotes,
			Date:      now.Format(dateFormat),
		}

		writeWithWarn(vault, doc, writeFile, logWarn, "seeding last_refit")

		return
	}

	fired, reason := evaluateVocabTriggers(totalNotes, untaggedCount, memberCounts, doc.LastRefit, now)
	if !fired {
		return
	}

	doc.RefitPending = true
	doc.RefitReason = reason

	writeWithWarn(vault, doc, writeFile, logWarn, "persisting refit_pending")
}

// collectTriggerVaultStats scans non-vocab note frontmatter for the trigger evaluation.
// Returns (totalNotes, untaggedCount, perTermMemberCounts).
// Unreadable or unparseable notes count as total but not tagged.
func collectTriggerVaultStats(
	vault string,
	listMD func(string) ([]string, error),
	readFile func(string) ([]byte, error),
) (int, int, map[string]int) {
	names, listErr := listMD(vault)
	if listErr != nil {
		return 0, 0, nil
	}

	return collectTriggerVaultStatsFromNames(vault, names, readFile)
}

// collectTriggerVaultStatsFromNames is the names-in-hand form of
// collectTriggerVaultStats, for callers that already listed the vault
// (e.g. emitRefitRequest) — avoids a second directory pass.
func collectTriggerVaultStatsFromNames(
	vault string,
	names []string,
	readFile func(string) ([]byte, error),
) (int, int, map[string]int) {
	memberCounts := make(map[string]int)
	totalNotes, untaggedCount := 0, 0

	scanNonVocabNotes(vault, names, readFile, func(_ string, raw []byte, readErr error) {
		totalNotes++

		if readErr != nil {
			untaggedCount++
			return
		}

		frontmatterBytes, ok := splitFrontmatter(raw)
		if !ok {
			untaggedCount++
			return
		}

		var doc noteMiniDoc

		if yaml.Unmarshal(frontmatterBytes, &doc) != nil || len(doc.Vocab) == 0 {
			untaggedCount++
			return
		}

		for _, term := range doc.Vocab {
			memberCounts[term]++
		}
	})

	return totalNotes, untaggedCount, memberCounts
}

// countNonVocabNoteFiles counts basenames that are not vocab-kind files.
// A pure helper reused by bootstrap/refit seeding and the trigger check.
func countNonVocabNoteFiles(names []string) int {
	count := 0

	for _, name := range names {
		if !isVocabKindFilename(name) {
			count++
		}
	}

	return count
}

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

// scanNonVocabNotes calls visit for each non-vocab filename in names.
// visit receives (name, raw bytes, readErr); raw is nil when readErr is non-nil.
// Shared primitive used by collectTriggerVaultStats (untaggedCount) and
// countMembersFromNotes (vocab_commands.go) to avoid duplicating the loop.
func scanNonVocabNotes(
	vault string,
	names []string,
	readFile func(string) ([]byte, error),
	visit func(name string, raw []byte, readErr error),
) {
	for _, name := range names {
		if isVocabKindFilename(name) {
			continue
		}

		raw, readErr := readFile(filepath.Join(vault, name))
		visit(name, raw, readErr)
	}
}

// writeCentroidsDocRaw marshals doc and writes it to vocab.centroids.json.
// Preserves all existing fields (terms, trigger state) in a single write.
func writeCentroidsDocRaw(vault string, doc vocabCentroidsDoc, writeFile func(string, []byte) error) error {
	data, marshalErr := json.Marshal(doc)
	if marshalErr != nil {
		return fmt.Errorf("marshaling centroids: %w", marshalErr)
	}

	return writeFile(filepath.Join(vault, vocabCentroidsFilename), data)
}

// writeWithWarn writes doc to the centroids file, logging any error via logWarn.
func writeWithWarn(
	vault string,
	doc vocabCentroidsDoc,
	writeFile func(string, []byte) error,
	logWarn func(string, ...any),
	operation string,
) {
	err := writeCentroidsDocRaw(vault, doc, writeFile)

	if err != nil && logWarn != nil {
		logWarn("vocab trigger: %s: %v", operation, err)
	}
}
