// Package audit_test tests the Stop Session Audit pipeline (UC-19).
package audit_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/audit"
)

// Run returns error when buildScope fails (non-NotExist read error).
func TestAuditor_BuildScopeError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	auditor := audit.New("/data",
		audit.WithReadFile(func(_ string) ([]byte, error) {
			return nil, errors.New("disk on fire")
		}),
		audit.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return "", nil
		}),
	)

	_, err := auditor.Run(context.Background(), "transcript")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("building scope")))
}

// buildCompliancePrompt with empty transcript.
func TestAuditor_CompliancePromptEmptyTranscript(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var capturedPrompt string

	fixedNow := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	auditor := audit.New("/data",
		audit.WithReadFile(func(name string) ([]byte, error) {
			if strings.Contains(name, "surfacing-log") {
				return []byte(`{"memory_path":"m1","effectiveness_score":90.0}`), nil
			}

			return nil, os.ErrNotExist
		}),
		audit.WithLLMCaller(func(_ context.Context, _, _, userPrompt string) (string, error) {
			capturedPrompt = userPrompt

			return `[{"instruction":"m1","compliant":true,"evidence":"ok"}]`, nil
		}),
		audit.WithWriteFile(func(_ string, _ []byte, _ os.FileMode) error { return nil }),
		audit.WithMkdirAll(func(_ string, _ os.FileMode) error { return nil }),
		audit.WithNow(func() time.Time { return fixedNow }),
	)

	_, err := auditor.Run(context.Background(), "")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(capturedPrompt).To(ContainSubstring("Session transcript:\n\n"))
}

// buildCompliancePrompt formats scope and transcript correctly.
func TestAuditor_CompliancePromptFormatsMultipleEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var capturedPrompt string

	fixedNow := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Multiple entries with different scores to test prompt formatting.
	surfacingLog := "" +
		`{"memory_path":"m1","effectiveness_score":90.0}` + "\n" +
		`{"memory_path":"m2","effectiveness_score":80.0}` + "\n" +
		`{"memory_path":"m3","effectiveness_score":70.0}` + "\n" +
		`{"memory_path":"m4","effectiveness_score":60.0}` + "\n" +
		`{"memory_path":"m5","effectiveness_score":50.0}`

	auditor := audit.New("/data",
		audit.WithReadFile(func(name string) ([]byte, error) {
			if strings.Contains(name, "surfacing-log") {
				return []byte(surfacingLog), nil
			}

			return nil, os.ErrNotExist
		}),
		audit.WithLLMCaller(func(_ context.Context, _, _, userPrompt string) (string, error) {
			capturedPrompt = userPrompt

			return `[{"instruction":"m1","compliant":true,"evidence":"ok"}]`, nil
		}),
		audit.WithWriteFile(func(_ string, _ []byte, _ os.FileMode) error { return nil }),
		audit.WithMkdirAll(func(_ string, _ os.FileMode) error { return nil }),
		audit.WithNow(func() time.Time { return fixedNow }),
	)

	_, err := auditor.Run(context.Background(), "the transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Top 20% of 5 = 1 entry (m1 with score 90).
	g.Expect(capturedPrompt).To(ContainSubstring("m1"))
	g.Expect(capturedPrompt).To(ContainSubstring("90.0%"))
	g.Expect(capturedPrompt).To(ContainSubstring("the transcript"))
	g.Expect(capturedPrompt).To(ContainSubstring("High-priority instructions"))
	g.Expect(capturedPrompt).To(ContainSubstring("Session transcript"))
}

// Run returns nil when surfacing log has only empty/whitespace lines.
func TestAuditor_EmptySurfacingLog(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	auditor := audit.New("/data",
		audit.WithReadFile(func(_ string) ([]byte, error) {
			return []byte("\n\n"), nil
		}),
		audit.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return "", nil
		}),
	)

	report, err := auditor.Run(context.Background(), "transcript")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report).To(BeNil())
}

