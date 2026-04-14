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
		Type:      "feedback",
		Source:    "observation",
		Situation: "when running tests",
		Content: memory.ContentFields{
			Behavior: "use go test directly",
			Impact:   "misses coverage and flags",
			Action:   "use targ test instead",
		},
		CreatedAt: "2026-01-01T00:00:00Z",
		UpdatedAt: "2026-01-02T00:00:00Z",
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

func TestMemoryRecord_RoundTrip_SchemaVersion(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	original := memory.MemoryRecord{
		SchemaVersion: 2,
		Type:          "feedback",
		Situation:     "when surfacing memories",
		Content: memory.ContentFields{
			Behavior: "miss relevant ones",
			Impact:   "agent repeats mistakes",
			Action:   "track missed_count",
		},
		CreatedAt: "2026-04-01T00:00:00Z",
		UpdatedAt: "2026-04-02T00:00:00Z",
	}

	var buf bytes.Buffer

	err := toml.NewEncoder(&buf).Encode(original)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	encoded := buf.String()
	g.Expect(encoded).To(ContainSubstring("schema_version = 2"))

	var decoded memory.MemoryRecord

	_, err = toml.Decode(encoded, &decoded)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(decoded).To(Equal(original))
}
