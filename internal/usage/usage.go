// Package usage provides token usage reporting from project logs.
package usage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/toejough/projctl/internal/log"
)

// CheckStatus represents the budget check result status.
type CheckStatus int

const (
	StatusOK      CheckStatus = 0
	StatusWarning CheckStatus = 1
	StatusLimit   CheckStatus = 2
)

// BudgetConfig holds token budget thresholds.
type BudgetConfig struct {
	WarningTokens int `toml:"warning_tokens"`
	LimitTokens   int `toml:"limit_tokens"`
}

// CheckResult holds the result of a budget check.
type CheckResult struct {
	Status         CheckStatus
	TotalTokens    int
	Recommendation string
}

// UsageReport contains aggregated token usage data.
type UsageReport struct {
	TotalTokens int            `json:"total_tokens"`
	EntryCount  int            `json:"entry_count"`
	ByModel     map[string]int `json:"by_model"`
}

// ReportOpts holds options for generating a usage report.
type ReportOpts struct {
	Model   string // Filter to specific model (empty = all)
	Session string // Filter to specific session (empty = all)
}

// Report generates a token usage report from project logs.
func Report(dir string, opts ReportOpts) (UsageReport, error) {
	entries, err := log.Read(dir, log.ReadOpts{
		Model:   opts.Model,
		Session: opts.Session,
	})
	if err != nil {
		return UsageReport{}, err
	}

	report := UsageReport{
		ByModel: make(map[string]int),
	}

	for _, entry := range entries {
		report.TotalTokens += entry.TokensEstimate
		report.EntryCount++
		report.ByModel[entry.Model] += entry.TokensEstimate
	}

	return report, nil
}

// ReportByProject generates a token usage report for a named project.
// The project is looked up in projctlDir/projects/{projectName}/.
func ReportByProject(projectName, projctlDir string, opts ReportOpts) (UsageReport, error) {
	projectDir := filepath.Join(projctlDir, "projects", projectName)

	// Check if project directory exists
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return UsageReport{}, fmt.Errorf("project not found: %s", projectName)
	}

	return Report(projectDir, opts)
}

// Check compares current token usage against budget thresholds.
func Check(dir string, budget BudgetConfig) CheckResult {
	report, err := Report(dir, ReportOpts{})
	if err != nil {
		return CheckResult{Status: StatusOK, TotalTokens: 0}
	}

	result := CheckResult{
		TotalTokens: report.TotalTokens,
		Status:      StatusOK,
	}

	// Check limit first (higher priority)
	if budget.LimitTokens > 0 && report.TotalTokens >= budget.LimitTokens {
		result.Status = StatusLimit
		result.Recommendation = "token limit exceeded"
		return result
	}

	// Check warning threshold
	if budget.WarningTokens > 0 && report.TotalTokens >= budget.WarningTokens {
		result.Status = StatusWarning
		result.Recommendation = "consider using haiku for remaining tasks"
		return result
	}

	return result
}
