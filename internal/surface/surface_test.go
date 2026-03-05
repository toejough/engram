package surface_test

import (
	"bytes"
	"context"
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
	s := surface.New(retriever, nil, nil, nil)

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
	s := surface.New(retriever, nil, nil, nil)

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
	s := surface.New(retriever, nil, nil, nil)

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
	s := surface.New(retriever, nil, nil, nil)

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
	s := surface.New(retriever, nil, nil, nil)

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
	s := surface.New(retriever, nil, nil, nil)

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

// TestUnknownModeReturnsError verifies that Run returns ErrUnknownMode for unrecognized modes.
func TestUnknownModeReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	s := surface.New(retriever, nil, nil, nil)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    "unknown-mode",
		DataDir: "/tmp/data",
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(surface.ErrUnknownMode))
}

// fakeRetriever is a test double for surface.MemoryRetriever.
type fakeRetriever struct {
	memories []*memory.Stored
	err      error
}

func (f *fakeRetriever) ListMemories(_ context.Context, _ string) ([]*memory.Stored, error) {
	return f.memories, f.err
}

func memPath(i int) string {
	return "memory-" + string(rune('a'+i%26)) + ".toml"
}

// --- Helpers ---

func memTitle(i int) string {
	return "Memory " + string(rune('A'+i%26))
}
