package cli_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestCollectVaultStats_QAQuestionExcluded_QAAnswerCounted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// qa.*.q.md file should NOT appear in totalNotes; qa.*.a.md should.
	names := []string{
		"qa.2026-07-03.slug.q.md", // Q note — excluded
		"qa.2026-07-03.slug.a.md", // A note — counted
		"vocab.agentic-recall-triggers.md",
		"vocab.index.md",
	}

	qaAContent := "---\ntype: qa-answer\ndate: \"2026-07-03\"\nvocab: [agentic-recall-triggers]\n---\n\nAnswer body.\n"
	// Gate A F1: the Q note MUST return real qa-question content — a read error would
	// exclude it from totalNotes even WITHOUT the new skip (extractNoteVocabTags returns
	// nil,false on read errors), making the RED phase a false pass. With real content,
	// the unmodified code counts it (totalNotes=2 → RED genuinely fails) and only the
	// added isQAQuestionFilename skip turns it GREEN (totalNotes=1).
	qaQContent := "---\ntype: qa-question\ndate: \"2026-07-03\"\nanswered_by: qa.2026-07-03.slug.a\n---\n\n" +
		"Question body.\n"

	deps := cli.VocabStatsDeps{
		ListMD: func(string) ([]string, error) { return names, nil },
		ReadFile: func(path string) ([]byte, error) {
			switch {
			case strings.HasSuffix(path, "slug.a.md"):
				return []byte(qaAContent), nil
			case strings.HasSuffix(path, "slug.q.md"):
				return []byte(qaQContent), nil
			}

			return nil, os.ErrNotExist
		},
	}

	_, _, totalNotes, _ := cli.ExportCollectVaultStats(names, deps, "/vault")
	g.Expect(totalNotes).To(Equal(1), "only A-note should count; Q-note excluded")
}

func TestCountQAPairs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	names := []string{
		"qa.2026-07-03.first.q.md",
		"qa.2026-07-03.first.a.md",
		"qa.2026-07-03.second.q.md", // no matching .a.md — orphan Q
		"100.2026-07-01.some-fact.md",
		"vocab.agentic-recall-triggers.md",
	}
	g.Expect(cli.ExportCountQAPairs(names)).To(Equal(1))
}

func TestIsQAQuestionFilename(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"question_file", "qa.2026-07-03.my-slug.q.md", true},
		{"answer_file", "qa.2026-07-03.my-slug.a.md", false},
		{"vocab_file", "vocab.agentic-recall-triggers.md", false},
		{"regular_fact", "100.2026-07-01.some-fact.md", false},
		{"bare_qa_prefix", "qa.md", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)
			g.Expect(cli.ExportIsQAQuestionFilename(tc.in)).To(Equal(tc.want))
		})
	}
}

func TestIsQueryExcludedKind(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		content string
		want    bool
	}{
		{"vocab_excluded", "---\ntype: vocab\n---\n", true},
		{"vocab_index_excluded", "---\ntype: vocab-index\n---\n", true},
		{"qa_question_excluded", "---\ntype: qa-question\n---\n", true},
		{"qa_answer_competes", "---\ntype: qa-answer\n---\n", false},
		{"fact_competes", "---\ntype: fact\n---\n", false},
		{"feedback_competes", "---\ntype: feedback\n---\n", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)
			g.Expect(cli.ExportIsQueryExcludedKind(tc.content)).To(Equal(tc.want))
		})
	}
}

func TestRenderQAAnswerNote_ContainsMachineLines(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	when := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
	contributors := []string{"100.2026-06-01.note", "101.2026-06-02.other-note"}
	got := cli.ExportRenderQAAnswerNote("The answer.", "my-slug", "session 2026-07-03", "medium", contributors, when)

	g.Expect(got).To(ContainSubstring("type: qa-answer"))
	g.Expect(got).To(ContainSubstring("answers: qa.2026-07-03.my-slug.q"))
	g.Expect(got).To(ContainSubstring("certainty: medium"))
	g.Expect(got).To(ContainSubstring("The answer."))
	g.Expect(got).To(ContainSubstring("Answers: [[qa.2026-07-03.my-slug.q]]"))
	g.Expect(got).To(ContainSubstring("Contributors: [[100.2026-06-01.note]], [[101.2026-06-02.other-note]]"))
}

func TestRenderQAAnswerNote_NoContributors_NoContributorsLine(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	when := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
	got := cli.ExportRenderQAAnswerNote("Answer.", "slug", "source", "high", nil, when)
	g.Expect(got).NotTo(ContainSubstring("Contributors:"))
}

