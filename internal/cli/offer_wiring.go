package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// unexported constants.
const (
	// offerDrainWarningFormat reports a drain that stopped or failed to
	// record a receipt; the local write it followed already succeeded.
	offerDrainWarningFormat = "offer: %v"
	// parentIDProbePhrase is the phrase of the vault-ID probe query; any
	// phrase works, the probe reads only the payload's vault_id.
	parentIDProbePhrase = "vault id"
)

// offerHooks connects a local write path to the parent outbox (design D5,
// D6). The zero value — no parent configured — offers nothing, stamps
// nothing and writes no exchange state.
type offerHooks struct {
	// stage runs under the vault lock with the note content about to be
	// written: it classifies the write and, when it is offered, stamps an
	// xid (lazily, D4). It returns the content to write and the xid to
	// enqueue ("" when the write is not offered).
	stage func(vault string, write offerWrite) ([]byte, string, error)
	// mintXID mints an xid for a note learn is about to create.
	mintXID func() (string, error)
	// enqueue adds the note's outbox entry, under the same lock, after the
	// write.
	enqueue func(vault, xid string) error
	// drain runs after the lock is released: the backoff gate, then the
	// drain, each printing its one warning.
	drain func(ctx context.Context, vault string)
	// warn reports a local exchange-state failure; the write it followed
	// already succeeded, so it never fails the command.
	warn func(format string, args ...any)
}

// enabled reports whether a parent is configured.
func (h offerHooks) enabled() bool { return h.stage != nil }

// queue enqueues xid after a write, under the lock; a failure is warned,
// never returned (the local write already succeeded). It reports whether
// an entry was queued.
func (h offerHooks) queue(vault, xid string) bool {
	if !h.enabled() || xid == "" {
		return false
	}

	enqueueErr := h.enqueue(vault, xid)
	if enqueueErr != nil {
		h.warn("offer: could not queue the offer: %v", enqueueErr)

		return false
	}

	return true
}

// stageWrite runs stage when a parent is configured. A staging failure is
// warned and the write proceeds unstaged and unqueued.
func (h offerHooks) stageWrite(vault string, write offerWrite) ([]byte, string) {
	if !h.enabled() {
		return write.raw, ""
	}

	content, xid, stageErr := h.stage(vault, write)
	if stageErr != nil {
		h.warn("offer: could not stage the offer: %v", stageErr)

		return write.raw, ""
	}

	return content, xid
}

// cachedParentVaultID is the parent's vault ID as last reported, when the
// cache belongs to parentURL ("" when not yet known).
func cachedParentVaultID(state exchangeState, vault, parentURL string) string {
	cache, cacheErr := loadParentCache(state, vault)
	if cacheErr != nil || cache.URL != parentURL {
		return ""
	}

	return cache.VaultID
}

// drainForCommand is a command's drain step (design D6): the backoff gate
// (which `engram update` ignores), then the drain. An unreachable parent
// prints the one "parent unreachable" warning with the queued count; any
// other drain error is warned. Neither fails the command.
func drainForCommand(ctx context.Context, deps Deps, vault, parentURL string, ignoreBackoff bool) {
	store := outboxStoreFromDeps(deps)

	// Kept receipts need no parent contact: record them before the gate
	// (#789 review finding 4).
	keptErr := recordKeptReceipts(store, vault, parentURL, newReceiptApplier(ctx, deps))
	if keptErr != nil {
		logWarningTo(deps.Stderr)(offerDrainWarningFormat, keptErr)
	}

	if !gateParentContact(store, vault, parentURL, ignoreBackoff) {
		return
	}

	if !ensureParentVaultID(ctx, deps, store, vault, parentURL) {
		return
	}

	result, drainErr := drainOutbox(ctx, store, vault, parentURL,
		newOfferSender(deps.Fetch, parentURL, newPayloadBuilder(ctx, deps, store, vault, parentURL)),
		newReceiptApplier(ctx, deps))

	if result.Unreachable {
		queued := 0

		box, boxErr := loadOutbox(store.state, vault)
		if boxErr == nil {
			queued = queuedOfferCount(box)
		}

		_, _ = fmt.Fprintf(deps.Stderr, parentBackoffWarningFormat, result.RetryAfter.Format(time.RFC3339), queued)
	}

	if drainErr != nil {
		logWarningTo(deps.Stderr)(offerDrainWarningFormat, drainErr)
	}
}

