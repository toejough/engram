package cli

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// TestCapRedFlagsForPreview_KeepsNewestEntryWhenOverBudget reproduces
// engram#763: a red_flags list whose combined size exceeds the preview
// budget silently dropped its most-recently-appended (last) entry under
// Claude Code's own ~2KB Bash-tool-output truncation. Entries are appended
// by callers (engram learn runbook / amend --red-flag), so the newest entry
// is conventionally last — capRedFlagsForPreview must keep entries from the
// END of the list, not the start, and must leave an explicit in-band marker
// naming the omission rather than silently dropping the rest.
func TestCapRedFlagsForPreview_KeepsNewestEntryWhenOverBudget(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var b strings.Builder

	b.WriteString("---\ntype: runbook\nsituation: x\ndone_when: y\nred_flags:\n")

	for range 20 {
		b.WriteString("    - " + strings.Repeat("filler ", 20) + "\n")
	}

	b.WriteString("    - the newest entry added to fix a specific defect\n")
	b.WriteString("luhmann: \"1\"\n---\n\nbody\n")

	content := b.String()

	got := capRedFlagsForPreview(content)

	g.Expect(len(got)).To(BeNumerically("<", len(content)),
		"expected the oversized red_flags block to shrink")
	g.Expect(got).To(ContainSubstring("the newest entry added to fix a specific defect"),
		"the most-recently-appended red_flags entry must survive truncation")
	g.Expect(got).To(ContainSubstring("EARLIER RED_FLAGS OMITTED"),
		"the omission must be visible in-band, not silent")
	g.Expect(got).To(HaveSuffix("luhmann: \"1\"\n---\n\nbody\n"),
		"content after the red_flags block must be untouched")
	g.Expect(got).To(HavePrefix("---\ntype: runbook\nsituation: x\ndone_when: y\nred_flags:\n"),
		"content before the red_flags block must be untouched")
}

// TestCapRedFlagsForPreview_NoRedFlagsBlockUnchanged proves a note without a
// red_flags field at all (fact/feedback kinds, or a runbook with none set)
// passes through unmodified.
func TestCapRedFlagsForPreview_NoRedFlagsBlockUnchanged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: fact\nsituation: x\n---\n\nbody\n"

	g.Expect(capRedFlagsForPreview(content)).To(Equal(content))
}

// TestCapRedFlagsForPreview_UnderBudgetUnchanged proves a red_flags block
// already within the preview budget is returned byte-identical — this
// function must not touch notes that were never at truncation risk.
func TestCapRedFlagsForPreview_UnderBudgetUnchanged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: runbook\nsituation: x\ndone_when: y\n" +
		"red_flags:\n    - short one\n    - short two\n" +
		"luhmann: \"1\"\n---\n\nbody\n"

	g.Expect(capRedFlagsForPreview(content)).To(Equal(content))
}
