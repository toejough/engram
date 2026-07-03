package cli_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestEvaluateVocabTriggers_GrowthBelowDaysFloor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// growth >= 40 but only 4 days: no fire
	last := &cli.ExportVocabLastRefitDoc{NoteCount: 100, Date: "2026-06-29"}
	now := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC) // 4 days

	fired, _ := cli.ExportEvaluateVocabTriggers(141, 5, nil, last, now)

	g.Expect(fired).To(BeFalse())
}

// ── Task 2: evaluateVocabTriggers ────────────────────────────────────────────

func TestEvaluateVocabTriggers_GrowthFires(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// growth >= 40 AND >= 14d: fires
	last := &cli.ExportVocabLastRefitDoc{NoteCount: 100, Date: "2026-06-15"}
	now := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC) // 18 days later

	fired, reason := cli.ExportEvaluateVocabTriggers(141, 5, nil, last, now)

	g.Expect(fired).To(BeTrue())
	g.Expect(reason).To(ContainSubstring("growth"))
}

func TestEvaluateVocabTriggers_HubFires(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	last := &cli.ExportVocabLastRefitDoc{NoteCount: 130, Date: "2026-06-01"}
	now := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)

	// term "x" has 30/100 = 30% > 25%
	counts := map[string]int{"x": 30, "y": 5}

	fired, reason := cli.ExportEvaluateVocabTriggers(100, 0, counts, last, now)

	g.Expect(fired).To(BeTrue())
	g.Expect(reason).To(ContainSubstring("hub"))
}

func TestEvaluateVocabTriggers_NilLastRefit_NoFire(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)

	fired, _ := cli.ExportEvaluateVocabTriggers(100, 5, nil, nil, now)

	g.Expect(fired).To(BeFalse())
}

func TestEvaluateVocabTriggers_UntaggedRateFires(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	last := &cli.ExportVocabLastRefitDoc{NoteCount: 130, Date: "2026-06-01"}
	now := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)

	// 10/100 = 10% > 8%
	fired, reason := cli.ExportEvaluateVocabTriggers(100, 10, nil, last, now)

	g.Expect(fired).To(BeTrue())
	g.Expect(reason).To(ContainSubstring("untagged"))
}
