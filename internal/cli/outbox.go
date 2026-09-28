package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"slices"
	"time"
)

// unexported constants.
const (
	learnRoute          = "/learn"
	outboxFileName      = "outbox.json"
	outboxStateQueued   = "queued"
	outboxStateRejected = "rejected"
	outboxVersion       = 1
)

// offerOutcome classifies one send.
type offerOutcome int

// offerOutcome values.
const (
	// offerAccepted: a 2xx with a full receipt.
	offerAccepted offerOutcome = iota + 1
	// offerRejected: a 4xx — mark the entry rejected, continue.
	offerRejected
	// offerFailed: a transport error, timeout, 5xx or unreadable 2xx — stop
	// the drain and back off.
	offerFailed
	// offerTooOld: a receipt without vault_id or basename (H7) — stop
	// exchange with errParentTooOld, keep the entry queued.
	offerTooOld
	// offerSkipped: the payload could not be built locally — the parent was
	// not contacted; record the error, keep the entry, continue.
	offerSkipped
)

// unexported variables.
var (
	errOfferRejected    = errors.New("the parent rejected the offer")
	errOfferServerError = errors.New("the parent failed the offer")
	errOfferUndecodable = errors.New("the parent's offer receipt does not decode")
	errParentTooOld     = errors.New("the parent is too old: its offer receipt carries no vault_id or basename " +
		"(upgrade engram on the parent host)")
)

// drainResult summarizes one drain for its caller: how many offers were
// accepted, rejected and dropped, and whether the parent was unreachable
// (with the retry time the backoff set).
type drainResult struct {
	Sent        int
	Rejected    int
	Dropped     int
	Unreachable bool
	RetryAfter  time.Time
}

// drainStep is one entry's outcome from the send phase, merged under the
// lock afterwards.
type drainStep struct {
	xid     string
	queued  time.Time
	drop    bool
	hash    string
	outcome offerSendResult
}

// offerNote is a queued note as the drain found it at send time (by xid,
// so a rename never orphans it): its current name, path, raw content and
// exchange hash. The payload builder works from this.
type offerNote struct {
	XID      string
	Basename string
	Path     string
	Raw      []byte
	Hash     string
}

// offerSendResult is one send's outcome: the receipt for an accepted offer,
// the cause otherwise.
type offerSendResult struct {
	Outcome offerOutcome
	Receipt offerReceipt
	Err     error
}

// outboxEntry is one queued note, keyed by its xid (design D6). Queued is
// the first-queued time and fixes its FIFO slot; State is queued or
// rejected; RejectedHash is the exchange hash the parent rejected.
type outboxEntry struct {
	XID          string    `json:"xid"`
	Queued       time.Time `json:"queued"`
	Attempts     int       `json:"attempts"`
	LastError    string    `json:"last_error,omitempty"` //nolint:tagliatelle // design D6 fixes outbox.json's keys
	State        string    `json:"state"`
	RejectedHash string    `json:"rejected_hash,omitempty"` //nolint:tagliatelle // design D6 fixes outbox.json's keys
}

// outboxFile is <vault>/.engram/outbox.json.
type outboxFile struct {
	Version int           `json:"version"`
	Entries []outboxEntry `json:"entries"`
}

// outboxStore is the outbox and parent-cache adapter: the exchange-state
// filesystem, the vault lock, the vault listing, the clock and the warning
// sink, all injected.
type outboxStore struct {
	state  exchangeState
	lock   func(vault string) (func(), error)
	listMD func(dir string) ([]string, error)
	now    func() time.Time
	stderr io.Writer
}

// scannedOffer is a vault note that carries an xid, as the drain sees it.
type scannedOffer struct {
	note    offerNote
	pending bool
}

