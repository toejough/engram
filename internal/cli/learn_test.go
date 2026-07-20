package cli_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestApplyVocabAssignmentAfterLearn_TriggerFires drives applyVocabAssignmentAfterLearn
// with a vault at the growth threshold and asserts the trigger flag is persisted.
// ListMD is set but LoadTermVectors is nil — the trigger check must still run.
func TestApplyVocabAssignmentAfterLearn_TriggerFires(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// 150 non-vocab notes, last_refit at 100 notes 20 days ago → growth trigger
	names := make([]string, 150)
	for i := range names {
		names[i] = fmt.Sprintf("%d.2026-01-01.note.md", i+1)
	}

	centroidsDoc := cli.ExportVocabCentroidsDoc{
		SchemaVersion: 1,
		LastRefit:     &cli.ExportVocabLastRefitDoc{NoteCount: 100, Date: "2026-06-13"},
	}

	centroidsData, marshalErr := json.Marshal(centroidsDoc)

	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	var centroidsWritten []byte

	deps := cli.LearnDeps{
		Now:    func() time.Time { return time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC) },
		ListMD: func(string) ([]string, error) { return names, nil },
		ReadSidecar: func(path string) ([]byte, error) {
			if strings.HasSuffix(path, "vocab.centroids.json") {
				return centroidsData, nil
			}

			return nil, os.ErrNotExist
		},
		WriteNote: func(path string, data []byte) error {
			if strings.HasSuffix(path, "vocab.centroids.json") {
				centroidsWritten = data
			}

			return nil
		},
		LogWarning: nil,
	}

	cli.ExportApplyVocabAssignmentAfterLearn(deps, "/vault", "/vault/150.note.md", "---\ntype: fact\n---\n")

	g.Expect(centroidsWritten).NotTo(BeNil(), "trigger check must write centroids")

	var got cli.ExportVocabCentroidsDoc

	g.Expect(json.Unmarshal(centroidsWritten, &got)).NotTo(HaveOccurred())

	if err := json.Unmarshal(centroidsWritten, &got); err != nil {
		return
	}

	g.Expect(got.RefitPending).To(BeTrue())
}

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

func TestExtractLuhmannFromFilename_RejectsNonMd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	_, ok := cli.ExportExtractLuhmannFromFilename("1a3.2026-05-09.subagent-recovery.txt")
	g.Expect(ok).To(BeFalse())
}

// TestFactAmend_VocabVersionRoundTrip verifies that a fact note's
// vocab_version frontmatter key — the vocab-definition family note's version
// marker (#678 Task 4) — survives an amend that changes an unrelated field.
// factFrontmatterDoc must carry VocabVersion (added between Sources and Tags)
// so applyTypedAmend's unmarshal→render cycle (amend.go) never silently
// drops it.
func TestFactAmend_VocabVersionRoundTrip(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	notePath := filepath.Join(dir, "210.2026-07-10.vocab-definition.md")

	original := "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\n" +
		"luhmann: \"210\"\ncreated: 2026-07-10\nsource: src\nvocab_version: \"6.0\"\ntags:\n    - vocab\n" +
		"---\n\nWhen s, per b: c.\n"

	writeErr := os.WriteFile(notePath, []byte(original), 0o600)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	deps := cli.ExportNewAmendDeps(cli.ExportNewTestOsDeps())
	deps.Now = func() time.Time { return time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC) }

	args := cli.AmendArgs{Vault: dir, Target: "210", Object: "updated description"}

	var buf strings.Builder

	amendErr := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(amendErr).NotTo(HaveOccurred())

	if amendErr != nil {
		return
	}

	final, readErr := os.ReadFile(notePath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(final)).To(ContainSubstring(`vocab_version: "6.0"`),
		"vocab_version must survive amend's unmarshal-render cycle")
	g.Expect(string(final)).To(ContainSubstring("object: updated description"))
}

