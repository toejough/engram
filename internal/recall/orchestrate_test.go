package recall_test

import (
	"context"
	"errors"
	"testing"

	"engram/internal/recall"

	. "github.com/onsi/gomega"
)

// --- Tests ---

func TestOrchestrator_Recall_ModeA(t *testing.T) {
	t.Parallel()

	t.Run("finds reads summarizes and surfaces", func(t *testing.T) {
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
		summarizer := &fakeSummarizer{summarizeResult: "summary of sessions"}
		surfacer := &fakeSurfacer{result: "relevant memories"}

		orch := recall.NewOrchestrator(finder, reader, summarizer, surfacer)

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(Equal("summary of sessions"))
		g.Expect(result.Memories).To(Equal("relevant memories"))
		g.Expect(surfacer.query).To(Equal("summary of sessions"))
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

	t.Run("surfacer error still returns summary", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		finder := &fakeFinder{paths: []string{"/a.jsonl"}}
		reader := &fakeReader{
			contents: map[string]string{"/a.jsonl": "content"},
			sizes:    map[string]int{"/a.jsonl": 7},
		}
		summarizer := &fakeSummarizer{summarizeResult: "the summary"}
		surfacer := &fakeSurfacer{err: errors.New("surfacer broke")}

		orch := recall.NewOrchestrator(finder, reader, summarizer, surfacer)

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(Equal("the summary"))
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
		summarizer := &fakeSummarizer{summarizeResult: "summary"}

		orch := recall.NewOrchestrator(finder, reader, summarizer, nil)

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(Equal("summary"))
	})

	t.Run("summarize error propagates", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		finder := &fakeFinder{paths: []string{"/a.jsonl"}}
		reader := &fakeReader{
			contents: map[string]string{"/a.jsonl": "content"},
			sizes:    map[string]int{"/a.jsonl": 7},
		}
		summarizer := &fakeSummarizer{summarizeErr: errors.New("llm down")}

		orch := recall.NewOrchestrator(finder, reader, summarizer, nil)

		_, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).To(HaveOccurred())

		if err != nil {
			g.Expect(err.Error()).To(ContainSubstring("recalling"))
		}
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
				"/a.jsonl": recall.DefaultStripBudget, // Exactly at budget
				"/b.jsonl": 100,
			},
		}
		summarizer := &fakeSummarizer{summarizeResult: "summary"}

		orch := recall.NewOrchestrator(finder, reader, summarizer, nil)

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		// Only first session's content should be in the summary input.
		g.Expect(result.Summary).To(Equal("summary"))
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
		summarizer := &fakeSummarizer{summarizeResult: "the summary"}

		orch := recall.NewOrchestrator(finder, reader, summarizer, nil)

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(Equal("the summary"))
		g.Expect(result.Memories).To(BeEmpty())
	})

	t.Run("nil summarizer with sessions returns raw content", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		finder := &fakeFinder{paths: []string{"/a.jsonl"}}
		reader := &fakeReader{
			contents: map[string]string{"/a.jsonl": "raw content"},
			sizes:    map[string]int{"/a.jsonl": 11},
		}

		orch := recall.NewOrchestrator(finder, reader, nil, nil)

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(Equal("raw content"))
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
		g.Expect(summarizer.extractCalls).To(Equal(2))
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

		// Should stop after 2 sessions (800+800=1600 >= 1500).
		g.Expect(summarizer.extractCalls).To(Equal(2))
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

		// Only the good session gets extracted.
		g.Expect(summarizer.extractCalls).To(Equal(1))
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
	summarizeResult string
	summarizeErr    error
	extractResult   string
	extractErr      error
	extractCalls    int
}

func (s *fakeSummarizer) ExtractRelevant(_ context.Context, _, _ string) (string, error) {
	s.extractCalls++

	return s.extractResult, s.extractErr
}

func (s *fakeSummarizer) Summarize(_ context.Context, _ string) (string, error) {
	return s.summarizeResult, s.summarizeErr
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
