# Recall Query Pipeline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the single-pass accumulate-then-summarize recall pipeline with a three-phase approach: memory search, per-session verbatim extraction, then structured summary.

**Architecture:** Phase 1 searches memories via Haiku. Phase 2 iterates sessions newest-first, calling Haiku per-session to extract verbatim quotes, stopping when the 10KB output buffer fills. Phase 3 summarizes the buffer into structured output. Status messages go to an injected io.Writer.

**Tech Stack:** Go, Haiku API via existing anthropic client, gomega for tests

---

### Task 1: Add `SummarizeFindings` to Summarizer

**Files:**
- Modify: `internal/recall/summarize.go:29-49`
- Modify: `internal/recall/summarize_test.go`

- [ ] **Step 1: Write failing test for SummarizeFindings**

In `internal/recall/summarize_test.go`, add:

```go
func TestSummarizeFindings_CallsHaikuCallerWithSummaryPrompt(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	caller := &fakeHaikuCaller{result: "structured summary"}
	summarizer := recall.NewSummarizer(caller)

	result, err := summarizer.SummarizeFindings(
		context.Background(),
		"memory excerpts and session snippets",
		"targ argument parsing",
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(caller.called).To(BeTrue())
	g.Expect(caller.systemPrompt).To(ContainSubstring("structured summary"))
	g.Expect(caller.userPrompt).To(ContainSubstring("targ argument parsing"))
	g.Expect(caller.userPrompt).To(ContainSubstring("memory excerpts"))
	g.Expect(result).To(Equal("structured summary"))
}

func TestSummarizeFindings_ReturnsErrorOnCallerFailure(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	caller := &fakeHaikuCaller{err: errors.New("rate limited")}
	summarizer := recall.NewSummarizer(caller)

	_, err := summarizer.SummarizeFindings(context.Background(), "content", "query")
	g.Expect(err).To(MatchError(ContainSubstring("rate limited")))
	g.Expect(err).To(MatchError(ContainSubstring("summarizing findings")))
}

func TestSummarizeFindings_NilCallerReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	summarizer := recall.NewSummarizer(nil)

	_, err := summarizer.SummarizeFindings(context.Background(), "content", "query")
	g.Expect(err).To(MatchError(recall.ErrNilCaller))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/recall/ -run TestSummarizeFindings -count=1`
Expected: compilation error — `SummarizeFindings` not defined

- [ ] **Step 3: Implement SummarizeFindings**

In `internal/recall/summarize.go`, add the method and constant:

```go
// SummarizeFindings produces a structured summary from accumulated findings.
func (s *Summarizer) SummarizeFindings(ctx context.Context, content, query string) (string, error) {
	if s.caller == nil {
		return "", ErrNilCaller
	}

	userPrompt := "Query: " + query + "\n\nFindings:\n" + content

	result, err := s.caller.Call(ctx, summarizeFindingsPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("summarizing findings: %w", err)
	}

	return result, nil
}
```

Add the constant alongside `extractSystemPrompt`:

```go
summarizeFindingsPrompt = `Create a structured summary of the following findings ` +
	`relevant to the query. Use markdown headers and bullet points. ` +
	`Preserve specific details, file paths, and code references.`
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/recall/ -run TestSummarizeFindings -count=1`
Expected: PASS

- [ ] **Step 5: Update ExtractRelevant system prompt**

In `internal/recall/summarize.go`, change `extractSystemPrompt` from:

```go
extractSystemPrompt = `Extract only content relevant to the following query. ` +
	`Return relevant excerpts verbatim or tightly paraphrased. Return nothing if irrelevant.`
```

To:

```go
extractSystemPrompt = `Extract only content relevant to the following query. ` +
	`Return relevant excerpts verbatim or very lightly paraphrased in service of ` +
	`grammatical correctness and consistency. Return nothing if irrelevant.`
```

- [ ] **Step 6: Run all summarize tests**

Run: `go test ./internal/recall/ -run "TestExtractRelevant|TestSummarizeFindings" -count=1`
Expected: PASS (existing ExtractRelevant tests still pass — prompt content check uses `ContainSubstring("Extract only content relevant")` which still matches)

- [ ] **Step 7: Commit**

```bash
git add internal/recall/summarize.go internal/recall/summarize_test.go
git commit -m "feat(recall): add SummarizeFindings and refine extraction prompt"
```

---

