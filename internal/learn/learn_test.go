package learn_test

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/creationlog"
	"engram/internal/dedup"
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

// T-201: Learn pipeline calls RegisterMemory for new memories.
func TestT201_RegistryRegistrarCalledForNewMemories(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []memory.CandidateLearning{
		{
			Tier:            "A",
			Title:           "Use targ",
			Content:         "use targ for builds",
			FilenameSummary: "use-targ",
		},
		{
			Tier:            "B",
			Title:           "DI pattern",
			Content:         "use DI everywhere",
			FilenameSummary: "di-pattern",
		},
	}

	extractor := &fakeExtractor{candidates: candidates}
	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	deduplicator := &fakeDeduplicator{surviving: candidates}
	writer := &fakeWriter{
		paths: map[string]string{
			"use-targ":   "/tmp/memories/use-targ.toml",
			"di-pattern": "/tmp/memories/di-pattern.toml",
		},
	}
	registrar := &fakeRegistrar{}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")
	learner.SetRegistryRegistrar(registrar)

	result, err := learner.Run(context.Background(), "some transcript")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.CreatedPaths).To(HaveLen(2))
	g.Expect(registrar.calls).To(HaveLen(2))

	if len(registrar.calls) < 2 {
		return
	}

	g.Expect(registrar.calls[0].filePath).To(Equal("/tmp/memories/use-targ.toml"))
	g.Expect(registrar.calls[0].title).To(Equal("Use targ"))
	g.Expect(registrar.calls[0].content).To(Equal("use targ for builds"))
	g.Expect(registrar.calls[1].filePath).To(Equal("/tmp/memories/di-pattern.toml"))
	g.Expect(registrar.calls[1].title).To(Equal("DI pattern"))
}

// T-201b: Registry error does not fail the learn pipeline (fire-and-forget).
func TestT201b_RegistrarErrorDoesNotFailPipeline(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []memory.CandidateLearning{
		{
			Tier:            "A",
			Title:           "Use targ",
			Content:         "use targ for builds",
			FilenameSummary: "use-targ",
		},
	}

	extractor := &fakeExtractor{candidates: candidates}
	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	deduplicator := &fakeDeduplicator{surviving: candidates}
	writer := &fakeWriter{
		paths: map[string]string{
			"use-targ": "/tmp/memories/use-targ.toml",
		},
	}
	registrar := &fakeRegistrar{err: errors.New("registry write failed")}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")
	learner.SetRegistryRegistrar(registrar)

	result, err := learner.Run(context.Background(), "some transcript")

	// Should succeed despite registry error.
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.CreatedPaths).To(ConsistOf("/tmp/memories/use-targ.toml"))
}

// T-201c: Nil registrar does not panic (backward compat).
func TestT201c_NilRegistrarNoPanic(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []memory.CandidateLearning{
		{
			Tier:            "A",
			Title:           "Use targ",
			Content:         "use targ for builds",
			FilenameSummary: "use-targ",
		},
	}

	extractor := &fakeExtractor{candidates: candidates}
	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	deduplicator := &fakeDeduplicator{surviving: candidates}
	writer := &fakeWriter{
		paths: map[string]string{
			"use-targ": "/tmp/memories/use-targ.toml",
		},
	}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")
	// No SetRegistryRegistrar — registrar is nil.

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
	g.Expect(recorder.calls).To(Equal([]string{"extract", "list", "classify", "write"}))
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

// T-95: Learner calls CreationLogger after each successful write
func TestT95_CreationLogger_CalledAfterEachWrite(t *testing.T) {
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
	}

	extractor := &fakeExtractor{candidates: candidates}
	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	deduplicator := &fakeDeduplicator{surviving: candidates}
	writer := &fakeWriter{
		paths: map[string]string{
			"use-targ":   "/tmp/memories/use-targ.toml",
			"di-pattern": "/tmp/memories/di-pattern.toml",
		},
	}
	logger := &fakeCreationLogger{}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")
	learner.SetCreationLogger(logger)

	result, err := learner.Run(context.Background(), "some transcript")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.CreatedPaths).To(HaveLen(2))
	g.Expect(logger.entries).To(HaveLen(2))

	g.Expect(logger.entries[0].Title).To(Equal("Use targ"))
	g.Expect(logger.entries[0].Tier).To(Equal("A"))
	g.Expect(logger.entries[0].Filename).To(Equal("use-targ.toml"))

	g.Expect(logger.entries[1].Title).To(Equal("DI pattern"))
	g.Expect(logger.entries[1].Tier).To(Equal("B"))
	g.Expect(logger.entries[1].Filename).To(Equal("di-pattern.toml"))
}

// T-96: Learner with nil CreationLogger skips logging (backward compatible)
func TestT96_NilCreationLogger_NoPanic(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []memory.CandidateLearning{
		{Tier: "A", Title: "Use targ", Content: "use targ for builds", FilenameSummary: "use-targ"},
	}

	extractor := &fakeExtractor{candidates: candidates}
	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	deduplicator := &fakeDeduplicator{surviving: candidates}
	writer := &fakeWriter{
		paths: map[string]string{
			"use-targ": "/tmp/memories/use-targ.toml",
		},
	}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")
	// No SetCreationLogger call — logger is nil

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
}

// T-97: Learner creation log error does not fail the pipeline
func TestT97_CreationLoggerError_PipelineSucceeds(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []memory.CandidateLearning{
		{Tier: "A", Title: "Use targ", Content: "use targ for builds", FilenameSummary: "use-targ"},
	}

	extractor := &fakeExtractor{candidates: candidates}
	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	deduplicator := &fakeDeduplicator{surviving: candidates}
	writer := &fakeWriter{
		paths: map[string]string{
			"use-targ": "/tmp/memories/use-targ.toml",
		},
	}
	logger := &fakeCreationLogger{err: errors.New("log append failed")}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")
	learner.SetCreationLogger(logger)

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

// fakeCreationLogger is a test double for learn.CreationLogger.
type fakeCreationLogger struct {
	entries []creationlog.LogEntry
	err     error
}

func (f *fakeCreationLogger) Append(entry creationlog.LogEntry, _ string) error {
	f.entries = append(f.entries, entry)
	return f.err
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

func (f *fakeDeduplicator) Classify(
	candidates []memory.CandidateLearning,
	_ []*memory.Stored,
) dedup.ClassifyResult {
	f.called = true

	if f.record != nil {
		f.record.record("classify")
	}

	// Return classified result with surviving candidates and no merges
	return dedup.ClassifyResult{
		Surviving:  f.surviving,
		MergePairs: []dedup.MergePair{},
	}
}

// fakeExtractor is a test double for learn.TranscriptExtractor.
type fakeExtractor struct {
	candidates []memory.CandidateLearning
	err        error
	called     bool
	record     *callRecord
}

func (f *fakeExtractor) Extract(_ context.Context, _ string) ([]memory.CandidateLearning, error) {
	f.called = true

	if f.record != nil {
		f.record.record("extract")
	}

	return f.candidates, f.err
}

// fakeRegistrar is a test double for learn.RegistryRegistrar.
type fakeRegistrar struct {
	calls []registrarCall
	err   error
}

func (f *fakeRegistrar) RegisterMemory(
	filePath, title, content string, _ time.Time,
) error {
	f.calls = append(f.calls, registrarCall{
		filePath: filePath,
		title:    title,
		content:  content,
	})

	return f.err
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

type registrarCall struct {
	filePath string
	title    string
	content  string
}
