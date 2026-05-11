package cli_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestExtractLuhmannFromFilename(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got, ok := cli.ExportExtractLuhmannFromFilename("1a3.2026-05-09.subagent-recovery.md")
	g.Expect(ok).To(BeTrue())
	g.Expect(got).To(Equal("1a3"))
}

func TestExtractLuhmannFromFilename_RejectsBadFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	_, ok := cli.ExportExtractLuhmannFromFilename("README.md")
	g.Expect(ok).To(BeFalse())
}

func TestLearnPath_MOC(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	got := cli.ExportLearnPath("/vault", "moc", "5", "llm-rationalization-patterns", when)
	g.Expect(got).To(Equal("/vault/MOCs/5.2026-05-09.llm-rationalization-patterns.md"))
}

func TestLearnPath_Permanent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	got := cli.ExportLearnPath("/vault", "feedback", "1a3", "subagent-driven-recovery", when)
	g.Expect(got).To(Equal("/vault/Permanent/1a3.2026-05-09.subagent-driven-recovery.md"))
}

func TestRenderBody_Fact(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportRenderFactBody(cli.ExportFactFields{
		Situation: "reasoning about agent coordination",
		Subject:   "subagent dispatch",
		Predicate: "is fundamentally",
		Object:    "a verification problem dressed as coordination",
	}, "Related to:\n- [[X]] — adjacent.\n")
	g.Expect(got).To(Equal(
		"Information learned: when in reasoning about agent coordination, " +
			"subagent dispatch is fundamentally a verification problem dressed as coordination.\n" +
			"\n" +
			"Related to:\n- [[X]] — adjacent.\n"))
}

func TestRenderBody_Feedback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	action := "set up a task list with self-contained briefs and dispatch; " +
		"if a small model cannot finish a subtask, shrink the task"
	got := cli.ExportRenderFeedbackBody(cli.ExportFeedbackFields{
		Situation: "orchestrating multi-step work as the main LLM under context pressure",
		Action:    action,
	}, "Related to:\n- [[1a.foo]] — same shape.\n- [[5.bar]] — the MOC.\n")
	g.Expect(got).To(Equal(
		"Lesson learned: when orchestrating multi-step work as the main LLM under context pressure, " +
			action + ".\n" +
			"\n" +
			"Related to:\n- [[1a.foo]] — same shape.\n- [[5.bar]] — the MOC.\n"))
}

func TestRenderBody_MOC_FramingOnly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportRenderMOCBody("This cluster names a recurring pattern of LLM rationalization under pressure.", "")
	g.Expect(got).To(Equal("This cluster names a recurring pattern of LLM rationalization under pressure.\n"))
}

func TestRenderBody_MOC_FramingPlusRelated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportRenderMOCBody("framing prose", "Related to:\n- [[X]] — r.\n")
	g.Expect(got).To(Equal("framing prose\n\nRelated to:\n- [[X]] — r.\n"))
}

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

func TestRenderRelatedSection_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportRenderRelatedSection(nil)).To(Equal(""))
}

func TestRenderRelatedSection_MultipleEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportRenderRelatedSection([]string{
		"1a.foo|same shape",
		"5.bar | the MOC",
	})
	g.Expect(got).To(Equal(
		"Related to:\n- [[1a.foo]] — same shape.\n- [[5.bar]] — the MOC.\n"))
}

func TestRenderRelatedSection_NoPipeMeansEmptyRationale(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportRenderRelatedSection([]string{"7"})
	g.Expect(got).To(Equal("Related to:\n- [[7]] — .\n"))
}

func TestRunLearn_Fact_WritesExpectedFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var (
		writtenPath    string
		writtenContent []byte
	)

	deps := cli.LearnDeps{
		Now:     func() time.Time { return time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC) },
		Getenv:  func(string) string { return "" },
		StatDir: func(string) error { return nil },
		ListIDs: func(string) ([]string, error) { return nil, nil },
		Lock:    func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(path string, data []byte) error {
			writtenPath = path
			writtenContent = data

			return nil
		},
	}

	args := cli.LearnArgs{
		Type:      "fact",
		Slug:      "fact-slug",
		Vault:     "/vault",
		Position:  "top",
		Situation: "s",
		Subject:   "subj",
		Predicate: "is",
		Object:    "obj",
	}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(writtenPath).To(Equal("/vault/Permanent/1.2026-05-09.fact-slug.md"))
	g.Expect(string(writtenContent)).To(ContainSubstring("type: fact"))
	g.Expect(string(writtenContent)).To(ContainSubstring("Information learned"))
}