// applyDrainStep folds one send outcome into the re-read outbox and the
// parent cache. Under the lock: a note that is gone or now pending drops
// its entry (and any receipt); an accepted offer records its receipt and
// stays queued only when the note changed since it was sent.
func applyDrainStep(
	step drainStep, box outboxFile, cache *parentCache, current map[string]scannedOffer,
	apply func(offerNote, offerReceipt) error, now time.Time, parentURL string, result *drainResult,
) (outboxFile, error) {
	found, ok := current[step.xid]
	if !ok || found.pending {
		result.Dropped++

		return removeOutboxEntry(box, step.xid), nil
	}

	if step.drop {
		return box, nil // live again since the snapshot: keep it
	}

	switch step.outcome.Outcome {
	case offerAccepted:
		*cache = noteParentSuccess(*cache, parentURL, step.outcome.Receipt.VaultID)
		result.Sent++

		applyErr := apply(found.note, step.outcome.Receipt)
		if applyErr != nil {
			return updateOutboxEntry(box, step, func(entry *outboxEntry) {
				entry.Attempts++
				entry.LastError = applyErr.Error()
			}), fmt.Errorf("outbox: recording the receipt on %s: %w", found.note.Basename, applyErr)
		}

		if exchangeHashChanged(step.hash, found.note.Hash) {
			return keepOutboxEntryQueued(box, step), nil
		}

		return removeOutboxEntry(box, step.xid), nil
	case offerRejected:
		*cache = noteParentSuccess(*cache, parentURL, "")
		result.Rejected++

		return updateOutboxEntry(box, step, func(entry *outboxEntry) {
			entry.Attempts++
			entry.State = outboxStateRejected
			entry.RejectedHash = step.hash
			entry.LastError = errorText(step.outcome.Err)
		}), nil
	case offerFailed:
		*cache = noteParentFailure(*cache, parentURL, now)
		result.Unreachable, result.RetryAfter = true, cache.BackoffUntil
	case offerTooOld, offerSkipped:
		// The parent was not usefully contacted: record the error only.
	}

	return updateOutboxEntry(box, step, func(entry *outboxEntry) {
		entry.Attempts++
		entry.LastError = errorText(step.outcome.Err)
	}), nil
}

// classifyOfferResponse maps a parent /learn response to the drain's
// outcome (design D6, H7).
func classifyOfferResponse(resp FetchResponse, fetchErr error) offerSendResult {
	if fetchErr != nil {
		return offerSendResult{Outcome: offerFailed, Err: fmt.Errorf("%w: %w", errParentUnreachable, fetchErr)}
	}

	switch {
	case resp.Status >= statusInternalServerError:
		return offerSendResult{Outcome: offerFailed, Err: fmt.Errorf("%w (%w): status %d: %s",
			errOfferServerError, errParentUnreachable, resp.Status, describeErrorBody(resp.Body))}
	case resp.Status >= statusBadRequest:
		return offerSendResult{Outcome: offerRejected, Err: fmt.Errorf("%w: status %d: %s",
			errOfferRejected, resp.Status, describeErrorBody(resp.Body))}
	case resp.Status < statusOK || resp.Status >= httpStatusMultipleChoices:
		return offerSendResult{Outcome: offerFailed, Err: fmt.Errorf("%w: unexpected status %d",
			errOfferServerError, resp.Status)}
	}

	var receipt offerReceipt

	decodeErr := json.Unmarshal(resp.Body, &receipt)
	if decodeErr != nil {
		return offerSendResult{Outcome: offerFailed, Err: fmt.Errorf("%w: %w", errOfferUndecodable, decodeErr)}
	}

	if receipt.VaultID == "" || receipt.Basename == "" {
		return offerSendResult{Outcome: offerTooOld, Receipt: receipt, Err: errParentTooOld}
	}

	return offerSendResult{Outcome: offerAccepted, Receipt: receipt}
}

// clearedForExchange runs the checks a child does before any write
// exchange (design D2, ruling S2): the location check, then the
// self-parent guard against the parent's cached vault ID. Each failure
// prints its one warning.
func clearedForExchange(store outboxStore, vault, parentURL string) (string, bool, error) {
	if !warnVaultLocation(store.state, vault, store.stderr) {
		return "", false, nil
	}

	localID, _, idErr := readVaultID(store.state, vault)
	if idErr != nil {
		return "", false, idErr
	}

	cache, cacheErr := loadParentCache(store.state, vault)
	if cacheErr != nil {
		return "", false, cacheErr
	}

	if cache.URL == parentURL && selfParentGuard(localID, cache.VaultID, store.stderr) {
		return "", false, nil
	}

	return localID, true, nil
}

