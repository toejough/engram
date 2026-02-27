package context

import (
	"fmt"

	"github.com/toejough/projctl/internal/log"
)

// BudgetStatus represents the current budget state.
type BudgetStatus int

// BudgetStatus values.
const (
	// BudgetOK indicates context usage is under warning threshold.
	BudgetOK BudgetStatus = iota
	// BudgetWarning indicates context usage is over warning but under limit.
	BudgetWarning
	// BudgetExceeded indicates context usage is over the limit.
	BudgetExceeded
)

// BudgetResult holds the result of a budget check.
type BudgetResult struct {
	Status          BudgetStatus
	CurrentEstimate int
	Percentage      int
	Message         string
	ExitCode        int
}

// BudgetThresholds defines the warning and limit thresholds.
type BudgetThresholds struct {
	Warning int // Tokens at which to warn
	Limit   int // Tokens at which to block
}

// CheckBudget reads the most recent context estimate from the log and compares
// it against the provided thresholds.
func CheckBudget(dir string, thresholds BudgetThresholds, fs log.FileSystem) (BudgetResult, error) {
	entries, err := log.Read(dir, log.ReadOpts{}, fs)
	if err != nil {
		return BudgetResult{}, fmt.Errorf("failed to read log: %w", err)
	}

	// Find the most recent context estimate
	var currentEstimate int

	for _, entry := range entries {
		if entry.ContextEstimate > 0 {
			currentEstimate = entry.ContextEstimate
		}
	}

	// Calculate percentage based on limit
	percentage := 0
	if thresholds.Limit > 0 {
		percentage = (currentEstimate * 100) / thresholds.Limit
	}

	// Determine status and exit code
	var (
		status   BudgetStatus
		exitCode int
		message  string
	)

	switch {
	case currentEstimate >= thresholds.Limit:
		status = BudgetExceeded
		exitCode = 2
		message = fmt.Sprintf("Context at %d%% - limit exceeded, compaction required", percentage)
	case currentEstimate >= thresholds.Warning:
		status = BudgetWarning
		exitCode = 1
		message = fmt.Sprintf("Context at %d%% - consider compaction", percentage)
	default:
		status = BudgetOK
		exitCode = 0
		message = fmt.Sprintf("Context at %d%% - within budget", percentage)
	}

	return BudgetResult{
		Status:          status,
		CurrentEstimate: currentEstimate,
		Percentage:      percentage,
		Message:         message,
		ExitCode:        exitCode,
	}, nil
}

// DefaultBudgetThresholds returns the default thresholds.
func DefaultBudgetThresholds() BudgetThresholds {
	return BudgetThresholds{
		Warning: 80000,
		Limit:   90000,
	}
}
