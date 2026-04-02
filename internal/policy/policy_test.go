package policy_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/policy"
)

func TestAppendChangeHistory_PreservesExistingSections(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	existingContent := "[parameters]\ncontext_byte_budget = 1024"

	readFile := func(string) ([]byte, error) {
		return []byte(existingContent), nil
	}

	var written []byte

	writeFile := func(_ string, data []byte) error {
		written = data

		return nil
	}

	entry := policy.ChangeEntry{
		Action:    "rewrite",
		Target:    "test-memory",
		Status:    "applied",
		Rationale: "test",
		ChangedAt: "2026-03-31T12:00:00Z",
	}

	err := policy.AppendChangeHistory(policy.Filename, entry, readFile, writeFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(written).NotTo(BeNil())
	g.Expect(string(written)).To(ContainSubstring("[parameters]"))
	g.Expect(string(written)).To(ContainSubstring("context_byte_budget = 1024"))
	g.Expect(string(written)).To(ContainSubstring("[[change_history]]"))
}

func TestAppendChangeHistory_ReadError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	readFile := func(string) ([]byte, error) {
		return nil, errors.New("disk failure")
	}

	writeFile := func(string, []byte) error {
		return nil
	}

	entry := policy.ChangeEntry{
		Action:    "rewrite",
		Target:    "test",
		Status:    "applied",
		Rationale: "test",
		ChangedAt: "2026-03-31T12:00:00Z",
	}

	err := policy.AppendChangeHistory(policy.Filename, entry, readFile, writeFile)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("appending change history"))
}

func TestAppendChangeHistory_TrimsToLimit(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var written []byte

	readFile := func(string) ([]byte, error) {
		if written == nil {
			return nil, os.ErrNotExist
		}

		return written, nil
	}

	writeFile := func(_ string, data []byte) error {
		written = data

		return nil
	}

	// Write 50 entries (the default limit).
	for idx := range 50 {
		entry := policy.ChangeEntry{
			Action:    "rewrite",
			Target:    "memory-" + strings.Repeat("x", idx),
			Status:    "applied",
			Rationale: "test",
			ChangedAt: "2026-03-31T12:00:00Z",
		}

		err := policy.AppendChangeHistory(policy.Filename, entry, readFile, writeFile)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}
	}

	// Verify we have 50 entries.
	entries, err := policy.ReadChangeHistory(policy.Filename, readFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).NotTo(BeNil())
	g.Expect(entries).To(HaveLen(50))

	// Add one more — should trim to 50, dropping the oldest.
	overflowEntry := policy.ChangeEntry{
		Action:    "consolidate",
		Target:    "overflow-entry",
		Status:    "applied",
		Rationale: "overflow test",
		ChangedAt: "2026-03-31T13:00:00Z",
	}

	err = policy.AppendChangeHistory(policy.Filename, overflowEntry, readFile, writeFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	entries, err = policy.ReadChangeHistory(policy.Filename, readFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).NotTo(BeNil())

	if entries == nil {
		return
	}

	g.Expect(entries).To(HaveLen(50))

	// The last entry should be our overflow entry.
	lastEntry := entries[len(entries)-1]
	g.Expect(lastEntry.Target).To(Equal("overflow-entry"))

	// The first entry should NOT be the original first entry (it was trimmed).
	g.Expect(entries[0].Target).NotTo(Equal("memory-"))
}

func TestAppendChangeHistory_WriteError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	readFile := func(string) ([]byte, error) {
		return nil, os.ErrNotExist
	}

	writeFile := func(string, []byte) error {
		return errors.New("write failed")
	}

	entry := policy.ChangeEntry{
		Action:    "rewrite",
		Target:    "test",
		Status:    "applied",
		Rationale: "test",
		ChangedAt: "2026-03-31T12:00:00Z",
	}

	err := policy.AppendChangeHistory(policy.Filename, entry, readFile, writeFile)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("write failed"))
}