// drainOutbox sends the queued offers to the parent (design D6): it reads a
// snapshot under the vault lock, sends each entry in first-queued order
// with no lock held (building the payload from the note's current content,
// found by xid), then re-takes the lock, re-reads the outbox and the notes
// from disk, and merges — so entries queued meanwhile survive and a receipt
// for a note deleted meanwhile is discarded (M10). The caller has already
// contacted the parent (or is `engram update`); the drain itself checks the
// vault's location and the self-parent guard first (ruling S2).
func drainOutbox(
	ctx context.Context, store outboxStore, vault, parentURL string,
	send func(context.Context, offerNote) offerSendResult, apply func(offerNote, offerReceipt) error,
) (drainResult, error) {
	snapshot, snapErr := lockedLoadOutbox(store, vault)
	if snapErr != nil || len(snapshot.Entries) == 0 {
		return drainResult{}, snapErr
	}

	localID, cleared, clearErr := clearedForExchange(store, vault, parentURL)
	if clearErr != nil || !cleared {
		return drainResult{}, clearErr
	}

	notes, scanErr := scanOfferNotes(store, vault)
	if scanErr != nil {
		return drainResult{}, scanErr
	}

	steps, stopErr := sendOutboxEntries(ctx, snapshot, notes, localID, store.stderr, send)
	if len(steps) == 0 {
		return drainResult{}, stopErr
	}

	result, mergeErr := mergeDrainSteps(store, vault, parentURL, steps, apply)

	return result, errors.Join(stopErr, mergeErr)
}

// enqueueOutbox adds the note's entry (keyed by xid) or leaves its existing
// one — at most one entry per note, keeping its first-queued slot — and
// saves the outbox atomically. The caller holds the vault lock, in the same
// critical section as the note write (design D6).
func enqueueOutbox(store outboxStore, vault, xid string) error {
	box, loadErr := loadOutbox(store.state, vault)
	if loadErr != nil {
		return loadErr
	}

	return saveOutbox(store.state, vault, outboxUpsert(box, xid, store.now()))
}

// errorText is err's message, or "" for nil.
func errorText(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}

// fifoEntries returns the entries oldest-queued first (stable for ties).
func fifoEntries(entries []outboxEntry) []outboxEntry {
	ordered := slices.Clone(entries)
	slices.SortStableFunc(ordered, func(first, second outboxEntry) int {
		return first.Queued.Compare(second.Queued)
	})

	return ordered
}

// judgeSendOutcome decides what the send phase does with one outcome:
// whether to record it for the merge, and whether the drain stops (an
// unreachable parent, a too-old parent, or a receipt naming this vault).
func judgeSendOutcome(outcome offerSendResult, localID string, stderr io.Writer) (bool, bool, error) {
	switch outcome.Outcome {
	case offerTooOld:
		return false, true, fmt.Errorf("outbox: %w", errParentTooOld)
	case offerAccepted:
		if selfParentGuard(localID, outcome.Receipt.VaultID, stderr) {
			return false, true, nil
		}
	case offerFailed:
		return true, true, nil
	case offerRejected, offerSkipped:
	}

	return true, false, nil
}

// keepOutboxEntryQueued keeps an accepted-but-since-changed note queued: its
// entry (re-added if it vanished) goes back to queued with no rejection.
func keepOutboxEntryQueued(box outboxFile, step drainStep) outboxFile {
	if !slices.ContainsFunc(box.Entries, func(entry outboxEntry) bool { return entry.XID == step.xid }) {
		box.Entries = append(box.Entries, outboxEntry{XID: step.xid, Queued: step.queued, State: outboxStateQueued})
	}

	return updateOutboxEntry(box, step, func(entry *outboxEntry) {
		entry.State = outboxStateQueued
		entry.RejectedHash = ""
		entry.LastError = ""
	})
}

// loadOutbox reads .engram/outbox.json; a missing file is an empty outbox.
func loadOutbox(state exchangeState, vault string) (outboxFile, error) {
	raw, readErr := state.fs.ReadFile(outboxPath(vault))
	if errors.Is(readErr, fs.ErrNotExist) {
		return outboxFile{Version: outboxVersion}, nil
	}

	if readErr != nil {
		return outboxFile{}, fmt.Errorf("outbox: %w", readErr)
	}

	var box outboxFile

	unmarshalErr := json.Unmarshal(raw, &box)
	if unmarshalErr != nil {
		return outboxFile{}, fmt.Errorf("outbox: %s: %w", outboxFileName, unmarshalErr)
	}

	box.Version = outboxVersion

	return box, nil
}

