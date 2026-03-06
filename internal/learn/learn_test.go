package learn_test

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/learn"
	"engram/internal/memory"
)

// Extract error — pipeline returns error
func TestExtractError_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	extractor := &fakeExtractor{err: errors.New("LLM unavailable")}
	retriever := &fakeRetriever{}
	deduplicator := &fakeDeduplicator{}
	writer := &fakeWriter{}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")
	result, err := learner.Run(context.Background(), "some transcript")

	g.Expect(err).To(MatchError(ContainSubstring("LLM unavailable")))
	g.Expect(result).To(BeNil())
}

// ListMemories error — pipeline returns error
func TestListMemoriesError_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []memory.CandidateLearning{
		{Title: "Use targ", Content: "use targ for builds", FilenameSummary: "use-targ"},
	}

	extractor := &fakeExtractor{candidates: candidates}
	retriever := &fakeRetriever{err: errors.New("disk read failed")}
	deduplicator := &fakeDeduplicator{}
	writer := &fakeWriter{}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")
	result, err := learner.Run(context.Background(), "some transcript")

	g.Expect(err).To(MatchError(ContainSubstring("disk read failed")))
	g.Expect(result).To(BeNil())
}

// T-57: Full pipeline — extract → dedup → write returns file paths
func TestT57_FullPipeline_ExtractDedupWrite(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	recorder := &callRecord{}

	candidates := []memory.CandidateLearning{
		{Title: "Use targ", Content: "use targ for builds", FilenameSummary: "use-targ"},
		{Title: "No magic numbers", Content: "name constants", FilenameSummary: "no-magic-numbers"},
	}

	surviving := []memory.CandidateLearning{
		{Title: "Use targ", Content: "use targ for builds", FilenameSummary: "use-targ"},
	}

	extractor := &fakeExtractor{candidates: candidates, record: recorder}
	retriever := &fakeRetriever{memories: []*memory.Stored{}, record: recorder}
	deduplicator := &fakeDeduplicator{surviving: surviving, record: recorder}
	writer := &fakeWriter{
		paths: map[string]string{
			"use-targ": "/tmp/memories/use-targ.toml",
		},
		record: recorder,
	}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")
	result, err := learner.Run(context.Background(), "some transcript")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.CreatedPaths).To(ConsistOf("/tmp/memories/use-targ.toml"))
	g.Expect(result.SkippedCount).To(Equal(1))
	g.Expect(recorder.calls).To(Equal([]string{"extract", "list", "filter", "write"}))
}

// T-58: No learnings extracted — pipeline short-circuits
func TestT58_NoLearningsExtracted_ShortCircuits(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	extractor := &fakeExtractor{candidates: []memory.CandidateLearning{}}
	retriever := &fakeRetriever{}
	deduplicator := &fakeDeduplicator{}
	writer := &fakeWriter{}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")
	result, err := learner.Run(context.Background(), "some transcript")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.CreatedPaths).To(BeEmpty())
	g.Expect(result.SkippedCount).To(Equal(0))
	g.Expect(retriever.called).To(BeFalse())
	g.Expect(deduplicator.called).To(BeFalse())
	g.Expect(writer.called).To(BeFalse())
}

// T-59: All candidates filtered — no files written
func TestT59_AllCandidatesFiltered_NoFilesWritten(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []memory.CandidateLearning{
		{Title: "Use targ", Content: "use targ for builds", FilenameSummary: "use-targ"},
		{Title: "No magic numbers", Content: "name constants", FilenameSummary: "no-magic-numbers"},
	}

	extractor := &fakeExtractor{candidates: candidates}
	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	// Deduplicator returns empty slice — all filtered
	deduplicator := &fakeDeduplicator{surviving: []memory.CandidateLearning{}}
	writer := &fakeWriter{}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")
	result, err := learner.Run(context.Background(), "some transcript")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.CreatedPaths).To(BeEmpty())
	g.Expect(result.SkippedCount).To(Equal(2))
	g.Expect(writer.called).To(BeFalse())
}

