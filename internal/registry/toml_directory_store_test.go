package registry_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/registry"
)

// traces: issue-310
func TestIssue310_RecordSurfacingWithAbsolutePath(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fixedTime := time.Date(2026, 3, 16, 18, 0, 0, 0, time.UTC)
	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, func() time.Time { return fixedTime })

	entry := registry.InstructionEntry{
		ID:            "memories/abs-path.toml",
		SourceType:    registry.SourceTypeMemory,
		Title:         "Absolute Path Test",
		SurfacedCount: 0,
	}

	err := store.Register(entry)
	g.Expect(err).NotTo(HaveOccurred())

	// Pass absolute path (as surface.go does with mem.FilePath).
	// The store dataDir is /testdata, so the absolute path is
	// /testdata/memories/abs-path.toml.
	err = store.RecordSurfacing("/testdata/memories/abs-path.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	got, err := store.Get("memories/abs-path.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.SurfacedCount).To(Equal(1))
}

// traces: issue-310
func TestIssue310_GetWithAbsolutePath(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, nil)

	err := store.Register(registry.InstructionEntry{
		ID:         "memories/abs-get.toml",
		SourceType: registry.SourceTypeMemory,
		Title:      "Abs Get",
	})
	g.Expect(err).NotTo(HaveOccurred())

	got, err := store.Get("/testdata/memories/abs-get.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.Title).To(Equal("Abs Get"))
	g.Expect(got.ID).To(Equal("memories/abs-get.toml"))
}

// traces: issue-310
func TestIssue310_RegisterWithAbsolutePath(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, nil)

	err := store.Register(registry.InstructionEntry{
		ID:         "/testdata/memories/abs-reg.toml",
		SourceType: registry.SourceTypeMemory,
		Title:      "Abs Register",
	})
	g.Expect(err).NotTo(HaveOccurred())

	// Should be retrievable via relative ID.
	got, err := store.Get("memories/abs-reg.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.Title).To(Equal("Abs Register"))
	g.Expect(got.ID).To(Equal("memories/abs-reg.toml"))
}

// traces: issue-310
func TestIssue310_RemoveWithAbsolutePath(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, nil)

	err := store.Register(registry.InstructionEntry{
		ID:         "memories/abs-rm.toml",
		SourceType: registry.SourceTypeMemory,
		Title:      "Abs Remove",
	})
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Remove("/testdata/memories/abs-rm.toml")
	g.Expect(err).NotTo(HaveOccurred())

	_, err = store.Get("memories/abs-rm.toml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, registry.ErrNotFound)).To(BeTrue())
}

// traces: issue-310
func TestIssue310_MergeWithAbsolutePaths(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mergeTime := time.Date(2026, 3, 16, 18, 0, 0, 0, time.UTC)
	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, func() time.Time { return mergeTime })

	err := store.Register(registry.InstructionEntry{
		ID:            "memories/abs-src.toml",
		SourceType:    registry.SourceTypeMemory,
		Title:         "Abs Source",
		SurfacedCount: 3,
	})
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Register(registry.InstructionEntry{
		ID:            "memories/abs-tgt.toml",
		SourceType:    registry.SourceTypeMemory,
		Title:         "Abs Target",
		SurfacedCount: 2,
	})
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Merge(
		"/testdata/memories/abs-src.toml",
		"/testdata/memories/abs-tgt.toml",
	)
	g.Expect(err).NotTo(HaveOccurred())

	got, err := store.Get("memories/abs-tgt.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.SurfacedCount).To(Equal(5))
}

// --- T-238: Register writes new TOML with zero metrics ---

func TestT238_TOMLDirectoryStore_Register(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, nil)

	entry := registry.InstructionEntry{
		ID:          "memories/test.toml",
		SourceType:  registry.SourceTypeMemory,
		Title:       "Test Memory",
		Content:     "Always use targ.",
		ContentHash: "sha256:abc",
	}

	err := store.Register(entry)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	got, err := store.Get("memories/test.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.Title).To(Equal("Test Memory"))
	g.Expect(got.SurfacedCount).To(Equal(0))
	g.Expect(got.Evaluations.Followed).To(Equal(0))
	g.Expect(got.Evaluations.Contradicted).To(Equal(0))
	g.Expect(got.Evaluations.Ignored).To(Equal(0))
	g.Expect(got.EnforcementLevel).To(Equal(registry.EnforcementAdvisory))
}

