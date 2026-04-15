package recall_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/recall"
)

func TestDefaultExtractCap(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// 10KB gives mode B enough budget for meaningful cross-session context.
	// Mode A uses 15KB raw; mode B extracts are denser, so 10KB is proportional.
	const expectedExtractCap = 10 * 1024
	g.Expect(recall.DefaultExtractCap).To(Equal(expectedExtractCap))
}

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

func TestOrchestrator_ModeB_IncludesMemories(t *testing.T) {
	t.Parallel()

	t.Run("memories are accumulated into summarizer input", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/a.jsonl", Mtime: now},
		}}
		reader := &fakeReader{
			contents: map[string]string{"/a.jsonl": "session content"},
			sizes:    map[string]int{"/a.jsonl": 15},
		}

		memLister := &fakeMemoryLister{memories: []*memory.Stored{
			{
				Type: "feedback", Situation: "Testing", Source: "human",
				Content:   memory.ContentFields{Behavior: "b", Action: "a"},
				UpdatedAt: now, FilePath: "/data/memory/feedback/testing.toml",
			},
		}}

		summarizer := &capturingSummarizer{extractResult: "summary with memories"}

		orch := recall.NewOrchestrator(finder, reader, summarizer, memLister, "/data")

		result, err := orch.Recall(context.Background(), "/proj", "test query")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		// Memories should be in the summarizer input, not as a separate field.
		g.Expect(summarizer.lastContent).To(ContainSubstring("session content"))
		g.Expect(summarizer.lastContent).To(ContainSubstring("[feedback]"))
		g.Expect(summarizer.lastContent).To(ContainSubstring("Testing"))
		g.Expect(result.Summary).To(Equal("summary with memories"))
	})

	t.Run("mode B with nil summarizer returns empty", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/a.jsonl", Mtime: now},
		}}
		reader := &fakeReader{
			contents: map[string]string{"/a.jsonl": "content"},
			sizes:    map[string]int{"/a.jsonl": 7},
		}

		memLister := &fakeMemoryLister{memories: []*memory.Stored{
			{
				Type: "feedback", Situation: "something", Source: "human",
				FilePath: "/data/memory/feedback/something.toml",
			},
		}}

		orch := recall.NewOrchestrator(finder, reader, nil, memLister, "/data")

		result, err := orch.Recall(context.Background(), "/proj", "query")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(BeEmpty())
		g.Expect(result.Memories).To(BeEmpty())
	})
}

