package review_test

import (
	"strings"
	"testing"

	"github.com/onsi/gomega"

	"engram/internal/review"
)

// T-194: engram review outputs budget utilization table
func TestT194_BudgetUtilizationTable(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	stats := []review.BudgetHookStat{
		{Hook: "SessionStart", Budget: 800, TotalSurfaced: 1200, Invocations: 2, CapHits: 0},
		{Hook: "UserPromptSubmit", Budget: 300, TotalSurfaced: 600, Invocations: 4, CapHits: 1},
	}

	var buf strings.Builder
	review.RenderBudget(stats, &buf)
	output := buf.String()

	g.Expect(output).To(gomega.ContainSubstring("[Budget Utilization]"))
	g.Expect(output).To(gomega.ContainSubstring("Hook"))
	g.Expect(output).To(gomega.ContainSubstring("Budget"))
	g.Expect(output).To(gomega.ContainSubstring("Surfaced"))
	g.Expect(output).To(gomega.ContainSubstring("Utilization %"))
	g.Expect(output).To(gomega.ContainSubstring("Warning"))
	g.Expect(output).To(gomega.ContainSubstring("SessionStart"))
	g.Expect(output).To(gomega.ContainSubstring("UserPromptSubmit"))
}

// T-195: Budget warning triggers at >50% cap hit rate
func TestT195_BudgetWarningAtHighCapHitRate(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	stat := review.BudgetHookStat{
		Hook:          "SessionStart",
		Budget:        800,
		TotalSurfaced: 7200,
		Invocations:   10,
		CapHits:       6,
	}

	g.Expect(stat.CapHitRate()).To(gomega.BeNumerically("==", 60.0))
	g.Expect(stat.Warning()).To(gomega.ContainSubstring("Hitting cap on 60% of invocations"))

	// Render and check output contains warning
	var buf strings.Builder
	review.RenderBudget([]review.BudgetHookStat{stat}, &buf)
	output := buf.String()

	g.Expect(output).To(gomega.ContainSubstring("⚠ Hitting cap on 60% of invocations"))
}

// T-196: Budget reporting with zero utilization
func TestT196_BudgetZeroUtilization(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	stat := review.BudgetHookStat{
		Hook:          "PreToolUse",
		Budget:        200,
		TotalSurfaced: 0,
		Invocations:   0,
		CapHits:       0,
	}

	g.Expect(stat.Utilization()).To(gomega.BeNumerically("==", 0))
	g.Expect(stat.Warning()).To(gomega.BeEmpty())

	var buf strings.Builder
	review.RenderBudget([]review.BudgetHookStat{stat}, &buf)
	output := buf.String()

	g.Expect(output).To(gomega.ContainSubstring("PreToolUse"))
	g.Expect(output).To(gomega.ContainSubstring("0"))
	g.Expect(output).NotTo(gomega.ContainSubstring("⚠"))
}
