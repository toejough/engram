package cli

import (
	"path/filepath"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/update"
	"github.com/toejough/engram/internal/vaultgraph"
)

// unexported constants.
const (
	// pendingOfferCurateCommand is the trigger query every automatic cue
	// carries. Its --text holds the curate runbook's trigger words ("curate",
	// "pending offers"), so the runbook surfaces first with trigger
	// provenance; the --phrase is the process-shaped semantic fallback. A
	// runbook has no harness-read description the way a skill did, so each
	// cue itself has to say what to run (curate-skill-to-runbook D4, D9).
	pendingOfferCurateCommand = "engram query --text \"curate pending offers\" --phrase \"reviewing pending " +
		"offers in a vault and judging each against existing notes\""
	// pendingOfferCurateInstruction is the one shared instruction: the query
	// payload's pending_offers_hint, the update notice and the write nudge
	// all embed this exact string, so they can never drift apart.
	// It tells the agent curation is expected upkeep, not an extra: an agent
	// seeing the cue mid-unrelated-work declined 9/9 as "you didn't ask for
	// that" (curate-skill-to-runbook D9, D11).
	pendingOfferCurateInstruction = "curating pending offers is expected vault upkeep, not an extra: after you " +
		"finish the user's request, curate them without asking — run `" + pendingOfferCurateCommand +
		"` and follow the curate runbook it returns"
	// pendingOfferUpdateNotice is the `engram update` detect-and-notify line
	// (ADR-0021 convention) for pending offers. Unlike the other notices, it
	// names no CLI fix command — curation is a runbook (vault-offer-curation),
	// not something `engram update` can do on the user's behalf — so it names
	// the query that surfaces the curate runbook.
	pendingOfferUpdateNotice = "vault holds pending offer(s) awaiting curation — see the pending_offers " +
		"flag in `engram query`'s payload; " + pendingOfferCurateInstruction + "\n"
	// pendingOfferWriteNudge is the write-path log-only nudge (task 6.4):
	// fired at the same call sites checkAndPersistVocabRefitTrigger already
	// runs from, but never persists anything — detection stays stateless.
	pendingOfferWriteNudge = "vault holds pending offer(s) awaiting curation — " +
		pendingOfferCurateInstruction
)

// excludePendingOffers filters notes down to those NOT carrying the
// pending-offer marker (vault-offer-curation), so served writes awaiting
// curation never surface in normal query results. Returns the filtered
// slice and whether at least one pending offer was found — computed in one
// pass (single read per note) so query pays no extra I/O for the
// pending_offers payload flag beyond the exclusion scan it already needs.
// A note that fails to read or parse is kept (fail-open: never silently
// drop a note query would otherwise return over an unrelated read glitch).
func excludePendingOffers(
	notes []vaultgraph.Note,
	vaultPath string,
	read func(string) ([]byte, error),
) ([]vaultgraph.Note, bool) {
	kept := make([]vaultgraph.Note, 0, len(notes))
	anyPending := false

	for _, note := range notes {
		raw, readErr := read(filepath.Join(vaultPath, pathOf(note.Basename)))
		if readErr != nil {
			kept = append(kept, note)

			continue
		}

		if noteHasPendingMarker(raw) {
			anyPending = true

			continue
		}

		kept = append(kept, note)
	}

	return kept, anyPending
}

// noteHasPendingMarker reports whether raw is a pending offer: a fact or
// feedback note carrying frontmatter `pending: true`, or a runbook note
// carrying both `pending: true` and a non-empty `skill_hash` (a pending
// skill-registration offer — vault-offer-curation, skill-runbook-registration).
// A runbook note with `pending: true` but no `skill_hash` is NOT a pending
// offer — unchanged behavior, so a stray pending flag on an ordinary
// hand-edited runbook doesn't vanish from query results. Any other note type
// (e.g. vocab definitions) and unparseable content report false.
func noteHasPendingMarker(raw []byte) bool {
	frontmatter, ok := splitFrontmatter(raw)
	if !ok {
		return false
	}

	noteType := peekNoteType(frontmatter)
	if noteType != typeFact && noteType != typeFeedback && noteType != typeRunbook {
		return false
	}

	var probe struct {
		Pending   bool   `yaml:"pending"`
		SkillHash string `yaml:"skill_hash"`
	}

	if yaml.Unmarshal(frontmatter, &probe) != nil || !probe.Pending {
		return false
	}

	if noteType == typeRunbook {
		return probe.SkillHash != ""
	}

	return true
}

// notesHavePendingOfferByName scans names-in-hand (a vault ListMD result)
// for the pending-offer marker, mirroring countTriggerVaultNotesFromNames'
// shape so warnIfPendingOffers reuses the same ListMD/readFile deps the
// vocab-refit trigger check already requires at its call sites.
func notesHavePendingOfferByName(vault string, names []string, readFile func(string) ([]byte, error)) bool {
	for _, name := range names {
		raw, readErr := readFile(filepath.Join(vault, name))
		if readErr != nil {
			continue
		}

		if noteHasPendingMarker(raw) {
			return true
		}
	}

	return false
}

// pendingOffersHint returns the query payload's pending_offers_hint value:
// the shared curate instruction when offers are pending, empty (omitted from
// the payload) otherwise. The hint is derived from the flag, never
// transported, so a merged or served payload cannot carry a stale or
// drifted copy.
func pendingOffersHint(pending bool) string {
	if pending {
		return pendingOfferCurateInstruction
	}

	return ""
}

// vaultHasPendingOffers reports whether vaultPath holds at least one
// fact/feedback note carrying the pending-offer marker — the signal that a
// served learn/amend write is awaiting curation. Stateless and unbatched (a
// fresh scan every call, nothing persisted), deliberately not the
// vocab-refit trigger's stateful/batched shape (design.md Decisions):
// offers are lower-volume and costlier to leave silently stale. A
// missing/unreadable vault directory, or a note this can't parse, is
// treated as no-signal (self-silencing, same convention as
// notesMissingIdentityFields) — a detection failure must never fail
// `engram update`'s primary job.
func vaultHasPendingOffers(vaultPath string, fileSystem update.Filesystem) bool {
	entries, readErr := fileSystem.ReadDir(vaultPath)
	if readErr != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		raw, fileErr := fileSystem.ReadFile(filepath.Join(vaultPath, entry.Name()))
		if fileErr != nil {
			continue
		}

		if noteHasPendingMarker(raw) {
			return true
		}
	}

	return false
}

// warnIfPendingOffers logs a stateless nudge when the vault holds at least
// one pending-offer note, from the same write-path call sites
// checkAndPersistVocabRefitTrigger already runs from (applyVocabAssignment
// After{Learn,Amend,Resituate}). Log-only: unlike the refit trigger, no
// state is persisted from this call site (design.md Decisions — detection
// stays stateless and unbatched). Silent no-op when any dep is nil.
func warnIfPendingOffers(
	vault string,
	listMD func(string) ([]string, error),
	readFile func(string) ([]byte, error),
	logWarn func(string, ...any),
) {
	if listMD == nil || readFile == nil || logWarn == nil {
		return
	}

	names, listErr := listMD(vault)
	if listErr != nil {
		return
	}

	if notesHavePendingOfferByName(vault, names, readFile) {
		logWarn("%s", pendingOfferWriteNudge)
	}
}
