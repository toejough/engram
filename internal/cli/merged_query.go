package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"

	"go.yaml.in/yaml/v3"
)

// unexported constants.
const (
	// mergeUnboundedBudget is the sentinel passed for --recent-fill and
	// --limit when fetching each source during a merge, so neither
	// pre-truncates candidates the merge might need — the user's real
	// requested values are applied exactly once, over the merged set
	// (design.md Decision 7). content-budget has its own unlimited value
	// (<=0, via capChunkContent's existing guard) and doesn't need this.
	mergeUnboundedBudget = 1_000_000
)

// unexported functions (alphabetical).

// encodeQueryPayload YAML-encodes payload to stdout — the write-only half
// of what renderQueryPayload does for a freshly-computed aggregatedSummary,
// reused here for a payload that was assembled by merging instead.
func encodeQueryPayload(stdout io.Writer, payload queryPayload) error {
	const yamlIndent = 2

	encoder := yaml.NewEncoder(stdout)
	encoder.SetIndent(yamlIndent)

	err := encoder.Encode(payload)
	if err != nil {
		return fmt.Errorf("query: encode: %w", err)
	}

	closeErr := encoder.Close()
	if closeErr != nil {
		return fmt.Errorf("query: close encoder: %w", closeErr)
	}

	return nil
}

// hasProvenance reports whether item carries the given provenance role.
func hasProvenance(item queryItem, role string) bool {
	return slices.Contains(item.Provenances, role)
}

// mergeQueryPayloads combines a local and parent queryPayload into one
// merged payload (vault-merged-recall, design D9):
//  1. every parent chunk item is dropped, including the parent's recency
//     channel (Q1);
//  2. parent notes that match a live local note are deduped, keeping the
//     local note (D4; substituted per M4 when it did not rank locally);
//  3. trigger hits lead, local before parent, exempt from --limit;
//  4. direct items then explore picks, each by score, are capped to
//     --limit by position with the note floor (M3, #744);
//  5. the local recency channel follows, capped to --recent-fill;
//  6. --content-budget (or --lazy-chunks) is applied once, and the budget
//     block reports the applied values (#743).
//
// Clusters stay the local node's own, and the pending-offer hint and the
// refit flag reflect the local vault only (H4, ruling S20): a child can
// neither curate its parent's offers nor run its parent's refit.
func mergeQueryPayloads(local, parent queryPayload, localNotes []exchangeNote, args QueryArgs) queryPayload {
	localItems := tagItems(local.Items, local.ModelID, false)
	parentItems := dedupeParentItems(dropParentChunks(tagItems(parent.Items, parent.ModelID, true)),
		localItems, localNotes, parent.VaultID, local.ModelID)

	localMain, localRecent := splitRecencyChannel(localItems)
	localTriggered, localMain := splitTriggerItems(localMain)
	parentTriggered, parentMain := splitTriggerItems(parentItems)

	limit := resolveLimit(args.Limit)
	channelOne := capMergedChannelOne(append(localMain, parentMain...), limit)
	recent := capItemsToLimit(localRecent, resolveRecentFill(args.RecentFill))

	items := make([]queryItem, 0, len(localTriggered)+len(parentTriggered)+len(channelOne)+len(recent))
	items = append(items, localTriggered...)
	items = append(items, parentTriggered...)
	items = append(items, channelOne...)
	items = append(items, recent...)
	stripDedupeKeys(items)

	items, snipped := mergedContentPolicy(items, args)

	return queryPayload{
		Version:           1,
		Phrases:           local.Phrases,
		Items:             items,
		Clusters:          local.Clusters,
		RefitPending:      local.RefitPending,
		PendingOffers:     local.PendingOffers,
		PendingOffersHint: pendingOffersHint(local.PendingOffers),
		ModelID:           local.ModelID,
		Budget: queryBudget{
			PhrasesQueried:       len(local.Phrases),
			TotalNotes:           local.Budget.TotalNotes,
			WithEmbeddings:       local.Budget.WithEmbeddings,
			ClustersFound:        len(local.Clusters),
			DirectHitsReturned:   countDirectHits(items),
			ItemsWithFullContent: countItemsWithContent(items) - snipped,
			Limit:                limit,
			ContentBudget:        resolveContentBudget(args.ContentBudget),
			ChunksSnippeted:      snipped,
			ExploreAllocated:     sumExploreAllocated(local.Budget.ExploreAllocated, parent.Budget.ExploreAllocated),
			LazyChunks:           args.LazyChunks,
		},
	}
}

// mergedContentPolicy is applyContentPolicy for the merged set: lazy mode
// clears chunk content (nothing is snippeted); otherwise --content-budget
// caps full-content chunks once over the merged items.
func mergedContentPolicy(items []queryItem, args QueryArgs) ([]queryItem, int) {
	if args.LazyChunks {
		return clearChunkContent(items), 0
	}

	return capChunkContent(items, resolveContentBudget(args.ContentBudget))
}

// prepareParentContact runs design D2's parent-contact bookkeeping for a
// command about to contact the parent: a missing vault ID is stamped, and a
// failed location check prints its one warning (read-only contacts still
// run). Neither is fatal. It reports whether the vault ID is stamped: when
// it is not, the caller skips the contact's own bookkeeping (backoff gate,
// outcome record, drain), so nothing creates .engram/ without a stamp
// (ruling S4). command prefixes the stamp warning.
func prepareParentContact(deps Deps, command, vault string) bool {
	state := exchangeStateFromDeps(deps)

	_, stampErr := stampVaultID(state, vault)
	if stampErr != nil {
		logWarningTo(deps.Stderr)("%s: could not stamp the vault ID: %v", command, stampErr)

		return false
	}

	warnVaultLocation(state, vault, deps.Stderr)

	return true
}