// Run returns error when injectSignals fails (mkdirAll error on evaluations dir).
func TestAuditor_InjectSignalsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = `{"memory_path":"/data/memories/m1.toml","effectiveness_score":90.0}`

	fixedNow := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	mkdirCallCount := 0

	auditor := audit.New("/data",
		audit.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return nil, os.ErrNotExist
		}),
		audit.WithWriteFile(func(_ string, _ []byte, _ os.FileMode) error { return nil }),
		audit.WithMkdirAll(func(_ string, _ os.FileMode) error {
			mkdirCallCount++
			// First call (audits dir) succeeds, second call (evaluations dir) fails.
			if mkdirCallCount >= 2 {
				return errors.New("evaluations dir denied")
			}

			return nil
		}),
		audit.WithNow(func() time.Time { return fixedNow }),
		audit.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return `[{"instruction": "/data/memories/m1.toml", "compliant": false, "evidence": "bad"}]`, nil
		}),
	)

	_, err := auditor.Run(context.Background(), "transcript")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("injecting signals")))
}

// T-205: Non-compliance results are injected into effectiveness history.
func TestAuditor_InjectsNegativeSignals(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = `{"memory_path":"/data/memories/m1.toml","effectiveness_score":90.0}`

	fixedNow := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	var evalWritten bool

	auditor := audit.New("/data",
		audit.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return nil, os.ErrNotExist
		}),
		audit.WithWriteFile(func(name string, data []byte, _ os.FileMode) error {
			if strings.Contains(name, "evaluations/audit-") {
				evalWritten = true

				lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
				g.Expect(lines).To(HaveLen(1))

				var entry map[string]string

				jsonErr := json.Unmarshal([]byte(lines[0]), &entry)
				g.Expect(jsonErr).NotTo(HaveOccurred())

				if jsonErr != nil {
					return jsonErr
				}

				g.Expect(entry["outcome"]).To(Equal("contradicted"))
			}

			return nil
		}),
		audit.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		audit.WithNow(func() time.Time { return fixedNow }),
		audit.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return `[{"instruction": "/data/memories/m1.toml", "compliant": false, "evidence": "nope"}]`, nil
		}),
	)

	_, err := auditor.Run(context.Background(), "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(evalWritten).To(BeTrue())
}

// Run returns error when LLM call fails.
func TestAuditor_LLMCallError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = `{"memory_path":"/data/memories/m1.toml","effectiveness_score":90.0}`

	auditor := audit.New("/data",
		audit.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return nil, os.ErrNotExist
		}),
		audit.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return "", errors.New("API unavailable")
		}),
	)

	_, err := auditor.Run(context.Background(), "transcript")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("calling LLM")))
}

// Run returns error when LLM returns invalid JSON.
func TestAuditor_LLMInvalidJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = `{"memory_path":"/data/memories/m1.toml","effectiveness_score":90.0}`

	auditor := audit.New("/data",
		audit.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return nil, os.ErrNotExist
		}),
		audit.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return "not valid json at all", nil
		}),
	)

	_, err := auditor.Run(context.Background(), "transcript")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("parsing LLM response")))
}

// T-198: Audit phase is skipped if Haiku API token is missing.
func TestAuditor_NoToken_ReturnsErrNoToken(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	auditor := audit.New("/data") // no WithLLMCaller → nil llmCaller

	_, err := auditor.Run(context.Background(), "transcript")
	g.Expect(err).To(MatchError(audit.ErrNoToken))
}

// T-202: Non-compliant instruction lowers follow rate signal.
func TestAuditor_NonCompliantRecordedAsSignal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = `{"memory_path":"/data/memories/m1.toml","effectiveness_score":90.0}`

	fixedNow := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	writtenFiles := make([]string, 0)
	writtenData := make([][]byte, 0)

	auditor := audit.New("/data",
		audit.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return nil, os.ErrNotExist
		}),
		audit.WithWriteFile(func(name string, data []byte, _ os.FileMode) error {
			writtenFiles = append(writtenFiles, name)
			writtenData = append(writtenData, data)

			return nil
		}),
		audit.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		audit.WithNow(func() time.Time { return fixedNow }),
		audit.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return `[{"instruction": "/data/memories/m1.toml", "compliant": false, "evidence": "violated"}]`, nil
		}),
	)

	_, err := auditor.Run(context.Background(), "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Should have written both audit report and evaluation signal.
	g.Expect(len(writtenFiles)).To(BeNumerically(">=", 2))

	if len(writtenFiles) < 2 {
		return
	}

	// Find the evaluation signal file.
	foundEval := false

	for idx, name := range writtenFiles {
		if !strings.Contains(name, "evaluations/") {
			continue
		}

		foundEval = true

		var entry map[string]string

		jsonErr := json.Unmarshal([]byte(strings.TrimSpace(string(writtenData[idx]))), &entry)
		g.Expect(jsonErr).NotTo(HaveOccurred())

		if jsonErr != nil {
			return
		}

		g.Expect(entry["memory_path"]).To(Equal("/data/memories/m1.toml"))
		g.Expect(entry["outcome"]).To(Equal("contradicted"))
	}

	g.Expect(foundEval).To(BeTrue())
}

