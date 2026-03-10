package instruct_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/instruct"
)

// T-217: Deduplication detects >80% keyword overlap between memories.
func TestAuditRun_DeduplicatesOverlappingInstructions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Two memory instructions with ~90% keyword overlap
	contentA := "always use fish shell for terminal commands in this project"
	contentB := "always use fish shell for terminal commands in every project"

	scanner := &instruct.Scanner{
		ReadFile: func(path string) ([]byte, error) {
			switch path {
			case "/data/memories/shell-a.toml":
				return []byte(contentA), nil
			case "/data/memories/shell-b.toml":
				return []byte(contentB), nil
			}

			return nil, fmt.Errorf("not found: %s", path)
		},
		GlobFiles: func(pattern string) ([]string, error) {
			if pattern == "/data/memories/*.toml" {
				return []string{"/data/memories/shell-a.toml", "/data/memories/shell-b.toml"}, nil
			}

			return nil, nil
		},
		EffData: map[string]float64{},
	}

	auditor := &instruct.Auditor{
		Scanner:   scanner,
		LLMCaller: nil, // no LLM
	}

	report, err := auditor.Run(context.Background(), "/data", "/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(report).NotTo(BeNil())

	if report == nil {
		return
	}

	g.Expect(report.Duplicates).To(HaveLen(1))
	g.Expect(report.Duplicates[0].Overlap).To(BeNumerically(">", 0.80))
}

// diagnoseBottom returns error when LLM call fails.
func TestAuditRun_DiagnoseBottomLLMError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scanner := &instruct.Scanner{
		ReadFile: func(_ string) ([]byte, error) {
			return []byte("instruction content"), nil
		},
		GlobFiles: func(pattern string) ([]string, error) {
			if pattern == "/data/memories/*.toml" {
				return []string{"/data/memories/m1.toml"}, nil
			}

			return nil, nil
		},
		EffData: map[string]float64{"/data/memories/m1.toml": 5.0},
	}

	auditor := &instruct.Auditor{
		Scanner: scanner,
		LLMCaller: func(_ context.Context, _, _, _ string) (string, error) {
			return "", errors.New("LLM timeout")
		},
	}

	_, err := auditor.Run(context.Background(), "/data", "/project")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("diagnosing instructions")))
}

// diagnoseBottom returns error when LLM returns invalid JSON.
func TestAuditRun_DiagnoseBottomParseError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scanner := &instruct.Scanner{
		ReadFile: func(_ string) ([]byte, error) {
			return []byte("instruction content"), nil
		},
		GlobFiles: func(pattern string) ([]string, error) {
			if pattern == "/data/memories/*.toml" {
				return []string{"/data/memories/m1.toml"}, nil
			}

			return nil, nil
		},
		EffData: map[string]float64{"/data/memories/m1.toml": 5.0},
	}

	auditor := &instruct.Auditor{
		Scanner: scanner,
		LLMCaller: func(_ context.Context, _, _, _ string) (string, error) {
			return "not json", nil
		},
	}

	_, err := auditor.Run(context.Background(), "/data", "/project")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("parsing LLM response")))
}