// traces: T-238
func TestT238b_TOMLDirectoryStore_RegisterRejectsDuplicate(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, nil)

	entry := registry.InstructionEntry{
		ID:         "memories/dup.toml",
		SourceType: registry.SourceTypeMemory,
		Title:      "Duplicate",
	}

	err := store.Register(entry)
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Register(entry)
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, registry.ErrDuplicateID)).To(BeTrue())
}

// --- T-239: RecordSurfacing increments surfaced_count, sets last_surfaced_at ---

func TestT239_TOMLDirectoryStore_RecordSurfacing(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fixedTime := time.Date(2026, 3, 10, 14, 30, 0, 0, time.UTC)
	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, func() time.Time { return fixedTime })

	entry := registry.InstructionEntry{
		ID:            "memories/surf.toml",
		SourceType:    registry.SourceTypeMemory,
		Title:         "Surf Test",
		SurfacedCount: 5,
	}

	err := store.Register(entry)
	g.Expect(err).NotTo(HaveOccurred())

	err = store.RecordSurfacing("memories/surf.toml")
	g.Expect(err).NotTo(HaveOccurred())

	got, err := store.Get("memories/surf.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.SurfacedCount).To(Equal(6))
	g.Expect(got.LastSurfaced).NotTo(BeNil())

	if got.LastSurfaced != nil {
		g.Expect(*got.LastSurfaced).To(Equal(fixedTime))
	}
}

// traces: T-239
func TestT239b_TOMLDirectoryStore_RecordSurfacingNotFound(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, nil)

	err := store.RecordSurfacing("memories/nonexistent.toml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, registry.ErrNotFound)).To(BeTrue())
}

// --- T-240: RecordEvaluation increments correct counter only ---

func TestT240_TOMLDirectoryStore_RecordEvaluation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, nil)

	entry := registry.InstructionEntry{
		ID:         "memories/eval.toml",
		SourceType: registry.SourceTypeMemory,
		Title:      "Eval Test",
		Evaluations: registry.EvaluationCounters{
			Followed: 3, Contradicted: 1, Ignored: 2,
		},
	}

	err := store.Register(entry)
	g.Expect(err).NotTo(HaveOccurred())

	err = store.RecordEvaluation("memories/eval.toml", registry.Followed)
	g.Expect(err).NotTo(HaveOccurred())

	got, err := store.Get("memories/eval.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.Evaluations.Followed).To(Equal(4))
	g.Expect(got.Evaluations.Contradicted).To(Equal(1))
	g.Expect(got.Evaluations.Ignored).To(Equal(2))
}

// --- RecordEvaluation: Contradicted and Ignored branches ---

func TestT240b_TOMLDirectoryStore_RecordEvaluationContradicted(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, nil)

	err := store.Register(registry.InstructionEntry{
		ID:         "memories/eval-c.toml",
		SourceType: registry.SourceTypeMemory,
		Title:      "Eval Contradicted",
	})
	g.Expect(err).NotTo(HaveOccurred())

	err = store.RecordEvaluation("memories/eval-c.toml", registry.Contradicted)
	g.Expect(err).NotTo(HaveOccurred())

	got, err := store.Get("memories/eval-c.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.Evaluations.Contradicted).To(Equal(1))
}

func TestT240c_TOMLDirectoryStore_RecordEvaluationIgnored(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, nil)

	err := store.Register(registry.InstructionEntry{
		ID:         "memories/eval-i.toml",
		SourceType: registry.SourceTypeMemory,
		Title:      "Eval Ignored",
	})
	g.Expect(err).NotTo(HaveOccurred())

	err = store.RecordEvaluation("memories/eval-i.toml", registry.Ignored)
	g.Expect(err).NotTo(HaveOccurred())

	got, err := store.Get("memories/eval-i.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.Evaluations.Ignored).To(Equal(1))
}

