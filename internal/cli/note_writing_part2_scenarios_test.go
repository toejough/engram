package cli_test

// Scenario-ID tests of the Note Writing product capability, part 2 (docs/specs/product/note-writing.md): 792a3n Embed
// on write, 792a3o Repository provenance, 792a3p User provenance, 792a3q Question-and-answer pairs, 792a3r
// Question-and-answer validation, 792a3s Question-and-answer partial failure, 792a3t Question-and-answer retrieval
// and 792a3u Vocabulary tagging by kind.
//
// Every scenario drives the real command path (cli.Targets over the production compositions) over the in-memory vault
// of the controlled-vocabulary scenarios (cv1MemFS, a cli.EdgeFS) and its deterministic topic embedder
// (cv1TopicEmbedder). git is the one boundary beyond the file system: nw2Git models the answers of git for a machine
// with one repository, so the repo: and user: scenarios need no real git, home, network or embedding model. Helpers
// carry the prefix nw2.

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792a3n1a_LearnEveryKind_EachNoteHasASidecarAndCheckFindsNoneMissing verifies 792a3n1a (note-writing.md:349): in
// an empty vault, engram learn fact, feedback, runbook and qa each leave a .vec.json sidecar beside every note they
// write, and engram check then reports no note missing a sidecar.
func Test792a3n1a_LearnEveryKind_EachNoteHasASidecarAndCheckFindsNoneMissing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given an empty vault
	vault := newNW2Vault()

	// When the user runs engram learn fact, feedback, runbook or qa
	written := make([]string, 0, len(nw2Kinds)+2)

	for _, kind := range nw2Kinds {
		result := vault.learnNote(kind, "sidecar-"+kind, "keeping a sidecar beside the "+kind)
		g.Expect(result.stderr).To(BeEmpty())
		g.Expect(result.failed).To(BeFalse())

		written = append(written, nw2Lines(result.stdout)...)
	}

	qaResult := vault.learnQA("--slug", "sidecar-qa", "--question", "Why keep a sidecar?", "--answer", "Because.",
		"--source", nw2Source)
	g.Expect(qaResult.stderr).To(BeEmpty())
	g.Expect(qaResult.failed).To(BeFalse())

	written = append(written, nw2Lines(qaResult.stdout)...)

	// Then every note written has a .vec.json sidecar beside it
	g.Expect(written).To(HaveLen(len(nw2Kinds) + 2))

	for _, notePath := range written {
		g.Expect(vault.exists(filepath.Base(embed.SidecarPath(notePath)))).To(BeTrue(), "sidecar of %s", notePath)
	}

	// ... and engram check reports no note missing a sidecar
	check := vault.run("check")
	g.Expect(check.failed).To(BeFalse(), check.stderr)
	g.Expect(check.stdout).To(ContainSubstring("PASS  S1 sidecar-schema"))
	g.Expect(check.stdout).NotTo(ContainSubstring("missing a sidecar"))
}

// Test792a3n1b_LearnFact_AFailedEmbeddingKeepsTheNote verifies 792a3n1b (note-writing.md:354): when embedding the note
// fails, engram learn fact still writes the note, stderr carries a warning naming it, and the command succeeds.
func Test792a3n1b_LearnFact_AFailedEmbeddingKeepsTheNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given embedding fails for the note being learned
	vault := newNW2Vault()
	vault.embedder.failOn = nw2Unembeddable

	// When the user runs engram learn fact
	result := vault.learnNote("fact", "kept-anyway", "a note that is "+nw2Unembeddable)

	// Then the note is written, stderr carries a warning naming it, and the command succeeds
	name := vault.requireNote(g, `\.kept-anyway\.md$`)
	notePath := filepath.Join(cv1VaultPath, name)

	g.Expect(result.stdout).To(Equal(notePath + "\n"))
	g.Expect(result.stderr).To(ContainSubstring("learn: embed failed for " + notePath))
	g.Expect(result.failed).To(BeFalse(), "the command succeeds")
	g.Expect(vault.read(name)).To(ContainSubstring("type: fact"))
	g.Expect(vault.exists(filepath.Base(embed.SidecarPath(notePath)))).To(BeFalse(), "only the sidecar is missing")
}

