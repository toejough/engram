package learn_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/dedup"
	"engram/internal/graph"
	"engram/internal/learn"
	"engram/internal/memory"
)

func TestTP5f7_LinkRecomputerCalledAfterProcessMerge(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	ctx := context.Background()

	const (
		existingFilePath   = "/path/existing.toml"
		candidatePrinciple = "new merged principle text"
	)

	existing := &memory.Stored{
		FilePath:  existingFilePath,
		Title:     "Existing Memory",
		Principle: "old principle",
		Keywords:  []string{"alpha", "beta"},
		Concepts:  []string{"concept1"},
	}

	candidate := memory.CandidateLearning{
		Title:     "candidate",
		Principle: candidatePrinciple,
		Keywords:  []string{"alpha", "gamma"},
		Concepts:  []string{"concept2"},
	}

	recomputer := &mockLinkRecomputer{}
	mergeWriter := &fakeMergeWriter{}

	learner := learn.New(
		&fakeExtractor{candidates: []memory.CandidateLearning{candidate}},
		&fakeRetriever{memories: []*memory.Stored{existing}},
		&fakeMergingDeduplicator{
			mergePairs: []dedup.MergePair{{Candidate: candidate, Existing: existing}},
		},
		&fakeWriter{},
		"/tmp/data",
	)
	learner.SetMergeWriter(mergeWriter)
	learner.SetLinkRecomputer(recomputer)

	_, err := learner.Run(ctx, "some transcript")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(recomputer.called).To(BeTrue())
	g.Expect(recomputer.result.MergedMemoryID).To(Equal(existingFilePath))
}

// mockLinkRecomputer is a test double for the LinkRecomputer interface.
type mockLinkRecomputer struct {
	called bool
	result graph.MergeResult
}

func (m *mockLinkRecomputer) RecomputeAfterMerge(result graph.MergeResult) error {
	m.called = true
	m.result = result

	return nil
}
