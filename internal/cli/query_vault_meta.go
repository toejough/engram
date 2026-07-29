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
// notes, for use by explore-half sampling (vocab-tagged membership) and
// supersession ride-along.
// Fields use exported names for testability via the exported type alias.
type AllVaultNotesMeta struct {
	// TermIndex maps a vocab term name to the notes carrying that term
	// (vocab/<term> tags), each carrying its BodyVector for centroid-proximal
	// selection. Definitions carry a display-only self-tag and CAN appear in
	// TermIndex; exclusion from explore sampling is enforced downstream via
	// isVocabDefinitionNote. Qa-question notes are excluded via
	// isQueryExcludedKind at index-build time.
	TermIndex map[string][]VaultTermMember
	// SupersedesInverse maps a superseded-note basename to the superseder entries,
	// built from every note's supersedes: frontmatter block.
	SupersedesInverse SupersedesInverse
	// ContentByBasename stores loaded note content keyed by basename (for ride-along
	// superseder insertion — the superseder may not be in the matched set).
	ContentByBasename map[string]string
}

// VaultTermMember holds a vault note carrying a vocab/<term> tag, eligible
// for explore-half sampling under that term. Fields use exported names so
// cli_test can build fixtures via the exported type alias without needing
// unexported accessors.
type VaultTermMember struct {
	// NotePath is the vault-relative path of the note (e.g. "1aa.note.md").
	NotePath string
	// Content is the note's raw text (wikilinks stripped) used for exclusion
	// checks (isVocabDefinitionNote) and rendering.
	Content string
	// Vector is the note's BodyVector from its compatible sidecar, used for
	// centroid-proximal within-cluster selection (design.md Decision 3).
	Vector []float32
}

// noteQueryFrontmatter is the minimal parsed shape of a note's frontmatter for
// query-integration purposes (vocab tags + supersedes entries). Parsed once per
// note in loadAllVaultNotesMeta and not re-parsed later.
type noteQueryFrontmatter struct {
	Tags       []string          `yaml:"tags"`
	Supersedes []supersedesEntry `yaml:"supersedes"`
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

// loadAllVaultNotesMeta reads every note in hits once, parsing their tags:
// frontmatter list (vocab/ namespace entries) and supersedes: fields. The
// results feed both explore-half sampling (TermIndex, with each member's
// BodyVector attached for centroid-proximal ranking) and supersession
// ride-along (SupersedesInverse + ContentByBasename).
//
// This is a no-op on vaults with no vocab or supersedes data: it always returns
// an AllVaultNotesMeta with initialised (but possibly empty) maps.
func loadAllVaultNotesMeta(
	hits []compatibleSidecar,
	vault string,
	read func(string) ([]byte, error),
) AllVaultNotesMeta {
	result := AllVaultNotesMeta{
		TermIndex:         make(map[string][]VaultTermMember),
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
		terms := vocabTermsFromTags(meta.Tags)

		// Populate TermIndex — qa-question notes are excluded via isQueryExcludedKind;
		// definitions may carry self-tags and appear here, but are filtered
		// downstream by isVocabDefinitionNote checks in explore-sampling helpers.
		if !isQueryExcludedKind(content) && len(terms) > 0 {
			member := VaultTermMember{NotePath: notePath, Content: content, Vector: hit.sidecar.BodyVector}

			for _, term := range terms {
				result.TermIndex[term] = append(result.TermIndex[term], member)
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

// parseNoteQueryFrontmatter extracts the tags: frontmatter list (vocab/
// namespace entries) and supersedes: fields from note content's YAML
// frontmatter. Returns zero-value fields on any parse failure.
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
