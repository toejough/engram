package usage_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/log"
	"github.com/toejough/projctl/internal/usage"
	"pgregory.net/rapid"
)

func nowFunc() func() time.Time {
	return func() time.Time { return time.Date(2026, 1, 27, 12, 0, 0, 0, time.UTC) }
}

// TEST-510 traces: TASK-028
// Test Report sums tokens from log entries.
func TestReport_SumsTokens(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Write entries with different token counts
	_ = log.Write(dir, "status", "task-status", "msg1", log.WriteOpts{Tokens: 100}, nowFunc())
	_ = log.Write(dir, "status", "task-status", "msg2", log.WriteOpts{Tokens: 200}, nowFunc())
	_ = log.Write(dir, "status", "task-status", "msg3", log.WriteOpts{Tokens: 50}, nowFunc())

	report, err := usage.Report(dir, usage.ReportOpts{})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(report.TotalTokens).To(Equal(350))
	g.Expect(report.EntryCount).To(Equal(3))
}

// TEST-511 traces: TASK-028
// Test Report provides breakdown by model.
func TestReport_BreakdownByModel(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	_ = log.Write(dir, "status", "task-status", "haiku1", log.WriteOpts{Tokens: 100, Model: "haiku"}, nowFunc())
	_ = log.Write(dir, "status", "task-status", "haiku2", log.WriteOpts{Tokens: 50, Model: "haiku"}, nowFunc())
	_ = log.Write(dir, "status", "task-status", "sonnet1", log.WriteOpts{Tokens: 200, Model: "sonnet"}, nowFunc())
	_ = log.Write(dir, "status", "task-status", "opus1", log.WriteOpts{Tokens: 500, Model: "opus"}, nowFunc())
	_ = log.Write(dir, "status", "task-status", "nomodel", log.WriteOpts{Tokens: 25}, nowFunc())

	report, err := usage.Report(dir, usage.ReportOpts{})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(report.TotalTokens).To(Equal(875))
	g.Expect(report.ByModel["haiku"]).To(Equal(150))
	g.Expect(report.ByModel["sonnet"]).To(Equal(200))
	g.Expect(report.ByModel["opus"]).To(Equal(500))
	g.Expect(report.ByModel[""]).To(Equal(25))
}

// TEST-512 traces: TASK-028
// Test Report returns empty result for empty log.
func TestReport_EmptyLog(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	report, err := usage.Report(dir, usage.ReportOpts{})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(report.TotalTokens).To(Equal(0))
	g.Expect(report.EntryCount).To(Equal(0))
}

// TEST-513 traces: TASK-028
// Test Report filters by model.
func TestReport_FilterByModel(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	_ = log.Write(dir, "status", "task-status", "haiku1", log.WriteOpts{Tokens: 100, Model: "haiku"}, nowFunc())
	_ = log.Write(dir, "status", "task-status", "sonnet1", log.WriteOpts{Tokens: 200, Model: "sonnet"}, nowFunc())

	report, err := usage.Report(dir, usage.ReportOpts{Model: "haiku"})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(report.TotalTokens).To(Equal(100))
	g.Expect(report.EntryCount).To(Equal(1))
}

// TEST-514 traces: TASK-028
// Test Report property: total equals sum of by-model values.
func TestReport_PropertyTotalMatchesBreakdown(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		models := []string{"haiku", "sonnet", "opus", ""}
		count := rapid.IntRange(1, 20).Draw(rt, "count")

		for i := 0; i < count; i++ {
			tokens := rapid.IntRange(1, 1000).Draw(rt, "tokens")
			model := rapid.SampledFrom(models).Draw(rt, "model")
			_ = log.Write(dir, "status", "task-status", "msg", log.WriteOpts{
				Tokens: tokens,
				Model:  model,
			}, nowFunc())
		}

		report, err := usage.Report(dir, usage.ReportOpts{})
		g.Expect(err).ToNot(HaveOccurred())

		// Sum of breakdown should equal total
		var sum int
		for _, v := range report.ByModel {
			sum += v
		}
		g.Expect(sum).To(Equal(report.TotalTokens))
	})
}
