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
		Situation:        "when running tests",
		Behavior:         "use go test directly",
		Impact:           "misses coverage and flags",
		Action:           "use targ test instead",
		ProjectScoped:    true,
		ProjectSlug:      "engram",
		CreatedAt:        "2026-01-01T00:00:00Z",
		UpdatedAt:        "2026-01-02T00:00:00Z",
		SurfacedCount:    5,
		FollowedCount:    3,
		NotFollowedCount: 1,
		IrrelevantCount:  4,
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

func TestMemoryRecord_RoundTrip_PendingEvaluations(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	original := memory.MemoryRecord{
		Situation: "when committing",
		Behavior:  "skip hooks",
		Impact:    "broken builds",
		Action:    "always run hooks",
		PendingEvaluations: []memory.PendingEvaluation{{
			SurfacedAt:  "2026-01-02T00:00:00Z",
			UserPrompt:  "commit this change",
			SessionID:   "sess-123",
			ProjectSlug: "engram",
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

func TestMemoryRecord_TotalEvaluations(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	record := memory.MemoryRecord{
		FollowedCount:    3,
		NotFollowedCount: 2,
		IrrelevantCount:  1,
	}

	g.Expect(record.TotalEvaluations()).To(Equal(6))
}