// Test792a3o1a_LearnNote_TheOriginRemoteNamesTheRepo verifies 792a3o1a (note-writing.md:374): from a working directory
// inside a git repository whose origin is git@example.com:o/r.git, engram learn fact, feedback and runbook stamp
// repo: git@example.com:o/r.git.
func Test792a3o1a_LearnNote_TheOriginRemoteNamesTheRepo(t *testing.T) {
	t.Parallel()

	for _, kind := range nw2Kinds {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given a working directory inside a git repository whose origin is git@example.com:o/r.git
			vault := newNW2Vault()
			vault.git = nw2Git{top: nw2RepoTop, origin: nw2Origin}
			vault.workDir = nw2RepoTop + "/internal/deep"

			// When the user runs engram learn fact
			result := vault.learnNote(kind, "origin-note", "naming the repo")
			g.Expect(result.failed).To(BeFalse(), result.stderr)

			// Then the note's repo: is git@example.com:o/r.git
			g.Expect(vault.frontOf("origin-note")).To(HaveKeyWithValue("repo", nw2Origin))
		})
	}
}

// Test792a3o1b_LearnNote_WithoutAnOriginTheTopLevelDirectoryNamesTheRepo verifies 792a3o1b (note-writing.md:379): from
// a subdirectory of a git repository whose top-level directory is named proj and which has no origin remote, engram
// learn fact, feedback and runbook stamp repo: proj, not the name of the subdirectory.
func Test792a3o1b_LearnNote_WithoutAnOriginTheTopLevelDirectoryNamesTheRepo(t *testing.T) {
	t.Parallel()

	for _, kind := range nw2Kinds {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given a working directory inside a git repository named proj with no origin remote
			vault := newNW2Vault()
			vault.git = nw2Git{top: nw2RepoTop}
			vault.workDir = nw2RepoTop + "/internal/deep"

			// When the user runs engram learn fact
			result := vault.learnNote(kind, "toplevel-note", "naming the repo")
			g.Expect(result.failed).To(BeFalse(), result.stderr)

			// Then the note's repo: is proj
			g.Expect(vault.frontOf("toplevel-note")).To(HaveKeyWithValue("repo", "proj"))
		})
	}
}

// Test792a3o1c_LearnNote_OutsideAGitRepositoryThereIsNoRepo verifies 792a3o1c (note-writing.md:384): from a working
// directory outside any git repository, engram learn fact, feedback and runbook write the note without a repo: field
// and without failing.
func Test792a3o1c_LearnNote_OutsideAGitRepositoryThereIsNoRepo(t *testing.T) {
	t.Parallel()

	for _, kind := range nw2Kinds {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given a working directory outside any git repository
			vault := newNW2Vault()
			vault.git = nw2Git{top: nw2RepoTop, origin: nw2Origin}
			vault.workDir = nw2OutsideDir

			// When the user runs engram learn fact
			result := vault.learnNote(kind, "stray-note", "naming no repo")
			g.Expect(result.failed).To(BeFalse(), result.stderr)

			// Then the note has no repo: field
			front := vault.frontOf("stray-note")
			g.Expect(front).To(HaveKey("type"), "the note was written")
			g.Expect(front).NotTo(HaveKey("repo"))
		})
	}
}

// Test792a3p1a_LearnNote_TheGitEmailNamesTheUser verifies 792a3p1a (note-writing.md:400): with git config user.email
// a@example.com, engram learn fact, feedback and runbook stamp user: a@example.com although an OS username exists.
func Test792a3p1a_LearnNote_TheGitEmailNamesTheUser(t *testing.T) {
	t.Parallel()

	for _, kind := range nw2Kinds {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given git config user.email resolves to a@example.com
			vault := newNW2Vault()
			vault.git = nw2Git{email: "a@example.com"}

			// When the user runs engram learn fact
			result := vault.learnNote(kind, "emailed-note", "naming the user")
			g.Expect(result.failed).To(BeFalse(), result.stderr)

			// Then the note's user: is a@example.com
			g.Expect(vault.frontOf("emailed-note")).To(HaveKeyWithValue("user", "a@example.com"))
		})
	}
}

