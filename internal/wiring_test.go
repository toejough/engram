//go:build integration

package internal_test

// Tests for ARCH-8: DI Wiring
// Integration tests — verifies no silent nil degradation.
// Won't compile yet — RED phase.

import (
	"testing"

	"engram/internal/catchup"
	"engram/internal/correct"
	"engram/internal/extract"
	"github.com/onsi/gomega"
)

// T-39: Constructing an Extractor with nil for any required dependency returns an error.
func TestWiring_ExtractorHasAllDependencies(t *testing.T) {
	g := gomega.NewWithT(t)

	_, err := extract.NewExtractor(extract.ExtractorConfig{})
	g.Expect(err).To(gomega.HaveOccurred())

	errMsg := err.Error()
	for _, dep := range []string{"Enricher", "Gate", "Classifier", "Reconciler", "Session", "Audit"} {
		g.Expect(errMsg).To(gomega.ContainSubstring(dep))
	}
}

// T-40: Constructing a CorrectionDetector with nil for any required dependency returns an error.
func TestWiring_CorrectionDetectorHasAllDependencies(t *testing.T) {
	g := gomega.NewWithT(t)

	_, err := correct.NewDetector(correct.DetectorConfig{})
	g.Expect(err).To(gomega.HaveOccurred())

	errMsg := err.Error()
	for _, dep := range []string{"Corpus", "Recon", "Session", "Audit"} {
		g.Expect(errMsg).To(gomega.ContainSubstring(dep))
	}
}

// T-41: Constructing a CatchupProcessor with nil for any required dependency returns an error.
func TestWiring_CatchupProcessorHasAllDependencies(t *testing.T) {
	g := gomega.NewWithT(t)

	_, err := catchup.NewProcessor(catchup.ProcessorConfig{})
	g.Expect(err).To(gomega.HaveOccurred())

	errMsg := err.Error()
	for _, dep := range []string{"Evaluator", "Reconciler", "Corpus", "Session", "Audit"} {
		g.Expect(errMsg).To(gomega.ContainSubstring(dep))
	}
}
