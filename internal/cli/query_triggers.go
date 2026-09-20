package cli

import (
	"io"
	"slices"
	"sort"
	"strings"

	"github.com/toejough/engram/internal/vaultgraph"
)

// applyTriggerHits marks each trigger-hit note with provenanceTrigger and
// moves it to the front of resolved, ahead of every similarity-ranked item
// (runbook-lexical-triggers spec). A hit already present keeps its score and
// other roles; a hit absent from resolved is added with score 0 and its
// (red_flags-capped) content. --project still filters: a runbook outside the
// requested project is not surfaced. Returns resolved unchanged when hits is
// empty.
func applyTriggerHits(
	resolved []resolvedItem,
	hits []string,
	meta AllVaultNotesMeta,
	project string,
) []resolvedItem {
	if len(hits) == 0 {
		return resolved
	}

	hitPaths := make(map[string]struct{}, len(hits))
	for _, basename := range hits {
		hitPaths[pathOf(basename)] = struct{}{}
	}

	triggered := make([]resolvedItem, 0, len(hits))
	rest := make([]resolvedItem, 0, len(resolved))
	seen := make(map[string]struct{}, len(hits))

	for _, item := range resolved {
		_, isHit := hitPaths[item.notePath]
		if !isHit || item.kind == chunkItemKind {
			rest = append(rest, item)

			continue
		}

		item.provenances = append(slices.Clone(item.provenances), provenanceTrigger)
		triggered = append(triggered, item)
		seen[item.notePath] = struct{}{}
	}

	for _, basename := range hits {
		notePath := pathOf(basename)
		if _, dup := seen[notePath]; dup {
			continue
		}

		item := resolvedItem{
			notePath:    notePath,
			content:     capRedFlagsForPreview(meta.ContentByBasename[basename]),
			provenances: []string{provenanceTrigger},
		}

		if project != "" && !itemMatchesProject(item, project) {
			continue
		}

		triggered = append(triggered, item)
	}

	sort.SliceStable(triggered, func(i, j int) bool {
		return resolvedItemLess(triggered[i], triggered[j])
	})

	return append(triggered, rest...)
}

// matchTriggers returns the sorted basenames whose triggers contain an entry
// that is a substring of text, comparing case-insensitively after collapsing
// whitespace runs to one space on both sides. Empty text matches nothing.
func matchTriggers(text string, index map[string][]string) []string {
	hits := make([]string, 0, len(index))

	normalized := normalizeTriggerText(text)
	if normalized == "" {
		return hits
	}

	for basename, triggers := range index {
		for _, trigger := range triggers {
			needle := normalizeTriggerText(trigger)
			if needle != "" && strings.Contains(normalized, needle) {
				hits = append(hits, basename)

				break
			}
		}
	}

	sort.Strings(hits)

	return hits
}

// normalizeTriggerText lowercases s and collapses whitespace runs to one space.
func normalizeTriggerText(text string) string {
	return strings.ToLower(strings.Join(strings.Fields(text), " "))
}

// runTriggerOnlyQuery serves `engram query --text …` with no --phrase: no
// embedding, matching, clustering, explore, or recency fill — just the
// trigger hits, rendered through the ordinary payload path.
func runTriggerOnlyQuery(
	args QueryArgs,
	notes []vaultgraph.Note,
	hits []compatibleSidecar,
	limit int,
	deps QueryDeps,
	timer *phaseTimer,
	stdout io.Writer,
	pendingOffers bool,
	modelID string,
) error {
	vaultMeta := loadAllVaultNotesMeta(hits, args.VaultPath, deps.Read)
	resolved := applyTriggerHits(nil, matchTriggers(args.Text, vaultMeta.TriggerIndex), vaultMeta, args.Project)

	timer.mark(stageScan)

	merged := buildAggregatedSummary(aggregatedSummaryInputs{
		args: args, notes: notes, hits: hits, limit: limit, deps: deps, timer: timer,
		resolved: resolved, matchSet: matchedSet{}, report: clusterMatchedSet(matchedSet{}, ""),
		exploreAllocated: map[string]int{}, pendingOffers: pendingOffers, modelID: modelID,
	})

	return renderQueryPayload(stdout, merged)
}

// splitTriggerItems separates trigger-hit items from the rest, preserving
// relative order. Trigger hits are exempt from --limit.
func splitTriggerItems(items []queryItem) (triggered, rest []queryItem) {
	for _, item := range items {
		if hasProvenance(item, provenanceTrigger) {
			triggered = append(triggered, item)

			continue
		}

		rest = append(rest, item)
	}

	return triggered, rest
}