func TestChangeHistory_RoundTrip(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var written []byte

	readFile := func(string) ([]byte, error) {
		if written == nil {
			return nil, os.ErrNotExist
		}

		return written, nil
	}

	writeFile := func(_ string, data []byte) error {
		written = data

		return nil
	}

	entry := policy.ChangeEntry{
		Action:    "rewrite",
		Target:    "use-targ-build",
		Field:     "situation",
		OldValue:  "When building Go code",
		NewValue:  "When running build commands in a Go project with a targ build system",
		Status:    "applied",
		Rationale: "Too vague — triggered in non-targ projects",
		ChangedAt: "2026-03-31T12:00:00Z",
	}

	err := policy.AppendChangeHistory(policy.Filename, entry, readFile, writeFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	entries, readErr := policy.ReadChangeHistory(policy.Filename, readFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(entries).NotTo(BeNil())

	if entries == nil {
		return
	}

	g.Expect(entries).To(HaveLen(1))
	g.Expect(entries[0].Action).To(Equal("rewrite"))
	g.Expect(entries[0].Target).To(Equal("use-targ-build"))
	g.Expect(entries[0].Field).To(Equal("situation"))
	g.Expect(entries[0].OldValue).To(Equal("When building Go code"))
	g.Expect(entries[0].NewValue).To(
		Equal("When running build commands in a Go project with a targ build system"),
	)
	g.Expect(entries[0].Status).To(Equal("applied"))
	g.Expect(entries[0].Rationale).To(Equal("Too vague — triggered in non-targ projects"))
	g.Expect(entries[0].ChangedAt).To(Equal("2026-03-31T12:00:00Z"))
}

func TestDefaults_EvaluateHaikuPrompt(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	pol := policy.Defaults()

	g.Expect(pol.EvaluateHaikuPrompt).NotTo(BeEmpty())
}

func TestDefaults_ExtractSonnetPromptContainsSBIADecisionTree(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	pol := policy.Defaults()

	// Prompt must instruct Sonnet to use the SBIA decision tree, not binary is_new/duplicate_of.
	g.Expect(pol.ExtractSonnetPrompt).NotTo(ContainSubstring("is_new"))
	g.Expect(pol.ExtractSonnetPrompt).NotTo(ContainSubstring("duplicate_of"))

	// Prompt must not contain vestigial array-of-memories language from old design.
	g.Expect(pol.ExtractSonnetPrompt).NotTo(ContainSubstring("memories if multiple"))

	// Prompt must reference all 8 disposition values from disposition.go.
	g.Expect(pol.ExtractSonnetPrompt).To(ContainSubstring("STORE"))
	g.Expect(pol.ExtractSonnetPrompt).To(ContainSubstring("DUPLICATE"))
	g.Expect(pol.ExtractSonnetPrompt).To(ContainSubstring("CONTRADICTION"))
	g.Expect(pol.ExtractSonnetPrompt).To(ContainSubstring("REFINEMENT"))
	g.Expect(pol.ExtractSonnetPrompt).To(ContainSubstring("IMPACT_UPDATE"))
	g.Expect(pol.ExtractSonnetPrompt).To(ContainSubstring("POTENTIAL_GENERALIZATION"))
	g.Expect(pol.ExtractSonnetPrompt).To(ContainSubstring("LEGITIMATE_SEPARATE"))
	g.Expect(pol.ExtractSonnetPrompt).To(ContainSubstring("STORE_BOTH"))

	// Prompt must instruct per-candidate disposition output with the correct JSON schema.
	g.Expect(pol.ExtractSonnetPrompt).To(ContainSubstring("candidates"))
	g.Expect(pol.ExtractSonnetPrompt).To(ContainSubstring("disposition"))
	g.Expect(pol.ExtractSonnetPrompt).To(ContainSubstring("reason"))

	// Prompt must include the decision tree walkthrough structure.
	g.Expect(pol.ExtractSonnetPrompt).To(ContainSubstring("Same situation"))
	g.Expect(pol.ExtractSonnetPrompt).To(ContainSubstring("Similar situation"))
	g.Expect(pol.ExtractSonnetPrompt).To(ContainSubstring("Different situation"))
}

func TestDefaults_MaintainFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	pol := policy.Defaults()

	g.Expect(pol.MaintainEffectivenessThreshold).To(BeNumerically("~", 50.0, 0.01))
	g.Expect(pol.MaintainMinSurfaced).To(Equal(5))
	g.Expect(pol.MaintainIrrelevanceThreshold).To(BeNumerically("~", 60.0, 0.01))
	g.Expect(pol.MaintainNotFollowedThreshold).To(BeNumerically("~", 50.0, 0.01))
	g.Expect(pol.AdaptChangeHistoryLimit).To(Equal(50))
	g.Expect(pol.MaintainRewritePrompt).NotTo(BeEmpty())
	g.Expect(pol.MaintainConsolidatePrompt).NotTo(BeEmpty())
	g.Expect(pol.AdaptSonnetPrompt).NotTo(BeEmpty())
}

func TestDefaults_RefineSonnetPrompt(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	pol := policy.Defaults()

	// Refine prompt must exist and be non-empty.
	g.Expect(pol.RefineSonnetPrompt).NotTo(BeEmpty())

	// Refine prompt must NOT contain extraction language.
	g.Expect(pol.RefineSonnetPrompt).NotTo(ContainSubstring("extract"))
	g.Expect(pol.RefineSonnetPrompt).NotTo(ContainSubstring("correction message"))
	g.Expect(pol.RefineSonnetPrompt).NotTo(ContainSubstring("candidates"))

	// Refine prompt must instruct rewriting existing SBIA fields.
	g.Expect(pol.RefineSonnetPrompt).To(ContainSubstring("rewrite"))
	g.Expect(pol.RefineSonnetPrompt).To(ContainSubstring("situation"))
	g.Expect(pol.RefineSonnetPrompt).To(ContainSubstring("behavior"))
	g.Expect(pol.RefineSonnetPrompt).To(ContainSubstring("impact"))
	g.Expect(pol.RefineSonnetPrompt).To(ContainSubstring("action"))

	// Refine prompt must output a single JSON object, not an array.
	g.Expect(pol.RefineSonnetPrompt).To(ContainSubstring("JSON object"))
	g.Expect(pol.RefineSonnetPrompt).NotTo(ContainSubstring("JSON array"))

	// Refine prompt must specify retention behavior per memory guidance.
	g.Expect(pol.RefineSonnetPrompt).To(ContainSubstring("existing memory"))
}

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

	// Preamble must not claim "full details" since SBIA fields are already displayed inline.
	g.Expect(pol.SurfaceInjectionPreamble).NotTo(ContainSubstring("full details"))
	// Preamble should reference engram show for the metadata it actually provides.
	g.Expect(pol.SurfaceInjectionPreamble).To(ContainSubstring("engram show"))
	// Preamble should mention what engram show actually adds beyond inline SBIA fields.
	g.Expect(pol.SurfaceInjectionPreamble).To(ContainSubstring("effectiveness"))
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

func TestLoad_MaintainOverrides(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	tomlContent := `
[parameters]
maintain_effectiveness_threshold = 70.0
maintain_min_surfaced = 10
maintain_irrelevance_threshold = 80.0
maintain_not_followed_threshold = 65.0
adapt_change_history_limit = 25

[prompts]
maintain_rewrite = "Custom rewrite prompt."
maintain_consolidate = "Custom consolidate prompt."
adapt_sonnet = "Custom adapt prompt."
`

	pol, err := policy.Load(func(string) ([]byte, error) {
		return []byte(tomlContent), nil
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(pol.MaintainEffectivenessThreshold).To(BeNumerically("~", 70.0, 0.01))
	g.Expect(pol.MaintainMinSurfaced).To(Equal(10))
	g.Expect(pol.MaintainIrrelevanceThreshold).To(BeNumerically("~", 80.0, 0.01))
	g.Expect(pol.MaintainNotFollowedThreshold).To(BeNumerically("~", 65.0, 0.01))
	g.Expect(pol.AdaptChangeHistoryLimit).To(Equal(25))
	g.Expect(pol.MaintainRewritePrompt).To(Equal("Custom rewrite prompt."))
	g.Expect(pol.MaintainConsolidatePrompt).To(Equal("Custom consolidate prompt."))
	g.Expect(pol.AdaptSonnetPrompt).To(Equal("Custom adapt prompt."))
}

func TestLoad_OverridesEvaluateHaikuPrompt(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	tomlContent := `
[prompts]
evaluate_haiku = "custom evaluate prompt"
`

	pol, err := policy.Load(func(string) ([]byte, error) {
		return []byte(tomlContent), nil
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(pol.EvaluateHaikuPrompt).To(Equal("custom evaluate prompt"))
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

func TestReadChangeHistory_ReturnsNilWhenMissing(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	readFile := func(string) ([]byte, error) {
		return nil, os.ErrNotExist
	}

	entries, err := policy.ReadChangeHistory(policy.Filename, readFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(BeNil())
}
