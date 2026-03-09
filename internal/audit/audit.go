// Package audit implements the Stop Session Audit (UC-19, ARCH-42/43/44).
// It reads surfacing logs and effectiveness data, calls Haiku for compliance
// assessment, writes an audit report, and injects negative signals for
// non-compliant instructions.
package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Exported variables.
var (
	ErrNoToken = errors.New("audit: API token missing or invalid, skipping audit")
)

// Auditor runs the session audit pipeline.
type Auditor struct {
	dataDir   string
	readFile  func(string) ([]byte, error)
	writeFile func(string, []byte, os.FileMode) error
	mkdirAll  func(string, os.FileMode) error
	llmCaller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)
	now       func() time.Time
}

// New creates an Auditor with real defaults. Use options to inject test doubles.
func New(dataDir string, opts ...Option) *Auditor {
	a := &Auditor{
		dataDir:   dataDir,
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
		mkdirAll:  os.MkdirAll,
		now:       time.Now,
	}
	for _, opt := range opts {
		opt(a)
	}

	return a
}

// Run executes the full audit pipeline: scope → LLM assessment → report → injection.
// Returns ErrNoToken if llmCaller is nil.
func (a *Auditor) Run(ctx context.Context, transcript string) (*Report, error) {
	if a.llmCaller == nil {
		return nil, ErrNoToken
	}

	scope, err := a.buildScope()
	if err != nil {
		return nil, fmt.Errorf("audit: building scope: %w", err)
	}

	if len(scope) == 0 {
		return nil, nil
	}

	userPrompt := buildCompliancePrompt(scope, transcript)

	llmResponse, err := a.llmCaller(ctx, auditModel, auditSystemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("audit: calling LLM: %w", err)
	}

	llmResponse = stripMarkdownFence(llmResponse)

	var results []ComplianceResult

	parseErr := json.Unmarshal([]byte(llmResponse), &results)
	if parseErr != nil {
		return nil, fmt.Errorf("audit: parsing LLM response: %w", parseErr)
	}

	report := a.buildReport(results)

	writeErr := a.writeReport(report)
	if writeErr != nil {
		return nil, fmt.Errorf("audit: writing report: %w", writeErr)
	}

	injectErr := a.injectSignals(report)
	if injectErr != nil {
		return nil, fmt.Errorf("audit: injecting signals: %w", injectErr)
	}

	return report, nil
}

func (a *Auditor) buildReport(results []ComplianceResult) *Report {
	var compliant, nonCompliant int

	for _, result := range results {
		if result.Compliant {
			compliant++
		} else {
			nonCompliant++
		}
	}

	return &Report{
		Timestamp:                a.now().UTC().Format(time.RFC3339),
		TotalInstructionsAudited: len(results),
		Compliant:                compliant,
		NonCompliant:             nonCompliant,
		Results:                  results,
	}
}

// buildScope reads the surfacing log and effectiveness data, returning the
// top 20% of memories by effectiveness score.
func (a *Auditor) buildScope() ([]ScopeEntry, error) {
	logPath := filepath.Join(a.dataDir, surfacingLogFilename)

	data, err := a.readFile(logPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("reading surfacing log: %w", err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	entries := make([]ScopeEntry, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry surfacingLogEntry

		jsonErr := json.Unmarshal([]byte(line), &entry)
		if jsonErr != nil {
			continue
		}

		entries = append(entries, ScopeEntry{
			MemoryID:           entry.MemoryPath,
			EffectivenessScore: entry.EffectivenessScore,
		})
	}

	if len(entries) == 0 {
		return nil, nil
	}

	// Sort descending by effectiveness score.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].EffectivenessScore > entries[j].EffectivenessScore
	})

	// Take top 20% (minimum 1).
	topCount := len(entries) / topPercentDivisor
	topCount = max(topCount, 1)

	return entries[:topCount], nil
}

// injectSignals writes a negative outcome for each non-compliant instruction
// into the evaluations directory as a .jsonl file.
func (a *Auditor) injectSignals(report *Report) error {
	var nonCompliant []ComplianceResult

	for _, result := range report.Results {
		if !result.Compliant {
			nonCompliant = append(nonCompliant, result)
		}
	}

	if len(nonCompliant) == 0 {
		return nil
	}

	evalDir := filepath.Join(a.dataDir, evaluationsDirName)

	err := a.mkdirAll(evalDir, dirPerm)
	if err != nil {
		return fmt.Errorf("creating evaluations dir: %w", err)
	}

	timestamp := strings.ReplaceAll(report.Timestamp, ":", "-")
	filePath := filepath.Join(evalDir, "audit-"+timestamp+".jsonl")

	var sb strings.Builder

	for _, result := range nonCompliant {
		if result.Instruction == "" {
			continue // T-206: skip missing memory ID
		}

		line, marshalErr := json.Marshal(evalEntry{
			MemoryPath: result.Instruction,
			Outcome:    "contradicted",
			Evidence:   "audit: " + result.Evidence,
		})
		if marshalErr != nil {
			continue
		}

		sb.Write(line)
		sb.WriteByte('\n')
	}

	if sb.Len() == 0 {
		return nil
	}

	return a.writeFile(filePath, []byte(sb.String()), filePerm)
}

