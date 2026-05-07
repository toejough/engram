package cycle_test

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cycle"
	"engram/internal/memory"
)

func TestOutput_MarshalsLearnedAndRecalled(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	out := cycle.Output{
		Learned: []cycle.LearnedMemory{
			{
				Name: "doing-x",
				MemoryRecord: memory.MemoryRecord{
					SchemaVersion: 2,
					Source:        "agent",
					Situation:     "doing X",
					Type:          "feedback",
					Content:       memory.ContentFields{Behavior: "b", Impact: "i", Action: "a"},
				},
			},
		},
		Recalled: []cycle.RecalledReport{
			{Query: "q1", Report: "r1"},
		},
	}

	encoded, err := json.Marshal(out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var roundtrip map[string]any
	g.Expect(json.Unmarshal(encoded, &roundtrip)).To(Succeed())
	g.Expect(roundtrip).To(HaveKey("learned"))
	g.Expect(roundtrip).To(HaveKey("recalled"))
}

func TestOutput_EmptyArraysWhenNothingHappened(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	out := cycle.NewOutput()

	encoded, err := json.Marshal(out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(encoded)).To(ContainSubstring(`"learned":[]`))
	g.Expect(string(encoded)).To(ContainSubstring(`"recalled":[]`))
}
