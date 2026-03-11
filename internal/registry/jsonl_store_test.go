package registry_test

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/registry"
)

func TestJSONLStore_RemoveNotFound(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := emptyStore()

	err := store.Remove("nonexistent")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, registry.ErrNotFound)).To(BeTrue())
}

// Remove returns error when save fails after deletion.
func TestJSONLStore_RemoveSaveError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	writeCallCount := 0

	store := registry.NewJSONLStore("test.jsonl",
		registry.WithReader(func(_ string) ([]byte, error) {
			return nil, errors.New("not found")
		}),
		registry.WithWriter(func(_ string, _ []byte) error {
			writeCallCount++
			// First write (Register) succeeds, second write (Remove) fails.
			if writeCallCount >= 2 {
				return errors.New("disk full")
			}

			return nil
		}),
	)

	err := store.Register(registry.InstructionEntry{ID: "will-remove"})
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Remove("will-remove")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("writing registry")))
}

func TestT187_AbsorbedHistoryPreservesCounters(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	store := emptyStoreWithClock(func() time.Time { return now })

	source := registry.InstructionEntry{
		ID:            "src-187",
		SourceType:    "memory",
		ContentHash:   "hash-src",
		SurfacedCount: 10,
		Evaluations: registry.EvaluationCounters{
			Followed: 7, Contradicted: 2, Ignored: 1,
		},
	}
	target := registry.InstructionEntry{
		ID:            "tgt-187",
		SourceType:    "memory",
		SurfacedCount: 3,
		Evaluations: registry.EvaluationCounters{
			Followed: 2, Contradicted: 1, Ignored: 0,
		},
	}

	err := store.Register(source)
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Register(target)
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Merge("src-187", "tgt-187")
	g.Expect(err).NotTo(HaveOccurred())

	got, err := store.Get("tgt-187")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	// Target accumulated source's counters.
	g.Expect(got.SurfacedCount).To(Equal(13))
	g.Expect(got.Evaluations.Followed).To(Equal(9))
	g.Expect(got.Evaluations.Contradicted).To(Equal(3))
	g.Expect(got.Evaluations.Ignored).To(Equal(1))

	// Absorbed record preserves original source counters.
	g.Expect(got.Absorbed).To(HaveLen(1))

	absorbed := got.Absorbed[0]
	g.Expect(absorbed.From).To(Equal("src-187"))
	g.Expect(absorbed.SurfacedCount).To(Equal(10))
	g.Expect(absorbed.Evaluations.Followed).To(Equal(7))
	g.Expect(absorbed.Evaluations.Contradicted).To(Equal(2))
	g.Expect(absorbed.Evaluations.Ignored).To(Equal(1))
	g.Expect(absorbed.ContentHash).To(Equal("hash-src"))
	g.Expect(absorbed.MergedAt).To(Equal(now))
}

func TestT188_IdempotentMerge(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := emptyStore()

	source := registry.InstructionEntry{
		ID:            "src-188",
		SourceType:    "memory",
		SurfacedCount: 5,
		Evaluations: registry.EvaluationCounters{
			Followed: 3, Contradicted: 1, Ignored: 0,
		},
	}
	target := registry.InstructionEntry{
		ID:            "tgt-188",
		SourceType:    "memory",
		SurfacedCount: 3,
		Evaluations: registry.EvaluationCounters{
			Followed: 1, Contradicted: 0, Ignored: 0,
		},
	}

	err := store.Register(source)
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Register(target)
	g.Expect(err).NotTo(HaveOccurred())

	// First merge succeeds.
	err = store.Merge("src-188", "tgt-188")
	g.Expect(err).NotTo(HaveOccurred())

	// Second merge returns not-found (source already absorbed).
	err = store.Merge("src-188", "tgt-188")
	g.Expect(errors.Is(err, registry.ErrMergeNotFound)).To(BeTrue())

	// Target state unchanged from first merge.
	got, err := store.Get("tgt-188")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.SurfacedCount).To(Equal(8))
	g.Expect(got.Absorbed).To(HaveLen(1))
}

