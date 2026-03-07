package learn_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/learn"
	"engram/internal/memory"
)

func TestIncrementalLearner_AllLinesStripped(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	delta := &fakeDeltaReader{
		lines:     []string{"tool result line"},
		newOffset: 100,
	}
	strip := func(_ []string) []string { return []string{} }
	offsetStore := &fakeOffsetStore{}
	extractor := &fakeExtractor{}
	retriever := &fakeRetriever{}
	deduplicator := &fakeDeduplicator{}
	writer := &fakeWriter{}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")

	var stderr bytes.Buffer

	inc := learn.NewIncrementalLearner(learner, delta, strip, offsetStore, &stderr)
	result, err := inc.RunIncremental(
		context.Background(), "/transcript.jsonl", "sess-1", "/offset.json",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.CreatedPaths).To(BeEmpty())
	g.Expect(extractor.called).To(BeFalse())
	// Offset should still be updated.
	g.Expect(offsetStore.written.Offset).To(Equal(int64(100)))
}

func TestIncrementalLearner_DeltaReaderError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	delta := &fakeDeltaReader{
		err: errors.New("file not found"),
	}
	strip := func(lines []string) []string { return lines }
	offsetStore := &fakeOffsetStore{}
	extractor := &fakeExtractor{}
	retriever := &fakeRetriever{}
	deduplicator := &fakeDeduplicator{}
	writer := &fakeWriter{}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")

	var stderr bytes.Buffer

	inc := learn.NewIncrementalLearner(learner, delta, strip, offsetStore, &stderr)
	result, err := inc.RunIncremental(
		context.Background(), "/transcript.jsonl", "sess-1", "/offset.json",
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(BeNil())
	g.Expect(stderr.String()).To(ContainSubstring("file not found"))
}

func TestIncrementalLearner_EmptyDelta(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	delta := &fakeDeltaReader{
		lines:     []string{},
		newOffset: 50,
	}
	strip := func(lines []string) []string { return lines }
	offsetStore := &fakeOffsetStore{
		stored: learn.Offset{Offset: 50, SessionID: "sess-1"},
	}
	extractor := &fakeExtractor{}
	retriever := &fakeRetriever{}
	deduplicator := &fakeDeduplicator{}
	writer := &fakeWriter{}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")

	var stderr bytes.Buffer

	inc := learn.NewIncrementalLearner(learner, delta, strip, offsetStore, &stderr)
	result, err := inc.RunIncremental(
		context.Background(), "/transcript.jsonl", "sess-1", "/offset.json",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.CreatedPaths).To(BeEmpty())
	g.Expect(extractor.called).To(BeFalse())
}

func TestIncrementalLearner_HappyPath(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	delta := &fakeDeltaReader{
		lines:     []string{"line1", "line2", "line3"},
		newOffset: 100,
	}
	strip := func(lines []string) []string {
		return lines[:2] // strip last line
	}
	offsetStore := &fakeOffsetStore{
		stored: learn.Offset{Offset: 0, SessionID: "sess-1"},
	}
	candidates := []memory.CandidateLearning{
		{Title: "Found it", Content: "content", FilenameSummary: "found-it"},
	}
	extractor := &fakeExtractor{candidates: candidates}
	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	deduplicator := &fakeDeduplicator{surviving: candidates}
	writer := &fakeWriter{
		paths: map[string]string{"found-it": "/tmp/found-it.toml"},
	}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")

	var stderr bytes.Buffer

	inc := learn.NewIncrementalLearner(learner, delta, strip, offsetStore, &stderr)
	result, err := inc.RunIncremental(
		context.Background(), "/transcript.jsonl", "sess-1", "/offset.json",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.CreatedPaths).To(HaveLen(1))
	g.Expect(offsetStore.written.Offset).To(Equal(int64(100)))
	g.Expect(offsetStore.written.SessionID).To(Equal("sess-1"))
}

func TestIncrementalLearner_LearnerRunError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	delta := &fakeDeltaReader{
		lines:     []string{"data"},
		newOffset: 100,
	}
	strip := func(lines []string) []string { return lines }
	offsetStore := &fakeOffsetStore{}
	extractor := &fakeExtractor{err: errors.New("API down")}
	retriever := &fakeRetriever{}
	deduplicator := &fakeDeduplicator{}
	writer := &fakeWriter{}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")

	var stderr bytes.Buffer

	inc := learn.NewIncrementalLearner(learner, delta, strip, offsetStore, &stderr)
	result, err := inc.RunIncremental(
		context.Background(), "/transcript.jsonl", "sess-1", "/offset.json",
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(BeNil())
	g.Expect(stderr.String()).To(ContainSubstring("API down"))
}

