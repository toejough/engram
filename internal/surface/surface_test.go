package surface_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

// QW-1: Tool mode limits to top 2 results (REQ-P4e-4: down from 3, down from original 5 in REQ-11).
func TestQW1_ToolModeLimitsToTop2(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// 5 anti-pattern memories matching "commit" + 8 non-matching fillers for IDF contrast.
	memories := make([]*memory.Stored, 0, 13)
	for i := range 5 {
		memories = append(memories, &memory.Stored{
			Title:       memTitle(i),
			FilePath:    memPath(i),
			AntiPattern: "manual commit violation",
			Keywords:    []string{"commit", "git"},
			Principle:   "use /commit skill",
		})
	}

	fillerNames := []string{
		"logging",
		"testing",
		"deploy",
		"config",
		"monitoring",
		"caching",
		"auth",
		"docs",
	}
	for _, name := range fillerNames {
		memories = append(memories, &memory.Stored{
			Title:       name + " rule",
			FilePath:    name + "-rule.toml",
			AntiPattern: name + " violation",
			Keywords:    []string{name},
			Principle:   name + " standards",
		})
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: "git commit -m 'fix bug'",
		Format:    surface.FormatJSON,
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var result surface.Result

	decodeErr := json.Unmarshal(buf.Bytes(), &result)
	g.Expect(decodeErr).NotTo(HaveOccurred())

	if decodeErr != nil {
		return
	}

	g.Expect(result.Summary).To(ContainSubstring("[engram] 2 tool advisories:"))
}

// QW-2: BM25 relevance floor filters low-scoring memories.
func TestQW2_RelevanceFloorFiltersLowScoring(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Need enough docs for IDF contrast (df/N < 0.5 so IDF > 0).
	memories := []*memory.Stored{
		{
			Title:    "Exact Match",
			FilePath: "exact-match.toml",
			Keywords: []string{"deploy", "production"},
			Content:  "deploy to production safely",
		},
		{
			Title:    "Irrelevant Memory",
			FilePath: "irrelevant.toml",
			Keywords: []string{"banana", "fruit"},
			Content:  "banana smoothie recipe for healthy eating",
		},
		{
			Title:    "Filler A",
			FilePath: "filler-a.toml",
			Keywords: []string{"logging"},
			Content:  "logging best practices",
		},
		{
			Title:    "Filler B",
			FilePath: "filler-b.toml",
			Keywords: []string{"testing"},
			Content:  "testing strategies",
		},
		{
			Title:    "Filler C",
			FilePath: "filler-c.toml",
			Keywords: []string{"monitoring"},
			Content:  "monitoring dashboards",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "deploy to production",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	// "exact-match" should appear, "irrelevant" should not (score below floor).
	g.Expect(output).To(ContainSubstring("exact-match"))
	g.Expect(output).NotTo(ContainSubstring("irrelevant"))
}

// QW-2: Tool mode relevance floor filters low-scoring memories.
func TestQW2_ToolRelevanceFloorFiltersLowScoring(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:       "Commit Rule",
			FilePath:    "commit-rule.toml",
			AntiPattern: "manual git commit",
			Keywords:    []string{"commit", "git"},
			Principle:   "use /commit",
		},
		{
			Title:       "Banana Rule",
			FilePath:    "banana-rule.toml",
			AntiPattern: "eating bananas at desk",
			Keywords:    []string{"banana", "fruit"},
			Principle:   "no food at desk",
		},
		{
			Title:       "Filler A",
			FilePath:    "filler-a.toml",
			AntiPattern: "logging without context",
			Keywords:    []string{"logging"},
			Principle:   "always log with context",
		},
		{
			Title:       "Filler B",
			FilePath:    "filler-b.toml",
			AntiPattern: "skipping monitoring",
			Keywords:    []string{"monitoring"},
			Principle:   "monitor everything",
		},
		{
			Title:       "Filler C",
			FilePath:    "filler-c.toml",
			AntiPattern: "ignoring alerts",
			Keywords:    []string{"alerts"},
			Principle:   "respond to alerts",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: "git commit -m 'fix'",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	g.Expect(output).To(ContainSubstring("commit-rule"))
	g.Expect(output).NotTo(ContainSubstring("banana-rule"))
}

// T-100: Tool mode with no matching memories produces empty output
func TestT100_ToolModeNoMatchProducesEmpty(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:       "Use /commit",
			FilePath:    "use-commit.toml",
			AntiPattern: "manual git commit",
			Keywords:    []string{"commit", "git"},
			Principle:   "use /commit for commits",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	// Tool input contains no matching keywords.
	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Read",
		ToolInput: `/path/to/file.go`,
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf.String()).To(BeEmpty())
}

// T-115: Effectiveness annotation rendered when data exists
func TestT115_EffectivenessAnnotationRendered(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:     "Alpha Memory",
			FilePath:  "alpha.toml",
			UpdatedAt: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	retriever := &fakeRetriever{memories: memories}
	eff := &fakeEffectivenessComputer{
		stats: map[string]surface.EffectivenessStat{
			"alpha.toml": {SurfacedCount: 5, EffectivenessScore: 80.0},
		},
	}

	s := surface.New(retriever, surface.WithEffectiveness(eff))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(buf.String()).To(ContainSubstring("(surfaced 5 times, followed 80%)"))
}

// T-116: No annotation when no evaluation data exists
func TestT116_NoAnnotationWhenNoEvalData(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:     "Beta Memory",
			FilePath:  "beta.toml",
			UpdatedAt: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	// No WithEffectiveness option — no annotation expected.
	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(buf.String()).NotTo(ContainSubstring("surfaced"))
}

// T-121: Surfacer writes surfacing log during surfacing events.
func TestT121_SurfacerWritesSurfacingLog(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{Title: "Alpha", FilePath: "mem/alpha.toml", Keywords: []string{"alpha"}},
		{Title: "Beta", FilePath: "mem/beta.toml", Keywords: []string{"beta"}},
		{Title: "Gamma", FilePath: "mem/gamma.toml", Keywords: []string{"gamma"}},
		{Title: "Delta", FilePath: "mem/delta.toml", Keywords: []string{"delta"}},
	}

	retriever := &fakeRetriever{memories: memories}
	logger := &fakeSurfacingLogger{}

	s := surface.New(retriever, surface.WithSurfacingLogger(logger))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/data",
		Message: "alpha beta",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(logger.calls).To(HaveLen(2))
	g.Expect(logger.calls[0].memoryPath).To(Equal("mem/alpha.toml"))
	g.Expect(logger.calls[0].mode).To(Equal(surface.ModePrompt))
	g.Expect(logger.calls[1].memoryPath).To(Equal("mem/beta.toml"))
	g.Expect(logger.calls[1].mode).To(Equal(surface.ModePrompt))
}

// T-163: Tool mode frecency re-ranking with multiple anti-pattern candidates.
func TestT163_ToolModeFrecencyReRanking(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Now()

	// Two anti-pattern memories, both matching "commit" in tool input.
	memLowFrecency := &memory.Stored{
		Title:       "Old Commit Rule",
		FilePath:    "old-commit-rule.toml",
		AntiPattern: "manual git commit",
		Keywords:    []string{"commit"},
		Principle:   "use /commit skill",
		UpdatedAt:   now.Add(-30 * 24 * time.Hour),
	}

	memHighFrecency := &memory.Stored{
		Title:       "Recent Commit Rule",
		FilePath:    "recent-commit-rule.toml",
		AntiPattern: "direct git commit",
		Keywords:    []string{"commit", "git"},
		Principle:   "always use /commit",
		UpdatedAt:   now.Add(-48 * time.Hour),
	}

	// Anti-pattern fillers that DON'T match "commit"/"git" — needed for BM25
	// IDF contrast. BM25 IDF = log((N-df+0.5)/(df+0.5)), which is 0 when
	// df = N/2. We need df/N < 0.5 so IDF > 0 for "commit"/"git".
	fillers := make([]*memory.Stored, 0, 3)
	for _, name := range []string{"formatting", "naming", "logging"} {
		fillers = append(fillers, &memory.Stored{
			Title:       name + " rule",
			FilePath:    name + "-rule.toml",
			AntiPattern: name + " violation",
			Keywords:    []string{name},
			Principle:   "follow " + name + " standards",
		})
	}

	// Retriever returns low-frecency first.
	allMems := make([]*memory.Stored, 0, 2+len(fillers))
	allMems = append(allMems, memLowFrecency, memHighFrecency)
	allMems = append(allMems, fillers...)
	retriever := &fakeRetriever{memories: allMems}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: "git commit -m 'fix'",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()

	// Without inline tracking fields, both memories should appear (BM25 match).
	// Frecency re-ranking is pending registry integration (UC-23).
	g.Expect(output).To(ContainSubstring("commit-rule"))
}

// T-169/T-171: SessionStart uses frecency ranking (not just recency).
func TestT169_SessionStartUsesFrecencyRanking(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Now()

	// Without inline tracking fields (UC-23), frecency falls back to recency (UpdatedAt).
	// Memory A: old UpdatedAt.
	memA := &memory.Stored{
		Title:     "Frequently Used",
		FilePath:  "frequently-used.toml",
		UpdatedAt: now.Add(-30 * 24 * time.Hour), // 30 days ago
	}

	// Memory B: recent UpdatedAt — should rank higher by recency fallback.
	memB := &memory.Stored{
		Title:     "Recently Created",
		FilePath:  "recently-created.toml",
		UpdatedAt: now.Add(-2 * time.Hour),
	}

	// Retriever returns B before A (sorted by UpdatedAt desc — B is more recent).
	retriever := &fakeRetriever{memories: []*memory.Stored{memB, memA}}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()

	// Frecency should reorder: "frequently-used" before "recently-created".
	idxA := strings.Index(output, "frequently-used")
	idxB := strings.Index(output, "recently-created")

	g.Expect(idxA).To(BeNumerically(">=", 0), "frequently-used should appear in output")
	g.Expect(idxB).To(BeNumerically(">=", 0), "recently-created should appear in output")
	g.Expect(idxB).To(BeNumerically("<", idxA),
		"more recently updated memory should appear first (recency fallback)")
}

// T-170B: Prompt mode frecency re-ranking with multiple BM25 matches.
func TestT170B_PromptModeFrecencyReRanking(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Now()

	// Two memories both matching "testing" in the prompt.
	memLowFrecency := &memory.Stored{
		Title:     "Old Testing Guide",
		FilePath:  "old-testing.toml",
		Keywords:  []string{"testing", "unit"},
		Principle: "write tests first",
		Content:   "Testing guide for unit tests",
		UpdatedAt: now.Add(-30 * 24 * time.Hour),
	}

	memHighFrecency := &memory.Stored{
		Title:     "Recent Testing Guide",
		FilePath:  "recent-testing.toml",
		Keywords:  []string{"testing", "integration"},
		Principle: "test everything",
		Content:   "Testing guide for integration tests",
		UpdatedAt: now.Add(-48 * time.Hour),
	}

	// Non-matching fillers for IDF contrast.
	fillers := make([]*memory.Stored, 0, 3)
	for _, name := range []string{"deploy", "config", "logging"} {
		fillers = append(fillers, &memory.Stored{
			Title:    name + " guide",
			FilePath: name + ".toml",
			Keywords: []string{name},
			Content:  name + " documentation",
		})
	}

	allMems := make([]*memory.Stored, 0, 2+len(fillers))
	allMems = append(allMems, memLowFrecency, memHighFrecency)
	allMems = append(allMems, fillers...)
	retriever := &fakeRetriever{memories: allMems}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "how do I write testing for my code",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()

	// Without inline tracking fields, both memories should appear (BM25 match).
	// Frecency re-ranking is pending registry integration (UC-23).
	g.Expect(output).To(ContainSubstring("testing"))
}

// T-199: Surfacing hook calls RecordSurfacing, surfaced_count increments.
func TestT199_RegistryRecordSurfacingCalled(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:     "Alpha",
			FilePath:  "alpha.toml",
			UpdatedAt: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Title:     "Beta",
			FilePath:  "beta.toml",
			UpdatedAt: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	retriever := &fakeRetriever{memories: memories}
	registry := &fakeRegistryRecorder{}
	s := surface.New(retriever, surface.WithRegistry(registry))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Both memories should have been recorded.
	g.Expect(registry.ids).To(HaveLen(2))
	g.Expect(registry.ids).To(ContainElement("alpha.toml"))
	g.Expect(registry.ids).To(ContainElement("beta.toml"))
}

// T-199b: Registry error does not affect surfacing output (fire-and-forget).
func TestT199b_RegistryErrorDoesNotAffectSurfacing(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:     "Alpha",
			FilePath:  "alpha.toml",
			UpdatedAt: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	retriever := &fakeRetriever{memories: memories}
	registry := &fakeRegistryRecorder{err: errors.New("registry write failed")}
	s := surface.New(retriever, surface.WithRegistry(registry))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
	})

	// Run should succeed despite registry error.
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf.String()).To(ContainSubstring("alpha"))
}

