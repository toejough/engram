package surface_test

// P4-full: Cluster dedup + cross-source suppression + transcript suppression
// REQ-P4f-1: Cluster dedup — keep highest-effectiveness linked pair member
// REQ-P4f-2: Cross-source suppression — skip if covered by CLAUDE.md/rule/skill
// REQ-P4f-3: Transcript suppression — skip if keywords appear in recent window
// REQ-P4f-4: Suppression logging — log each decision
// REQ-P4f-5: Suppression rate metric — suppressions / (surfaced + suppressed)

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

// T-P4f-10: ClusterDedup suppression event has correct fields.
func TestTP4f10_ClusterDedupEventFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{Title: "Winner", FilePath: "winner.toml", UpdatedAt: time.Now()},
		{Title: "Loser", FilePath: "loser.toml", UpdatedAt: time.Now()},
	}

	eff := &fakeEffectivenessComputer{
		stats: map[string]surface.EffectivenessStat{
			"winner.toml": {SurfacedCount: 10, EffectivenessScore: 90.0},
			"loser.toml":  {SurfacedCount: 10, EffectivenessScore: 55.0},
		},
	}

	linkReader := &fakeP3LinkReaderByPath{
		links: map[string][]surface.LinkGraphLink{
			"winner.toml": {{Target: "loser.toml", Weight: 0.7, Basis: "concept_overlap"}},
		},
	}

	logger := &fakeSuppressionLogger{}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever,
		surface.WithEffectiveness(eff),
		surface.WithClusterDedupReader(linkReader),
		surface.WithSuppressionEventLogger(logger),
	)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Find the cluster_dedup event.
	var dedupEvent *surface.SuppressionEvent

	for i := range logger.events {
		if logger.events[i].Reason == surface.SuppressionReasonClusterDedup {
			dedupEvent = &logger.events[i]

			break
		}
	}

	g.Expect(dedupEvent).NotTo(BeNil(), "cluster_dedup suppression event should be logged")

	if dedupEvent == nil {
		return
	}

	g.Expect(dedupEvent.MemoryID).To(Equal("loser.toml"))
	g.Expect(dedupEvent.SuppressedBy).To(Equal("winner.toml"))
}

// T-P4f-1: ClusterDedup keeps higher-effectiveness memory when linked pair both surface.
func TestTP4f1_ClusterDedupKeepsHigherEffectiveness(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:     "High Eff Memory",
			FilePath:  "high-eff.toml",
			UpdatedAt: time.Now(),
		},
		{
			Title:     "Low Eff Memory",
			FilePath:  "low-eff.toml",
			UpdatedAt: time.Now(),
		},
	}

	eff := &fakeEffectivenessComputer{
		stats: map[string]surface.EffectivenessStat{
			"high-eff.toml": {SurfacedCount: 10, EffectivenessScore: 80.0},
			"low-eff.toml":  {SurfacedCount: 10, EffectivenessScore: 50.0},
		},
	}

	// Link high-eff → low-eff so they form a cluster.
	linkReader := &fakeP3LinkReaderByPath{
		links: map[string][]surface.LinkGraphLink{
			"high-eff.toml": {
				{Target: "low-eff.toml", Weight: 0.8, Basis: "concept_overlap"},
			},
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever,
		surface.WithEffectiveness(eff),
		surface.WithClusterDedupReader(linkReader),
	)

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
	g.Expect(output).To(ContainSubstring("high-eff"), "higher-effectiveness memory should surface")
	g.Expect(output).NotTo(ContainSubstring("low-eff"), "lower-effectiveness linked memory should be suppressed")
}

// T-P4f-2: ClusterDedup: two unlinked memories both surface normally.
func TestTP4f2_ClusterDedupDoesNotSuppressUnlinkedMemories(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{Title: "Alpha", FilePath: "alpha.toml", UpdatedAt: time.Now()},
		{Title: "Beta", FilePath: "beta.toml", UpdatedAt: time.Now()},
	}

	// No links between alpha and beta.
	linkReader := &fakeP3LinkReader{}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever, surface.WithClusterDedupReader(linkReader))

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
	g.Expect(output).To(ContainSubstring("alpha"), "unlinked memory alpha should surface")
	g.Expect(output).To(ContainSubstring("beta"), "unlinked memory beta should surface")
}

// T-P4f-3: CrossRefChecker suppresses covered memories in session-start.
func TestTP4f3_CrossRefCheckerSuppressesCoveredMemory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{Title: "Covered Memory", FilePath: "covered.toml", UpdatedAt: time.Now()},
		{Title: "Unique Memory", FilePath: "unique.toml", UpdatedAt: time.Now()},
	}

	checker := &fakeCrossRefChecker{
		covered: map[string]string{
			"covered.toml": "CLAUDE.md",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever, surface.WithCrossRefChecker(checker))

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
	g.Expect(output).NotTo(ContainSubstring("covered"), "CLAUDE.md-covered memory should be suppressed")
	g.Expect(output).To(ContainSubstring("unique"), "uncovered memory should surface")
}

// T-P4f-4: Transcript suppression skips memories whose keywords appear in transcript window.
func TestTP4f4_TranscriptSuppressionSkipsMatchingKeywords(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:     "Targ Rule",
			FilePath:  "targ-rule.toml",
			Keywords:  []string{"targ", "build"},
			UpdatedAt: time.Now(),
		},
		{
			Title:     "Git Rule",
			FilePath:  "git-rule.toml",
			Keywords:  []string{"git", "commit"},
			UpdatedAt: time.Now(),
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	// Transcript window mentions "targ" — should suppress targ-rule, not git-rule.
	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:             surface.ModeSessionStart,
		DataDir:          "/tmp/data",
		TranscriptWindow: "I ran targ check-full and it passed all tests",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	g.Expect(output).NotTo(ContainSubstring("targ-rule"), "memory with transcript-matched keyword should be suppressed")
	g.Expect(output).To(ContainSubstring("git-rule"), "memory without matching keyword should surface")
}

