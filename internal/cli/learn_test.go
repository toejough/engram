package cli_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
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

// TestMarshalFrontmatter_WrapsValidValue verifies the helper produces the
// expected "---"-delimited block. Error returns are unreachable for the
// typed-string struct callers used in production, so only the happy path is
// covered here.
func TestMarshalFrontmatter_WrapsValidValue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportMarshalFrontmatter(map[string]string{"k": "v"})
	g.Expect(got).To(Equal("---\nk: v\n---\n\n"))
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
			"Related to:\n- [[1a.foo]] — same shape.\n- [[5.bar]] — the MOC.\n",
	))
}

func TestRenderBody_MOC_FramingOnly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportRenderMOCBody(
		"This cluster names a recurring pattern of LLM rationalization under pressure.",
		"",
	)
	g.Expect(got).
		To(Equal("This cluster names a recurring pattern of LLM rationalization under pressure.\n"))
}

func TestRenderBody_MOC_FramingPlusRelated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportRenderMOCBody("framing prose", "Related to:\n- [[X]] — r.\n")
	g.Expect(got).To(Equal("framing prose\n\nRelated to:\n- [[X]] — r.\n"))
}

// TestRenderFactFrontmatter_SafelyEncodesTrickyValues mirrors the feedback
// safety check for the fact frontmatter.
func TestRenderFactFrontmatter_SafelyEncodesTrickyValues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	fields := cli.ExportFactFields{
		Situation: "context: tricky",
		Subject:   "- subj",
		Predicate: "is\nmultiline",
		Object:    "* obj",
		Luhmann:   "11",
		Source:    "src",
	}
	got := cli.ExportRenderFactFrontmatter(fields, when)
	parsed := parseFrontmatter(t, got)
	g.Expect(parsed["situation"]).To(Equal(fields.Situation))
	g.Expect(parsed["subject"]).To(Equal(fields.Subject))
	g.Expect(parsed["predicate"]).To(Equal(fields.Predicate))
	g.Expect(parsed["object"]).To(Equal(fields.Object))
}

// TestRenderFeedbackFrontmatter_RoundtripFidelity is a property test: for any
// printable string values, the rendered frontmatter parses back to the same
// values. This is the invariant the YAML library buys us — verify it holds
// across the input space, not just hand-picked examples.
func TestRenderFeedbackFrontmatter_RoundtripFidelity(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		// Restricted to printable ASCII plus newline. Tab is excluded because
		// yaml.v3's block-scalar emitter and parser disagree about indented
		// tabs; CLI flag values don't carry tabs in practice, so this is not
		// a meaningful gap for engram learn.
		gen := rapid.StringMatching(`[ -~\n]{0,40}`)
		fields := cli.ExportFeedbackFields{
			Situation: gen.Draw(rt, "situation"),
			Behavior:  gen.Draw(rt, "behavior"),
			Impact:    gen.Draw(rt, "impact"),
			Action:    gen.Draw(rt, "action"),
			Luhmann:   rapid.StringMatching(`[0-9][0-9a-z]{0,3}`).Draw(rt, "luhmann"),
			Source:    gen.Draw(rt, "source"),
		}
		when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
		got := cli.ExportRenderFeedbackFrontmatter(fields, when)

		// Use Unmarshal directly to surface decode errors as property failures.
		const delim = "---\n"

		body := strings.TrimPrefix(got, delim)
		end := strings.Index(body, "\n"+delim)

		if end < 0 {
			rt.Fatalf("no closing delimiter in %q", got)
		}

		parsed := map[string]string{}

		if err := yaml.Unmarshal([]byte(body[:end+1]), &parsed); err != nil {
			rt.Fatalf("unmarshal %q: %v", body[:end+1], err)
		}

		for key, want := range map[string]string{
			"situation": fields.Situation, "behavior": fields.Behavior,
			"impact": fields.Impact, "action": fields.Action,
			"luhmann": fields.Luhmann, "source": fields.Source,
		} {
			if parsed[key] != want {
				rt.Fatalf("%s: got %q want %q\nfull:\n%s", key, parsed[key], want, got)
			}
		}
	})
}

