package signal_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/signal"
)

// TestConsolidateBatch_Apply_WritesConsolidated verifies that apply mode writes the result.
func TestConsolidateBatch_Apply_WritesConsolidated(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

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

	extracted := &memory.MemoryRecord{Title: "consolidated", SourcePath: "/c.toml"}

	records := []*memory.MemoryRecord{memA, memB, memC}
	lister := &migrationFakeLister{records: records}
	writer := &migrationFakeWriter{}

	consolidator := signal.NewConsolidator(
		signal.WithScorer(&mockScorer{candidates: candidates}),
		signal.WithConfirmer(&mockConfirmer{clusters: confirmedClusters}),
		signal.WithExtractor(&mockExtractor{result: extracted}),
	)

	runner := signal.NewMigrationRunner(lister, nil, writer, nil)
	runner.SetConsolidator(consolidator)

	result, err := runner.ConsolidateBatch(context.Background(), false)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if result != nil {
		g.Expect(result.ConsolidatedCount).To(gomega.BeNumerically(">=", 1))
	}

	g.Expect(writer.written).NotTo(gomega.BeEmpty())
}

// TestConsolidateBatch_DryRun_LogsClusters verifies that dry-run mode logs clusters via logMigrateStderrf.
func TestConsolidateBatch_DryRun_LogsClusters(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

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

	records := []*memory.MemoryRecord{memA, memB, memC}
	lister := &migrationFakeLister{records: records}

	consolidator := signal.NewConsolidator(
		signal.WithScorer(&mockScorer{candidates: candidates}),
		signal.WithConfirmer(&mockConfirmer{clusters: confirmedClusters}),
	)

	var stderrBuf bytes.Buffer

	runner := signal.NewMigrationRunner(lister, nil, nil, &stderrBuf)
	runner.SetConsolidator(consolidator)

	result, err := runner.ConsolidateBatch(context.Background(), true)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if result != nil {
		g.Expect(result.ClusterCount).To(gomega.BeNumerically(">=", 1))
	}

	// logMigrateStderrf was called with the cluster info.
	g.Expect(stderrBuf.String()).To(gomega.ContainSubstring("Cluster"))
}

// TestConsolidatorLogStderrf_NilStderr verifies that logStderrf is safe with nil stderr.
func TestConsolidatorLogStderrf_NilStderr(t *testing.T) {
	t.Parallel()

	// Consolidator with no stderr set — logStderrf should not panic.
	consolidator := signal.NewConsolidator()
	consolidator.ExportLogStderrf("test %d", 42)
}

// TestConsolidatorLogStderrf_WritesToStderr verifies logStderrf emits to stderr.
func TestConsolidatorLogStderrf_WritesToStderr(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	var buf bytes.Buffer

	consolidator := signal.ExportNewConsolidatorWithStderr(&buf)
	consolidator.ExportLogStderrf("hello %s", "stderr")
	g.Expect(buf.String()).To(gomega.Equal("hello stderr"))
}

// TestLogMigrateStderrf_NilWriter verifies the nil-guard early return.
func TestLogMigrateStderrf_NilWriter(t *testing.T) {
	t.Parallel()

	// Should not panic — nil writer returns early.
	signal.ExportLogMigrateStderrf(nil, "test %s", "msg")
}

// TestLogMigrateStderrf_WritesToWriter verifies output is written when writer is non-nil.
func TestLogMigrateStderrf_WritesToWriter(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	var buf bytes.Buffer

	signal.ExportLogMigrateStderrf(&buf, "hello %s", "world")
	g.Expect(buf.String()).To(gomega.Equal("hello world"))
}

// TestNewMigrationRunner_ConsolidateBatch_ListerError verifies error propagation.
func TestNewMigrationRunner_ConsolidateBatch_ListerError(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	lister := &migrationFakeLister{err: errors.New("disk read failed")}
	scorer := &migrationFakeScorer{}
	runner := signal.NewMigrationRunner(lister, scorer, nil, nil)
	runner.SetConsolidator(signal.NewConsolidator())

	_, err := runner.ConsolidateBatch(context.Background(), false)
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("listing records")))
}