func TestRenderQAQuestionNote_ContainsExpectedParts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	when := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
	got := cli.ExportRenderQAQuestionNote("What is the question?", "my-slug", "session 2026-07-03", when)

	g.Expect(got).To(ContainSubstring("type: qa-question"))
	g.Expect(got).To(ContainSubstring("answered_by: qa.2026-07-03.my-slug.a"))
	g.Expect(got).To(ContainSubstring("What is the question?"))
	g.Expect(got).To(ContainSubstring("Answered by: [[qa.2026-07-03.my-slug.a]]"))
	// Answered by: line must be in the BODY, not frontmatter
	body := strings.SplitN(got, "---\n\n", 2)
	g.Expect(body).To(HaveLen(2))

	if len(body) < 2 {
		return
	}

	g.Expect(body[1]).To(ContainSubstring("Answered by:"))
}

func TestRunLearnQA_AWriteAndRemoveFailure_OrphanWarning(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	writeCount := 0
	deps := cli.LearnQADeps{
		Now:       func() time.Time { return time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC) },
		Getenv:    func(string) string { return "" },
		StatDir:   func(string) error { return nil },
		InitVault: func(string) error { return nil },
		ListMD:    func(string) ([]string, error) { return nil, nil },
		Lock:      func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(_ string, _ []byte) error {
			writeCount++
			if writeCount == 2 {
				return errors.New("disk full")
			}

			return nil
		},
		RemoveFile: func(string) error { return errors.New("remove failed") },
		ReadFile:   func(string) ([]byte, error) { return nil, nil },
	}

	var buf strings.Builder

	err := cli.RunLearnQA(context.Background(), cli.LearnQAArgs{
		Slug: "slug", Question: "Q?", Answer: "body-a", Source: "src",
	}, deps, &buf)
	g.Expect(err).To(MatchError(ContainSubstring("orphan")))
}

func TestRunLearnQA_AWriteFailure_RemovesQAndErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var removed []string

	writeCount := 0
	deps := cli.LearnQADeps{
		Now:       func() time.Time { return time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC) },
		Getenv:    func(string) string { return "" },
		StatDir:   func(string) error { return nil },
		InitVault: func(string) error { return nil },
		ListMD:    func(string) ([]string, error) { return nil, nil },
		Lock:      func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(_ string, _ []byte) error {
			writeCount++
			if writeCount == 2 {
				return errors.New("disk full")
			}

			return nil
		},
		RemoveFile: func(path string) error { removed = append(removed, path); return nil },
		ReadFile:   func(string) ([]byte, error) { return nil, nil },
	}

	var buf strings.Builder

	err := cli.RunLearnQA(context.Background(), cli.LearnQAArgs{
		Slug: "slug", Question: "Q?", Answer: "body-a", Source: "src",
	}, deps, &buf)
	g.Expect(err).To(HaveOccurred())
	g.Expect(removed).To(HaveLen(1), "Q note must be removed on A-write failure")

	if len(removed) < 1 {
		return
	}

	g.Expect(removed[0]).To(ContainSubstring(".q.md"))
}

func TestRunLearnQA_InitVaultFailure_Error(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	initErr := errors.New("mkdir: read-only filesystem")
	deps := cli.LearnQADeps{
		Now:       time.Now,
		Getenv:    func(string) string { return "" },
		StatDir:   func(string) error { return os.ErrNotExist },
		InitVault: func(string) error { return initErr },
	}

	var buf strings.Builder

	err := cli.RunLearnQA(context.Background(), cli.LearnQAArgs{
		Slug: "slug", Question: "Q?", Answer: "body-a", Source: "src",
	}, deps, &buf)
	g.Expect(err).To(MatchError(initErr))
}

// Coverage tests for RunLearnQA and writeQANotesUnderLock branches.

func TestRunLearnQA_InvalidArgs_Error(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Empty question fails validateLearnQAArgs — covers the validateErr != nil return.
	deps := cli.LearnQADeps{
		Now:    time.Now,
		Getenv: func(string) string { return "" },
	}

	var buf strings.Builder

	err := cli.RunLearnQA(context.Background(), cli.LearnQAArgs{
		Slug:   "slug",
		Answer: "body",
		Source: "src",
	}, deps, &buf)
	g.Expect(err).To(MatchError(cli.ErrQAQuestionRequired))
}

