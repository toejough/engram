package policy_test

import (
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/policy"
)

func TestDefaults_ReturnsAllFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	pol := policy.Defaults()

	g.Expect(pol.DetectFastPathKeywords).NotTo(BeEmpty())
	g.Expect(pol.ContextByteBudget).To(BeNumerically(">", 0))
	g.Expect(pol.ContextToolArgsTruncate).To(BeNumerically(">", 0))
	g.Expect(pol.ContextToolResultTruncate).To(BeNumerically(">", 0))
	g.Expect(pol.ExtractCandidateCountMin).To(BeNumerically(">", 0))
	g.Expect(pol.ExtractCandidateCountMax).To(BeNumerically(">", pol.ExtractCandidateCountMin))
	g.Expect(pol.ExtractBM25Threshold).To(BeNumerically(">", 0))
	g.Expect(pol.DetectHaikuPrompt).NotTo(BeEmpty())
	g.Expect(pol.ExtractSonnetPrompt).NotTo(BeEmpty())
}

func TestDefaults_SurfaceFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	pol := policy.Defaults()

	g.Expect(pol.SurfaceCandidateCountMin).To(BeNumerically(">", 0))
	g.Expect(pol.SurfaceCandidateCountMax).To(BeNumerically(">", pol.SurfaceCandidateCountMin))
	g.Expect(pol.SurfaceBM25Threshold).To(BeNumerically(">", 0))
	g.Expect(pol.SurfaceColdStartBudget).To(BeNumerically(">", 0))
	g.Expect(pol.SurfaceIrrelevanceHalfLife).To(BeNumerically(">", 0))
	g.Expect(pol.SurfaceGateHaikuPrompt).NotTo(BeEmpty())
	g.Expect(pol.SurfaceInjectionPreamble).NotTo(BeEmpty())
}

