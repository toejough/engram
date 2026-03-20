package graph_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/graph"
	"engram/internal/memory"
)

func TestTP5f1_MergeResultCarriesMergedFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	result := graph.MergeResult{
		MergedMemoryID:   "/path/mem.toml",
		AbsorbedMemoryID: "",
		MergedTitle:      "Test Title",
		MergedContent:    "merged principle text",
		MergedConceptSet: []string{"concept-a", "concept-b"},
	}

	g.Expect(result.MergedMemoryID).To(Equal("/path/mem.toml"))
	g.Expect(result.AbsorbedMemoryID).To(BeEmpty())
	g.Expect(result.MergedTitle).To(Equal("Test Title"))
	g.Expect(result.MergedContent).To(Equal("merged principle text"))
	g.Expect(result.MergedConceptSet).To(ConsistOf("concept-a", "concept-b"))
}

func TestTP5f2_AbsorbedMemoryLinksRemovedFromAllEntries(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	const (
		absorbedID = "/path/absorbed.toml"
		mergedID   = "/path/merged.toml"
	)

	mock := &mockMemoryLinker{
		records: []memory.StoredRecord{
			{
				Path: "/path/entry-a.toml",
				Record: memory.MemoryRecord{
					Title: "Entry A",
					Links: []memory.LinkRecord{
						{Target: absorbedID, Basis: "concept_overlap", Weight: 0.5},
						{Target: "/path/other.toml", Basis: "co_surfacing", Weight: 0.3},
					},
				},
			},
			{
				Path: "/path/entry-b.toml",
				Record: memory.MemoryRecord{
					Title: "Entry B",
					Links: []memory.LinkRecord{
						{Target: absorbedID, Basis: "content_similarity", Weight: 0.4},
					},
				},
			},
			{
				Path: "/path/entry-c.toml",
				Record: memory.MemoryRecord{
					Title: "Entry C",
					Links: []memory.LinkRecord{
						{Target: "/path/other.toml", Basis: "co_surfacing", Weight: 0.2},
					},
				},
			},
			{
				Path: mergedID,
				Record: memory.MemoryRecord{
					Title:   "Merged",
					Content: "merged content",
				},
			},
		},
	}

	result := graph.MergeResult{
		MergedMemoryID:   mergedID,
		AbsorbedMemoryID: absorbedID,
		MergedTitle:      "Merged",
		MergedContent:    "merged content",
		MergedConceptSet: []string{},
	}

	builder := graph.New()
	err := builder.RecomputeMergeLinks(result, mock, mock)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Entry A: absorbed link removed, co_surfacing retained
	linksA, updatedA := mock.writeCalls["/path/entry-a.toml"]
	g.Expect(updatedA).To(BeTrue())

	for _, link := range linksA {
		g.Expect(link.Target).NotTo(Equal(absorbedID))
	}

	// Entry B: absorbed link removed
	linksB, updatedB := mock.writeCalls["/path/entry-b.toml"]
	g.Expect(updatedB).To(BeTrue())
	g.Expect(linksB).To(BeEmpty())

	// Entry C: not updated (no absorbed links)
	_, updatedC := mock.writeCalls["/path/entry-c.toml"]
	g.Expect(updatedC).To(BeFalse())
}

func TestTP5f3_ConceptOverlapLinksRecomputedForMergedMemory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	const mergedID = "/path/merged.toml"

	mock := &mockMemoryLinker{
		records: []memory.StoredRecord{
			{
				Path: mergedID,
				Record: memory.MemoryRecord{
					Title:   "Memory about error handling",
					Content: "always wrap errors with context",
					Links: []memory.LinkRecord{
						{Target: "/path/stale.toml", Basis: "concept_overlap", Weight: 0.8},
						{Target: "/path/other.toml", Basis: "co_surfacing", Weight: 0.3},
					},
				},
			},
			{
				Path: "/path/similar.toml",
				Record: memory.MemoryRecord{
					Title:   "Error handling practice",
					Content: "wrap errors with context always",
				},
			},
		},
	}

	result := graph.MergeResult{
		MergedMemoryID:   mergedID,
		AbsorbedMemoryID: "",
		MergedTitle:      "Memory about error handling",
		MergedContent:    "always wrap errors with context updated",
		MergedConceptSet: []string{"error-handling"},
	}

	builder := graph.New()
	err := builder.RecomputeMergeLinks(result, mock, mock)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	mergedLinks, updated := mock.writeCalls[mergedID]
	g.Expect(updated).To(BeTrue())

	// Old stale concept_overlap link should be gone
	for _, link := range mergedLinks {
		if link.Basis == "concept_overlap" {
			g.Expect(link.Target).NotTo(Equal("/path/stale.toml"))
		}
	}

	// co_surfacing link preserved
	hasCosurfacing := false

	for _, link := range mergedLinks {
		if link.Basis == "co_surfacing" {
			hasCosurfacing = true
		}
	}

	g.Expect(hasCosurfacing).To(BeTrue())
}