### Task 2: Add `SummarizeFindings` to `SummarizerI` interface and update fakes

**Files:**
- Modify: `internal/recall/orchestrate.go` (SummarizerI interface, ~line 321)
- Modify: `internal/recall/orchestrate_test.go` (fakes: capturingSummarizer, fakeSummarizer)

- [ ] **Step 1: Add SummarizeFindings to SummarizerI**

In `internal/recall/orchestrate.go`, update the interface:

```go
// SummarizerI extracts relevant content from transcripts via LLM.
type SummarizerI interface {
	ExtractRelevant(ctx context.Context, content, query string) (string, error)
	SummarizeFindings(ctx context.Context, content, query string) (string, error)
}
```

- [ ] **Step 2: Update capturingSummarizer fake**

In `internal/recall/orchestrate_test.go`, add fields and method to `capturingSummarizer`:

```go
type capturingSummarizer struct {
	extractResult        string
	extractErr           error
	lastContent          string
	lastQuery            string
	extractCalls         atomic.Int32
	summarizeResult      string
	summarizeErr         error
	lastSummarizeContent string
	lastSummarizeQuery   string
	summarizeCalls       atomic.Int32
}

func (s *capturingSummarizer) SummarizeFindings(
	_ context.Context, content, query string,
) (string, error) {
	s.summarizeCalls.Add(1)
	s.lastSummarizeContent = content
	s.lastSummarizeQuery = query

	return s.summarizeResult, s.summarizeErr
}
```

- [ ] **Step 3: Update fakeSummarizer fake**

Add the method to `fakeSummarizer`:

```go
func (s *fakeSummarizer) SummarizeFindings(_ context.Context, _, _ string) (string, error) {
	return s.extractResult, s.extractErr
}
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/recall/`
Expected: compiles cleanly

- [ ] **Step 5: Run all recall tests**

Run: `go test ./internal/recall/ -count=1`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/recall/orchestrate.go internal/recall/orchestrate_test.go
git commit -m "refactor(recall): add SummarizeFindings to SummarizerI interface"
```

---

### Task 3: Add `OrchestratorOption` and `WithStatusWriter`

**Files:**
- Modify: `internal/recall/orchestrate.go` (Orchestrator struct, NewOrchestrator)

- [ ] **Step 1: Add statusWriter field and option type**

In `internal/recall/orchestrate.go`, update the Orchestrator struct:

```go
type Orchestrator struct {
	finder       Finder
	reader       Reader
	summarizer   SummarizerI
	memoryLister MemoryLister
	dataDir      string
	statusWriter io.Writer
}
```

Add the option type and constructor:

```go
// OrchestratorOption configures optional Orchestrator dependencies.
type OrchestratorOption func(*Orchestrator)

