# Keyword Refinement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Accumulate bad-match query contexts on irrelevant feedback and use them for LLM-backed keyword refinement proposals during maintain triage.

**Architecture:** Three touch points — data model (new field on MemoryRecord + Stored), feedback path (persist surfacing query), maintain path (LLM-backed proposals), apply path (new refine_keywords handler). Each is independently testable.

**Tech Stack:** Go, TOML serialization (BurntSushi/toml), gomega test assertions, Haiku LLM via injected function.

**Spec:** `docs/superpowers/specs/2026-03-24-keyword-refinement-design.md`

---

### Task 1: Add IrrelevantQueries field to data model

**Files:**
- Modify: `internal/memory/record.go:48-54` (add field to MemoryRecord)
- Modify: `internal/memory/memory.go:84-99` (add field to Stored)
- Modify: `internal/retrieve/retrieve.go:84-98` (add field to MemoryRecord→Stored mapping)
- Modify: `internal/cli/signal.go:262-275` (add field to Stored→MemoryRecord writer mapping)
- Modify: `internal/cli/signal.go:430-444` (add field to MemoryRecord→Stored reader mapping)

- [ ] **Step 1: Add field to MemoryRecord**

In `internal/memory/record.go`, add after the `IrrelevantCount` line (line 53):

```go
IrrelevantQueries []string `toml:"irrelevant_queries,omitempty"`
```

- [ ] **Step 2: Add field to Stored**

In `internal/memory/memory.go`, add after the `IrrelevantCount` line (line 96):

```go
IrrelevantQueries []string
```

- [ ] **Step 3: Add field to retrieve mapping**

In `internal/retrieve/retrieve.go`, add after the `IrrelevantCount` line (line 95) in the `&memory.Stored{` literal:

```go
IrrelevantQueries: record.IrrelevantQueries,
```

- [ ] **Step 4: Add field to signal writer mapping**

In `internal/cli/signal.go`, add after the `IrrelevantCount` line (line 273) in the `memory.MemoryRecord{` literal:

```go
IrrelevantQueries: stored.IrrelevantQueries,
```

- [ ] **Step 5: Add field to signal reader mapping**

In `internal/cli/signal.go`, add after the `IrrelevantCount` line (line 441) in the `&memory.Stored{` literal inside `readStoredMemory`:

```go
IrrelevantQueries: record.IrrelevantQueries,
```

- [ ] **Step 6: Run tests**

Run: `targ test`
Expected: All existing tests pass (field is additive, no behavior change).

- [ ] **Step 7: Commit**

```
feat(memory): add IrrelevantQueries field to data model (#372)

Adds irrelevant_queries []string to MemoryRecord and Stored, plus
all three mappings (retrieve, signal-writer, signal-reader). No
behavior change yet — the field is populated in the next commit.
```

---

### Task 2: Persist surfacing query on irrelevant feedback

**Files:**
- Modify: `internal/cli/feedback.go:115-128` (append query after counter update)
- Test: `internal/cli/feedback_test.go` (new test + update existing)

- [ ] **Step 1: Write the failing test**

In `internal/cli/feedback_test.go`, add a new test:

```go
func TestFeedback_Irrelevant_WithSurfacingQuery_PersistsQuery(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tomlContent := "title = \"persist-query\"\nsurfaced_count = 1\n"
	err = os.WriteFile(
		filepath.Join(memDir, "persist-query.toml"),
		[]byte(tomlContent),
		0o640,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var stdout, stderr bytes.Buffer

	err = cli.Run(
		[]string{
			"engram", "feedback",
			"--name", "persist-query",
			"--data-dir", dataDir,
			"--surfacing-query", "how to test",
			"--irrelevant",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Re-read the TOML and verify query was persisted.
	data, readErr := os.ReadFile(filepath.Join(memDir, "persist-query.toml"))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	var record memory.MemoryRecord

	_, decErr := toml.Decode(string(data), &record)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	g.Expect(record.IrrelevantQueries).To(Equal([]string{"how to test"}))
}
```

- [ ] **Step 2: Write test for cap at 20 entries**

Note: This test uses `fmt.Sprintf` — add `"fmt"` to the test file's import block if not already present.

