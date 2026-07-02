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
	// for assignment to a note. Slice-2 sweep may tune this per-vault.
	DefaultVocabFloor = float32(0.30)
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
// vector, returning the names of the top-2 terms whose score ≥ floor, plus a
// 3rd (the close-3rd rider) if it also qualifies (≥ floor) and is within
// closeThirdRiderMargin cosine of the 2nd. Returns nil when no term qualifies.
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

	// Top-2 always selected (up to however many qualify).
	selectCount := min(topVocabTermCount, len(candidates))

	// Close-3rd rider: 3rd qualifies iff it exists, is ≥ floor (already guaranteed
	// by the filter above), and is within closeThirdRiderMargin of the 2nd.
	if selectCount == 2 && len(candidates) >= 3 {
		if candidates[1].score-candidates[2].score <= closeThirdRiderMargin {
			selectCount = 3
		}
	}

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

// WriteVocabAssignment replaces the `Vocab:` body line and the `vocab:` YAML
// frontmatter list in content with the supplied term names. Idempotency rule:
// the full line and full list are replaced on every call — never appended.
// When terms is empty, both channels are removed entirely.
//
// This is the single writer for both the body-graph channel
// ([[vocab.<term>]] wikilinks) and the Dataview/search channel (frontmatter list).
func WriteVocabAssignment(content string, terms []string) string {
	content = replaceVocabFrontmatterList(content, terms)
	content = replaceVocabBodyLine(content, terms)

	return content
}

// unexported constants.
const (
	// closeThirdRiderMargin is the maximum cosine gap between the 2nd and 3rd
	// qualifying term for the close-3rd rider to activate.
	closeThirdRiderMargin = float32(0.02)
	// topVocabTermCount is the maximum number of top-ranked terms selected
	// before the close-3rd rider check.
	topVocabTermCount = 2
	typeVocab         = "vocab"
	typeVocabIndex    = "vocab-index"
	// vocabBodyMarker is the line-start prefix of a Vocab body line on a member note.
	vocabBodyMarker = "Vocab:"
)

// termScore is the internal working type for scoring terms against a note vector.
type termScore struct {
	term  string
	score float32
}

// isVocabKind reports whether the note content's type field marks it as a vocab
// or vocab-index note. These are filtered from the matched set, note-floor
// reservation, and clustering so they do not surface in recall results.
func isVocabKind(content string) bool {
	kind := kindFromContent(content)
	return kind == typeVocab || kind == typeVocabIndex
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

// renderVocabBodyLine produces the `Vocab: [[vocab.term-a]], [[vocab.term-b]]`
// body line for the graph/Obsidian channel.
func renderVocabBodyLine(terms []string) string {
	links := make([]string, len(terms))

	for i, term := range terms {
		links[i] = "[[vocab." + term + "]]"
	}

	return "Vocab: " + strings.Join(links, ", ")
}

// renderVocabYAMLList produces the inline-list form `vocab: [term-a, term-b]`
// for the frontmatter channel.
func renderVocabYAMLList(terms []string) string {
	return "vocab: [" + strings.Join(terms, ", ") + "]"
}

// replaceVocabBodyLine replaces (or removes) the `Vocab: [[...]]` line in the
// note body. Operates only in the body section (after the frontmatter delimiters).
func replaceVocabBodyLine(content string, terms []string) string {
	const (
		fmStart = "---\n"
		fmEnd   = "\n---\n"
	)

	if !strings.HasPrefix(content, fmStart) {
		return replaceVocabBodyLineInSection(content, terms)
	}

	rest := content[len(fmStart):]

	endIdx := strings.Index(rest, fmEnd)
	if endIdx < 0 {
		return content
	}

	bodyStart := len(fmStart) + endIdx + len(fmEnd)
	head := content[:bodyStart]
	body := content[bodyStart:]

	return head + replaceVocabBodyLineInSection(body, terms)
}

// replaceVocabBodyLineInSection removes existing `Vocab:` lines from section
// and, when terms is non-empty, appends the new `Vocab:` line.
func replaceVocabBodyLineInSection(section string, terms []string) string {
	lines := strings.Split(section, "\n")
	out := make([]string, 0, len(lines))

	for _, line := range lines {
		if strings.HasPrefix(line, vocabBodyMarker) {
			continue
		}

		out = append(out, line)
	}

	// Trim trailing blank lines before re-adding the Vocab line.
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}

	if len(terms) > 0 {
		out = append(out, "", renderVocabBodyLine(terms))
	}

	// Preserve trailing newline.
	result := strings.Join(out, "\n")
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}

	return result
}

// replaceVocabFrontmatterList locates the `vocab:` key in the YAML frontmatter
// block and replaces its value with the supplied terms. When terms is empty
// the key is removed. Operates only inside the `---` delimiters so any
// `vocab:` text in the body is unaffected.
func replaceVocabFrontmatterList(content string, terms []string) string {
	const (
		fmStart = "---\n"
		fmEnd   = "\n---\n"
	)

	if !strings.HasPrefix(content, fmStart) {
		return content
	}

	rest := content[len(fmStart):]

	frontmatter, after, found := strings.Cut(rest, fmEnd)
	if !found {
		return content
	}

	frontmatter = removeYAMLKey(frontmatter, "vocab")

	if len(terms) > 0 {
		frontmatter = strings.TrimRight(frontmatter, "\n") + "\n" + renderVocabYAMLList(terms)
	}

	return fmStart + frontmatter + fmEnd + after
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
