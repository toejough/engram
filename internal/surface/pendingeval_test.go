package surface_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

func TestWithPendingEvalModifier_WiresModifierOnSurfacer(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var modifiedPaths []string

	modifier := func(path string, mutate func(*memory.MemoryRecord)) error {
		record := memory.MemoryRecord{}
		mutate(&record)

		modifiedPaths = append(modifiedPaths, path)

		return nil
	}

	memories := []*memory.Stored{
		{Situation: "commit context", Content: memory.ContentFields{Behavior: "bad commit", Action: "good commit"},
			FilePath: "mem/commit.toml"},
		{Situation: "build context", Content: memory.ContentFields{Behavior: "bad build", Action: "good build"},
			FilePath: "mem/build.toml"},
		{Situation: "review context", Content: memory.ContentFields{Behavior: "bad review", Action: "good review"},
			FilePath: "mem/review.toml"},
		{Situation: "deploy context", Content: memory.ContentFields{Behavior: "bad deploy", Action: "good deploy"},
			FilePath: "mem/deploy.toml"},
	}

	retriever := &fakeRetriever{memories: memories}
	surfacer := surface.New(
		retriever,
		surface.WithPendingEvalModifier(modifier),
	)

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:       surface.ModePrompt,
		DataDir:    "/data",
		Message:    "commit build",
		SessionID:  "session-xyz",
		UserPrompt: "commit build",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(modifiedPaths).NotTo(BeEmpty())
}

func TestWritePendingEvaluations_ErrorContinues_ReturnsJoinedErrors(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writeErr := errors.New("disk full")

	var successPaths []string

	modifier := func(path string, mutate func(*memory.MemoryRecord)) error {
		if path == "mem/bad.toml" {
			return writeErr
		}

		record := memory.MemoryRecord{}
		mutate(&record)

		successPaths = append(successPaths, path)

		return nil
	}

	memories := []*memory.Stored{
		{FilePath: "mem/good.toml"},
		{FilePath: "mem/bad.toml"},
		{FilePath: "mem/also-good.toml"},
	}

	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)

	err := surface.WritePendingEvaluations(
		memories, modifier,
		"session-xyz", "my-project", "test query",
		now,
	)

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("disk full"))
		g.Expect(err.Error()).To(ContainSubstring("mem/bad.toml"))
	}

	// Continues after the error — both good paths are still written.
	g.Expect(successPaths).To(ConsistOf("mem/good.toml", "mem/also-good.toml"))
}

func TestWritePendingEvaluations_WritesMutationsForEachMemory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	type capturedMutation struct {
		path   string
		record memory.MemoryRecord
	}

	var captured []capturedMutation

	modifier := func(path string, mutate func(*memory.MemoryRecord)) error {
		record := memory.MemoryRecord{}
		mutate(&record)
		captured = append(captured, capturedMutation{path: path, record: record})

		return nil
	}

	memories := []*memory.Stored{
		{FilePath: "mem/alpha.toml"},
		{FilePath: "mem/beta.toml"},
	}

	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)

	err := surface.WritePendingEvaluations(
		memories, modifier,
		"session-abc", "my-project", "how do I commit?",
		now,
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(captured).To(HaveLen(2))

	if len(captured) < 2 {
		return
	}

	g.Expect(captured[0].path).To(Equal("mem/alpha.toml"))
	g.Expect(captured[0].record.PendingEvaluations).To(HaveLen(1))
	g.Expect(captured[0].record.PendingEvaluations[0].SessionID).To(Equal("session-abc"))
	g.Expect(captured[0].record.PendingEvaluations[0].ProjectSlug).To(Equal("my-project"))
	g.Expect(captured[0].record.PendingEvaluations[0].UserPrompt).To(Equal("how do I commit?"))

	g.Expect(captured[1].path).To(Equal("mem/beta.toml"))
	g.Expect(captured[1].record.PendingEvaluations).To(HaveLen(1))
	g.Expect(captured[1].record.PendingEvaluations[0].SessionID).To(Equal("session-abc"))
	g.Expect(captured[1].record.PendingEvaluations[0].ProjectSlug).To(Equal("my-project"))
	g.Expect(captured[1].record.PendingEvaluations[0].UserPrompt).To(Equal("how do I commit?"))
}
