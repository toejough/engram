// Package usage provides token usage reporting from project logs.
package usage

import (
	"github.com/toejough/projctl/internal/log"
)

// UsageReport contains aggregated token usage data.
type UsageReport struct {
	TotalTokens int            `json:"total_tokens"`
	EntryCount  int            `json:"entry_count"`
	ByModel     map[string]int `json:"by_model"`
}

// ReportOpts holds options for generating a usage report.
type ReportOpts struct {
	Model string // Filter to specific model (empty = all)
}

// Report generates a token usage report from project logs.
func Report(dir string, opts ReportOpts) (UsageReport, error) {
	entries, err := log.Read(dir, log.ReadOpts{
		Model: opts.Model,
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