func TestLearnFact_ChunkSources_WrittenToFrontmatter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var written []byte

	args := cli.LearnArgs{
		Type: "fact", Slug: "test-slug", Vault: t.TempDir(), Position: "top",
		Source: "test", Situation: "testing chunk sources",
		Subject: "A", Predicate: "has", Object: "B",
		ChunkSources: []string{"/sessions/s.jsonl#turn-1", "/sessions/s.jsonl#turn-2"},
	}
	deps := cli.LearnDeps{
		Now:           func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
		Getenv:        func(string) string { return "" },
		StatDir:       func(string) error { return nil },
		InitVault:     func(string) error { return nil },
		ListIDs:       func(string) ([]string, error) { return nil, nil },
		ListBasenames: func(string) ([]string, error) { return nil, nil },
		Lock:          func(string) (func(), error) { return func() {}, nil },
		WriteNew:      func(_ string, data []byte) error { written = data; return nil },
	}

	var buf strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(written)).To(ContainSubstring("sources:"))
	g.Expect(string(written)).To(ContainSubstring("/sessions/s.jsonl#turn-1"))
	g.Expect(string(written)).To(ContainSubstring("/sessions/s.jsonl#turn-2"))
}

func TestLearnFact_EmptyChunkSources_NoSourcesKey(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var written []byte

	args := cli.LearnArgs{
		Type: "fact", Slug: "test-slug", Vault: t.TempDir(), Position: "top",
		Source: "test", Situation: "no chunk sources",
		Subject: "A", Predicate: "has", Object: "B",
	}
	deps := cli.LearnDeps{
		Now:           func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
		Getenv:        func(string) string { return "" },
		StatDir:       func(string) error { return nil },
		InitVault:     func(string) error { return nil },
		ListIDs:       func(string) ([]string, error) { return nil, nil },
		ListBasenames: func(string) ([]string, error) { return nil, nil },
		Lock:          func(string) (func(), error) { return func() {}, nil },
		WriteNew:      func(_ string, data []byte) error { written = data; return nil },
	}

	var buf strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(written)).NotTo(ContainSubstring("sources:"))
}

// TestLearnFact_EmptyTags_NoTagsKey: no --tag flags → no tags: key (omitempty).
func TestLearnFact_EmptyTags_NoTagsKey(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var written []byte

	args := cli.LearnArgs{
		Type: "fact", Slug: "untagged", Vault: t.TempDir(), Position: "top",
		Source: "test", Situation: "no tags", Subject: "A", Predicate: "has", Object: "B",
	}
	deps := cli.LearnDeps{
		Now:           func() time.Time { return time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC) },
		Getenv:        func(string) string { return "" },
		StatDir:       func(string) error { return nil },
		InitVault:     func(string) error { return nil },
		ListIDs:       func(string) ([]string, error) { return nil, nil },
		ListBasenames: func(string) ([]string, error) { return nil, nil },
		Lock:          func(string) (func(), error) { return func() {}, nil },
		WriteNew:      func(_ string, data []byte) error { written = data; return nil },
	}

	var buf strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(written)).NotTo(ContainSubstring("tags:"))
}