func TestT189_ConcurrentWritesSafety(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var (
		mu          sync.Mutex
		lastWritten []byte
	)

	store := registry.NewJSONLStore("test.jsonl",
		registry.WithReader(func(_ string) ([]byte, error) {
			return nil, errors.New("not found")
		}),
		registry.WithWriter(func(_ string, data []byte) error {
			mu.Lock()
			lastWritten = data
			mu.Unlock()

			return nil
		}),
	)

	// Register two entries sequentially.
	err := store.Register(registry.InstructionEntry{ID: "conc-a"})
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Register(registry.InstructionEntry{ID: "conc-b"})
	g.Expect(err).NotTo(HaveOccurred())

	// Concurrent surfacing updates.
	const concurrentUpdates = 20

	var wg sync.WaitGroup

	wg.Add(concurrentUpdates * 2)

	for range concurrentUpdates {
		go func() {
			defer wg.Done()

			_ = store.RecordSurfacing("conc-a")
		}()

		go func() {
			defer wg.Done()

			_ = store.RecordSurfacing("conc-b")
		}()
	}

	wg.Wait()

	// Both entries must exist with correct total surfacings.
	gotA, err := store.Get("conc-a")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(gotA).NotTo(BeNil())

	if gotA == nil {
		return
	}

	gotB, err := store.Get("conc-b")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(gotB).NotTo(BeNil())

	if gotB == nil {
		return
	}

	totalSurfacings := gotA.SurfacedCount + gotB.SurfacedCount
	g.Expect(totalSurfacings).To(Equal(concurrentUpdates * 2))

	// Verify written data is valid JSONL (not corrupted).
	mu.Lock()
	written := lastWritten
	mu.Unlock()
	g.Expect(written).NotTo(BeEmpty())

	lines := strings.Split(strings.TrimRight(string(written), "\n"), "\n")
	g.Expect(lines).To(HaveLen(2))
}

func TestT260_JSONLStoreRegisterAndGet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var written []byte

	store := registry.NewJSONLStore("test.jsonl",
		registry.WithReader(func(_ string) ([]byte, error) {
			return nil, errors.New("not found")
		}),
		registry.WithWriter(func(_ string, data []byte) error {
			written = data
			return nil
		}),
	)

	now := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	entry := registry.InstructionEntry{
		ID:           "test-id",
		SourceType:   "memory",
		SourcePath:   "memories/test.toml",
		Title:        "Test",
		ContentHash:  "abc123",
		RegisteredAt: now,
		UpdatedAt:    now,
	}

	err := store.Register(entry)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(written).NotTo(BeEmpty())

	got, err := store.Get("test-id")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.Title).To(Equal("Test"))
}

func TestT261_JSONLStoreRejectsDuplicateID(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := emptyStore()

	entry := registry.InstructionEntry{ID: "dup"}
	err := store.Register(entry)
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Register(entry)
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, registry.ErrDuplicateID)).To(BeTrue())
}

func TestT262_JSONLStoreRecordSurfacing(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	store := emptyStoreWithClock(func() time.Time { return now })

	entry := registry.InstructionEntry{ID: "surf-test", SurfacedCount: 2}
	err := store.Register(entry)
	g.Expect(err).NotTo(HaveOccurred())

	err = store.RecordSurfacing("surf-test")
	g.Expect(err).NotTo(HaveOccurred())

	got, err := store.Get("surf-test")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.SurfacedCount).To(Equal(3))
	g.Expect(got.LastSurfaced).NotTo(BeNil())

	if got.LastSurfaced != nil {
		g.Expect(*got.LastSurfaced).To(Equal(now))
	}
}

func TestT263_JSONLStoreRecordSurfacingNotFound(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := emptyStore()
	err := store.RecordSurfacing("nonexistent")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, registry.ErrNotFound)).To(BeTrue())
}

