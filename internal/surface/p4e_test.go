package surface_test

// P4-early: Budget quick wins — effectiveness gating + BM25 floor (issue #88)
// REQ-P4e-1: SessionStart ranks by effectiveness, gates on >40% (no-data defaults to 50%)
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
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	count := strings.Count(output, "  - mem-") + strings.Count(output, "  - memory-")
	g.Expect(count).To(Equal(2), "expected 2 memories (top-2 limit), got %d", count)
}

// T-P4e-7: PreToolUse gates out memories with effectiveness <= 40%.
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
		{
			Title:       "logging",
			FilePath:    "log.toml",
			AntiPattern: "no logging",
			Keywords:    []string{"logging"},
			Principle:   "log",
		},
		{
			Title:       "testing",
			FilePath:    "test.toml",
			AntiPattern: "no tests",
			Keywords:    []string{"testing"},
			Principle:   "test",
		},
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
	g.Expect(output).
		NotTo(ContainSubstring("low-eff-commit"), "low effectiveness memory should be gated out")
}

// T-P4e-8: InvocationTokenLogger is called with output token count after surface.
func TestTP4e8_InvocationTokenLoggerCalled(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:     "Alpha Memory",
			FilePath:  "alpha.toml",
			Keywords:  []string{"alphatoken"},
			Principle: "always check alphatoken",
			UpdatedAt: time.Now(),
		},
		{
			Title:    "Filler B",
			FilePath: "filler-b.toml",
			Keywords: []string{"unrelated"},
		},
		{
			Title:    "Filler C",
			FilePath: "filler-c.toml",
			Keywords: []string{"other"},
		},
	}

	retriever := &fakeRetriever{memories: memories}
	tokenLogger := &fakeInvocationTokenLogger{}
	s := surface.New(retriever, surface.WithInvocationTokenLogger(tokenLogger))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "alphatoken check",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(tokenLogger.calls).To(HaveLen(1))
	g.Expect(tokenLogger.calls[0].mode).To(Equal(surface.ModePrompt))
	g.Expect(tokenLogger.calls[0].tokenCount).To(BeNumerically(">", 0))
}

// fakeInvocationTokenLogger captures LogInvocationTokens calls.
type fakeInvocationTokenLogger struct {
	calls []invocationTokenCall
}

func (f *fakeInvocationTokenLogger) LogInvocationTokens(
	mode string,
	tokenCount int,
	_ time.Time,
) error {
	f.calls = append(f.calls, invocationTokenCall{mode: mode, tokenCount: tokenCount})

	return nil
}

type invocationTokenCall struct {
	mode       string
	tokenCount int
}