// TestLearnFact_InvalidTag_RejectedBeforeWrite: every malformed tag shape is
// rejected with the sentinel error, before any file write.
func TestLearnFact_InvalidTag_RejectedBeforeWrite(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	invalidTags := []string{
		"Work-Kind/Rename", // uppercase
		"a/b/c",            // two slashes
		"work kind/rename", // space
		"/rename",          // empty family
		"work-kind/",       // empty value
		"tier=cheap",       // wrong separator
		"",                 // empty
	}

	for _, tag := range invalidTags {
		var wrote bool

		args := cli.LearnArgs{
			Type: "fact", Slug: "bad-tag", Vault: t.TempDir(), Position: "top",
			Source: "test", Situation: "s", Subject: "a", Predicate: "b", Object: "c",
			Tags: []string{"work-kind/ok-sibling", tag},
		}
		deps := cli.LearnDeps{
			Now:           func() time.Time { return time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC) },
			Getenv:        func(string) string { return "" },
			StatDir:       func(string) error { return nil },
			InitVault:     func(string) error { return nil },
			ListIDs:       func(string) ([]string, error) { return nil, nil },
			ListBasenames: func(string) ([]string, error) { return nil, nil },
			Lock:          func(string) (func(), error) { return func() {}, nil },
			WriteNew:      func(string, []byte) error { wrote = true; return nil },
		}

		var buf strings.Builder

		err := cli.ExportRunLearn(t.Context(), args, deps, &buf)
		g.Expect(err).To(MatchError(ContainSubstring("tag must be")), "tag %q must be rejected", tag)
		g.Expect(wrote).To(BeFalse(), "tag %q must be rejected before any write", tag)
	}
}

// TestLearnFact_Tags_WrittenToFrontmatter locks the --tag surface (#674): tags
// on LearnArgs land in the frontmatter tags: list, preserving order.
func TestLearnFact_Tags_WrittenToFrontmatter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var written []byte

	args := cli.LearnArgs{
		Type: "fact", Slug: "route-dispatch-rename", Vault: t.TempDir(), Position: "top",
		Source: "test", Situation: "routing rename work",
		Subject: "rename dispatch at cheap (haiku)", Predicate: "resolved as", Object: "pass",
		Tags: []string{"work-kind/rename", "tier/cheap", "outcome/pass"},
	}
	deps := cli.LearnDeps{
		Now:           func() time.Time { return time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC) },
		Getenv:        func(string) string { return "" },
		StatDir:       func(string) error { return nil },
		InitVault:     func(string) error { return nil },
		ListIDs:       func(string) ([]string, error) { return nil, nil },
		ListBasenames: func(string) ([]string, error) { return nil, nil },
		Lock:          func(string) (func(), error) { return func() {}, nil },
		WriteNew:      func(_ string, data []byte) error { written = data; return nil },
	}

	var buf strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	note := string(written)
	g.Expect(note).To(ContainSubstring("tags:"))
	g.Expect(note).To(ContainSubstring("work-kind/rename"))
	g.Expect(note).To(ContainSubstring("tier/cheap"))
	g.Expect(note).To(ContainSubstring("outcome/pass"))
	g.Expect(strings.Index(note, "work-kind/rename")).
		To(BeNumerically("<", strings.Index(note, "tier/cheap")), "tag order must be preserved")
	g.Expect(strings.Index(note, "tier/cheap")).
		To(BeNumerically("<", strings.Index(note, "outcome/pass")), "tag order must be preserved")
}

// TestLearnFeedback_Tags_WrittenToFrontmatter: the feedback form carries tags too.
func TestLearnFeedback_Tags_WrittenToFrontmatter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var written []byte

	args := cli.LearnArgs{
		Type: "feedback", Slug: "tagged-feedback", Vault: t.TempDir(), Position: "top",
		Source: "test", Situation: "routing rename work",
		Behavior: "b", Impact: "i", Action: "a",
		Tags: []string{"work-kind/rename", "outcome/fail"},
	}
	deps := cli.LearnDeps{
		Now:           func() time.Time { return time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC) },
		Getenv:        func(string) string { return "" },
		StatDir:       func(string) error { return nil },
		InitVault:     func(string) error { return nil },
		ListIDs:       func(string) ([]string, error) { return nil, nil },
		ListBasenames: func(string) ([]string, error) { return nil, nil },
		Lock:          func(string) (func(), error) { return func() {}, nil },
		WriteNew:      func(_ string, data []byte) error { written = data; return nil },
	}

	var buf strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(written)).To(ContainSubstring("tags:"))
	g.Expect(string(written)).To(ContainSubstring("work-kind/rename"))
	g.Expect(string(written)).To(ContainSubstring("outcome/fail"))
}

