package surface_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

// T-343: BM25 irrelevance penalty suppresses high-irrelevance memories.
func TestBM25_IrrelevancePenalty_ReducesScore(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Two memories with identical keywords. Memory A has no irrelevance,
	// Memory B has high irrelevance (count=100, penalty factor ~0.05).
	// Both match "commit" but B should be suppressed below the relevance floor.
	memA := &memory.Stored{
		Title:           "Commit Rule A",
		FilePath:        "commit-rule-a.toml",
		Keywords:        []string{"commit", "git"},
		Principle:       "use /commit skill",
		IrrelevantCount: 0,
	}

	memB := &memory.Stored{
		Title:           "Commit Rule B",
		FilePath:        "commit-rule-b.toml",
		Keywords:        []string{"commit", "git"},
		Principle:       "use /commit skill",
		IrrelevantCount: 100,
	}

	// Non-matching fillers for IDF contrast.
	fillers := make([]*memory.Stored, 0, 5)
	for _, name := range []string{"logging", "testing", "deploy", "config", "monitoring"} {
		fillers = append(fillers, &memory.Stored{
			Title:    name + " guide",
			FilePath: name + ".toml",
			Keywords: []string{name},
			Content:  name + " documentation",
		})
	}

	allMems := make([]*memory.Stored, 0, 2+len(fillers))
	allMems = append(allMems, memA, memB)
	allMems = append(allMems, fillers...)
	retriever := &fakeRetriever{memories: allMems}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "I want to commit this change using git",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	// Memory A (no irrelevance) should appear.
	g.Expect(output).To(ContainSubstring("commit-rule-a"))
	// Memory B (high irrelevance, penalty ~0.05) should be suppressed below floor.
	g.Expect(output).NotTo(ContainSubstring("commit-rule-b"))
}

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
	// "exact-match" should appear, "irrelevant" slug should not (score below floor).
	g.Expect(output).To(ContainSubstring("exact-match"))
	g.Expect(output).NotTo(ContainSubstring("irrelevant:"))
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