func TestOrchestrator_RecallMemoriesOnly(t *testing.T) {
	t.Parallel()

	t.Run("returns matched memories from fake summarizer", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		memLister := &fakeMemoryLister{memories: []*memory.Stored{
			{
				Type:      "feedback",
				Situation: "When testing",
				Source:    "human",
				Content:   memory.ContentFields{Behavior: "skipping tests", Action: "run tests first"},
				UpdatedAt: now.Add(-time.Hour),
				FilePath:  "/data/memory/feedback/test-first.toml",
			},
			{
				Type:      "fact",
				Situation: "About Go",
				Source:    "agent",
				Content:   memory.ContentFields{Subject: "Go", Predicate: "uses", Object: "goroutines"},
				UpdatedAt: now.Add(-2 * time.Hour),
				FilePath:  "/data/memory/facts/go-goroutines.toml",
			},
			{
				Type:      "fact",
				Situation: "About Python",
				Source:    "agent",
				Content:   memory.ContentFields{Subject: "Python"},
				UpdatedAt: now.Add(-3 * time.Hour),
				FilePath:  "/data/memory/facts/python.toml",
			},
		}}

		// Summarizer returns only the first two names as relevant.
		summarizer := &fakeSummarizer{extractResult: "test-first\ngo-goroutines"}

		orch := recall.NewOrchestrator(nil, nil, summarizer, memLister, "/data")

		result, err := orch.RecallMemoriesOnly(context.Background(), "testing", 0)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Memories).To(ContainSubstring("[feedback]"))
		g.Expect(result.Memories).To(ContainSubstring("When testing"))
		g.Expect(result.Memories).NotTo(ContainSubstring("Python"))
	})

	t.Run("nil summarizer returns empty", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		memLister := &fakeMemoryLister{memories: []*memory.Stored{
			{
				Type:      "feedback",
				Situation: "something",
				Source:    "human",
				FilePath:  "/data/memory/feedback/something.toml",
			},
		}}

		orch := recall.NewOrchestrator(nil, nil, nil, memLister, "/data")

		result, err := orch.RecallMemoriesOnly(context.Background(), "query", 0)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(BeEmpty())
		g.Expect(result.Memories).To(BeEmpty())
	})

	t.Run("empty memory list returns empty", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		memLister := &fakeMemoryLister{memories: []*memory.Stored{}}
		summarizer := &fakeSummarizer{extractResult: "anything"}

		orch := recall.NewOrchestrator(nil, nil, summarizer, memLister, "/data")

		result, err := orch.RecallMemoriesOnly(context.Background(), "query", 0)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(BeEmpty())
		g.Expect(result.Memories).To(BeEmpty())
	})

	t.Run("nil memory lister returns empty", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		summarizer := &fakeSummarizer{extractResult: "anything"}

		orch := recall.NewOrchestrator(nil, nil, summarizer, nil, "/data")

		result, err := orch.RecallMemoriesOnly(context.Background(), "query", 0)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(BeEmpty())
		g.Expect(result.Memories).To(BeEmpty())
	})

	t.Run("respects limit", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		memLister := &fakeMemoryLister{memories: []*memory.Stored{
			{
				Type: "fact", Situation: "A", Source: "human",
				Content: memory.ContentFields{Subject: "A"}, UpdatedAt: now,
				FilePath: "/data/memory/facts/a.toml",
			},
			{
				Type: "fact", Situation: "B", Source: "human",
				Content: memory.ContentFields{Subject: "B"}, UpdatedAt: now.Add(-time.Hour),
				FilePath: "/data/memory/facts/b.toml",
			},
			{
				Type: "fact", Situation: "C", Source: "human",
				Content: memory.ContentFields{Subject: "C"}, UpdatedAt: now.Add(-2 * time.Hour),
				FilePath: "/data/memory/facts/c.toml",
			},
		}}

		// Summarizer returns all three names.
		summarizer := &fakeSummarizer{extractResult: "a\nb\nc"}

		orch := recall.NewOrchestrator(nil, nil, summarizer, memLister, "/data")

		const limitTwo = 2

		result, err := orch.RecallMemoriesOnly(context.Background(), "query", limitTwo)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		// Should contain A and B but not C.
		g.Expect(result.Memories).To(ContainSubstring("subject: A"))
		g.Expect(result.Memories).To(ContainSubstring("subject: B"))
		g.Expect(result.Memories).NotTo(ContainSubstring("subject: C"))
	})

	t.Run("builds correct index for summarizer", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		memLister := &fakeMemoryLister{memories: []*memory.Stored{
			{
				Type: "feedback", Situation: "When coding", Source: "human",
				Content:  memory.ContentFields{Behavior: "b"},
				FilePath: "/data/memory/feedback/coding.toml", UpdatedAt: now,
			},
		}}

		summarizer := &capturingSummarizer{extractResult: "coding"}

		orch := recall.NewOrchestrator(nil, nil, summarizer, memLister, "/data")

		_, err := orch.RecallMemoriesOnly(context.Background(), "test query", 0)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		// Verify the index was built correctly.
		g.Expect(summarizer.lastContent).To(ContainSubstring("feedback | coding | When coding"))
		// Verify the query was passed correctly.
		g.Expect(summarizer.lastQuery).To(ContainSubstring("test query"))
		g.Expect(summarizer.lastQuery).To(ContainSubstring("Max 10 names"))
	})
}