// TestNewMigrationRunner_ConsolidateBatch_NilConsolidator verifies that
// ConsolidateBatch returns an empty result when no consolidator is set.
func TestNewMigrationRunner_ConsolidateBatch_NilConsolidator(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	lister := &migrationFakeLister{records: []*memory.MemoryRecord{}}
	runner := signal.NewMigrationRunner(lister, nil, nil, nil)

	result, err := runner.ConsolidateBatch(context.Background(), false)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(result).NotTo(gomega.BeNil())
}

// TestNewMigrationRunner_ScoreUnscored_Empty verifies that ScoreUnscored returns 0 with no memories.
func TestNewMigrationRunner_ScoreUnscored_Empty(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	lister := &migrationFakeLister{records: []*memory.MemoryRecord{}}
	runner := signal.NewMigrationRunner(lister, &migrationFakeScorer{}, nil, nil)

	scored, err := runner.ScoreUnscored(context.Background())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(scored).To(gomega.Equal(0))
}

// TestNewMigrationRunner_ScoreUnscored_ListerError verifies error propagation.
func TestNewMigrationRunner_ScoreUnscored_ListerError(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	lister := &migrationFakeLister{err: errors.New("read failed")}
	runner := signal.NewMigrationRunner(lister, &migrationFakeScorer{}, nil, nil)

	_, err := runner.ScoreUnscored(context.Background())
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("listing records")))
}

// TestNewMigrationRunner_ScoreUnscored_WritesScores verifies that memories with
// generalizability==0 are scored and written.
func TestNewMigrationRunner_ScoreUnscored_WritesScores(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	records := []*memory.MemoryRecord{
		{Title: "mem-a", Generalizability: 0},
		{Title: "mem-b", Generalizability: 3},
	}

	lister := &migrationFakeLister{records: records}
	scorer := &migrationFakeScorer{scores: []int{4}}
	writer := &migrationFakeWriter{}

	var stderrBuf bytes.Buffer

	runner := signal.NewMigrationRunner(lister, scorer, writer, &stderrBuf)

	scored, err := runner.ScoreUnscored(context.Background())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(scored).To(gomega.Equal(1))
	g.Expect(writer.written).To(gomega.HaveLen(1))
}

// TestSetConsolidator verifies that SetConsolidator wires the consolidator.
func TestSetConsolidator(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	lister := &migrationFakeLister{records: []*memory.MemoryRecord{}}
	runner := signal.NewMigrationRunner(lister, nil, nil, nil)

	// Before SetConsolidator: NilConsolidator returns empty result.
	result, err := runner.ConsolidateBatch(context.Background(), false)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if result != nil {
		g.Expect(result.ClusterCount).To(gomega.Equal(0))
	}

	runner.SetConsolidator(signal.NewConsolidator())

	// After SetConsolidator: Consolidator runs (empty lister, no clusters).
	result2, err2 := runner.ConsolidateBatch(context.Background(), false)
	g.Expect(err2).NotTo(gomega.HaveOccurred())

	if result2 != nil {
		g.Expect(result2.ClusterCount).To(gomega.Equal(0))
	}
}

// unexported test helpers.

type migrationFakeLister struct {
	records []*memory.MemoryRecord
	err     error
}

func (f *migrationFakeLister) ListAllRecords(_ context.Context) ([]*memory.MemoryRecord, error) {
	return f.records, f.err
}

type migrationFakeScorer struct {
	scores []int
	err    error
}

func (f *migrationFakeScorer) ScoreBatch(_ context.Context, _ []*memory.MemoryRecord) ([]int, error) {
	return f.scores, f.err
}

type migrationFakeWriter struct {
	written []*memory.MemoryRecord
	err     error
}

func (f *migrationFakeWriter) WriteRecord(rec *memory.MemoryRecord) error {
	f.written = append(f.written, rec)
	return f.err
}
