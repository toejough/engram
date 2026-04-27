package memory_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
)

func TestBuildIndex_EmptyList_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(memory.BuildIndex(nil)).To(BeEmpty())
}

func TestBuildIndex_FormatsCorrectly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Type:      "feedback",
			Situation: "when running tests",
			FilePath:  "/data/memory/feedback/use-targ.toml",
		},
		{
			Type:      "fact",
			Situation: "Go projects",
			FilePath:  "/data/memory/facts/engram-uses-go.toml",
		},
	}

	result := memory.BuildIndex(memories)
	g.Expect(result).To(ContainSubstring("feedback | use-targ | when running tests"))
	g.Expect(result).To(ContainSubstring("fact | engram-uses-go | Go projects"))
}

func TestFactsDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(memory.FactsDir("/data")).To(Equal("/data/memory/facts"))
}

func TestFeedbackDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(memory.FeedbackDir("/data")).To(Equal("/data/memory/feedback"))
}

func TestResolveMemoryPath_FactsSecond(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	existingFiles := map[string]bool{
		"/data/memory/facts/my-fact.toml": true,
	}
	exists := func(path string) bool { return existingFiles[path] }

	result := memory.ResolveMemoryPath("/data", "my-fact", exists)
	g.Expect(result).To(Equal("/data/memory/facts/my-fact.toml"))
}

func TestResolveMemoryPath_FeedbackFirst(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	existingFiles := map[string]bool{
		"/data/memory/feedback/my-mem.toml": true,
		"/data/memories/my-mem.toml":        true,
	}
	exists := func(path string) bool { return existingFiles[path] }

	result := memory.ResolveMemoryPath("/data", "my-mem", exists)
	g.Expect(result).To(Equal("/data/memory/feedback/my-mem.toml"))
}

func TestResolveMemoryPath_LegacyFallback(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	existingFiles := map[string]bool{
		"/data/memories/old-mem.toml": true,
	}
	exists := func(path string) bool { return existingFiles[path] }

	result := memory.ResolveMemoryPath("/data", "old-mem", exists)
	g.Expect(result).To(Equal("/data/memories/old-mem.toml"))
}

func TestResolveMemoryPath_NoneExist_ReturnsLegacy(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	exists := func(_ string) bool { return false }

	result := memory.ResolveMemoryPath("/data", "missing", exists)
	g.Expect(result).To(Equal("/data/memories/missing.toml"))
}

func TestToStored_MalformedUpdatedAt_LogsWarning(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	record := &memory.MemoryRecord{
		Situation: "test",
		UpdatedAt: "not-a-date",
	}

	stored := record.ToStored("memories/test.toml")
	g.Expect(stored.Situation).To(Equal("test"))
	g.Expect(stored.UpdatedAt.IsZero()).To(BeTrue())
}

func TestToStored_PreservesSource(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	record := &memory.MemoryRecord{
		Type:      "feedback",
		Situation: "test",
		Source:    "user",
	}

	stored := record.ToStored("memories/test.toml")
	g.Expect(stored.Source).To(Equal("user"))
}

func TestToStored_ValidUpdatedAt_Parses(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	record := &memory.MemoryRecord{
		Situation: "test",
		UpdatedAt: "2024-01-15T10:30:00Z",
	}

	stored := record.ToStored("memories/test.toml")
	g.Expect(stored.UpdatedAt.IsZero()).To(BeFalse())
	g.Expect(stored.UpdatedAt.Year()).To(Equal(2024))
}
