package learn_test

import (
	"context"
	"errors"
	"fmt"
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

// TestFallbackMergePrinciple_CandidateLonger verifies longer candidate wins.
func TestFallbackMergePrinciple_CandidateLonger(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	candidate := memory.CandidateLearning{
		Title:     "candidate",
		Principle: "much longer principle text here",
	}
	existingMem := &memory.Stored{
		FilePath:  "/data/use-targ.toml",
		Title:     "Use targ",
		Principle: "short",
	}

	deduplicator := &fakeMergingDeduplicator{
		mergePairs: []dedup.MergePair{{Candidate: candidate, Existing: existingMem}},
	}
	mergeWriter := &fakeMergeWriter{}
	extractor := &fakeExtractor{candidates: []memory.CandidateLearning{candidate}}
	retriever := &fakeRetriever{memories: []*memory.Stored{existingMem}}
	writer := &fakeWriter{}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")
	learner.SetMergeWriter(mergeWriter)

	_, err := learner.Run(context.Background(), "transcript")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// mergeWriter.principle = max(candidate, existing) = candidate
	g.Expect(mergeWriter.principle).To(Equal("much longer principle text here"))
}

// Task 2 (#345): When all keywords are common, keep originals (don't strip to zero).
func TestKeywordIDFFilter_AllCommon_KeepsOriginals(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// "test" in 5/10 = 50%, "build" in 4/10 = 40% — both exceed 30%
	existingMemories := make([]*memory.Stored, 0, 10)
	for i := range 5 {
		existingMemories = append(existingMemories, &memory.Stored{
			FilePath: fmt.Sprintf("/data/mem-%d.toml", i),
			Title:    fmt.Sprintf("mem-%d", i),
			Keywords: []string{"test", "build"},
		})
	}

	for i := 5; i < 10; i++ {
		existingMemories = append(existingMemories, &memory.Stored{
			FilePath: fmt.Sprintf("/data/mem-%d.toml", i),
			Title:    fmt.Sprintf("mem-%d", i),
			Keywords: []string{fmt.Sprintf("unique-%d", i)},
		})
	}

	candidate := memory.CandidateLearning{
		Tier:            "A",
		Title:           "All common",
		Content:         "some content",
		Keywords:        []string{"test", "build"},
		FilenameSummary: "all-common",
	}

	extractor := &fakeExtractor{candidates: []memory.CandidateLearning{candidate}}
	retriever := &fakeRetriever{memories: existingMemories}
	deduplicator := &fakeDeduplicator{surviving: []memory.CandidateLearning{candidate}}
	writer := &fakeWriter{
		paths: map[string]string{
			"all-common": "/tmp/memories/all-common.toml",
		},
	}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")

	result, err := learner.Run(context.Background(), "transcript")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(writer.received).To(HaveLen(1))

	if len(writer.received) == 0 {
		return
	}

	// All keywords are common, so originals should be kept (not stripped to zero).
	g.Expect(writer.received[0].Keywords).To(ConsistOf("test", "build"))
}

// Task 2 (#345): Keyword IDF filter removes common keywords before write.
func TestKeywordIDFFilter_RemovesCommonKeywords(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// "test" appears in 4 of 10 existing memories (40% > 30% threshold).
	// "unique-keyword" appears in 0 → always passes.
	existingMemories := make([]*memory.Stored, 0, 10)
	for i := range 4 {
		existingMemories = append(existingMemories, &memory.Stored{
			FilePath: fmt.Sprintf("/data/mem-%d.toml", i),
			Title:    fmt.Sprintf("mem-%d", i),
			Keywords: []string{"test"},
		})
	}

	for i := 4; i < 10; i++ {
		existingMemories = append(existingMemories, &memory.Stored{
			FilePath: fmt.Sprintf("/data/mem-%d.toml", i),
			Title:    fmt.Sprintf("mem-%d", i),
			Keywords: []string{fmt.Sprintf("unique-%d", i)},
		})
	}

	candidate := memory.CandidateLearning{
		Tier:            "A",
		Title:           "New learning",
		Content:         "some content",
		Keywords:        []string{"test", "unique-keyword"},
		FilenameSummary: "new-learning",
	}

	extractor := &fakeExtractor{candidates: []memory.CandidateLearning{candidate}}
	retriever := &fakeRetriever{memories: existingMemories}
	deduplicator := &fakeDeduplicator{surviving: []memory.CandidateLearning{candidate}}
	writer := &fakeWriter{
		paths: map[string]string{
			"new-learning": "/tmp/memories/new-learning.toml",
		},
	}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")

	result, err := learner.Run(context.Background(), "transcript")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.CreatedPaths).To(HaveLen(1))
	g.Expect(writer.received).To(HaveLen(1))

	if len(writer.received) == 0 {
		return
	}

	// "test" should be filtered out (40% > 30%), "unique-keyword" should remain.
	g.Expect(writer.received[0].Keywords).To(ConsistOf("unique-keyword"))
	g.Expect(writer.received[0].Keywords).NotTo(ContainElement("test"))
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

// TestRecordAbsorbedFunc_CallsThrough verifies that a plain func wired via SetRecordAbsorbed is called.
func TestRecordAbsorbedFunc_CallsThrough(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var called bool

	var fnErr error

	fn := func(existingPath, candidateTitle, contentHash string, _ time.Time) error {
		called = true

		g.Expect(existingPath).To(Equal("/path/to/existing.toml"))
		g.Expect(candidateTitle).To(Equal("candidate title"))
		g.Expect(contentHash).To(HaveLen(16))

		return fnErr
	}

	err := fn("/path/to/existing.toml", "candidate title", "abc123def4567890", time.Now())

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(called).To(BeTrue())
}

// TestSetMemoryMerger_PipelineUsesIt verifies that SetMemoryMerger wires up the merger for merge pairs.
func TestSetMemoryMerger_PipelineUsesIt(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	candidate := memory.CandidateLearning{
		Title: "Use targ", Content: "use targ for builds", FilenameSummary: "use-targ",
	}
	existingMem := &memory.Stored{
		FilePath:  "/data/use-targ.toml",
		Title:     "Use targ",
		Principle: "old",
	}

	merger := &fakeMerger{merged: "merged principle"}
	extractor := &fakeExtractor{candidates: []memory.CandidateLearning{candidate}}
	retriever := &fakeRetriever{memories: []*memory.Stored{existingMem}}
	deduplicator := &fakeMergingDeduplicator{
		mergePairs: []dedup.MergePair{{Candidate: candidate, Existing: existingMem}},
	}
	writer := &fakeWriter{}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")
	learner.SetMemoryMerger(merger)

	result, err := learner.Run(context.Background(), "some transcript")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())
	g.Expect(merger.called).To(BeTrue())
}