// Test792a3p1b_LearnNote_TheOSUsernameIsTheFallback verifies 792a3p1b (note-writing.md:405): with git config
// user.email unset and the OS username joe, engram learn fact, feedback and runbook stamp user: joe.
func Test792a3p1b_LearnNote_TheOSUsernameIsTheFallback(t *testing.T) {
	t.Parallel()

	for _, kind := range nw2Kinds {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given git config user.email resolves to nothing and the OS username is joe
			vault := newNW2Vault()
			vault.git = nw2Git{}
			vault.username = func() (string, error) { return "joe", nil }

			// When the user runs engram learn fact
			result := vault.learnNote(kind, "fallback-note", "naming the user")
			g.Expect(result.failed).To(BeFalse(), result.stderr)

			// Then the note's user: is joe
			g.Expect(vault.frontOf("fallback-note")).To(HaveKeyWithValue("user", "joe"))
		})
	}
}

// Test792a3p1c_LearnNote_NoDetectableUserOmitsTheUserWithOneWarning verifies 792a3p1c (note-writing.md:410): with
// neither git config user.email nor the OS username resolving, engram learn fact, feedback and runbook write the note
// with no user: field (never an empty one), still stamp vault:, and print one warning that user detection resolved
// empty.
func Test792a3p1c_LearnNote_NoDetectableUserOmitsTheUserWithOneWarning(t *testing.T) {
	t.Parallel()

	for _, kind := range nw2Kinds {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given neither git config user.email nor the OS username resolves
			vault := newNW2Vault()
			vault.git = nw2Git{}
			vault.username = func() (string, error) { return "", errNw2NoUser }

			// When the user runs engram learn fact
			result := vault.learnNote(kind, "anonymous-note", "naming no user")
			g.Expect(result.failed).To(BeFalse(), result.stderr)

			// Then the note has no user: field, vault: is stamped as usual, and stderr carries one warning
			front := vault.frontOf("anonymous-note")
			g.Expect(front).NotTo(HaveKey("user"))
			g.Expect(front).To(HaveKeyWithValue("vault", "personal"))
			g.Expect(strings.Count(result.stderr, "user detection resolved empty")).To(Equal(1), result.stderr)
		})
	}
}

// Test792a3q1a_LearnQA_APairIsCaptured verifies 792a3q1a (note-writing.md:436): over a vault holding note
// 1.2026-10-09.a-fact, engram learn qa --slug why-x --question "Why X?" --answer "Because." --source src
// --contributors 1.2026-10-09.a-fact prints the paths of the question note and then the answer note, the question
// note ends "Answered by: [[qa.<today>.why-x.a]]", and the answer note ends "Answers: [[qa.<today>.why-x.q]]" and
// "Contributors: [[1.2026-10-09.a-fact]]". The frontmatter fields the requirement lists are read back too.
func Test792a3q1a_LearnQA_APairIsCaptured(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault holding note 1.2026-10-09.a-fact
	vault := newNW2Vault()
	vault.clock = time.Date(2026, 10, 9, 9, 0, 0, 0, time.UTC)

	seeded := vault.learnNote("fact", "a-fact", "seeding the vault")
	g.Expect(seeded.stdout).To(Equal(filepath.Join(cv1VaultPath, "1.2026-10-09.a-fact.md") + "\n"))

	// When the user runs engram learn qa ... --contributors 1.2026-10-09.a-fact
	result := vault.learnQA("--slug", "why-x", "--question", "Why X?", "--answer", "Because.", "--source", "src",
		"--contributors", "1.2026-10-09.a-fact")
	g.Expect(result.failed).To(BeFalse(), result.stderr)

	// Then stdout prints the paths of the question note and then the answer note
	questionName := "qa.2026-10-09.why-x.q.md"
	answerName := "qa.2026-10-09.why-x.a.md"

	g.Expect(nw2Lines(result.stdout)).To(Equal([]string{
		filepath.Join(cv1VaultPath, questionName), filepath.Join(cv1VaultPath, answerName),
	}))

	// ... the question note ends with its Answered by line, and the answer note with its Answers and Contributors lines
	questionFront, questionBody := nw2ParseQA(g, vault.read(questionName))
	answerFront, answerBody := nw2ParseQA(g, vault.read(answerName))

	g.Expect(nw2LastLines(questionBody, 1)).To(Equal([]string{"Answered by: [[qa.2026-10-09.why-x.a]]"}))
	g.Expect(nw2LastLines(answerBody, 2)).To(Equal([]string{
		"Answers: [[qa.2026-10-09.why-x.q]]", "Contributors: [[1.2026-10-09.a-fact]]",
	}))

	// ... and the frontmatter carries the fields of the requirement
	g.Expect(questionFront).To(Equal(nw2QAFront{
		Type: "qa-question", Date: "2026-10-09", AnsweredBy: "qa.2026-10-09.why-x.a", Source: "src",
	}))
	g.Expect(answerFront).To(Equal(nw2QAFront{
		Type: "qa-answer", Date: "2026-10-09", Answers: "qa.2026-10-09.why-x.q", Certainty: "medium",
		Contributors: []string{"1.2026-10-09.a-fact"}, Source: "src",
	}))
	g.Expect(answerBody).To(HavePrefix("Because.\n"))
	g.Expect(questionBody).To(HavePrefix("Why X?\n"))
}

