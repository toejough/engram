package correct_test

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/correct"
	"engram/internal/memory"
)

// callRecord tracks which pipeline stages were called and in what order.
type callRecord struct {
	calls []string
}

func (r *callRecord) record(name string) {
	r.calls = append(r.calls, name)
}

// fakePatternMatcher is a test double for correct.PatternMatcher.
type fakePatternMatcher struct {
	match  *memory.PatternMatch
	record *callRecord
}

func (f *fakePatternMatcher) Match(_ string) *memory.PatternMatch {
	if f.record != nil {
		f.record.record("match")
	}

	return f.match
}

// fakeEnricher is a test double for correct.Enricher.
type fakeEnricher struct {
	mem    *memory.Enriched
	err    error
	called bool
	record *callRecord
}

func (f *fakeEnricher) Enrich(
	_ context.Context,
	_ string,
	_ *memory.PatternMatch,
) (*memory.Enriched, error) {
	f.called = true

	if f.record != nil {
		f.record.record("enrich")
	}

	return f.mem, f.err
}

// fakeWriter is a test double for correct.MemoryWriter.
type fakeWriter struct {
	path   string
	err    error
	called bool
	record *callRecord
}

func (f *fakeWriter) Write(_ *memory.Enriched, _ string) (string, error) {
	f.called = true

	if f.record != nil {
		f.record.record("write")
	}

	return f.path, f.err
}

// fakeRenderer is a test double for correct.Renderer.
type fakeRenderer struct {
	output string
	called bool
	record *callRecord
}

func (f *fakeRenderer) Render(_ *memory.Enriched, _ string) string {
	f.called = true

	if f.record != nil {
		f.record.record("render")
	}

	return f.output
}

// T-15: Full pipeline — match → enrich → write → render
func TestT15_FullPipelineMatchEnrichWriteRender(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	recorder := &callRecord{}
	now := time.Now()

	matcher := &fakePatternMatcher{
		match: &memory.PatternMatch{
			Pattern:    `\bremember\s+(that|to)`,
			Label:      "reminder",
			Confidence: "A",
		},
		record: recorder,
	}

	enricher := &fakeEnricher{
		mem: &memory.Enriched{
			Title:           "Use Targ for Builds",
			Content:         "remember to use targ",
			ObservationType: "reminder",
			Concepts:        []string{"build-tools"},
			Keywords:        []string{"targ"},
			Principle:       "Use targ for all builds",
			FilenameSummary: "use targ for builds",
			Confidence:      "A",
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		record: recorder,
	}

	writer := &fakeWriter{
		path:   "/tmp/memories/use-targ-for-builds.toml",
		record: recorder,
	}

	reminderText := "<system-reminder source=\"engram\">\n[engram] Memory captured.\n</system-reminder>\n"
	renderer := &fakeRenderer{
		output: reminderText,
		record: recorder,
	}

	corrector := correct.New(matcher, enricher, writer, renderer, "/tmp")
	result, err := corrector.Run(context.Background(), "remember to use targ")

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(ContainSubstring("[engram] Memory captured."))
	g.Expect(enricher.called).To(BeTrue())
	g.Expect(writer.called).To(BeTrue())
	g.Expect(renderer.called).To(BeTrue())
	g.Expect(recorder.calls).To(Equal([]string{"match", "enrich", "write", "render"}))
}

// T-16: No match — pipeline short-circuits
func TestT16_NoMatchPipelineShortCircuits(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	matcher := &fakePatternMatcher{match: nil}
	enricher := &fakeEnricher{}
	writer := &fakeWriter{}
	renderer := &fakeRenderer{}

	corrector := correct.New(matcher, enricher, writer, renderer, "/tmp")
	result, err := corrector.Run(context.Background(), "hello world")

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(BeEmpty())
	g.Expect(enricher.called).To(BeFalse())
	g.Expect(writer.called).To(BeFalse())
	g.Expect(renderer.called).To(BeFalse())
}
