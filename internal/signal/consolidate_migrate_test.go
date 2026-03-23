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

func TestConsolidateBatch_Apply_ConsolidatesClusters(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memA := &memory.MemoryRecord{Title: "mem-a", SourcePath: "/a.toml"}
	memB := &memory.MemoryRecord{Title: "mem-b", SourcePath: "/b.toml"}
	memC := &memory.MemoryRecord{Title: "mem-c", SourcePath: "/c.toml"}

	candidates := []signal.ScoredCandidate{
		{Memory: memB, Score: 0.9},
		{Memory: memC, Score: 0.85},
	}

	confirmedClusters := []signal.ConfirmedCluster{
		{
			Members:   []*memory.MemoryRecord{memA, memB, memC},
			Principle: "shared principle",
		},
	}

	consolidated := &memory.MemoryRecord{
		Title:      "consolidated",
		SourcePath: "/consolidated.toml",
	}

	lister := &migrateLister{records: []*memory.MemoryRecord{memA, memB, memC}}
	writer := &mockMigrationWriter{}

	consolidator := signal.NewConsolidator(
		signal.WithScorer(&batchMockScorer{candidates: candidates}),
		signal.WithConfirmer(&batchMockConfirmer{clusters: confirmedClusters}),
		signal.WithExtractor(&batchMockExtractor{result: consolidated}),
	)

	runner := signal.NewMigrationRunner(lister, nil, writer, nil)
	runner.SetConsolidator(consolidator)

	result, err := runner.ConsolidateBatch(context.Background(), false)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.ConsolidatedCount).To(Equal(1))
	g.Expect(writer.written).To(HaveLen(1))
	g.Expect(writer.written[0].Title).To(Equal("consolidated"))
}

func TestConsolidateBatch_ConsolidateError_LogsAndContinues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memA := &memory.MemoryRecord{Title: "mem-a", SourcePath: "/a.toml"}
	memB := &memory.MemoryRecord{Title: "mem-b", SourcePath: "/b.toml"}
	memC := &memory.MemoryRecord{Title: "mem-c", SourcePath: "/c.toml"}

	candidates := []signal.ScoredCandidate{
		{Memory: memB, Score: 0.9},
		{Memory: memC, Score: 0.85},
	}

	confirmedClusters := []signal.ConfirmedCluster{
		{
			Members:   []*memory.MemoryRecord{memA, memB, memC},
			Principle: "shared principle",
		},
	}

	lister := &migrateLister{records: []*memory.MemoryRecord{memA, memB, memC}}

	var stderr bytes.Buffer

	// No extractor → consolidateCluster returns ErrNilExtractor.
	consolidator := signal.NewConsolidator(
		signal.WithScorer(&batchMockScorer{candidates: candidates}),
		signal.WithConfirmer(&batchMockConfirmer{clusters: confirmedClusters}),
	)

	runner := signal.NewMigrationRunner(lister, nil, nil, &stderr)
	runner.SetConsolidator(consolidator)

	result, err := runner.ConsolidateBatch(context.Background(), false)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.ConsolidatedCount).To(Equal(0))
	g.Expect(stderr.String()).To(ContainSubstring("consolidation error"))
}

func TestConsolidateBatch_DryRun_FindsClustersNoWrite(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memA := &memory.MemoryRecord{Title: "mem-a", SourcePath: "/a.toml"}
	memB := &memory.MemoryRecord{Title: "mem-b", SourcePath: "/b.toml"}
	memC := &memory.MemoryRecord{Title: "mem-c", SourcePath: "/c.toml"}

	candidates := []signal.ScoredCandidate{
		{Memory: memB, Score: 0.9},
		{Memory: memC, Score: 0.85},
	}

	confirmedClusters := []signal.ConfirmedCluster{
		{
			Members:   []*memory.MemoryRecord{memA, memB, memC},
			Principle: "shared principle",
		},
	}

	lister := &migrateLister{records: []*memory.MemoryRecord{memA, memB, memC}}
	writer := &mockMigrationWriter{}

	var stderr bytes.Buffer

	consolidator := signal.NewConsolidator(
		signal.WithScorer(&batchMockScorer{candidates: candidates}),
		signal.WithConfirmer(&batchMockConfirmer{clusters: confirmedClusters}),
		signal.WithExtractor(&batchMockExtractor{
			result: &memory.MemoryRecord{Title: "consolidated"},
		}),
	)

	runner := signal.NewMigrationRunner(lister, nil, writer, &stderr)
	runner.SetConsolidator(consolidator)

	result, err := runner.ConsolidateBatch(context.Background(), true)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.ClusterCount).To(Equal(1))
	g.Expect(writer.written).To(BeEmpty())
	g.Expect(stderr.String()).To(ContainSubstring("shared principle"))
}

func TestConsolidateBatch_ListError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	errList := errors.New("disk failure")
	lister := &migrateLister{err: errList}

	consolidator := signal.NewConsolidator(
		signal.WithScorer(&batchMockScorer{}),
		signal.WithConfirmer(&batchMockConfirmer{}),
	)

	runner := signal.NewMigrationRunner(lister, nil, nil, nil)
	runner.SetConsolidator(consolidator)

	_, err := runner.ConsolidateBatch(context.Background(), false)
	g.Expect(err).To(MatchError(ContainSubstring("listing records")))
	g.Expect(err).To(MatchError(ContainSubstring("disk failure")))
}

func TestConsolidateBatch_NilConsolidator_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &migrateLister{records: []*memory.MemoryRecord{}}
	runner := signal.NewMigrationRunner(lister, nil, nil, nil)

	result, err := runner.ConsolidateBatch(context.Background(), false)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.ClusterCount).To(Equal(0))
}

