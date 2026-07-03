package cli

import (
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Exported constants.
const (
	// ProvenanceRideAlong tags superseding-note items inserted by the supersession
	// ride-along feature. Identifiable in the YAML payload so Gate 2 regression
	// analysis can detect when a ride-along insertion pushed a baseline top-5 note
	// past the cut.
	ProvenanceRideAlong = "ride_along"
)

// AllVaultNotesMeta holds per-note metadata scanned in one pass over all vault
// notes, for use by tag-match nomination and supersession ride-along.
// Fields use exported names for testability via the exported type alias.
type AllVaultNotesMeta struct {
	// TermIndex maps a vocab term name to the notes carrying that term. Vocab and
	// vocab-index notes are excluded from this map — they are never nominated.
	TermIndex map[string][]NominationEntry
	// SupersedesInverse maps a superseded-note basename to the superseder entries,
	// built from every note's supersedes: frontmatter block.
	SupersedesInverse SupersedesInverse
	// ContentByBasename stores loaded note content keyed by basename (for ride-along
	// superseder insertion — the superseder may not be in the matched set).
	ContentByBasename map[string]string
}

// NominationEntry holds a vault note that is eligible for tag-match nomination.
// Both fields use exported names so cli_test can build fixtures via the exported
// type alias without needing unexported accessors.
type NominationEntry struct {
	// NotePath is the vault-relative path of the note (e.g. "1aa.note.md").
	NotePath string
	// Content is the note's raw text (wikilinks stripped) used for candidate_l2s.
	Content string
}

// unexported constants.
const (
	// nominationCapPerCluster is the maximum number of tag-nominated additional
	// candidates appended to any single cluster's candidate_l2s. Prevents payload
	// explosion when a high-coverage vocab term is shared by many notes. Truncation
	// is not silent — callers report the count in the query budget.
	//
	// Anchored to the plan's gate threshold: "median nomination pool ≤ 40". The
	// S2 probe (54.2% recovery) used an uncapped flat pool with median size ~34.
	// A per-cluster cap of 40, combined with cross-cluster deduplication in
	// buildTagNominations, bounds the total unique additions to ≤ 40 in practice
	// (the first-processed cluster wins most slots; subsequent clusters see fewer
	// un-nominated notes due to the shared nominated map).
	nominationCapPerCluster = 40
	// topNForNomination is the number of top-ranked delivered notes whose vocab:
	// frontmatter terms seed the tag-match nomination pool.
	topNForNomination = 3
)

// noteQueryFrontmatter is the minimal parsed shape of a note's frontmatter for
// query-integration purposes (vocab terms + supersedes entries). Parsed once per
// note in loadAllVaultNotesMeta and not re-parsed later.
type noteQueryFrontmatter struct {
	Vocab      []string          `yaml:"vocab"`
	Supersedes []supersedesEntry `yaml:"supersedes"`
}

// tagNominationTally reports the nomination outcome for the query budget —
// the no-silent-caps rule: every truncation by nominationCapPerCluster is
// counted here and emitted as tag_nominations_added / tag_nominations_dropped
// in the payload budget, so a capped pool is always visible to the caller.
type tagNominationTally struct {
	added   int // nominations kept (post-cap) across all clusters
	dropped int // nominations truncated by nominationCapPerCluster
}

// addNominationsForTerm appends candidate notes from entries to nominations[clusterID],
// skipping notes already in results or already nominated, and vocab-kind notes.
// It updates nominated in-place so cross-term dedup is maintained by the caller.
func addNominationsForTerm(
	entries []NominationEntry,
	clusterID int,
	alreadyInResults, nominated map[string]bool,
	nominations map[int][]queryCandidateNote,
) {
	for _, entry := range entries {
		if alreadyInResults[entry.NotePath] || nominated[entry.NotePath] {
			continue
		}

		if isQueryExcludedKind(entry.Content) {
			// Safety guard: the TermIndex builder excludes vocab/qa-question notes,
			// but double-check here so nomination is always safe to call.
			continue
		}

		nominations[clusterID] = append(nominations[clusterID], queryCandidateNote{
			Path:    entry.NotePath,
			Cosine:  0, // nominated by tag, not by centroid cosine
			Content: entry.Content,
		})
		nominated[entry.NotePath] = true
	}
}

// applySupersedesRideAlong inserts superseding notes directly after any delivered
// note that has a recorded superseder in the inverse map.
//
// Design decisions:
//
//   - Only non-chunk, non-recent, non-ride_along items are examined for supersession.
//
//   - The superseder is inserted immediately after the superseded note in resolved,
//     carrying ProvenanceRideAlong so Gate 2 analysis can detect rank shifts.
//
//   - Deduplication: a superseder already present anywhere in resolved (whether as
//     a direct hit, cluster rep, or prior ride-along insertion) is not inserted again.
//
//   - Multiple superseders for one note: each is inserted in order after the
//     superseded note, subject to the same dedup rule.
//
//   - A superseder absent from AllVaultNotesMeta.ContentByBasename (not in the vault
//     or had no compatible sidecar) is skipped silently.
//
//   - Ride-along insertions carry score=0 (no independent ranking signal); kind is
//     derived from content at render time.
//
// Returns the original slice unchanged when SupersedesInverse is empty (no-op for
// backward compatibility on vaults with no supersedes: frontmatter).
func applySupersedesRideAlong(resolved []resolvedItem, meta AllVaultNotesMeta) []resolvedItem {
	if len(meta.SupersedesInverse) == 0 {
		return resolved
	}

	// presentBasenames tracks basenames already in the output for dedup.
	presentBasenames := make(map[string]bool, len(resolved))

	for _, item := range resolved {
		presentBasenames[basenameFromNotePath(item.notePath)] = true
	}

	out := make([]resolvedItem, 0, len(resolved))

	for _, item := range resolved {
		out = append(out, item)

		// Only examine direct-hit note items (non-chunk, non-recent, non-ride_along).
		if item.kind == chunkItemKind {
			continue
		}

		if slices.Contains(item.provenances, provenanceRecent) {
			continue
		}

		if slices.Contains(item.provenances, ProvenanceRideAlong) {
			continue
		}

		basename := basenameFromNotePath(item.notePath)
		superseders := meta.SupersedesInverse[basename]

		for _, superseder := range superseders {
			if presentBasenames[superseder.Note] {
				continue // already in resolved or already inserted
			}

			content, ok := meta.ContentByBasename[superseder.Note]
			if !ok {
				continue // superseder not in vault (no sidecar hit)
			}

			out = append(out, resolvedItem{
				notePath:    pathOf(superseder.Note),
				content:     content,
				score:       0, // ride-along has no independent ranking score
				provenances: []string{ProvenanceRideAlong},
			})
			presentBasenames[superseder.Note] = true
		}
	}

	return out
}

// applyTagNominationAndRideAlong runs the Slice-3 query integration: one
// metadata scan over all vault hits (vocab: + supersedes: frontmatter), the
// supersession ride-along insertion into the ranked items, and tag-match
// nomination for the per-cluster candidate_l2s. Returns the (possibly
// extended) items, the per-cluster nominations, and the nomination tally
// emitted in the payload budget (no-silent-caps rule).
func applyTagNominationAndRideAlong(
	resolved []resolvedItem,
	hits []compatibleSidecar,
	vaultPath string,
	read func(string) ([]byte, error),
	matchSet matchedSet,
	report clusterReport,
) ([]resolvedItem, map[int][]queryCandidateNote, tagNominationTally) {
	vaultMeta := loadAllVaultNotesMeta(hits, vaultPath, read)

	// Supersession ride-along: insert each superseding note directly after its
	// superseded note in the ranked items (note items only; deduped).
	resolved = applySupersedesRideAlong(resolved, vaultMeta)

	// Tag-match nomination: for the top-3 delivered notes, find all vault notes
	// sharing a vocab term and add them to the per-cluster candidate_l2s.
	tagNominations, tally := buildTagNominations(resolved, vaultMeta, matchSet, report)

	return resolved, tagNominations, tally
}

// buildTagNominations computes per-cluster nomination candidates from the vault's
// vocab term index.
//
// Design decisions:
//
//   - Nomination triggers: the top-3 delivered notes (topNForNomination items from
//     resolved that are non-chunk, non-recent, non-ride_along).
//
//   - Cluster assignment: each nominated note is assigned to the cluster that
//     contains its triggering top-3 note. When a note shares terms with top-3 notes
//     in multiple clusters, it is assigned to the cluster of the highest-ranked
//     triggering note (the first match in top-3 order, since resolved is rank-ordered).
//
//   - Deduplication: a note is added at most once across all clusters, and never
//     if it is already in the ranked items (resolved).
//
//   - Per-cluster cap: nominationCapPerCluster additional candidates per cluster.
//     Truncation is not silent — the returned tally reports kept and dropped
//     counts, emitted in the payload budget.
//
//   - Vocab/vocab-index notes are excluded upstream in AllVaultNotesMeta.TermIndex.
//
// Returns (nil, zero tally) when there is no vocab data (TermIndex is empty) or
// no top-3 delivered notes — the no-op path for backward compatibility.
func buildTagNominations(
	resolved []resolvedItem,
	meta AllVaultNotesMeta,
	matchSet matchedSet,
	report clusterReport,
) (map[int][]queryCandidateNote, tagNominationTally) {
	if len(meta.TermIndex) == 0 {
		return nil, tagNominationTally{}
	}

	top3 := topDeliveredNotes(resolved, topNForNomination)
	if len(top3) == 0 {
		return nil, tagNominationTally{}
	}

	// alreadyInResults tracks paths present in the ranked output — nominees must
	// not duplicate items[] content.
	alreadyInResults := make(map[string]bool, len(resolved))

	for _, item := range resolved {
		alreadyInResults[item.notePath] = true
	}

	nominations := make(map[int][]queryCandidateNote)
	// nominated tracks paths already nominated to prevent cross-cluster duplicates.
	nominated := make(map[string]bool)

	for _, top := range top3 {
		terms := parseNoteQueryFrontmatter(top.content).Vocab
		if len(terms) == 0 {
			continue
		}

		clusterID := noteClusterIDForPath(top.notePath, matchSet, report)

		for _, term := range terms {
			addNominationsForTerm(meta.TermIndex[term], clusterID, alreadyInResults, nominated, nominations)
		}
	}

	// Apply the per-cluster cap. Excess entries are dropped (not silently —
	// the tally's added/dropped counts are emitted in the query budget as
	// tag_nominations_added / tag_nominations_dropped).
	tally := tagNominationTally{}

	for clusterID, notes := range nominations {
		if len(notes) > nominationCapPerCluster {
			tally.dropped += len(notes) - nominationCapPerCluster
			nominations[clusterID] = notes[:nominationCapPerCluster]
		}

		tally.added += len(nominations[clusterID])
	}

	if len(nominations) == 0 {
		return nil, tagNominationTally{}
	}

	return nominations, tally
}

// loadAllVaultNotesMeta reads every note in hits once, parsing their vocab: and
// supersedes: frontmatter fields. The results feed both tag-match nomination
// (TermIndex) and supersession ride-along (SupersedesInverse + ContentByBasename).
//
// This is a no-op on vaults with no vocab or supersedes data: it always returns
// an AllVaultNotesMeta with initialised (but possibly empty) maps.
func loadAllVaultNotesMeta(
	hits []compatibleSidecar,
	vault string,
	read func(string) ([]byte, error),
) AllVaultNotesMeta {
	result := AllVaultNotesMeta{
		TermIndex:         make(map[string][]NominationEntry),
		SupersedesInverse: make(SupersedesInverse),
		ContentByBasename: make(map[string]string),
	}

	supersedersByNote := make(map[string][]supersedesEntry)

	for _, hit := range hits {
		notePath := pathOf(hit.note.Basename)
		full := filepath.Join(vault, notePath)

		noteBytes, err := read(full)
		if err != nil {
			continue
		}

		content := stripWikilinks(string(noteBytes))
		basename := hit.note.Basename
		result.ContentByBasename[basename] = content

		meta := parseNoteQueryFrontmatter(content)

		// Populate TermIndex — excluded kinds (vocab/vocab-index/qa-question) are never nominated.
		if !isQueryExcludedKind(content) && len(meta.Vocab) > 0 {
			entry := NominationEntry{NotePath: notePath, Content: content}

			for _, term := range meta.Vocab {
				result.TermIndex[term] = append(result.TermIndex[term], entry)
			}
		}

		// Populate SupersedesInverse via BuildSupersedesInverse after scanning all notes.
		if len(meta.Supersedes) > 0 {
			supersedersByNote[basename] = meta.Supersedes
		}
	}

	result.SupersedesInverse = BuildSupersedesInverse(supersedersByNote)

	return result
}

// noteClusterIDForPath returns the cluster ID containing the note at notePath in
// the matched set. Falls back to cluster 0 when the note is absent from all
// clusters (e.g., not in the matched set at all, or the report is empty).
//
// Design: each matched-set member belongs to exactly one cluster per the AutoK
// assignment. The fallback to 0 handles the single-cluster path and the rare case
// of a top-3 note that was a cluster-rep without appearing in memberIDs.
func noteClusterIDForPath(notePath string, matchSet matchedSet, report clusterReport) int {
	if len(report.memberIDs) == 0 {
		return 0
	}

	// Find the note's index in matchSet.members.
	memberIdx := -1

	for i, member := range matchSet.members {
		if member.notePath == notePath {
			memberIdx = i
			break
		}
	}

	if memberIdx < 0 {
		return 0
	}

	// Find the cluster that contains this member index.
	for clusterID, indices := range report.memberIDs {
		if slices.Contains(indices, memberIdx) {
			return clusterID
		}
	}

	return 0
}

// parseNoteQueryFrontmatter extracts the vocab: and supersedes: fields from note
// content's YAML frontmatter. Returns zero-value fields on any parse failure.
func parseNoteQueryFrontmatter(content string) noteQueryFrontmatter {
	if !strings.HasPrefix(content, fmStart) {
		return noteQueryFrontmatter{}
	}

	rest := content[len(fmStart):]

	frontmatter, _, ok := strings.Cut(rest, fmEnd)
	if !ok {
		return noteQueryFrontmatter{}
	}

	var doc noteQueryFrontmatter

	err := yaml.Unmarshal([]byte(frontmatter), &doc)
	if err != nil {
		return noteQueryFrontmatter{}
	}

	return doc
}

// topDeliveredNotes returns the first n non-chunk, non-recent, non-ride_along
// items from resolved. These are the delivered notes whose vocab: frontmatter
// terms seed the tag-match nomination pool.
func topDeliveredNotes(resolved []resolvedItem, n int) []resolvedItem {
	out := make([]resolvedItem, 0, n)

	for _, item := range resolved {
		if len(out) >= n {
			break
		}

		if item.kind == chunkItemKind {
			continue
		}

		if slices.Contains(item.provenances, provenanceRecent) {
			continue
		}

		if slices.Contains(item.provenances, ProvenanceRideAlong) {
			continue
		}

		out = append(out, item)
	}

	return out
}