// --- T-241: List scans directory, returns all entries ---

func TestT241_TOMLDirectoryStore_List(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, nil)

	for _, id := range []string{
		"memories/a.toml",
		"memories/b.toml",
		"memories/c.toml",
	} {
		err := store.Register(registry.InstructionEntry{
			ID:         id,
			SourceType: registry.SourceTypeMemory,
			Title:      id,
		})
		g.Expect(err).NotTo(HaveOccurred())
	}

	entries, err := store.List()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(HaveLen(3))
}

// traces: T-241
func TestT241b_TOMLDirectoryStore_ListEmptyDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, nil)

	entries, err := store.List()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(BeEmpty())
}

// --- T-242: Get reads single memory by path ---

func TestT242_TOMLDirectoryStore_Get(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()

	// Pre-populate a TOML file directly.
	mfs.setFile("/testdata/memories/foo.toml", []byte(
		"title = \"Foo Memory\"\nsource_type = \"memory\"\nsurfaced_count = 10\n"+
			"followed_count = 5\ncontradicted_count = 2\nignored_count = 3\n",
	))

	store := newTOMLUnitStore(mfs, nil)

	got, err := store.Get("memories/foo.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.ID).To(Equal("memories/foo.toml"))
	g.Expect(got.SurfacedCount).To(Equal(10))
	g.Expect(got.Evaluations.Followed).To(Equal(5))
	g.Expect(got.Evaluations.Contradicted).To(Equal(2))
	g.Expect(got.Evaluations.Ignored).To(Equal(3))
}

// traces: T-242
func TestT242b_TOMLDirectoryStore_GetNotFound(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, nil)

	_, err := store.Get("memories/missing.toml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, registry.ErrNotFound)).To(BeTrue())
}

// --- T-243: Remove deletes TOML file, subsequent Get returns ErrNotFound ---

func TestT243_TOMLDirectoryStore_Remove(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, nil)

	entry := registry.InstructionEntry{
		ID:         "memories/removable.toml",
		SourceType: registry.SourceTypeMemory,
		Title:      "Removable",
	}

	err := store.Register(entry)
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Remove("memories/removable.toml")
	g.Expect(err).NotTo(HaveOccurred())

	_, err = store.Get("memories/removable.toml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, registry.ErrNotFound)).To(BeTrue())
}

// traces: T-243
func TestT243b_TOMLDirectoryStore_RemoveNotFound(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, nil)

	err := store.Remove("memories/nonexistent.toml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, registry.ErrNotFound)).To(BeTrue())
}

// --- T-244: Merge absorbs source metrics into target, deletes source ---

func TestT244_TOMLDirectoryStore_Merge(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mergeTime := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, func() time.Time { return mergeTime })

	source := registry.InstructionEntry{
		ID:            "memories/source.toml",
		SourceType:    registry.SourceTypeMemory,
		Title:         "Source Memory",
		SurfacedCount: 10,
		ContentHash:   "sha256:src",
		Evaluations: registry.EvaluationCounters{
			Followed: 8, Contradicted: 1, Ignored: 1,
		},
	}
	target := registry.InstructionEntry{
		ID:            "memories/target.toml",
		SourceType:    registry.SourceTypeMemory,
		Title:         "Target Memory",
		SurfacedCount: 5,
		Evaluations: registry.EvaluationCounters{
			Followed: 3, Contradicted: 0, Ignored: 0,
		},
	}

	err := store.Register(source)
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Register(target)
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Merge("memories/source.toml", "memories/target.toml")
	g.Expect(err).NotTo(HaveOccurred())

	// Source must be gone.
	_, err = store.Get("memories/source.toml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, registry.ErrNotFound)).To(BeTrue())

	// Target has accumulated source counters.
	got, err := store.Get("memories/target.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.SurfacedCount).To(Equal(15))
	g.Expect(got.Evaluations.Followed).To(Equal(11))
	g.Expect(got.Evaluations.Contradicted).To(Equal(1))
	g.Expect(got.Evaluations.Ignored).To(Equal(1))
	g.Expect(got.Absorbed).To(HaveLen(1))
	g.Expect(got.Absorbed[0].From).To(Equal("memories/source.toml"))
	g.Expect(got.Absorbed[0].SurfacedCount).To(Equal(10))
	g.Expect(got.Absorbed[0].ContentHash).To(Equal("sha256:src"))
	g.Expect(got.Absorbed[0].Evaluations.Followed).To(Equal(8))
	g.Expect(got.Absorbed[0].MergedAt).To(Equal(mergeTime))
}

// traces: T-244
func TestT244b_TOMLDirectoryStore_MergeNotFound(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, nil)

	err := store.Register(registry.InstructionEntry{
		ID: "memories/only.toml", SourceType: registry.SourceTypeMemory,
	})
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Merge("memories/missing.toml", "memories/only.toml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, registry.ErrMergeNotFound)).To(BeTrue())
}

// --- Merge: non-memory source type rejection ---

func TestT244c_TOMLDirectoryStore_MergeRejectsNonMemorySource(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, nil)

	// Create a source with non-memory source_type via raw TOML.
	mfs.setFile("/testdata/memories/skill-src.toml", []byte(
		"title = \"Skill Source\"\nsource_type = \"skill\"\n",
	))

	err := store.Register(registry.InstructionEntry{
		ID:         "memories/target.toml",
		SourceType: registry.SourceTypeMemory,
		Title:      "Target",
	})
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Merge("memories/skill-src.toml", "memories/target.toml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, registry.ErrMergeSourceType)).To(BeTrue())
}

// --- Merge: target not found ---

func TestT244d_TOMLDirectoryStore_MergeTargetNotFound(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, nil)

	err := store.Register(registry.InstructionEntry{
		ID:         "memories/source.toml",
		SourceType: registry.SourceTypeMemory,
		Title:      "Source",
	})
	g.Expect(err).NotTo(HaveOccurred())

	err = store.Merge("memories/source.toml", "memories/no-target.toml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, registry.ErrMergeNotFound)).To(BeTrue())
}

// --- T-245: Missing metrics fields default to zero ---

func TestT245_TOMLDirectoryStore_MissingMetricsDefaultToZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()

	// TOML with only content fields, no metric fields.
	mfs.setFile("/testdata/memories/no-metrics.toml", []byte(
		"title = \"Content Only\"\ncontent = \"Some content\"\n",
	))

	store := newTOMLUnitStore(mfs, nil)

	got, err := store.Get("memories/no-metrics.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.SurfacedCount).To(Equal(0))
	g.Expect(got.Evaluations.Followed).To(Equal(0))
	g.Expect(got.Evaluations.Contradicted).To(Equal(0))
	g.Expect(got.Evaluations.Ignored).To(Equal(0))
	g.Expect(got.LastSurfaced).To(BeNil())
}

// --- T-246: Per-file locking serializes 10 concurrent writes to same file ---

func TestT246_TOMLDirectoryStore_ConcurrentWritesSameFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	store := registry.NewTOMLDirectoryStore(dataDir)

	entry := registry.InstructionEntry{
		ID:         "memories/concurrent.toml",
		SourceType: registry.SourceTypeMemory,
		Title:      "Concurrent Test",
	}

	err := store.Register(entry)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	const goroutines = 10

	var wg sync.WaitGroup

	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			_ = store.RecordSurfacing("memories/concurrent.toml")
		}()
	}

	wg.Wait()

	got, err := store.Get("memories/concurrent.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.SurfacedCount).To(Equal(goroutines))
}

