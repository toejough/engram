package cli

import (
	"slices"
	"sort"
	"strings"
)

// unexported types.

// dedupeCandidate is a live local note with its exchange hash, as the merged
// query's dedupe sees it ("" when the hash can't be computed: such a note
// can still match by link, never by hash).
type dedupeCandidate struct {
	note exchangeNote
	hash string
}

// unexported functions (alphabetical).

// applyMergedNoteFloor is the merged cap's note floor (design D9, M3): when
// the kept direct items hold fewer than min(noteFloorK, |Q|) members of Q —
// the direct non-chunk items scoring at least matchRelevanceFloor — the
// lowest-positioned kept chunks are replaced, one for one, by the
// highest-scoring excluded members of Q. The kept set is then restored to
// score order. excluded is in score order.
func applyMergedNoteFloor(kept, excluded []queryItem) []queryItem {
	keptNotes := countFloorNotes(kept)
	promotable := make([]queryItem, 0, len(excluded))

	for _, item := range excluded {
		if isMergedFloorNote(item) {
			promotable = append(promotable, item)
		}
	}

	need := min(noteFloorK, keptNotes+len(promotable)) - keptNotes

	for index := len(kept) - 1; index >= 0 && need > 0; index-- {
		if kept[index].Kind != chunkItemKind {
			continue
		}

		kept[index] = promotable[0]
		promotable = promotable[1:]
		need--
	}

	sortItemsByScoreDesc(kept)

	return kept
}

// capMergedChannelOne orders the merged non-trigger Channel 1 items —
// direct items, then explore picks, each by descending score — and caps
// them to limit by position, keeping the note floor (design D9, #744).
// limit <= 0 is unbounded.
func capMergedChannelOne(items []queryItem, limit int) []queryItem {
	direct := make([]queryItem, 0, len(items))
	explore := make([]queryItem, 0, len(items))

	for _, item := range items {
		if hasProvenance(item, provenanceExplore) {
			explore = append(explore, item)

			continue
		}

		direct = append(direct, item)
	}

	sortItemsByScoreDesc(direct)
	sortItemsByScoreDesc(explore)

	if limit > 0 && len(direct) >= limit {
		kept := make([]queryItem, limit)
		copy(kept, direct[:limit])

		return applyMergedNoteFloor(kept, direct[limit:])
	}

	return capItemsToLimit(append(direct, explore...), limit)
}

// countFloorNotes counts the members of the note floor's set Q in items.
func countFloorNotes(items []queryItem) int {
	count := 0

	for _, item := range items {
		if isMergedFloorNote(item) {
			count++
		}
	}

	return count
}

// dedupeCandidates returns the live local notes with their exchange hashes.
// Pending notes never suppress a parent item.
func dedupeCandidates(notes []exchangeNote) []dedupeCandidate {
	out := make([]dedupeCandidate, 0, len(notes))

	for _, note := range notes {
		if note.pending {
			continue
		}

		hash, hashErr := exchangeHash(note.raw)
		if hashErr != nil {
			hash = ""
		}

		out = append(out, dedupeCandidate{note: note, hash: hash})
	}

	return out
}

// dedupeParentItems applies the D4 dedupe rule to the parent's items: an
// item that matches a live local note L is dropped when L is already among
// the local items (or was already substituted), and is otherwise replaced
// by a substituted item for L at the same position.
func dedupeParentItems(
	parentItems, localItems []queryItem, notes []exchangeNote, parentVaultID, localModelID string,
) []queryItem {
	present := make(map[string]bool, len(localItems))
	for _, item := range localItems {
		present[item.Path] = true
	}

	candidates := dedupeCandidates(notes)
	out := make([]queryItem, 0, len(parentItems))

	for _, item := range parentItems {
		local, matched := matchLocalNote(item, candidates, parentVaultID)
		if !matched {
			out = append(out, item)

			continue
		}

		path := local.basename + mdExt
		if present[path] {
			continue
		}

		present[path] = true

		out = append(out, substituteLocalItem(item, local, localModelID))
	}

	return out
}

// dropParentChunks removes every parent chunk item, including the parent's
// recency channel, before any other merge step (design D9, Q1).
func dropParentChunks(items []queryItem) []queryItem {
	out := make([]queryItem, 0, len(items))

	for _, item := range items {
		if item.Kind == chunkItemKind || hasProvenance(item, provenanceRecent) {
			continue
		}

		out = append(out, item)
	}

	return out
}

// isMergedFloorNote reports whether item belongs to the note floor's set Q:
// a direct non-chunk item scoring at least matchRelevanceFloor (ruling S19,
// matching the local capWithNoteFloor).
func isMergedFloorNote(item queryItem) bool {
	return item.Kind != chunkItemKind && item.Score >= matchRelevanceFloor
}

// linksToAny reports whether note holds a link, under parentVaultID, to any
// of names. An unknown parent vault ID never matches (S16).
func linksToAny(note exchangeNote, parentVaultID string, names []string) bool {
	if parentVaultID == "" || note.exchange.Parent.Vault != parentVaultID {
		return false
	}

	return slices.ContainsFunc(note.exchange.Parent.Links, func(link parentLink) bool {
		return slices.Contains(names, link.Note)
	})
}

// matchLocalNote finds the live local note a parent item is the same note
// as (design D4): a link under the parent's reported vault ID naming the
// item's basename or one of its aliases, or an equal, known exchange hash.
func matchLocalNote(item queryItem, candidates []dedupeCandidate, parentVaultID string) (exchangeNote, bool) {
	names := append([]string{strings.TrimSuffix(item.Path, mdExt)}, item.Aliases...)

	for _, candidate := range candidates {
		if linksToAny(candidate.note, parentVaultID, names) ||
			exchangeHashesMatch(candidate.hash, item.ExchangeHash) {
			return candidate.note, true
		}
	}

	return exchangeNote{}, false
}

// sortItemsByScoreDesc stably sorts items by descending score.
func sortItemsByScoreDesc(items []queryItem) {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Score > items[j].Score
	})
}

// stripDedupeKeys clears every item's dedupe keys, which never appear in a
// merged payload.
func stripDedupeKeys(items []queryItem) {
	for index := range items {
		items[index].ExchangeHash = ""
		items[index].Aliases = nil
	}
}

// substituteLocalItem is the item for local note L taking parent item P's
// place (design D9, M4): L's path, kind and content (rendered as a local
// query renders it), P's score and provenances, P's source_term only for an
// explore pick, and the local model.
func substituteLocalItem(parentItem queryItem, local exchangeNote, localModelID string) queryItem {
	content := stripWikilinks(string(local.raw))
	sourceTerm := ""

	if hasProvenance(parentItem, provenanceExplore) {
		sourceTerm = parentItem.SourceTerm
	} else {
		content = capRedFlagsForPreview(content)
	}

	return queryItem{
		Path:        local.basename + mdExt,
		Kind:        kindFromContent(content),
		Score:       parentItem.Score,
		Provenances: parentItem.Provenances,
		Content:     content,
		SourceTerm:  sourceTerm,
		ModelID:     localModelID,
		FromParent:  new(false),
	}
}

// sumExploreAllocated sums two explore_allocated maps per term; the result
// is never nil, so the block always shows the field.
func sumExploreAllocated(first, second map[string]int) map[string]int {
	out := make(map[string]int, len(first)+len(second))

	for term, count := range first {
		out[term] += count
	}

	for term, count := range second {
		out[term] += count
	}

	return out
}