// Test792a3q1b_LearnQA_APairWithoutContributors verifies 792a3q1b (note-writing.md:441): engram learn qa without
// --contributors writes an answer note with no contributors: field and no Contributors: line.
func Test792a3q1b_LearnQA_APairWithoutContributors(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given an existing vault
	vault := newNW2Vault()
	vault.learnNote("fact", "a-fact", "seeding the vault")

	// When the user runs engram learn qa without --contributors
	result := vault.learnQA("--slug", "why-x", "--question", "Why X?", "--answer", "Because.", "--source", "src")
	g.Expect(result.failed).To(BeFalse(), result.stderr)

	// Then the answer note has no contributors: field and no Contributors: line
	answer := vault.read("qa." + cv1Today + ".why-x.a.md")
	g.Expect(answer).To(ContainSubstring("type: qa-answer"), "the answer note was written")
	g.Expect(answer).NotTo(MatchRegexp(`(?m)^contributors:`))
	g.Expect(answer).NotTo(ContainSubstring("Contributors:"))
}

// Test792a3q1c_LearnQA_TheAnswerCanComeFromAFile verifies 792a3q1c (note-writing.md:446): engram learn qa with
// --answer-file naming a file writes an answer note whose body is that file's text.
func Test792a3q1c_LearnQA_TheAnswerCanComeFromAFile(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a file holding the answer text
	const answerText = "Because the cache is stale.\n\nSecond paragraph of the answer.\n"

	vault := newNW2Vault()
	vault.fsys.put(nw2AnswerFile, []byte(answerText))

	// When the user runs engram learn qa with --answer-file naming it
	result := vault.learnQA("--slug", "why-x", "--question", "Why X?", "--answer-file", nw2AnswerFile,
		"--source", "src")
	g.Expect(result.failed).To(BeFalse(), result.stderr)

	// Then the answer note's body is that file's text
	_, body := nw2ParseQA(g, vault.read("qa."+cv1Today+".why-x.a.md"))
	g.Expect(body).To(HavePrefix(answerText))
	g.Expect(body).To(Equal(answerText + "\nAnswers: [[qa." + cv1Today + ".why-x.q]]\n"))
}