// T-60: Written memories use tier from extraction (not hardcoded "C")
func TestT60_WrittenMemories_UseTierFromExtraction(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []memory.CandidateLearning{
		{Tier: "A", Title: "Use targ", Content: "use targ for builds", FilenameSummary: "use-targ"},
		{
			Tier:            "B",
			Title:           "DI pattern",
			Content:         "use DI everywhere",
			FilenameSummary: "di-pattern",
		},
		{
			Tier:            "C",
			Title:           "Uses SQLite",
			Content:         "project uses sqlite",
			FilenameSummary: "uses-sqlite",
		},
	}

	extractor := &fakeExtractor{candidates: candidates}
	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	deduplicator := &fakeDeduplicator{surviving: candidates}
	writer := &fakeWriter{
		paths: map[string]string{
			"use-targ":    "/tmp/memories/use-targ.toml",
			"di-pattern":  "/tmp/memories/di-pattern.toml",
			"uses-sqlite": "/tmp/memories/uses-sqlite.toml",
		},
	}

	before := time.Now()
	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")
	result, err := learner.Run(context.Background(), "some transcript")
	after := time.Now()

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.CreatedPaths).To(HaveLen(3))

	g.Expect(writer.received).To(HaveLen(3))

	// Tier A candidate → Confidence "A".
	g.Expect(writer.received[0].Confidence).To(Equal("A"))
	// Tier B candidate → Confidence "B".
	g.Expect(writer.received[1].Confidence).To(Equal("B"))
	// Tier C candidate → Confidence "C".
	g.Expect(writer.received[2].Confidence).To(Equal("C"))

	// Timestamps still valid.
	g.Expect(writer.received[0].CreatedAt).To(BeTemporally(">=", before))
	g.Expect(writer.received[0].CreatedAt).To(BeTemporally("<=", after))
	g.Expect(writer.received[0].UpdatedAt).To(BeTemporally(">=", before))
	g.Expect(writer.received[0].UpdatedAt).To(BeTemporally("<=", after))
}

// Write error — pipeline returns error
func TestWriteError_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []memory.CandidateLearning{
		{Title: "Use targ", Content: "use targ for builds", FilenameSummary: "use-targ"},
	}

	extractor := &fakeExtractor{candidates: candidates}
	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	deduplicator := &fakeDeduplicator{surviving: candidates}
	writer := &fakeWriter{err: errors.New("disk write failed")}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")
	result, err := learner.Run(context.Background(), "some transcript")

	g.Expect(err).To(MatchError(ContainSubstring("disk write failed")))
	g.Expect(result).To(BeNil())
}

// callRecord tracks which pipeline stages were called and in what order.
type callRecord struct {
	calls []string
}

func (r *callRecord) record(name string) {
	r.calls = append(r.calls, name)
}

// fakeDeduplicator is a test double for learn.Deduplicator.
type fakeDeduplicator struct {
	surviving []memory.CandidateLearning
	called    bool
	record    *callRecord
}

func (f *fakeDeduplicator) Filter(
	_ []memory.CandidateLearning,
	_ []*memory.Stored,
) []memory.CandidateLearning {
	f.called = true

	if f.record != nil {
		f.record.record("filter")
	}

	return f.surviving
}

// fakeExtractor is a test double for learn.TranscriptExtractor.
type fakeExtractor struct {
	candidates []memory.CandidateLearning
	err        error
	record     *callRecord
}

func (f *fakeExtractor) Extract(_ context.Context, _ string) ([]memory.CandidateLearning, error) {
	if f.record != nil {
		f.record.record("extract")
	}

	return f.candidates, f.err
}

// fakeRetriever is a test double for learn.MemoryRetriever.
type fakeRetriever struct {
	memories []*memory.Stored
	err      error
	called   bool
	record   *callRecord
}

func (f *fakeRetriever) ListMemories(_ context.Context, _ string) ([]*memory.Stored, error) {
	f.called = true

	if f.record != nil {
		f.record.record("list")
	}

	return f.memories, f.err
}

// fakeWriter is a test double for learn.MemoryWriter.
type fakeWriter struct {
	paths    map[string]string // keyed by FilenameSummary
	err      error
	called   bool
	received []*memory.Enriched
	record   *callRecord
}

func (f *fakeWriter) Write(mem *memory.Enriched, _ string) (string, error) {
	f.called = true
	f.received = append(f.received, mem)

	if f.record != nil {
		f.record.record("write")
	}

	if f.paths != nil {
		if path, ok := f.paths[mem.FilenameSummary]; ok {
			return path, f.err
		}
	}

	return "", f.err
}
