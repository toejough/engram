package registry_test

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/registry"
)

// --- Entry / EvaluationCounters ---

func TestT240_EvaluationCountersTotal(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	counters := registry.EvaluationCounters{
		Followed: 5, Contradicted: 2, Ignored: 1,
	}
	g.Expect(counters.Total()).To(Equal(8))
}

func TestT241_EvaluationCountersTotalZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	counters := registry.EvaluationCounters{}
	g.Expect(counters.Total()).To(Equal(0))
}

// --- Signals: Effectiveness ---

func TestT242_EffectivenessWithSufficientData(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := &registry.InstructionEntry{
		Evaluations: registry.EvaluationCounters{
			Followed: 7, Contradicted: 2, Ignored: 1,
		},
	}
	result := registry.Effectiveness(entry)
	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(*result).To(BeNumerically("~", 70.0, 0.01))
}

func TestT243_EffectivenessNilBelowMinEvaluations(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := &registry.InstructionEntry{
		Evaluations: registry.EvaluationCounters{
			Followed: 1, Contradicted: 1, Ignored: 0,
		},
	}
	result := registry.Effectiveness(entry)
	g.Expect(result).To(BeNil())
}

func TestT244_EffectivenessExactlyAtMinEvaluations(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := &registry.InstructionEntry{
		Evaluations: registry.EvaluationCounters{
			Followed: 3, Contradicted: 0, Ignored: 0,
		},
	}
	result := registry.Effectiveness(entry)
	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(*result).To(BeNumerically("~", 100.0, 0.01))
}

func TestT245_EffectivenessZeroFollowed(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := &registry.InstructionEntry{
		Evaluations: registry.EvaluationCounters{
			Followed: 0, Contradicted: 2, Ignored: 1,
		},
	}
	result := registry.Effectiveness(entry)
	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(*result).To(BeNumerically("~", 0.0, 0.01))
}

// --- Signals: Frecency ---

func TestT246_FrecencyDecaysWithTime(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	lastSurfaced := now.Add(-7 * 24 * time.Hour) // 7 days ago

	const halfLifeDays = 7.0

	entry := &registry.InstructionEntry{
		SurfacedCount: 10,
		LastSurfaced:  &lastSurfaced,
		UpdatedAt:     now.Add(-30 * 24 * time.Hour),
	}

	score := registry.Frecency(entry, now, halfLifeDays)
	// 10 * exp(-7/7) = 10 * exp(-1) ≈ 3.679
	g.Expect(score).To(BeNumerically("~", 10*math.Exp(-1), 0.01))
}

func TestT247_FrecencyUsesUpdatedAtWhenNoLastSurfaced(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)

	const halfLifeDays = 7.0

	entry := &registry.InstructionEntry{
		SurfacedCount: 5,
		UpdatedAt:     now.Add(-14 * 24 * time.Hour), // 14 days ago
	}

	score := registry.Frecency(entry, now, halfLifeDays)
	// 5 * exp(-14/7) = 5 * exp(-2) ≈ 0.677
	g.Expect(score).To(BeNumerically("~", 5*math.Exp(-2), 0.01))
}

func TestT248_FrecencyZeroSurfacedCountReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	entry := &registry.InstructionEntry{
		SurfacedCount: 0,
		UpdatedAt:     now,
	}

	score := registry.Frecency(entry, now, 7.0)
	g.Expect(score).To(BeNumerically("~", 0.0, 0.001))
}

func TestT249_FrecencyFutureLastSurfacedClampsToZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	future := now.Add(24 * time.Hour)

	entry := &registry.InstructionEntry{
		SurfacedCount: 10,
		LastSurfaced:  &future,
		UpdatedAt:     now,
	}

	score := registry.Frecency(entry, now, 7.0)
	// daysSince clamped to 0 → exp(0) = 1 → 10 * 1 = 10
	g.Expect(score).To(BeNumerically("~", 10.0, 0.01))
}

// --- Classify ---

func TestT250_ClassifyWorkingHighSurfacingHighEffectiveness(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := &registry.InstructionEntry{
		SourceType:    "memory",
		SurfacedCount: 10,
		Evaluations: registry.EvaluationCounters{
			Followed: 8, Contradicted: 1, Ignored: 1,
		},
	}
	quadrant := registry.Classify(entry, 3, 50.0)
	g.Expect(quadrant).To(Equal(registry.Working))
}