// T-201: Haiku compliance assessment returns JSON with instruction compliance.
func TestAuditor_ParsesComplianceJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = `{"memory_path":"/data/memories/m1.toml","effectiveness_score":90.0}`

	fixedNow := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	auditor := audit.New("/data",
		audit.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return nil, os.ErrNotExist
		}),
		audit.WithWriteFile(func(string, []byte, os.FileMode) error { return nil }),
		audit.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		audit.WithNow(func() time.Time { return fixedNow }),
		audit.WithLLMCaller(func(_ context.Context, model, _, _ string) (string, error) {
			g.Expect(model).To(Equal("claude-haiku-4-5-20251001"))

			return `[{"instruction": "/data/memories/m1.toml", "compliant": false, "evidence": "used go build"}]`, nil
		}),
	)

	report, err := auditor.Run(context.Background(), "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(report).NotTo(BeNil())

	if report == nil {
		return
	}

	g.Expect(report.Results).To(HaveLen(1))

	if len(report.Results) < 1 {
		return
	}

	g.Expect(report.Results[0].Instruction).To(Equal("/data/memories/m1.toml"))
	g.Expect(report.Results[0].Compliant).To(BeFalse())
	g.Expect(report.Results[0].Evidence).To(Equal("used go build"))
}

// T-204: Audit report includes metadata and results.
func TestAuditor_ReportIncludesMetadata(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = "" +
		`{"memory_path":"/data/memories/m1.toml","effectiveness_score":90.0}` + "\n" +
		`{"memory_path":"/data/memories/m2.toml","effectiveness_score":80.0}`

	fixedNow := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	var reportData []byte

	auditor := audit.New("/data",
		audit.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return nil, os.ErrNotExist
		}),
		audit.WithWriteFile(func(name string, data []byte, _ os.FileMode) error {
			if strings.Contains(name, "audits/") {
				reportData = data
			}

			return nil
		}),
		audit.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		audit.WithNow(func() time.Time { return fixedNow }),
		audit.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return `[
				{"instruction": "/data/memories/m1.toml", "compliant": true, "evidence": "ok"},
				{"instruction": "/data/memories/m2.toml", "compliant": false, "evidence": "violated"}
			]`, nil
		}),
	)

	_, err := auditor.Run(context.Background(), "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(reportData).NotTo(BeNil())

	if reportData == nil {
		return
	}

	var report map[string]any

	parseErr := json.Unmarshal(reportData, &report)
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	g.Expect(report).To(HaveKey("timestamp"))
	g.Expect(report).To(HaveKey("total_instructions_audited"))
	g.Expect(report).To(HaveKey("compliant"))
	g.Expect(report).To(HaveKey("non_compliant"))
	g.Expect(report).To(HaveKey("results"))
	g.Expect(report["total_instructions_audited"]).To(BeNumerically("==", 2))
	g.Expect(report["compliant"]).To(BeNumerically("==", 1))
	g.Expect(report["non_compliant"]).To(BeNumerically("==", 1))
}

// T-203: Audit report written to audits/<timestamp>.json.
func TestAuditor_ReportWrittenToAuditsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = `{"memory_path":"/data/memories/m1.toml","effectiveness_score":90.0}`

	fixedNow := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	var reportPath string

	auditor := audit.New("/data",
		audit.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return nil, os.ErrNotExist
		}),
		audit.WithWriteFile(func(name string, _ []byte, _ os.FileMode) error {
			if strings.Contains(name, "audits/") {
				reportPath = name
			}

			return nil
		}),
		audit.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		audit.WithNow(func() time.Time { return fixedNow }),
		audit.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return `[{"instruction": "/data/memories/m1.toml", "compliant": true, "evidence": "ok"}]`, nil
		}),
	)

	_, err := auditor.Run(context.Background(), "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(reportPath).To(Equal("/data/audits/2024-01-15T10-30-00Z.json"))
}

