package surface_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

// T-33: Pre-filter matches memory keywords in tool input
func TestT33_PreFilterMatchesKeywordsInToolInput(t *testing.T) {
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
	enforcer := &fakeEnforcer{violated: true}
	s := surface.New(retriever, enforcer, nil)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: `git commit -m 'fix'`,
		Token:     "test-token",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Keyword "commit" matched → enforcer was called → violation detected.
	g.Expect(enforcer.called).To(BeTrue())
	g.Expect(buf.String()).To(ContainSubstring(`"decision": "block"`))
}

// T-34: Pre-filter skips memories without anti_pattern
func TestT34_PreFilterSkipsWithoutAntiPattern(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:    "Has Keywords But No AntiPattern",
			FilePath: "no-anti.toml",
			Keywords: []string{"commit"},
			// AntiPattern is empty — should not be a candidate.
		},
	}

	retriever := &fakeRetriever{memories: memories}
	enforcer := &fakeEnforcer{violated: true}
	s := surface.New(retriever, enforcer, nil)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: "git commit -m 'fix'",
		Token:     "test-token",
	})

	g.Expect(err).NotTo(HaveOccurred())
	// Enforcer should NOT have been called — no candidates.
	g.Expect(enforcer.called).To(BeFalse())
	g.Expect(buf.String()).To(BeEmpty())
}

// T-35: Pre-filter returns empty when no keywords match
func TestT35_PreFilterNoKeywordMatch(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:       "Commit Rules",
			FilePath:    "commit-rules.toml",
			AntiPattern: "manual git commit",
			Keywords:    []string{"commit", "git"},
			Principle:   "use /commit",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	enforcer := &fakeEnforcer{violated: true}
	s := surface.New(retriever, enforcer, nil)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Read",
		ToolInput: "/path/to/file.go",
		Token:     "test-token",
	})

	g.Expect(err).NotTo(HaveOccurred())
	// No keyword match → enforcer not called.
	g.Expect(enforcer.called).To(BeFalse())
	g.Expect(buf.String()).To(BeEmpty())
}

// T-36: Violated anti-pattern returns violated=true
func TestT36_ViolatedAntiPatternBlocks(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:       "Use /commit",
			FilePath:    "use-commit.toml",
			AntiPattern: "manual git commit",
			Keywords:    []string{"commit"},
			Principle:   "use /commit for commits",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	enforcer := &fakeEnforcer{violated: true}
	s := surface.New(retriever, enforcer, nil)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: `git commit -m 'fix'`,
		Token:     "test-token",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	g.Expect(output).To(ContainSubstring(`"decision": "block"`))
	g.Expect(output).To(ContainSubstring("Use /commit"))
	g.Expect(output).To(ContainSubstring("use /commit for commits"))
	g.Expect(output).To(ContainSubstring("use-commit.toml"))
}

// T-37: Non-violated anti-pattern returns violated=false
func TestT37_NonViolatedAllowsSilently(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:       "Use /commit",
			FilePath:    "use-commit.toml",
			AntiPattern: "manual git commit",
			Keywords:    []string{"commit"},
			Principle:   "use /commit for commits",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	enforcer := &fakeEnforcer{violated: false}
	s := surface.New(retriever, enforcer, nil)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: `git commit -m 'fix'`,
		Token:     "test-token",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf.String()).To(BeEmpty()) // allowed silently
}

// T-38: Missing token returns error (not violated)
func TestT38_MissingTokenGracefulDegradation(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:       "Use /commit",
			FilePath:    "use-commit.toml",
			AntiPattern: "manual git commit",
			Keywords:    []string{"commit"},
			Principle:   "use /commit",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	enforcer := &fakeEnforcer{err: errors.New("no API token")}

	var stderr bytes.Buffer

	s := surface.New(retriever, enforcer, &stderr)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: "git commit -m 'fix'",
		Token:     "",
	})

	g.Expect(err).NotTo(HaveOccurred())

	// No block — graceful degradation.
	g.Expect(buf.String()).To(BeEmpty())
	// Warning emitted to stderr.
	g.Expect(stderr.String()).To(ContainSubstring("[engram] Warning: enforcement skipped"))
}

// T-39: LLM timeout returns error (not violated)
func TestT39_LLMTimeoutGracefulDegradation(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:       "Use /commit",
			FilePath:    "use-commit.toml",
			AntiPattern: "manual git commit",
			Keywords:    []string{"commit"},
			Principle:   "use /commit",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	enforcer := &fakeEnforcer{err: errors.New("timeout")}

	var stderr bytes.Buffer

	s := surface.New(retriever, enforcer, &stderr)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: "git commit -m 'fix'",
		Token:     "test-token",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf.String()).To(BeEmpty())
	g.Expect(stderr.String()).To(ContainSubstring("[engram] Warning: enforcement skipped"))
}

// fakeEnforcer is a test double for surface.ToolEnforcer.
type fakeEnforcer struct {
	violated bool
	err      error
	called   bool
}

func (f *fakeEnforcer) JudgeViolation(
	_ context.Context, _, _ string, _ *memory.Stored, _ string,
) (bool, error) {
	f.called = true

	if f.err != nil {
		return false, f.err
	}

	return f.violated, nil
}