// --- T-247: Concurrent writes to different files don't block ---

func TestT247_TOMLDirectoryStore_ConcurrentWritesDifferentFiles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	store := registry.NewTOMLDirectoryStore(dataDir)

	for _, id := range []string{"memories/file-a.toml", "memories/file-b.toml"} {
		err := store.Register(registry.InstructionEntry{
			ID:         id,
			SourceType: registry.SourceTypeMemory,
			Title:      id,
		})
		g.Expect(err).NotTo(HaveOccurred())
	}

	const writesPerFile = 5

	var wg sync.WaitGroup

	wg.Add(writesPerFile * 2)

	for range writesPerFile {
		go func() {
			defer wg.Done()

			_ = store.RecordSurfacing("memories/file-a.toml")
		}()

		go func() {
			defer wg.Done()

			_ = store.RecordSurfacing("memories/file-b.toml")
		}()
	}

	wg.Wait()

	gotA, err := store.Get("memories/file-a.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(gotA).NotTo(BeNil())

	if gotA == nil {
		return
	}

	gotB, err := store.Get("memories/file-b.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(gotB).NotTo(BeNil())

	if gotB == nil {
		return
	}

	g.Expect(gotA.SurfacedCount).To(Equal(writesPerFile))
	g.Expect(gotB.SurfacedCount).To(Equal(writesPerFile))
}

// --- T-248: Atomic write — failed rename leaves original unchanged ---

func TestT248_TOMLDirectoryStore_AtomicWriteFailedRename(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()

	// Inject failing rename only for the second call (during RecordSurfacing).
	// First call is Register's writeAtomic.
	renameCount := 0
	failingRename := func(oldpath, newpath string) error {
		renameCount++
		if renameCount > 1 {
			return errors.New("simulated rename failure")
		}

		return mfs.rename(oldpath, newpath)
	}

	store := registry.NewTOMLDirectoryStore("/testdata",
		registry.WithTOMLReadFile(mfs.readFile),
		registry.WithTOMLWriteFile(mfs.writeFile),
		registry.WithTOMLReadDir(mfs.readDir),
		registry.WithTOMLRemove(mfs.remove),
		registry.WithTOMLRename(failingRename),
		registry.WithTOMLMkdirAll(func(_ string, _ os.FileMode) error { return nil }),
		registry.WithTOMLOpenFile(noopOpenFile),
		registry.WithTOMLLockFile(noopLock),
		registry.WithTOMLUnlockFile(noopLock),
	)

	entry := registry.InstructionEntry{
		ID:            "memories/atomic.toml",
		SourceType:    registry.SourceTypeMemory,
		Title:         "Atomic Test",
		SurfacedCount: 0,
	}

	err := store.Register(entry)
	g.Expect(err).NotTo(HaveOccurred())

	// RecordSurfacing's rename fails — original should be unchanged.
	err = store.RecordSurfacing("memories/atomic.toml")
	g.Expect(err).To(HaveOccurred())

	got, err := store.Get("memories/atomic.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.SurfacedCount).To(Equal(0))

	// No temp file should remain.
	_, readErr := mfs.readFile("/testdata/memories/atomic.toml.tmp")
	g.Expect(readErr).To(HaveOccurred())
}

// --- Register with full fields exercises entryToRecord, entryAbsorbedToRecord,
// entryTransitionsToRecord ---

func TestTOMLDirectoryStore_RegisterWithFullFields(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, nil)

	regTime := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	updTime := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	surfTime := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	mergeTime := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	transTime := time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC)

	entry := registry.InstructionEntry{
		ID:               "memories/full.toml",
		SourceType:       registry.SourceTypeMemory,
		Title:            "Full Fields",
		Content:          "Test content",
		ContentHash:      "sha256:full",
		SurfacedCount:    20,
		RegisteredAt:     regTime,
		UpdatedAt:        updTime,
		LastSurfaced:     &surfTime,
		EnforcementLevel: registry.EnforcementEmphasizedAdvisory,
		Evaluations: registry.EvaluationCounters{
			Followed: 15, Contradicted: 3, Ignored: 2,
		},
		Links: []registry.Link{
			{Target: "memories/related.toml", Weight: 0.9, Basis: "co-surfacing"},
		},
		Absorbed: []registry.AbsorbedRecord{
			{
				From:          "memories/old.toml",
				SurfacedCount: 5,
				ContentHash:   "sha256:old",
				MergedAt:      mergeTime,
				Evaluations: registry.EvaluationCounters{
					Followed: 4, Contradicted: 1, Ignored: 0,
				},
			},
		},
		Transitions: []registry.EnforcementTransition{
			{
				From:   registry.EnforcementAdvisory,
				To:     registry.EnforcementEmphasizedAdvisory,
				At:     transTime,
				Reason: "low effectiveness",
			},
		},
	}

	err := store.Register(entry)
	g.Expect(err).NotTo(HaveOccurred())

	got, err := store.Get("memories/full.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	// Verify all fields round-tripped.
	g.Expect(got.Title).To(Equal("Full Fields"))
	g.Expect(got.SurfacedCount).To(Equal(20))
	g.Expect(got.LastSurfaced).NotTo(BeNil())
	g.Expect(got.RegisteredAt).To(Equal(regTime))
	g.Expect(got.UpdatedAt).To(Equal(updTime))
	g.Expect(got.EnforcementLevel).To(
		Equal(registry.EnforcementEmphasizedAdvisory),
	)
	g.Expect(got.Links).To(HaveLen(1))
	g.Expect(got.Links[0].Target).To(Equal("memories/related.toml"))
	g.Expect(got.Absorbed).To(HaveLen(1))
	g.Expect(got.Absorbed[0].From).To(Equal("memories/old.toml"))
	g.Expect(got.Absorbed[0].SurfacedCount).To(Equal(5))
	g.Expect(got.Absorbed[0].MergedAt).To(Equal(mergeTime))
	g.Expect(got.Transitions).To(HaveLen(1))
	g.Expect(got.Transitions[0].From).To(
		Equal(registry.EnforcementAdvisory),
	)
	g.Expect(got.Transitions[0].To).To(
		Equal(registry.EnforcementEmphasizedAdvisory),
	)
	g.Expect(got.Transitions[0].At).To(Equal(transTime))
}

// --- SetEnforcementLevel ---

func TestTOMLDirectoryStore_SetEnforcementLevel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	transitionTime := time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)
	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, func() time.Time { return transitionTime })

	err := store.Register(registry.InstructionEntry{
		ID:         "memories/enforce.toml",
		SourceType: registry.SourceTypeMemory,
		Title:      "Enforce Test",
	})
	g.Expect(err).NotTo(HaveOccurred())

	err = store.SetEnforcementLevel(
		"memories/enforce.toml",
		registry.EnforcementEmphasizedAdvisory,
		"low effectiveness",
	)
	g.Expect(err).NotTo(HaveOccurred())

	got, err := store.Get("memories/enforce.toml")
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
	g.Expect(got.Transitions[0].At).To(Equal(transitionTime))
	g.Expect(got.Transitions[0].Reason).To(Equal("low effectiveness"))
}