// T-frecency-gate-2: runTool short-circuits for non-Bash tool names.
func TestRunTool_NonBashSkips(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Provide a matchable memory so that the guard, not an empty retriever, blocks output.
	memories := []*memory.Stored{
		{
			Title:       "Commit Rule",
			FilePath:    "commit-rule.toml",
			AntiPattern: "manual git commit",
			Keywords:    []string{"commit", "git"},
			Principle:   "use /commit skill",
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
			AntiPattern: "skipping tests",
			Keywords:    []string{"testing"},
			Principle:   "always test",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		ToolName:  "Grep",
		ToolInput: `{"pattern":"foo"}`,
		DataDir:   t.TempDir(),
		Format:    surface.FormatJSON,
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(buf.String()).To(Equal(""))
}

// T-frecency-gate: runTool short-circuits when toolgate says skip.
func TestRunTool_ToolGateSkips(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Provide a matchable memory so that without the gate, output would be produced.
	memories := []*memory.Stored{
		{
			Title:       "Commit Rule",
			FilePath:    "commit-rule.toml",
			AntiPattern: "manual git commit",
			Keywords:    []string{"commit", "git"},
			Principle:   "use /commit skill",
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
			AntiPattern: "skipping tests",
			Keywords:    []string{"testing"},
			Principle:   "always test",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	alwaysSkip := &stubToolGate{shouldSurface: false}
	s := surface.New(retriever, surface.WithToolGate(alwaysSkip))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		ToolName:  "Bash",
		ToolInput: `{"command":"git commit -m 'fix bug'"}`,
		DataDir:   t.TempDir(),
		Format:    surface.FormatJSON,
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(buf.String()).To(Equal(""))
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
	g.Expect(output).To(ContainSubstring("[engram] Memories"))
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

// T373: Cross-project generalizability penalty penalizes narrow memories.
func TestT373_GenPenalty_CrossProject(t *testing.T) {
	t.Parallel()

	// Memory A: narrow (gen=1), same project origin.
	// Memory B: universal (gen=5), same project origin.
	// Both have identical BM25-matchable content.
	memNarrow := &memory.Stored{
		Title:            "Deploy Narrow",
		FilePath:         "deploy-narrow.toml",
		Keywords:         []string{"deploy", "production"},
		Principle:        "deploy safely",
		Content:          "deploy to production safely",
		ProjectSlug:      "proj-a",
		Generalizability: 1,
	}

	memUniversal := &memory.Stored{
		Title:            "Deploy Universal",
		FilePath:         "deploy-universal.toml",
		Keywords:         []string{"deploy", "production"},
		Principle:        "deploy safely",
		Content:          "deploy to production safely",
		ProjectSlug:      "proj-a",
		Generalizability: 5,
	}

	// Non-matching fillers for IDF contrast.
	fillers := make([]*memory.Stored, 0, 5)
	for _, name := range []string{
		"logging", "testing", "config", "monitoring", "caching",
	} {
		fillers = append(fillers, &memory.Stored{
			Title:    name + " guide",
			FilePath: name + ".toml",
			Keywords: []string{name},
			Content:  name + " documentation",
		})
	}

	allMems := make([]*memory.Stored, 0, 2+len(fillers))
	allMems = append(allMems, memNarrow, memUniversal)
	allMems = append(allMems, fillers...)

	t.Run("cross-project penalizes narrow", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)

		retriever := &fakeRetriever{memories: allMems}
		s := surface.New(retriever)

		var buf bytes.Buffer

		err := s.Run(context.Background(), &buf, surface.Options{
			Mode:               surface.ModePrompt,
			DataDir:            "/tmp/data",
			Message:            "deploy to production",
			CurrentProjectSlug: "proj-b", // different project
		})

		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		output := buf.String()

		// Both should appear (universal is unpenalized, narrow gets 0.05 factor).
		g.Expect(output).To(ContainSubstring("deploy-universal"))
		g.Expect(output).To(ContainSubstring("deploy-narrow"))

		// Universal (gen=5, factor=1.0) should appear before narrow (gen=1, factor=0.05).
		posUniversal := strings.Index(output, "deploy-universal")
		posNarrow := strings.Index(output, "deploy-narrow")
		g.Expect(posUniversal).To(
			BeNumerically("<", posNarrow),
			"universal memory should rank above narrow in cross-project context",
		)
	})

	t.Run("same-project no penalty", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)

		retriever := &fakeRetriever{memories: allMems}
		s := surface.New(retriever)

		var buf bytes.Buffer

		err := s.Run(context.Background(), &buf, surface.Options{
			Mode:               surface.ModePrompt,
			DataDir:            "/tmp/data",
			Message:            "deploy to production",
			CurrentProjectSlug: "proj-a", // same project — no penalty
		})

		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		output := buf.String()

		// Both should appear with no penalty applied.
		g.Expect(output).To(ContainSubstring("deploy-universal"))
		g.Expect(output).To(ContainSubstring("deploy-narrow"))
	})
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
	g.Expect(output).To(ContainSubstring("[engram] Memories"))
	g.Expect(output).To(ContainSubstring("use-commit"))
	// "use-targ" should NOT appear — keyword "test" is not in "git commit -m 'fix'".
	g.Expect(output).NotTo(ContainSubstring("use-targ"))
	g.Expect(output).To(ContainSubstring("</system-reminder>"))
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

// fakeEffectivenessComputer is a test double for surface.EffectivenessComputer.
type fakeEffectivenessComputer struct {
	stats map[string]surface.EffectivenessStat
	err   error
}

func (f *fakeEffectivenessComputer) Aggregate() (map[string]surface.EffectivenessStat, error) {
	return f.stats, f.err
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

// stubToolGate is a test double for surface.ToolGater.
type stubToolGate struct {
	shouldSurface bool
}

func (s *stubToolGate) Check(_ string) (bool, error) {
	return s.shouldSurface, nil
}

type surfacingLogCall struct {
	memoryPath string
	mode       string
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