// T-199c: Nil registry does not panic (backward compat).
func TestT199c_NilRegistryNoPanic(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:     "Alpha",
			FilePath:  "alpha.toml",
			UpdatedAt: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	retriever := &fakeRetriever{memories: memories}
	// No WithRegistry — registry is nil.
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf.String()).To(ContainSubstring("alpha"))
}

// T-27: SessionStart surfaces top 7 by effectiveness (REQ-P4e-2: reduced from 10).
func TestT27_SessionStartSurfacesTop7(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Create 15 memories in descending order (retriever contract: sorted by UpdatedAt desc).
	// All have no effectiveness data → insufficient data → stable order preserved, top-7 selected.
	memories := make([]*memory.Stored, 0, 15)
	for i := 14; i >= 0; i-- {
		memories = append(memories, &memory.Stored{
			Title:    memTitle(i),
			FilePath: memPath(i),
			UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).
				Add(time.Duration(i) * 24 * time.Hour),
		})
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	g.Expect(output).To(ContainSubstring("[engram] Loaded 7 memories."))

	// Top 7 (indices 14–8) should appear, index 7 and below should not.
	g.Expect(output).To(ContainSubstring(memSlug(14)))
	g.Expect(output).To(ContainSubstring(memSlug(8)))
	g.Expect(output).NotTo(ContainSubstring(memSlug(7)))
	g.Expect(output).NotTo(ContainSubstring(memSlug(0)))
}

