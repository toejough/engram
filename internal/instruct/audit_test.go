package instruct_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/instruct"
)

// T-217: Deduplication detects >80% keyword overlap.
func TestAuditRun_DeduplicatesOverlappingInstructions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Two instructions with ~90% keyword overlap but different sources
	contentA := "always use fish shell for terminal commands in this project"
	contentB := "always use fish shell for terminal commands in every project"

	scanner := &instruct.Scanner{
		ReadFile: func(path string) ([]byte, error) {
			switch path {
			case "/project/CLAUDE.md":
				return []byte(contentA), nil
			case "/data/memories/shell.toml":
				return []byte(contentB), nil
			}

			return nil, fmt.Errorf("not found: %s", path)
		},
		GlobFiles: func(pattern string) ([]string, error) {
			if pattern == "/data/memories/*.toml" {
				return []string{"/data/memories/shell.toml"}, nil
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
	// CLAUDE.md has higher salience → keep it
	g.Expect(report.Duplicates[0].KeepSource).To(Equal("/project/CLAUDE.md"))
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

// T-221: Skill decomposition identifies low-effectiveness lines.
func TestAuditRun_FlagsLowEffectivenessSkillLines(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skillContent := "line one\nline two\nline three\nline four\nline five\n" +
		"line six\nline seven\nline eight\nline nine\nline ten"

	files := map[string]string{
		"/project/.claude-plugin/skills/test.md": skillContent,
	}

	const (
		lowRate1 = 10.0
		lowRate2 = 15.0
		lowRate3 = 5.0
	)

	effData := map[string]float64{
		"/project/.claude-plugin/skills/test.md:2":  lowRate1,
		"/project/.claude-plugin/skills/test.md:5":  lowRate2,
		"/project/.claude-plugin/skills/test.md:9":  lowRate3,
		"/project/.claude-plugin/skills/test.md:1":  50.0, // above threshold
		"/project/.claude-plugin/skills/test.md:10": 80.0, // above threshold
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
			if pattern == "/project/.claude-plugin/skills/*.md" {
				return []string{"/project/.claude-plugin/skills/test.md"}, nil
			}

			return nil, nil
		},
		EffData: effData,
	}

	auditor := &instruct.Auditor{
		Scanner:   scanner,
		LLMCaller: nil,
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

	const expectedIssues = 3
	g.Expect(report.Skills).To(HaveLen(expectedIssues))

	for _, issue := range report.Skills {
		g.Expect(issue.FollowRate).To(BeNumerically("<", 20.0))
		g.Expect(issue.Content).NotTo(BeEmpty())
		g.Expect(issue.LineNumber).To(BeNumerically(">", 0))
	}
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

	// Duplicates, gaps, skills still run (may be empty but not nil)
	g.Expect(report.Duplicates).NotTo(BeNil())
	g.Expect(report.Gaps).NotTo(BeNil())
	g.Expect(report.Skills).NotTo(BeNil())
}
