package surface_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

// T-27: SessionStart surfaces top 20 by recency
func TestT27_SessionStartSurfacesTop20(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Create 25 memories in descending order (retriever contract: sorted by UpdatedAt desc).
	memories := make([]*memory.Stored, 0, 25)
	for i := 24; i >= 0; i-- {
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
	g.Expect(output).To(ContainSubstring("[engram] Loaded 20 memories."))

	// Most recent (index 24) should appear, oldest (index 0-4) should not.
	g.Expect(output).To(ContainSubstring(memTitle(24)))
	g.Expect(output).To(ContainSubstring(memTitle(5)))
	g.Expect(output).NotTo(ContainSubstring(memTitle(4)))
	g.Expect(output).NotTo(ContainSubstring(memTitle(0)))
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
	g.Expect(output).To(ContainSubstring("First"))
	g.Expect(output).To(ContainSubstring("Second"))
	g.Expect(output).To(ContainSubstring("Third"))
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
	g.Expect(output).To(ContainSubstring("Commit Conventions"))
	g.Expect(output).To(ContainSubstring("commit"))
	g.Expect(output).NotTo(ContainSubstring("Build Tools"))
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

// T-32: Keyword matching is case-insensitive and whole-word
func TestT32_KeywordMatchingCaseInsensitiveWholeWord(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:    "Commit Rules",
			FilePath: "commit-rules.toml",
			Keywords: []string{"commit"},
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
	g.Expect(buf1.String()).To(ContainSubstring("Commit Rules"))

	// Whole-word: "recommit" should NOT match keyword "commit".
	var buf2 bytes.Buffer

	err = s.Run(context.Background(), &buf2, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "recommit the file",
	})

	g.Expect(err).NotTo(HaveOccurred())
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
	// Memory should appear because keyword "commit" matched in tool input.
	g.Expect(output).To(ContainSubstring("Manual git commit"))
	g.Expect(output).To(ContainSubstring("use /commit skill instead"))
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
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	// Tool input contains keyword "commit" → both memories should not match
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
	g.Expect(output).To(ContainSubstring("Use /commit"))
	g.Expect(output).To(ContainSubstring("always use /commit for commits"))
	g.Expect(output).To(ContainSubstring("use-commit.toml"))
	// "Use targ test" should NOT appear — keyword "test" is not in "git commit -m 'fix'".
	g.Expect(output).NotTo(ContainSubstring("Use targ test"))
	g.Expect(output).To(ContainSubstring("</system-reminder>"))
}

// T-45: Tool mode with no matching memories produces empty output
func TestT45_ToolModeNoMatchProducesEmpty(t *testing.T) {
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
	g.Expect(result.Summary).To(ContainSubstring("\"First\" (first.toml)"))
	g.Expect(result.Context).To(ContainSubstring("<system-reminder"))
	g.Expect(result.Context).To(ContainSubstring("First"))
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
	g.Expect(result.Summary).
		To(ContainSubstring("\"Commit Conventions\" (commit-conventions.toml)"))
	g.Expect(result.Context).To(ContainSubstring("Commit Conventions"))
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
	g.Expect(result.Summary).
		To(ContainSubstring("\"Use /commit\" — always use /commit for commits (use-commit.toml)"))
	g.Expect(result.Context).To(ContainSubstring("Use /commit"))
	g.Expect(result.Context).To(ContainSubstring("always use /commit for commits"))
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
	g.Expect(buf.String()).To(ContainSubstring("First"))
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
	g.Expect(output).To(ContainSubstring("Commit Conventions"))
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
	g.Expect(result.Summary).To(ContainSubstring("\"Alpha\" (alpha.toml)"))
	g.Expect(result.Context).To(ContainSubstring("Created 2 memories since last session:"))
	g.Expect(result.Context).To(ContainSubstring("\"New Memory One\" [A] (new-memory-one.toml)"))
	g.Expect(result.Context).To(ContainSubstring("\"New Memory Two\" [B] (new-memory-two.toml)"))
	g.Expect(result.Context).To(ContainSubstring("[engram] Loaded 3 memories."))
	g.Expect(result.Context).To(ContainSubstring("Alpha"))
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
	g.Expect(result.Summary).To(ContainSubstring("\"Alpha\" (alpha.toml)"))
	g.Expect(result.Summary).To(ContainSubstring("\"Beta\" (beta.toml)"))
	g.Expect(result.Summary).To(ContainSubstring("\"Gamma\" (gamma.toml)"))
	g.Expect(result.Context).NotTo(ContainSubstring("Created"))
	g.Expect(result.Context).To(ContainSubstring("Alpha"))
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

// fakeRetriever is a test double for surface.MemoryRetriever.
type fakeRetriever struct {
	memories []*memory.Stored
	err      error
}

func (f *fakeRetriever) ListMemories(_ context.Context, _ string) ([]*memory.Stored, error) {
	return f.memories, f.err
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

type trackerCall struct {
	memories []*memory.Stored
	mode     string
}

func memPath(i int) string {
	return "memory-" + string(rune('a'+i%26)) + ".toml"
}

// --- Helpers ---

func memTitle(i int) string {
	return "Memory " + string(rune('A'+i%26))
}
