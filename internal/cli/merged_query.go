package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"slices"
	"sort"

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

// interleaveAlternating combines two item slices by alternating between
// them (a, b, a, b, ...) rather than claiming a true cross-source
// chronological order — queryItem carries no timestamp field, so no such
// order is derivable from a rendered payload (design.md Decision 7). Each
// source's own items are already newest-first in their own relative order.
func interleaveAlternating(a, b []queryItem) []queryItem {
	out := make([]queryItem, 0, len(a)+len(b))

	for i, j := 0, 0; i < len(a) || j < len(b); {
		if i < len(a) {
			out = append(out, a[i])
			i++
		}

		if j < len(b) {
			out = append(out, b[j])
			j++
		}
	}

	return out
}

// mergeByScoreDesc combines two item slices into one, sorted by descending
// score.
func mergeByScoreDesc(a, b []queryItem) []queryItem {
	out := make([]queryItem, 0, len(a)+len(b))
	out = append(out, a...)
	out = append(out, b...)

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Score > out[j].Score
	})

	return out
}

// mergeQueryPayloads combines a local and parent queryPayload into one
// merged payload (vault-merged-recall): items are tagged with model_id and
// from_parent, score-ranked items from both sources are interleaved by
// descending score, recency-channel items are combined separately and
// capped to --recent-fill, the whole set is capped to --limit, and
// --content-budget is reapplied over the final merged set. Clusters are
// not fused across nodes (spec: "Merged results are not re-clustered
// across nodes") — the merged payload keeps only the local node's own
// cluster structure.
func mergeQueryPayloads(local, parent queryPayload, args QueryArgs) queryPayload {
	localMain, localRecent := splitRecencyChannel(tagItems(local.Items, local.ModelID, false))
	parentMain, parentRecent := splitRecencyChannel(tagItems(parent.Items, parent.ModelID, true))

	// --limit caps Channel 1 (relevance-ranked) only — Channel 2 (recency)
	// has its own dedicated budget (--recent-fill) and must not be
	// displaced just because Channel 1 alone already reaches --limit
	// (mirrors renderQueryPayload's same fix for the single-source path).
	mainMerged := capItemsToLimit(mergeByScoreDesc(localMain, parentMain), resolveLimit(args.Limit))
	recentMerged := capItemsToLimit(
		interleaveAlternating(localRecent, parentRecent), resolveRecentFill(args.RecentFill))

	mainMerged = append(mainMerged, recentMerged...)
	items, _ := capChunkContent(mainMerged, resolveContentBudget(args.ContentBudget))

	return queryPayload{
		Version:       1,
		Phrases:       local.Phrases,
		Items:         items,
		Clusters:      local.Clusters,
		RefitPending:  local.RefitPending || parent.RefitPending,
		PendingOffers: local.PendingOffers || parent.PendingOffers,
		ModelID:       local.ModelID,
	}
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
// (vault-merged-recall). The parent is fetched first: on failure, it
// degrades to an ordinary local-only RunQuery call (no merge, no unbounded
// dance) and emits a non-fatal warning (spec: "Parent unavailability
// degrades to local-only results"). On success, both sources are fetched
// at unbounded budgets and merged.
func runMergedQuery(ctx context.Context, deps Deps, parentBaseURL string, args QueryArgs, stdout io.Writer) error {
	parentPayload, parentErr := fetchQueryPayload(ctx, deps, parentBaseURL, unboundedQueryArgs(args))
	if parentErr != nil {
		logWarningTo(deps.Stderr)("query: parent unavailable, returning local-only results: %v", parentErr)

		return RunQuery(ctx, args, newQueryDeps(deps), stdout)
	}

	localPayload, localErr := runLocalQueryPayload(ctx, deps, unboundedQueryArgs(args))
	if localErr != nil {
		return localErr
	}

	return encodeQueryPayload(stdout, mergeQueryPayloads(localPayload, parentPayload, args))
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