func TestOrchestrator_RecallMemoriesOnly_Ranking(t *testing.T) {
	t.Parallel()

	t.Run("human-sourced before agent-sourced, recent before old", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		memLister := &fakeMemoryLister{memories: []*memory.Stored{
			{
				Type: "fact", Situation: "Agent old", Source: "agent",
				Content: memory.ContentFields{Subject: "AgentOld"}, UpdatedAt: now.Add(-3 * time.Hour),
				FilePath: "/data/memory/facts/agent-old.toml",
			},
			{
				Type: "fact", Situation: "Human old", Source: "human",
				Content: memory.ContentFields{Subject: "HumanOld"}, UpdatedAt: now.Add(-2 * time.Hour),
				FilePath: "/data/memory/facts/human-old.toml",
			},
			{
				Type: "fact", Situation: "Agent new", Source: "agent",
				Content: memory.ContentFields{Subject: "AgentNew"}, UpdatedAt: now.Add(-time.Hour),
				FilePath: "/data/memory/facts/agent-new.toml",
			},
			{
				Type: "fact", Situation: "Human new", Source: "human",
				Content: memory.ContentFields{Subject: "HumanNew"}, UpdatedAt: now,
				FilePath: "/data/memory/facts/human-new.toml",
			},
		}}

		summarizer := &fakeSummarizer{
			extractResult: "agent-old\nhuman-old\nagent-new\nhuman-new",
		}

		orch := recall.NewOrchestrator(nil, nil, summarizer, memLister, "/data")

		result, err := orch.RecallMemoriesOnly(context.Background(), "query", 0)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		// Expected order: HumanNew, HumanOld, AgentNew, AgentOld.
		humanNewIdx := strings.Index(result.Memories, "HumanNew")
		humanOldIdx := strings.Index(result.Memories, "HumanOld")
		agentNewIdx := strings.Index(result.Memories, "AgentNew")
		agentOldIdx := strings.Index(result.Memories, "AgentOld")

		g.Expect(humanNewIdx).To(BeNumerically("<", humanOldIdx),
			"human new should come before human old")
		g.Expect(humanOldIdx).To(BeNumerically("<", agentNewIdx),
			"human old should come before agent new")
		g.Expect(agentNewIdx).To(BeNumerically("<", agentOldIdx),
			"agent new should come before agent old")
	})
}

func TestOrchestrator_Recall_ModeA(t *testing.T) {
	t.Parallel()

	t.Run("returns raw stripped content without summarization", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/a.jsonl", Mtime: now},
			{Path: "/b.jsonl", Mtime: now.Add(-time.Hour)},
		}}
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

		orch := recall.NewOrchestrator(finder, reader, nil, nil, "")

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(Equal("session a contentsession b content"))
	})

	t.Run("no sessions found returns empty result", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		finder := &fakeFinder{entries: []recall.FileEntry{}}
		orch := recall.NewOrchestrator(finder, nil, nil, nil, "")

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(BeEmpty())
		g.Expect(result.Memories).To(BeEmpty())
	})

	t.Run("reader error skips session and continues", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/bad.jsonl", Mtime: now},
			{Path: "/good.jsonl", Mtime: now.Add(-time.Hour)},
		}}
		reader := &fakeReader{
			contents: map[string]string{"/good.jsonl": "good content"},
			sizes:    map[string]int{"/good.jsonl": 12},
			errs:     map[string]error{"/bad.jsonl": errors.New("read failed")},
		}

		orch := recall.NewOrchestrator(finder, reader, nil, nil, "")

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

		now := time.Now()
		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/a.jsonl", Mtime: now},
			{Path: "/b.jsonl", Mtime: now.Add(-time.Hour)},
		}}
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

		orch := recall.NewOrchestrator(finder, reader, nil, nil, "")

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
		orch := recall.NewOrchestrator(finder, nil, nil, nil, "")

		_, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).To(HaveOccurred())

		if err != nil {
			g.Expect(err.Error()).To(ContainSubstring("recalling"))
		}
	})
}