// T-P4f-5: Transcript suppression is case-insensitive.
func TestTP4f5_TranscriptSuppressionIsCaseInsensitive(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:     "Deploy Rule",
			FilePath:  "deploy-rule.toml",
			Keywords:  []string{"Deploy"},
			UpdatedAt: time.Now(),
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:             surface.ModeSessionStart,
		DataDir:          "/tmp/data",
		TranscriptWindow: "we should deploy to staging first",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(buf.String()).NotTo(ContainSubstring("deploy-rule"),
		"case-insensitive keyword match should suppress memory")
}

// T-P4f-6: SuppressionEventLogger receives events with correct fields.
func TestTP4f6_SuppressionLoggerReceivesEvents(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{Title: "Covered", FilePath: "covered.toml", UpdatedAt: time.Now()},
		{Title: "Other", FilePath: "other.toml", UpdatedAt: time.Now()},
	}

	checker := &fakeCrossRefChecker{
		covered: map[string]string{"covered.toml": "rules/go.md"},
	}

	logger := &fakeSuppressionLogger{}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever,
		surface.WithCrossRefChecker(checker),
		surface.WithSuppressionEventLogger(logger),
	)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(logger.events).To(HaveLen(1))
	g.Expect(logger.events[0].MemoryID).To(Equal("covered.toml"))
	g.Expect(logger.events[0].Reason).To(Equal(surface.SuppressionReasonCrossSource))
	g.Expect(logger.events[0].SuppressedBy).To(Equal("rules/go.md"))
	g.Expect(logger.events[0].Timestamp).NotTo(BeZero())
}

// T-P4f-7: SuppressionStats in Result shows correct suppression rate.
func TestTP4f7_SuppressionStatsInResult(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// 2 memories, 1 suppressed by cross-ref → rate = 1/2 = 0.5
	memories := []*memory.Stored{
		{Title: "Covered", FilePath: "covered.toml", UpdatedAt: time.Now()},
		{Title: "Surfaced", FilePath: "surfaced.toml", UpdatedAt: time.Now()},
	}

	checker := &fakeCrossRefChecker{
		covered: map[string]string{"covered.toml": "CLAUDE.md"},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever, surface.WithCrossRefChecker(checker))

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

	// Check that result contains suppression stats.
	output := buf.String()
	g.Expect(output).To(ContainSubstring(`"suppressionStats"`))
	g.Expect(output).To(ContainSubstring(`"suppressed":1`))
	g.Expect(output).To(ContainSubstring(`"surfaced":1`))
}

// T-P4f-8: Empty transcript window does not suppress any memories.
func TestTP4f8_EmptyTranscriptWindowNoSuppression(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:     "Rule A",
			FilePath:  "rule-a.toml",
			Keywords:  []string{"targ", "build"},
			UpdatedAt: time.Now(),
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:             surface.ModeSessionStart,
		DataDir:          "/tmp/data",
		TranscriptWindow: "", // empty — no suppression
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(buf.String()).To(ContainSubstring("rule-a"), "no transcript suppression when window is empty")
}

// T-P4f-9: Transcript suppression also applies in prompt mode.
func TestTP4f9_TranscriptSuppressionInPromptMode(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:     "Targ Rule",
			FilePath:  "targ-rule.toml",
			Keywords:  []string{"targ"},
			Content:   "use targ for builds",
			Principle: "use targ",
			UpdatedAt: time.Now(),
		},
		{
			Title:     "Filler A",
			FilePath:  "filler-a.toml",
			Keywords:  []string{"unrelated"},
			Content:   "unrelated rule content",
			UpdatedAt: time.Now(),
		},
		{
			Title:     "Filler B",
			FilePath:  "filler-b.toml",
			Keywords:  []string{"testing"},
			Content:   "write tests",
			UpdatedAt: time.Now(),
		},
		{
			Title:     "Filler C",
			FilePath:  "filler-c.toml",
			Keywords:  []string{"deploy"},
			Content:   "deploy safely",
			UpdatedAt: time.Now(),
		},
		{
			Title:     "Filler D",
			FilePath:  "filler-d.toml",
			Keywords:  []string{"monitor"},
			Content:   "monitor metrics",
			UpdatedAt: time.Now(),
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:             surface.ModePrompt,
		DataDir:          "/tmp/data",
		Message:          "should I use targ for builds",
		TranscriptWindow: "yes run targ test to verify",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(buf.String()).NotTo(ContainSubstring("targ-rule"),
		"transcript-matched memory should be suppressed in prompt mode")
}

// unexported variables.
var (
	_ = strings.Contains
	_ = memSlug
)

// --- Fake helpers ---

// fakeCrossRefChecker is a test double for surface.CrossRefChecker.
type fakeCrossRefChecker struct {
	covered map[string]string // memoryID → source
	err     error
}

func (f *fakeCrossRefChecker) IsCoveredBySource(memoryID string) (bool, string, error) {
	if f.err != nil {
		return false, "", f.err
	}

	source, ok := f.covered[memoryID]

	return ok, source, nil
}

// fakeSuppressionLogger is a test double for surface.SuppressionEventLogger.
type fakeSuppressionLogger struct {
	events []surface.SuppressionEvent
}

func (f *fakeSuppressionLogger) LogSuppression(event surface.SuppressionEvent) error {
	f.events = append(f.events, event)

	return nil
}
