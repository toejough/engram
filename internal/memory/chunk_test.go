// chunk_test.go
package memory_test

import (
	"strings"
	"testing"

	"github.com/toejough/projctl/internal/memory"
)

func TestChunkText_SmallInput(t *testing.T) {
	// Input smaller than chunk size → single chunk
	text := "line 1\nline 2\nline 3"
	chunks := memory.ChunkText(text, 1000)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Text != text {
		t.Errorf("chunk text mismatch")
	}
	if chunks[0].StartLine != 1 || chunks[0].EndLine != 3 {
		t.Errorf("chunk lines: want 1-3, got %d-%d", chunks[0].StartLine, chunks[0].EndLine)
	}
}

func TestChunkText_SplitsOnLineBoundary(t *testing.T) {
	// Build text that requires splitting
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, strings.Repeat("x", 100)) // 100 bytes per line
	}
	text := strings.Join(lines, "\n") // ~10KB total

	chunks := memory.ChunkText(text, 2500) // ~25 lines per chunk
	if len(chunks) < 3 {
		t.Fatalf("expected at least 3 chunks, got %d", len(chunks))
	}

	// Verify no content lost
	var reassembled []string
	for _, c := range chunks {
		reassembled = append(reassembled, c.Text)
	}
	if strings.Join(reassembled, "\n") != text {
		t.Error("reassembled chunks don't match original")
	}

	// Verify line numbers are contiguous
	for i := 1; i < len(chunks); i++ {
		if chunks[i].StartLine != chunks[i-1].EndLine+1 {
			t.Errorf("gap between chunk %d (end %d) and %d (start %d)",
				i-1, chunks[i-1].EndLine, i, chunks[i].StartLine)
		}
	}
}

func TestChunkText_EmptyInput(t *testing.T) {
	chunks := memory.ChunkText("", 1000)
	if len(chunks) != 0 {
		t.Fatalf("expected 0 chunks for empty input, got %d", len(chunks))
	}
}