func TestOrchestrator_Recall_ModeA_CancellationStopsProcessing(t *testing.T) {
	t.Parallel()

	t.Run("pre-cancelled context returns early", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		const totalSessions = 10

		now := time.Now()
		entries := make([]recall.FileEntry, 0, totalSessions)
		contents := make(map[string]string, totalSessions)
		sizes := make(map[string]int, totalSessions)

		for i := range totalSessions {
			path := fmt.Sprintf("/s%d.jsonl", i)
			entries = append(entries, recall.FileEntry{
				Path:  path,
				Mtime: now.Add(-time.Duration(i) * time.Hour),
			})
			contents[path] = "content"
			sizes[path] = 7
		}

		finder := &fakeFinder{entries: entries}
		reader := &countingReader{
			contents: contents,
			sizes:    sizes,
		}

		orch := recall.NewOrchestrator(finder, reader, nil, nil, "")

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // pre-cancel

		result, err := orch.Recall(ctx, "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		// With a pre-cancelled context, mode A should read zero sessions.
		g.Expect(result.Summary).To(BeEmpty())
		g.Expect(int(reader.readCalls.Load())).To(Equal(0))
	})
}

func TestOrchestrator_Recall_ModeA_MemoryFormatting(t *testing.T) {
	t.Parallel()

	t.Run("multiple sessions use inter-session time windows", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		newerMtime := now.Add(-time.Hour)
		olderMtime := now.Add(-3 * time.Hour)

		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/newer.jsonl", Mtime: newerMtime},
			{Path: "/older.jsonl", Mtime: olderMtime},
		}}
		reader := &fakeReader{
			contents: map[string]string{
				"/newer.jsonl": "newer content",
				"/older.jsonl": "older content",
			},
			sizes: map[string]int{
				"/newer.jsonl": 14,
				"/older.jsonl": 14,
			},
		}

		// Memory between the two sessions' mtimes -- within newer session window.
		memLister := &fakeMemoryLister{memories: []*memory.Stored{
			{
				Type:      "feedback",
				Situation: "Between sessions",
				Content:   memory.ContentFields{Behavior: "b", Action: "a"},
				UpdatedAt: now.Add(-2 * time.Hour),
				FilePath:  "/data/memory/feedback/between.toml",
			},
		}}

		orch := recall.NewOrchestrator(finder, reader, nil, memLister, "/data")

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Memories).To(ContainSubstring("[feedback]"))
		g.Expect(result.Memories).To(ContainSubstring("Between sessions"))
	})

	t.Run("formats fact memory with partial fields", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		sessionMtime := now.Add(-time.Hour)

		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/session.jsonl", Mtime: sessionMtime},
		}}
		reader := &fakeReader{
			contents: map[string]string{"/session.jsonl": "content"},
			sizes:    map[string]int{"/session.jsonl": 7},
		}

		// Fact with only subject (no predicate/object).
		memLister := &fakeMemoryLister{memories: []*memory.Stored{
			{
				Type:      "fact",
				Situation: "About Go",
				Content:   memory.ContentFields{Subject: "Go"},
				UpdatedAt: sessionMtime.Add(-time.Hour),
				FilePath:  "/data/memory/facts/go.toml",
			},
		}}

		orch := recall.NewOrchestrator(finder, reader, nil, memLister, "/data")

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Memories).To(ContainSubstring("[fact]"))
		g.Expect(result.Memories).To(ContainSubstring("subject: Go"))
		g.Expect(result.Memories).NotTo(ContainSubstring("predicate"))
		g.Expect(result.Memories).NotTo(ContainSubstring("object"))
	})

	t.Run("formats feedback memory with partial fields", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		sessionMtime := now.Add(-time.Hour)

		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/session.jsonl", Mtime: sessionMtime},
		}}
		reader := &fakeReader{
			contents: map[string]string{"/session.jsonl": "content"},
			sizes:    map[string]int{"/session.jsonl": 7},
		}

		// Feedback with only action (no behavior).
		memLister := &fakeMemoryLister{memories: []*memory.Stored{
			{
				Type:      "feedback",
				Situation: "When coding",
				Content:   memory.ContentFields{Action: "use DI"},
				UpdatedAt: sessionMtime.Add(-time.Hour),
				FilePath:  "/data/memory/feedback/di.toml",
			},
		}}

		orch := recall.NewOrchestrator(finder, reader, nil, memLister, "/data")

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Memories).To(ContainSubstring("[feedback]"))
		g.Expect(result.Memories).To(ContainSubstring("action: use DI"))
		g.Expect(result.Memories).NotTo(ContainSubstring("behavior"))
	})
}

