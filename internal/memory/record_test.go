package memory_test

import (
	"bytes"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/memory"
)

func TestMemoryRecord_PendingEvaluations_Cleanup(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	t.Run("omits pending_evaluations when empty", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)

		record := memory.MemoryRecord{
			Situation: "test",
			Content: memory.ContentFields{
				Behavior: "test",
				Impact:   "test",
				Action:   "test",
			},
		}

		var buf bytes.Buffer

		err := toml.NewEncoder(&buf).Encode(record)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(buf.String()).NotTo(ContainSubstring("pending_evaluations"))
	})

	t.Run("drops pending_evaluations on re-encode without them", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)

		withPending := `situation = "test"

[content]
behavior = "test"
impact = "test"
action = "test"

[[pending_evaluations]]
surfaced_at = "2026-01-01T00:00:00Z"
user_prompt = "do something"
session_id = "sess-1"
project_slug = "proj"
`

		var record memory.MemoryRecord

		_, err := toml.Decode(withPending, &record)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(record.PendingEvaluations).To(HaveLen(1))

		// Clear pending evaluations and re-encode
		record.PendingEvaluations = nil

		var buf bytes.Buffer

		err = toml.NewEncoder(&buf).Encode(record)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(buf.String()).NotTo(ContainSubstring("pending_evaluations"))
	})

	_ = g // parent gomega used for structure
}

func TestMemoryRecord_RoundTrip(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	original := memory.MemoryRecord{
		Type:      "feedback",
		Situation: "when running tests",
		Content: memory.ContentFields{
			Behavior: "use go test directly",
			Impact:   "misses coverage and flags",
			Action:   "use targ test instead",
		},
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

func TestMemoryRecord_RoundTrip_FactContent(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	original := memory.MemoryRecord{
		Type:      "fact",
		Situation: "Go project conventions",
		Content: memory.ContentFields{
			Subject:   "this project",
			Predicate: "uses",
			Object:    "targ build system",
		},
	}

	var buf bytes.Buffer

	err := toml.NewEncoder(&buf).Encode(original)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	encoded := buf.String()
	g.Expect(encoded).To(ContainSubstring("[content]"))
	g.Expect(encoded).To(ContainSubstring(`subject = "this project"`))
	g.Expect(encoded).To(ContainSubstring(`predicate = "uses"`))
	g.Expect(encoded).To(ContainSubstring(`object = "targ build system"`))
	g.Expect(encoded).NotTo(ContainSubstring("behavior"))
	g.Expect(encoded).NotTo(ContainSubstring("impact"))
	g.Expect(encoded).NotTo(ContainSubstring("action"))

	var decoded memory.MemoryRecord

	_, err = toml.Decode(encoded, &decoded)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(decoded).To(Equal(original))
}

func TestMemoryRecord_RoundTrip_FeedbackContent(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	original := memory.MemoryRecord{
		Type:      "feedback",
		Situation: "when running tests",
		Content: memory.ContentFields{
			Behavior: "use go test directly",
			Impact:   "misses coverage",
			Action:   "use targ test",
		},
	}

	var buf bytes.Buffer

	err := toml.NewEncoder(&buf).Encode(original)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	encoded := buf.String()
	g.Expect(encoded).To(ContainSubstring("[content]"))
	g.Expect(encoded).To(ContainSubstring(`behavior = "use go test directly"`))
	g.Expect(encoded).NotTo(ContainSubstring("subject"))
	g.Expect(encoded).NotTo(ContainSubstring("predicate"))
	g.Expect(encoded).NotTo(ContainSubstring("object"))

	var decoded memory.MemoryRecord

	_, err = toml.Decode(encoded, &decoded)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(decoded).To(Equal(original))
}

func TestMemoryRecord_RoundTrip_NewFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	original := memory.MemoryRecord{
		SchemaVersion: 1,
		Type:          "feedback",
		Situation:     "when surfacing memories",
		Content: memory.ContentFields{
			Behavior: "miss relevant ones",
			Impact:   "agent repeats mistakes",
			Action:   "track missed_count",
		},
		CreatedAt:         "2026-04-01T00:00:00Z",
		UpdatedAt:         "2026-04-02T00:00:00Z",
		SurfacedCount:     10,
		FollowedCount:     7,
		NotFollowedCount:  1,
		IrrelevantCount:   2,
		MissedCount:       3,
		InitialConfidence: 0.7,
	}

	var buf bytes.Buffer

	err := toml.NewEncoder(&buf).Encode(original)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	encoded := buf.String()
	g.Expect(encoded).To(ContainSubstring("schema_version = 1"))
	g.Expect(encoded).To(ContainSubstring("missed_count = 3"))
	g.Expect(encoded).To(ContainSubstring("initial_confidence = 0.7"))

	var decoded memory.MemoryRecord

	_, err = toml.Decode(encoded, &decoded)
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
		Content: memory.ContentFields{
			Behavior: "skip hooks",
			Impact:   "broken builds",
			Action:   "always run hooks",
		},
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
