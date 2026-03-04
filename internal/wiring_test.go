package internal_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/catchup"
	"engram/internal/extract"
	"engram/internal/surface"
)

func TestT39_ExtractorConstructorRequiresAllDependencies(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Given an empty ExtractorConfig (all fields nil/zero)
	// When NewExtractor is called with empty config
	_, err := extract.NewExtractor(extract.Config{})
	// Then returns non-nil error containing each dependency name
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("Enricher"))
	g.Expect(err.Error()).To(ContainSubstring("Classifier"))
	g.Expect(err.Error()).To(ContainSubstring("Reconciler"))
}

func TestT41_CatchupProcessorConstructorRequiresAllDependencies(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Given an empty ProcessorConfig (all fields nil/zero)
	// When NewProcessor is called with empty config
	_, err := catchup.NewProcessor(catchup.Config{})
	// Then returns non-nil error containing each dependency name
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("Evaluator"))
	g.Expect(err.Error()).To(ContainSubstring("Reconciler"))
}

func TestT58_SurfacePipelineConstructorRequiresAllDependencies(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Given an empty SurfaceConfig (all fields nil/zero)
	// When test calls surface.NewPipeline with (SurfaceConfig{})
	_, err := surface.NewPipeline(surface.Config{})
	// Then NewPipeline returns non-nil error
	g.Expect(err).To(HaveOccurred())
	// And error message contains each dependency name: "Store", "Formatter", "Audit"
	g.Expect(err.Error()).To(ContainSubstring("Store"))
	g.Expect(err.Error()).To(ContainSubstring("Formatter"))
	g.Expect(err.Error()).To(ContainSubstring("Audit"))
}