// ensureParentVaultID makes sure the configured parent's vault ID is known
// before any offer is built (ruling S16): links, loop suppression, offer.for
// and the supersedes translation all key on it, and a link recorded for
// another vault never stands in. When the cache has no ID for parentURL and
// offers are queued, the drain's first contact is a cheap `GET
// /query?dedupe-keys=1`, whose vault_id is cached. It reports whether the
// drain may proceed; each failure prints its one warning.
func ensureParentVaultID(ctx context.Context, deps Deps, store outboxStore, vault, parentURL string) bool {
	box, boxErr := loadOutbox(store.state, vault)
	if boxErr != nil || len(box.Entries) == 0 || cachedParentVaultID(store.state, vault, parentURL) != "" {
		return true // an empty or unreadable outbox is the drain's own business
	}

	parentID, fetchErr := fetchParentVaultID(ctx, deps, parentURL)
	if fetchErr != nil {
		if !errors.Is(fetchErr, errParentUnreachable) {
			_ = recordParentSuccess(store, vault, parentURL, "")
			logWarningTo(deps.Stderr)(offerDrainWarningFormat, fetchErr)

			return false
		}

		retry, _ := recordParentFailure(store, vault, parentURL)
		_, _ = fmt.Fprintf(deps.Stderr, parentBackoffWarningFormat, retry.Format(time.RFC3339), queuedOfferCount(box))

		return false
	}

	if parentID == "" {
		_ = recordParentSuccess(store, vault, parentURL, "")
		logWarningTo(deps.Stderr)(offerDrainWarningFormat, errParentTooOld)

		return false
	}

	recordErr := recordParentSuccess(store, vault, parentURL, parentID)
	if recordErr != nil {
		logWarningTo(deps.Stderr)(offerDrainWarningFormat, recordErr)

		return false
	}

	return true
}

// fetchParentVaultID asks the parent for its vault ID with a one-item
// `GET /query?dedupe-keys=1` ("" from a parent too old to report it).
func fetchParentVaultID(ctx context.Context, deps Deps, parentURL string) (string, error) {
	resp, fetchErr := fetchRaw(ctx, deps, parentURL, methodGet, "/query", map[string][]string{
		"phrase": {parentIDProbePhrase}, "limit": {"1"}, "lazy-chunks": {"true"}, "dedupe-keys": {"1"},
	}, nil)
	if fetchErr != nil {
		return "", fetchErr
	}

	var payload struct {
		VaultID string `yaml:"vault_id"`
	}

	if yaml.Unmarshal(resp.Body, &payload) != nil {
		return "", nil //nolint:nilerr // an undecodable payload is a parent too old to report its ID
	}

	return payload.VaultID, nil
}

// newOfferHooks composes the production hooks: disabled when ENGRAM_PARENT
// is unset.
func newOfferHooks(deps Deps) offerHooks {
	parentURL := parentBase(deps)
	if parentURL == "" {
		return offerHooks{}
	}

	store := outboxStoreFromDeps(deps)

	return offerHooks{
		stage: func(vault string, write offerWrite) ([]byte, string, error) {
			return stageOffer(store.state, vault, parentURL, write)
		},
		mintXID: func() (string, error) { return mintXID(deps.RandRead) },
		enqueue: func(vault, xid string) error { return enqueueOutbox(store, vault, xid) },
		drain: func(ctx context.Context, vault string) {
			drainForCommand(ctx, deps, vault, parentURL, false)
		},
		warn: logWarningTo(deps.Stderr),
	}
}

// newPayloadBuilder is the production payload step: this vault's ID and
// the parent's cached ID are read when the first offer is built, and the
// supersedes translation table is built once per drain.
func newPayloadBuilder(
	ctx context.Context, deps Deps, store outboxStore, vault, parentURL string,
) func(offerNote) ([]byte, error) {
	var (
		pctx    offerPayloadContext
		targets map[string]string
		ready   bool
	)

	return func(note offerNote) ([]byte, error) {
		if !ready {
			localID, _, idErr := readVaultID(store.state, vault)
			if idErr != nil {
				return nil, idErr
			}

			parentID := cachedParentVaultID(store.state, vault, parentURL)

			table, tableErr := supersedesTargets(store, vault, parentID)
			if tableErr != nil {
				return nil, tableErr
			}

			targets = table
			pctx = offerPayloadContext{
				localVaultID:  localID,
				parentVaultID: parentID,
				detectRepo:    func() string { return detectRepo(ctx, deps.Getwd, deps.Commander) },
				detectUser:    func() string { return detectUser(ctx, deps.Commander, deps.Username) },
				supersedesTarget: func(local string) (string, bool) {
					target, found := targets[local]

					return target, found
				},
			}
			ready = true
		}

		return buildOfferPayload(note, pctx)
	}
}

// newReceiptApplier is the production receipt step: a frontmatter-only
// rewrite of the note (applyReceiptToContent), written atomically. It
// never re-stamps identity, and re-embeds only a note the write converted
// from CRLF.
func newReceiptApplier(ctx context.Context, deps Deps) func(offerNote, offerReceipt) error {
	write := writeAtomicFromFS(deps.FS, "write note")

	return func(note offerNote, receipt offerReceipt) error {
		updated, applyErr := applyReceiptToContent(note.Raw, receipt)
		if applyErr != nil {
			return applyErr
		}

		lfRaw := toLF(note.Raw)
		if updated == string(lfRaw) {
			return nil // nothing to record: a CRLF note is not converted
		}

		writeErr := write(note.Path, []byte(updated))
		if writeErr != nil || len(lfRaw) == len(note.Raw) {
			return writeErr
		}

		// The receipt write converted a CRLF note, which changes its
		// content hash: rebuild its sidecar (#789 design D4).
		return rebuildConvertedSidecar(ctx, deps.Embed, write, note.Path, updated)
	}
}

