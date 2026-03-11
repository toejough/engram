package surface_test

// P4-early: Budget quick wins — effectiveness gating + BM25 floor (issue #88)
// REQ-P4e-1: SessionStart ranks by effectiveness, gates on >40% or <5 surfacings
// REQ-P4e-2: SessionStart top-7 limit, 600 token default budget
// REQ-P4e-3: UserPromptSubmit 250 token default budget
// REQ-P4e-4: PreToolUse top-2 limit, effectiveness floor 40%, 150 token default budget
// REQ-P4e-5: InvocationTokenLogger called after each surface invocation

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

// T-P4e-1: SessionStart limits to top 7 (down from 10).
func TestTP4e1_SessionStartLimitsToTop7(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// 12 memories all with <5 surfacings (insufficient data) → all pass gating.
	eff := &fakeEffectivenessComputer{
		stats: map[string]surface.EffectivenessStat{},
	}

	memories := make([]*memory.Stored, 12)

	for i := range 12 {
		path := memPath(i)
		memories[i] = &memory.Stored{
			Title:     memTitle(i),
			FilePath:  path,
			UpdatedAt: time.Date(2025, 1, i+1, 0, 0, 0, 0, time.UTC),
		}
		eff.stats[path] = surface.EffectivenessStat{SurfacedCount: 2, EffectivenessScore: 80.0}
	}

	retriever := &fakeRetriever{memories: memories}
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

	output := buf.String()
	count := strings.Count(output, "  - memory-")
	g.Expect(count).To(Equal(7), "expected 7 memories (top-7 limit), got %d", count)
}

// T-P4e-2: SessionStart gates out memories with >=5 surfacings AND effectiveness <= 40%.
func TestTP4e2_SessionStartEffectivenessGating(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{Title: "High Eff", FilePath: "high-eff.toml", UpdatedAt: time.Now()},
		{Title: "Low Eff", FilePath: "low-eff.toml", UpdatedAt: time.Now()},
	}

	eff := &fakeEffectivenessComputer{
		stats: map[string]surface.EffectivenessStat{
			"high-eff.toml": {SurfacedCount: 10, EffectivenessScore: 75.0},
			"low-eff.toml":  {SurfacedCount: 10, EffectivenessScore: 20.0}, // below 40%
		},
	}

	retriever := &fakeRetriever{memories: memories}
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

	output := buf.String()
	g.Expect(output).To(ContainSubstring("high-eff"))
	g.Expect(output).NotTo(ContainSubstring("low-eff"), "low effectiveness memory should be gated out")
}

// T-P4e-3: SessionStart includes memories with <5 surfacings regardless of effectiveness.
func TestTP4e3_SessionStartInsufficientDataAlwaysIncluded(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{Title: "New Memory", FilePath: "new-mem.toml", UpdatedAt: time.Now()},
		{Title: "No Data", FilePath: "no-data.toml", UpdatedAt: time.Now()},
	}

	eff := &fakeEffectivenessComputer{
		stats: map[string]surface.EffectivenessStat{
			// 3 surfacings (< 5) but 0% effectiveness → still included (insufficient data).
			"new-mem.toml": {SurfacedCount: 3, EffectivenessScore: 0.0},
		},
	}

	retriever := &fakeRetriever{memories: memories}
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

	output := buf.String()
	g.Expect(output).To(ContainSubstring("new-mem"), "insufficient-data memory should be included")
	g.Expect(output).To(ContainSubstring("no-data"), "memory with no effectiveness data should be included")
}

// T-P4e-4: SessionStart ranks by effectiveness descending (high-eff appears before low-eff).
func TestTP4e4_SessionStartRanksByEffectiveness(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// low-eff listed first in retriever order to prove sorting overrides list order.
	memories := []*memory.Stored{
		{Title: "Low Scorer", FilePath: "low-scorer.toml", UpdatedAt: time.Now()},
		{Title: "High Scorer", FilePath: "high-scorer.toml", UpdatedAt: time.Now()},
	}

	eff := &fakeEffectivenessComputer{
		stats: map[string]surface.EffectivenessStat{
			"low-scorer.toml":  {SurfacedCount: 10, EffectivenessScore: 50.0},
			"high-scorer.toml": {SurfacedCount: 10, EffectivenessScore: 90.0},
		},
	}

	retriever := &fakeRetriever{memories: memories}
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

	output := buf.String()
	highIdx := strings.Index(output, "high-scorer")
	lowIdx := strings.Index(output, "low-scorer")
	g.Expect(highIdx).To(BeNumerically("<", lowIdx), "high-eff memory should appear before low-eff memory")
}