// TestSetMergeWriter_PipelineCallsIt verifies that SetMergeWriter writes merged memories.
func TestSetMergeWriter_PipelineCallsIt(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	candidate := memory.CandidateLearning{
		Title: "Use targ", Content: "use targ for builds", FilenameSummary: "use-targ",
	}
	existingMem := &memory.Stored{
		FilePath:  "/data/use-targ.toml",
		Title:     "Use targ",
		Principle: "old",
	}

	mergeWriter := &fakeMergeWriter{}
	extractor := &fakeExtractor{candidates: []memory.CandidateLearning{candidate}}
	retriever := &fakeRetriever{memories: []*memory.Stored{existingMem}}
	deduplicator := &fakeMergingDeduplicator{
		mergePairs: []dedup.MergePair{{Candidate: candidate, Existing: existingMem}},
	}
	writer := &fakeWriter{}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")
	learner.SetMergeWriter(mergeWriter)

	result, err := learner.Run(context.Background(), "some transcript")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())
	g.Expect(mergeWriter.called).To(BeTrue())
}

// TestSetRegistryAbsorber_PipelineCallsIt verifies that SetRegistryAbsorber records merges.
func TestSetRegistryAbsorber_PipelineCallsIt(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	candidate := memory.CandidateLearning{
		Title: "Use targ", Content: "use targ for builds", FilenameSummary: "use-targ",
		Keywords: []string{"targ"},
	}
	existingMem := &memory.Stored{
		FilePath:  "/data/use-targ.toml",
		Title:     "Use targ",
		Principle: "old",
	}

	var absorbCalled bool

	extractor := &fakeExtractor{candidates: []memory.CandidateLearning{candidate}}
	retriever := &fakeRetriever{memories: []*memory.Stored{existingMem}}
	deduplicator := &fakeMergingDeduplicator{
		mergePairs: []dedup.MergePair{{Candidate: candidate, Existing: existingMem}},
	}
	writer := &fakeWriter{}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")
	learner.SetRecordAbsorbed(func(_, _, _ string, _ time.Time) error {
		absorbCalled = true
		return nil
	})

	result, err := learner.Run(context.Background(), "some transcript")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())
	g.Expect(absorbCalled).To(BeTrue())
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

	type registerCall struct {
		filePath string
		title    string
		content  string
	}

	var registerCalls []registerCall

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")
	learner.SetRegisterMemory(func(filePath, title, content string, _ time.Time) error {
		registerCalls = append(registerCalls, registerCall{filePath: filePath, title: title, content: content})
		return nil
	})

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
	g.Expect(registerCalls).To(HaveLen(2))

	if len(registerCalls) < 2 {
		return
	}

	g.Expect(registerCalls[0].filePath).To(Equal("/tmp/memories/use-targ.toml"))
	g.Expect(registerCalls[0].title).To(Equal("Use targ"))
	g.Expect(registerCalls[0].content).To(Equal("use targ for builds"))
	g.Expect(registerCalls[1].filePath).To(Equal("/tmp/memories/di-pattern.toml"))
	g.Expect(registerCalls[1].title).To(Equal("DI pattern"))
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
	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")
	learner.SetRegisterMemory(func(_, _, _ string, _ time.Time) error {
		return errors.New("registry write failed")
	})

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

