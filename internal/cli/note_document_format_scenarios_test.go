package cli_test

// Scenario-ID tests of the Note Document Format technical component (docs/specs/technical/note-document-format.md),
// 792b6c1 Note filenames. The pair's filenames are formed by the question-and-answer writer, so the scenario drives
// cli.RunLearnQA over recording dependencies: no file system, lock, clock or embedding model is touched.

import (
	"context"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// Test792b6c1a2_RunLearnQA_NamesAPairByDateAndSlug verifies 792b6c1a2 (note-document-format.md:59): in vault /vault on
// creation date 2026-07-03 with slug my-slug, the pair's question and answer paths are
// /vault/qa.2026-07-03.my-slug.q.md and /vault/qa.2026-07-03.my-slug.a.md.
func Test792b6c1a2_RunLearnQA_NamesAPairByDateAndSlug(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		vault        = "/vault"
		slug         = "my-slug"
		wantQuestion = "/vault/qa.2026-07-03.my-slug.q.md"
		wantAnswer   = "/vault/qa.2026-07-03.my-slug.a.md"
	)

	// Given vault /vault, creation date 2026-07-03 and slug my-slug
	creation := time.Date(2026, time.July, 3, 0, 0, 0, 0, time.UTC)

	var (
		written []string
		stdout  strings.Builder
	)

	deps := cli.LearnQADeps{
		Now:        func() time.Time { return creation },
		Getenv:     func(string) string { return "" },
		ListMD:     func(string) ([]string, error) { return nil, nil },
		Lock:       func(string) (func(), error) { return func() {}, nil },
		WriteNew:   func(path string, _ []byte) error { written = append(written, path); return nil },
		RemoveFile: func(string) error { return nil },
		ReadFile:   func(string) ([]byte, error) { return nil, nil },
	}

	// When the pair's paths are formed
	err := cli.RunLearnQA(context.Background(), cli.LearnQAArgs{
		Vault:    vault,
		Slug:     slug,
		Question: "What is my-slug?",
		Answer:   "my-slug is a test slug.",
		Source:   "scenario",
	}, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Then they are /vault/qa.2026-07-03.my-slug.q.md and /vault/qa.2026-07-03.my-slug.a.md
	g.Expect(written).To(Equal([]string{wantQuestion, wantAnswer}))
	g.Expect(stdout.String()).To(Equal(wantQuestion + "\n" + wantAnswer + "\n"))
}