// recordParentContact records a parent request's outcome for backoff: a
// transport error, timeout or 5xx backs off; any answer, including a 4xx,
// resets it (ruling S14) and caches the vault ID the parent reported, when
// known. Failing to record either is not fatal.
func recordParentContact(store outboxStore, vault, parentURL, vaultID string, fetchErr error) {
	if errors.Is(fetchErr, errParentUnreachable) {
		_, _ = recordParentFailure(store, vault, parentURL)

		return
	}

	_ = recordParentSuccess(store, vault, parentURL, vaultID)
}

// resolveLimit maps the raw --limit flag value to the effective cap: 0
// (unset) → the baked default (defaultQueryLimit); positive → that
// explicit value.
func resolveLimit(raw int) int {
	if raw == 0 {
		return defaultQueryLimit
	}

	return raw
}

// runLocalQueryPayload runs RunQuery into an in-memory buffer and decodes
// the result into a queryPayload — mirrors how serveQuery already captures
// RunQuery's stdout into a buffer, so the merge orchestrator can combine
// the local result with the parent's before a single final render.
func runLocalQueryPayload(ctx context.Context, deps Deps, args QueryArgs) (queryPayload, error) {
	var buf bytes.Buffer

	err := RunQuery(ctx, args, newQueryDeps(deps), &buf)
	if err != nil {
		return queryPayload{}, err
	}

	var payload queryPayload

	unmarshalErr := yaml.Unmarshal(buf.Bytes(), &payload)
	if unmarshalErr != nil {
		return queryPayload{}, fmt.Errorf("query: decoding local payload for merge: %w", unmarshalErr)
	}

	return payload, nil
}

// runMergedQuery orchestrates ENGRAM_PARENT-gated local+parent query fusion
// (vault-merged-recall, design D9). It degrades to an ordinary local-only
// RunQuery (no merge, no drain) when the parent is backed off, is this
// vault itself (the self-parent guard, checked on the cached ID before the
// request and on the reported ID after it), or fails — the last with a
// non-fatal warning. Otherwise both sources are fetched at unbounded
// budgets, merged, and the outbox drains.
func runMergedQuery(ctx context.Context, deps Deps, parentBaseURL string, args QueryArgs, stdout io.Writer) error {
	vault := args.VaultPath
	localOnly := func() error { return RunQuery(ctx, args, newQueryDeps(deps), stdout) }

	stamped := prepareParentContact(deps, "query", vault)

	store := outboxStoreFromDeps(deps)
	if stamped && !gateParentContact(store, vault, parentBaseURL, false) {
		return localOnly()
	}

	localID, _, _ := readVaultID(store.state, vault)
	if selfParentGuard(localID, cachedParentVaultID(store.state, vault, parentBaseURL), deps.Stderr) {
		return localOnly()
	}

	parentPayload, parentErr := fetchQueryPayload(ctx, deps, parentBaseURL, unboundedQueryArgs(args))
	if stamped {
		recordParentContact(store, vault, parentBaseURL, parentPayload.VaultID, parentErr)
	}

	if parentErr != nil {
		logWarningTo(deps.Stderr)("query: parent unavailable, returning local-only results: %v", parentErr)

		return localOnly()
	}

	if selfParentGuard(localID, parentPayload.VaultID, deps.Stderr) {
		return localOnly()
	}

	localPayload, localErr := runLocalQueryPayload(ctx, deps, unboundedQueryArgs(args))
	if localErr != nil {
		return localErr
	}

	encodeErr := encodeQueryPayload(stdout,
		mergeQueryPayloads(localPayload, parentPayload, scanDedupeNotes(deps, vault), args))
	if encodeErr != nil {
		return encodeErr
	}

	// The parent answered, so the outbox drains (design D6); the gate
	// already passed for this command. An unstamped vault has no outbox.
	if stamped {
		drainForCommand(ctx, deps, vault, parentBaseURL, true)
	}

	return nil
}

// scanDedupeNotes reads the local exchange notes for the merged query's
// dedupe. A scan failure is warned and merges without dedupe rather than
// failing a read-only query.
func scanDedupeNotes(deps Deps, vault string) []exchangeNote {
	notes, scanErr := scanExchangeNotes(vault, listMDFromFS(deps.FS), deps.FS.ReadFile)
	if scanErr != nil {
		logWarningTo(deps.Stderr)("query: could not read local notes for dedupe: %v", scanErr)

		return nil
	}

	return notes
}

// splitRecencyChannel separates items into main (non-recency) and
// recency-channel (provenanceRecent) subsets, preserving relative order.
func splitRecencyChannel(items []queryItem) (main, recent []queryItem) {
	for _, item := range items {
		if hasProvenance(item, provenanceRecent) {
			recent = append(recent, item)

			continue
		}

		main = append(main, item)
	}

	return main, recent
}

// tagItems returns a copy of items with ModelID and FromParent set per the
// node that produced them.
func tagItems(items []queryItem, modelID string, fromParent bool) []queryItem {
	out := make([]queryItem, len(items))

	for i, item := range items {
		item.ModelID = modelID
		item.FromParent = fromParent
		out[i] = item
	}

	return out
}

// unboundedQueryArgs returns a copy of args with content-budget, recent-fill,
// and limit overridden to effectively-unbounded values, so a source fetched
// for merging doesn't pre-truncate candidates before the real requested
// budgets are applied once over the merged set (design.md Decision 7).
func unboundedQueryArgs(args QueryArgs) QueryArgs {
	out := args
	out.ContentBudget = -1
	out.RecentFill = mergeUnboundedBudget
	out.Limit = mergeUnboundedBudget

	return out
}
