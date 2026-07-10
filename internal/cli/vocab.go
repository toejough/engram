package cli

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/embed"
)

// Exported constants.
const (
	// DefaultVocabFloor is the minimum cosine similarity for a term to qualify
	// for assignment to a note. Set by the Slice-2 centroid two-pass sweep:
	// twopass|0.35|K3 = 56.2% recovery @ median pool 31.0 on the 48-case miss
	// population — PASS against the re-anchored gate (≥54.2% @ pool ≤40).
	DefaultVocabFloor = float32(0.35)
)

// TermWithVector pairs a vocab term name with its embedding vector loaded from
// the term note's sidecar. Used by AssignVocabTerms at write time.
type TermWithVector struct {
	Term   string
	Vector []float32
}

// VocabFrontmatter is the parsed frontmatter of a vocab term note
// (type: vocab). Vocab notes are written by `engram vocab bootstrap`
// (slice 2), not by learn/amend.
type VocabFrontmatter struct {
	Type         string `yaml:"type"`
	Term         string `yaml:"term"`
	Description  string `yaml:"description"`
	VocabVersion string `yaml:"vocab_version,omitempty"`
	Created      string `yaml:"created,omitempty"`
}

// AssignVocabTerms computes cosine similarity between bodyVec and each term's
// vector, returning the top-3 terms whose score ≥ floor (plain top-3, no
// close-3rd rider). Config from the Slice-2 centroid two-pass sweep: within
// the shipped twopass arm K3 beats K2+rider at every floor (56.2% vs ≤50.0%
// recovery) and twopass|0.35|K3 passes the re-anchored gate (≥54.2% @ pool
// ≤40) at median pool 31.0. Returns nil when no term qualifies.
func AssignVocabTerms(bodyVec []float32, terms []TermWithVector, floor float32) []string {
	if len(bodyVec) == 0 || len(terms) == 0 {
		return nil
	}

	candidates := make([]termScore, 0, len(terms))

	for _, term := range terms {
		sim := embed.Cosine(bodyVec, term.Vector)
		if sim >= floor {
			candidates = append(candidates, termScore{term: term.Term, score: sim})
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	sortTermScores(candidates)

	selectCount := min(topVocabTermCount, len(candidates))

	result := make([]string, selectCount)
	for i := range selectCount {
		result[i] = candidates[i].term
	}

	return result
}

// ParseVocabFrontmatter unmarshals the YAML frontmatter block bytes of a vocab
// note. The caller extracts the frontmatter bytes before calling this.
func ParseVocabFrontmatter(frontmatterBytes []byte) (VocabFrontmatter, error) {
	var doc VocabFrontmatter

	err := yaml.Unmarshal(frontmatterBytes, &doc)
	if err != nil {
		return VocabFrontmatter{}, fmt.Errorf("parsing vocab frontmatter: %w", err)
	}

	return doc, nil
}

// WriteVocabAssignment rewrites the vocab/<term> namespace of the note's
// tags: frontmatter list to exactly terms, preserving all non-vocab tags and
// their order. It also strips the legacy vocab: frontmatter key and Vocab:
// body line when present (migration-by-touch). Idempotency rule: the vocab
// namespace is replaced on every call — never appended. When terms is empty,
// the vocab namespace is removed; an emptied tags: key is removed entirely.
func WriteVocabAssignment(content string, terms []string) string {
	frontmatter, rest, ok := splitFrontmatterAndBody(content)
	if !ok {
		return content
	}

	kept := nonVocabTags(parseTagsFromFrontmatter(frontmatter))

	merged := make([]string, 0, len(kept)+len(terms))
	merged = append(merged, kept...)

	for _, term := range terms {
		merged = append(merged, vocabTagPrefix+term)
	}

	// Compute insertion index before removals shift line positions. When tags:
	// exists, we'll remove vocab: first (shifting tags: up if needed), then
	// recompute the tags: index on the vocab-free text, then remove tags: at
	// that index; removal at insertAt places insertYAMLBlock at exactly that
	// position. When tags: is absent, insertAt is the vocab: index (or -1 if
	// vocab: is also absent, yielding append-at-end behavior).
	var insertAt int

	if yamlKeyLineIndex(frontmatter, "tags") >= 0 {
		frontmatter = removeYAMLKey(frontmatter, "vocab") // may shift tags up; that's fine
		insertAt = yamlKeyLineIndex(frontmatter, "tags")  // recompute on the vocab-free text
		frontmatter = removeYAMLKey(frontmatter, "tags")  // removal at insertAt shifts followers into insertAt
	} else {
		insertAt = yamlKeyLineIndex(frontmatter, "vocab") // -1 when absent → append
		frontmatter = removeYAMLKey(frontmatter, "vocab")
	}

	if len(merged) > 0 {
		frontmatter = insertYAMLBlock(frontmatter, renderTagsBlock(merged), insertAt)
	}

	return fmStart + frontmatter + fmEnd + removeVocabBodyLine(rest)
}

// unexported constants.
const (
	fmEnd = "\n---\n"
	// fmStart/fmEnd delimit YAML frontmatter in a note's raw content.
	fmStart = "---\n"
	// topVocabTermCount is the maximum number of top-ranked terms selected
	// (plain top-3 — the sweep-chosen K; see AssignVocabTerms).
	topVocabTermCount = 3
	typeVocab         = "vocab"
	typeVocabIndex    = "vocab-index"
	// vocabBodyMarker is the line-start prefix of a Vocab body line on a member
	// note. Aliased to the embed marker so the writer's line matching and the
	// BodyText/ContentHash exclusion can never drift apart.
	vocabBodyMarker = embed.VocabBodyMarker
	// vocabTagBlockIndent is the 4-space "- " block-sequence item prefix,
	// byte-identical to yaml.v3's default indent (matches the #674 learn
	// renderer's tags: output; see renderTagsBlock).
	vocabTagBlockIndent = "    - "
	vocabTagPrefix      = "vocab/"
)

// termScore is the internal working type for scoring terms against a note vector.
type termScore struct {
	term  string
	score float32
}

// applyVocabAssignmentCore is the shared body of the write-site assignment
// helpers (learn/amend/resituate) — one copy of the load-assign-write flow;
// each site's wrapper plucks its deps fields and forwards. Silent no-op when
// any required func is nil, terms are absent, or the sidecar is unreadable.
func applyVocabAssignmentCore(
	loadTermVectors func(string) ([]TermWithVector, error),
	read func(string) ([]byte, error),
	write func(string, []byte) error,
	logWarning func(string, ...any),
	vault, notePath, content, site string,
) {
	if loadTermVectors == nil || read == nil || write == nil {
		return
	}

	terms, termsErr := loadTermVectors(vault)
	if termsErr != nil || len(terms) == 0 {
		return
	}

	bodyVec, ok := loadBodyVectorForNote(read, notePath)
	if !ok {
		return
	}

	assigned := AssignVocabTerms(bodyVec, terms, DefaultVocabFloor)
	updated := WriteVocabAssignment(content, assigned)

	if updated == content {
		return
	}

	writeErr := write(notePath, []byte(updated))
	if writeErr != nil && logWarning != nil {
		logWarning("%s: vocab assignment write failed for %s: %v", site, notePath, writeErr)
	}
}

// insertYAMLBlock inserts block at the given line index (append at end when
// index is -1 or out of range).
func insertYAMLBlock(frontmatter, block string, atLine int) string {
	lines := strings.Split(frontmatter, "\n")

	if atLine < 0 || atLine > len(lines) {
		atLine = len(lines)
	}

	result := make([]string, 0, len(lines)+1)
	result = append(result, lines[:atLine]...)
	result = append(result, block)
	result = append(result, lines[atLine:]...)

	return strings.Join(result, "\n")
}

// isVocabKind reports whether the note content's type field marks it as a vocab
// or vocab-index note. These are filtered from the matched set, note-floor
// reservation, and clustering so they do not surface in recall results.
func isVocabKind(content string) bool {
	kind := kindFromContent(content)
	return kind == typeVocab || kind == typeVocabIndex
}

// loadBodyVectorForNote reads the sidecar of notePath via readFn and returns
// its BodyVector. Returns nil, false when the sidecar is absent, unreadable,
// fails to unmarshal, or has an empty BodyVector.
func loadBodyVectorForNote(readFn func(string) ([]byte, error), notePath string) ([]float32, bool) {
	sidecarData, sidecarErr := readFn(embed.SidecarPath(notePath))
	if sidecarErr != nil {
		return nil, false
	}

	sidecar, unmarshalErr := embed.UnmarshalSidecar(sidecarData)
	if unmarshalErr != nil || len(sidecar.BodyVector) == 0 {
		return nil, false
	}

	return sidecar.BodyVector, true
}

// nonVocabTags filters out entries in the vocab namespace (vocab/<term>)
// AND the bare "vocab" definition marker, preserving order.
func nonVocabTags(tags []string) []string {
	kept := make([]string, 0, len(tags))

	for _, tag := range tags {
		if tag == typeVocab || strings.HasPrefix(tag, vocabTagPrefix) {
			continue
		}

		kept = append(kept, tag)
	}

	return kept
}

// parseTagsFromFrontmatter returns the tags: list values, handling both
// block style ("tags:\n    - a") and inline style ("tags: [a, b]").
// Absent key or empty list returns nil.
func parseTagsFromFrontmatter(frontmatter string) []string {
	var doc struct {
		Tags []string `yaml:"tags"`
	}

	unmarshalErr := yaml.Unmarshal([]byte(frontmatter), &doc)
	if unmarshalErr != nil || len(doc.Tags) == 0 {
		return nil
	}

	return doc.Tags
}

// removeVocabBodyLine strips the Vocab: machine line (and one preceding
// blank line) from the body; unchanged when absent.
func removeVocabBodyLine(body string) string {
	lines := strings.Split(body, "\n")

	idx := -1

	for i, line := range lines {
		if strings.HasPrefix(line, vocabBodyMarker) {
			idx = i

			break
		}
	}

	if idx < 0 {
		return body
	}

	out := make([]string, 0, len(lines)-1)
	out = append(out, lines[:idx]...)

	// Drop exactly one preceding blank line, if present.
	if len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}

	out = append(out, lines[idx+1:]...)

	return strings.Join(out, "\n")
}

// removeYAMLKey removes a top-level YAML key and its value (scalar or
// sequence block) from a raw frontmatter string. Only removes the first
// occurrence. Handles both scalar (`key: value`) and sequence block forms.
func removeYAMLKey(frontmatter, key string) string {
	keyPrefix := key + ":"

	lines := strings.Split(frontmatter, "\n")

	start := -1

	for i, line := range lines {
		if strings.HasPrefix(line, keyPrefix) {
			start = i
			break
		}
	}

	if start < 0 {
		return frontmatter
	}

	// Determine end: the block ends at the next non-continuation line.
	// Continuation lines are blank or start with whitespace (YAML sequence items
	// use "  - " so they start with spaces).
	end := start + 1

	for end < len(lines) {
		line := lines[end]
		if strings.TrimSpace(line) == "" ||
			strings.HasPrefix(line, " ") ||
			strings.HasPrefix(line, "\t") {
			end++
		} else {
			break
		}
	}

	result := make([]string, 0, len(lines))
	result = append(result, lines[:start]...)
	result = append(result, lines[end:]...)

	return strings.Join(result, "\n")
}

// renderTagsBlock renders the block-style list, 4-space indent:
// "tags:\n    - a\n    - b" (no trailing newline).
func renderTagsBlock(tags []string) string {
	lines := make([]string, 0, len(tags)+1)
	lines = append(lines, "tags:")

	for _, tag := range tags {
		lines = append(lines, vocabTagBlockIndent+tag)
	}

	return strings.Join(lines, "\n")
}

// sortTermScores sorts a termScore slice descending by score, with term name
// ascending as the tie-breaker for determinism. Uses insertion sort — slices
// are small (≤ 25 term-note names).
func sortTermScores(candidates []termScore) {
	for i := 1; i < len(candidates); i++ {
		for j := i; j > 0; j-- {
			prevScore := candidates[j-1].score
			currScore := candidates[j].score

			if prevScore > currScore || (prevScore == currScore && candidates[j-1].term < candidates[j].term) {
				break
			}

			candidates[j-1], candidates[j] = candidates[j], candidates[j-1]
		}
	}
}

// splitFrontmatterAndBody cuts content into (frontmatter-without-delims,
// body-after-closing-delim, ok). ok is false when content has no leading
// frontmatter block.
func splitFrontmatterAndBody(content string) (string, string, bool) {
	if !strings.HasPrefix(content, fmStart) {
		return "", "", false
	}

	frontmatter, body, found := strings.Cut(content[len(fmStart):], fmEnd)
	if !found {
		return "", "", false
	}

	return frontmatter, body, true
}

// vocabTermsFromTags returns the terms of the vocab namespace entries
// (prefix stripped), preserving order. The bare "vocab" tag is not a term.
func vocabTermsFromTags(tags []string) []string {
	terms := make([]string, 0, len(tags))

	for _, tag := range tags {
		term, ok := strings.CutPrefix(tag, vocabTagPrefix)
		if !ok {
			continue
		}

		terms = append(terms, term)
	}

	return terms
}

// yamlKeyLineIndex returns the line index of a top-level key, or -1.
func yamlKeyLineIndex(frontmatter, key string) int {
	keyPrefix := key + ":"

	for i, line := range strings.Split(frontmatter, "\n") {
		if strings.HasPrefix(line, keyPrefix) {
			return i
		}
	}

	return -1
}