// T-199: Audit scope extracts high-priority memories from surfacing log.
func TestAuditor_ScopeExtractsTopTwentyPercent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Build 20 surfacing log entries with scores 1..20.
	var lines []string

	for i := 1; i <= 20; i++ {
		entry := map[string]any{
			"memory_path":         fmt.Sprintf("/data/memories/m%02d.toml", i),
			"effectiveness_score": float64(i),
		}

		data, marshalErr := json.Marshal(entry)
		g.Expect(marshalErr).NotTo(HaveOccurred())

		lines = append(lines, string(data))
	}

	surfacingLog := strings.Join(lines, "\n")

	var capturedPrompt string

	fixedNow := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	auditor := audit.New("/data",
		audit.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return nil, os.ErrNotExist
		}),
		audit.WithWriteFile(func(string, []byte, os.FileMode) error { return nil }),
		audit.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		audit.WithNow(func() time.Time { return fixedNow }),
		audit.WithLLMCaller(func(_ context.Context, _, _, userPrompt string) (string, error) {
			capturedPrompt = userPrompt

			return `[
				{"instruction": "/data/memories/m20.toml", "compliant": true, "evidence": "ok"},
				{"instruction": "/data/memories/m19.toml", "compliant": true, "evidence": "ok"},
				{"instruction": "/data/memories/m18.toml", "compliant": true, "evidence": "ok"},
				{"instruction": "/data/memories/m17.toml", "compliant": true, "evidence": "ok"}
			]`, nil
		}),
	)

	report, err := auditor.Run(context.Background(), "test transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Top 20% of 20 = 4 memories. Should include m17..m20 (highest scores).
	g.Expect(report).NotTo(BeNil())

	if report == nil {
		return
	}

	g.Expect(report.TotalInstructionsAudited).To(Equal(4))
	g.Expect(capturedPrompt).To(ContainSubstring("m20.toml"))
	g.Expect(capturedPrompt).To(ContainSubstring("m17.toml"))
	g.Expect(capturedPrompt).NotTo(ContainSubstring("m01.toml"))
}

// T-206: Skipped injection on missing memory ID (non-fatal).
func TestAuditor_SkipsMissingMemoryID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = `{"memory_path":"/data/memories/m1.toml","effectiveness_score":90.0}`

	fixedNow := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	var evalFileWritten bool

	auditor := audit.New("/data",
		audit.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return nil, os.ErrNotExist
		}),
		audit.WithWriteFile(func(name string, _ []byte, _ os.FileMode) error {
			if strings.Contains(name, "evaluations/audit-") {
				evalFileWritten = true
			}

			return nil
		}),
		audit.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		audit.WithNow(func() time.Time { return fixedNow }),
		audit.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return `[{"instruction": "", "compliant": false, "evidence": "missing"}]`, nil
		}),
	)

	report, err := auditor.Run(context.Background(), "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(report).NotTo(BeNil())
	g.Expect(evalFileWritten).To(BeFalse())
}

// T-200: Audit scope parsing reads transcript for compliance check.
func TestAuditor_TranscriptPassedToLLM(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = `{"memory_path":"/data/memories/m1.toml","effectiveness_score":90.0}`

	var capturedPrompt string

	fixedNow := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	auditor := audit.New("/data",
		audit.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return nil, os.ErrNotExist
		}),
		audit.WithWriteFile(func(string, []byte, os.FileMode) error { return nil }),
		audit.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		audit.WithNow(func() time.Time { return fixedNow }),
		audit.WithLLMCaller(func(_ context.Context, _, _, userPrompt string) (string, error) {
			capturedPrompt = userPrompt

			return `[{"instruction": "/data/memories/m1.toml", "compliant": true, "evidence": "ok"}]`, nil
		}),
	)

	_, err := auditor.Run(context.Background(), "my session transcript content")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(capturedPrompt).To(ContainSubstring("my session transcript content"))
}

// Run returns error when writeReport fails (mkdirAll error).
func TestAuditor_WriteReportError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = `{"memory_path":"/data/memories/m1.toml","effectiveness_score":90.0}`

	fixedNow := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	auditor := audit.New("/data",
		audit.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return nil, os.ErrNotExist
		}),
		audit.WithMkdirAll(func(_ string, _ os.FileMode) error {
			return errors.New("permission denied")
		}),
		audit.WithNow(func() time.Time { return fixedNow }),
		audit.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return `[{"instruction": "/data/memories/m1.toml", "compliant": true, "evidence": "ok"}]`, nil
		}),
	)

	_, err := auditor.Run(context.Background(), "transcript")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("writing report")))
}