func TestLoadFromPath_ReturnsDefaults_WhenMissing(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	pol, err := policy.LoadFromPath("/nonexistent/path/policy.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	defaults := policy.Defaults()
	g.Expect(pol.ContextByteBudget).To(Equal(defaults.ContextByteBudget))
	g.Expect(pol.DetectFastPathKeywords).To(Equal(defaults.DetectFastPathKeywords))
}

func TestLoad_DefaultKeywordsContainExpected(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	pol := policy.Defaults()

	allKeywords := strings.Join(pol.DetectFastPathKeywords, " ")
	g.Expect(allKeywords).To(ContainSubstring("remember"))
	g.Expect(allKeywords).To(ContainSubstring("stop"))
}

func TestLoad_DefaultsWhenMissing(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	pol, err := policy.Load(func(string) ([]byte, error) {
		return nil, os.ErrNotExist
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	defaults := policy.Defaults()
	g.Expect(pol).To(Equal(defaults))
}

func TestLoad_ErrorOnInvalidTOML(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	_, err := policy.Load(func(string) ([]byte, error) {
		return []byte("not valid toml [[["), nil
	})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("parsing policy"))
}

func TestLoad_OverridesSurfaceFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	tomlContent := `
[parameters]
surface_candidate_count_min = 5
surface_candidate_count_max = 15
surface_bm25_threshold = 0.6
surface_cold_start_budget = 4
surface_irrelevance_half_life = 10

[prompts]
surface_gate_haiku = "Custom gate prompt."
surface_injection_preamble = "Custom preamble."
`

	pol, err := policy.Load(func(string) ([]byte, error) {
		return []byte(tomlContent), nil
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	defaults := policy.Defaults()

	// Overridden surface fields
	g.Expect(pol.SurfaceCandidateCountMin).To(Equal(5))
	g.Expect(pol.SurfaceCandidateCountMax).To(Equal(15))
	g.Expect(pol.SurfaceBM25Threshold).To(BeNumerically("~", 0.6, 0.0001))
	g.Expect(pol.SurfaceColdStartBudget).To(Equal(4))
	g.Expect(pol.SurfaceIrrelevanceHalfLife).To(Equal(10))
	g.Expect(pol.SurfaceGateHaikuPrompt).To(Equal("Custom gate prompt."))
	g.Expect(pol.SurfaceInjectionPreamble).To(Equal("Custom preamble."))

	// Non-overridden fields keep defaults
	g.Expect(pol.ContextByteBudget).To(Equal(defaults.ContextByteBudget))
	g.Expect(pol.ExtractCandidateCountMin).To(Equal(defaults.ExtractCandidateCountMin))
	g.Expect(pol.DetectHaikuPrompt).To(Equal(defaults.DetectHaikuPrompt))
}

func TestLoad_ParsesFile(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	tomlContent := `
[parameters]
detect_fast_path_keywords = ["fix", "stop"]
context_byte_budget = 25600
context_tool_args_truncate = 100
context_tool_result_truncate = 250
extract_candidate_count_min = 2
extract_candidate_count_max = 5
extract_bm25_threshold = 0.5
surface_candidate_count_min = 1
surface_candidate_count_max = 4
surface_bm25_threshold = 0.4
surface_cold_start_budget = 1
surface_irrelevance_half_life = 3

[prompts]
detect_haiku = "Is this a correction?"
extract_sonnet = "Extract SBIA fields."
surface_gate_haiku = "Gate prompt."
surface_injection_preamble = "Preamble."
`

	pol, err := policy.Load(func(string) ([]byte, error) {
		return []byte(tomlContent), nil
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(pol.DetectFastPathKeywords).To(Equal([]string{"fix", "stop"}))
	g.Expect(pol.ContextByteBudget).To(Equal(25600))
	g.Expect(pol.ContextToolArgsTruncate).To(Equal(100))
	g.Expect(pol.ContextToolResultTruncate).To(Equal(250))
	g.Expect(pol.ExtractCandidateCountMin).To(Equal(2))
	g.Expect(pol.ExtractCandidateCountMax).To(Equal(5))
	g.Expect(pol.ExtractBM25Threshold).To(BeNumerically("~", 0.5, 0.0001))
	g.Expect(pol.SurfaceCandidateCountMin).To(Equal(1))
	g.Expect(pol.SurfaceCandidateCountMax).To(Equal(4))
	g.Expect(pol.SurfaceBM25Threshold).To(BeNumerically("~", 0.4, 0.0001))
	g.Expect(pol.SurfaceColdStartBudget).To(Equal(1))
	g.Expect(pol.SurfaceIrrelevanceHalfLife).To(Equal(3))
	g.Expect(pol.DetectHaikuPrompt).To(Equal("Is this a correction?"))
	g.Expect(pol.ExtractSonnetPrompt).To(Equal("Extract SBIA fields."))
	g.Expect(pol.SurfaceGateHaikuPrompt).To(Equal("Gate prompt."))
	g.Expect(pol.SurfaceInjectionPreamble).To(Equal("Preamble."))
}

func TestLoad_PartialOverride(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	tomlContent := `
[parameters]
context_byte_budget = 10240
`

	pol, err := policy.Load(func(string) ([]byte, error) {
		return []byte(tomlContent), nil
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	defaults := policy.Defaults()

	// Overridden field
	g.Expect(pol.ContextByteBudget).To(Equal(10240))

	// Non-overridden fields keep defaults
	g.Expect(pol.DetectFastPathKeywords).To(Equal(defaults.DetectFastPathKeywords))
	g.Expect(pol.ContextToolArgsTruncate).To(Equal(defaults.ContextToolArgsTruncate))
	g.Expect(pol.ContextToolResultTruncate).To(Equal(defaults.ContextToolResultTruncate))
	g.Expect(pol.ExtractCandidateCountMin).To(Equal(defaults.ExtractCandidateCountMin))
	g.Expect(pol.ExtractCandidateCountMax).To(Equal(defaults.ExtractCandidateCountMax))
	g.Expect(pol.ExtractBM25Threshold).To(BeNumerically("~", defaults.ExtractBM25Threshold, 0.0001))
	g.Expect(pol.DetectHaikuPrompt).To(Equal(defaults.DetectHaikuPrompt))
	g.Expect(pol.ExtractSonnetPrompt).To(Equal(defaults.ExtractSonnetPrompt))
}