func TestLearnPath_Permanent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	got := cli.ExportLearnPath("/vault", "1a3", "subagent-driven-recovery", when)
	g.Expect(got).To(Equal("/vault/1a3.2026-05-09.subagent-driven-recovery.md"))
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
	})
	g.Expect(got).To(Equal(
		"Information learned: when in reasoning about agent coordination, " +
			"subagent dispatch is fundamentally a verification problem dressed as coordination.\n" +
			"\n"))
}

func TestRenderBody_Feedback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	action := "set up a task list with self-contained briefs and dispatch; " +
		"if a small model cannot finish a subtask, shrink the task"
	got := cli.ExportRenderFeedbackBody(cli.ExportFeedbackFields{
		Situation: "orchestrating multi-step work as the main LLM under context pressure",
		Action:    action,
	})
	g.Expect(got).To(Equal(
		"Lesson learned: when orchestrating multi-step work as the main LLM under context pressure, " +
			action + ".\n" +
			"\n",
	))
}

// TestRenderFactBody_StripsLeadingWhenFromSituation is the fact-type variant of
// the double-"when" bug guard.
func TestRenderFactBody_StripsLeadingWhenFromSituation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportRenderFactBody(cli.ExportFactFields{
		Situation: "When reasoning about agent coordination",
		Subject:   "subagent dispatch",
		Predicate: "is fundamentally",
		Object:    "a verification problem",
	})
	g.Expect(got).
		To(HavePrefix("Information learned: when in reasoning about agent coordination, " +
			"subagent dispatch is fundamentally a verification problem."))
	g.Expect(got).NotTo(ContainSubstring("when in When"))
}

func TestRenderFactFrontmatter_EmitsProjectAndIssueBelowSource(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 26, 0, 0, 0, 0, time.UTC)
	fields := cli.ExportFactFields{
		Situation: "s", Subject: "subj", Predicate: "pred", Object: "obj",
		Luhmann: "1", Source: "src",
		Project: "engram", Issue: "636",
	}
	got := cli.ExportRenderFactFrontmatter(fields, when)
	g.Expect(got).To(ContainSubstring("source: src\n"))
	g.Expect(got).To(ContainSubstring("project: engram\n"))
	g.Expect(got).To(ContainSubstring("issue: \"636\"\n"))

	srcIdx := strings.Index(got, "source:")
	projIdx := strings.Index(got, "project:")
	issueIdx := strings.Index(got, "issue:")

	g.Expect(srcIdx).To(BeNumerically("<", projIdx))
	g.Expect(projIdx).To(BeNumerically("<", issueIdx))
}

func TestRenderFactFrontmatter_OmitsProjectAndIssueWhenEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 26, 0, 0, 0, 0, time.UTC)
	fields := cli.ExportFactFields{
		Situation: "s", Subject: "subj", Predicate: "pred", Object: "obj",
		Luhmann: "1", Source: "src",
	}
	got := cli.ExportRenderFactFrontmatter(fields, when)
	g.Expect(got).NotTo(ContainSubstring("project:"))
	g.Expect(got).NotTo(ContainSubstring("issue:"))
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