// lockedLoadOutbox reads the outbox snapshot under the vault lock.
func lockedLoadOutbox(store outboxStore, vault string) (outboxFile, error) {
	unlock, lockErr := store.lock(vault)
	if lockErr != nil {
		return outboxFile{}, fmt.Errorf("outbox: %w", lockErr)
	}

	defer unlock()

	return loadOutbox(store.state, vault)
}

// mergeDrainSteps is the drain's locked merge: re-read the outbox and the
// notes from disk, fold in every send outcome, and save the outbox and the
// parent cache.
func mergeDrainSteps(
	store outboxStore, vault, parentURL string, steps []drainStep, apply func(offerNote, offerReceipt) error,
) (drainResult, error) {
	unlock, lockErr := store.lock(vault)
	if lockErr != nil {
		return drainResult{}, fmt.Errorf("outbox: %w", lockErr)
	}

	defer unlock()

	box, loadErr := loadOutbox(store.state, vault)
	if loadErr != nil {
		return drainResult{}, loadErr
	}

	cache, cacheErr := loadParentCache(store.state, vault)
	if cacheErr != nil {
		return drainResult{}, cacheErr
	}

	current, scanErr := scanOfferNotes(store, vault)
	if scanErr != nil {
		return drainResult{}, scanErr
	}

	var (
		result    drainResult
		applyErrs []error
	)

	originalCache := cache

	for _, step := range steps {
		var stepErr error

		box, stepErr = applyDrainStep(step, box, &cache, current, apply, store.now(), parentURL, &result)
		applyErrs = append(applyErrs, stepErr)
	}

	saveErr := saveOutbox(store.state, vault, box)
	if saveErr != nil {
		return result, saveErr
	}

	if cache != originalCache {
		cacheSaveErr := saveParentCache(store.state, vault, cache)
		if cacheSaveErr != nil {
			return result, cacheSaveErr
		}
	}

	return result, errors.Join(applyErrs...)
}

// newOfferSender is the production send step: build the payload from the
// note as it is now, POST it to the parent's /learn route, and classify
// the response. A payload that cannot be built is skipped without
// contacting the parent.
func newOfferSender(
	fetch func(ctx context.Context, method, url string, body []byte) (FetchResponse, error),
	base string,
	buildPayload func(offerNote) ([]byte, error),
) func(context.Context, offerNote) offerSendResult {
	return func(ctx context.Context, note offerNote) offerSendResult {
		body, buildErr := buildPayload(note)
		if buildErr != nil {
			return offerSendResult{
				Outcome: offerSkipped,
				Err:     fmt.Errorf("outbox: building the offer for %s: %w", note.Basename, buildErr),
			}
		}

		resp, fetchErr := fetch(ctx, methodPost, buildURL(base, learnRoute, nil), body)

		return classifyOfferResponse(resp, fetchErr)
	}
}

func newOutboxStore(
	state exchangeState,
	lock func(vault string) (func(), error),
	listMD func(dir string) ([]string, error),
	now func() time.Time,
	stderr io.Writer,
) outboxStore {
	return outboxStore{state: state, lock: lock, listMD: listMD, now: now, stderr: stderr}
}

// outboxPath is <vault>/.engram/outbox.json.
func outboxPath(vault string) string {
	return filepath.Join(vault, stateDirName, outboxFileName)
}

// outboxStoreFromDeps composes the store over the production carrier.
func outboxStoreFromDeps(deps Deps) outboxStore {
	return newOutboxStore(
		exchangeStateFromDeps(deps), vaultLockFromLocker(deps.Lock), listMDFromFS(deps.FS), deps.Now, deps.Stderr)
}

// outboxUpsert adds a queued entry for xid unless the note already has one
// (which keeps its first-queued slot; a rejected entry re-arms only when
// its note's exchange hash changes, which the drain checks).
func outboxUpsert(box outboxFile, xid string, now time.Time) outboxFile {
	if slices.ContainsFunc(box.Entries, func(entry outboxEntry) bool { return entry.XID == xid }) {
		return box
	}

	box.Entries = append(slices.Clone(box.Entries), outboxEntry{XID: xid, Queued: now, State: outboxStateQueued})

	return box
}

// queuedOfferCount counts the entries still waiting to be sent (rejected
// entries are not counted).
func queuedOfferCount(box outboxFile) int {
	count := 0

	for _, entry := range box.Entries {
		if entry.State != outboxStateRejected {
			count++
		}
	}

	return count
}

