package cli_test

import (
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestRenderFrontmatter_Fact(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	got := cli.ExportRenderFactFrontmatter(cli.ExportFactFields{
		Situation: "reasoning about agent coordination",
		Subject:   "subagent dispatch",
		Predicate: "is fundamentally",
		Object:    "a verification problem dressed as coordination",
		Luhmann:   "11",
		Source:    "session log bar, 2026-05-09 13:00 UTC",
	}, when)
	g.Expect(got).To(ContainSubstring("type: fact"))
	g.Expect(got).To(ContainSubstring("subject: subagent dispatch"))
}

func TestRenderFrontmatter_Feedback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	got := cli.ExportRenderFeedbackFrontmatter(cli.ExportFeedbackFields{
		Situation: "writing concurrent Go code with context",
		Behavior:  "ignoring context cancellation",
		Impact:    "leaks goroutines on shutdown",
		Action:    "always check ctx.Done() in select loops",
		Luhmann:   "9z",
		Source:    "session log foo, 2026-05-09 12:00 UTC",
	}, when)
	g.Expect(got).To(Equal(strings.Join([]string{
		"---",
		"type: feedback",
		"situation: writing concurrent Go code with context",
		"behavior: ignoring context cancellation",
		"impact: leaks goroutines on shutdown",
		"action: always check ctx.Done() in select loops",
		`luhmann: "9z"`,
		"created: 2026-05-09",
		"source: session log foo, 2026-05-09 12:00 UTC",
		"---",
		"",
	}, "\n")))
}

func TestRenderFrontmatter_MOC(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	got := cli.ExportRenderMOCFrontmatter(cli.ExportMOCFields{
		Topic:   "llm rationalization patterns under pressure",
		Luhmann: "5",
		Source:  "constructed from cluster analysis, 2026-05-09",
	}, when)
	g.Expect(got).To(ContainSubstring("type: moc"))
	g.Expect(got).To(ContainSubstring("topic: llm rationalization patterns under pressure"))
}