// Test792a3r1a_LearnQA_AnInvalidPairIsRefusedBeforeAnyWrite verifies the seven examples of 792a3r1a
// (note-writing.md:463): engram learn qa with no --question, with both or neither of --answer and --answer-file,
// with --certainty huge, with --contributors nope, with --slug X_y, or with no --source fails with an error saying
// the problem, and neither note is written.
func Test792a3r1a_LearnQA_AnInvalidPairIsRefusedBeforeAnyWrite(t *testing.T) {
	t.Parallel()

	const (
		answerOrFile = `exactly one of --answer or --answer-file is required`
		valid        = "--slug why-x --question Why? --answer Because. --source src"
	)

	cases := []struct {
		name    string
		args    []string
		problem string
	}{
		{"no --question", strings.Fields("--slug why-x --answer Because. --source src"), `--question is required`},
		{"both --answer and --answer-file", strings.Fields(valid + " --answer-file " + nw2AnswerFile), answerOrFile},
		{"neither --answer nor --answer-file", strings.Fields("--slug why-x --question Why? --source src"), answerOrFile},
		{"--certainty huge", strings.Fields(valid + " --certainty huge"), `certainty must be high, medium, or low`},
		{"--contributors nope", strings.Fields(valid + " --contributors nope"), `contributor not found in vault`},
		{"--slug X_y", strings.Fields("--slug X_y --question Why? --answer Because. --source src"),
			`slug must match \[a-z0-9-\]\+`},
		{"no --source", strings.Fields("--slug why-x --question Why? --answer Because."), `missing required flag.*--source`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given an existing vault
			vault := newNW2Vault()
			vault.learnNote("fact", "a-fact", "seeding the vault")
			vault.fsys.put(nw2AnswerFile, []byte("an answer in a file\n"))

			before := vault.fsys.snapshot()

			// When the user runs engram learn qa with <argument>
			result := vault.learnQA(tc.args...)

			// Then it fails with an error saying <problem>, and neither note is written
			g.Expect(result.failed).To(BeTrue(), result.stderr)
			g.Expect(result.stderr).To(MatchRegexp(tc.problem))
			g.Expect(vault.namesMatching(`^qa\.`)).To(BeEmpty())
			g.Expect(vault.fsys.snapshot()).To(Equal(before), "the vault is untouched")
		})
	}
}

// Test792a3r1b_LearnQA_CertaintyDefaultsToMedium verifies 792a3r1b (note-writing.md:478): engram learn qa without
// --certainty writes an answer note carrying certainty: medium.
func Test792a3r1b_LearnQA_CertaintyDefaultsToMedium(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given an existing vault
	vault := newNW2Vault()
	vault.learnNote("fact", "a-fact", "seeding the vault")

	// When the user runs engram learn qa without --certainty
	result := vault.learnQA("--slug", "why-x", "--question", "Why X?", "--answer", "Because.", "--source", "src")
	g.Expect(result.failed).To(BeFalse(), result.stderr)

	// Then the answer note carries certainty: medium
	front, _ := nw2ParseQA(g, vault.read("qa."+cv1Today+".why-x.a.md"))
	g.Expect(front.Certainty).To(Equal("medium"))
}

// Test792a3s1a_LearnQA_AFailedAnswerRemovesTheQuestion verifies 792a3s1a (note-writing.md:492): over a vault that
// already holds a file named like today's answer note for slug why-x, engram learn qa --slug why-x fails saying the
// answer note could not be written, and no question note for why-x remains.
func Test792a3s1a_LearnQA_AFailedAnswerRemovesTheQuestion(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault that already holds a file named like today's answer note for slug why-x
	answerName := "qa." + cv1Today + ".why-x.a.md"
	questionName := "qa." + cv1Today + ".why-x.q.md"

	vault := newNW2Vault()
	vault.fsys.put(filepath.Join(cv1VaultPath, answerName), []byte("an earlier answer\n"))

	// When the user runs engram learn qa --slug why-x
	result := vault.learnQA("--slug", "why-x", "--question", "Why X?", "--answer", "Because.", "--source", "src")

	// Then it fails saying the answer note could not be written, and no question note for why-x remains
	g.Expect(result.failed).To(BeTrue(), result.stderr)
	g.Expect(result.stderr).To(ContainSubstring("writing A note"))
	g.Expect(result.stdout).To(BeEmpty())
	g.Expect(vault.exists(questionName)).To(BeFalse(), "the question note was removed")
	g.Expect(vault.fsys.removeCount()).To(Equal(1), "the question note had been written first")
	g.Expect(vault.read(answerName)).To(Equal("an earlier answer\n"), "the earlier file is untouched")
}