// WithLLMCaller overrides the default nil llmCaller.
func TestWithLLMCaller_OverridesDefault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	called := false
	caller := func(_ context.Context, _, _, _ string) (string, error) {
		called = true

		return "[]", nil
	}

	auditor := audit.New("/data",
		audit.WithLLMCaller(caller),
		audit.WithReadFile(func(_ string) ([]byte, error) {
			return nil, os.ErrNotExist
		}),
	)

	// With empty scope, LLM won't be called, but the auditor won't return ErrNoToken.
	report, err := auditor.Run(context.Background(), "t")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report).To(BeNil())   // empty scope
	g.Expect(called).To(BeFalse()) // not called because scope was empty
}

// WithMkdirAll overrides the default os.MkdirAll.
func TestWithMkdirAll_OverridesDefault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	mkdirCalled := false
	fixedNow := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	auditor := audit.New("/data",
		audit.WithReadFile(func(name string) ([]byte, error) {
			if strings.Contains(name, "surfacing-log") {
				return []byte(`{"memory_path":"m1","effectiveness_score":50}`), nil
			}

			return nil, os.ErrNotExist
		}),
		audit.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return `[{"instruction":"m1","compliant":true,"evidence":"ok"}]`, nil
		}),
		audit.WithWriteFile(func(_ string, _ []byte, _ os.FileMode) error { return nil }),
		audit.WithMkdirAll(func(_ string, _ os.FileMode) error {
			mkdirCalled = true

			return nil
		}),
		audit.WithNow(func() time.Time { return fixedNow }),
	)

	_, err := auditor.Run(context.Background(), "t")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(mkdirCalled).To(BeTrue())
}

// WithNow overrides the default time.Now.
func TestWithNow_OverridesDefault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixedNow := time.Date(2099, 12, 31, 23, 59, 59, 0, time.UTC)

	var reportPath string

	auditor := audit.New("/data",
		audit.WithReadFile(func(name string) ([]byte, error) {
			if strings.Contains(name, "surfacing-log") {
				return []byte(`{"memory_path":"m1","effectiveness_score":50}`), nil
			}

			return nil, os.ErrNotExist
		}),
		audit.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return `[{"instruction":"m1","compliant":true,"evidence":"ok"}]`, nil
		}),
		audit.WithWriteFile(func(name string, _ []byte, _ os.FileMode) error {
			if strings.Contains(name, "audits/") {
				reportPath = name
			}

			return nil
		}),
		audit.WithMkdirAll(func(_ string, _ os.FileMode) error { return nil }),
		audit.WithNow(func() time.Time { return fixedNow }),
	)

	_, err := auditor.Run(context.Background(), "t")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(reportPath).To(ContainSubstring("2099"))
}

// WithReadFile overrides the default os.ReadFile.
func TestWithReadFile_OverridesDefault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	customCalled := false

	auditor := audit.New("/data",
		audit.WithReadFile(func(_ string) ([]byte, error) {
			customCalled = true

			return nil, os.ErrNotExist
		}),
		audit.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return "[]", nil
		}),
	)

	_, err := auditor.Run(context.Background(), "t")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(customCalled).To(BeTrue())
}

// WithWriteFile overrides the default os.WriteFile.
func TestWithWriteFile_OverridesDefault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	writeCalled := false
	fixedNow := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	auditor := audit.New("/data",
		audit.WithReadFile(func(name string) ([]byte, error) {
			if strings.Contains(name, "surfacing-log") {
				return []byte(`{"memory_path":"m1","effectiveness_score":50}`), nil
			}

			return nil, os.ErrNotExist
		}),
		audit.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return `[{"instruction":"m1","compliant":true,"evidence":"ok"}]`, nil
		}),
		audit.WithWriteFile(func(_ string, _ []byte, _ os.FileMode) error {
			writeCalled = true

			return nil
		}),
		audit.WithMkdirAll(func(_ string, _ os.FileMode) error { return nil }),
		audit.WithNow(func() time.Time { return fixedNow }),
	)

	_, err := auditor.Run(context.Background(), "t")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(writeCalled).To(BeTrue())
}
