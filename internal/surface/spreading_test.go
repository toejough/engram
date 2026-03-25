package surface_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

// TestComputeSpreading_LinkReaderError skips matchers that return errors.
func TestComputeSpreading_LinkReaderError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reader := &fakeSpreadingLinkReaderError{}

	bm25Matches := map[string]float64{"match1.toml": 0.9}

	result := surface.ExportComputeSpreading(bm25Matches, reader)

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result).To(BeEmpty())
}

// TestComputeSpreading_NilLinkReader returns nil when no link reader is provided.
func TestComputeSpreading_NilLinkReader(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	bm25Matches := map[string]float64{"mem1.toml": 0.8}
	result := surface.ExportComputeSpreading(bm25Matches, nil)

	g.Expect(result).To(BeNil())
}

// TestComputeSpreading_NormalizesbyLinkerCount verifies that spreading scores are
// normalized by the number of BM25 matchers that link to each neighbor.
func TestComputeSpreading_NormalizesbyLinkerCount(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Two BM25 matches both link to "neighbor.toml".
	// match1 (score=0.8) links with weight=1.0 → contributes 0.8
	// match2 (score=0.4) links with weight=0.5 → contributes 0.2
	// raw sum = 1.0, linkerCount = 2 → normalized = 0.5
	reader := &fakeSpreadingLinkReader{
		links: map[string][]surface.LinkGraphLink{
			"match1.toml": {{Target: "neighbor.toml", Weight: 1.0}},
			"match2.toml": {{Target: "neighbor.toml", Weight: 0.5}},
		},
	}

	bm25Matches := map[string]float64{
		"match1.toml": 0.8,
		"match2.toml": 0.4,
	}

	result := surface.ExportComputeSpreading(bm25Matches, reader)

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	// match1: 0.8 * 1.0 = 0.8
	// match2: 0.4 * 0.5 = 0.2
	// raw sum for neighbor = 1.0, divided by 2 linkers = 0.5
	g.Expect(result["neighbor.toml"]).To(BeNumerically("~", 0.5, 0.001))
}

// TestComputeSpreading_TwoMatchesDifferentNeighbors verifies that distinct neighbors get
// independent spreading scores based on their respective linker BM25 × weight.
func TestComputeSpreading_TwoMatchesDifferentNeighbors(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// match1 links to neighborA, match2 links to neighborB — no overlap.
	reader := &fakeSpreadingLinkReader{
		links: map[string][]surface.LinkGraphLink{
			"match1.toml": {{Target: "neighborA.toml", Weight: 0.8}},
			"match2.toml": {{Target: "neighborB.toml", Weight: 0.5}},
		},
	}

	bm25Matches := map[string]float64{
		"match1.toml": 1.0,
		"match2.toml": 0.6,
	}

	result := surface.ExportComputeSpreading(bm25Matches, reader)

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	// neighborA: 1.0 * 0.8 / 1 = 0.8
	g.Expect(result["neighborA.toml"]).To(BeNumerically("~", 0.8, 0.001))
	// neighborB: 0.6 * 0.5 / 1 = 0.3
	g.Expect(result["neighborB.toml"]).To(BeNumerically("~", 0.3, 0.001))
}

// TestPromptMode_SpreadingNeighborAppearsInResults verifies that a memory with 0 BM25
// but positive spreading score appears in results when BM25 matchers link to it.
// BM25 requires >= 3 docs for positive IDF (with 2 docs, a term in 1 doc gets IDF=0).
func TestPromptMode_SpreadingNeighborAppearsInResults(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Now()

	// "xyzbuildsystem" matches query uniquely (appears in 1 of 3 docs → positive IDF).
	// "abcspread" does NOT match the query but is linked from the BM25 match.
	// "defunrelated" is a third document to ensure positive IDF for xyzbuildsystem.
	memories := []*memory.Stored{
		{
			FilePath:  "bm25mem.toml",
			Title:     "xyzbuildsystem",
			Principle: "use xyzbuildsystem for all builds",
			UpdatedAt: now,
		},
		{
			FilePath:  "spreadmem.toml",
			Title:     "abcspread memory",
			Principle: "abcspread linked memory principle",
			UpdatedAt: now,
		},
		{
			FilePath:  "defunrelated.toml",
			Title:     "defunrelated other",
			Principle: "defunrelated something else entirely",
			UpdatedAt: now,
		},
	}

	// bm25mem links to spreadmem with weight 1.0
	reader := &fakeSpreadingLinkReader{
		links: map[string][]surface.LinkGraphLink{
			"bm25mem.toml": {{Target: "spreadmem.toml", Weight: 1.0}},
		},
	}

	surfacer := surface.New(
		&fakeRetriever{memories: memories},
		surface.WithLinkReader(reader),
	)

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "xyzbuildsystem for all builds",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	// Both the BM25 match and the spreading neighbor should appear.
	g.Expect(output).To(ContainSubstring("bm25mem"))
	g.Expect(output).To(ContainSubstring("spreadmem"))
}

// TestPromptMode_SpreadingWithNoLinkReader returns only BM25 matches (no spreading).
func TestPromptMode_SpreadingWithNoLinkReader(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Now()

	// Three docs needed for positive IDF (see TestPromptMode_SpreadingNeighborAppearsInResults).
	memories := []*memory.Stored{
		{
			FilePath:  "bm25mem.toml",
			Title:     "xyzbuildsystem",
			Principle: "use xyzbuildsystem for all builds",
			UpdatedAt: now,
		},
		{
			FilePath:  "spreadmem.toml",
			Title:     "abcspread memory",
			Principle: "abcspread linked memory principle",
			UpdatedAt: now,
		},
		{
			FilePath:  "defunrelated.toml",
			Title:     "defunrelated other",
			Principle: "defunrelated something else entirely",
			UpdatedAt: now,
		},
	}

	// No WithLinkReader — spreading should be skipped entirely.
	surfacer := surface.New(
		&fakeRetriever{memories: memories},
	)

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "xyzbuildsystem for all builds",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	g.Expect(output).To(ContainSubstring("bm25mem"))
	// spreadmem should NOT appear — it has no BM25 match and no spreading without a reader.
	g.Expect(strings.Contains(output, "spreadmem")).To(BeFalse())
}

// fakeSpreadingLinkReader returns links keyed by source path.
type fakeSpreadingLinkReader struct {
	links map[string][]surface.LinkGraphLink
}

func (f *fakeSpreadingLinkReader) GetEntryLinks(path string) ([]surface.LinkGraphLink, error) {
	return f.links[path], nil
}

// fakeSpreadingLinkReaderError always returns an error.
type fakeSpreadingLinkReaderError struct{}

func (f *fakeSpreadingLinkReaderError) GetEntryLinks(_ string) ([]surface.LinkGraphLink, error) {
	return nil, errors.New("read error")
}