// --- UpdateLinks ---

func TestTOMLDirectoryStore_UpdateLinks(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, nil)

	entry := registry.InstructionEntry{
		ID:         "memories/links.toml",
		SourceType: registry.SourceTypeMemory,
		Title:      "Links Test",
	}

	err := store.Register(entry)
	g.Expect(err).NotTo(HaveOccurred())

	links := []registry.Link{
		{Target: "memories/other.toml", Weight: 0.8, Basis: "co-surfacing", CoSurfacingCount: 5},
	}

	err = store.UpdateLinks("memories/links.toml", links)
	g.Expect(err).NotTo(HaveOccurred())

	got, err := store.Get("memories/links.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.Links).To(HaveLen(1))
	g.Expect(got.Links[0].Target).To(Equal("memories/other.toml"))
	g.Expect(got.Links[0].Weight).To(Equal(0.8))
	g.Expect(got.Links[0].CoSurfacingCount).To(Equal(5))
}

// --- UpdateLinks: empty links clears ---

func TestTOMLDirectoryStore_UpdateLinksEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mfs := newTOMLMemFS()
	store := newTOMLUnitStore(mfs, nil)

	err := store.Register(registry.InstructionEntry{
		ID:         "memories/clearlinks.toml",
		SourceType: registry.SourceTypeMemory,
		Title:      "Clear Links",
	})
	g.Expect(err).NotTo(HaveOccurred())

	// First set some links.
	err = store.UpdateLinks("memories/clearlinks.toml", []registry.Link{
		{Target: "memories/x.toml", Weight: 0.5, Basis: "test"},
	})
	g.Expect(err).NotTo(HaveOccurred())

	// Clear them with empty slice.
	err = store.UpdateLinks("memories/clearlinks.toml", nil)
	g.Expect(err).NotTo(HaveOccurred())

	got, err := store.Get("memories/clearlinks.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).NotTo(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.Links).To(BeEmpty())
}