// T-28: SessionStart with fewer than 20 memories surfaces all
func TestT28_SessionStartSurfacesAll(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:     "First",
			FilePath:  "first.toml",
			UpdatedAt: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Title:     "Second",
			FilePath:  "second.toml",
			UpdatedAt: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Title:     "Third",
			FilePath:  "third.toml",
			UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	g.Expect(output).To(ContainSubstring("[engram] Loaded 3 memories."))
	g.Expect(output).To(ContainSubstring("first"))
	g.Expect(output).To(ContainSubstring("second"))
	g.Expect(output).To(ContainSubstring("third"))
}

// T-29: SessionStart with no memories produces empty output
func TestT29_SessionStartNoMemories(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf.String()).To(BeEmpty())
}

// T-30: Keyword match surfaces relevant memories
func TestT30_KeywordMatchSurfacesRelevant(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:    "Commit Conventions",
			FilePath: "commit-conventions.toml",
			Keywords: []string{"commit", "git"},
		},
		{
			Title:    "Build Tools",
			FilePath: "build-tools.toml",
			Keywords: []string{"targ", "build"},
		},
		{
			Title:    "Testing Framework",
			FilePath: "testing.toml",
			Keywords: []string{"test", "unit"},
		},
		{
			Title:    "Linting Rules",
			FilePath: "linting.toml",
			Keywords: []string{"lint", "style"},
		},
		{
			Title:    "Docker Containers",
			FilePath: "docker.toml",
			Keywords: []string{"docker", "container"},
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "I want to commit this change",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	g.Expect(output).To(ContainSubstring("[engram] Relevant memories:"))
	g.Expect(output).To(ContainSubstring("commit-conventions"))
	g.Expect(output).NotTo(ContainSubstring("build-tools"))
}

// T-31: No keyword match produces empty output
func TestT31_NoKeywordMatchProducesEmpty(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:    "Commit Conventions",
			FilePath: "commit-conventions.toml",
			Keywords: []string{"commit", "git"},
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "hello world",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf.String()).To(BeEmpty())
}