func TestOrchestrator_Recall_ModeA_MemoryWindowing(t *testing.T) {
	t.Parallel()

	t.Run("includes memories within session time window", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		sessionMtime := now.Add(-time.Hour)

		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/session.jsonl", Mtime: sessionMtime},
		}}
		reader := &fakeReader{
			contents: map[string]string{"/session.jsonl": "session content"},
			sizes:    map[string]int{"/session.jsonl": 15},
		}

		// Memory updated within the session window (24h before session mtime).
		memLister := &fakeMemoryLister{memories: []*memory.Stored{
			{
				Type:      "feedback",
				Situation: "When running tests",
				Content:   memory.ContentFields{Behavior: "running go test directly", Action: "use targ test instead"},
				UpdatedAt: sessionMtime.Add(-2 * time.Hour),
				FilePath:  "/data/memory/feedback/use-targ.toml",
			},
			{
				Type:      "fact",
				Situation: "When building engram",
				Content:   memory.ContentFields{Subject: "DI", Predicate: "means", Object: "Dependency Injection"},
				UpdatedAt: sessionMtime.Add(-3 * time.Hour),
				FilePath:  "/data/memory/facts/di.toml",
			},
		}}

		orch := recall.NewOrchestrator(finder, reader, nil, memLister, "/data")

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(Equal("session content"))
		g.Expect(result.Memories).To(ContainSubstring("[feedback]"))
		g.Expect(result.Memories).To(ContainSubstring("use targ test instead"))
		g.Expect(result.Memories).To(ContainSubstring("[fact]"))
		g.Expect(result.Memories).To(ContainSubstring("Dependency Injection"))
	})

	t.Run("nil memory lister works as before", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/session.jsonl", Mtime: now},
		}}
		reader := &fakeReader{
			contents: map[string]string{"/session.jsonl": "session content"},
			sizes:    map[string]int{"/session.jsonl": 15},
		}

		orch := recall.NewOrchestrator(finder, reader, nil, nil, "")

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(Equal("session content"))
		g.Expect(result.Memories).To(BeEmpty())
	})

	t.Run("excludes memories outside session time window", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		sessionMtime := now.Add(-time.Hour)

		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/session.jsonl", Mtime: sessionMtime},
		}}
		reader := &fakeReader{
			contents: map[string]string{"/session.jsonl": "session content"},
			sizes:    map[string]int{"/session.jsonl": 15},
		}

		// Memory updated well outside the 24h window.
		memLister := &fakeMemoryLister{memories: []*memory.Stored{
			{
				Type:      "feedback",
				Situation: "Old feedback",
				Content:   memory.ContentFields{Behavior: "old", Action: "old action"},
				UpdatedAt: sessionMtime.Add(-48 * time.Hour),
				FilePath:  "/data/memory/feedback/old.toml",
			},
		}}

		orch := recall.NewOrchestrator(finder, reader, nil, memLister, "/data")

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Memories).To(BeEmpty())
	})

	t.Run("memory lister error returns empty memories", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/session.jsonl", Mtime: now},
		}}
		reader := &fakeReader{
			contents: map[string]string{"/session.jsonl": "session content"},
			sizes:    map[string]int{"/session.jsonl": 15},
		}

		memLister := &fakeMemoryLister{err: errors.New("disk error")}

		orch := recall.NewOrchestrator(finder, reader, nil, memLister, "/data")

		result, err := orch.Recall(context.Background(), "/proj", "")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(Equal("session content"))
		g.Expect(result.Memories).To(BeEmpty())
	})
}