// Test792a3t1a_Query_TheAnswerIsFoundAndTheQuestionIsNot verifies 792a3t1a (note-writing.md:506): over a vault holding
// a pair whose question and answer both match a phrase, engram query with that phrase returns the answer note and
// never the question note.
func Test792a3t1a_Query_TheAnswerIsFoundAndTheQuestionIsNot(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault holding a pair whose question and answer both match a phrase
	vault := newNW2Vault()
	learned := vault.learnQA("--slug", "hot-kitchen", "--question", "Why does a recipe need a hot kitchen?",
		"--answer", "Because the recipe cooks in the kitchen at high heat.", "--source", "src")
	g.Expect(learned.failed).To(BeFalse(), learned.stderr)

	questionName := "qa." + cv1Today + ".hot-kitchen.q.md"
	answerName := "qa." + cv1Today + ".hot-kitchen.a.md"

	g.Expect(vault.exists(embed.SidecarPath(questionName))).To(BeTrue(), "the question is embedded, so it could match")
	g.Expect(vault.exists(embed.SidecarPath(answerName))).To(BeTrue())

	// When the user runs engram query with that phrase
	result := vault.run("query", "--phrase", "recipe kitchen", "--chunks-dir", nw2ChunksDir)
	g.Expect(result.failed).To(BeFalse(), result.stderr)

	// Then the answer note appears in the results, and the question note never does
	returned := nw2QueriedNotes(g, result.stdout)
	g.Expect(returned).To(ContainElement(answerName))
	g.Expect(returned).NotTo(ContainElement(questionName))
}

// Test792a3u1a_LearnRunbook_ARunbookIsTagged verifies 792a3u1a (note-writing.md:523): over a vault whose vocabulary
// holds the term build-tooling, engram learn runbook with text about a linter and a compiler writes a runbook that
// carries the vocab/build-tooling tag.
func Test792a3u1a_LearnRunbook_ARunbookIsTagged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault with a vocabulary term close to a runbook's text
	vault := newNW2Vault()
	vault.bootstrap(cv1BuildTooling)

	// When the user runs engram learn runbook with that text
	result := vault.learnNote("runbook", "lint-runbook", "running the linter and the compiler before a commit")
	g.Expect(result.failed).To(BeFalse(), result.stderr)

	// Then the runbook carries the term's vocab tag
	g.Expect(vault.read(vault.requireNote(g, `\.lint-runbook\.md$`))).To(ContainSubstring("type: runbook"))
	g.Expect(vault.tagsOf("lint-runbook")).To(Equal([]string{"vocab/build-tooling"}))
}

// Test792a3u1b_LearnQA_TheAnswerIsTaggedAndTheQuestionIsNot verifies 792a3u1b (note-writing.md:528): over a vault
// whose vocabulary holds the term build-tooling, engram learn qa with a question and an answer about a linter writes
// an answer note carrying vocab/build-tooling and a question note with no tags: field.
func Test792a3u1b_LearnQA_TheAnswerIsTaggedAndTheQuestionIsNot(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault with a vocabulary term close to both a question and its answer
	vault := newNW2Vault()
	vault.bootstrap(cv1BuildTooling)

	// When the user runs engram learn qa with them
	result := vault.learnQA("--slug", "which-linter", "--question", "Which linter should run before a commit?",
		"--answer", "The project linter, because the compiler misses style issues.", "--source", "src")
	g.Expect(result.failed).To(BeFalse(), result.stderr)

	// Then the answer note carries the term's vocab tag, and the question note has no tags: field
	answer := vault.read("qa." + cv1Today + ".which-linter.a.md")
	question := vault.read("qa." + cv1Today + ".which-linter.q.md")

	g.Expect(question).To(ContainSubstring("type: qa-question"), "the question note was written")
	g.Expect(parseCV1Front(answer).Tags).To(Equal([]string{"vocab/build-tooling"}))
	g.Expect(question).NotTo(MatchRegexp(`(?m)^tags:`))
}