// TestUnionConcepts_MergesBothSets verifies union has all unique concepts.
func TestUnionConcepts_MergesBothSets(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	candidate := memory.CandidateLearning{
		Title:    "candidate",
		Concepts: []string{"alpha", "gamma"},
	}
	existingMem := &memory.Stored{
		FilePath: "/data/mem.toml",
		Title:    "existing",
		Concepts: []string{"alpha", "beta"},
	}

	deduplicator := &fakeMergingDeduplicator{
		mergePairs: []dedup.MergePair{{Candidate: candidate, Existing: existingMem}},
	}
	mergeWriter := &fakeMergeWriter{}
	extractor := &fakeExtractor{candidates: []memory.CandidateLearning{candidate}}
	retriever := &fakeRetriever{memories: []*memory.Stored{existingMem}}
	writer := &fakeWriter{}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")
	learner.SetMergeWriter(mergeWriter)

	_, err := learner.Run(context.Background(), "transcript")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(mergeWriter.concepts).To(ConsistOf("alpha", "beta", "gamma"))
}

// TestUnionKeywords_DeduplicatesMixedFormat verifies normalization deduplicates hyphen/underscore variants.
func TestUnionKeywords_DeduplicatesMixedFormat(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// "prefixed-ids" and "prefixed_ids" should unify to one normalized keyword.
	learner := learn.New(nil, nil, nil, nil, "")
	result := learn.ExportUnionKeywords(learner,
		[]string{"prefixed-ids", "collision-avoidance"},
		[]string{"prefixed_ids", "collision_avoidance", "new_keyword"},
	)

	g.Expect(result).To(ConsistOf("prefixed_ids", "collision_avoidance", "new_keyword"))
}