func (a *Auditor) writeReport(report *Report) error {
	auditDir := filepath.Join(a.dataDir, auditsDirName)

	err := a.mkdirAll(auditDir, dirPerm)
	if err != nil {
		return fmt.Errorf("creating audits dir: %w", err)
	}

	timestamp := strings.ReplaceAll(report.Timestamp, ":", "-")
	filePath := filepath.Join(auditDir, timestamp+".json")

	data, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshaling report: %w", err)
	}

	return a.writeFile(filePath, data, filePerm)
}

// ComplianceResult is a single instruction compliance result from the LLM.
//

type ComplianceResult struct {
	Instruction string `json:"instruction"`
	Compliant   bool   `json:"compliant"`
	Evidence    string `json:"evidence"`
}

// Option configures an Auditor.
type Option func(*Auditor)

// Report is the audit report written to audits/<timestamp>.json.
//
//nolint:tagliatelle // spec requires snake_case JSON field names.
type Report struct {
	SessionID                string             `json:"session_id,omitempty"`
	Timestamp                string             `json:"timestamp"`
	TotalInstructionsAudited int                `json:"total_instructions_audited"`
	Compliant                int                `json:"compliant"`
	NonCompliant             int                `json:"non_compliant"`
	Results                  []ComplianceResult `json:"results"`
}

// ScopeEntry is one memory in the audit scope.
//
//nolint:tagliatelle // spec requires snake_case JSON field names.
type ScopeEntry struct {
	MemoryID           string  `json:"memory_id"`
	EffectivenessScore float64 `json:"effectiveness_score"`
}

// WithLLMCaller injects an LLM caller.
func WithLLMCaller(
	fn func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error),
) Option {
	return func(a *Auditor) { a.llmCaller = fn }
}

// WithMkdirAll injects a directory creator.
func WithMkdirAll(fn func(string, os.FileMode) error) Option {
	return func(a *Auditor) { a.mkdirAll = fn }
}

// WithNow injects a clock function.
func WithNow(fn func() time.Time) Option {
	return func(a *Auditor) { a.now = fn }
}

// WithReadFile injects a file reader.
func WithReadFile(fn func(string) ([]byte, error)) Option {
	return func(a *Auditor) { a.readFile = fn }
}

// WithWriteFile injects a file writer.
func WithWriteFile(fn func(string, []byte, os.FileMode) error) Option {
	return func(a *Auditor) { a.writeFile = fn }
}

// unexported constants.
const (
	auditModel        = "claude-haiku-4-5-20251001"
	auditSystemPrompt = "You are auditing whether an AI agent complied with its high-priority " +
		"memory instructions during a session. For each instruction, determine compliance."
	auditsDirName        = "audits"
	dirPerm              = os.FileMode(0o755)
	evaluationsDirName   = "evaluations"
	filePerm             = os.FileMode(0o644)
	surfacingLogFilename = "surfacing-log.jsonl"
	topPercentDivisor    = 5 // 100/20 = top 20%
)

// evalEntry is the JSON structure for injected evaluation signals.
//
//nolint:tagliatelle // spec requires snake_case JSON field names.
type evalEntry struct {
	MemoryPath string `json:"memory_path"`
	Outcome    string `json:"outcome"`
	Evidence   string `json:"evidence"`
}

// surfacingLogEntry is one line in surfacing-log.jsonl with optional effectiveness.
//
//nolint:tagliatelle // spec requires snake_case JSON field names.
type surfacingLogEntry struct {
	MemoryPath         string  `json:"memory_path"`
	EffectivenessScore float64 `json:"effectiveness_score"`
}

func buildCompliancePrompt(scope []ScopeEntry, transcript string) string {
	var sb strings.Builder

	sb.WriteString("High-priority instructions to audit:\n")

	for _, entry := range scope {
		sb.WriteString("- ")
		sb.WriteString(entry.MemoryID)
		fmt.Fprintf(&sb, " (effectiveness: %.1f%%)\n", entry.EffectivenessScore)
	}

	sb.WriteString("\nSession transcript:\n")
	sb.WriteString(transcript)
	sb.WriteString("\n\nReturn a JSON array with one object per instruction:\n")
	sb.WriteString(`[{"instruction": "memory_path", "compliant": true/false, "evidence": "..."}]`)

	return sb.String()
}

func stripMarkdownFence(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "```") {
		return text
	}

	firstNewline := strings.Index(trimmed, "\n")
	if firstNewline < 0 {
		return text
	}

	trimmed = trimmed[firstNewline+1:]

	if idx := strings.LastIndex(trimmed, "```"); idx >= 0 {
		trimmed = trimmed[:idx]
	}

	return strings.TrimSpace(trimmed)
}