// unexported constants.
const (
	nw2AnswerFile   = "/answer.md"
	nw2ChunksDir    = "/chunks"
	nw2OSUser       = "nw2-user"
	nw2Origin       = "git@example.com:o/r.git"
	nw2OutsideDir   = "/home/nw2/elsewhere"
	nw2RepoTop      = "/home/nw2/src/proj"
	nw2Source       = "nw2"
	nw2Unembeddable = "unembeddable"
	nw2WorkDir      = "/home/nw2/work"
)

// unexported variables.
var (
	errNw2Git    = errors.New("nw2Git: exit status 128")
	errNw2NoUser = errors.New("nw2: no OS user")
	// nw2Kinds are the kinds of note that stamp provenance and embed on write.
	nw2Kinds = []string{"fact", "feedback", "runbook"}
)

// nw2Git answers the git commands of the identity detection the way git does on a machine with one repository: top
// is the top-level directory of that repository (empty when the machine has none), origin the URL of its origin
// remote (empty when it has none), email the git config user.email (empty when unset). A command run outside the
// repository, or asking for what is not configured, fails with git's exit status.
type nw2Git struct {
	top    string
	origin string
	email  string
}

func (machine nw2Git) Run(_ context.Context, dir, name string, args ...string) ([]byte, []byte, error) {
	if name != "git" {
		return nil, nil, errNw2Git
	}

	switch strings.Join(args, " ") {
	case "config user.email":
		if machine.email == "" {
			return nil, nil, errNw2Git
		}

		return []byte(machine.email + "\n"), nil, nil
	case "remote get-url origin":
		if !machine.inside(dir) || machine.origin == "" {
			return nil, []byte("error: No such remote 'origin'"), errNw2Git
		}

		return []byte(machine.origin + "\n"), nil, nil
	case "rev-parse --show-toplevel":
		if !machine.inside(dir) {
			return nil, []byte("fatal: not a git repository"), errNw2Git
		}

		return []byte(machine.top + "\n"), nil, nil
	default:
		return nil, nil, errNw2Git
	}
}

// inside reports whether dir is the repository's top-level directory or below it.
func (machine nw2Git) inside(dir string) bool {
	return machine.top != "" && (dir == machine.top || strings.HasPrefix(dir, machine.top+"/"))
}

// nw2QAFront is the frontmatter of a question or an answer note.
type nw2QAFront struct {
	Type         string   `yaml:"type"`
	Date         string   `yaml:"date"`
	AnsweredBy   string   `yaml:"answered_by"`
	Answers      string   `yaml:"answers"`
	Certainty    string   `yaml:"certainty"`
	Contributors []string `yaml:"contributors"`
	Source       string   `yaml:"source"`
}

// nw2Result is what one command printed, and whether it failed: it asked for an exit code, or targ refused its
// arguments.
type nw2Result struct {
	stdout string
	stderr string
	failed bool
}

// nw2Vault is one in-memory vault (the cv1Vault) and the environment engram runs in: the clock, the working
// directory, git and the OS user.
type nw2Vault struct {
	*cv1Vault

	clock    time.Time
	git      nw2Git
	username func() (string, error)
	workDir  string
}

// frontOf is the frontmatter of the note with the slug, as a map; it is empty when the note is not there.
func (v *nw2Vault) frontOf(slug string) map[string]any {
	front := map[string]any{}

	name := v.onlyNameMatching(`\.` + regexp.QuoteMeta(slug) + `\.md$`)

	block, _, found := embed.SplitFrontmatter([]byte(v.read(name)))
	if !found || yaml.Unmarshal(block, &front) != nil {
		return map[string]any{}
	}

	return front
}