func TestT251_ClassifyLeechHighSurfacingLowEffectiveness(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := &registry.InstructionEntry{
		SourceType:    "memory",
		SurfacedCount: 10,
		Evaluations: registry.EvaluationCounters{
			Followed: 1, Contradicted: 5, Ignored: 4,
		},
	}
	quadrant := registry.Classify(entry, 3, 50.0)
	g.Expect(quadrant).To(Equal(registry.Leech))
}

func TestT252_ClassifyHiddenGemLowSurfacingHighEffectiveness(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := &registry.InstructionEntry{
		SourceType:    "memory",
		SurfacedCount: 1,
		Evaluations: registry.EvaluationCounters{
			Followed: 3, Contradicted: 0, Ignored: 0,
		},
	}
	quadrant := registry.Classify(entry, 3, 50.0)
	g.Expect(quadrant).To(Equal(registry.HiddenGem))
}

func TestT253_ClassifyNoiseLowSurfacingLowEffectiveness(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := &registry.InstructionEntry{
		SourceType:    "memory",
		SurfacedCount: 1,
		Evaluations: registry.EvaluationCounters{
			Followed: 0, Contradicted: 2, Ignored: 1,
		},
	}
	quadrant := registry.Classify(entry, 3, 50.0)
	g.Expect(quadrant).To(Equal(registry.Noise))
}

func TestT254_ClassifyInsufficientDataBelowMinEvals(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := &registry.InstructionEntry{
		SourceType:    "memory",
		SurfacedCount: 10,
		Evaluations: registry.EvaluationCounters{
			Followed: 1, Contradicted: 0, Ignored: 0,
		},
	}
	quadrant := registry.Classify(entry, 3, 50.0)
	g.Expect(quadrant).To(Equal(registry.Insufficient))
}

func TestT255_ClassifyAlwaysLoadedWorkingOrLeechOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	for _, sourceType := range []string{"claude-md", "memory-md"} {
		t.Run(sourceType+"_working", func(t *testing.T) {
			t.Parallel()

			entry := &registry.InstructionEntry{
				SourceType:    sourceType,
				SurfacedCount: 0, // low surfacing, but always-loaded → no HiddenGem
				Evaluations: registry.EvaluationCounters{
					Followed: 5, Contradicted: 0, Ignored: 0,
				},
			}
			quadrant := registry.Classify(entry, 3, 50.0)
			g.Expect(quadrant).To(Equal(registry.Working))
		})

		t.Run(sourceType+"_leech", func(t *testing.T) {
			t.Parallel()

			entry := &registry.InstructionEntry{
				SourceType:    sourceType,
				SurfacedCount: 0,
				Evaluations: registry.EvaluationCounters{
					Followed: 0, Contradicted: 3, Ignored: 0,
				},
			}
			quadrant := registry.Classify(entry, 3, 50.0)
			g.Expect(quadrant).To(Equal(registry.Leech))
		})
	}
}

// --- Backfill ---

func TestT256_BackfillCreatesEntriesFromMemories(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	created := now.Add(-48 * time.Hour)
	lastSurf := now.Add(-24 * time.Hour)

	config := registry.BackfillConfig{
		Scanner: &fakeScanner{memories: []registry.ScannedMemory{
			{
				FilePath:  "memories/test-memory.toml",
				Title:     "Test Memory",
				Content:   "test content",
				UpdatedAt: now.Add(-1 * time.Hour),
			},
		}},
		SurfacingLog: &fakeSurfacingLog{data: map[string]registry.SurfacingData{
			"memories/test-memory.toml": {Count: 5, LastSurfaced: &lastSurf},
		}},
		CreationLog: &fakeCreationLog{times: map[string]time.Time{
			"memories/test-memory.toml": created,
		}},
		Evaluations: &fakeEvaluations{data: map[string]registry.EvaluationCounters{
			"memories/test-memory.toml": {Followed: 3, Contradicted: 1, Ignored: 0},
		}},
		Now: now,
	}

	entries, err := registry.Backfill(config)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(HaveLen(1))

	entry := entries[0]
	g.Expect(entry.ID).To(Equal("memories/test-memory.toml"))
	g.Expect(entry.SourceType).To(Equal("memory"))
	g.Expect(entry.Title).To(Equal("Test Memory"))
	g.Expect(entry.SurfacedCount).To(Equal(5))
	g.Expect(entry.RegisteredAt).To(Equal(created))
	g.Expect(entry.Evaluations.Followed).To(Equal(3))
	g.Expect(entry.Evaluations.Contradicted).To(Equal(1))
	g.Expect(entry.LastSurfaced).NotTo(BeNil())

	if entry.LastSurfaced != nil {
		g.Expect(*entry.LastSurfaced).To(Equal(lastSurf))
	}

	expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte("test content")))
	g.Expect(entry.ContentHash).To(Equal(expectedHash))
}