func TestTP5f4_ContentSimilarityLinksRecomputedForMergedMemory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	const (
		mergedID  = "/path/merged.toml"
		similarID = "/path/similar.toml"
	)

	// Use content that will produce BM25 similarity
	mergedContent := "dependency injection interface testing unit"

	mock := &mockMemoryLinker{
		records: []memory.StoredRecord{
			{
				Path: mergedID,
				Record: memory.MemoryRecord{
					Title:   "DI Patterns",
					Content: "old content before merge",
					Links: []memory.LinkRecord{
						{Target: "/path/stale.toml", Basis: "content_similarity", Weight: 0.6},
					},
				},
			},
			{
				Path: similarID,
				Record: memory.MemoryRecord{
					Title:   "Interface Testing",
					Content: "dependency injection interface testing unit patterns",
				},
			},
		},
	}

	result := graph.MergeResult{
		MergedMemoryID:   mergedID,
		AbsorbedMemoryID: "",
		MergedTitle:      "DI Patterns",
		MergedContent:    mergedContent,
		MergedConceptSet: []string{"dependency-injection"},
	}

	builder := graph.New()
	err := builder.RecomputeMergeLinks(result, mock, mock)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	mergedLinks, updated := mock.writeCalls[mergedID]
	g.Expect(updated).To(BeTrue())

	// Old stale content_similarity link should be gone
	for _, link := range mergedLinks {
		if link.Basis == "content_similarity" {
			g.Expect(link.Target).NotTo(Equal("/path/stale.toml"))
		}
	}
}

func TestTP5f5_CoSurfacingLinksPreservedAfterRecomputation(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	const mergedID = "/path/merged.toml"

	mock := &mockMemoryLinker{
		records: []memory.StoredRecord{
			{
				Path: mergedID,
				Record: memory.MemoryRecord{
					Title:   "Memory",
					Content: "some content",
					Links: []memory.LinkRecord{
						{Target: "/path/other.toml", Basis: "co_surfacing", Weight: 0.5, CoSurfacingCount: 5},
					},
				},
			},
		},
	}

	result := graph.MergeResult{
		MergedMemoryID:   mergedID,
		AbsorbedMemoryID: "",
		MergedTitle:      "Memory",
		MergedContent:    "some content updated",
		MergedConceptSet: []string{},
	}

	builder := graph.New()
	err := builder.RecomputeMergeLinks(result, mock, mock)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	mergedLinks, updated := mock.writeCalls[mergedID]
	g.Expect(updated).To(BeTrue())

	var coSurfacingLink *memory.LinkRecord

	for i := range mergedLinks {
		if mergedLinks[i].Basis == "co_surfacing" {
			coSurfacingLink = &mergedLinks[i]

			break
		}
	}

	g.Expect(coSurfacingLink).NotTo(BeNil())

	if coSurfacingLink != nil {
		g.Expect(coSurfacingLink.Weight).To(Equal(0.5))
		g.Expect(coSurfacingLink.CoSurfacingCount).To(Equal(5))
	}
}

func TestTP5f6_EvalCorrelationLinksPreservedAfterRecomputation(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	const mergedID = "/path/merged.toml"

	mock := &mockMemoryLinker{
		records: []memory.StoredRecord{
			{
				Path: mergedID,
				Record: memory.MemoryRecord{
					Title:   "Memory",
					Content: "some content",
					Links: []memory.LinkRecord{
						{Target: "/path/other.toml", Basis: "evaluation_correlation", Weight: 0.3},
					},
				},
			},
		},
	}

	result := graph.MergeResult{
		MergedMemoryID:   mergedID,
		AbsorbedMemoryID: "",
		MergedTitle:      "Memory",
		MergedContent:    "some content updated",
		MergedConceptSet: []string{},
	}

	builder := graph.New()
	err := builder.RecomputeMergeLinks(result, mock, mock)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	mergedLinks, updated := mock.writeCalls[mergedID]
	g.Expect(updated).To(BeTrue())

	var evalLink *memory.LinkRecord

	for i := range mergedLinks {
		if mergedLinks[i].Basis == "evaluation_correlation" {
			evalLink = &mergedLinks[i]

			break
		}
	}

	g.Expect(evalLink).NotTo(BeNil())

	if evalLink != nil {
		g.Expect(evalLink.Weight).To(Equal(0.3))
	}
}

// mockMemoryLinker implements both MemoryLister and LinkWriter for tests.
type mockMemoryLinker struct {
	records    []memory.StoredRecord
	writeCalls map[string][]memory.LinkRecord
	listErr    error
	writeErr   error
}

func (m *mockMemoryLinker) ListAll() ([]memory.StoredRecord, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}

	return m.records, nil
}

func (m *mockMemoryLinker) WriteLinks(path string, links []memory.LinkRecord) error {
	if m.writeErr != nil {
		return m.writeErr
	}

	if m.writeCalls == nil {
		m.writeCalls = make(map[string][]memory.LinkRecord)
	}

	m.writeCalls[path] = links

	return nil
}