func TestOrchestrator_Recall_ModeB(t *testing.T) {
	t.Parallel()

	t.Run("accumulates sessions then makes one summarizer call", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/a.jsonl", Mtime: now},
			{Path: "/b.jsonl", Mtime: now.Add(-time.Hour)},
		}}
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
		summarizer := &capturingSummarizer{extractResult: "relevant summary"}

		orch := recall.NewOrchestrator(finder, reader, summarizer, nil, "")

		result, err := orch.Recall(context.Background(), "/proj", "my query")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		// Exactly 1 summarizer call with accumulated content from both sessions.
		g.Expect(int(summarizer.extractCalls.Load())).To(Equal(1))
		g.Expect(summarizer.lastContent).To(ContainSubstring("session a"))
		g.Expect(summarizer.lastContent).To(ContainSubstring("session b"))
		g.Expect(result.Summary).To(Equal("relevant summary"))
	})

	t.Run("includes per-session memories in accumulated content", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		sessionMtime := now.Add(-time.Hour)

		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/a.jsonl", Mtime: sessionMtime},
		}}
		reader := &fakeReader{
			contents: map[string]string{"/a.jsonl": "session content"},
			sizes:    map[string]int{"/a.jsonl": 15},
		}

		// Memory within the session's time window.
		memLister := &fakeMemoryLister{memories: []*memory.Stored{
			{
				Type: "feedback", Situation: "When testing",
				Content:   memory.ContentFields{Behavior: "b", Action: "a"},
				UpdatedAt: sessionMtime.Add(-2 * time.Hour),
				FilePath:  "/data/memory/feedback/testing.toml",
			},
		}}

		summarizer := &capturingSummarizer{extractResult: "summary"}

		orch := recall.NewOrchestrator(finder, reader, summarizer, memLister, "/data")

		result, err := orch.Recall(context.Background(), "/proj", "query")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		// Summarizer should receive session content AND memories.
		g.Expect(summarizer.lastContent).To(ContainSubstring("session content"))
		g.Expect(summarizer.lastContent).To(ContainSubstring("[feedback]"))
		g.Expect(summarizer.lastContent).To(ContainSubstring("When testing"))
		g.Expect(result.Summary).To(Equal("summary"))
	})

	t.Run("stops accumulating when input budget reached", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/a.jsonl", Mtime: now},
			{Path: "/b.jsonl", Mtime: now.Add(-time.Hour)},
		}}
		reader := &fakeReader{
			contents: map[string]string{
				"/a.jsonl": "big content",
				"/b.jsonl": "should not appear",
			},
			sizes: map[string]int{
				"/a.jsonl": recall.DefaultModeBInputBudget, // Exactly at budget
				"/b.jsonl": 100,
			},
		}
		summarizer := &capturingSummarizer{extractResult: "summary"}

		orch := recall.NewOrchestrator(finder, reader, summarizer, nil, "")

		result, err := orch.Recall(context.Background(), "/proj", "query")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		// Only first session should be in the accumulated content.
		g.Expect(summarizer.lastContent).To(ContainSubstring("big content"))
		g.Expect(summarizer.lastContent).NotTo(ContainSubstring("should not appear"))
		g.Expect(result.Summary).To(Equal("summary"))
	})

	t.Run("reader error skips session and continues", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/bad.jsonl", Mtime: now},
			{Path: "/good.jsonl", Mtime: now.Add(-time.Hour)},
		}}
		reader := &fakeReader{
			contents: map[string]string{"/good.jsonl": "good"},
			sizes:    map[string]int{"/good.jsonl": 4},
			errs:     map[string]error{"/bad.jsonl": errors.New("read err")},
		}
		summarizer := &capturingSummarizer{extractResult: "extracted"}

		orch := recall.NewOrchestrator(finder, reader, summarizer, nil, "")

		result, err := orch.Recall(context.Background(), "/proj", "query")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(int(summarizer.extractCalls.Load())).To(Equal(1))
		g.Expect(summarizer.lastContent).To(ContainSubstring("good"))
		g.Expect(result.Summary).To(Equal("extracted"))
	})

	t.Run("summarizer error propagates", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/a.jsonl", Mtime: now},
		}}
		reader := &fakeReader{
			contents: map[string]string{"/a.jsonl": "content"},
			sizes:    map[string]int{"/a.jsonl": 7},
		}
		summarizer := &fakeSummarizer{
			extractErr: errors.New("summarize err"),
		}

		orch := recall.NewOrchestrator(finder, reader, summarizer, nil, "")

		_, err := orch.Recall(context.Background(), "/proj", "query")
		g.Expect(err).To(HaveOccurred())

		if err != nil {
			g.Expect(err.Error()).To(ContainSubstring("summariz"))
		}
	})

	t.Run("nil summarizer returns empty result", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/a.jsonl", Mtime: now},
		}}
		reader := &fakeReader{
			contents: map[string]string{"/a.jsonl": "content"},
			sizes:    map[string]int{"/a.jsonl": 7},
		}

		orch := recall.NewOrchestrator(finder, reader, nil, nil, "")

		result, err := orch.Recall(context.Background(), "/proj", "query")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(BeEmpty())
	})

	t.Run("all sessions unreadable returns empty result", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/bad.jsonl", Mtime: now},
		}}
		reader := &fakeReader{
			errs: map[string]error{"/bad.jsonl": errors.New("read err")},
		}
		summarizer := &capturingSummarizer{extractResult: "anything"}

		orch := recall.NewOrchestrator(finder, reader, summarizer, nil, "")

		result, err := orch.Recall(context.Background(), "/proj", "query")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		// No content accumulated → no summarizer call → empty result.
		g.Expect(int(summarizer.extractCalls.Load())).To(Equal(0))
		g.Expect(result.Summary).To(BeEmpty())
	})
}

