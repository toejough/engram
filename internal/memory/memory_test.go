package memory_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
)

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

func TestStored_SearchText(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		Situation: "when running tests",
		Content: memory.ContentFields{
			Behavior: "use go test directly",
			Impact:   "misses coverage flags",
			Action:   "use targ test instead",
		},
	}

	text := mem.SearchText()
	g.Expect(text).To(ContainSubstring("when running tests"))
	g.Expect(text).To(ContainSubstring("use go test directly"))
	g.Expect(text).To(ContainSubstring("misses coverage flags"))
	g.Expect(text).To(ContainSubstring("use targ test instead"))
}

func TestStored_SearchText_EmptyFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{}
	g.Expect(mem.SearchText()).To(Equal(""))
}

func TestStored_SearchText_FactType(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		Type:      "fact",
		Situation: "Go project",
		Content: memory.ContentFields{
			Subject:   "this project",
			Predicate: "uses",
			Object:    "targ build system",
		},
	}

	text := mem.SearchText()
	g.Expect(text).To(ContainSubstring("Go project"))
	g.Expect(text).To(ContainSubstring("this project"))
	g.Expect(text).To(ContainSubstring("uses"))
	g.Expect(text).To(ContainSubstring("targ build system"))
}

func TestStored_SearchText_PartialFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		Situation: "when deploying",
		Content: memory.ContentFields{
			Action: "run smoke tests",
		},
	}

	text := mem.SearchText()
	g.Expect(text).To(ContainSubstring("when deploying"))
	g.Expect(text).To(ContainSubstring("run smoke tests"))
	g.Expect(text).NotTo(ContainSubstring("  ")) // no double spaces from empty fields
}

func TestStored_TotalEvaluations(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		FollowedCount:    5,
		NotFollowedCount: 2,
		IrrelevantCount:  1,
	}

	g.Expect(mem.TotalEvaluations()).To(Equal(8))
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
