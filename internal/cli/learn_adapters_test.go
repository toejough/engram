package cli_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// Test792b12c2a1_RunLearnFromFactArgs_RequiresSituation asserts a fact write rejects an
// empty or whitespace --situation (M5). Situation is rendered into the fact
// body formula and drives recall-mirror retrieval, so an absent situation
// must fail loudly rather than silently produce an unretrievable note.
func Test792b12c2a1_RunLearnFromFactArgs_RequiresSituation(t *testing.T) {
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

// Test792b12c2a1_RunLearnFromFeedbackArgs_RequiresSituation asserts a feedback write
// rejects an empty or whitespace --situation (M5), mirroring the fact case:
// situation feeds the feedback body formula and recall-mirror retrieval.
func Test792b12c2a1_RunLearnFromFeedbackArgs_RequiresSituation(t *testing.T) {
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

func Test792b12c2a2_RunLearnFromRunbookArgs_RequiresSituation(t *testing.T) {
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

			args := cli.LearnRunbookArgs{
				CommonLearnArgs: cli.CommonLearnArgs{
					Slug:     "runbook-slug",
					Vault:    vault,
					Position: "top",
					Source:   "test",
				},
				Situation: tc.situation,
				DoneWhen:  "the tag is pushed",
				Body:      "1. Do the thing",
			}

			err := cli.ExportRunLearnFromRunbookArgs(context.Background(), args, newTestDeps(io.Discard, io.Discard), io.Discard)
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

func Test792b12c3a1_RunLearnFromFactArgs_WritesFile(t *testing.T) {
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

func Test792b12c3a1_RunLearnFromFeedbackArgs_WritesFile(t *testing.T) {
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

func Test792b12c3a1_RunLearnFromRunbookArgs_WritesFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o750)).To(Succeed())

	args := cli.LearnRunbookArgs{
		CommonLearnArgs: cli.CommonLearnArgs{
			Slug:     "runbook-slug",
			Vault:    vault,
			Position: "top",
		},
		Situation: "releasing a new Go module version",
		DoneWhen:  "the tag is pushed and the changelog is updated",
		Body:      "1. Run the tests\n2. Tag the release",
	}

	err := cli.ExportRunLearnFromRunbookArgs(context.Background(), args, newTestDeps(io.Discard, io.Discard), io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	entries, readErr := os.ReadDir(vault)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(entries).NotTo(BeEmpty())
}

func Test792b12c4a2_LearnFactArgs_AcceptsProjectAndIssueFlags(t *testing.T) {
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

// Test792b12c4a6_RunLearnFromRunbookArgs_NoRedFlag_NoFieldNoError proves the absent-flag
// case writes cleanly with no red_flags field and no error
// (note-writing 792a3d1c, "Neither flag means neither field").
func Test792b12c4a6_RunLearnFromRunbookArgs_NoRedFlag_NoFieldNoError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o750)).To(Succeed())

	args := cli.LearnRunbookArgs{
		CommonLearnArgs: cli.CommonLearnArgs{
			Slug:     "no-red-flags",
			Vault:    vault,
			Position: "top",
			Source:   "test",
		},
		Situation: "releasing a new Go module version",
		DoneWhen:  "the tag is pushed and the changelog is updated",
		Body:      "1. Run the tests\n2. Tag the release",
	}

	err := cli.ExportRunLearnFromRunbookArgs(context.Background(), args, newTestDeps(io.Discard, io.Discard), io.Discard)
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
	g.Expect(string(body)).NotTo(ContainSubstring("red_flags"))
}

// Test792b12c4a6_RunLearnFromRunbookArgs_RedFlagRepeatableFlagWritesField proves the
// repeatable --red-flag CLI flag (bound to LearnRunbookArgs.RedFlags) reaches
// the written note, in order (note-writing 792a3d).
func Test792b12c4a6_RunLearnFromRunbookArgs_RedFlagRepeatableFlagWritesField(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o750)).To(Succeed())

	args := cli.LearnRunbookArgs{
		CommonLearnArgs: cli.CommonLearnArgs{
			Slug:     "history-rewrite",
			Vault:    vault,
			Position: "top",
			Source:   "test",
		},
		Situation: "rewriting git history",
		DoneWhen:  "the backup branch still has every original commit",
		Body:      "1. Create a backup branch\n2. Rewrite history",
		RedFlags: []string{
			"filter-branch on all refs sweeps the backup branch",
			"force-push without --force-with-lease",
		},
	}

	err := cli.ExportRunLearnFromRunbookArgs(context.Background(), args, newTestDeps(io.Discard, io.Discard), io.Discard)
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
	g.Expect(string(body)).To(ContainSubstring(
		"red_flags:\n    - filter-branch on all refs sweeps the backup branch\n" +
			"    - force-push without --force-with-lease\n"))
}

// Test792b12c4a7_RunLearnFromRunbookArgs_TriggerRepeatableFlagWritesField proves the
// repeatable --trigger CLI flag (bound to LearnRunbookArgs.Triggers) reaches
// the written note, in order (792a3d).
func Test792b12c4a7_RunLearnFromRunbookArgs_TriggerRepeatableFlagWritesField(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()

	args := cli.LearnRunbookArgs{
		CommonLearnArgs: cli.CommonLearnArgs{
			Slug:     "drive-ask",
			Vault:    vault,
			Position: "top",
			Source:   "test",
		},
		Situation: "driving a task end to end",
		DoneWhen:  "the task is finished",
		Body:      "1. Do it",
		Triggers:  []string{"/please", "take this end-to-end"},
	}

	err := cli.ExportRunLearnFromRunbookArgs(context.Background(), args, newTestDeps(io.Discard, io.Discard), io.Discard)
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

	text := string(body)
	first := strings.Index(text, "/please")
	second := strings.Index(text, "take this end-to-end")
	g.Expect(first).To(BeNumerically(">", 0))
	g.Expect(second).To(BeNumerically(">", first))
	g.Expect(text).To(ContainSubstring("triggers:"))
}

// Test792b2c3a1_LearnFact_BootstrapsMissingVault: `engram learn fact` on a missing
// vault creates it in dispatch (design D2's single ensureVault path) and
// the note lands.
func Test792b2c3a1_LearnFact_BootstrapsMissingVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := filepath.Join(t.TempDir(), "fresh-vault")

	stdout, stderr := runWithRand(t, []string{
		"engram", "learn", "fact", "--vault", vault, "--slug", "bootstrap-fact", "--source", "test", "--position", "top",
		"--situation", "first run", "--subject", "engram", "--predicate", "bootstraps", "--object", "the vault",
	}, 1)

	g.Expect(filepath.Join(vault, ".obsidian")).To(BeADirectory())
	g.Expect(strings.TrimSpace(stdout)).To(HaveSuffix(".bootstrap-fact.md"), stderr)
	g.Expect(strings.TrimSpace(stdout)).To(BeAnExistingFile())
}
