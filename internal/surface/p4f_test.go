package surface_test

// P4-full: Cross-source suppression + transcript suppression
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

// T-P4f-3: CrossRefChecker suppresses covered memories in prompt mode.
func TestTP4f3_CrossRefCheckerSuppressesCoveredMemory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:    "Covered Memory",
			FilePath: "covered.toml",
			Keywords: []string{"xyzcrossref"},
			Content:  "xyzcrossref rule content",
		},
		{
			Title:    "Unique Memory",
			FilePath: "unique.toml",
			Keywords: []string{"xyzcrossref"},
			Content:  "xyzcrossref other content",
		},
		{Title: "Filler A", FilePath: "filler-a.toml", Keywords: []string{"logging"}},
		{Title: "Filler B", FilePath: "filler-b.toml", Keywords: []string{"testing"}},
		{Title: "Filler C", FilePath: "filler-c.toml", Keywords: []string{"deploy"}},
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
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "xyzcrossref rule",
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
			Title:    "Targ Rule",
			FilePath: "targ-rule.toml",
			Keywords: []string{"targ", "build"},
			Content:  "xyztranscript targ build rule",
		},
		{
			Title:    "Git Rule",
			FilePath: "git-rule.toml",
			Keywords: []string{"git", "commit"},
			Content:  "xyztranscript git commit rule",
		},
		{Title: "Filler A", FilePath: "filler-a.toml", Keywords: []string{"logging"}},
		{Title: "Filler B", FilePath: "filler-b.toml", Keywords: []string{"testing"}},
		{Title: "Filler C", FilePath: "filler-c.toml", Keywords: []string{"deploy"}},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	// Transcript window mentions "targ" — should suppress targ-rule, not git-rule.
	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:             surface.ModePrompt,
		DataDir:          "/tmp/data",
		Message:          "xyztranscript rule",
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
			Title:    "Deploy Rule",
			FilePath: "deploy-rule.toml",
			Keywords: []string{"Deploy"},
			Content:  "xyzdeployrule content for matching",
		},
		{Title: "Filler A", FilePath: "filler-a.toml", Keywords: []string{"logging"}},
		{Title: "Filler B", FilePath: "filler-b.toml", Keywords: []string{"testing"}},
		{Title: "Filler C", FilePath: "filler-c.toml", Keywords: []string{"config"}},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:             surface.ModePrompt,
		DataDir:          "/tmp/data",
		Message:          "xyzdeployrule content",
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
		{
			Title:    "Covered",
			FilePath: "covered.toml",
			Keywords: []string{"xyzsupplog"},
			Content:  "xyzsupplog covered rule",
		},
		{
			Title:    "Other",
			FilePath: "other.toml",
			Keywords: []string{"xyzsupplog"},
			Content:  "xyzsupplog other rule",
		},
		{Title: "Filler A", FilePath: "filler-a.toml", Keywords: []string{"logging"}},
		{Title: "Filler B", FilePath: "filler-b.toml", Keywords: []string{"testing"}},
		{Title: "Filler C", FilePath: "filler-c.toml", Keywords: []string{"deploy"}},
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
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "xyzsupplog rule",
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

	// 2 matching memories, 1 suppressed by cross-ref → rate = 1/2 = 0.5
	memories := []*memory.Stored{
		{
			Title:    "Covered",
			FilePath: "covered.toml",
			Keywords: []string{"xyzstatscheck"},
			Content:  "xyzstatscheck covered rule",
		},
		{
			Title:    "Surfaced",
			FilePath: "surfaced.toml",
			Keywords: []string{"xyzstatscheck"},
			Content:  "xyzstatscheck surfaced rule",
		},
		{Title: "Filler A", FilePath: "filler-a.toml", Keywords: []string{"logging"}},
		{Title: "Filler B", FilePath: "filler-b.toml", Keywords: []string{"testing"}},
		{Title: "Filler C", FilePath: "filler-c.toml", Keywords: []string{"deploy"}},
	}

	checker := &fakeCrossRefChecker{
		covered: map[string]string{"covered.toml": "CLAUDE.md"},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever, surface.WithCrossRefChecker(checker))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "xyzstatscheck rule",
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
			Title:    "Rule A",
			FilePath: "rule-a.toml",
			Keywords: []string{"targ", "build"},
			Content:  "xyznosuppress targ build rule",
		},
		{Title: "Filler A", FilePath: "filler-a.toml", Keywords: []string{"logging"}},
		{Title: "Filler B", FilePath: "filler-b.toml", Keywords: []string{"testing"}},
		{Title: "Filler C", FilePath: "filler-c.toml", Keywords: []string{"deploy"}},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:             surface.ModePrompt,
		DataDir:          "/tmp/data",
		Message:          "xyznosuppress targ build",
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