```go
func TestFeedback_IrrelevantQueries_CappedAt20(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Pre-populate with 20 existing queries.
	var existingQueries string
	for i := range 20 {
		existingQueries += fmt.Sprintf("  \"query-%d\",\n", i)
	}

	tomlContent := fmt.Sprintf(
		"title = \"cap-test\"\nsurfaced_count = 1\nirrelevant_queries = [\n%s]\n",
		existingQueries,
	)
	err = os.WriteFile(
		filepath.Join(memDir, "cap-test.toml"),
		[]byte(tomlContent),
		0o640,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var stdout, stderr bytes.Buffer

	err = cli.Run(
		[]string{
			"engram", "feedback",
			"--name", "cap-test",
			"--data-dir", dataDir,
			"--surfacing-query", "new query",
			"--irrelevant",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr := os.ReadFile(filepath.Join(memDir, "cap-test.toml"))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	var record memory.MemoryRecord

	_, decErr := toml.Decode(string(data), &record)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	const maxIrrelevantQueries = 20

	g.Expect(record.IrrelevantQueries).To(HaveLen(maxIrrelevantQueries))
	// Oldest ("query-0") dropped, newest ("new query") appended.
	g.Expect(record.IrrelevantQueries[0]).To(Equal("query-1"))
	g.Expect(record.IrrelevantQueries[maxIrrelevantQueries-1]).To(Equal("new query"))
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `targ test`
Expected: Both new tests fail — IrrelevantQueries is empty because feedback.go doesn't persist it yet.

- [ ] **Step 4: Implement the persistence logic**

In `internal/cli/feedback.go`, in `runFeedback`, between the `applyFeedbackCounters` call (line 115) and the `writeFeedbackTOML` call (line 117), add:

```go
const maxIrrelevantQueries = 20