// WithStatusWriter sets a writer for progress messages during recall.
func WithStatusWriter(w io.Writer) OrchestratorOption {
	return func(o *Orchestrator) {
		o.statusWriter = w
	}
}
```

- [ ] **Step 2: Update NewOrchestrator to accept options**

```go
func NewOrchestrator(
	finder Finder,
	reader Reader,
	summarizer SummarizerI,
	memoryLister MemoryLister,
	dataDir string,
	opts ...OrchestratorOption,
) *Orchestrator {
	orch := &Orchestrator{
		finder:       finder,
		reader:       reader,
		summarizer:   summarizer,
		memoryLister: memoryLister,
		dataDir:      dataDir,
	}

	for _, opt := range opts {
		opt(orch)
	}

	return orch
}
```

Add a helper for writing status:

```go
// writeStatus writes a progress message if a status writer is configured.
func (o *Orchestrator) writeStatus(format string, args ...any) {
	if o.statusWriter == nil {
		return
	}

	fmt.Fprintf(o.statusWriter, format+"\n", args...)
}
```

- [ ] **Step 3: Verify compilation and tests pass**

Run: `go test ./internal/recall/ -count=1`
Expected: PASS (variadic opts don't break existing callers)

- [ ] **Step 4: Commit**

```bash
git add internal/recall/orchestrate.go
git commit -m "feat(recall): add OrchestratorOption and WithStatusWriter"
```

---

### Task 4: Rewrite `recallModeB` with three-phase pipeline

**Files:**
- Modify: `internal/recall/orchestrate.go` (recallModeB, remove accumulateSessionsAndMemories)
- Modify: `internal/recall/orchestrate_test.go` (rewrite ModeB tests)

- [ ] **Step 1: Write failing tests for the new pipeline**

Replace the existing `TestOrchestrator_Recall_ModeB` and `TestOrchestrator_ModeB_IncludesMemories` test functions. The new tests verify the three-phase pipeline:

```go
func TestOrchestrator_Recall_ModeB(t *testing.T) {
	t.Parallel()

	t.Run("phase 1 searches memories then phase 2 extracts from sessions", func(t *testing.T) {
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
			sizes: map[string]int{"/a.jsonl": 17, "/b.jsonl": 17},
		}

		memLister := &fakeMemoryLister{memories: []*memory.Stored{
			{
				Type: "feedback", Situation: "When testing",
				Content:   memory.ContentFields{Behavior: "b", Action: "a"},
				UpdatedAt: now, FilePath: "/data/memory/feedback/testing.toml",
			},
		}}

		summarizer := &pipelineSummarizer{
			// First ExtractRelevant call: memory matching returns the name.
			// Subsequent calls: per-session extraction returns snippets.
			extractResults: []string{"testing", "snippet from a", "snippet from b"},
			summarizeResult: "final structured summary",
		}

		orch := recall.NewOrchestrator(finder, reader, summarizer, memLister, "/data")

		result, err := orch.Recall(context.Background(), "/proj", "my query")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		// Phase 1: 1 call for memory matching.
		// Phase 2: 2 calls for per-session extraction.
		g.Expect(int(summarizer.extractCalls.Load())).To(Equal(3))
		// Phase 3: 1 summarize call.
		g.Expect(int(summarizer.summarizeCalls.Load())).To(Equal(1))
		// Final summary should contain memories + snippets.
		g.Expect(summarizer.lastSummarizeContent).To(ContainSubstring("[feedback]"))
		g.Expect(summarizer.lastSummarizeContent).To(ContainSubstring("snippet from a"))
		g.Expect(summarizer.lastSummarizeContent).To(ContainSubstring("snippet from b"))
		g.Expect(result.Summary).To(Equal("final structured summary"))
	})

	t.Run("stops per-session extraction when buffer full", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		const bigSnippet = 11 * 1024 // > DefaultExtractCap

		bigContent := make([]byte, bigSnippet)
		for i := range bigContent {
			bigContent[i] = 'x'
		}

		now := time.Now()
		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/a.jsonl", Mtime: now},
			{Path: "/b.jsonl", Mtime: now.Add(-time.Hour)},
		}}
		reader := &fakeReader{
			contents: map[string]string{
				"/a.jsonl": "a", "/b.jsonl": "b",
			},
			sizes: map[string]int{"/a.jsonl": 1, "/b.jsonl": 1},
		}

		summarizer := &pipelineSummarizer{
			extractResults:  []string{string(bigContent), "should not appear"},
			summarizeResult: "summary",
		}

		orch := recall.NewOrchestrator(finder, reader, summarizer, nil, "")

		result, err := orch.Recall(context.Background(), "/proj", "query")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		// Only 1 extract call — buffer full after first session.
		g.Expect(int(summarizer.extractCalls.Load())).To(Equal(1))
		g.Expect(summarizer.lastSummarizeContent).NotTo(ContainSubstring("should not appear"))
		g.Expect(result.Summary).To(Equal("summary"))
	})

	t.Run("skips sessions with empty extraction", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/empty.jsonl", Mtime: now},
			{Path: "/good.jsonl", Mtime: now.Add(-time.Hour)},
		}}
		reader := &fakeReader{
			contents: map[string]string{
				"/empty.jsonl": "irrelevant", "/good.jsonl": "relevant",
			},
			sizes: map[string]int{"/empty.jsonl": 10, "/good.jsonl": 8},
		}

		summarizer := &pipelineSummarizer{
			extractResults:  []string{"", "good snippet"},
			summarizeResult: "summary",
		}

		orch := recall.NewOrchestrator(finder, reader, summarizer, nil, "")

		result, err := orch.Recall(context.Background(), "/proj", "query")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(int(summarizer.extractCalls.Load())).To(Equal(2))
		g.Expect(summarizer.lastSummarizeContent).To(ContainSubstring("good snippet"))
		g.Expect(summarizer.lastSummarizeContent).NotTo(ContainSubstring("irrelevant"))
		g.Expect(result.Summary).To(Equal("summary"))
	})

	t.Run("empty buffer returns empty result without summarizing", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		now := time.Now()
		finder := &fakeFinder{entries: []recall.FileEntry{
			{Path: "/a.jsonl", Mtime: now},
		}}
		reader := &fakeReader{
			contents: map[string]string{"/a.jsonl": "stuff"},
			sizes:    map[string]int{"/a.jsonl": 5},
		}

		summarizer := &pipelineSummarizer{
			extractResults:  []string{""},
			summarizeResult: "should not be called",
		}

		orch := recall.NewOrchestrator(finder, reader, summarizer, nil, "")

		result, err := orch.Recall(context.Background(), "/proj", "query")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(int(summarizer.summarizeCalls.Load())).To(Equal(0))
		g.Expect(result.Summary).To(BeEmpty())
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

	t.Run("summarize error propagates", func(t *testing.T) {
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

		summarizer := &pipelineSummarizer{
			extractResults: []string{"snippet"},
			summarizeErr:   errors.New("api down"),
		}

		orch := recall.NewOrchestrator(finder, reader, summarizer, nil, "")

		_, err := orch.Recall(context.Background(), "/proj", "query")
		g.Expect(err).To(HaveOccurred())

		if err != nil {
			g.Expect(err.Error()).To(ContainSubstring("summariz"))
		}
	})

	t.Run("writes status messages when status writer configured", func(t *testing.T) {
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

		summarizer := &pipelineSummarizer{
			extractResults:  []string{"snippet"},
			summarizeResult: "summary",
		}

		var statusBuf bytes.Buffer

		orch := recall.NewOrchestrator(finder, reader, summarizer, nil, "",
			recall.WithStatusWriter(&statusBuf))

		_, err := orch.Recall(context.Background(), "/proj", "query")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		status := statusBuf.String()
		g.Expect(status).To(ContainSubstring("memor"))
		g.Expect(status).To(ContainSubstring("snippet"))
		g.Expect(status).To(ContainSubstring("summar"))
	})
}
```

Add the `pipelineSummarizer` fake alongside the other fakes:

```go
// pipelineSummarizer returns queued results for successive ExtractRelevant calls
// and a fixed result for SummarizeFindings.
type pipelineSummarizer struct {
	extractResults       []string
	extractErr           error
	extractCalls         atomic.Int32
	summarizeResult      string
	summarizeErr         error
	summarizeCalls       atomic.Int32
	lastSummarizeContent string
	lastSummarizeQuery   string
}

