package review

import (
	"fmt"
	"io"
)

// BudgetHookStat holds budget utilization data for a single hook type.
type BudgetHookStat struct {
	Hook          string
	Budget        int
	TotalSurfaced int
	Invocations   int
	CapHits       int
}

// Utilization returns the average utilization percentage for this hook.
func (s BudgetHookStat) Utilization() float64 {
	if s.Budget == 0 || s.Invocations == 0 {
		return 0
	}

	avg := float64(s.TotalSurfaced) / float64(s.Invocations)

	return (avg / float64(s.Budget)) * percentMultiplier
}

// CapHitRate returns the percentage of invocations that hit the budget cap.
func (s BudgetHookStat) CapHitRate() float64 {
	if s.Invocations == 0 {
		return 0
	}

	return float64(s.CapHits) / float64(s.Invocations) * percentMultiplier
}

// Warning returns a warning string if cap hit rate exceeds the threshold, or "".
func (s BudgetHookStat) Warning() string {
	rate := s.CapHitRate()
	if rate > capHitWarningThreshold {
		return fmt.Sprintf("⚠ Hitting cap on %.0f%% of invocations", rate)
	}

	return ""
}

// RenderBudget writes a [Budget Utilization] section to w.
func RenderBudget(stats []BudgetHookStat, w io.Writer) {
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "  [Budget Utilization]\n")
	_, _ = fmt.Fprintf(w, "    %-20s %8s %10s %14s   %s\n",
		"Hook", "Budget", "Surfaced", "Utilization %", "Warning")

	for _, stat := range stats {
		avgSurfaced := 0
		if stat.Invocations > 0 {
			avgSurfaced = stat.TotalSurfaced / stat.Invocations
		}

		warning := stat.Warning()

		_, _ = fmt.Fprintf(w, "    %-20s %8d %10d %13.0f%%   %s\n",
			stat.Hook, stat.Budget, avgSurfaced, stat.Utilization(), warning)
	}
}

const (
	capHitWarningThreshold = 50.0
	percentMultiplier      = 100.0
)