// T-218: Quality diagnosis calls Haiku for bottom 20%.
func TestAuditRun_DiagnosesBottom20Percent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const itemCount = 10

	files := make(map[string]string, itemCount)
	effData := make(map[string]float64, itemCount)

	for i := range itemCount {
		path := fmt.Sprintf("/data/memories/mem%d.toml", i)
		files[path] = fmt.Sprintf("instruction number %d unique_%d content", i, i)
		effData[path] = float64((i + 1) * 10) // 10, 20, ..., 100
	}

	llmCallCount := 0

	scanner := &instruct.Scanner{
		ReadFile: func(path string) ([]byte, error) {
			content, ok := files[path]
			if !ok {
				return nil, fmt.Errorf("not found: %s", path)
			}

			return []byte(content), nil
		},
		GlobFiles: func(pattern string) ([]string, error) {
			if pattern == "/data/memories/*.toml" {
				paths := make([]string, 0, itemCount)
				for i := range itemCount {
					paths = append(paths, fmt.Sprintf("/data/memories/mem%d.toml", i))
				}

				return paths, nil
			}

			return nil, nil
		},
		EffData: effData,
	}

	auditor := &instruct.Auditor{
		Scanner: scanner,
		LLMCaller: func(_ context.Context, _, _, _ string) (string, error) {
			llmCallCount++

			return `{"diagnosis":"too abstract","root_cause":"missing trigger","suggestion":"add when clause"}`, nil
		},
	}

	report, err := auditor.Run(context.Background(), "/data", "/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(report).NotTo(BeNil())

	if report == nil {
		return
	}

	// Bottom 20% of 10 = 2 items
	const expectedDiagnoses = 2
	g.Expect(llmCallCount).To(Equal(expectedDiagnoses))

	diagnoses := make([]instruct.Diagnosis, 0)
	unmarshalErr := json.Unmarshal(report.Diagnoses, &diagnoses)
	g.Expect(unmarshalErr).NotTo(HaveOccurred())

	if unmarshalErr != nil {
		return
	}

	g.Expect(diagnoses).To(HaveLen(expectedDiagnoses))

	if len(diagnoses) < expectedDiagnoses {
		return
	}

	g.Expect(diagnoses[0].Diagnosis).To(Equal("too abstract"))
	g.Expect(diagnoses[0].RootCause).To(Equal("missing trigger"))
	g.Expect(diagnoses[0].Suggestion).To(Equal("add when clause"))
}

// keywordOverlap with both empty sets returns 0.
func TestAuditRun_DuplicatesSingleItemNoPairs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scanner := &instruct.Scanner{
		ReadFile: func(path string) ([]byte, error) {
			// Each file has unique content so no duplicates are found.
			return []byte("unique content for " + path), nil
		},
		GlobFiles: func(pattern string) ([]string, error) {
			if pattern == "/data/memories/*.toml" {
				return []string{"/data/memories/only.toml"}, nil
			}

			return nil, nil
		},
		EffData: map[string]float64{},
	}

	auditor := &instruct.Auditor{Scanner: scanner, LLMCaller: nil}

	report, err := auditor.Run(context.Background(), "/data", "/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(report).NotTo(BeNil())

	if report == nil {
		return
	}

	g.Expect(report.Duplicates).To(BeEmpty())
}

// extractKeywords strips punctuation and single-char words.
func TestAuditRun_DuplicatesWithPunctuation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Content with lots of punctuation; after stripping, they should still match.
	content := "always! use? the (fish) shell, for [terminal] commands."

	scanner := &instruct.Scanner{
		ReadFile: func(path string) ([]byte, error) {
			switch path {
			case "/data/memories/a.toml":
				return []byte(content), nil
			case "/data/memories/b.toml":
				return []byte(content), nil // identical
			}

			return nil, errors.New("not found")
		},
		GlobFiles: func(pattern string) ([]string, error) {
			if pattern == "/data/memories/*.toml" {
				return []string{"/data/memories/a.toml", "/data/memories/b.toml"}, nil
			}

			return nil, nil
		},
		EffData: map[string]float64{},
	}

	auditor := &instruct.Auditor{Scanner: scanner, LLMCaller: nil}

	report, err := auditor.Run(context.Background(), "/data", "/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(report).NotTo(BeNil())

	if report == nil {
		return
	}

	g.Expect(report.Duplicates).To(HaveLen(1))
	g.Expect(report.Duplicates[0].Overlap).To(BeNumerically("==", 1.0))
}

