package context_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/context"
)

// TEST-700 traces: TASK-065
func TestBudgetCheck(t *testing.T) {
	t.Run("returns OK when under warning threshold", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create log with context estimates under warning threshold
		logContent := `{"timestamp":"2026-01-31T12:00:00Z","level":"status","subject":"task-status","message":"test","context_estimate":50000}
{"timestamp":"2026-01-31T12:01:00Z","level":"status","subject":"task-status","message":"test2","context_estimate":60000}
`
		_ = os.WriteFile(filepath.Join(dir, "updates.jsonl"), []byte(logContent), 0o644)

		result, err := context.CheckBudget(dir, context.BudgetThresholds{
			Warning: 80000,
			Limit:   90000,
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Status).To(Equal(context.BudgetOK))
		g.Expect(result.CurrentEstimate).To(Equal(60000))
		g.Expect(result.ExitCode).To(Equal(0))
	})

	t.Run("returns warning when over warning but under limit", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		logContent := `{"timestamp":"2026-01-31T12:00:00Z","level":"status","subject":"task-status","message":"test","context_estimate":85000}
`
		_ = os.WriteFile(filepath.Join(dir, "updates.jsonl"), []byte(logContent), 0o644)

		result, err := context.CheckBudget(dir, context.BudgetThresholds{
			Warning: 80000,
			Limit:   90000,
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Status).To(Equal(context.BudgetWarning))
		g.Expect(result.CurrentEstimate).To(Equal(85000))
		g.Expect(result.ExitCode).To(Equal(1))
		g.Expect(result.Message).To(ContainSubstring("consider compaction"))
	})

	t.Run("returns limit exceeded when over limit", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		logContent := `{"timestamp":"2026-01-31T12:00:00Z","level":"status","subject":"task-status","message":"test","context_estimate":95000}
`
		_ = os.WriteFile(filepath.Join(dir, "updates.jsonl"), []byte(logContent), 0o644)

		result, err := context.CheckBudget(dir, context.BudgetThresholds{
			Warning: 80000,
			Limit:   90000,
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Status).To(Equal(context.BudgetExceeded))
		g.Expect(result.CurrentEstimate).To(Equal(95000))
		g.Expect(result.ExitCode).To(Equal(2))
	})

	t.Run("uses most recent context estimate from log", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Multiple entries with different estimates - should use the most recent
		logContent := `{"timestamp":"2026-01-31T12:00:00Z","level":"status","subject":"task-status","message":"test","context_estimate":50000}
{"timestamp":"2026-01-31T12:01:00Z","level":"status","subject":"task-status","message":"test2","context_estimate":75000}
{"timestamp":"2026-01-31T12:02:00Z","level":"status","subject":"task-status","message":"test3","context_estimate":60000}
`
		_ = os.WriteFile(filepath.Join(dir, "updates.jsonl"), []byte(logContent), 0o644)

		result, err := context.CheckBudget(dir, context.BudgetThresholds{
			Warning: 80000,
			Limit:   90000,
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.CurrentEstimate).To(Equal(60000)) // Most recent
	})

	t.Run("returns OK when no log entries", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// No log file
		result, err := context.CheckBudget(dir, context.BudgetThresholds{
			Warning: 80000,
			Limit:   90000,
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Status).To(Equal(context.BudgetOK))
		g.Expect(result.CurrentEstimate).To(Equal(0))
	})

	t.Run("calculates percentage correctly", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		logContent := `{"timestamp":"2026-01-31T12:00:00Z","level":"status","subject":"task-status","message":"test","context_estimate":45000}
`
		_ = os.WriteFile(filepath.Join(dir, "updates.jsonl"), []byte(logContent), 0o644)

		result, err := context.CheckBudget(dir, context.BudgetThresholds{
			Warning: 80000,
			Limit:   90000,
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Percentage).To(Equal(50)) // 45000 / 90000 = 50%
	})
}

func TestDefaultThresholds(t *testing.T) {
	g := NewWithT(t)

	thresholds := context.DefaultBudgetThresholds()
	g.Expect(thresholds.Warning).To(Equal(80000))
	g.Expect(thresholds.Limit).To(Equal(90000))
}
