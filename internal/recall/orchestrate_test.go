package recall_test

import (
	"bytes"
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"engram/internal/recall"

	. "github.com/onsi/gomega"
)

func TestFormatResult(t *testing.T) {
	t.Parallel()

	t.Run("summary only", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		var buf bytes.Buffer

		err := recall.FormatResult(&buf, &recall.Result{Summary: "session content"})
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(buf.String()).To(Equal("session content"))
	})

	t.Run("summary with memories", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		var buf bytes.Buffer

		err := recall.FormatResult(&buf, &recall.Result{
			Summary:  "session content",
			Memories: "memory1\nmemory2",
		})
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(buf.String()).To(Equal("session content\n=== MEMORIES ===\nmemory1\nmemory2"))
	})

	t.Run("write error on summary", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		err := recall.FormatResult(&failWriter{}, &recall.Result{Summary: "content"})
		g.Expect(err).To(HaveOccurred())

		if err != nil {
			g.Expect(err.Error()).To(ContainSubstring("writing summary"))
		}
	})

	t.Run("write error on memories", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		w := &failAfterNWriter{remaining: len("session content")}

		err := recall.FormatResult(w, &recall.Result{
			Summary:  "session content",
			Memories: "mem",
		})
		g.Expect(err).To(HaveOccurred())

		if err != nil {
			g.Expect(err.Error()).To(ContainSubstring("writing memories"))
		}
	})

	t.Run("empty result", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		var buf bytes.Buffer

		err := recall.FormatResult(&buf, &recall.Result{})
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(buf.String()).To(BeEmpty())
	})
}

// --- Tests ---

func TestOrchestrator_Recall_ModeA(t *testing.T) {
	t.Parallel()

	t.Run("returns raw stripped content without summarization", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		finder := &fakeFinder{paths: []string{"/a.jsonl", "/b.jsonl"}}
		reader := &fakeReader{
			contents: map[string]string{
				"/a.jsonl": "session a content",
				"/b.jsonl": "session b content",
			},
			sizes: map[string]int{
				"/a.jsonl": 17,
				"/b.jsonl": 17,
			},
		}
		surfacer := &fakeSurfacer{result: "relevant memories"}

		orch := recall.NewOrchestrator(finder, reader, nil, surfacer)

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(Equal("session a contentsession b content"))
		g.Expect(result.Memories).To(Equal("relevant memories"))
	})

	t.Run("no sessions found returns empty result", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		finder := &fakeFinder{paths: []string{}}
		orch := recall.NewOrchestrator(finder, nil, nil, nil)

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(BeEmpty())
		g.Expect(result.Memories).To(BeEmpty())
	})

	t.Run("surfacer error still returns content", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		finder := &fakeFinder{paths: []string{"/a.jsonl"}}
		reader := &fakeReader{
			contents: map[string]string{"/a.jsonl": "content"},
			sizes:    map[string]int{"/a.jsonl": 7},
		}
		surfacer := &fakeSurfacer{err: errors.New("surfacer broke")}

		orch := recall.NewOrchestrator(finder, reader, nil, surfacer)

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(Equal("content"))
		g.Expect(result.Memories).To(BeEmpty())
	})

	t.Run("reader error skips session and continues", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		finder := &fakeFinder{paths: []string{"/bad.jsonl", "/good.jsonl"}}
		reader := &fakeReader{
			contents: map[string]string{"/good.jsonl": "good content"},
			sizes:    map[string]int{"/good.jsonl": 12},
			errs:     map[string]error{"/bad.jsonl": errors.New("read failed")},
		}

		orch := recall.NewOrchestrator(finder, reader, nil, nil)

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(Equal("good content"))
	})

	t.Run("budget exceeded stops reading sessions", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		finder := &fakeFinder{paths: []string{"/a.jsonl", "/b.jsonl"}}
		reader := &fakeReader{
			contents: map[string]string{
				"/a.jsonl": "big content",
				"/b.jsonl": "should not read",
			},
			sizes: map[string]int{
				"/a.jsonl": recall.DefaultModeABudget, // Exactly at budget
				"/b.jsonl": 100,
			},
		}

		orch := recall.NewOrchestrator(finder, reader, nil, nil)

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		// Only first session's content should be returned.
		g.Expect(result.Summary).To(Equal("big content"))
	})

	t.Run("finder error propagates", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		finder := &fakeFinder{err: errors.New("find failed")}
		orch := recall.NewOrchestrator(finder, nil, nil, nil)

		_, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).To(HaveOccurred())

		if err != nil {
			g.Expect(err.Error()).To(ContainSubstring("recalling"))
		}
	})

	t.Run("nil surfacer works without memories", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		finder := &fakeFinder{paths: []string{"/a.jsonl"}}
		reader := &fakeReader{
			contents: map[string]string{"/a.jsonl": "content"},
			sizes:    map[string]int{"/a.jsonl": 7},
		}

		orch := recall.NewOrchestrator(finder, reader, nil, nil)

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(Equal("content"))
		g.Expect(result.Memories).To(BeEmpty())
	})
}