func TestT264_JSONLStoreRecordEvaluation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := emptyStore()

	entry := registry.InstructionEntry{ID: "eval-test"}
	err := store.Register(entry)
	g.Expect(err).NotTo(HaveOccurred())

	err = store.RecordEvaluation("eval-test", registry.Followed)
	g.Expect(err).NotTo(HaveOccurred())

	err = store.RecordEvaluation("eval-test", registry.Contradicted)
	g.Expect(err).NotTo(HaveOccurred())

	err = store.RecordEvaluation("eval-test", registry.Ignored)
	g.Expect(err).NotTo(HaveOccurred())

	got, err := store.Get("eval-test")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.Evaluations.Followed).To(Equal(1))
	g.Expect(got.Evaluations.Contradicted).To(Equal(1))
	g.Expect(got.Evaluations.Ignored).To(Equal(1))
}

func TestT265_JSONLStoreMerge(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	store := emptyStoreWithClock(func() time.Time { return now })

	source := registry.InstructionEntry{
		ID:            "source",
		SourceType:    "memory",
		ContentHash:   "hash-s",
		SurfacedCount: 5,
		Evaluations: registry.EvaluationCounters{
			Followed: 3, Contradicted: 1, Ignored: 0,
		},
	}
	target := registry.InstructionEntry{
		ID:            "target",
		SourceType:    "memory",
		SurfacedCount: 2,
		Evaluations: registry.EvaluationCounters{
			Followed: 1, Contradicted: 0, Ignored: 0,
		},
	}

	err := store.Register(source)
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Register(target)
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Merge("source", "target")
	g.Expect(err).NotTo(HaveOccurred())

	// Source should be removed
	_, err = store.Get("source")
	g.Expect(errors.Is(err, registry.ErrNotFound)).To(BeTrue())

	// Target should have absorbed counters
	got, err := store.Get("target")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.SurfacedCount).To(Equal(7))
	g.Expect(got.Evaluations.Followed).To(Equal(4))
	g.Expect(got.Absorbed).To(HaveLen(1))
	g.Expect(got.Absorbed[0].From).To(Equal("source"))
	g.Expect(got.Absorbed[0].MergedAt).To(Equal(now))
}

func TestT266_JSONLStoreMergeNotFound(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := emptyStore()

	entry := registry.InstructionEntry{ID: "only"}
	err := store.Register(entry)
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Merge("missing", "only")
	g.Expect(errors.Is(err, registry.ErrMergeNotFound)).To(BeTrue())
}

func TestT267_JSONLStoreRemove(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := emptyStore()

	entry := registry.InstructionEntry{ID: "removable"}
	err := store.Register(entry)
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Remove("removable")
	g.Expect(err).NotTo(HaveOccurred())

	_, err = store.Get("removable")
	g.Expect(errors.Is(err, registry.ErrNotFound)).To(BeTrue())
}

func TestT268_JSONLStoreList(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := emptyStore()

	err := store.Register(registry.InstructionEntry{ID: "a"})
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Register(registry.InstructionEntry{ID: "b"})
	g.Expect(err).NotTo(HaveOccurred())

	entries, err := store.List()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(HaveLen(2))
}

func TestT269_JSONLStoreLoadsExistingData(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	existing := registry.InstructionEntry{
		ID:           "existing",
		SourceType:   "memory",
		Title:        "Existing Entry",
		RegisteredAt: now,
		UpdatedAt:    now,
	}

	line, err := json.Marshal(existing)
	g.Expect(err).NotTo(HaveOccurred())

	store := registry.NewJSONLStore("test.jsonl",
		registry.WithReader(func(_ string) ([]byte, error) {
			return append(line, '\n'), nil
		}),
		registry.WithWriter(func(_ string, _ []byte) error {
			return nil
		}),
	)

	got, err := store.Get("existing")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.Title).To(Equal("Existing Entry"))
}

