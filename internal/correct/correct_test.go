package correct_test

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/correct"
	"engram/internal/memory"
)

// TestConsolidator_BeforeStoreError_FallsThrough verifies that when BeforeStore
// returns an error, the memory is written normally (fallthrough).
func TestConsolidator_BeforeStoreError_FallsThrough(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	classifier := &fakeClassifier{
		result: &memory.ClassifiedMemory{
			Tier:            "A",
			Title:           "Error fallthrough",
			Content:         "content",
			ObservationType: "reminder",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	spyW := &spyWriter{path: "/tmp/memories/fallthrough.toml"}
	renderer := &fakeRenderer{output: "<system-reminder>ok</system-reminder>"}
	cons := &fakeConsolidator{
		action: correct.ConsolidationAction{Type: correct.Consolidated},
		err:    errors.New("consolidation failed"),
	}

	corrector := correct.New(classifier, spyW, renderer, "/tmp")
	corrector.SetConsolidator(cons)

	result, err := corrector.Run(context.Background(), "content", "")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeEmpty())
	g.Expect(spyW.received).NotTo(BeNil())

	if spyW.received == nil {
		return
	}

	g.Expect(spyW.received.Title).To(Equal("Error fallthrough"))
}

// TestConsolidator_Consolidated_WriteError_ReturnsError verifies that a write
// error during consolidated memory storage propagates correctly.
func TestConsolidator_Consolidated_WriteError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	classifier := &fakeClassifier{
		result: &memory.ClassifiedMemory{
			Tier:            "A",
			Title:           "Will fail write",
			Content:         "content",
			ObservationType: "reminder",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	failWriter := &fakeWriter{err: errors.New("disk full")}
	renderer := &fakeRenderer{output: "unused"}
	cons := &fakeConsolidator{
		action: correct.ConsolidationAction{
			Type: correct.Consolidated,
			ConsolidatedMem: &memory.MemoryRecord{
				Title:   "Merged",
				Content: "merged content",
			},
		},
	}

	corrector := correct.New(classifier, failWriter, renderer, "/tmp")
	corrector.SetConsolidator(cons)

	result, err := corrector.Run(context.Background(), "content", "")

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("write consolidated"))
	g.Expect(result).To(BeEmpty())
}

// TestConsolidator_Consolidated_WritesConsolidatedMemory verifies that
// when BeforeStore returns Consolidated, the consolidated memory is written.
func TestConsolidator_Consolidated_WritesConsolidatedMemory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	classifier := &fakeClassifier{
		result: &memory.ClassifiedMemory{
			Tier:            "A",
			Title:           "Original title",
			Content:         "original content",
			ObservationType: "reminder",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	spyW := &spyWriter{path: "/tmp/memories/consolidated.toml"}
	renderer := &fakeRenderer{output: "<system-reminder>consolidated</system-reminder>"}
	cons := &fakeConsolidator{
		action: correct.ConsolidationAction{
			Type: correct.Consolidated,
			ConsolidatedMem: &memory.MemoryRecord{
				Title:     "Merged principle",
				Content:   "generalized content",
				Keywords:  []string{"merged"},
				Concepts:  []string{"consolidation"},
				Principle: "Always consolidate",
			},
		},
	}

	corrector := correct.New(classifier, spyW, renderer, "/tmp")
	corrector.SetConsolidator(cons)
	corrector.SetProjectSlug("test-proj")

	result, err := corrector.Run(context.Background(), "original content", "")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(cons.called).To(BeTrue())
	g.Expect(result).To(ContainSubstring("consolidated"))
	g.Expect(spyW.received).NotTo(BeNil())

	if spyW.received == nil {
		return
	}

	g.Expect(spyW.received.Title).To(Equal("Merged principle"))
	g.Expect(spyW.received.Content).To(Equal("generalized content"))
	g.Expect(spyW.received.ProjectSlug).To(Equal("test-proj"))
	g.Expect(spyW.received.Keywords).To(Equal([]string{"merged"}))
	g.Expect(spyW.received.Concepts).To(Equal([]string{"consolidation"}))
	g.Expect(spyW.received.Principle).To(Equal("Always consolidate"))
}

// TestConsolidator_Nil_ExistingBehavior verifies normal write when no consolidator is set.
func TestConsolidator_Nil_ExistingBehavior(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	classifier := &fakeClassifier{
		result: &memory.ClassifiedMemory{
			Tier:            "A",
			Title:           "Normal memory",
			Content:         "some content",
			ObservationType: "reminder",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	spyW := &spyWriter{path: "/tmp/memories/normal.toml"}
	renderer := &fakeRenderer{output: "<system-reminder>ok</system-reminder>"}

	corrector := correct.New(classifier, spyW, renderer, "/tmp")
	// No SetConsolidator call — consolidator is nil.

	result, err := corrector.Run(context.Background(), "some content", "")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeEmpty())
	g.Expect(spyW.received).NotTo(BeNil())

	if spyW.received == nil {
		return
	}

	g.Expect(spyW.received.Title).To(Equal("Normal memory"))
}

// TestConsolidator_StoreAsIs_WritesNormally verifies normal write
// when BeforeStore returns StoreAsIs.
func TestConsolidator_StoreAsIs_WritesNormally(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	classifier := &fakeClassifier{
		result: &memory.ClassifiedMemory{
			Tier:            "A",
			Title:           "No cluster memory",
			Content:         "unique content",
			ObservationType: "reminder",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	spyW := &spyWriter{path: "/tmp/memories/no-cluster.toml"}
	renderer := &fakeRenderer{output: "<system-reminder>ok</system-reminder>"}
	cons := &fakeConsolidator{
		action: correct.ConsolidationAction{Type: correct.StoreAsIs},
	}

	corrector := correct.New(classifier, spyW, renderer, "/tmp")
	corrector.SetConsolidator(cons)

	result, err := corrector.Run(context.Background(), "unique content", "")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(cons.called).To(BeTrue())
	g.Expect(result).NotTo(BeEmpty())
	g.Expect(spyW.received).NotTo(BeNil())

	if spyW.received == nil {
		return
	}

	g.Expect(spyW.received.Title).To(Equal("No cluster memory"))
}

// T-15: Full pipeline — classify → write → render
func TestT15_FullPipelineClassifyWriteRender(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	recorder := &callRecord{}
	now := time.Now()

	classifier := &fakeClassifier{
		result: &memory.ClassifiedMemory{
			Tier:            "A",
			Title:           "Use Targ for Builds",
			Content:         "remember to use targ",
			ObservationType: "reminder",
			Concepts:        []string{"build-tools"},
			Keywords:        []string{"targ"},
			Principle:       "Use targ for all builds",
			FilenameSummary: "use targ for builds",
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		record: recorder,
	}

	writer := &fakeWriter{
		path:   "/tmp/memories/use-targ-for-builds.toml",
		record: recorder,
	}

	reminderText := "<system-reminder source=\"engram\">\n" +
		"[engram] Memory captured (tier A).\n</system-reminder>\n"
	renderer := &fakeRenderer{
		output: reminderText,
		record: recorder,
	}

	corrector := correct.New(classifier, writer, renderer, "/tmp")
	result, err := corrector.Run(
		context.Background(), "remember to use targ", "",
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(ContainSubstring("[engram] Memory captured"))
	g.Expect(writer.called).To(BeTrue())
	g.Expect(renderer.called).To(BeTrue())
	g.Expect(recorder.calls).To(Equal([]string{
		"classify", "write", "render",
	}))
}

// T-16: No signal — pipeline short-circuits
func TestT16_NoSignalPipelineShortCircuits(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	classifier := &fakeClassifier{result: nil}
	writer := &fakeWriter{}
	renderer := &fakeRenderer{}

	corrector := correct.New(classifier, writer, renderer, "/tmp")
	result, err := corrector.Run(context.Background(), "hello world", "")

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(BeEmpty())
	g.Expect(writer.called).To(BeFalse())
	g.Expect(renderer.called).To(BeFalse())
}

// T-17: Low generalizability — memory with Generalizability 1 is dropped (hard gate)
func TestT17_LowGeneralizabilityMemoryIsDropped(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	classifier := &fakeClassifier{
		result: &memory.ClassifiedMemory{
			Tier:             "A",
			Title:            "Narrow Note",
			Content:          "very specific thing",
			ObservationType:  "reminder",
			Generalizability: 1,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}

	writer := &fakeWriter{}
	renderer := &fakeRenderer{}

	corrector := correct.New(classifier, writer, renderer, "/tmp")
	result, err := corrector.Run(context.Background(), "very specific thing", "")

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(BeEmpty())
	g.Expect(writer.called).To(BeFalse())
}

// T-18: Zero generalizability — backward compat, memory is written
func TestT18_ZeroGeneralizabilityMemoryIsWritten(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	classifier := &fakeClassifier{
		result: &memory.ClassifiedMemory{
			Tier:             "A",
			Title:            "Unscored Memory",
			Content:          "some memory without score",
			ObservationType:  "reminder",
			Generalizability: 0,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}

	writer := &fakeWriter{path: "/tmp/memories/unscored.toml"}
	renderer := &fakeRenderer{output: "<system-reminder>ok</system-reminder>"}

	corrector := correct.New(classifier, writer, renderer, "/tmp")
	result, err := corrector.Run(context.Background(), "some memory without score", "")

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).NotTo(BeEmpty())
	g.Expect(writer.called).To(BeTrue())
}

// T-19: SetProjectSlug — Corrector passes project slug to written Enriched
func TestT19_SetProjectSlug_CorrectorPassesSlugToEnriched(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	classifier := &fakeClassifier{
		result: &memory.ClassifiedMemory{
			Tier:             "A",
			Title:            "Project slug test",
			Content:          "some content",
			ObservationType:  "reminder",
			Generalizability: 3,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}

	spyWriter := &spyWriter{path: "/tmp/memories/project-slug-test.toml"}
	renderer := &fakeRenderer{output: "<system-reminder>ok</system-reminder>"}

	corrector := correct.New(classifier, spyWriter, renderer, "/tmp")
	corrector.SetProjectSlug("test-project")

	result, err := corrector.Run(context.Background(), "some content", "")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeEmpty())
	g.Expect(spyWriter.received).NotTo(BeNil())

	if spyWriter.received == nil {
		return
	}

	g.Expect(spyWriter.received.ProjectSlug).To(Equal("test-project"))
	g.Expect(spyWriter.received.Generalizability).To(Equal(3))
}

// callRecord tracks which pipeline stages were called and in what order.
type callRecord struct {
	calls []string
}

func (r *callRecord) record(name string) {
	r.calls = append(r.calls, name)
}

// fakeClassifier is a test double for correct.Classifier.
type fakeClassifier struct {
	result *memory.ClassifiedMemory
	err    error
	record *callRecord
}

func (f *fakeClassifier) Classify(
	_ context.Context,
	_, _ string,
) (*memory.ClassifiedMemory, error) {
	if f.record != nil {
		f.record.record("classify")
	}

	return f.result, f.err
}

// fakeConsolidator is a test double for the consolidator interface.
type fakeConsolidator struct {
	action correct.ConsolidationAction
	err    error
	called bool
}

func (f *fakeConsolidator) BeforeStore(
	_ context.Context,
	_ *memory.MemoryRecord,
) (correct.ConsolidationAction, error) {
	f.called = true

	return f.action, f.err
}

// fakeRenderer is a test double for correct.Renderer.
type fakeRenderer struct {
	output string
	called bool
	record *callRecord
}

func (f *fakeRenderer) Render(
	_ *memory.ClassifiedMemory,
	_ string,
) string {
	f.called = true

	if f.record != nil {
		f.record.record("render")
	}

	return f.output
}

// fakeWriter is a test double for correct.MemoryWriter.
type fakeWriter struct {
	path   string
	err    error
	called bool
	record *callRecord
}

func (f *fakeWriter) Write(
	_ *memory.Enriched,
	_ string,
) (string, error) {
	f.called = true

	if f.record != nil {
		f.record.record("write")
	}

	return f.path, f.err
}

// spyWriter captures the *memory.Enriched argument passed to Write.
type spyWriter struct {
	path     string
	err      error
	received *memory.Enriched
}

func (s *spyWriter) Write(mem *memory.Enriched, _ string) (string, error) {
	s.received = mem
	return s.path, s.err
}