// TestRenderFeedbackFrontmatter_SafelyEncodesTrickyValues verifies that values
// containing YAML-significant characters (newlines, colons, leading dashes,
// asterisks) survive a roundtrip — the original bug was that raw string
// concatenation let a multi-line Behavior end the frontmatter mid-document.
func TestRenderFeedbackFrontmatter_SafelyEncodesTrickyValues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	fields := cli.ExportFeedbackFields{
		Situation: "writing tests: a guide",
		Behavior:  "first line\nsecond line",
		Impact:    "- leading dash list marker",
		Action:    "* alias-looking marker",
		Luhmann:   "11",
		Source:    "src: with colon",
	}
	got := cli.ExportRenderFeedbackFrontmatter(fields, when)
	parsed := parseFrontmatter(t, got)
	g.Expect(parsed["situation"]).To(Equal(fields.Situation))
	g.Expect(parsed["behavior"]).To(Equal(fields.Behavior))
	g.Expect(parsed["impact"]).To(Equal(fields.Impact))
	g.Expect(parsed["action"]).To(Equal(fields.Action))
	g.Expect(parsed["luhmann"]).To(Equal(fields.Luhmann))
	g.Expect(parsed["source"]).To(Equal(fields.Source))
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
	parsed := parseFrontmatter(t, got)
	g.Expect(parsed["type"]).To(Equal("fact"))
	g.Expect(parsed["subject"]).To(Equal("subagent dispatch"))
	g.Expect(parsed["luhmann"]).To(Equal("11"))
	g.Expect(parsed["created"]).To(Equal("2026-05-09"))
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
	parsed := parseFrontmatter(t, got)
	g.Expect(parsed).To(Equal(map[string]string{
		"type":      "feedback",
		"situation": "writing concurrent Go code with context",
		"behavior":  "ignoring context cancellation",
		"impact":    "leaks goroutines on shutdown",
		"action":    "always check ctx.Done() in select loops",
		"luhmann":   "9z",
		"created":   "2026-05-09",
		"source":    "session log foo, 2026-05-09 12:00 UTC",
	}))
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
	parsed := parseFrontmatter(t, got)
	g.Expect(parsed["type"]).To(Equal("moc"))
	g.Expect(parsed["topic"]).To(Equal("llm rationalization patterns under pressure"))
}

// TestRenderMOCFrontmatter_SafelyEncodesTrickyValues mirrors the safety check
// for MOC frontmatter.
func TestRenderMOCFrontmatter_SafelyEncodesTrickyValues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	fields := cli.ExportMOCFields{
		Topic:   "topic:\nwith newline",
		Luhmann: "11",
		Source:  "- src",
	}
	got := cli.ExportRenderMOCFrontmatter(fields, when)
	parsed := parseFrontmatter(t, got)
	g.Expect(parsed["topic"]).To(Equal(fields.Topic))
	g.Expect(parsed["source"]).To(Equal(fields.Source))
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
	g.Expect(string(writtenContent)).
		To(ContainSubstring("Lesson learned: when writing concurrent Go code"))
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

// parseFrontmatter strips the "---" delimiters from a rendered frontmatter
// block and decodes the inner YAML mapping into key→string pairs. Tests use
// it to assert frontmatter values survive a YAML roundtrip regardless of the
// quoting style the encoder happens to choose.
func parseFrontmatter(t *testing.T, rendered string) map[string]string {
	t.Helper()

	g := NewWithT(t)

	const delim = "---\n"

	g.Expect(strings.HasPrefix(rendered, delim)).To(BeTrue(), "missing opening ---")

	body := strings.TrimPrefix(rendered, delim)
	end := strings.Index(body, "\n"+delim)
	g.Expect(end).To(BeNumerically(">=", 0), "missing closing ---")

	parsed := map[string]string{}
	g.Expect(yaml.Unmarshal([]byte(body[:end+1]), &parsed)).To(Succeed())

	return parsed
}
