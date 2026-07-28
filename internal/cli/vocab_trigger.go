package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
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
	// refitUntaggedRateMax is the vault-wide untagged rate above which
	// `vocab stats` flags the rate as a [high] diagnostic (exclusive: >8%).
	// Diagnostic only — it does not set refit_pending (growth is the sole trigger).
	refitUntaggedRateMax = 0.08
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

	totalNotes := countTriggerVaultNotes(vault, listMD, readFile)

	if doc.LastRefit == nil {
		// Seed baseline — no trigger fires this call.
		doc.LastRefit = &vocabLastRefitDoc{
			NoteCount: totalNotes,
			Date:      now.Format(dateFormat),
		}

		writeWithWarn(vault, doc, writeFile, logWarn, "seeding last_refit")

		return
	}

	fired, reason := evaluateVocabTriggers(totalNotes, doc.LastRefit, now)
	if !fired {
		return
	}

	doc.RefitPending = true
	doc.RefitReason = reason

	writeWithWarn(vault, doc, writeFile, logWarn, "persisting refit_pending")
}

// countTriggerVaultNotes counts the vault's non-definition notes for the
// growth-trigger evaluation. Returns 0 when listing fails.
func countTriggerVaultNotes(
	vault string,
	listMD func(string) ([]string, error),
	readFile func(string) ([]byte, error),
) int {
	names, listErr := listMD(vault)
	if listErr != nil {
		return 0
	}

	return countTriggerVaultNotesFromNames(vault, names, readFile)
}

// countTriggerVaultNotesFromNames is the names-in-hand form of
// countTriggerVaultNotes, for callers that already listed the vault
// (e.g. buildLastRefitDoc) — avoids a second directory pass. A bare-vocab
// DEFINITION note (isVocabDefinitionNote) does not count — it is not a member
// note at all; unreadable notes DO count. This is the ONE content-based
// note-count measure shared by both the refit-baseline seed (buildLastRefitDoc)
// and the trigger check itself (checkAndPersistVocabRefitTrigger) — they must
// never diverge in units.
func countTriggerVaultNotesFromNames(
	vault string,
	names []string,
	readFile func(string) ([]byte, error),
) int {
	totalNotes := 0

	scanNonVocabNotes(vault, names, readFile, func(_ string, raw []byte, readErr error) {
		if readErr == nil && isVocabDefinitionNote(string(raw)) {
			return // a definition note is not a member note
		}

		totalNotes++
	})

	return totalNotes
}

// evaluateVocabTriggers returns (fired, reason) for the in-process threshold check.
// Growth is the SOLE trigger (vocab-derivational-refit): ≥refitGrowthMinNotes new
// notes since last_refit AND ≥refitGrowthMinDays elapsed. Untagged rate and hub
// concentration are `vocab stats` diagnostics only and never set refit_pending.
// Returns (false, "") when lastRefit is nil (no baseline yet — caller seeds and returns).
func evaluateVocabTriggers(totalNotes int, lastRefit *vocabLastRefitDoc, now time.Time) (bool, string) {
	if lastRefit == nil {
		return false, "" // no baseline — caller seeds and returns
	}

	lastRefitDate, parseErr := time.Parse(dateFormat, lastRefit.Date)
	if parseErr != nil {
		return false, ""
	}

	growth := totalNotes - lastRefit.NoteCount
	daysSince := int(now.Sub(lastRefitDate).Hours() / hoursPerDay)

	if growth >= refitGrowthMinNotes && daysSince >= refitGrowthMinDays {
		return true, fmt.Sprintf("growth: %d notes, %d days", growth, daysSince)
	}

	return false, ""
}

// scanNonVocabNotes calls visit for each .md file except QA question notes.
// visit receives (name, raw bytes, readErr); raw is nil when readErr is non-nil.
// Definition-note exclusion happens content-based in callers. Shared primitive
// used by countTriggerVaultNotes.
func scanNonVocabNotes(
	vault string,
	names []string,
	readFile func(string) ([]byte, error),
	visit func(name string, raw []byte, readErr error),
) {
	for _, name := range names {
		if isQAQuestionFilename(name) {
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
