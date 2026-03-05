package memory_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
)

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
