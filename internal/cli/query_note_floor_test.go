package cli_test

import (
	"strconv"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

// TestMergePhraseIntoUnion_FloorKeepsRelevantNoteVsManyChunks proves the
// note-floor: a relevance-qualified note (baseScore >= matchRelevanceFloor) must
// survive the per-phrase matchPhraseLimit cap even when more than matchPhraseLimit
// higher-scoring chunks compete. Without the floor the note is evicted by the
// bare top-30 truncation (the measured drowning).
func TestMergePhraseIntoUnion_FloorKeepsRelevantNoteVsManyChunks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const notePath = "vault/relevant-note.md"

	// A relevance-qualified note (baseScore >= 0.25) scoring below every chunk.
	note := cli.ExportNewScoredCandidate(notePath, 0.40, 0.40)

	// 30 chunks, every one scoring above the note — enough to fill matchPhraseLimit alone.
	chunks := make([]cli.ExportScoredChunk, 0, 30)

	for i := range 30 {
		rec := chunk.Record{Source: "/s/x.jsonl", Anchor: "turn-" + strconv.Itoa(i+2)}
		chunks = append(chunks, cli.ExportNewScoredChunk(rec, 0.50+float32(i)*0.005))
	}

	keys := cli.ExportMergePhraseIntoUnion([]cli.ExportScoredCandidate{note}, chunks)

	g.Expect(keys).To(ContainElement(notePath))
}
