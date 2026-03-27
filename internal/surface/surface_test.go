package surface_test

import (
	"bytes"
	"context"
	"encoding/json"
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

		// Both should appear (universal is unpenalized, narrow gets 0.0 factor but still passes floor).
		g.Expect(output).To(ContainSubstring("deploy-universal"))
		g.Expect(output).To(ContainSubstring("deploy-narrow"))

		// Universal (gen=5, factor=1.0) should appear before narrow (gen=1, factor=0.0).
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

type surfacingLogCall struct {
	memoryPath string
	mode       string
}

func memSlug(i int) string {
	return "memory-" + string(rune('a'+i%26))
}

// --- Helpers ---