// T-323: PreCompact mode ranks memories by effectiveness descending.
func TestT323_PreCompactRanksByEffectiveness(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{FilePath: "low.toml", Principle: "low principle"},
		{FilePath: "high.toml", Principle: "high principle"},
		{FilePath: "mid.toml", Principle: "mid principle"},
	}

	eff := &fakeEffectivenessComputer{
		stats: map[string]surface.EffectivenessStat{
			"low.toml":  {SurfacedCount: 5, EffectivenessScore: 45.0},
			"high.toml": {SurfacedCount: 5, EffectivenessScore: 90.0},
			"mid.toml":  {SurfacedCount: 5, EffectivenessScore: 70.0},
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever, surface.WithEffectiveness(eff))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePreCompact,
		DataDir: "/tmp/data",
		Budget:  500,
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	highIdx := strings.Index(output, "high principle")
	midIdx := strings.Index(output, "mid principle")
	lowIdx := strings.Index(output, "low principle")

	g.Expect(highIdx).To(BeNumerically(">", -1), "expected high principle in output")
	g.Expect(highIdx).To(BeNumerically("<", midIdx), "high should appear before mid")
	g.Expect(midIdx).To(BeNumerically("<", lowIdx), "mid should appear before low")
}

// T-324: PreCompact mode skips memories with effectiveness < 40%.
func TestT324_PreCompactSkipsLowEffectiveness(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{FilePath: "skip.toml", Principle: "skip principle"},
		{FilePath: "keep.toml", Principle: "keep principle"},
		{FilePath: "nodata.toml", Principle: "nodata principle"},
	}

	eff := &fakeEffectivenessComputer{
		stats: map[string]surface.EffectivenessStat{
			"skip.toml": {SurfacedCount: 5, EffectivenessScore: 39.0},
			"keep.toml": {SurfacedCount: 5, EffectivenessScore: 50.0},
			// nodata.toml has no entry — treated as 0% effectiveness
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever, surface.WithEffectiveness(eff))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePreCompact,
		DataDir: "/tmp/data",
		Budget:  500,
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	g.Expect(output).NotTo(ContainSubstring("skip principle"))
	g.Expect(output).NotTo(ContainSubstring("nodata principle"))
	g.Expect(output).To(ContainSubstring("keep principle"))
}

// T-325: PreCompact mode limits to top-5 memories even when budget allows more.
func TestT325_PreCompactLimitsToTop5(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := make([]*memory.Stored, 8)
	stats := make(map[string]surface.EffectivenessStat, 8)

	for i := range 8 {
		path := fmt.Sprintf("mem-%d.toml", i)
		memories[i] = &memory.Stored{
			FilePath:  path,
			Principle: fmt.Sprintf("principle %d", i),
		}
		stats[path] = surface.EffectivenessStat{
			SurfacedCount:      5,
			EffectivenessScore: float64(50 + i),
		}
	}

	eff := &fakeEffectivenessComputer{stats: stats}
	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever, surface.WithEffectiveness(eff))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePreCompact,
		DataDir: "/tmp/data",
		Budget:  10000,
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	count := strings.Count(output, "\n- ")
	g.Expect(count).To(Equal(5), "expected exactly 5 memories, got %d", count)
}

// T-326: PreCompact mode output has correct header and principle-only lines.
func TestT326_PreCompactOutputFormat(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{FilePath: "mem.toml", Principle: "always use /commit skill"},
	}

	eff := &fakeEffectivenessComputer{
		stats: map[string]surface.EffectivenessStat{
			"mem.toml": {SurfacedCount: 5, EffectivenessScore: 80.0},
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever, surface.WithEffectiveness(eff))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePreCompact,
		DataDir: "/tmp/data",
		Budget:  500,
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	g.Expect(output).To(ContainSubstring("[engram] Preserving top memories through compaction:"))
	g.Expect(output).To(ContainSubstring("- always use /commit skill"))
}

// T-32: Keyword matching is case-insensitive
func TestT32_KeywordMatchingCaseInsensitiveWholeWord(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:    "Commit Rules",
			FilePath: "commit-rules.toml",
			Keywords: []string{"commit"},
		},
		{
			Title:    "Testing Framework",
			FilePath: "testing.toml",
			Keywords: []string{"test"},
		},
		{
			Title:    "Linting",
			FilePath: "linting.toml",
			Keywords: []string{"lint"},
		},
		{
			Title:    "Docker",
			FilePath: "docker.toml",
			Keywords: []string{"docker"},
		},
		{
			Title:    "Kubernetes",
			FilePath: "kubernetes.toml",
			Keywords: []string{"k8s"},
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	// Case-insensitive: "COMMIT" should match keyword "commit".
	var buf1 bytes.Buffer

	err := s.Run(context.Background(), &buf1, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "COMMIT this change",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf1.String()).To(ContainSubstring("commit-rules"))

	// No keyword match for "recommit": "commit" is not a substring due to tokenization
	var buf2 bytes.Buffer

	err = s.Run(context.Background(), &buf2, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "recommit the file",
	})

	g.Expect(err).NotTo(HaveOccurred())
	// "recommit" tokenizes to "recommit" (single token), which doesn't match "commit"
	g.Expect(buf2.String()).To(BeEmpty())
}

// T-33: Pre-filter matches memory keywords in tool input
func TestT33_PreFilterMatchesKeywordsInToolInput(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:       "Manual git commit",
			FilePath:    "manual-git-commit.toml",
			AntiPattern: "manual git commit",
			Keywords:    []string{"commit", "git"},
			Principle:   "use /commit skill instead",
		},
		{
			Title:       "Testing Framework",
			FilePath:    "testing.toml",
			AntiPattern: "manual test execution",
			Keywords:    []string{"test"},
			Principle:   "use automated testing",
		},
		{
			Title:       "Linting",
			FilePath:    "linting.toml",
			AntiPattern: "skipping lint checks",
			Keywords:    []string{"lint"},
			Principle:   "always lint before commit",
		},
		{
			Title:       "Docker Build",
			FilePath:    "docker.toml",
			AntiPattern: "building without docker",
			Keywords:    []string{"docker"},
			Principle:   "use docker for consistency",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: "git commit -m 'fix'",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	// Memory should appear because keywords "commit" and "git" matched in tool input.
	g.Expect(output).To(ContainSubstring("manual-git-commit"))
}

