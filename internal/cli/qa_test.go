package cli_test

import (
	"os"
	"strings"
	"testing"

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
