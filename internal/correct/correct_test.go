package correct_test

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/correct"
	"engram/internal/memory"
)

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
			Tier:            "A",
			Title:           "Narrow Note",
			Content:         "very specific thing",
			ObservationType: "reminder",
			Generalizability: 1,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
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
			Tier:            "A",
			Title:           "Unscored Memory",
			Content:         "some memory without score",
			ObservationType: "reminder",
			Generalizability: 0,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
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