// TestRenderFactFrontmatter_TagsRoundtripFidelity is a property test: any tag
// list drawn from the valid grammar passes validateTags AND survives the
// render→parse YAML roundtrip identically (order and values). Mirrors
// TestRenderFeedbackFrontmatter_RoundtripFidelity.
func TestRenderFactFrontmatter_TagsRoundtripFidelity(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		tagGen := rapid.StringMatching(`[a-z0-9-]{1,12}(/[a-z0-9-]{1,12})?`)
		tags := rapid.SliceOfN(tagGen, 1, 4).Draw(rt, "tags")

		if err := cli.ExportValidateTags(tags); err != nil {
			rt.Fatalf("grammar-valid tags rejected: %v", err)
		}

		fields := cli.ExportFactFields{
			Situation: "s", Subject: "a", Predicate: "b", Object: "c",
			Luhmann: "1", Source: "src", Tags: tags,
		}
		when := time.Date(2026, time.July, 10, 0, 0, 0, 0, time.UTC)
		got := cli.ExportRenderFactFrontmatter(fields, when)

		const delim = "---\n"

		body := strings.TrimPrefix(got, delim)
		end := strings.Index(body, "\n"+delim)

		if end < 0 {
			rt.Fatalf("no closing delimiter in %q", got)
		}

		var doc struct {
			Tags []string `yaml:"tags"`
		}

		if err := yaml.Unmarshal([]byte(body[:end+1]), &doc); err != nil {
			rt.Fatalf("unmarshal %q: %v", body[:end+1], err)
		}

		if !slices.Equal(doc.Tags, tags) {
			rt.Fatalf("tags: got %v want %v\nfull:\n%s", doc.Tags, tags, got)
		}
	})
}

// TestRenderFeedbackBody_StripsLeadingWhenFromSituation guards against the
// double-"when" bug where the body template prepended "when " to a situation
// that already started with "When" — producing "Lesson learned: when When ...".
func TestRenderFeedbackBody_StripsLeadingWhenFromSituation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportRenderFeedbackBody(cli.ExportFeedbackFields{
		Situation: "When writing concurrent Go code",
		Action:    "check ctx.Done()",
	})
	g.Expect(got).
		To(HavePrefix("Lesson learned: when writing concurrent Go code, check ctx.Done()."))
	g.Expect(got).NotTo(ContainSubstring("when When"))
}

func TestRenderFeedbackFrontmatter_EmitsProjectAndIssueBelowSource(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 26, 0, 0, 0, 0, time.UTC)
	fields := cli.ExportFeedbackFields{
		Situation: "s", Behavior: "b", Impact: "i", Action: "a",
		Luhmann: "1", Source: "src",
		Project: "engram", Issue: "636",
	}
	got := cli.ExportRenderFeedbackFrontmatter(fields, when)
	g.Expect(got).To(ContainSubstring("project: engram\n"))
	g.Expect(got).To(ContainSubstring("issue: \"636\"\n"))

	srcIdx := strings.Index(got, "source:")
	projIdx := strings.Index(got, "project:")
	g.Expect(srcIdx).To(BeNumerically("<", projIdx))
}

// TestRenderFeedbackFrontmatter_LuhmannIsQuoted guards against yaml.v3's
// default behavior of emitting alphanumeric scalars unquoted. The vault
// convention is luhmann: "<id>" (double-quoted) so reads stay consistent
// across hand-written, migrated, and engram-learn-written notes; the existing
// pre-migration vault and the 218 migrated notes all quote this field.
func TestRenderFeedbackFrontmatter_LuhmannIsQuoted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	got := cli.ExportRenderFeedbackFrontmatter(cli.ExportFeedbackFields{
		Situation: "x", Behavior: "x", Impact: "x", Action: "x",
		Luhmann: "9aa", Source: "src",
	}, when)
	g.Expect(got).To(ContainSubstring(`luhmann: "9aa"`))
}

func TestRenderFeedbackFrontmatter_OmitsProjectAndIssueWhenEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 26, 0, 0, 0, 0, time.UTC)
	fields := cli.ExportFeedbackFields{
		Situation: "s", Behavior: "b", Impact: "i", Action: "a",
		Luhmann: "1", Source: "src",
	}
	got := cli.ExportRenderFeedbackFrontmatter(fields, when)
	g.Expect(got).NotTo(ContainSubstring("project:"))
	g.Expect(got).NotTo(ContainSubstring("issue:"))
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

