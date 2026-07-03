package cli

import (
	"strings"
)

// unexported constants.
const (
	qaAnswerSuffix   = ".a.md"
	qaNotePrefix     = "qa."
	qaQuestionSuffix = ".q.md"
	typeQAQuestion   = "qa-question"
)

// countQAPairs counts vault files where both the .q.md and matching .a.md exist.
// Pure read-time scan; no new state.
func countQAPairs(names []string) int {
	nameSet := make(map[string]struct{}, len(names))
	for _, name := range names {
		nameSet[name] = struct{}{}
	}

	count := 0

	for _, name := range names {
		if !isQAQuestionFilename(name) {
			continue
		}
		// Derive expected A filename: replace .q.md suffix with .a.md.
		base := strings.TrimSuffix(name, qaQuestionSuffix)
		aName := base + qaAnswerSuffix

		if _, ok := nameSet[aName]; ok {
			count++
		}
	}

	return count
}

// isQAQuestionFilename reports whether a filename is a QA question note
// (prefix "qa." AND suffix ".q.md").
func isQAQuestionFilename(name string) bool {
	return strings.HasPrefix(name, qaNotePrefix) && strings.HasSuffix(name, qaQuestionSuffix)
}

// isQAQuestionKind reports whether the note's frontmatter type is qa-question.
func isQAQuestionKind(content string) bool {
	return kindFromContent(content) == typeQAQuestion
}

// isQueryExcludedKind reports whether a note should be excluded from the
// query pipeline's main matched set. Excludes vocab kinds AND qa-question.
// qa-answer COMPETES in the main set (D5′ — A notes are synthesis notes).
func isQueryExcludedKind(content string) bool {
	return isVocabKind(content) || isQAQuestionKind(content)
}