// learnNote runs engram learn <kind> with the slug and the situation; the other fields mention no vocabulary topic.
// A runbook's steps (its body, the text vocabulary tagging reads) repeat the situation.
func (v *nw2Vault) learnNote(kind, slug, situation string) nw2Result {
	args := []string{
		"learn", kind, "--slug", slug, "--position", "top", "--source", nw2Source, "--situation", situation,
	}

	switch kind {
	case "fact":
		args = append(args, "--subject", "the team", "--predicate", "follows", "--object", "the routine")
	case "feedback":
		args = append(args, "--behavior", "the team skips it", "--impact", "it hurts", "--action", "do it")
	default:
		args = append(args, "--done-when", "the routine is done", "--body", "1. Follow the routine for "+situation+".")
	}

	return v.run(args...)
}

// learnQA runs engram learn qa with the arguments.
func (v *nw2Vault) learnQA(args ...string) nw2Result {
	return v.run(append([]string{"learn", "qa"}, args...)...)
}

// run runs `engram <args> --vault /vault` through the real targets over the in-memory vault.
func (v *nw2Vault) run(args ...string) nw2Result {
	var (
		stdout, stderr bytes.Buffer
		exits          []int
	)

	deps := cli.Deps{
		Stdout:       &stdout,
		Stderr:       &stderr,
		Exit:         func(code int) { exits = append(exits, code) },
		Getenv:       func(string) string { return "" },
		Now:          func() time.Time { return v.clock },
		Getwd:        func() (string, error) { return v.workDir, nil },
		UserHomeDir:  func() (string, error) { return "/home/nw2", nil },
		Username:     v.username,
		FS:           v.fsys,
		Lock:         fakeLocker{lock: func(string) (func() error, error) { return func() error { return nil }, nil }},
		Commander:    v.git,
		Embed:        v.embedder,
		RandRead:     seqRand(1),
		EvalSymlinks: identityPath,
	}

	command := append([]string{"engram"}, args...)
	command = append(command, "--vault", cv1VaultPath)

	executed, err := targ.Execute(command, cli.Targets(deps)...)
	if err != nil {
		stderr.WriteString(executed.Output)
		stderr.WriteString(err.Error())
		stderr.WriteString("\n")
	}

	return nw2Result{stdout: stdout.String(), stderr: stderr.String(), failed: len(exits) > 0 || err != nil}
}

// newNW2Vault is an empty vault on the clock of the cv1 scenarios, with a machine that has no git repository, no git
// user and the OS user nw2-user, and a working directory outside any repository.
func newNW2Vault() *nw2Vault {
	return &nw2Vault{
		cv1Vault: newCV1Vault(),
		clock:    cv1Clock,
		git:      nw2Git{},
		username: func() (string, error) { return nw2OSUser, nil },
		workDir:  nw2WorkDir,
	}
}

// nw2LastLines is the last count non-empty lines of text.
func nw2LastLines(text string, count int) []string {
	lines := nw2Lines(text)

	return lines[max(0, len(lines)-count):]
}

// nw2Lines are the non-empty lines of text.
func nw2Lines(text string) []string {
	lines := []string{}

	for line := range strings.SplitSeq(text, "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}

	return lines
}

// nw2ParseQA splits a question or answer note into its frontmatter and its body (the text after the frontmatter, from
// the first line that is not blank).
func nw2ParseQA(g *WithT, content string) (nw2QAFront, string) {
	var front nw2QAFront

	block, rest, found := embed.SplitFrontmatter([]byte(content))
	g.Expect(found).To(BeTrue(), "the note has frontmatter: %q", content)
	g.Expect(yaml.Unmarshal(block, &front)).To(Succeed())

	return front, strings.TrimLeft(string(rest), "\n")
}

// nw2QueriedNotes are the basenames of the notes in a query payload: its items and the members of its clusters.
func nw2QueriedNotes(g *WithT, stdout string) []string {
	var parsed queryParsed

	g.Expect(yaml.Unmarshal([]byte(stdout), &parsed)).To(Succeed())

	names := []string{}

	for _, item := range parsed.Items {
		names = append(names, filepath.Base(item.Path))
	}

	for _, cluster := range parsed.Clusters {
		for _, member := range cluster.Members {
			names = append(names, filepath.Base(member.Path))
		}

		for _, candidate := range cluster.CandidateL2s {
			names = append(names, filepath.Base(candidate.Path))
		}
	}

	return names
}