func TestRunLearn_Feedback_WritesExpectedFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var (
		lockAcquired, lockReleased bool
		writtenPath                string
		writtenContent             []byte
	)

	deps := cli.LearnDeps{
		Now:     func() time.Time { return time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC) },
		Getenv:  func(string) string { return "" },
		StatDir: func(string) error { return nil },
		ListIDs: func(string) ([]string, error) {
			return []string{"1", "2"}, nil
		},
		Lock: func(string) (func(), error) {
			lockAcquired = true

			return func() { lockReleased = true }, nil
		},
		WriteNew: func(path string, data []byte) error {
			writtenPath = path
			writtenContent = data

			return nil
		},
	}

	args := cli.LearnArgs{
		Type:      "feedback",
		Slug:      "ctx-cancellation-rule",
		Vault:     "/vault",
		Target:    "",
		Position:  "top",
		Source:    "session log foo, 2026-05-09 12:00 UTC",
		Situation: "writing concurrent Go code",
		Behavior:  "ignoring ctx.Done()",
		Impact:    "leaks goroutines",
		Action:    "always check ctx.Done() in select",
	}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(lockAcquired).To(BeTrue())
	g.Expect(lockReleased).To(BeTrue())
	g.Expect(writtenPath).To(Equal("/vault/Permanent/3.2026-05-09.ctx-cancellation-rule.md"))
	g.Expect(string(writtenContent)).To(ContainSubstring("type: feedback"))
	g.Expect(string(writtenContent)).To(ContainSubstring("Lesson learned: when writing concurrent Go code"))
}

func TestRunLearn_MOC_WritesExpectedFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var writtenPath string

	deps := cli.LearnDeps{
		Now:     func() time.Time { return time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC) },
		Getenv:  func(string) string { return "" },
		StatDir: func(string) error { return nil },
		ListIDs: func(string) ([]string, error) { return nil, nil },
		Lock:    func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(path string, _ []byte) error {
			writtenPath = path

			return nil
		},
	}

	args := cli.LearnArgs{
		Type:     "moc",
		Slug:     "moc-slug",
		Vault:    "/vault",
		Position: "top",
		Topic:    "the topic",
	}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(writtenPath).To(Equal("/vault/MOCs/1.2026-05-09.moc-slug.md"))
}

func TestRunLearn_PropagatesListIDsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.LearnDeps{
		Now:      func() time.Time { return time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC) },
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		ListIDs:  func(string) ([]string, error) { return nil, errors.New("io fail") },
		Lock:     func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(string, []byte) error { return nil },
	}
	args := cli.LearnArgs{Type: "moc", Slug: "x", Vault: "/v", Position: "top", Topic: "t"}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).To(MatchError(ContainSubstring("listing existing IDs")))
}

func TestRunLearn_PropagatesLockError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.LearnDeps{
		Now:      func() time.Time { return time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC) },
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		ListIDs:  func(string) ([]string, error) { return nil, nil },
		Lock:     func(string) (func(), error) { return nil, errors.New("locked") },
		WriteNew: func(string, []byte) error { return nil },
	}
	args := cli.LearnArgs{Type: "moc", Slug: "x", Vault: "/v", Position: "top", Topic: "t"}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).To(MatchError(ContainSubstring("acquiring lock")))
}

func TestRunLearn_PropagatesStatDirError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.LearnDeps{
		Now:      time.Now,
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return errors.New("nope") },
		ListIDs:  func(string) ([]string, error) { return nil, nil },
		Lock:     func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(string, []byte) error { return nil },
	}
	args := cli.LearnArgs{Type: "moc", Slug: "x", Vault: "/v", Position: "top"}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).To(MatchError(ContainSubstring("vault")))
}

func TestRunLearn_RejectsInvalidSlug(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.LearnDeps{
		Now:      time.Now,
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		ListIDs:  func(string) ([]string, error) { return nil, nil },
		Lock:     func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(string, []byte) error { return nil },
	}
	args := cli.LearnArgs{Type: "moc", Slug: "Bad Slug", Vault: "/v", Position: "top"}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).To(HaveOccurred())
}

func TestRunLearn_RejectsMissingVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.LearnDeps{
		Now:      time.Now,
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		ListIDs:  func(string) ([]string, error) { return nil, nil },
		Lock:     func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(string, []byte) error { return nil },
	}
	args := cli.LearnArgs{Type: "moc", Slug: "x", Vault: "", Position: "top"}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).To(HaveOccurred())
}

func TestRunLearn_RejectsUnknownType(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	deps := cli.LearnDeps{
		Now:      time.Now,
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		ListIDs:  func(string) ([]string, error) { return nil, nil },
		Lock:     func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(string, []byte) error { return nil },
	}
	args := cli.LearnArgs{Type: "principle", Slug: "x", Vault: "/v", Position: "top"}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).To(HaveOccurred())
}
