package cli

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// unexported constants.
const (
	// offerDrainWarningFormat reports a drain that stopped or failed to
	// record a receipt; the local write it followed already succeeded.
	offerDrainWarningFormat = "offer: %v"
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
	if !gateParentContact(store, vault, parentURL, ignoreBackoff) {
		return
	}

	result, drainErr := drainOutbox(ctx, store, vault, parentURL,
		newOfferSender(deps.Fetch, parentURL, newPayloadBuilder(ctx, deps, store, vault, parentURL)),
		newReceiptApplier(deps))

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
// never re-embeds and never re-stamps identity.
func newReceiptApplier(deps Deps) func(offerNote, offerReceipt) error {
	write := writeAtomicFromFS(deps.FS, "write note")

	return func(note offerNote, receipt offerReceipt) error {
		updated, applyErr := applyReceiptToContent(note.Raw, receipt)
		if applyErr != nil {
			return applyErr
		}

		if updated == string(note.Raw) {
			return nil
		}

		return write(note.Path, []byte(updated))
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

// outboxNotice is update's notify-only outbox report: the queued count,
// the oldest entry's age, each rejected entry with its note and error, and
// the backoff retry time. It is "" when the outbox is empty and no backoff
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
		oldest := now

		for _, entry := range box.Entries {
			if entry.Queued.Before(oldest) {
				oldest = entry.Queued
			}
		}

		rejected := len(box.Entries) - queuedOfferCount(box)

		fmt.Fprintf(&notice, "parent outbox: %d offer(s) queued, %d rejected, oldest queued %s ago\n",
			queuedOfferCount(box), rejected, now.Sub(oldest).Round(time.Second))
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

// writeRejectedEntries lists each rejected entry by its note's current
// basename (its xid when the note is gone) and its error.
func writeRejectedEntries(notice *strings.Builder, store outboxStore, vault string, box outboxFile) {
	names := map[string]string{}

	notes, scanErr := scanOfferNotes(store, vault)
	if scanErr == nil {
		for xid, found := range notes {
			names[xid] = found.note.Basename
		}
	}

	for _, entry := range fifoEntries(box.Entries) {
		if entry.State != outboxStateRejected {
			continue
		}

		name := names[entry.XID]
		if name == "" {
			name = entry.XID
		}

		fmt.Fprintf(notice, "  rejected: %s: %s\n", name, entry.LastError)
	}
}