func TestRunLearn_BootstrapsVaultWhenMissing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	initCalled := false
	deps := cli.LearnDeps{
		Now:       func() time.Time { return time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC) },
		Getenv:    func(string) string { return "" },
		StatDir:   func(string) error { return fs.ErrNotExist },
		InitVault: func(string) error { initCalled = true; return nil },
		ListIDs:   func(string) ([]string, error) { return nil, nil },
		Lock:      func(string) (func(), error) { return func() {}, nil },
		WriteNew:  func(string, []byte) error { return nil },
	}
	args := cli.LearnArgs{
		Type:      "feedback",
		Slug:      "x",
		Vault:     "/v",
		Position:  "top",
		Source:    "test",
		Situation: "bootstrapping the vault",
	}

	var stdout strings.Builder

	g.Expect(cli.ExportRunLearn(t.Context(), args, deps, &stdout)).To(Succeed())
	g.Expect(initCalled).To(BeTrue())
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

	g.Expect(writtenPath).To(Equal("/vault/1.2026-05-09.fact-slug.md"))
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
	g.Expect(writtenPath).To(Equal("/vault/3.2026-05-09.ctx-cancellation-rule.md"))
	g.Expect(string(writtenContent)).To(ContainSubstring("type: feedback"))
	g.Expect(string(writtenContent)).
		To(ContainSubstring("Lesson learned: when writing concurrent Go code"))
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
	args := cli.LearnArgs{Type: "fact", Slug: "x", Vault: "/v", Position: "top"}

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
	args := cli.LearnArgs{Type: "fact", Slug: "x", Vault: "/v", Position: "top"}

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
	args := cli.LearnArgs{Type: "fact", Slug: "x", Vault: "/v", Position: "top"}

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
	args := cli.LearnArgs{Type: "fact", Slug: "Bad Slug", Vault: "/v", Position: "top"}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).To(HaveOccurred())
}

func TestRunLearn_RejectsUnknownType(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	deps := cli.LearnDeps{
		Now:       time.Now,
		Getenv:    func(string) string { return "" },
		StatDir:   func(string) error { return nil },
		InitVault: func(string) error { return nil },
		ListIDs:   func(string) ([]string, error) { return nil, nil },
		Lock:      func(string) (func(), error) { return func() {}, nil },
		WriteNew:  func(string, []byte) error { return nil },
	}
	args := cli.LearnArgs{Type: "principle", Slug: "x", Vault: "/v", Position: "top"}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).To(HaveOccurred())
}

// TestTierFrontmatter_BadTierRejected verifies that an invalid --tier value
// returns errLearnBadTier.
func TestTierFrontmatter_BadTierRejected(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.LearnDeps{
		Now:      func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		ListIDs:  func(string) ([]string, error) { return nil, nil },
		Lock:     func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(string, []byte) error { return nil },
	}

	args := cli.LearnArgs{
		Type:      "fact",
		Slug:      "tier-bad",
		Vault:     "/v",
		Position:  "top",
		Source:    "src",
		Situation: "tier bad check",
		Subject:   "subj",
		Predicate: "pred",
		Object:    "obj",
		Tier:      "L9",
	}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).To(MatchError(cli.ErrLearnBadTierForTest))
}

// TestTierFrontmatter_FactDefaultsToL2 verifies that a rendered fact note
// carries tier: L2 derived from its type.
func TestTierFrontmatter_FactDefaultsToL2(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var writtenContent []byte

	deps := cli.LearnDeps{
		Now:      func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		ListIDs:  func(string) ([]string, error) { return nil, nil },
		Lock:     func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(_ string, data []byte) error { writtenContent = data; return nil },
	}

	args := cli.LearnArgs{
		Type:      "fact",
		Slug:      "tier-fact",
		Vault:     "/v",
		Position:  "top",
		Source:    "src",
		Situation: "tier check",
		Subject:   "subj",
		Predicate: "pred",
		Object:    "obj",
	}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(writtenContent)).To(ContainSubstring("tier: L2"))
}

