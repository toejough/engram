package cli

import (
	"strings"

	"github.com/toejough/engram/internal/embed"
)

// unexported constants.
const (
	// relatedSectionMarker is derived from embed.RelatedSectionMarker so the
	// rendering convention (here) and the body-hash strip (in embed) share one
	// source of truth and cannot drift.
	relatedSectionMarker = embed.RelatedSectionMarker
)

// indexBasenamesByID maps each note's leading Luhmann id (the segment before the
// first dot of its basename) to the full basename.
func indexBasenamesByID(basenames []string) map[string]string {
	idToBasename := make(map[string]string, len(basenames))

	for _, basename := range basenames {
		id, _, _ := strings.Cut(basename, ".")
		if id != "" {
			idToBasename[id] = basename
		}
	}

	return idToBasename
}

// migrateRelationLinks rewrites bare-id wikilinks ([[105]]) to full basenames
// ([[105.<date>.<slug>]]) within a note's "Related to:" section ONLY — never in
// the transcript or formula above it — so verbatim content is left untouched
// (D2/G5). Returns the rewritten body and the number of links changed.
func migrateRelationLinks(body string, idToBasename map[string]string) (string, int) {
	idx := strings.LastIndex(body, relatedSectionMarker)
	if idx == -1 {
		return body, 0
	}

	head, tail := body[:idx], body[idx:]
	count := 0

	newTail := wikilinkRE.ReplaceAllStringFunc(tail, func(match string) string {
		sub := wikilinkRE.FindStringSubmatch(match)

		basename, ok := idToBasename[sub[1]]
		if !ok {
			return match
		}

		count++

		if sub[2] != "" {
			return "[[" + basename + "|" + sub[2] + "]]"
		}

		return "[[" + basename + "]]"
	})

	return head + newTail, count
}

// resolveRelationTargets rewrites each "<target>|<rationale>" relation's target
// from a bare Luhmann id to the full note basename (D1, the Obsidian convention)
// so the wikilink resolves in both Obsidian and engram. A target already in
// basename form, or a bare id with no matching note, is left unchanged.
func resolveRelationTargets(relations, basenames []string) []string {
	idToBasename := indexBasenamesByID(basenames)

	resolved := make([]string, len(relations))

	for i, relation := range relations {
		target, rationale, hasRationale := strings.Cut(relation, "|")

		if basename, ok := idToBasename[strings.TrimSpace(target)]; ok {
			target = basename
		}

		if hasRationale {
			resolved[i] = target + "|" + rationale
		} else {
			resolved[i] = target
		}
	}

	return resolved
}