func TestT257_BackfillAbsorbsRetiredMemoryCounters(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)

	config := registry.BackfillConfig{
		Scanner: &fakeScanner{memories: []registry.ScannedMemory{
			{
				FilePath:  "memories/active.toml",
				Title:     "Active",
				Content:   "active content",
				UpdatedAt: now,
			},
			{
				FilePath:  "memories/retired.toml",
				Title:     "Retired",
				Content:   "retired content",
				RetiredBy: "memories/active.toml",
				UpdatedAt: now.Add(-1 * time.Hour),
			},
		}},
		SurfacingLog: &fakeSurfacingLog{data: map[string]registry.SurfacingData{
			"memories/active.toml":  {Count: 3},
			"memories/retired.toml": {Count: 7},
		}},
		CreationLog: &fakeCreationLog{times: map[string]time.Time{}},
		Evaluations: &fakeEvaluations{data: map[string]registry.EvaluationCounters{
			"memories/active.toml":  {Followed: 2, Contradicted: 0, Ignored: 0},
			"memories/retired.toml": {Followed: 5, Contradicted: 1, Ignored: 0},
		}},
		Now: now,
	}

	entries, err := registry.Backfill(config)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(HaveLen(1)) // only active, not retired

	entry := entries[0]
	g.Expect(entry.SurfacedCount).To(Equal(10))       // 3 + 7
	g.Expect(entry.Evaluations.Followed).To(Equal(7)) // 2 + 5
	g.Expect(entry.Evaluations.Contradicted).To(Equal(1))
	g.Expect(entry.Absorbed).To(HaveLen(1))

	absorbed := entry.Absorbed[0]
	g.Expect(absorbed.From).To(Equal("memories/retired.toml"))
	g.Expect(absorbed.SurfacedCount).To(Equal(7))
	g.Expect(absorbed.Evaluations.Followed).To(Equal(5))
}

func TestT258_BackfillUsesNowWhenNoCreationTime(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)

	config := registry.BackfillConfig{
		Scanner: &fakeScanner{memories: []registry.ScannedMemory{
			{
				FilePath:  "memories/no-creation.toml",
				Title:     "No Creation Time",
				Content:   "content",
				UpdatedAt: now,
			},
		}},
		SurfacingLog: &fakeSurfacingLog{data: map[string]registry.SurfacingData{}},
		CreationLog:  &fakeCreationLog{times: map[string]time.Time{}},
		Evaluations:  &fakeEvaluations{data: map[string]registry.EvaluationCounters{}},
		Now:          now,
	}

	entries, err := registry.Backfill(config)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(HaveLen(1))
	g.Expect(entries[0].RegisteredAt).To(Equal(now))
}

func TestT259_BackfillScannerErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	config := registry.BackfillConfig{
		Scanner:      &fakeScanner{err: errors.New("scan failed")},
		SurfacingLog: &fakeSurfacingLog{},
		CreationLog:  &fakeCreationLog{},
		Evaluations:  &fakeEvaluations{},
		Now:          time.Now(),
	}

	_, err := registry.Backfill(config)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("scanning memories"))
}

type fakeCreationLog struct {
	times map[string]time.Time
	err   error
}

func (f *fakeCreationLog) CreationTimes() (map[string]time.Time, error) {
	if f.times == nil {
		return make(map[string]time.Time), f.err
	}

	return f.times, f.err
}

type fakeEvaluations struct {
	data map[string]registry.EvaluationCounters
	err  error
}

func (f *fakeEvaluations) AggregateEvaluations() (map[string]registry.EvaluationCounters, error) {
	if f.data == nil {
		return make(map[string]registry.EvaluationCounters), f.err
	}

	return f.data, f.err
}

// --- Test fakes ---

type fakeScanner struct {
	memories []registry.ScannedMemory
	err      error
}

func (f *fakeScanner) ScanMemories() ([]registry.ScannedMemory, error) {
	return f.memories, f.err
}

type fakeSurfacingLog struct {
	data map[string]registry.SurfacingData
	err  error
}

func (f *fakeSurfacingLog) AggregateSurfacing() (map[string]registry.SurfacingData, error) {
	if f.data == nil {
		return make(map[string]registry.SurfacingData), f.err
	}

	return f.data, f.err
}