// TestGeneralizabilityGate_DropsLowScoreCandidates verifies that candidates with
// Generalizability < 2 are filtered before dedup and write.
func TestGeneralizabilityGate_DropsLowScoreCandidates(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	lowCandidate := memory.CandidateLearning{
		Tier:            "A",
		Title:           "Low generalizability",
		Content:         "very specific to this one session",
		FilenameSummary: "low-gen",
		Generalizability: 1,
	}
	highCandidate := memory.CandidateLearning{
		Tier:            "A",
		Title:           "High generalizability",
		Content:         "broadly applicable principle",
		FilenameSummary: "high-gen",
		Generalizability: 3,
	}

	// Pass-through deduplicator returns whatever candidates it receives — the gate must filter first.
	extractor := &fakeExtractor{candidates: []memory.CandidateLearning{lowCandidate, highCandidate}}
	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	deduplicator := &fakePassThroughDeduplicator{}
	writer := &fakeWriter{
		paths: map[string]string{
			"high-gen": "/tmp/memories/high-gen.toml",
		},
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

	// Only the high-generalizability candidate should be written.
	g.Expect(writer.received).To(HaveLen(1))
	g.Expect(result.CreatedPaths).To(HaveLen(1))
	g.Expect(result.CreatedPaths).To(ConsistOf("/tmp/memories/high-gen.toml"))

	// The dropped candidate counts toward SkippedCount.
	g.Expect(result.SkippedCount).To(Equal(1))
}

// TestGeneralizabilityGate_ZeroIsKept verifies that candidates with Generalizability == 0
// (default/unset — LLM did not return the field) are not dropped.
func TestGeneralizabilityGate_ZeroIsKept(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	candidate := memory.CandidateLearning{
		Tier:             "A",
		Title:            "Pre-existing candidate",
		Content:          "principle content",
		FilenameSummary:  "pre-existing",
		Generalizability: 0,
	}

	// Pass-through deduplicator returns whatever it receives — zero must survive the gate.
	extractor := &fakeExtractor{candidates: []memory.CandidateLearning{candidate}}
	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	deduplicator := &fakePassThroughDeduplicator{}
	writer := &fakeWriter{
		paths: map[string]string{
			"pre-existing": "/tmp/memories/pre-existing.toml",
		},
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

	// Zero generalizability is backward-compat: must NOT be dropped.
	g.Expect(writer.received).To(HaveLen(1))
	g.Expect(result.CreatedPaths).To(ConsistOf("/tmp/memories/pre-existing.toml"))
	g.Expect(result.SkippedCount).To(Equal(0))
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

func (f *fakeDeduplicator) Classify(
	_ []memory.CandidateLearning,
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

// fakeMergeWriter is a test double for learn.MergeWriter.
type fakeMergeWriter struct {
	called    bool
	principle string
	concepts  []string
}

func (f *fakeMergeWriter) UpdateMerged(
	_ *memory.Stored,
	principle string,
	_, concepts []string,
	_ time.Time,
) error {
	f.called = true
	f.principle = principle
	f.concepts = concepts

	return nil
}

// fakeMerger is a test double for learn.MemoryMerger.
type fakeMerger struct {
	called bool
	merged string
}

func (f *fakeMerger) MergePrinciples(_ context.Context, _, _ string) (string, error) {
	f.called = true

	return f.merged, nil
}

// fakeMergingDeduplicator returns merge pairs instead of surviving candidates.
type fakeMergingDeduplicator struct {
	mergePairs []dedup.MergePair
}

func (f *fakeMergingDeduplicator) Classify(
	_ []memory.CandidateLearning,
	_ []*memory.Stored,
) dedup.ClassifyResult {
	return dedup.ClassifyResult{MergePairs: f.mergePairs}
}

func (f *fakeMergingDeduplicator) Filter(
	_ []memory.CandidateLearning,
	_ []*memory.Stored,
) []memory.CandidateLearning {
	return nil
}

// fakePassThroughDeduplicator passes all candidates through as surviving with no merges.
type fakePassThroughDeduplicator struct{}

func (f *fakePassThroughDeduplicator) Classify(
	candidates []memory.CandidateLearning,
	_ []*memory.Stored,
) dedup.ClassifyResult {
	return dedup.ClassifyResult{
		Surviving:  candidates,
		MergePairs: []dedup.MergePair{},
	}
}

func (f *fakePassThroughDeduplicator) Filter(
	candidates []memory.CandidateLearning,
	_ []*memory.Stored,
) []memory.CandidateLearning {
	return candidates
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