if *irrelevant && *surfacingQuery != "" {
	record.IrrelevantQueries = append(record.IrrelevantQueries, *surfacingQuery)
	if len(record.IrrelevantQueries) > maxIrrelevantQueries {
		record.IrrelevantQueries = record.IrrelevantQueries[len(record.IrrelevantQueries)-maxIrrelevantQueries:]
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `targ test`
Expected: All tests pass including the two new ones.

- [ ] **Step 6: Commit**

```
feat(feedback): persist surfacing query to IrrelevantQueries (#372)

When --irrelevant and --surfacing-query are both provided, the query
is appended to the memory's irrelevant_queries field (capped at 20).
This accumulates evidence for keyword refinement during maintain.
```

---

### Task 3: LLM-backed keyword suggestions in maintain checkIrrelevance

**Files:**
- Modify: `internal/maintain/maintain.go:76-106` (add LLM call when queries present)
- Test: `internal/maintain/maintain_test.go` (new tests)

- [ ] **Step 1: Write failing test — LLM called when queries present**

In `internal/maintain/maintain_test.go`, add:

```go
func TestGenerate_HighIrrelevance_WithQueries_CallsLLM(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var capturedUserPrompt string

	llmCaller := func(
		_ context.Context, _, _, userPrompt string,
	) (string, error) {
		capturedUserPrompt = userPrompt

		return `{"remove_keywords":["code"],"add_keywords":["go-testing"],"rationale":"too generic"}`, nil
	}

	gen := maintain.New(
		maintain.WithNow(fixedNow),
		maintain.WithLLMCaller(llmCaller),
	)

	classified := []review.ClassifiedMemory{
		{Name: "generic-mem", Quadrant: review.Working},
	}
	memories := map[string]*memory.Stored{
		"generic-mem": {
			Title:             "Generic Memory",
			Keywords:          []string{"code"},
			Principle:         "Some principle",
			FollowedCount:     3,
			IrrelevantCount:   8,
			IrrelevantQueries: []string{"how to test", "dependency injection"},
			UpdatedAt:         fixedNow(),
		},
	}

	proposals := gen.Generate(context.Background(), classified, memories)

	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].Action).To(gomega.Equal("refine_keywords"))
	g.Expect(proposals[0].Details).NotTo(gomega.BeEmpty())
	g.Expect(string(proposals[0].Details)).To(gomega.ContainSubstring("remove_keywords"))
	g.Expect(capturedUserPrompt).To(gomega.ContainSubstring("how to test"))
	g.Expect(capturedUserPrompt).To(gomega.ContainSubstring("dependency injection"))
}
```

- [ ] **Step 2: Write failing test — no LLM call when queries empty**

```go
func TestGenerate_HighIrrelevance_NoQueries_NoLLMCall(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	llmCalled := false

	llmCaller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		llmCalled = true

		return "{}", nil
	}

	gen := maintain.New(
		maintain.WithNow(fixedNow),
		maintain.WithLLMCaller(llmCaller),
	)

	classified := []review.ClassifiedMemory{
		{Name: "no-queries-mem", Quadrant: review.Working},
	}
	memories := map[string]*memory.Stored{
		"no-queries-mem": {
			Title:           "No Queries Memory",
			Keywords:        []string{"generic"},
			FollowedCount:   2,
			IrrelevantCount: 8,
			UpdatedAt:       fixedNow(),
		},
	}

	proposals := gen.Generate(context.Background(), classified, memories)

	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].Action).To(gomega.Equal("refine_keywords"))
	g.Expect(proposals[0].Details).To(gomega.BeEmpty())
	g.Expect(llmCalled).To(gomega.BeFalse())
}
```

- [ ] **Step 3: Write failing test — LLM failure falls back gracefully**

```go
func TestGenerate_HighIrrelevance_LLMFailure_ProposesWithoutDetails(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	llmCaller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		return "", errors.New("LLM unavailable")
	}

	gen := maintain.New(
		maintain.WithNow(fixedNow),
		maintain.WithLLMCaller(llmCaller),
	)

	classified := []review.ClassifiedMemory{
		{Name: "llm-fail-mem", Quadrant: review.Working},
	}
	memories := map[string]*memory.Stored{
		"llm-fail-mem": {
			Title:             "LLM Fail Memory",
			Keywords:          []string{"broad"},
			FollowedCount:     2,
			IrrelevantCount:   8,
			IrrelevantQueries: []string{"some query"},
			UpdatedAt:         fixedNow(),
		},
	}

	proposals := gen.Generate(context.Background(), classified, memories)

	// Should still produce a proposal, just without LLM details.
	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].Action).To(gomega.Equal("refine_keywords"))
	g.Expect(proposals[0].Diagnosis).To(gomega.ContainSubstring("irrelevant"))
}
```

- [ ] **Step 4: Run tests to verify they fail**

Run: `targ test`
Expected: First test fails (Details empty, no LLM call made). Second and third may pass already (current behavior has no LLM call and no Details).

- [ ] **Step 5: Implement LLM-backed checkIrrelevance**

In `internal/maintain/maintain.go`, update `checkIrrelevance` to accept `ctx` and call the LLM when queries are present. The signature changes from:

```go
func (g *Generator) checkIrrelevance(
	classifiedMem review.ClassifiedMemory,
	stored *memory.Stored,
) (Proposal, bool) {
```

to:

```go
func (g *Generator) checkIrrelevance(
	ctx context.Context,
	classifiedMem review.ClassifiedMemory,
	stored *memory.Stored,
) (Proposal, bool) {
```

Update the call site in `generateOne` (line 116) to pass `ctx`.

After building the base proposal (line 97-105), add LLM enrichment:

```go
// Enrich with LLM-suggested keyword changes if query evidence exists.
if g.llmCaller != nil && len(stored.IrrelevantQueries) > 0 {
	systemPrompt := refineKeywordsSystemPrompt
	userPrompt := buildRefineDescription(classifiedMem, stored)

	response, llmErr := g.llmCaller(ctx, maintainModel, systemPrompt, userPrompt)
	if llmErr == nil {
		proposal.Details = safeLLMDetails(response)
	}
}
```

Add the constant and helper:

```go
const refineKeywordsSystemPrompt = "You are a memory maintenance assistant. " +
	"A memory keeps surfacing in irrelevant contexts. " +
	"Given the memory's current keywords and the queries that caused false matches, " +
	"identify which keywords are too generic and suggest specific replacements. " +
	"Output: " +
	`{"remove_keywords":[...],"add_keywords":[...],"rationale":"..."}`
```

```go
func buildRefineDescription(
	classifiedMem review.ClassifiedMemory,
	stored *memory.Stored,
) string {
	base := buildMemoryDescription(classifiedMem, stored)

	return fmt.Sprintf(
		"%s\nIrrelevant queries that caused false matches:\n- %s",
		base,
		strings.Join(stored.IrrelevantQueries, "\n- "),
	)
}
```

Note: Add `"strings"` to the import block if not already present.

- [ ] **Step 6: Run tests to verify they pass**

Run: `targ test`
Expected: All tests pass.

- [ ] **Step 7: Run full quality check**

Run: `targ check-full`
Expected: All lints and tests pass.

- [ ] **Step 8: Commit**

```
feat(maintain): LLM-backed keyword suggestions for refine_keywords (#372)

When a memory triggers refine_keywords and has accumulated
irrelevant_queries, calls Haiku to identify generic keywords and
suggest replacements. The LLM response is stored in the proposal's
Details field for human review during triage. Falls back gracefully
if LLM is unavailable or queries are empty.
```

---

### Task 4: Add refine_keywords handler to apply

**Files:**
- Modify: `internal/signal/apply.go:50-61` (add case to switch)
- Test: `internal/signal/apply_test.go` (new tests)

- [ ] **Step 1: Write failing test — successful refine**

In `internal/signal/apply_test.go`, add:

```go
func TestApply_RefineKeywords(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	writer := &stubMemoryWriter{written: make(map[string]*memory.Stored)}

	applier := signal.NewApplier(
		signal.WithReadMemory(func(_ string) (*memory.Stored, error) {
			return &memory.Stored{
				Title:             "Noisy",
				Keywords:          []string{"code", "testing", "specific-good"},
				IrrelevantQueries: []string{"how to test", "dependency injection"},
			}, nil
		}),
		signal.WithWriteMemory(writer),
	)

	action := signal.ApplyAction{
		Action: "refine_keywords",
		Memory: "memories/noisy.toml",
		Fields: map[string]any{
			"remove_keywords": []any{"code", "testing"},
			"add_keywords":    []any{"go-test-isolation", "parallel-test-state"},
		},
	}

	result, err := applier.Apply(context.Background(), action)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Success).To(gomega.BeTrue())

	stored := writer.written["memories/noisy.toml"]
	g.Expect(stored).NotTo(gomega.BeNil())

	if stored == nil {
		return
	}

	// "code" and "testing" removed, two new ones added, "specific-good" kept.
	g.Expect(stored.Keywords).To(gomega.ConsistOf(
		"specific-good", "go-test-isolation", "parallel-test-state",
	))
	// IrrelevantQueries cleared after refinement.
	g.Expect(stored.IrrelevantQueries).To(gomega.BeEmpty())
}
```

- [ ] **Step 2: Write failing test — nil memory**

```go
func TestApply_RefineKeywordsNilMemory(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	applier := signal.NewApplier(
		signal.WithReadMemory(func(_ string) (*memory.Stored, error) {
			return nil, nil //nolint:nilnil // testing nil memory path
		}),
	)

	action := signal.ApplyAction{
		Action: "refine_keywords",
		Memory: "memories/gone.toml",
		Fields: map[string]any{
			"remove_keywords": []any{"old"},
			"add_keywords":    []any{"new"},
		},
	}

	_, err := applier.Apply(context.Background(), action)
	g.Expect(err).To(gomega.HaveOccurred())
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `targ test`
Expected: Both fail with `unsupported action: refine_keywords`.

- [ ] **Step 4: Implement applyRefine**

In `internal/signal/apply.go`, add the case to the switch (after `actionBroadenKeywords`):

```go
case actionRefineKeywords:
	err = a.applyRefine(action)
```

Add the constant (it may already exist — check `maintain` package imports):

```go
const actionRefineKeywords = "refine_keywords"
```

Add the handler:

```go
func (a *Applier) applyRefine(action ApplyAction) error {
	stored, err := a.readMemory(action.Memory)
	if err != nil {
		return fmt.Errorf("reading memory for refine: %w", err)
	}

	if stored == nil {
		return fmt.Errorf("reading memory for refine: %w", os.ErrNotExist)
	}

	removeSet := toStringSet(action.Fields["remove_keywords"])
	addKeywords := toStringSlice(action.Fields["add_keywords"])

	// Remove specified keywords.
	filtered := make([]string, 0, len(stored.Keywords))
	for _, kw := range stored.Keywords {
		if !removeSet[kw] {
			filtered = append(filtered, kw)
		}
	}

	stored.Keywords = append(filtered, keyword.NormalizeAll(addKeywords)...)
	stored.IrrelevantQueries = nil

	writeErr := a.writeMem.Write(action.Memory, stored)
	if writeErr != nil {
		return fmt.Errorf("writing refined memory: %w", writeErr)
	}

	return nil
}

func toStringSet(val any) map[string]bool {
	set := make(map[string]bool)

	items, ok := val.([]any)
	if !ok {
		return set
	}

	for _, item := range items {
		if s, ok := item.(string); ok {
			set[s] = true
		}
	}

	return set
}

func toStringSlice(val any) []string {
	items, ok := val.([]any)
	if !ok {
		return nil
	}

	result := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}

	return result
}
```

Note: Check if `keyword` package is already imported — it's used by `applyBroaden`. Also check if `os` is imported.

- [ ] **Step 5: Run tests to verify they pass**

Run: `targ test`
Expected: All tests pass.

- [ ] **Step 6: Run full quality check**

Run: `targ check-full`
Expected: All lints and tests pass.

- [ ] **Step 7: Commit**

```
feat(apply): add refine_keywords handler (#372)

Removes specified keywords, adds normalized replacements, and clears
irrelevant_queries after refinement. Follows the same pattern as
applyBroaden and applyRewrite.
```

---

### Task 5: Integration verification

**Files:**
- No new files — manual verification of the full pipeline.

- [ ] **Step 1: Run full quality check**

Run: `targ check-full`
Expected: All lints, tests, and coverage pass.

- [ ] **Step 2: Verify TOML round-trip**

Create a temporary memory TOML with `irrelevant_queries` populated and verify it round-trips through read/write without data loss. This can be done by running the existing test suite — if round-trip is broken, existing tests for the retrieve and signal-writer paths will fail.

- [ ] **Step 3: Commit (if any fixes needed)**

Only if previous steps revealed issues.
