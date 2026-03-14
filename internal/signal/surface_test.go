package signal_test

import (
	"encoding/json"
	"testing"

	"github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/signal"
)

func TestFormatContext_Empty(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	result, err := signal.FormatContext(nil)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(gomega.BeEmpty())
}

func TestFormatContext_InstructionsByKind(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	enriched := []signal.EnrichedSignal{
		{SourceID: "a.toml", SignalKind: signal.KindNoiseRemoval},
		{SourceID: "b.toml", SignalKind: signal.KindLeechRewrite},
		{SourceID: "c.toml", SignalKind: signal.KindHiddenGemBroaden},
		{
			SourceID:   "e.toml",
			SignalKind: signal.KindGraduation,
			Summary:    "Consider promoting this skill to CLAUDE.md — it has met the promotion threshold",
		},
		{
			SourceID:   "f.toml",
			SignalKind: signal.KindGraduation,
			Summary:    "Consider demoting this CLAUDE.md entry to a skill — it has not been frequently needed",
		},
	}

	result, err := signal.FormatContext(enriched)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(gomega.ContainSubstring("Recommend removal"))
	g.Expect(result).To(gomega.ContainSubstring("Recommend rewrite"))
	g.Expect(result).To(gomega.ContainSubstring("Recommend broadening"))
	g.Expect(result).To(gomega.ContainSubstring("promoting this skill to CLAUDE.md"))
	g.Expect(result).To(gomega.ContainSubstring("demoting this CLAUDE.md entry"))
}

func TestFormatContext_WithSignals(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	enriched := []signal.EnrichedSignal{{
		Type:       signal.TypeMaintain,
		SourceID:   "test.toml",
		SignalKind: signal.KindNoiseRemoval,
		Summary:    "remove it",
		Title:      "Test",
	}}

	result, err := signal.FormatContext(enriched)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(gomega.BeEmpty())

	var parsed map[string]json.RawMessage

	g.Expect(json.Unmarshal([]byte(result), &parsed)).To(gomega.Succeed())
	g.Expect(parsed).To(gomega.HaveKey("signals"))
	g.Expect(parsed).To(gomega.HaveKey("instructions"))
}

func TestSurfacer_EnrichesWithMemoryContent(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	loader := &stubLoader{
		memories: map[string]*memory.Stored{
			"test.toml": {
				Title:     "Test Memory",
				Principle: "Always test",
				Keywords:  []string{"testing", "tdd"},
			},
		},
	}

	surfacer := signal.NewSurfacer(signal.WithLoader(loader))
	signals := []signal.Signal{{
		Type:       signal.TypeMaintain,
		SourceID:   "test.toml",
		SignalKind: signal.KindLeechRewrite,
		Quadrant:   "Leech",
		Summary:    "needs rewrite",
	}}

	enriched, err := surfacer.Surface(signals)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(enriched).To(gomega.HaveLen(1))
	g.Expect(enriched[0].Title).To(gomega.Equal("Test Memory"))
	g.Expect(enriched[0].Principle).To(gomega.Equal("Always test"))
	g.Expect(enriched[0].Keywords).To(gomega.Equal("testing, tdd"))
}

func TestSurfacer_MissingMemoryStillEnriches(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	loader := &stubLoader{memories: map[string]*memory.Stored{}}
	surfacer := signal.NewSurfacer(signal.WithLoader(loader))

	signals := []signal.Signal{{
		Type:       signal.TypeMaintain,
		SourceID:   "missing.toml",
		SignalKind: signal.KindGraduation,
		Summary:    "graduated",
	}}

	enriched, err := surfacer.Surface(signals)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(enriched).To(gomega.HaveLen(1))
	g.Expect(enriched[0].Title).To(gomega.BeEmpty())
	g.Expect(enriched[0].SignalKind).To(gomega.Equal(signal.KindGraduation))
}

func TestSurfacer_NoLoader(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	surfacer := signal.NewSurfacer()
	signals := []signal.Signal{{
		Type:       signal.TypeMaintain,
		SourceID:   "test.toml",
		SignalKind: signal.KindNoiseRemoval,
		Summary:    "noise",
	}}

	enriched, err := surfacer.Surface(signals)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(enriched).To(gomega.HaveLen(1))
	g.Expect(enriched[0].SourceID).To(gomega.Equal("test.toml"))
}

type stubLoader struct {
	memories map[string]*memory.Stored
}

func (s *stubLoader) Load(path string) (*memory.Stored, error) {
	mem, ok := s.memories[path]
	if !ok {
		return nil, nil //nolint:nilnil // missing memory returns nil
	}

	return mem, nil
}
