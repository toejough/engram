package memory

import "strings"

// TextChunk represents a portion of stripped session text.
type TextChunk struct {
	Text      string
	StartLine int
	EndLine   int
	Index     int
}

// ChunkText splits text into chunks of approximately maxBytes each,
// splitting on line boundaries. Returns empty slice for empty input.
func ChunkText(text string, maxBytes int) []TextChunk {
	if text == "" {
		return nil
	}

	lines := strings.Split(text, "\n")
	totalLines := len(lines)

	// Estimate lines per chunk based on average line length
	avgLineLen := len(text) / totalLines
	if avgLineLen == 0 {
		avgLineLen = 1
	}

	linesPerChunk := max(maxBytes/avgLineLen, 1)

	var chunks []TextChunk

	for i := 0; i < totalLines; i += linesPerChunk {
		end := min(i+linesPerChunk, totalLines)

		chunkLines := lines[i:end]
		chunks = append(chunks, TextChunk{
			Text:      strings.Join(chunkLines, "\n"),
			StartLine: i + 1,
			EndLine:   end,
			Index:     len(chunks),
		})
	}

	return chunks
}