// unexported variables.
var (
	errNoFileInfo = errors.New("no file info available")
)

// tomlMemDirEntry is a minimal os.DirEntry implementation for in-memory tests.
type tomlMemDirEntry struct {
	name string
}

func (e tomlMemDirEntry) Info() (os.FileInfo, error) { return nil, errNoFileInfo }

func (e tomlMemDirEntry) IsDir() bool { return false }

func (e tomlMemDirEntry) Name() string { return e.name }

func (e tomlMemDirEntry) Type() os.FileMode { return 0 }

// tomlMemFS is an in-memory filesystem for unit tests.
type tomlMemFS struct {
	mu    sync.Mutex
	files map[string][]byte
}

func (fs *tomlMemFS) readDir(name string) ([]os.DirEntry, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	var entries []os.DirEntry

	for path := range fs.files {
		if filepath.Dir(path) == name && strings.HasSuffix(path, ".toml") {
			entries = append(entries, tomlMemDirEntry{name: filepath.Base(path)})
		}
	}

	return entries, nil
}

func (fs *tomlMemFS) readFile(name string) ([]byte, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, ok := fs.files[name]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", name)
	}

	return append([]byte(nil), data...), nil
}

func (fs *tomlMemFS) remove(name string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, ok := fs.files[name]; !ok {
		return fmt.Errorf("file not found: %s", name)
	}

	delete(fs.files, name)

	return nil
}