// T-P4e-5: Default budgets are 600/250/150 for SessionStart/UserPromptSubmit/PreToolUse.
func TestTP4e5_DefaultBudgetsUpdated(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	g.Expect(surface.DefaultSessionStartBudget).To(Equal(600))
	g.Expect(surface.DefaultUserPromptSubmitBudget).To(Equal(250))
	g.Expect(surface.DefaultPreToolUseBudget).To(Equal(150))
}

// T-P4e-6: PreToolUse limits to top 2 (down from 3).
func TestTP4e6_PreToolUseLimitsToTop2(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// 5 anti-pattern memories matching "commit" + 8 fillers for IDF contrast.
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

	fillerNames := []string{"logging", "testing", "deploy", "config", "monitoring", "caching", "auth", "docs"}

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
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	count := strings.Count(output, "  - mem-") + strings.Count(output, "  - memory-")
	g.Expect(count).To(Equal(2), "expected 2 memories (top-2 limit), got %d", count)
}

// T-P4e-7: PreToolUse gates out memories with >=5 surfacings AND effectiveness <= 40%.
func TestTP4e7_PreToolUseEffectivenessGating(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:       "high eff commit",
			FilePath:    "high-eff-commit.toml",
			AntiPattern: "manual commit violation",
			Keywords:    []string{"commit", "git"},
			Principle:   "use /commit",
		},
		{
			Title:       "low eff commit",
			FilePath:    "low-eff-commit.toml",
			AntiPattern: "commit without review",
			Keywords:    []string{"commit", "git"},
			Principle:   "review first",
		},
		{Title: "logging", FilePath: "log.toml", AntiPattern: "no logging", Keywords: []string{"logging"}, Principle: "log"},
		{Title: "testing", FilePath: "test.toml", AntiPattern: "no tests", Keywords: []string{"testing"}, Principle: "test"},
		{
			Title:       "deploy",
			FilePath:    "deploy.toml",
			AntiPattern: "manual deploy",
			Keywords:    []string{"deploy"},
			Principle:   "ci",
		},
		{
			Title:       "config",
			FilePath:    "config.toml",
			AntiPattern: "hardcode",
			Keywords:    []string{"config"},
			Principle:   "env",
		},
		{
			Title:       "monitor",
			FilePath:    "monitor.toml",
			AntiPattern: "no metrics",
			Keywords:    []string{"monitoring"},
			Principle:   "metrics",
		},
	}

	eff := &fakeEffectivenessComputer{
		stats: map[string]surface.EffectivenessStat{
			"high-eff-commit.toml": {SurfacedCount: 10, EffectivenessScore: 75.0},
			"low-eff-commit.toml":  {SurfacedCount: 10, EffectivenessScore: 15.0},
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever, surface.WithEffectiveness(eff))

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
	g.Expect(output).To(ContainSubstring("high-eff-commit"))
	g.Expect(output).NotTo(ContainSubstring("low-eff-commit"), "low effectiveness memory should be gated out")
}

// T-P4e-8: InvocationTokenLogger is called with output token count after surface.
func TestTP4e8_InvocationTokenLoggerCalled(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:     "Alpha Memory",
			FilePath:  "alpha.toml",
			UpdatedAt: time.Now(),
		},
	}

	retriever := &fakeRetriever{memories: memories}
	tokenLogger := &fakeInvocationTokenLogger{}
	s := surface.New(retriever, surface.WithInvocationTokenLogger(tokenLogger))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(tokenLogger.calls).To(HaveLen(1))
	g.Expect(tokenLogger.calls[0].mode).To(Equal(surface.ModeSessionStart))
	g.Expect(tokenLogger.calls[0].tokenCount).To(BeNumerically(">", 0))
}

// fakeInvocationTokenLogger captures LogInvocationTokens calls.
type fakeInvocationTokenLogger struct {
	calls []invocationTokenCall
}

func (f *fakeInvocationTokenLogger) LogInvocationTokens(mode string, tokenCount int, _ time.Time) error {
	f.calls = append(f.calls, invocationTokenCall{mode: mode, tokenCount: tokenCount})

	return nil
}

type invocationTokenCall struct {
	mode       string
	tokenCount int
}
