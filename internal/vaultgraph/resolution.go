package vaultgraph

// UnresolvedLink is an authored wikilink whose target does not resolve to a
// note in the vault — exactly the edges BuildGraph silently drops.
type UnresolvedLink struct {
	Source string // basename of the linking note
	Target string // the unresolved wikilink target (e.g. a bare Luhmann id)
}

// UnresolvedTargets returns every authored wikilink whose target is neither a
// self-link nor an existing note basename. The G0 invariant ("every authored
// link resolves") is broken when this is non-empty — which is how a vault that
// writes bare-id links (`[[105]]`) against a basename-keyed resolver loses most
// of its edges.
func UnresolvedTargets(notes []Note) []UnresolvedLink {
	byName := make(map[string]struct{}, len(notes))
	for _, note := range notes {
		byName[note.Basename] = struct{}{}
	}

	unresolved := make([]UnresolvedLink, 0)

	for _, note := range notes {
		for _, target := range note.Outgoing {
			if target == note.Basename {
				continue
			}

			if _, ok := byName[target]; ok {
				continue
			}

			unresolved = append(unresolved, UnresolvedLink{Source: note.Basename, Target: target})
		}
	}

	return unresolved
}