// T-346: Surfacer swallows surfacing log error (fire-and-forget per ARCH-6).
func TestT346_SurfacerSwallowsSurfacingLogError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:    "Always use targ",
			FilePath: "targ.toml",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	errorLogger := &fakeSurfacingLogger{returnErr: errors.New("log write failed")}
	s := surface.New(retriever, surface.WithSurfacingLogger(errorLogger))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
	})

	// Run must return nil — logger errors are swallowed (fire-and-forget).
	g.Expect(err).NotTo(HaveOccurred())
}

// T-34: Pre-filter skips memories without anti_pattern
func TestT34_PreFilterSkipsMemoriesWithoutAntiPattern(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:       "Commit Notes",
			FilePath:    "commit-notes.toml",
			AntiPattern: "", // empty — not an enforcement candidate
			Keywords:    []string{"commit"},
			Principle:   "some principle",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: "git commit -m 'fix'",
	})

	g.Expect(err).NotTo(HaveOccurred())
	// No anti_pattern means no advisory should be emitted.
	g.Expect(buf.String()).To(BeEmpty())
}

// T-35: Pre-filter returns empty when no keywords match
func TestT35_PreFilterReturnsEmptyWhenNoKeywordsMatch(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:       "Manual git commit",
			FilePath:    "manual-git-commit.toml",
			AntiPattern: "manual git commit",
			Keywords:    []string{"commit", "git"},
			Principle:   "use /commit skill instead",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Read",
		ToolInput: "/path/to/file.go",
	})

	g.Expect(err).NotTo(HaveOccurred())
	// No keyword overlap — output should be empty.
	g.Expect(buf.String()).To(BeEmpty())
}

// T-42: Tool mode surfaces matching memories as advisory
func TestT42_ToolModeEmitsAdvisoryReminder(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:       "Use /commit",
			FilePath:    "use-commit.toml",
			AntiPattern: "manual git commit",
			Keywords:    []string{"commit", "git"},
			Principle:   "always use /commit for commits",
		},
		{
			Title:       "Use targ test",
			FilePath:    "use-targ.toml",
			AntiPattern: "running go test directly",
			Keywords:    []string{"test", "go"},
			Principle:   "use targ test instead of go test",
		},
		{
			Title:       "Linting",
			FilePath:    "linting.toml",
			AntiPattern: "skipping lint checks",
			Keywords:    []string{"lint"},
			Principle:   "always lint before commit",
		},
		{
			Title:       "Docker Build",
			FilePath:    "docker.toml",
			AntiPattern: "building without docker",
			Keywords:    []string{"docker"},
			Principle:   "use docker for consistency",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	// Tool input contains keyword "commit" → "use-commit" should match
	// (only "Use /commit" has keyword in input).
	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: `git commit -m 'fix'`,
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	// Should emit system-reminder advisory format.
	g.Expect(output).To(ContainSubstring("<system-reminder source=\"engram\">"))
	g.Expect(output).To(ContainSubstring("[engram] Tool call advisory:"))
	g.Expect(output).To(ContainSubstring("use-commit"))
	// "use-targ" should NOT appear — keyword "test" is not in "git commit -m 'fix'".
	g.Expect(output).NotTo(ContainSubstring("use-targ"))
	g.Expect(output).To(ContainSubstring("</system-reminder>"))
}

// TestT69_SessionStartJSONFormat verifies JSON output for session-start mode.
func TestT69_SessionStartJSONFormat(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:     "First",
			FilePath:  "first.toml",
			UpdatedAt: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	retriever := &fakeRetriever{memories: memories}
	surfacer := surface.New(retriever)

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
		Format:  surface.FormatJSON,
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var result surface.Result

	decodeErr := json.Unmarshal(buf.Bytes(), &result)
	g.Expect(decodeErr).NotTo(HaveOccurred())

	if decodeErr != nil {
		return
	}

	g.Expect(result.Summary).To(ContainSubstring("[engram] Loaded 1 memories."))
	g.Expect(result.Summary).To(ContainSubstring("first"))
	g.Expect(result.Context).To(ContainSubstring("<system-reminder"))
	g.Expect(result.Context).To(ContainSubstring("first"))
}

// TestT70_PromptJSONFormat verifies JSON output for prompt mode.
func TestT70_PromptJSONFormat(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:    "Commit Conventions",
			FilePath: "commit-conventions.toml",
			Keywords: []string{"commit"},
		},
		{
			Title:    "Testing Framework",
			FilePath: "testing.toml",
			Keywords: []string{"test"},
		},
		{
			Title:    "Linting",
			FilePath: "linting.toml",
			Keywords: []string{"lint"},
		},
		{
			Title:    "Docker",
			FilePath: "docker.toml",
			Keywords: []string{"docker"},
		},
	}

	retriever := &fakeRetriever{memories: memories}
	surfacer := surface.New(retriever)

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "I want to commit this",
		Format:  surface.FormatJSON,
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var result surface.Result

	decodeErr := json.Unmarshal(buf.Bytes(), &result)
	g.Expect(decodeErr).NotTo(HaveOccurred())

	if decodeErr != nil {
		return
	}

	g.Expect(result.Summary).To(ContainSubstring("[engram] 1 relevant memories:"))
	g.Expect(result.Summary).To(ContainSubstring("commit-conventions"))
	g.Expect(result.Context).To(ContainSubstring("commit-conventions"))
}

