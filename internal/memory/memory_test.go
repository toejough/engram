package memory_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
)

func TestStored_SearchText(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		Title:     "Test Title",
		Content:   "some content",
		Principle: "a principle",
		Keywords:  []string{"kw1", "kw2"},
		Concepts:  []string{"concept1"},
	}

	text := mem.SearchText()
	g.Expect(text).To(ContainSubstring("Test Title"))
	g.Expect(text).To(ContainSubstring("some content"))
	g.Expect(text).To(ContainSubstring("a principle"))
	g.Expect(text).To(ContainSubstring("kw1"))
	g.Expect(text).To(ContainSubstring("kw2"))
	g.Expect(text).To(ContainSubstring("concept1"))
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
		Title:   "Only Title",
		Content: "Only Content",
	}

	text := mem.SearchText()
	g.Expect(text).To(ContainSubstring("Only Title"))
	g.Expect(text).To(ContainSubstring("Only Content"))
	g.Expect(text).NotTo(ContainSubstring("  ")) // no double spaces from empty fields
}

func TestClassifiedMemory_ToEnriched(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Now()
	classified := &memory.ClassifiedMemory{
		Tier:            "A",
		Title:           "Use Targ",
		Content:         "remember to use targ",
		ObservationType: "reminder",
		Concepts:        []string{"build"},
		Keywords:        []string{"targ"},
		Principle:       "Use targ for builds",
		AntiPattern:     "Running go test",
		Rationale:       "Convention",
		FilenameSummary: "use targ",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	enriched := classified.ToEnriched()

	g.Expect(enriched).NotTo(BeNil())

	if enriched == nil {
		return
	}

	g.Expect(enriched.Title).To(Equal(classified.Title))
	g.Expect(enriched.Content).To(Equal(classified.Content))
	g.Expect(enriched.Confidence).To(Equal("A"))
	g.Expect(enriched.Keywords).To(Equal(classified.Keywords))
	g.Expect(enriched.AntiPattern).To(Equal(classified.AntiPattern))
	g.Expect(enriched.CreatedAt).To(Equal(now))
}
