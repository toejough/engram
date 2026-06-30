package cli

import "strings"

// normalizeNoteRef canonicalizes a user-supplied note ref to a basename or bare
// id: trims surrounding whitespace, strips [[ ]] wikilink brackets and an
// optional |display segment, then drops a trailing .md extension. It is shared
// by `show` (RunShow) and `findNote` so both accept the full form list:
// full basename, [[wikilink]], trailing .md, or bare Luhmann id.
func normalizeNoteRef(ref string) string {
	ref = strings.TrimSpace(ref)
	ref = strings.TrimPrefix(ref, "[[")
	ref = strings.TrimSuffix(ref, "]]")

	if pipe := strings.IndexByte(ref, '|'); pipe >= 0 {
		ref = ref[:pipe]
	}

	ref = strings.TrimSpace(ref)
	ref = strings.TrimSuffix(ref, ".md")

	return strings.TrimSpace(ref)
}
