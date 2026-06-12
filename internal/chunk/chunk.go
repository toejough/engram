// Package chunk splits memory sources (stripped transcripts, markdown docs)
// into embedding-sized chunks for the auto-ingested vector space. Pure string
// logic — no I/O — so ingestion wiring stays at the CLI edge.
package chunk

import (
	"strconv"
	"strings"
)

// minChunkChars filters semantically-empty fragments ("USER: ok") that would
// pollute cosine scoring with meaningless points.
const minChunkChars = 40

// Chunk is one embeddable unit of a source document.
type Chunk struct {
	// Text is the chunk content handed to the embedder.
	Text string
	// Anchor locates the chunk in its source: "turn-N" for transcripts,
	// the heading text for markdown sections.
	Anchor string
}

// Transcript splits stripped USER:/ASSISTANT: text into chunks of roughly
// target chars (merging consecutive turns) and at most maxLen chars
// (splitting oversized turns on line boundaries). Chunks under the noise
// threshold are dropped.
func Transcript(stripped string, target, maxLen int) []Chunk {
	turns := splitTurns(stripped)

	var chunks []Chunk

	var buf strings.Builder

	anchor := ""

	flush := func() {
		text := strings.TrimSpace(buf.String())
		if len(text) >= minChunkChars {
			chunks = append(chunks, splitOversized(text, anchor, maxLen)...)
		}

		buf.Reset()
	}

	for _, turn := range turns {
		if buf.Len() == 0 {
			anchor = turn.anchor
		}

		buf.WriteString(turn.text)
		buf.WriteString("\n")

		if buf.Len() >= target {
			flush()
		}
	}

	flush()

	return chunks
}

// turn is one user turn plus its following assistant lines.
type turn struct {
	text   string
	anchor string
}

// splitTurns groups stripped lines into user-turn units. Lines before the
// first USER: line form their own leading unit.
func splitTurns(stripped string) []turn {
	var turns []turn

	var buf strings.Builder

	userTurns := 0

	flush := func() {
		if buf.Len() == 0 {
			return
		}

		turns = append(turns, turn{text: strings.TrimRight(buf.String(), "\n"), anchor: anchorFor(userTurns)})
		buf.Reset()
	}

	for line := range strings.Lines(stripped) {
		if strings.HasPrefix(line, "USER: ") {
			flush()

			userTurns++
		}

		buf.WriteString(line)

		if !strings.HasSuffix(line, "\n") {
			buf.WriteString("\n")
		}
	}

	flush()

	return turns
}

// anchorFor names a turn unit; turn 0 is content before any USER: line.
func anchorFor(userTurn int) string {
	if userTurn == 0 {
		return "preamble"
	}

	return "turn-" + strconv.Itoa(userTurn)
}

// Markdown splits raw markdown into one chunk per heading section
// (fence-aware), paragraph-splitting any section over maxLen. Chunks under
// the noise threshold are dropped.
func Markdown(raw string, maxLen int) []Chunk {
	var chunks []Chunk

	var buf strings.Builder

	anchor := "preamble"
	inFence := false

	flush := func() {
		text := strings.TrimSpace(buf.String())
		if len(text) >= minChunkChars {
			chunks = append(chunks, splitOversized(text, anchor, maxLen)...)
		}

		buf.Reset()
	}

	for line := range strings.Lines(raw) {
		trimmed := strings.TrimRight(line, "\n")
		if strings.HasPrefix(strings.TrimSpace(trimmed), "```") {
			inFence = !inFence
		}

		if !inFence && isHeading(trimmed) {
			flush()

			anchor = strings.TrimSpace(strings.TrimLeft(trimmed, "# "))
		}

		buf.WriteString(trimmed)
		buf.WriteString("\n")
	}

	flush()

	return chunks
}

// isHeading reports whether a line is an ATX heading (#..###### + space).
func isHeading(line string) bool {
	rest := strings.TrimLeft(line, "#")

	return line != rest && len(line)-len(rest) <= 6 && strings.HasPrefix(rest, " ")
}

// splitOversized breaks text over maxLen into pieces on paragraph (blank
// line) boundaries first, then line boundaries, so every piece embeds within
// the model's input window. All pieces share the section's anchor.
func splitOversized(text, anchor string, maxLen int) []Chunk {
	if len(text) <= maxLen {
		return []Chunk{{Text: text, Anchor: anchor}}
	}

	var pieces []Chunk

	var buf strings.Builder

	emit := func() {
		piece := strings.TrimSpace(buf.String())
		if len(piece) >= minChunkChars {
			pieces = append(pieces, Chunk{Text: piece, Anchor: anchor})
		}

		buf.Reset()
	}

	for _, unit := range splitUnits(text) {
		if buf.Len() > 0 && buf.Len()+len(unit) > maxLen {
			emit()
		}

		buf.WriteString(unit)
		buf.WriteString("\n")
	}

	emit()

	return pieces
}

// splitUnits yields paragraph-or-line units no longer than needed for
// greedy packing: paragraphs normally, individual lines when a paragraph is
// itself enormous.
func splitUnits(text string) []string {
	var units []string

	for para := range strings.SplitSeq(text, "\n\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		units = append(units, strings.Split(para, "\n")...)
	}

	return units
}
