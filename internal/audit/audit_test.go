// Package audit_test tests the Stop Session Audit pipeline (UC-19).
package audit_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/audit"
)

// T-198: Audit phase is skipped if Haiku API token is missing.
func TestAuditor_NoToken_ReturnsErrNoToken(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	auditor := audit.New("/data") // no WithLLMCaller → nil llmCaller

	_, err := auditor.Run(context.Background(), "transcript")
	g.Expect(err).To(MatchError(audit.ErrNoToken))
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

		data, _ := json.Marshal(entry)
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
					return nil
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