// removeOutboxEntry drops xid's entry.
func removeOutboxEntry(box outboxFile, xid string) outboxFile {
	box.Entries = slices.DeleteFunc(slices.Clone(box.Entries), func(entry outboxEntry) bool {
		return entry.XID == xid
	})

	return box
}

// saveOutbox writes the outbox atomically. The state directory is created
// only after the vault ID is stamped (ruling S4).
func saveOutbox(state exchangeState, vault string, box outboxFile) error {
	dirErr := ensureExchangeStateDir(state, vault)
	if dirErr != nil {
		return dirErr
	}

	box.Version = outboxVersion
	if box.Entries == nil {
		box.Entries = []outboxEntry{}
	}

	encoded, marshalErr := json.MarshalIndent(box, "", "  ")
	if marshalErr != nil {
		return fmt.Errorf("outbox: %w", marshalErr)
	}

	writeErr := state.fs.WriteFileAtomic(outboxPath(vault), append(encoded, '\n'), vaultFilePerm)
	if writeErr != nil {
		return fmt.Errorf("outbox: %w", writeErr)
	}

	return nil
}

// scanOfferNotes finds every fact, feedback and runbook note that carries
// an xid (a frontmatter scan, so renames are safe), keyed by xid.
func scanOfferNotes(store outboxStore, vault string) (map[string]scannedOffer, error) {
	notes, scanErr := scanExchangeNotes(vault, store.listMD, store.state.fs.ReadFile)
	if scanErr != nil {
		return nil, fmt.Errorf("outbox: %w", scanErr)
	}

	byXID := make(map[string]scannedOffer, len(notes))

	for _, note := range notes {
		if note.exchange.XID == "" {
			continue
		}

		hash, hashErr := exchangeHash(note.raw)
		if hashErr != nil {
			return nil, fmt.Errorf("outbox: %s: %w", note.basename, hashErr)
		}

		byXID[note.exchange.XID] = scannedOffer{
			note: offerNote{
				XID: note.exchange.XID, Basename: note.basename, Path: note.path, Raw: note.raw, Hash: hash,
			},
			pending: note.pending,
		}
	}

	return byXID, nil
}

// sendOutboxEntries is the drain's unlocked send phase: each entry in
// first-queued order, dropping entries whose note is gone or pending and
// skipping rejected entries whose note has not changed. It stops at an
// unreachable parent, a too-old parent, or a receipt that names this
// vault itself (the self-parent guard).
func sendOutboxEntries(
	ctx context.Context, snapshot outboxFile, notes map[string]scannedOffer, localID string, stderr io.Writer,
	send func(context.Context, offerNote) offerSendResult,
) ([]drainStep, error) {
	steps := make([]drainStep, 0, len(snapshot.Entries))

	for _, entry := range fifoEntries(snapshot.Entries) {
		found, ok := notes[entry.XID]
		if !ok || found.pending {
			steps = append(steps, drainStep{xid: entry.XID, queued: entry.Queued, drop: true})

			continue
		}

		if entry.State == outboxStateRejected && !exchangeHashChanged(entry.RejectedHash, found.note.Hash) {
			continue
		}

		outcome := send(ctx, found.note)

		record, stop, stopErr := judgeSendOutcome(outcome, localID, stderr)
		if record {
			steps = append(steps, drainStep{xid: entry.XID, queued: entry.Queued, hash: found.note.Hash, outcome: outcome})
		}

		if stop {
			return steps, stopErr
		}
	}

	return steps, nil
}

// stampNoteXID returns the note's xid, minting one when it has none (the
// lazy stamp the first time a note enters the outbox; there is no
// backfill). stamped reports whether a new xid was minted.
func stampNoteXID(current string, randRead func([]byte) (int, error)) (string, bool, error) {
	if current != "" {
		return current, false, nil
	}

	minted, mintErr := mintXID(randRead)
	if mintErr != nil {
		return "", false, mintErr
	}

	return minted, true, nil
}

// updateOutboxEntry applies change to xid's entry when the re-read outbox
// still has it (an entry another drain already cleared is not revived).
func updateOutboxEntry(box outboxFile, step drainStep, change func(*outboxEntry)) outboxFile {
	entries := slices.Clone(box.Entries)

	for index := range entries {
		if entries[index].XID == step.xid {
			change(&entries[index])
		}
	}

	box.Entries = entries

	return box
}