func TestT270_JSONLStoreSkipsMalformedLines(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data := `{"id":"good","title":"Good"}
not valid json
{"id":"also-good","title":"Also Good"}
`
	store := registry.NewJSONLStore("test.jsonl",
		registry.WithReader(func(_ string) ([]byte, error) {
			return []byte(data), nil
		}),
		registry.WithWriter(func(_ string, _ []byte) error {
			return nil
		}),
	)

	entries, err := store.List()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(HaveLen(2))
}

func TestT271_JSONLStoreBulkLoad(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var written []byte

	store := registry.NewJSONLStore("test.jsonl",
		registry.WithReader(func(_ string) ([]byte, error) {
			return nil, errors.New("not found")
		}),
		registry.WithWriter(func(_ string, data []byte) error {
			written = data
			return nil
		}),
	)

	entries := []registry.InstructionEntry{
		{ID: "bulk-1", Title: "First"},
		{ID: "bulk-2", Title: "Second"},
	}

	err := store.BulkLoad(entries)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify written JSONL has 2 lines
	lines := strings.Split(strings.TrimRight(string(written), "\n"), "\n")
	g.Expect(lines).To(HaveLen(2))

	// Verify can Get both
	got, err := store.Get("bulk-1")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.Title).To(Equal("First"))
}

// traces: T-323
func TestT323_MergeRejectsNonMemorySourceType(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := emptyStore()

	err := store.Register(registry.InstructionEntry{ID: "rule-entry", SourceType: "rule"})
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Register(registry.InstructionEntry{ID: "mem-entry", SourceType: "memory"})
	g.Expect(err).NotTo(HaveOccurred())

	// Non-memory source into memory target → rejected.
	err = store.Merge("rule-entry", "mem-entry")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, registry.ErrMergeSourceType)).To(BeTrue())

	// Memory source into non-memory target → rejected.
	err = store.Merge("mem-entry", "rule-entry")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, registry.ErrMergeSourceType)).To(BeTrue())
}

// traces: T-P0a-1
func TestTP0a1_NewEntryDefaultsEnforcementLevelToAdvisory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := emptyStore()

	err := store.Register(registry.InstructionEntry{ID: "t324", SourceType: "memory"})
	g.Expect(err).NotTo(HaveOccurred())

	got, err := store.Get("t324")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.EnforcementLevel).To(Equal(registry.EnforcementAdvisory))
}

// traces: T-P0a-2
func TestTP0a2_LoadBackfillsMissingEnforcementLevelToAdvisory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// JSONL without enforcement_level field
	data := "{\"id\":\"t325\",\"source_type\":\"memory\",\"title\":\"Old Entry\"}\n"

	store := registry.NewJSONLStore("test.jsonl",
		registry.WithReader(func(_ string) ([]byte, error) {
			return []byte(data), nil
		}),
		registry.WithWriter(func(_ string, _ []byte) error {
			return nil
		}),
	)

	got, err := store.Get("t325")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.EnforcementLevel).To(Equal(registry.EnforcementAdvisory))
}

// traces: T-P0b-1
func TestTP0b1_SetEnforcementLevelRecordsTransition(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)
	store := emptyStoreWithClock(func() time.Time { return now })

	err := store.Register(registry.InstructionEntry{ID: "p0b-1", SourceType: "memory"})
	g.Expect(err).NotTo(HaveOccurred())

	err = store.SetEnforcementLevel("p0b-1", registry.EnforcementEmphasizedAdvisory,
		"low effectiveness after 5 surfacings")
	g.Expect(err).NotTo(HaveOccurred())

	got, err := store.Get("p0b-1")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.EnforcementLevel).To(Equal(registry.EnforcementEmphasizedAdvisory))
	g.Expect(got.Transitions).To(HaveLen(1))
	g.Expect(got.Transitions[0].From).To(Equal(registry.EnforcementAdvisory))
	g.Expect(got.Transitions[0].To).To(Equal(registry.EnforcementEmphasizedAdvisory))
	g.Expect(got.Transitions[0].At).To(Equal(now))
	g.Expect(got.Transitions[0].Reason).To(Equal("low effectiveness after 5 surfacings"))
}

