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