func (fs *tomlMemFS) rename(oldpath, newpath string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, ok := fs.files[oldpath]
	if !ok {
		return fmt.Errorf("file not found: %s", oldpath)
	}

	fs.files[newpath] = data
	delete(fs.files, oldpath)

	return nil
}

func (fs *tomlMemFS) setFile(path string, data []byte) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.files[path] = data
}

func (fs *tomlMemFS) writeFile(name string, data []byte, _ os.FileMode) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.files[name] = append([]byte(nil), data...)

	return nil
}

func newTOMLMemFS() *tomlMemFS {
	return &tomlMemFS{files: make(map[string][]byte)}
}

// newTOMLUnitStore creates a TOMLDirectoryStore backed by the given memFS.
// clock may be nil for tests that don't depend on time.
func newTOMLUnitStore(mfs *tomlMemFS, clock func() time.Time) *registry.TOMLDirectoryStore {
	opts := []registry.TOMLDirOption{
		registry.WithTOMLReadFile(mfs.readFile),
		registry.WithTOMLWriteFile(mfs.writeFile),
		registry.WithTOMLReadDir(mfs.readDir),
		registry.WithTOMLRemove(mfs.remove),
		registry.WithTOMLRename(mfs.rename),
		registry.WithTOMLMkdirAll(func(_ string, _ os.FileMode) error { return nil }),
		registry.WithTOMLOpenFile(noopOpenFile),
		registry.WithTOMLLockFile(noopLock),
		registry.WithTOMLUnlockFile(noopLock),
	}

	if clock != nil {
		opts = append(opts, registry.WithTOMLClock(clock))
	}

	return registry.NewTOMLDirectoryStore("/testdata", opts...)
}

// noopLock is a no-op lock/unlock function.
func noopLock(_ *os.File) error { return nil }

// --- Helpers ---

// noopOpenFile opens /dev/null for use as a no-op lock file descriptor.
func noopOpenFile(_ string, _ int, _ os.FileMode) (*os.File, error) {
	return os.OpenFile(os.DevNull, os.O_RDONLY, 0)
}