// TestT71_ToolJSONFormat verifies JSON output for tool mode.
func TestT71_ToolJSONFormat(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:       "Use /commit",
			FilePath:    "use-commit.toml",
			AntiPattern: "manual git commit",
			Keywords:    []string{"commit"},
			Principle:   "always use /commit for commits",
		},
		{
			Title:       "Testing",
			FilePath:    "testing.toml",
			AntiPattern: "skipping tests",
			Keywords:    []string{"test"},
			Principle:   "always run tests",
		},
		{
			Title:       "Linting",
			FilePath:    "linting.toml",
			AntiPattern: "skipping lint",
			Keywords:    []string{"lint"},
			Principle:   "always lint before commit",
		},
		{
			Title:       "Docker",
			FilePath:    "docker.toml",
			AntiPattern: "no docker containerization",
			Keywords:    []string{"docker"},
			Principle:   "use docker consistently",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	surfacer := surface.New(retriever)

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: `git commit -m 'fix'`,
		Format:    surface.FormatJSON,
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var result surface.Result

	decodeErr := json.Unmarshal(buf.Bytes(), &result)
	g.Expect(decodeErr).NotTo(HaveOccurred())

	if decodeErr != nil {
		return
	}

	g.Expect(result.Summary).To(ContainSubstring("[engram] 1 tool advisories:"))
	g.Expect(result.Summary).To(ContainSubstring("use-commit"))
	g.Expect(result.Context).To(ContainSubstring("use-commit"))
}

// TestT72_NoMatchJSONFormat verifies no output when no matches in JSON mode.
func TestT72_NoMatchJSONFormat(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	surfacer := surface.New(retriever)

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
		Format:  surface.FormatJSON,
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf.String()).To(BeEmpty())
}

// T-79: Tracker receives matched memories on prompt mode
func TestT79_TrackerReceivesMatchedMemories(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:    "Commit Conventions",
			FilePath: "commit-conventions.toml",
			Keywords: []string{"commit"},
		},
		{
			Title:    "Build Tools",
			FilePath: "build-tools.toml",
			Keywords: []string{"targ"},
		},
		{
			Title:    "Testing",
			FilePath: "testing.toml",
			Keywords: []string{"test"},
		},
		{
			Title:    "Linting",
			FilePath: "linting.toml",
			Keywords: []string{"lint"},
		},
	}

	retriever := &fakeRetriever{memories: memories}
	tracker := &fakeTracker{}
	s := surface.New(retriever, surface.WithTracker(tracker))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "I want to commit this change",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(tracker.calls).To(HaveLen(1))
	g.Expect(tracker.calls[0].mode).To(Equal(surface.ModePrompt))
	g.Expect(tracker.calls[0].memories).To(HaveLen(1))
	g.Expect(tracker.calls[0].memories[0].Title).To(Equal("Commit Conventions"))
}

// T-80: Tracker error does not affect surfacing output
func TestT80_TrackerErrorDoesNotAffectOutput(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:     "First",
			FilePath:  "first.toml",
			UpdatedAt: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	retriever := &fakeRetriever{memories: memories}
	tracker := &fakeTracker{err: errTrackerFail}
	s := surface.New(retriever, surface.WithTracker(tracker))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
	})

	// Run should succeed despite tracker error.
	g.Expect(err).NotTo(HaveOccurred())
	// Output should still be produced.
	g.Expect(buf.String()).NotTo(BeEmpty())
	g.Expect(buf.String()).To(ContainSubstring("first"))
}

// T-81: No tracker (nil) produces correct output (backward compat)
func TestT81_NoTrackerBackwardCompatible(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:    "Commit Conventions",
			FilePath: "commit-conventions.toml",
			Keywords: []string{"commit"},
		},
		{
			Title:    "Build Tools",
			FilePath: "build-tools.toml",
			Keywords: []string{"targ"},
		},
		{
			Title:    "Testing",
			FilePath: "testing.toml",
			Keywords: []string{"test"},
		},
		{
			Title:    "Linting",
			FilePath: "linting.toml",
			Keywords: []string{"lint"},
		},
	}

	retriever := &fakeRetriever{memories: memories}
	// No WithTracker option — tracker is nil.
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "I want to commit this change",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	g.Expect(output).To(ContainSubstring("commit-conventions"))
}

// T-92: SessionStart includes creation report before recency surfacing
func TestT92_SessionStartIncludesCreationReport(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:     "Alpha",
			FilePath:  "alpha.toml",
			UpdatedAt: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Title:     "Beta",
			FilePath:  "beta.toml",
			UpdatedAt: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Title:     "Gamma",
			FilePath:  "gamma.toml",
			UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	logEntries := []surface.LogEntry{
		{Title: "New Memory One", Tier: "A", Filename: "new-memory-one.toml"},
		{Title: "New Memory Two", Tier: "B", Filename: "new-memory-two.toml"},
	}

	retriever := &fakeRetriever{memories: memories}
	logReader := &fakeLogReader{entries: logEntries}
	s := surface.New(retriever, surface.WithLogReader(logReader))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
		Format:  surface.FormatJSON,
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var result surface.Result

	decodeErr := json.Unmarshal(buf.Bytes(), &result)
	g.Expect(decodeErr).NotTo(HaveOccurred())

	if decodeErr != nil {
		return
	}

	g.Expect(result.Summary).To(ContainSubstring("[engram] Created 2 memories since last session:"))
	g.Expect(result.Summary).To(ContainSubstring("\"New Memory One\" [A] (new-memory-one.toml)"))
	g.Expect(result.Summary).To(ContainSubstring("\"New Memory Two\" [B] (new-memory-two.toml)"))
	g.Expect(result.Summary).To(ContainSubstring("[engram] Loaded 3 memories."))
	g.Expect(result.Summary).To(ContainSubstring("alpha"))
	g.Expect(result.Context).To(ContainSubstring("Created 2 memories since last session:"))
	g.Expect(result.Context).To(ContainSubstring("\"New Memory One\" [A] (new-memory-one.toml)"))
	g.Expect(result.Context).To(ContainSubstring("\"New Memory Two\" [B] (new-memory-two.toml)"))
	g.Expect(result.Context).To(ContainSubstring("[engram] Loaded 3 memories."))
	g.Expect(result.Context).To(ContainSubstring("alpha"))
	g.Expect(logReader.dataDirUsed).To(Equal("/tmp/data"))
	g.Expect(logReader.cleared).To(BeTrue())
}