func TestConsolidateBatch_NoClusters(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memA := &memory.MemoryRecord{Title: "mem-a", SourcePath: "/a.toml"}

	lister := &migrateLister{records: []*memory.MemoryRecord{memA}}
	writer := &mockMigrationWriter{}

	// Scorer returns no candidates → no clusters.
	consolidator := signal.NewConsolidator(
		signal.WithScorer(&batchMockScorer{candidates: nil}),
		signal.WithConfirmer(&batchMockConfirmer{}),
	)

	runner := signal.NewMigrationRunner(lister, nil, writer, nil)
	runner.SetConsolidator(consolidator)

	result, err := runner.ConsolidateBatch(context.Background(), false)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.ClusterCount).To(Equal(0))
	g.Expect(result.ConsolidatedCount).To(Equal(0))
}

func TestConsolidateBatch_SkipsAbsorbed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	absorbed := &memory.MemoryRecord{
		Title:      "absorbed-mem",
		SourcePath: "/absorbed.toml",
		Absorbed: []memory.AbsorbedRecord{
			{From: "old-mem"},
		},
	}
	fresh := &memory.MemoryRecord{Title: "fresh-mem", SourcePath: "/fresh.toml"}

	lister := &migrateLister{records: []*memory.MemoryRecord{absorbed, fresh}}
	writer := &mockMigrationWriter{}

	// Scorer returns no candidates for the fresh record → no clusters.
	consolidator := signal.NewConsolidator(
		signal.WithScorer(&batchMockScorer{candidates: nil}),
		signal.WithConfirmer(&batchMockConfirmer{}),
	)

	runner := signal.NewMigrationRunner(lister, nil, writer, nil)
	runner.SetConsolidator(consolidator)

	result, err := runner.ConsolidateBatch(context.Background(), false)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	// The absorbed memory should be skipped; only fresh was considered.
	g.Expect(result.ClusterCount).To(Equal(0))
}

func TestConsolidateBatch_WriteError_LogsAndContinues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memA := &memory.MemoryRecord{Title: "mem-a", SourcePath: "/a.toml"}
	memB := &memory.MemoryRecord{Title: "mem-b", SourcePath: "/b.toml"}
	memC := &memory.MemoryRecord{Title: "mem-c", SourcePath: "/c.toml"}

	candidates := []signal.ScoredCandidate{
		{Memory: memB, Score: 0.9},
		{Memory: memC, Score: 0.85},
	}

	confirmedClusters := []signal.ConfirmedCluster{
		{
			Members:   []*memory.MemoryRecord{memA, memB, memC},
			Principle: "shared principle",
		},
	}

	consolidated := &memory.MemoryRecord{
		Title:      "consolidated",
		SourcePath: "/consolidated.toml",
	}

	lister := &migrateLister{records: []*memory.MemoryRecord{memA, memB, memC}}

	errWrite := errors.New("disk full")
	writer := &mockMigrationWriter{err: errWrite}

	var stderr bytes.Buffer

	consolidator := signal.NewConsolidator(
		signal.WithScorer(&batchMockScorer{candidates: candidates}),
		signal.WithConfirmer(&batchMockConfirmer{clusters: confirmedClusters}),
		signal.WithExtractor(&batchMockExtractor{result: consolidated}),
	)

	runner := signal.NewMigrationRunner(lister, nil, writer, &stderr)
	runner.SetConsolidator(consolidator)

	result, err := runner.ConsolidateBatch(context.Background(), false)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.ConsolidatedCount).To(Equal(1))
	g.Expect(stderr.String()).To(ContainSubstring("write failed"))
	g.Expect(stderr.String()).To(ContainSubstring("disk full"))
}

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

	const (
		scoreA = 3
		scoreB = 5
	)

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

func TestScoreUnscored_WriteError_NilStderr(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &migrateLister{
		records: []*memory.MemoryRecord{
			{Title: "unscored", Generalizability: 0},
		},
	}
	scorer := &mockGeneralizabilityScorer{scores: []int{3}}
	writer := &mockMigrationWriter{err: errors.New("disk full")}

	// nil stderr — logMigrateStderrf should not panic.
	runner := signal.NewMigrationRunner(lister, scorer, writer, nil)

	scored, err := runner.ScoreUnscored(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Write failed but no panic, scored count is 0.
	g.Expect(scored).To(Equal(0))
}

// batchMockConfirmer implements signal.Confirmer for batch consolidation tests.
type batchMockConfirmer struct {
	clusters []signal.ConfirmedCluster
}

func (m *batchMockConfirmer) ConfirmClusters(
	_ context.Context,
	_ *memory.MemoryRecord,
	_ []signal.ScoredCandidate,
) ([]signal.ConfirmedCluster, error) {
	return m.clusters, nil
}

// batchMockExtractor implements signal.Extractor for batch consolidation tests.
type batchMockExtractor struct {
	result *memory.MemoryRecord
	err    error
}

func (m *batchMockExtractor) ExtractPrinciple(
	_ context.Context,
	_ signal.ConfirmedCluster,
) (*memory.MemoryRecord, error) {
	return m.result, m.err
}

// batchMockScorer implements signal.Scorer for batch consolidation tests.
type batchMockScorer struct {
	candidates []signal.ScoredCandidate
}

func (m *batchMockScorer) FindSimilar(
	_ context.Context,
	_ *memory.MemoryRecord,
	_ []string,
) ([]signal.ScoredCandidate, error) {
	return m.candidates, nil
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
