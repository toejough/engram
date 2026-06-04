package cli

import "strings"

// resolveRelationTargets rewrites each "<target>|<rationale>" relation's target
// from a bare Luhmann id to the full note basename (D1, the Obsidian convention)
// so the wikilink resolves in both Obsidian and engram. A target already in
// basename form, or a bare id with no matching note, is left unchanged.
func resolveRelationTargets(relations, basenames []string) []string {
	idToBasename := make(map[string]string, len(basenames))

	for _, basename := range basenames {
		id, _, _ := strings.Cut(basename, ".")
		if id != "" {
			idToBasename[id] = basename
		}
	}

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
