package signal_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/signal"
)

func TestMigrationRunner_LLMError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	unscored := &memory.MemoryRecord{Title: "fail-mem", Generalizability: 0}
	lister := &migrateLister{records: []*memory.MemoryRecord{unscored}}
	writer := &mockMigrationWriter{}

	errLLM := errors.New("LLM unavailable")
	scorer := &mockGeneralizabilityScorer{err: errLLM}

	runner := signal.NewMigrationRunner(lister, scorer, writer, nil)

	_, err := runner.ScoreUnscored(context.Background())
	g.Expect(err).To(MatchError(ContainSubstring("scoring batch")))
	g.Expect(err).To(MatchError(ContainSubstring("LLM unavailable")))
}

func TestMigrationRunner_ScoresUnscoredMemories(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	unscoredA := &memory.MemoryRecord{Title: "mem-a", Generalizability: 0}
	unscoredB := &memory.MemoryRecord{Title: "mem-b", Generalizability: 0}

	lister := &migrateLister{records: []*memory.MemoryRecord{unscoredA, unscoredB}}
	writer := &mockMigrationWriter{}

	const scoreA = 3
	const scoreB = 5

	scorer := &mockGeneralizabilityScorer{scores: []int{scoreA, scoreB}}

	runner := signal.NewMigrationRunner(lister, scorer, writer, nil)

	scored, err := runner.ScoreUnscored(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(scored).To(Equal(2))
	g.Expect(unscoredA.Generalizability).To(Equal(scoreA))
	g.Expect(unscoredB.Generalizability).To(Equal(scoreB))
}

func TestMigrationRunner_SkipsAlreadyScored(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const existingScore = 4

	alreadyScored := &memory.MemoryRecord{Title: "scored-mem", Generalizability: existingScore}
	unscored := &memory.MemoryRecord{Title: "unscored-mem", Generalizability: 0}

	lister := &migrateLister{
		records: []*memory.MemoryRecord{alreadyScored, unscored},
	}
	writer := &mockMigrationWriter{}

	const newScore = 2

	scorer := &mockGeneralizabilityScorer{scores: []int{newScore}}

	runner := signal.NewMigrationRunner(lister, scorer, writer, nil)

	scored, err := runner.ScoreUnscored(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(scored).To(Equal(1))
	g.Expect(alreadyScored.Generalizability).To(Equal(existingScore))
	g.Expect(unscored.Generalizability).To(Equal(newScore))
}

func TestMigrationRunner_WriteError_LogsAndContinues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := &memory.MemoryRecord{Title: "write-fail", Generalizability: 0}
	lister := &migrateLister{records: []*memory.MemoryRecord{record}}

	errWrite := errors.New("disk full")
	writer := &mockMigrationWriter{err: errWrite}

	const assignedScore = 4

	scorer := &mockGeneralizabilityScorer{scores: []int{assignedScore}}

	var stderr bytes.Buffer

	runner := signal.NewMigrationRunner(lister, scorer, writer, &stderr)

	scored, err := runner.ScoreUnscored(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(scored).To(Equal(0))
	g.Expect(stderr.String()).To(ContainSubstring("write failed"))
	g.Expect(stderr.String()).To(ContainSubstring("disk full"))
}

func TestMigrationRunner_WritesScoresBack(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := &memory.MemoryRecord{
		Title:            "write-test",
		Generalizability: 0,
		SourcePath:       "/data/memories/write-test.toml",
	}

	lister := &migrateLister{records: []*memory.MemoryRecord{record}}
	writer := &mockMigrationWriter{}

	const expectedScore = 3

	scorer := &mockGeneralizabilityScorer{scores: []int{expectedScore}}

	runner := signal.NewMigrationRunner(lister, scorer, writer, nil)

	scored, err := runner.ScoreUnscored(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(scored).To(Equal(1))
	g.Expect(writer.written).To(HaveLen(1))
	g.Expect(writer.written[0].Title).To(Equal("write-test"))
	g.Expect(writer.written[0].Generalizability).To(Equal(expectedScore))
}

// migrateLister implements signal.MemoryRecordLister for migration tests.
type migrateLister struct {
	records []*memory.MemoryRecord
	err     error
}

func (m *migrateLister) ListAllRecords(_ context.Context) ([]*memory.MemoryRecord, error) {
	return m.records, m.err
}

// mockGeneralizabilityScorer implements signal.GeneralizabilityScorer for tests.
type mockGeneralizabilityScorer struct {
	scores []int
	err    error
}

func (m *mockGeneralizabilityScorer) ScoreBatch(
	_ context.Context,
	_ []*memory.MemoryRecord,
) ([]int, error) {
	return m.scores, m.err
}

// mockMigrationWriter implements signal.MigrationWriter for tests.
type mockMigrationWriter struct {
	written []*memory.MemoryRecord
	err     error
}

func (m *mockMigrationWriter) WriteRecord(record *memory.MemoryRecord) error {
	if m.err != nil {
		return m.err
	}

	m.written = append(m.written, record)

	return nil
}