// newUpdateExchange is `engram update`'s parent-exchange step (design D6):
// with a parent configured and outside --dry-run it drains, ignoring the
// backoff window; then it returns the outbox notice ("" when idle).
func newUpdateExchange(deps Deps) func(ctx context.Context, vault string, dryRun bool) string {
	return func(ctx context.Context, vault string, dryRun bool) string {
		parentURL := parentBase(deps)
		if parentURL != "" && !dryRun {
			drainForCommand(ctx, deps, vault, parentURL, true)
		}

		return outboxNotice(outboxStoreFromDeps(deps), vault, parentURL)
	}
}

// outboxCountLine is the notice's first line: the queued and rejected
// counts, the attention count when there is any (#789 design D3, so an
// outbox without such entries reports exactly as before), and the oldest
// entry's age.
func outboxCountLine(box outboxFile, now time.Time) string {
	oldest := now

	for _, entry := range box.Entries {
		if entry.Queued.Before(oldest) {
			oldest = entry.Queued
		}
	}

	attention := ""
	if count := outboxStateCount(box, outboxStateAttention); count > 0 {
		attention = fmt.Sprintf(", %d need attention", count)
	}

	return fmt.Sprintf("parent outbox: %d offer(s) queued, %d rejected%s, oldest queued %s ago\n",
		queuedOfferCount(box), outboxStateCount(box, outboxStateRejected), attention,
		now.Sub(oldest).Round(time.Second))
}

// outboxNotice is update's notify-only outbox report: the queued count,
// the oldest entry's age, each rejected or attention entry with its note
// and error, and the backoff retry time. It is "" when the outbox is empty and no backoff
// is active.
func outboxNotice(store outboxStore, vault, parentURL string) string {
	box, boxErr := loadOutbox(store.state, vault)
	if boxErr != nil {
		return fmt.Sprintf("parent outbox: unreadable: %v\n", boxErr)
	}

	cache, _ := loadParentCache(store.state, vault)
	now := store.now()
	backoff := parentURL != "" && cache.URL == parentURL && now.Before(cache.BackoffUntil)

	if len(box.Entries) == 0 && !backoff {
		return ""
	}

	var notice strings.Builder

	if len(box.Entries) > 0 {
		notice.WriteString(outboxCountLine(box, now))
		writeRejectedEntries(&notice, store, vault, box)
	}

	if backoff {
		fmt.Fprintf(&notice, "parent unreachable: retrying after %s (%d consecutive failure(s))\n",
			cache.BackoffUntil.Format(time.RFC3339), cache.Failures)
	}

	return notice.String()
}

// stageOffer classifies a write (classifyOffer) and, when it is offered,
// stamps an xid on a note that has none. The caller holds the vault lock.
func stageOffer(state exchangeState, vault, parentURL string, write offerWrite) ([]byte, string, error) {
	if !classifyOffer(write, cachedParentVaultID(state, vault, parentURL)) {
		return write.raw, "", nil
	}

	note, _ := parseOfferClassNote(write.raw)
	if note.exchange.XID != "" {
		return write.raw, note.exchange.XID, nil
	}

	xid, mintErr := mintXID(state.randRead)
	if mintErr != nil {
		return nil, "", mintErr
	}

	stamped, stampErr := setXIDField(string(write.raw), xid)
	if stampErr != nil {
		return nil, "", stampErr
	}

	return []byte(stamped), xid, nil
}

// supersedesTargets maps each local note (by basename and every alias)
// that has a primary link to the parent onto that parent note's basename.
func supersedesTargets(store outboxStore, vault, parentVaultID string) (map[string]string, error) {
	notes, scanErr := scanExchangeNotes(vault, store.listMD, store.state.fs.ReadFile)
	if scanErr != nil {
		return nil, fmt.Errorf("offer: %w", scanErr)
	}

	targets := make(map[string]string, len(notes))

	for _, note := range notes {
		link, linked := primaryParentLink(note.exchange, parentVaultID)
		if !linked {
			continue
		}

		targets[note.basename] = link.Note

		for _, alias := range note.exchange.Aliases {
			targets[alias] = link.Note
		}
	}

	return targets, nil
}

// writeRejectedEntries lists each rejected entry, then each entry that
// needs attention (#789 design D3), by its note's current basename (its xid
// when the note is gone) and its error.
func writeRejectedEntries(notice *strings.Builder, store outboxStore, vault string, box outboxFile) {
	names := map[string]string{}

	notes, scanErr := scanOfferNotes(store, vault)
	if scanErr == nil {
		for xid, found := range notes {
			names[xid] = found.note.Basename
		}
	}

	labels := []struct{ state, label string }{
		{outboxStateRejected, "rejected"}, {outboxStateAttention, "needs attention"},
	}

	for _, kind := range labels {
		for _, entry := range fifoEntries(box.Entries) {
			if entry.State != kind.state {
				continue
			}

			name := names[entry.XID]
			if name == "" {
				name = entry.XID
			}

			fmt.Fprintf(notice, "  %s: %s: %s\n", kind.label, name, entry.LastError)
		}
	}
}