func TestRunLearnQA_LockFailure_Error(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lockErr := errors.New("flock: permission denied")
	deps := cli.LearnQADeps{
		Now:        func() time.Time { return time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC) },
		Getenv:     func(string) string { return "" },
		StatDir:    func(string) error { return nil },
		InitVault:  func(string) error { return nil },
		ListMD:     func(string) ([]string, error) { return nil, nil },
		Lock:       func(string) (func(), error) { return nil, lockErr },
		WriteNew:   func(string, []byte) error { return nil },
		RemoveFile: func(string) error { return nil },
		ReadFile:   func(string) ([]byte, error) { return nil, nil },
	}

	var buf strings.Builder

	err := cli.RunLearnQA(context.Background(), cli.LearnQAArgs{
		Slug: "slug", Question: "Q?", Answer: "body-a", Source: "src",
	}, deps, &buf)
	g.Expect(err).To(MatchError(ContainSubstring("acquiring lock")))
}

func TestRunLearnQA_MissingVault_Initialized(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	initCalled := 0
	deps := cli.LearnQADeps{
		Now:        func() time.Time { return time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC) },
		Getenv:     func(string) string { return "" },
		StatDir:    func(string) error { return os.ErrNotExist },
		InitVault:  func(string) error { initCalled++; return nil },
		ListMD:     func(string) ([]string, error) { return nil, nil },
		Lock:       func(string) (func(), error) { return func() {}, nil },
		WriteNew:   func(string, []byte) error { return nil },
		RemoveFile: func(string) error { return nil },
		ReadFile:   func(string) ([]byte, error) { return nil, nil },
	}

	var buf strings.Builder

	err := cli.RunLearnQA(context.Background(), cli.LearnQAArgs{
		Slug: "slug", Question: "Q?", Answer: "body-a", Source: "src",
	}, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(initCalled).To(Equal(1), "missing vault must be initialized")
}

func TestRunLearnQA_StatDirFailure_Error(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	statErr := errors.New("stat: permission denied")
	deps := cli.LearnQADeps{
		Now:     time.Now,
		Getenv:  func(string) string { return "" },
		StatDir: func(string) error { return statErr },
	}

	var buf strings.Builder

	err := cli.RunLearnQA(context.Background(), cli.LearnQAArgs{
		Slug: "slug", Question: "Q?", Answer: "body-a", Source: "src",
	}, deps, &buf)
	g.Expect(err).To(MatchError(statErr))
}

func TestRunLearnQA_UnknownContributor_ErrorBeforeWrite(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	writeCallCount := 0
	deps := cli.LearnQADeps{
		Now:        time.Now,
		Getenv:     func(string) string { return "" },
		StatDir:    func(string) error { return nil },
		InitVault:  func(string) error { return nil },
		ListMD:     func(string) ([]string, error) { return []string{"100.note.md"}, nil },
		Lock:       func(string) (func(), error) { return func() {}, nil },
		WriteNew:   func(string, []byte) error { writeCallCount++; return nil },
		RemoveFile: func(string) error { return nil },
		ReadFile:   func(string) ([]byte, error) { return nil, nil },
	}

	var buf strings.Builder

	err := cli.RunLearnQA(context.Background(), cli.LearnQAArgs{
		Slug: "slug", Question: "Q?", Answer: "body-a", Source: "src",
		Contributors: []string{"999.ghost"},
	}, deps, &buf)
	g.Expect(err).To(MatchError(ContainSubstring("contributor not found")))
	g.Expect(writeCallCount).To(Equal(0), "no writes before validation error")
}

func TestRunLearnQA_WithAnswerFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var written []string

	fileContent := []byte("Answer read from file.")
	deps := cli.LearnQADeps{
		Now:        func() time.Time { return time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC) },
		Getenv:     func(string) string { return "" },
		StatDir:    func(string) error { return nil },
		InitVault:  func(string) error { return nil },
		ListMD:     func(string) ([]string, error) { return nil, nil },
		Lock:       func(string) (func(), error) { return func() {}, nil },
		WriteNew:   func(path string, _ []byte) error { written = append(written, path); return nil },
		RemoveFile: func(string) error { return nil },
		ReadFile:   func(string) ([]byte, error) { return fileContent, nil },
	}

	var buf strings.Builder

	err := cli.RunLearnQA(context.Background(), cli.LearnQAArgs{
		Slug:       "slug",
		Question:   "Q?",
		AnswerFile: "/tmp/answer.md",
		Source:     "src",
	}, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(written).To(HaveLen(2))
}