// TestTierFrontmatter_FeedbackDefaultsToL2 verifies that a rendered feedback
// note carries tier: L2 derived from its type.
func TestTierFrontmatter_FeedbackDefaultsToL2(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var writtenContent []byte

	deps := cli.LearnDeps{
		Now:      func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		ListIDs:  func(string) ([]string, error) { return nil, nil },
		Lock:     func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(_ string, data []byte) error { writtenContent = data; return nil },
	}

	args := cli.LearnArgs{
		Type:      "feedback",
		Slug:      "tier-feedback",
		Vault:     "/v",
		Position:  "top",
		Source:    "src",
		Situation: "tier check",
		Behavior:  "beh",
		Impact:    "imp",
		Action:    "act",
	}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(writtenContent)).To(ContainSubstring("tier: L2"))
}

// TestTierFrontmatter_OverrideL3 verifies that --tier L3 on a fact note
// overrides the default L2 tier.
func TestTierFrontmatter_OverrideL3(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var writtenContent []byte

	deps := cli.LearnDeps{
		Now:      func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		ListIDs:  func(string) ([]string, error) { return nil, nil },
		Lock:     func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(_ string, data []byte) error { writtenContent = data; return nil },
	}

	args := cli.LearnArgs{
		Type:      "fact",
		Slug:      "tier-override",
		Vault:     "/v",
		Position:  "top",
		Source:    "src",
		Situation: "tier override check",
		Subject:   "subj",
		Predicate: "pred",
		Object:    "obj",
		Tier:      "L3",
	}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(writtenContent)).To(ContainSubstring("tier: L3"))
}

func TestValidateIssueID_AcceptsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportValidateIssueID("")).To(Succeed())
}

func TestValidateIssueID_AcceptsNonWhitespace(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportValidateIssueID("636")).To(Succeed())
	g.Expect(cli.ExportValidateIssueID("#636")).To(Succeed())
	g.Expect(cli.ExportValidateIssueID("PROJ-1234")).To(Succeed())
	g.Expect(cli.ExportValidateIssueID("gh-636")).To(Succeed())
}

func TestValidateIssueID_RejectsWhitespace(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportValidateIssueID("636 ")).To(HaveOccurred())
	g.Expect(cli.ExportValidateIssueID("two words")).To(HaveOccurred())
	g.Expect(cli.ExportValidateIssueID("with\ttab")).To(HaveOccurred())
}

func TestValidateProjectSlug_AcceptsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportValidateProjectSlug("")).To(Succeed())
}

func TestValidateProjectSlug_AcceptsKebabCase(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportValidateProjectSlug("engram")).To(Succeed())
	g.Expect(cli.ExportValidateProjectSlug("opencode-plugin")).To(Succeed())
	g.Expect(cli.ExportValidateProjectSlug("proj-123")).To(Succeed())
}

func TestValidateProjectSlug_RejectsBadShape(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportValidateProjectSlug("Engram")).To(HaveOccurred())
	g.Expect(cli.ExportValidateProjectSlug("with spaces")).To(HaveOccurred())
	g.Expect(cli.ExportValidateProjectSlug("punct!")).To(HaveOccurred())
}

// TestValidateTags_* exercise the validator directly, split per behavior
// (repo convention — cf. TestValidateProjectSlug_AcceptsEmpty /
// _AcceptsKebabCase / _RejectsBadShape, learn_test.go:903-917).
func TestValidateTags_AcceptsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportValidateTags(nil)).To(Succeed())
}

func TestValidateTags_AcceptsValidShapes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportValidateTags([]string{"work-kind"})).To(Succeed())
	g.Expect(cli.ExportValidateTags([]string{"work-kind/rename", "tier/cheap", "outcome/pass"})).To(Succeed())
	g.Expect(cli.ExportValidateTags([]string{"a1-b2/c3-d4"})).To(Succeed())
}

func TestValidateTags_RejectsBadShape(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportValidateTags([]string{"outcome/pass", "BAD"})).To(HaveOccurred())
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