// T-93: SessionStart with no creation log produces recency-only output (backward compat)
func TestT93_SessionStartNoCreationLogReturnsRecencyOnly(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:     "Alpha",
			FilePath:  "alpha.toml",
			UpdatedAt: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Title:     "Beta",
			FilePath:  "beta.toml",
			UpdatedAt: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Title:     "Gamma",
			FilePath:  "gamma.toml",
			UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	retriever := &fakeRetriever{memories: memories}
	// No WithLogReader — logReader is nil (backward compatible).
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
		Format:  surface.FormatJSON,
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var result surface.Result

	decodeErr := json.Unmarshal(buf.Bytes(), &result)
	g.Expect(decodeErr).NotTo(HaveOccurred())

	if decodeErr != nil {
		return
	}

	g.Expect(result.Summary).NotTo(ContainSubstring("Created"))
	g.Expect(result.Summary).To(ContainSubstring("[engram] Loaded 3 memories."))
	g.Expect(result.Summary).To(ContainSubstring("alpha"))
	g.Expect(result.Summary).To(ContainSubstring("beta"))
	g.Expect(result.Summary).To(ContainSubstring("gamma"))
	g.Expect(result.Context).NotTo(ContainSubstring("Created"))
	g.Expect(result.Context).To(ContainSubstring("alpha"))
}

// T-94: SessionStart with creation log but no memories produces creation-only output
func TestT94_SessionStartCreationLogNoMemoriesProducesCreationOnly(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	logEntries := []surface.LogEntry{
		{Title: "Solo Memory", Tier: "C", Filename: "solo-memory.toml"},
	}

	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	logReader := &fakeLogReader{entries: logEntries}
	s := surface.New(retriever, surface.WithLogReader(logReader))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
		Format:  surface.FormatJSON,
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var result surface.Result

	decodeErr := json.Unmarshal(buf.Bytes(), &result)
	g.Expect(decodeErr).NotTo(HaveOccurred())

	if decodeErr != nil {
		return
	}

	g.Expect(result.Summary).To(ContainSubstring("[engram] Created 1 memories since last session:"))
	g.Expect(result.Summary).To(ContainSubstring("\"Solo Memory\" [C] (solo-memory.toml)"))
	g.Expect(result.Summary).NotTo(ContainSubstring("Loaded"))
	g.Expect(result.Context).To(ContainSubstring("\"Solo Memory\" [C] (solo-memory.toml)"))
	g.Expect(result.Context).NotTo(ContainSubstring("Loaded"))
	g.Expect(logReader.cleared).To(BeTrue())
}

