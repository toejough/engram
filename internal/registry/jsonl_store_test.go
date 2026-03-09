package registry_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/registry"
)

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
		ContentHash:   "hash-s",
		SurfacedCount: 5,
		Evaluations: registry.EvaluationCounters{
			Followed: 3, Contradicted: 1, Ignored: 0,
		},
	}
	target := registry.InstructionEntry{
		ID:            "target",
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
