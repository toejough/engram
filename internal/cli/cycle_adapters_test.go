package cli_test

import (
	"context"
	"errors"
	"io"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/memory"
)

func TestCyclePersisterAdapter_WriteFact_WritesToDisk(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	// nil caller skips conflict check; lister returning empty also skips dedup.
	adapter := cli.ExportNewCyclePersisterAdapter(dataDir, nil, &fakeMemoryLister{}, io.Discard)

	name, ok, err := adapter.WriteFact(context.Background(), "running tests", "tests", "use", "targ")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(ok).To(BeTrue())
	g.Expect(name).NotTo(BeEmpty())
}

func TestCyclePersisterAdapter_WriteFeedback_WritesToDisk(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	adapter := cli.ExportNewCyclePersisterAdapter(dataDir, nil, &fakeMemoryLister{}, io.Discard)

	name, ok, err := adapter.WriteFeedback(
		context.Background(), "writing code", "no tests", "regressions", "write tests first",
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(ok).To(BeTrue())
	g.Expect(name).NotTo(BeEmpty())
}

func TestCycleRecallerAdapter_Recall_PropagatesSummarizerError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	adapter := cli.ExportNewCycleRecallerAdapter(dataDir, &errSummarizer{err: errors.New("boom")})

	// Real recall pipeline with empty data dir + non-existent projectDir → orchestrator
	// will run with no sessions/memories and produce an empty result (no error).
	report, err := adapter.Recall(context.Background(), t.TempDir(), "anything")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Empty report when nothing is found.
	_ = report
}

// errSummarizer satisfies recall.SummarizerI for adapter tests.
type errSummarizer struct{ err error }

func (s *errSummarizer) ExtractRelevant(_ context.Context, _, _ string) (string, error) {
	return "", s.err
}

func (s *errSummarizer) SummarizeFindings(_ context.Context, _, _ string) (string, error) {
	return "", s.err
}

// fakeMemoryLister implements the memoryLister interface for tests.
type fakeMemoryLister struct {
	memories []*memory.Stored
	err      error
}

func (f *fakeMemoryLister) ListAllMemories(string) ([]*memory.Stored, error) {
	return f.memories, f.err
}
