package cli

import (
	"path/filepath"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/update"
	"github.com/toejough/engram/internal/vaultgraph"
)

// unexported constants.
const (
	// pendingOfferUpdateNotice is the `engram update` detect-and-notify line
	// (ADR-0021 convention) for pending offers. Unlike the other notices, it
	// names no CLI fix command — curation is a skill (vault-offer-curation),
	// not something `engram update` can do on the user's behalf.
	pendingOfferUpdateNotice = "vault holds pending offer(s) awaiting curation — see the pending_offers " +
		"flag in `engram query`'s payload; curated by the offer-curation skill, not a CLI command\n"
	// pendingOfferWriteNudge is the write-path log-only nudge (task 6.4):
	// fired at the same call sites checkAndPersistVocabRefitTrigger already
	// runs from, but never persists anything — detection stays stateless.
	pendingOfferWriteNudge = "vault holds pending offer(s) awaiting curation"
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

// noteHasPendingMarker reports whether raw is a fact/feedback note carrying
// the pending-offer marker (frontmatter `pending: true`). Non-fact/feedback
// notes (e.g. vocab definitions) and unparseable content report false.
func noteHasPendingMarker(raw []byte) bool {
	frontmatter, ok := splitFrontmatter(raw)
	if !ok {
		return false
	}

	noteType := peekNoteType(frontmatter)
	if noteType != typeFact && noteType != typeFeedback {
		return false
	}

	var probe struct {
		Pending bool `yaml:"pending"`
	}

	return yaml.Unmarshal(frontmatter, &probe) == nil && probe.Pending
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
