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

// TestRepeatToConfirm_DifferentCallDoesNotOverride verifies that a different
// tool call after a block does NOT get the repeat-to-confirm pass.
func TestRepeatToConfirm_DifferentCallDoesNotOverride(t *testing.T) {
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
	blockStore := &fakeBlockHashStore{}
	s := surface.New(retriever, enforcer, blockStore, nil)

	// First call: blocks.
	var buf1 bytes.Buffer

	err := s.Run(context.Background(), &buf1, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: `git commit -m 'fix'`,
		Token:     "test-token",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf1.String()).To(ContainSubstring(`"decision": "block"`))

	// Second call with DIFFERENT input: should still block.
	var buf2 bytes.Buffer

	err = s.Run(context.Background(), &buf2, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: `git commit -m 'different message'`,
		Token:     "test-token",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf2.String()).To(ContainSubstring(`"decision": "block"`))
}

// TestRepeatToConfirm_FirstCallBlocks verifies that a violated tool call
// blocks and saves the hash, then a repeated identical call is allowed.
func TestRepeatToConfirm_FirstCallBlocks(t *testing.T) {
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
	blockStore := &fakeBlockHashStore{}
	s := surface.New(retriever, enforcer, blockStore, nil)

	opts := surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: `git commit -m 'fix'`,
		Token:     "test-token",
	}

	// First call: should block and save hash.
	var buf1 bytes.Buffer

	err := s.Run(context.Background(), &buf1, opts)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf1.String()).To(ContainSubstring(`"decision": "block"`))
	g.Expect(buf1.String()).To(ContainSubstring("Repeat to override."))
	g.Expect(blockStore.saved).NotTo(BeEmpty())

	// Second call (identical): should be allowed via repeat-to-confirm.
	var buf2 bytes.Buffer

	err = s.Run(context.Background(), &buf2, opts)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf2.String()).To(BeEmpty())
	g.Expect(blockStore.cleared).To(BeTrue())
}

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
	s := surface.New(retriever, enforcer, nil, nil)

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
	s := surface.New(retriever, enforcer, nil, nil)

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
	s := surface.New(retriever, enforcer, nil, nil)

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
	s := surface.New(retriever, enforcer, nil, nil)

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
	s := surface.New(retriever, enforcer, nil, nil)

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

	s := surface.New(retriever, enforcer, nil, &stderr)

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

	s := surface.New(retriever, enforcer, nil, &stderr)

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

// TestToolInputKeywordStillMatches verifies that keywords matching in the
// tool input continue to trigger enforcement even after removing tool-name matching.
func TestToolInputKeywordStillMatches(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:       "Prefer Fish",
			FilePath:    "prefer-fish.toml",
			AntiPattern: "writing bash scripts",
			Keywords:    []string{"bash"},
			Principle:   "use fish instead of bash",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	enforcer := &fakeEnforcer{violated: true}
	s := surface.New(retriever, enforcer, nil, nil)

	var buf bytes.Buffer

	// Keyword "bash" appears in tool input "cat script.bash".
	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: `cat script.bash`,
		Token:     "test-token",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(enforcer.called).To(BeTrue())
	g.Expect(buf.String()).To(ContainSubstring(`"decision": "block"`))
}

// TestToolNameOnlyKeywordDoesNotMatch verifies that keywords matching only the
// tool name (not toolInput) do not trigger enforcement. Tool-level blocking
// should be done via Claude Code config, not memory keywords.
func TestToolNameOnlyKeywordDoesNotMatch(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:       "Prefer Fish",
			FilePath:    "prefer-fish.toml",
			AntiPattern: "writing bash scripts",
			Keywords:    []string{"bash"},
			Principle:   "use fish instead of bash",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	enforcer := &fakeEnforcer{violated: true}
	s := surface.New(retriever, enforcer, nil, nil)

	var buf bytes.Buffer

	// Keyword "bash" matches tool name "Bash" but NOT the tool input.
	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: `ls -la /tmp`,
		Token:     "test-token",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(enforcer.called).To(BeFalse())
	g.Expect(buf.String()).To(BeEmpty())
}

// fakeBlockHashStore is a test double for surface.BlockHashStore.
type fakeBlockHashStore struct {
	saved   string
	cleared bool
}

func (f *fakeBlockHashStore) ClearHash(_ context.Context) error {
	f.saved = ""
	f.cleared = true

	return nil
}

func (f *fakeBlockHashStore) LastHash(_ context.Context) (string, error) {
	if f.saved == "" {
		return "", nil
	}

	return f.saved, nil
}

func (f *fakeBlockHashStore) SaveHash(_ context.Context, hash string) error {
	f.saved = hash
	f.cleared = false

	return nil
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