// traces: T-P0b-2
func TestTP0b2_SetEnforcementLevelSameLevelNoTransition(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := emptyStore()

	err := store.Register(registry.InstructionEntry{ID: "p0b-2", SourceType: "memory"})
	g.Expect(err).NotTo(HaveOccurred())

	// Setting same level (advisory → advisory) should not record a transition.
	err = store.SetEnforcementLevel("p0b-2", registry.EnforcementAdvisory, "no change")
	g.Expect(err).NotTo(HaveOccurred())

	got, err := store.Get("p0b-2")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.Transitions).To(BeEmpty())
}

// traces: T-P0b-3
func TestTP0b3_SetEnforcementLevelAccumulatesHistory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tick := 0
	times := []time.Time{
		time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 11, 9, 0, 0, 0, time.UTC),
	}
	store := emptyStoreWithClock(func() time.Time {
		current := times[tick]

		if tick < len(times)-1 {
			tick++
		}

		return current
	})

	err := store.Register(registry.InstructionEntry{ID: "p0b-3", SourceType: "memory"})
	g.Expect(err).NotTo(HaveOccurred())

	err = store.SetEnforcementLevel("p0b-3", registry.EnforcementEmphasizedAdvisory, "first escalation")
	g.Expect(err).NotTo(HaveOccurred())

	err = store.SetEnforcementLevel("p0b-3", registry.EnforcementReminder, "second escalation")
	g.Expect(err).NotTo(HaveOccurred())

	got, err := store.Get("p0b-3")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.EnforcementLevel).To(Equal(registry.EnforcementReminder))
	g.Expect(got.Transitions).To(HaveLen(2))
	g.Expect(got.Transitions[0].From).To(Equal(registry.EnforcementAdvisory))
	g.Expect(got.Transitions[0].To).To(Equal(registry.EnforcementEmphasizedAdvisory))
	g.Expect(got.Transitions[1].From).To(Equal(registry.EnforcementEmphasizedAdvisory))
	g.Expect(got.Transitions[1].To).To(Equal(registry.EnforcementReminder))
}

// traces: T-P0b-4
func TestTP0b4_TransitionsRoundTripThroughJSONL(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	transitionAt := time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)

	// Construct JSONL with an existing transition.
	entry := registry.InstructionEntry{
		ID:               "p0b-4",
		SourceType:       "memory",
		EnforcementLevel: registry.EnforcementEmphasizedAdvisory,
		Transitions: []registry.EnforcementTransition{
			{
				From:   registry.EnforcementAdvisory,
				To:     registry.EnforcementEmphasizedAdvisory,
				At:     transitionAt,
				Reason: "persisted reason",
			},
		},
	}

	line, err := json.Marshal(entry)
	g.Expect(err).NotTo(HaveOccurred())

	store := registry.NewJSONLStore("test.jsonl",
		registry.WithReader(func(_ string) ([]byte, error) {
			return append(line, '\n'), nil
		}),
		registry.WithWriter(func(_ string, _ []byte) error {
			return nil
		}),
	)

	got, err := store.Get("p0b-4")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.Transitions).To(HaveLen(1))
	g.Expect(got.Transitions[0].From).To(Equal(registry.EnforcementAdvisory))
	g.Expect(got.Transitions[0].To).To(Equal(registry.EnforcementEmphasizedAdvisory))
	g.Expect(got.Transitions[0].At).To(Equal(transitionAt))
	g.Expect(got.Transitions[0].Reason).To(Equal("persisted reason"))
}

// traces: T-P0b-5
func TestTP0b5_SetEnforcementLevelNotFound(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := emptyStore()

	err := store.SetEnforcementLevel("missing", registry.EnforcementReminder, "reason")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, registry.ErrNotFound)).To(BeTrue())
}

