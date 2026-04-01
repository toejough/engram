package memory_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
)

func TestStored_SearchText(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		Situation: "when running tests",
		Behavior:  "use go test directly",
		Impact:    "misses coverage flags",
		Action:    "use targ test instead",
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

func TestStored_SearchText_PartialFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		Situation: "when deploying",
		Action:    "run smoke tests",
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
