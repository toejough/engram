package memory_test

import (
	"bytes"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/memory"
)

func TestMemoryRecord_RoundTrip(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	original := memory.MemoryRecord{
		Title:             "test title",
		Content:           "test content",
		ObservationType:   "workflow_instruction",
		Concepts:          []string{"a", "b"},
		Keywords:          []string{"k1", "k2"},
		Principle:         "test principle",
		AntiPattern:       "test anti-pattern",
		Rationale:         "test rationale",
		Confidence:        "A",
		CreatedAt:         "2026-01-01T00:00:00Z",
		UpdatedAt:         "2026-01-02T00:00:00Z",
		SurfacedCount:     5,
		FollowedCount:     3,
		ContradictedCount: 1,
		IgnoredCount:      2,
		IrrelevantCount:   4,
		LastSurfacedAt:    "2026-01-03T00:00:00Z",
	}

	var buf bytes.Buffer

	err := toml.NewEncoder(&buf).Encode(original)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var decoded memory.MemoryRecord

	_, err = toml.Decode(buf.String(), &decoded)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(decoded).To(Equal(original))
}

func TestMemoryRecord_RoundTrip_RegistryFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	original := memory.MemoryRecord{
		Title:            "test",
		Content:          "content",
		SourceType:       "memory",
		SourcePath:       "/path/to/source",
		ContentHash:      "abc123",
		EnforcementLevel: "advisory",
		Transitions: []memory.TransitionRecord{{
			From: "advisory", To: "reminder", At: "2026-01-01T00:00:00Z", Reason: "test",
		}},
		Absorbed: []memory.AbsorbedRecord{{
			From: "old.toml", SurfacedCount: 5, ContentHash: "def456", MergedAt: "2026-01-02T00:00:00Z",
			Evaluations: memory.EvaluationCounters{Followed: 2, Contradicted: 1, Ignored: 0},
		}},
	}

	var buf bytes.Buffer

	err := toml.NewEncoder(&buf).Encode(original)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var decoded memory.MemoryRecord

	_, err = toml.Decode(buf.String(), &decoded)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(decoded).To(Equal(original))
}