// traces: T-P0b-6
func TestTP0b6_SetEnforcementLevelSaveError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	writeCallCount := 0

	store := registry.NewJSONLStore("test.jsonl",
		registry.WithReader(func(_ string) ([]byte, error) {
			return nil, errors.New("not found")
		}),
		registry.WithWriter(func(_ string, _ []byte) error {
			writeCallCount++
			// First write (Register) succeeds; second write (SetEnforcementLevel) fails.
			if writeCallCount >= 2 {
				return errors.New("disk full")
			}

			return nil
		}),
	)

	err := store.Register(registry.InstructionEntry{ID: "p0b-6", SourceType: "memory"})
	g.Expect(err).NotTo(HaveOccurred())

	err = store.SetEnforcementLevel("p0b-6", registry.EnforcementEmphasizedAdvisory, "escalate")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("writing registry")))
}

// Remove returns ErrNotFound for nonexistent ID.

// --- Helpers ---

func emptyStore() *registry.JSONLStore {
	return registry.NewJSONLStore("test.jsonl",
		registry.WithReader(func(_ string) ([]byte, error) {
			return nil, errors.New("not found")
		}),
		registry.WithWriter(func(_ string, _ []byte) error {
			return nil
		}),
	)
}

func emptyStoreWithClock(clock func() time.Time) *registry.JSONLStore {
	return registry.NewJSONLStore("test.jsonl",
		registry.WithReader(func(_ string) ([]byte, error) {
			return nil, errors.New("not found")
		}),
		registry.WithWriter(func(_ string, _ []byte) error {
			return nil
		}),
		registry.WithNow(clock),
	)
}

// T-P3-13: Registry.UpdateLinks stores and retrieves links
func TestUpdateLinksStoresAndRetrieves(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := emptyStore()

	entry := registry.InstructionEntry{
		ID:         "mem1",
		SourceType: "memory",
		Title:      "test",
	}
	err := store.Register(entry)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	links := []registry.Link{
		{Target: "mem2", Weight: 0.5, Basis: "concept_overlap"},
		{Target: "mem3", Weight: 0.3, Basis: "co_surfacing", CoSurfacingCount: 2},
	}

	err = store.UpdateLinks("mem1", links)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	retrieved, err := store.Get("mem1")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(retrieved.Links).To(HaveLen(2))
	g.Expect(retrieved.Links[0].Target).To(Equal("mem2"))
	g.Expect(retrieved.Links[1].Target).To(Equal("mem3"))
}

// T-P3-14: Registry.UpdateLinks returns ErrNotFound for unknown id
func TestUpdateLinksNotFound(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := emptyStore()

	links := []registry.Link{{Target: "ex", Weight: 0.5, Basis: "co_surfacing"}}

	err := store.UpdateLinks("nonexistent", links)
	g.Expect(err).To(MatchError(ContainSubstring("not found")))
}

// T-P3-15: Registry.UpdateLinks replaces existing links entirely
func TestUpdateLinksReplaces(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := emptyStore()

	entry := registry.InstructionEntry{
		ID:         "mem1",
		SourceType: "memory",
		Title:      "test",
		Links: []registry.Link{
			{Target: "old1", Weight: 0.3, Basis: "concept_overlap"},
			{Target: "old2", Weight: 0.2, Basis: "concept_overlap"},
			{Target: "old3", Weight: 0.1, Basis: "concept_overlap"},
		},
	}
	err := store.Register(entry)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	// Replace with single new link
	newLinks := []registry.Link{{Target: "new1", Weight: 0.5, Basis: "co_surfacing"}}
	err = store.UpdateLinks("mem1", newLinks)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	retrieved, err := store.Get("mem1")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(retrieved.Links).To(HaveLen(1))
	g.Expect(retrieved.Links[0].Target).To(Equal("new1"))
}
