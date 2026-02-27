package memory

import (
	"testing"

	. "github.com/onsi/gomega"
)

// TestDetectFlags_HookViolationsIncreasing verifies flag added for increasing violations.
func TestDetectFlags_HookViolationsIncreasing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	metrics := &MetricsSnapshot{
		HookViolationTrend: map[string]string{
			"no-skip-tests": "degrading",
		},
	}

	flags := detectFlags(nil, metrics)

	g.Expect(flags).To(ContainElement(ContainSubstring("violations")))
}

// TestDetectFlags_LowRetrievalPrecision verifies flag added when precision < 0.5.
func TestDetectFlags_LowRetrievalPrecision(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	metrics := &MetricsSnapshot{
		RetrievalPrecision: 0.4,
	}

	flags := detectFlags(nil, metrics)

	g.Expect(flags).To(ContainElement(ContainSubstring("precision")))
}

// TestDetectFlags_NilMetrics verifies no metrics-based flags when metrics is nil.
func TestDetectFlags_NilMetrics(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	flags := detectFlags(nil, nil)

	g.Expect(flags).To(BeEmpty())
}

// TestDetectFlags_RecurrenceCountZero verifies no flag when count <= 1.
func TestDetectFlags_RecurrenceCountZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entries := []ChangelogEntry{
		{
			Action: "correction_recurrence",
			Metadata: map[string]string{
				"count": "1", // count == 1, threshold is > 1
			},
		},
	}

	flags := detectFlags(entries, nil)

	for _, f := range flags {
		g.Expect(f).ToNot(ContainSubstring("Recurring"))
	}
}

// TestDetectFlags_RetrievalPrecisionAboveThreshold verifies no flag when precision >= 0.5.
func TestDetectFlags_RetrievalPrecisionAboveThreshold(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	metrics := &MetricsSnapshot{
		RetrievalPrecision: 0.7,
	}

	flags := detectFlags(nil, metrics)

	for _, f := range flags {
		g.Expect(f).ToNot(ContainSubstring("precision"))
	}
}

// TestDetectFlags_SkillsAwaitingTest verifies flag added when skills await testing.
func TestDetectFlags_SkillsAwaitingTest(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	metrics := &MetricsSnapshot{
		SkillsAwaitingTest: 3,
	}

	flags := detectFlags(nil, metrics)

	g.Expect(flags).To(ContainElement(ContainSubstring("awaiting")))
}