func TestOrchestrator_Recall_ModeB_CancellationStopsProcessing(t *testing.T) {
	t.Parallel()

	t.Run("pre-cancelled context returns empty without summarizing", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		const totalSessions = 10

		now := time.Now()
		entries := make([]recall.FileEntry, 0, totalSessions)
		contents := make(map[string]string, totalSessions)
		sizes := make(map[string]int, totalSessions)

		for i := range totalSessions {
			path := fmt.Sprintf("/s%d.jsonl", i)
			entries = append(entries, recall.FileEntry{
				Path:  path,
				Mtime: now.Add(-time.Duration(i) * time.Hour),
			})
			contents[path] = "content"
			sizes[path] = 7
		}

		finder := &fakeFinder{entries: entries}
		reader := &countingReader{contents: contents, sizes: sizes}
		summarizer := &capturingSummarizer{extractResult: "anything"}

		orch := recall.NewOrchestrator(finder, reader, summarizer, nil, "")

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // pre-cancel

		result, err := orch.Recall(ctx, "/proj", "query")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result.Summary).To(BeEmpty())
		g.Expect(int(reader.readCalls.Load())).To(Equal(0))
		g.Expect(int(summarizer.extractCalls.Load())).To(Equal(0))
	})
}

// capturingSummarizer records content and query for inspection.
type capturingSummarizer struct {
	extractResult string
	extractErr    error
	lastContent   string
	lastQuery     string
	extractCalls  atomic.Int32
}

func (s *capturingSummarizer) ExtractRelevant(
	_ context.Context, content, query string,
) (string, error) {
	s.extractCalls.Add(1)
	s.lastContent = content
	s.lastQuery = query

	return s.extractResult, s.extractErr
}

// countingReader counts Read calls to verify early exit.
type countingReader struct {
	contents  map[string]string
	sizes     map[string]int
	readCalls atomic.Int32
}

func (r *countingReader) Read(path string, budget int) (string, int, error) {
	r.readCalls.Add(1)

	_ = budget

	return r.contents[path], r.sizes[path], nil
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
	entries []recall.FileEntry
	err     error
}

func (f *fakeFinder) Find(_ string) ([]recall.FileEntry, error) {
	return f.entries, f.err
}

type fakeMemoryLister struct {
	memories []*memory.Stored
	err      error
}

func (f *fakeMemoryLister) ListAllMemories(_ string) ([]*memory.Stored, error) {
	return f.memories, f.err
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
