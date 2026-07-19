package cli_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestLearnFactArgs_AcceptsProjectAndIssueFlags(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o750)).To(Succeed())

	args := cli.LearnFactArgs{
		CommonLearnArgs: cli.CommonLearnArgs{
			Slug:     "with-project",
			Vault:    vault,
			Position: "top",
			Source:   "test",
			Project:  "engram",
			Issue:    "636",
		},
		Situation: "running tests",
		Subject:   "engram",
		Predicate: "supports",
		Object:    "project metadata",
	}

	err := cli.ExportRunLearnFromFactArgs(context.Background(), args, newTestDeps(io.Discard, io.Discard), io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	matches, globErr := filepath.Glob(filepath.Join(vault, "*.md"))
	g.Expect(globErr).NotTo(HaveOccurred())
	g.Expect(matches).To(HaveLen(1))

	if len(matches) == 0 {
		return
	}

	body, readErr := os.ReadFile(matches[0])
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(body)).To(ContainSubstring("project: engram\n"))
	g.Expect(string(body)).To(ContainSubstring("issue: \"636\"\n"))
}

func TestOsLearnFS_Lock_BadVaultReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := cli.ExportNewOsLearnFS()
	_, err := fs.Lock("/nonexistent/parent/that/does/not/exist")
	g.Expect(err).To(HaveOccurred())
}

func TestRunLearnFromFactArgs_BootstrapsMissingVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Vault dir does NOT exist; runLearn must bootstrap it before writing.
	vault := filepath.Join(t.TempDir(), "fresh-vault")

	args := cli.LearnFactArgs{
		CommonLearnArgs: cli.CommonLearnArgs{
			Slug:     "bootstrap-fact",
			Vault:    vault,
			Position: "top",
			Source:   "test",
		},
		Situation: "first run",
		Subject:   "engram",
		Predicate: "bootstraps",
		Object:    "the vault",
	}

	err := cli.ExportRunLearnFromFactArgs(context.Background(), args, newTestDeps(io.Discard, io.Discard), io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Bootstrap created  and .obsidian/.
	for _, sub := range []string{".obsidian"} {
		info, statErr := os.Stat(filepath.Join(vault, sub))
		g.Expect(statErr).NotTo(HaveOccurred())

		if statErr != nil {
			return
		}

		g.Expect(info.IsDir()).To(BeTrue())
	}

	// And the actual fact note landed.
	entries, readErr := os.ReadDir(vault)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(entries).NotTo(BeEmpty())
}

// TestRunLearnFromFactArgs_RequiresSituation asserts a fact write rejects an
// empty or whitespace --situation (M5). Situation is rendered into the fact
// body formula and drives recall-mirror retrieval, so an absent situation
// must fail loudly rather than silently produce an unretrievable note.
func TestRunLearnFromFactArgs_RequiresSituation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		situation string
	}{
		{name: "empty", situation: ""},
		{name: "whitespace", situation: "   "},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := t.TempDir()
			g.Expect(os.MkdirAll(vault, 0o750)).To(Succeed())

			args := cli.LearnFactArgs{
				CommonLearnArgs: cli.CommonLearnArgs{
					Slug:     "fact-slug",
					Vault:    vault,
					Position: "top",
					Source:   "test",
				},
				Situation: tc.situation,
				Subject:   "subj",
				Predicate: "is",
				Object:    "obj",
			}

			err := cli.ExportRunLearnFromFactArgs(context.Background(), args, newTestDeps(io.Discard, io.Discard), io.Discard)
			g.Expect(err).To(MatchError(ContainSubstring("situation")))

			entries, readErr := os.ReadDir(vault)
			g.Expect(readErr).NotTo(HaveOccurred())

			for _, entry := range entries {
				// the luhmann lock lives at the vault root; only NOTE files count
				g.Expect(entry.Name()).NotTo(HaveSuffix(".md"), "no note may be written")
			}
		})
	}
}

// runLearnFrom*Args use newLearnDeps(d) and call runLearn. Driving these
// with a real vault dir exercises the full struct-conversion + delegation path
// (which previously had 0% coverage).

func TestRunLearnFromFactArgs_WritesFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o750)).To(Succeed())

	args := cli.LearnFactArgs{
		CommonLearnArgs: cli.CommonLearnArgs{
			Slug:     "fact-slug",
			Vault:    vault,
			Position: "top",
		},
		Situation: "running tests",
		Subject:   "subj",
		Predicate: "is",
		Object:    "obj",
	}

	err := cli.ExportRunLearnFromFactArgs(context.Background(), args, newTestDeps(io.Discard, io.Discard), io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// One file landed in .
	entries, readErr := os.ReadDir(vault)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(entries).NotTo(BeEmpty())
}

// TestRunLearnFromFeedbackArgs_RequiresSituation asserts a feedback write
// rejects an empty or whitespace --situation (M5), mirroring the fact case:
// situation feeds the feedback body formula and recall-mirror retrieval.
func TestRunLearnFromFeedbackArgs_RequiresSituation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		situation string
	}{
		{name: "empty", situation: ""},
		{name: "whitespace", situation: "   "},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := t.TempDir()
			g.Expect(os.MkdirAll(vault, 0o750)).To(Succeed())

			args := cli.LearnFeedbackArgs{
				CommonLearnArgs: cli.CommonLearnArgs{
					Slug:     "feedback-slug",
					Vault:    vault,
					Position: "top",
					Source:   "test",
				},
				Situation: tc.situation,
				Behavior:  "no tests",
				Impact:    "regressions",
				Action:    "write tests",
			}

			deps := newTestDeps(io.Discard, io.Discard)
			err := cli.ExportRunLearnFromFeedbackArgs(context.Background(), args, deps, io.Discard)
			g.Expect(err).To(MatchError(ContainSubstring("situation")))

			entries, readErr := os.ReadDir(vault)
			g.Expect(readErr).NotTo(HaveOccurred())

			for _, entry := range entries {
				// the luhmann lock lives at the vault root; only NOTE files count
				g.Expect(entry.Name()).NotTo(HaveSuffix(".md"), "no note may be written")
			}
		})
	}
}

func TestRunLearnFromFeedbackArgs_WritesFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o750)).To(Succeed())

	args := cli.LearnFeedbackArgs{
		CommonLearnArgs: cli.CommonLearnArgs{
			Slug:     "feedback-slug",
			Vault:    vault,
			Position: "top",
		},
		Situation: "writing code",
		Behavior:  "no tests",
		Impact:    "regressions",
		Action:    "write tests",
	}

	err := cli.ExportRunLearnFromFeedbackArgs(context.Background(), args, newTestDeps(io.Discard, io.Discard), io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	entries, readErr := os.ReadDir(vault)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(entries).NotTo(BeEmpty())
}