func TestOrchestrator_Recall_ModeB(t *testing.T) {
	t.Parallel()

	t.Run("extracts relevant content from sessions", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		finder := &fakeFinder{paths: []string{"/a.jsonl", "/b.jsonl"}}
		reader := &fakeReader{
			contents: map[string]string{
				"/a.jsonl": "session a",
				"/b.jsonl": "session b",
			},
			sizes: map[string]int{
				"/a.jsonl": 9,
				"/b.jsonl": 9,
			},
		}
		summarizer := &fakeSummarizer{extractResult: "relevant bit"}
		surfacer := &fakeSurfacer{result: "memories"}

		orch := recall.NewOrchestrator(finder, reader, summarizer, surfacer)

		result, err := orch.Recall(context.Background(), "/proj", "my query")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(ContainSubstring("relevant bit"))
		g.Expect(result.Memories).To(Equal("memories"))
		g.Expect(int(summarizer.extractCalls.Load())).To(Equal(2))
	})

	t.Run("stops at byte cap", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// Each extract returns a string of 800 bytes.
		longResult := make([]byte, 800)
		for i := range longResult {
			longResult[i] = 'x'
		}

		finder := &fakeFinder{paths: []string{"/a.jsonl", "/b.jsonl", "/c.jsonl"}}
		reader := &fakeReader{
			contents: map[string]string{
				"/a.jsonl": "a",
				"/b.jsonl": "b",
				"/c.jsonl": "c",
			},
			sizes: map[string]int{
				"/a.jsonl": 1,
				"/b.jsonl": 1,
				"/c.jsonl": 1,
			},
		}
		summarizer := &fakeSummarizer{extractResult: string(longResult)}

		orch := recall.NewOrchestrator(finder, reader, summarizer, nil)

		result, err := orch.Recall(context.Background(), "/proj", "query")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		// All 3 sessions are extracted in parallel, but only 2 results
		// are concatenated before exceeding the byte cap.
		g.Expect(int(summarizer.extractCalls.Load())).To(Equal(3))
		g.Expect(len(result.Summary)).To(BeNumerically(">=", 1500))
	})

	t.Run("reader error skips session in mode B", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		finder := &fakeFinder{paths: []string{"/bad.jsonl", "/good.jsonl"}}
		reader := &fakeReader{
			contents: map[string]string{"/good.jsonl": "good"},
			sizes:    map[string]int{"/good.jsonl": 4},
			errs:     map[string]error{"/bad.jsonl": errors.New("read err")},
		}
		summarizer := &fakeSummarizer{extractResult: "extracted"}

		orch := recall.NewOrchestrator(finder, reader, summarizer, nil)

		result, err := orch.Recall(context.Background(), "/proj", "query")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		// Only the good session gets extracted (bad session read fails).
		g.Expect(int(summarizer.extractCalls.Load())).To(Equal(1))
		g.Expect(result.Summary).To(Equal("extracted"))
	})

	t.Run("extract error skips session in mode B", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		finder := &fakeFinder{paths: []string{"/a.jsonl", "/b.jsonl"}}
		reader := &fakeReader{
			contents: map[string]string{
				"/a.jsonl": "a content",
				"/b.jsonl": "b content",
			},
			sizes: map[string]int{"/a.jsonl": 9, "/b.jsonl": 9},
		}
		summarizer := &fakeSummarizer{
			extractResult: "good",
			extractErr:    errors.New("extract err"),
		}

		// The fake always returns the error — both sessions get skipped.
		orch := recall.NewOrchestrator(finder, reader, summarizer, nil)

		result, err := orch.Recall(context.Background(), "/proj", "query")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(BeEmpty())
	})

	t.Run("nil summarizer returns empty result in mode B", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		finder := &fakeFinder{paths: []string{"/a.jsonl"}}
		reader := &fakeReader{
			contents: map[string]string{"/a.jsonl": "content"},
			sizes:    map[string]int{"/a.jsonl": 7},
		}

		orch := recall.NewOrchestrator(finder, reader, nil, nil)

		result, err := orch.Recall(context.Background(), "/proj", "query")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(BeEmpty())
	})

	t.Run("surfaces memories using query not extracted content", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		finder := &fakeFinder{paths: []string{"/a.jsonl"}}
		reader := &fakeReader{
			contents: map[string]string{"/a.jsonl": "content"},
			sizes:    map[string]int{"/a.jsonl": 7},
		}
		summarizer := &fakeSummarizer{extractResult: "extracted stuff"}
		surfacer := &fakeSurfacer{result: "memories"}

		orch := recall.NewOrchestrator(finder, reader, summarizer, surfacer)

		result, err := orch.Recall(context.Background(), "/proj", "original query")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(surfacer.query).To(Equal("original query"))
		g.Expect(result.Memories).To(Equal("memories"))
	})
}

// failAfterNWriter succeeds for the first `remaining` bytes, then fails.
type failAfterNWriter struct {
	remaining int
}

func (w *failAfterNWriter) Write(p []byte) (int, error) {
	if w.remaining <= 0 {
		return 0, errors.New("write failed")
	}

	n := min(len(p), w.remaining)
	w.remaining -= n

	return n, nil
}

type failWriter struct{}

func (w *failWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

// --- Fakes ---

type fakeFinder struct {
	paths []string
	err   error
}

func (f *fakeFinder) Find(_ string) ([]string, error) {
	return f.paths, f.err
}

type fakeReader struct {
	contents map[string]string
	sizes    map[string]int
	errs     map[string]error
}

func (r *fakeReader) Read(path string, _ int) (string, int, error) {
	if r.errs != nil {
		if err, ok := r.errs[path]; ok {
			return "", 0, err
		}
	}

	content := r.contents[path]
	size := r.sizes[path]

	return content, size, nil
}

type fakeSummarizer struct {
	extractResult string
	extractErr    error
	extractCalls  atomic.Int32
}

func (s *fakeSummarizer) ExtractRelevant(_ context.Context, _, _ string) (string, error) {
	s.extractCalls.Add(1)

	return s.extractResult, s.extractErr
}

type fakeSurfacer struct {
	result string
	err    error
	query  string
}

func (s *fakeSurfacer) Surface(query string) (string, error) {
	s.query = query

	return s.result, s.err
}