func TestRunLearnQA_WritesQAndAFiles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var written []string

	deps := cli.LearnQADeps{
		Now:        func() time.Time { return time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC) },
		Getenv:     func(string) string { return "" },
		StatDir:    func(string) error { return nil },
		InitVault:  func(string) error { return nil },
		ListMD:     func(string) ([]string, error) { return []string{"100.note.md"}, nil },
		Lock:       func(string) (func(), error) { return func() {}, nil },
		WriteNew:   func(path string, _ []byte) error { written = append(written, path); return nil },
		RemoveFile: func(string) error { return nil },
		ReadFile:   func(string) ([]byte, error) { return nil, nil },
	}

	var buf strings.Builder

	err := cli.RunLearnQA(context.Background(), cli.LearnQAArgs{
		Slug:     "test-qa",
		Question: "What is X?",
		Answer:   "X is Y.",
		Source:   "test",
	}, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(written).To(HaveLen(2))

	if len(written) < 2 {
		return
	}

	g.Expect(written[0]).To(ContainSubstring(".q.md"))
	g.Expect(written[1]).To(ContainSubstring(".a.md"))
}

func TestScanNonVocabNotes_QAQuestionFilenameSkipped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	names := []string{
		"qa.2026-07-03.slug.q.md", // should be skipped
		"qa.2026-07-03.slug.a.md", // should be visited
		"100.2026-07-01.note.md",  // should be visited
		"vocab.some-term.md",      // should be skipped
	}

	var visited []string

	cli.ExportScanNonVocabNotes("/vault", names,
		func(string) ([]byte, error) { return []byte("---\ntype: fact\n---\n"), nil },
		func(name string, _ []byte, _ error) { visited = append(visited, name) },
	)

	g.Expect(visited).To(ConsistOf("qa.2026-07-03.slug.a.md", "100.2026-07-01.note.md"))
}

func TestValidateContributors_KnownBasename_OK(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vaultNames := []string{"100.2026-01-01.note.md"}
	g.Expect(cli.ExportValidateContributors([]string{"100.2026-01-01.note"}, vaultNames)).To(Succeed())
}

func TestValidateContributors_UnknownBasename_Error(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vaultNames := []string{"100.2026-01-01.note.md"}
	err := cli.ExportValidateContributors([]string{"999.2026-01-01.ghost"}, vaultNames)
	g.Expect(err).To(MatchError(cli.ErrQAContributorNotFound))
}

func TestValidateLearnQAArgs_BothAnswerAndFile_Error(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	args := cli.LearnQAArgs{Slug: "slug", Question: "Q?", Answer: "body", AnswerFile: "/tmp/f.md", Source: "src"}
	g.Expect(cli.ExportValidateLearnQAArgs(args)).To(MatchError(cli.ErrQAAnswerSourceRequired))
}

func TestValidateLearnQAArgs_InvalidCertainty_Error(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	args := cli.LearnQAArgs{Slug: "slug", Question: "Q?", Answer: "body", Source: "src", Certainty: "bad"}
	g.Expect(cli.ExportValidateLearnQAArgs(args)).To(MatchError(cli.ErrQACertaintyInvalid))
}

func TestValidateLearnQAArgs_InvalidSlug_Error(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// A slug with spaces / uppercase is rejected by validateSlug.
	args := cli.LearnQAArgs{Slug: "INVALID SLUG", Question: "Q?", Answer: "body", Source: "src"}
	g.Expect(cli.ExportValidateLearnQAArgs(args)).To(HaveOccurred())
}

func TestValidateLearnQAArgs_MissingQuestion_Error(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	args := cli.LearnQAArgs{Slug: "slug", Answer: "body", Source: "src"}
	g.Expect(cli.ExportValidateLearnQAArgs(args)).To(MatchError(cli.ErrQAQuestionRequired))
}

func TestValidateLearnQAArgs_MissingSource_Error(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	args := cli.LearnQAArgs{Slug: "slug", Question: "Q?", Answer: "body"}
	g.Expect(cli.ExportValidateLearnQAArgs(args)).To(MatchError(cli.ErrQASourceRequired))
}

func TestValidateLearnQAArgs_Valid_NoError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	args := cli.LearnQAArgs{Slug: "slug", Question: "Q?", Answer: "body", Source: "src"}
	g.Expect(cli.ExportValidateLearnQAArgs(args)).To(Succeed())
}