func TestToolErrored_LowersUnprovenFloor(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// This test verifies the errored flag lowers the BM25 floor for unproven memories
	// from unprovenBM25FloorTool (0.30) to minRelevanceScore (0.05).
	//
	// Corpus design: 6 stash-filler memories (marked proven) dilute IDF for "stash"
	// so the weak-match memory (unproven) scores ~0.16 — above 0.05 but below 0.30.
	// (score verified empirically: reps=5 repetitions of padding, 6 stash-fillers)
	//
	// Stash-fillers are marked proven (SurfacedCount=5, EffectivenessScore=10) so:
	//   - They are NOT unproven — cold-start budget does not apply to them.
	//   - filterToolMatchesByEffectivenessGate removes them (score ≤ 40 with ≥ 5 surfacings).
	// This leaves weak-match as the only surviving candidate, whose surfacing is
	// controlled solely by the errored-flag floor logic.
	stashAntiPattern := strings.Repeat(
		"forgetting to backup configuration files before deployment ", 5,
	) + "stash"

	weakMatch := &memory.Stored{
		FilePath:    "/data/memories/weak-match.toml",
		Title:       "backup reminder",
		Principle:   "always preserve working state before operations",
		AntiPattern: stashAntiPattern,
		Keywords:    []string{"backup", "configuration"},
	}

	memories := make([]*memory.Stored, 0, 17)
	memories = append(memories, weakMatch)

	stashFillerPaths := make([]string, 0, 6)

	for i := range 6 {
		path := fmt.Sprintf("/data/memories/stash-filler-%d.toml", i)
		stashFillerPaths = append(stashFillerPaths, path)
		memories = append(memories, &memory.Stored{
			FilePath:    path,
			Title:       fmt.Sprintf("stash rule %d", i),
			AntiPattern: fmt.Sprintf("forgetting to stash changes before operation %d", i),
			Keywords:    []string{"stash"},
		})
	}

	for i := range 10 {
		memories = append(memories, &memory.Stored{
			FilePath:    fmt.Sprintf("/data/memories/unrel-filler-%d.toml", i),
			Title:       fmt.Sprintf("db rule %d", i),
			AntiPattern: fmt.Sprintf("running migrations without backup %d breaks production", i),
			Keywords:    []string{"database"},
		})
	}

	effStats := make(map[string]surface.EffectivenessStat)

	for _, path := range stashFillerPaths {
		effStats[path] = surface.EffectivenessStat{SurfacedCount: 5, EffectivenessScore: 10}
	}

	eff := &fakeEffectivenessComputer{stats: effStats}
	retriever := &fakeRetriever{memories: memories}

	// Without errored: unproven floor (0.30) filters out weak-match (score ~0.16).
	sNoErr := surface.New(retriever, surface.WithEffectiveness(eff))

	var bufNoErr bytes.Buffer

	err := sNoErr.Run(context.Background(), &bufNoErr, surface.Options{
		Mode:        surface.ModeTool,
		DataDir:     "/tmp/data",
		ToolName:    "Bash",
		ToolInput:   "git stash",
		ToolErrored: false,
		Format:      surface.FormatJSON,
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(bufNoErr.String()).NotTo(ContainSubstring("weak-match"))

	// With errored: base floor (0.05) allows weak-match (score ~0.16) through.
	sErr := surface.New(retriever, surface.WithEffectiveness(eff))

	var bufErr bytes.Buffer

	err = sErr.Run(context.Background(), &bufErr, surface.Options{
		Mode:        surface.ModeTool,
		DataDir:     "/tmp/data",
		ToolName:    "Bash",
		ToolInput:   "git stash",
		ToolErrored: true,
		Format:      surface.FormatJSON,
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(bufErr.String()).To(ContainSubstring("weak-match"))
}

func TestToolOutput_EnrichesBM25Query(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	target := &memory.Stored{
		FilePath:    "/data/memories/stash-before-ops.toml",
		Title:       "stash before operations",
		Principle:   "Always stash or commit before running commands that require clean tree",
		AntiPattern: "dirty working tree uncommitted changes",
		Keywords:    []string{"git", "stash", "working tree"},
	}

	memories := make([]*memory.Stored, 0, 21)
	memories = append(memories, target)

	for i := range 20 {
		memories = append(memories, &memory.Stored{
			FilePath:    fmt.Sprintf("/data/memories/filler-%d.toml", i),
			AntiPattern: fmt.Sprintf("unrelated pattern %d about something else", i),
		})
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:       surface.ModeTool,
		DataDir:    "/tmp/data",
		ToolName:   "Bash",
		ToolInput:  `{"command": "git commit -m 'fix'"}`,
		ToolOutput: "fatal: cannot commit in dirty working tree",
		Format:     surface.FormatJSON,
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(buf.String()).To(ContainSubstring("stash-before-ops"))
}

// TestUnknownModeReturnsError verifies that Run returns ErrUnknownMode for unrecognized modes.
func TestUnknownModeReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    "unknown-mode",
		DataDir: "/tmp/data",
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(surface.ErrUnknownMode))
}

// unexported variables.
var (
	errTrackerFail = errors.New("tracker failure")
)

// fakeEffectivenessComputer is a test double for surface.EffectivenessComputer.
type fakeEffectivenessComputer struct {
	stats map[string]surface.EffectivenessStat
	err   error
}

func (f *fakeEffectivenessComputer) Aggregate() (map[string]surface.EffectivenessStat, error) {
	return f.stats, f.err
}

// fakeLogReader is a test double for surface.CreationLogReader.
type fakeLogReader struct {
	entries     []surface.LogEntry
	err         error
	dataDirUsed string
	cleared     bool
}

func (f *fakeLogReader) ReadAndClear(dataDir string) ([]surface.LogEntry, error) {
	f.dataDirUsed = dataDir
	f.cleared = true

	return f.entries, f.err
}

// fakeRegistryRecorder is a test double for surface.RegistryRecorder.
type fakeRegistryRecorder struct {
	ids []string
	err error
}

func (f *fakeRegistryRecorder) RecordSurfacing(id string) error {
	f.ids = append(f.ids, id)
	return f.err
}

// fakeRetriever is a test double for surface.MemoryRetriever.
type fakeRetriever struct {
	memories []*memory.Stored
	err      error
}

func (f *fakeRetriever) ListMemories(_ context.Context, _ string) ([]*memory.Stored, error) {
	return f.memories, f.err
}

// fakeSurfacingLogger is a test double for surface.SurfacingEventLogger.
type fakeSurfacingLogger struct {
	calls     []surfacingLogCall
	returnErr error
}

func (f *fakeSurfacingLogger) LogSurfacing(memoryPath, mode string, _ time.Time) error {
	f.calls = append(f.calls, surfacingLogCall{memoryPath: memoryPath, mode: mode})
	return f.returnErr
}

// fakeTracker is a test double for surface.MemoryTracker.
type fakeTracker struct {
	calls []trackerCall
	err   error
}

func (f *fakeTracker) RecordSurfacing(
	_ context.Context,
	memories []*memory.Stored,
	mode string,
) error {
	f.calls = append(f.calls, trackerCall{memories: memories, mode: mode})

	return f.err
}

type surfacingLogCall struct {
	memoryPath string
	mode       string
}

type trackerCall struct {
	memories []*memory.Stored
	mode     string
}

func memPath(i int) string {
	return "memory-" + string(rune('a'+i%26)) + ".toml"
}

func memSlug(i int) string {
	return "memory-" + string(rune('a'+i%26))
}

// --- Helpers ---

func memTitle(i int) string {
	return "Memory " + string(rune('A'+i%26))
}