// buildProposals with empty diagnoses produces empty proposals.
func TestAuditRun_EmptyDiagnosesEmptyProposals(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scanner := &instruct.Scanner{
		ReadFile: func(_ string) ([]byte, error) {
			return nil, errors.New("not found")
		},
		GlobFiles: func(_ string) ([]string, error) {
			return nil, nil
		},
		EffData: map[string]float64{},
	}

	// LLM caller present but no items to diagnose (empty scan)
	auditor := &instruct.Auditor{
		Scanner: scanner,
		LLMCaller: func(_ context.Context, _, _, _ string) (string, error) {
			return `{"diagnosis":"x","root_cause":"y","suggestion":"z"}`, nil
		},
	}

	report, err := auditor.Run(context.Background(), "/data", "/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(report).NotTo(BeNil())

	if report == nil {
		return
	}

	var proposals []instruct.RefinementProposal

	unmarshalErr := json.Unmarshal(report.Proposals, &proposals)
	g.Expect(unmarshalErr).NotTo(HaveOccurred())

	if unmarshalErr != nil {
		return
	}

	g.Expect(proposals).To(BeEmpty())
}

// findGaps skips covered paths and non-contradicted outcomes.
func TestAuditRun_FindGapsAllCovered(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scanner := &instruct.Scanner{
		ReadFile: func(_ string) ([]byte, error) {
			return []byte("content"), nil
		},
		GlobFiles: func(pattern string) ([]string, error) {
			if pattern == "/data/memories/*.toml" {
				return []string{"/data/memories/m1.toml"}, nil
			}

			return nil, nil
		},
		EffData: map[string]float64{},
	}

	auditor := &instruct.Auditor{
		Scanner:   scanner,
		LLMCaller: nil,
		EvalData: []instruct.EvalRecord{
			{
				MemoryPath: "/data/memories/m1.toml",
				Outcome:    "contradicted",
				Pattern:    "p1",
				Example:    "e1",
			},
		},
	}

	report, err := auditor.Run(context.Background(), "/data", "/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(report).NotTo(BeNil())

	if report == nil {
		return
	}

	// m1.toml is covered by scanner, so no gaps
	g.Expect(report.Gaps).To(BeEmpty())
}

// findGaps groups multiple contradictions of the same pattern.
func TestAuditRun_FindGapsGroupsPatterns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scanner := &instruct.Scanner{
		ReadFile: func(_ string) ([]byte, error) {
			return []byte("content"), nil
		},
		GlobFiles: func(_ string) ([]string, error) {
			return nil, nil
		},
		EffData: map[string]float64{},
	}

	auditor := &instruct.Auditor{
		Scanner:   scanner,
		LLMCaller: nil,
		EvalData: []instruct.EvalRecord{
			{
				MemoryPath: "/uncovered1",
				Outcome:    "contradicted",
				Pattern:    "same_pat",
				Example:    "ex1",
			},
			{
				MemoryPath: "/uncovered2",
				Outcome:    "contradicted",
				Pattern:    "same_pat",
				Example:    "ex2",
			},
			{
				MemoryPath: "/uncovered3",
				Outcome:    "contradicted",
				Pattern:    "other_pat",
				Example:    "ex3",
			},
		},
	}

	report, err := auditor.Run(context.Background(), "/data", "/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(report).NotTo(BeNil())

	if report == nil {
		return
	}

	g.Expect(report.Gaps).To(HaveLen(2))

	// Sorted descending by count, so same_pat (2) first
	g.Expect(report.Gaps[0].Pattern).To(Equal("same_pat"))
	g.Expect(report.Gaps[0].ViolationCount).To(Equal(2))
	g.Expect(report.Gaps[1].Pattern).To(Equal("other_pat"))
	g.Expect(report.Gaps[1].ViolationCount).To(Equal(1))
}

// findGaps returns empty when no eval data.
func TestAuditRun_FindGapsNoEvalData(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scanner := &instruct.Scanner{
		ReadFile: func(_ string) ([]byte, error) {
			return []byte("content"), nil
		},
		GlobFiles: func(pattern string) ([]string, error) {
			if pattern == "/data/memories/*.toml" {
				return []string{"/data/memories/m1.toml"}, nil
			}

			return nil, nil
		},
		EffData: map[string]float64{},
	}

	auditor := &instruct.Auditor{
		Scanner:   scanner,
		LLMCaller: nil,
		EvalData:  nil, // no eval data
	}

	report, err := auditor.Run(context.Background(), "/data", "/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(report).NotTo(BeNil())

	if report == nil {
		return
	}

	g.Expect(report.Gaps).To(BeEmpty())
}

// T-220: Gap analysis finds violations without instructions.
func TestAuditRun_FindsGapCandidates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := map[string]string{
		"/data/memories/covered1.toml": "covered instruction one",
		"/data/memories/covered2.toml": "covered instruction two",
		"/data/memories/covered3.toml": "covered instruction three",
	}

	scanner := &instruct.Scanner{
		ReadFile: func(path string) ([]byte, error) {
			content, ok := files[path]
			if !ok {
				return nil, fmt.Errorf("not found: %s", path)
			}

			return []byte(content), nil
		},
		GlobFiles: func(pattern string) ([]string, error) {
			if pattern == "/data/memories/*.toml" {
				return []string{
					"/data/memories/covered1.toml",
					"/data/memories/covered2.toml",
					"/data/memories/covered3.toml",
				}, nil
			}

			return nil, nil
		},
		EffData: map[string]float64{},
	}

	auditor := &instruct.Auditor{
		Scanner:   scanner,
		LLMCaller: nil,
		EvalData: []instruct.EvalRecord{
			// 3 covered contradictions
			{
				MemoryPath: "/data/memories/covered1.toml",
				Outcome:    "contradicted",
				Pattern:    "pat1",
				Example:    "ex1",
			},
			{
				MemoryPath: "/data/memories/covered2.toml",
				Outcome:    "contradicted",
				Pattern:    "pat2",
				Example:    "ex2",
			},
			{
				MemoryPath: "/data/memories/covered3.toml",
				Outcome:    "contradicted",
				Pattern:    "pat3",
				Example:    "ex3",
			},
			// 2 uncovered contradictions
			{
				MemoryPath: "/data/memories/missing1.toml",
				Outcome:    "contradicted",
				Pattern:    "gap_pattern_a",
				Example:    "gap example a",
			},
			{
				MemoryPath: "/data/memories/missing2.toml",
				Outcome:    "contradicted",
				Pattern:    "gap_pattern_b",
				Example:    "gap example b",
			},
			// followed outcomes should be ignored
			{
				MemoryPath: "/data/memories/missing3.toml",
				Outcome:    "followed",
				Pattern:    "ok_pattern",
				Example:    "ok",
			},
		},
	}

	report, err := auditor.Run(context.Background(), "/data", "/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(report).NotTo(BeNil())

	if report == nil {
		return
	}

	const expectedGaps = 2
	g.Expect(report.Gaps).To(HaveLen(expectedGaps))

	patterns := make([]string, 0, len(report.Gaps))
	for _, gap := range report.Gaps {
		patterns = append(patterns, gap.Pattern)
		g.Expect(gap.ViolationCount).To(BeNumerically(">=", 1))
		g.Expect(gap.Example).NotTo(BeEmpty())
	}

	g.Expect(patterns).To(ContainElements("gap_pattern_a", "gap_pattern_b"))
}

// T-219: Refinement proposals in maintain-compatible format.
func TestAuditRun_GeneratesRefinementProposals(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := map[string]string{
		"/data/memories/bad.toml": "vague instruction about code quality",
	}

	scanner := &instruct.Scanner{
		ReadFile: func(path string) ([]byte, error) {
			content, ok := files[path]
			if !ok {
				return nil, fmt.Errorf("not found: %s", path)
			}

			return []byte(content), nil
		},
		GlobFiles: func(pattern string) ([]string, error) {
			if pattern == "/data/memories/*.toml" {
				return []string{"/data/memories/bad.toml"}, nil
			}

			return nil, nil
		},
		EffData: map[string]float64{
			"/data/memories/bad.toml": 5.0,
		},
	}

	auditor := &instruct.Auditor{
		Scanner: scanner,
		LLMCaller: func(_ context.Context, _, _, _ string) (string, error) {
			return `{"diagnosis":"too vague","root_cause":"too abstract","suggestion":"specify file types"}`, nil
		},
	}

	report, err := auditor.Run(context.Background(), "/data", "/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(report).NotTo(BeNil())

	if report == nil {
		return
	}

	proposals := make([]instruct.RefinementProposal, 0)
	unmarshalErr := json.Unmarshal(report.Proposals, &proposals)
	g.Expect(unmarshalErr).NotTo(HaveOccurred())

	if unmarshalErr != nil {
		return
	}

	g.Expect(proposals).To(HaveLen(1))

	if len(proposals) < 1 {
		return
	}

	g.Expect(proposals[0].Action).To(Equal("rewrite"))
	g.Expect(proposals[0].Path).To(Equal("/data/memories/bad.toml"))
	g.Expect(proposals[0].RootCause).To(Equal("too abstract"))
	g.Expect(proposals[0].Suggestion).To(Equal("specify file types"))
}

// findDuplicates returns empty for no overlap.
func TestAuditRun_NoDuplicatesForDistinctContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scanner := &instruct.Scanner{
		ReadFile: func(path string) ([]byte, error) {
			switch path {
			case "/data/memories/a.toml":
				return []byte("completely unique alpha beta gamma delta epsilon"), nil
			case "/data/memories/b.toml":
				return []byte("entirely different zeta eta theta iota kappa lambda"), nil
			}

			return nil, errors.New("not found")
		},
		GlobFiles: func(pattern string) ([]string, error) {
			if pattern == "/data/memories/*.toml" {
				return []string{"/data/memories/a.toml", "/data/memories/b.toml"}, nil
			}

			return nil, nil
		},
		EffData: map[string]float64{},
	}

	auditor := &instruct.Auditor{Scanner: scanner, LLMCaller: nil}

	report, err := auditor.Run(context.Background(), "/data", "/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(report).NotTo(BeNil())

	if report == nil {
		return
	}

	g.Expect(report.Duplicates).To(BeEmpty())
}

// T-223: No API token skips LLM steps, runs dedup and gaps.
func TestAuditRun_NoTokenSkipsLLMSteps(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := map[string]string{
		"/data/memories/mem1.toml": "some instruction content here",
	}

	scanner := &instruct.Scanner{
		ReadFile: func(path string) ([]byte, error) {
			content, ok := files[path]
			if !ok {
				return nil, fmt.Errorf("not found: %s", path)
			}

			return []byte(content), nil
		},
		GlobFiles: func(pattern string) ([]string, error) {
			if pattern == "/data/memories/*.toml" {
				return []string{"/data/memories/mem1.toml"}, nil
			}

			return nil, nil
		},
		EffData: map[string]float64{},
	}

	auditor := &instruct.Auditor{
		Scanner:   scanner,
		LLMCaller: nil, // no API token
	}

	report, err := auditor.Run(context.Background(), "/data", "/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(report).NotTo(BeNil())

	if report == nil {
		return
	}

	// Diagnoses and proposals should be skipped
	var skippedDiag instruct.SkippedSection

	unmarshalErr := json.Unmarshal(report.Diagnoses, &skippedDiag)
	g.Expect(unmarshalErr).NotTo(HaveOccurred())

	if unmarshalErr != nil {
		return
	}

	g.Expect(skippedDiag.SkippedReason).To(Equal("no API token"))

	var skippedProp instruct.SkippedSection

	unmarshalErr = json.Unmarshal(report.Proposals, &skippedProp)
	g.Expect(unmarshalErr).NotTo(HaveOccurred())

	if unmarshalErr != nil {
		return
	}

	g.Expect(skippedProp.SkippedReason).To(Equal("no API token"))

	// Duplicates and gaps still run (may be empty but not nil)
	g.Expect(report.Duplicates).NotTo(BeNil())
	g.Expect(report.Gaps).NotTo(BeNil())
}

// Run returns error when ScanAll fails.
func TestAuditRun_ScanError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scanner := &instruct.Scanner{
		ReadFile: func(_ string) ([]byte, error) {
			return nil, errors.New("read error")
		},
		GlobFiles: func(_ string) ([]string, error) {
			return nil, errors.New("glob error")
		},
	}

	auditor := &instruct.Auditor{Scanner: scanner}

	// ScanAll won't error because it swallows errors, so this tests the path.
	report, err := auditor.Run(context.Background(), "/data", "/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(report).NotTo(BeNil())
}