func TestIncrementalLearner_OffsetStoreReadError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	delta := &fakeDeltaReader{
		lines:     []string{"data"},
		newOffset: 100,
	}
	strip := func(lines []string) []string { return lines }
	offsetStore := &fakeOffsetStore{
		readErr: errors.New("corrupt file"),
	}
	candidates := []memory.CandidateLearning{
		{Title: "X", Content: "c", FilenameSummary: "x"},
	}
	extractor := &fakeExtractor{candidates: candidates}
	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	deduplicator := &fakeDeduplicator{surviving: candidates}
	writer := &fakeWriter{
		paths: map[string]string{"x": "/tmp/x.toml"},
	}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")

	var stderr bytes.Buffer

	inc := learn.NewIncrementalLearner(learner, delta, strip, offsetStore, &stderr)
	result, err := inc.RunIncremental(
		context.Background(), "/transcript.jsonl", "sess-1", "/offset.json",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	// Treated as fresh start: offset 0.
	g.Expect(delta.calledWithOffset).To(Equal(int64(0)))
}

func TestIncrementalLearner_OffsetStoreWriteError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	delta := &fakeDeltaReader{
		lines:     []string{"data"},
		newOffset: 100,
	}
	strip := func(lines []string) []string { return lines }
	offsetStore := &fakeOffsetStore{
		writeErr: errors.New("disk full"),
	}
	candidates := []memory.CandidateLearning{
		{Title: "Y", Content: "c", FilenameSummary: "y"},
	}
	extractor := &fakeExtractor{candidates: candidates}
	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	deduplicator := &fakeDeduplicator{surviving: candidates}
	writer := &fakeWriter{
		paths: map[string]string{"y": "/tmp/y.toml"},
	}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")

	var stderr bytes.Buffer

	inc := learn.NewIncrementalLearner(learner, delta, strip, offsetStore, &stderr)
	result, err := inc.RunIncremental(
		context.Background(), "/transcript.jsonl", "sess-1", "/offset.json",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Result should still be returned despite write error.
	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.CreatedPaths).To(HaveLen(1))
	g.Expect(stderr.String()).To(ContainSubstring("disk full"))
}

func TestIncrementalLearner_SessionIDChange(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	delta := &fakeDeltaReader{
		lines:     []string{"new session data"},
		newOffset: 200,
	}
	strip := func(lines []string) []string { return lines }
	offsetStore := &fakeOffsetStore{
		stored: learn.Offset{Offset: 500, SessionID: "old-sess"},
	}
	candidates := []memory.CandidateLearning{
		{Title: "New", Content: "c", FilenameSummary: "new"},
	}
	extractor := &fakeExtractor{candidates: candidates}
	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	deduplicator := &fakeDeduplicator{surviving: candidates}
	writer := &fakeWriter{
		paths: map[string]string{"new": "/tmp/new.toml"},
	}

	learner := learn.New(extractor, retriever, deduplicator, writer, "/tmp")

	var stderr bytes.Buffer

	inc := learn.NewIncrementalLearner(learner, delta, strip, offsetStore, &stderr)
	result, err := inc.RunIncremental(
		context.Background(), "/transcript.jsonl", "new-sess", "/offset.json",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	// Session changed → offset reset to 0, so delta.Read was called with offset 0.
	g.Expect(delta.calledWithOffset).To(Equal(int64(0)))
	g.Expect(offsetStore.written.SessionID).To(Equal("new-sess"))
}

// fakeDeltaReader implements learn.DeltaReader for testing.
type fakeDeltaReader struct {
	lines            []string
	newOffset        int64
	err              error
	calledWithOffset int64
}

func (f *fakeDeltaReader) Read(_ string, offset int64) ([]string, int64, error) {
	f.calledWithOffset = offset

	return f.lines, f.newOffset, f.err
}

// fakeOffsetStore implements learn.OffsetStore for testing.
type fakeOffsetStore struct {
	stored   learn.Offset
	readErr  error
	writeErr error
	written  learn.Offset
}

func (f *fakeOffsetStore) Read(_ string) (learn.Offset, error) {
	return f.stored, f.readErr
}

func (f *fakeOffsetStore) Write(_ string, offset learn.Offset) error {
	f.written = offset

	return f.writeErr
}