func (s *pipelineSummarizer) ExtractRelevant(
	_ context.Context, _, _ string,
) (string, error) {
	idx := int(s.extractCalls.Add(1)) - 1

	if s.extractErr != nil {
		return "", s.extractErr
	}

	if idx < len(s.extractResults) {
		return s.extractResults[idx], nil
	}

	return "", nil
}

func (s *pipelineSummarizer) SummarizeFindings(
	_ context.Context, content, query string,
) (string, error) {
	s.summarizeCalls.Add(1)
	s.lastSummarizeContent = content
	s.lastSummarizeQuery = query

	return s.summarizeResult, s.summarizeErr
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/recall/ -run TestOrchestrator_Recall_ModeB -count=1`
Expected: FAIL (current implementation doesn't use the pipeline pattern)

- [ ] **Step 3: Implement the three-phase pipeline**

In `internal/recall/orchestrate.go`, replace `recallModeB` and remove `accumulateSessionsAndMemories`:

```go
func (o *Orchestrator) recallModeB(
	ctx context.Context,
	sessions []FileEntry,
	query string,
) (*Result, error) {
	if o.summarizer == nil {
		return &Result{}, nil
	}

	var buffer strings.Builder

	// Phase 1: Search memories.
	memoriesLen := o.searchMemories(ctx, query, &buffer)

	// Phase 2: Per-session verbatim extraction.
	o.extractFromSessions(ctx, sessions, query, &buffer, memoriesLen)

	if buffer.Len() == 0 {
		return &Result{}, nil
	}

	// Phase 3: Structured summary.
	o.writeStatus("summarizing %d bytes of findings", buffer.Len())

	summary, err := o.summarizer.SummarizeFindings(ctx, buffer.String(), query)
	if err != nil {
		return nil, fmt.Errorf("summarizing recall: %w", err)
	}

	return &Result{Summary: summary}, nil
}

// searchMemories runs phase 1: find and format relevant memories.
// Returns the number of bytes added to the buffer.
func (o *Orchestrator) searchMemories(
	ctx context.Context,
	query string,
	buffer *strings.Builder,
) int {
	memories, err := o.listAndMatchMemories(ctx, query, DefaultMemoryLimit)
	if err != nil || len(memories) == 0 {
		o.writeStatus("found 0 relevant memories")
		return 0
	}

	text := formatMemories(memories)
	buffer.WriteString(text)

	o.writeStatus("found %d relevant memories", len(memories))

	return len(text)
}

// extractFromSessions runs phase 2: extract verbatim snippets per session.
func (o *Orchestrator) extractFromSessions(
	ctx context.Context,
	sessions []FileEntry,
	query string,
	buffer *strings.Builder,
	bytesUsed int,
) {
	for _, entry := range sessions {
		if ctx.Err() != nil {
			break
		}

		if bytesUsed >= DefaultExtractCap {
			break
		}

		content, _, readErr := o.reader.Read(entry.Path, DefaultStripBudget)
		if readErr != nil {
			continue
		}

		snippet, extractErr := o.summarizer.ExtractRelevant(ctx, content, query)
		if extractErr != nil || snippet == "" {
			continue
		}

		buffer.WriteString(snippet)

		bytesUsed += len(snippet)

		o.writeStatus("found %d bytes from %s", len(snippet), filepath.Base(entry.Path))
	}
}
```

Add the `"path/filepath"` import to the import block.

Delete the `accumulateSessionsAndMemories` function, `loadAllMemories` helper, `buildSingleTimeWindow` function, `DefaultModeBInputBudget` constant, and `findSessionMemories` method (only used by mode A — check first).

**Important:** `findSessionMemories` is still used by `recallModeA`. Keep it. `loadAllMemories` is used by the deleted `accumulateSessionsAndMemories` only — delete it. `buildSingleTimeWindow` is used only by `accumulateSessionsAndMemories` — delete it.

- [ ] **Step 4: Run the new tests**

Run: `go test ./internal/recall/ -run TestOrchestrator_Recall_ModeB -count=1`
Expected: PASS

- [ ] **Step 5: Run all recall tests**

Run: `go test ./internal/recall/ -count=1`
Expected: PASS

- [ ] **Step 6: Run targ check-full, fix issues**

Run: `targ check-full`
Fix: lint, dead code, declaration ordering, coverage as needed.

- [ ] **Step 7: Commit**

```bash
git add internal/recall/orchestrate.go internal/recall/orchestrate_test.go
git commit -m "feat(recall): three-phase pipeline for mode B queries

Memories first, then per-session verbatim extraction newest-first
until the 10KB buffer fills, then one structured summary call.
Replaces single-pass accumulate-then-summarize which sent irrelevant
content to the summarizer."
```

---

### Task 5: Wire `WithStatusWriter` in CLI

**Files:**
- Modify: `internal/cli/cli.go:236`

- [ ] **Step 1: Pass WithStatusWriter(os.Stderr) in runRecallSessions**

In `internal/cli/cli.go`, change:

```go
orch := recall.NewOrchestrator(finder, reader, summarizer, memLister, dataDir)
```

To:

```go
orch := recall.NewOrchestrator(finder, reader, summarizer, memLister, dataDir,
	recall.WithStatusWriter(os.Stderr))
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./cmd/engram/`
Expected: compiles cleanly

- [ ] **Step 3: Commit**

```bash
git add internal/cli/cli.go
git commit -m "feat(cli): wire status writer for recall progress output"
```

---

### Task 6: E2E Validation

- [ ] **Step 1: Build and install**

```bash
go build -o ~/go/bin/engram ./cmd/engram
```

- [ ] **Step 2: Run a query and observe output**

```bash
time engram recall --query "targ argument parsing command CLI"
```

Verify:
- stderr shows progress (memory count, per-session snippet sizes)
- stdout contains a structured summary
- Total time is reasonable (~10-20s depending on session count before buffer fills)
- Summary is actually about targ argument parsing (not about signal handling or whatever the most recent session was about)

- [ ] **Step 3: Run targ check-full**

Run: `targ check-full`
Expected: only pre-existing failures (cli coverage)

- [ ] **Step 4: Test with no-query mode (mode A regression check)**

```bash
engram recall | head -20
```

Verify: mode A still returns raw transcript content (no regression).
