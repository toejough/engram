package cli_test

// Scenario tests for the note-revision product capability, part 1 (861c2zzv): note references (792a4a),
// field replacement (792a4b), other note kinds (792a4c), supersession (792a4d), chunk sources (792a4e), argument
// validation (792a4f), re-embedding (792a4g) and the identity re-stamp (792a4h). Spec:
// docs/specs/product/note-revision.md.
//
// Every scenario runs `engram amend` through the real command wiring (cli.Targets, flag parsing, the vault
// lock, the scan, RunAmend and its production dependencies) over an in-memory vault (nr1FS, a complete
// cli.EdgeFS), with the environment a scenario names (the working directory's git remote, git config user.email,
// the OS username, ENGRAM_VAULT_NAME) pinned by nr1Env. The notes of a Given are written by `engram learn` in the
// same rig, in the environment the scenario's note was made in, so they carry the frontmatter, body and sidecar
// that the system writes; a Given that needs another shape (a note without identity, an ID of 107) edits that
// output by one named line. No temporary directory, no real home, no git, no network.

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

// Test792a4a1a_Amend_EveryReferenceFormFindsTheNote verifies 792a4a1a (Every reference form finds the note): the
// full basename, the basename with .md, a wikilink with and without a display part, and the bare Luhmann ID each
// name note 1.2026-10-09.a, which is amended and whose path is printed on stdout.
func Test792a4a1a_Amend_EveryReferenceFormFindsTheNote(t *testing.T) {
	t.Parallel()

	for name, ref := range map[string]string{
		"the full basename":            "1.2026-10-09.a",
		"the basename followed by .md": "1.2026-10-09.a.md",
		"a wikilink":                   "[[1.2026-10-09.a]]",
		"a wikilink with a display":    "[[1.2026-10-09.a|x]]",
		"the bare Luhmann ID":          "1",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			rig := nr1NewRig(t, nr1Alice)
			rig.learnFact()

			stdout, stderr := rig.amend("--target", ref, "--object", "new")

			g.Expect(stderr).To(BeEmpty())
			g.Expect(stdout).To(Equal(nr1Path(nr1BasenameA) + "\n"))
			g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("object", "new"))
		})
	}
}

// Test792a4b1a_Amend_AFactFieldIsReplaced verifies 792a4b1a (A fact field is replaced): amending the object of a
// fact whose situation is "testing", subject s, predicate p and object o leaves every other field as it was,
// rebuilds the body formula and prints the note's path.
func Test792a4b1a_Amend_AFactFieldIsReplaced(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learnFact()

	before := rig.fields(nr1BasenameA)

	stdout, stderr := rig.amend("--target", nr1BasenameA, "--object", "o2")

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal(nr1Path(nr1BasenameA) + "\n"))

	after := rig.fields(nr1BasenameA)
	g.Expect(after).To(HaveKeyWithValue("object", "o2"))

	delete(before, "object")
	delete(after, "object")
	g.Expect(after).To(Equal(before), "every other field is unchanged")
	g.Expect(rig.body(nr1BasenameA)).To(Equal("Information learned: when in testing, s p o2."))
}

// Test792a4b1b_Amend_AFeedbackFieldIsReplaced verifies 792a4b1b (A feedback field is replaced): amending the
// action of a feedback note whose situation is x and action a sets action: to b and rebuilds the body as
// "Lesson learned: when x, b.".
func Test792a4b1b_Amend_AFeedbackFieldIsReplaced(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learn("feedback", "--situation", "x", "--action", "a")

	_, stderr := rig.amend("--target", nr1BasenameA, "--action", "b")

	g.Expect(stderr).To(BeEmpty())
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("action", "b"))
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("situation", "x"))
	g.Expect(rig.body(nr1BasenameA)).To(Equal("Lesson learned: when x, b."))
}

// Test792a4b1c_Amend_ARunbooksStepsAndRedFlagsAreReplaced verifies 792a4b1c (A runbook's steps and red flags are
// replaced): amending a runbook whose body is "1. old" and whose red flags are A and B with --body "1. new" and
// --red-flag C leaves the body "1. new" and red_flags: listing only C.
func Test792a4b1c_Amend_ARunbooksStepsAndRedFlagsAreReplaced(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learn("runbook", "--situation", "x", "--done-when", "done", "--body", "1. old",
		"--red-flag", "A", "--red-flag", "B")

	g.Expect(rig.body(nr1BasenameA)).To(Equal("1. old"), "the Given: the runbook as learn wrote it")
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("red_flags", []any{"A", "B"}))

	_, stderr := rig.amend("--target", nr1BasenameA, "--body", "1. new", "--red-flag", "C")

	g.Expect(stderr).To(BeEmpty())
	g.Expect(rig.body(nr1BasenameA)).To(Equal("1. new"))
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("red_flags", []any{"C"}))
}

// Test792a4c1a_Amend_AQuestionAndAnswerNoteIsRefused verifies 792a4c1a (A question-and-answer note is refused):
// amending the answer note qa.2026-10-09.why.a with --situation fails saying the note type is unknown and leaves
// the note as it was.
func Test792a4c1a_Amend_AQuestionAndAnswerNoteIsRefused(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)

	_, learnErr := rig.run("learn", "qa", "--slug", "why", "--question", "Why?", "--answer", "Because.",
		"--source", "test")
	g.Expect(learnErr).To(BeEmpty())

	const answer = "qa.2026-10-09.why.a"

	before := rig.note(answer)
	g.Expect(before).NotTo(BeEmpty(), "the Given: the answer note exists")

	stdout, stderr := rig.amend("--target", answer, "--situation", "s")

	g.Expect(stderr).To(ContainSubstring("unknown note type"))
	g.Expect(stdout).To(BeEmpty())
	g.Expect(rig.note(answer)).To(Equal(before), "the note is unchanged")
}

// Test792a4d1a_Amend_ASupersessionIsWrittenInCanonicalForm verifies 792a4d1a (A supersession is written in
// canonical form): a supersession given as "<note>.md|refutes|claim" is written as the body line
// "Supersedes: [[<note>]] — refutes: claim" with no .md in the link, and its supersedes: entry keeps the note
// exactly as given, with the .md.
func Test792a4d1a_Amend_ASupersessionIsWrittenInCanonicalForm(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		basename = "107.2026-06-27.validate-lazy-retrieval-by-measuring-fetch-behavior"
		ref      = basename + ".md"
	)

	rig := nr1NewRig(t, nr1Alice)
	rig.learnFact()
	rig.relabel(nr1BasenameA, basename, "107")

	_, stderr := rig.amend("--target", basename, "--supersedes", ref+"|refutes|claim")

	g.Expect(stderr).To(BeEmpty())
	g.Expect(rig.body(basename)).To(ContainSubstring("Supersedes: [[" + basename + "]] — refutes: claim"))
	g.Expect(rig.body(basename)).NotTo(ContainSubstring(".md]]"))

	entries := nr1Supersedes(rig.fields(basename))
	g.Expect(entries).To(HaveLen(1))

	if len(entries) != 1 {
		return
	}

	g.Expect(entries[0]).To(HaveKeyWithValue("note", ref))
	g.Expect(entries[0]).To(HaveKeyWithValue("type", "refutes"))
	g.Expect(entries[0]).To(HaveKeyWithValue("claim", "claim"))
}

// Test792a4d1b_Amend_TheSupersessionListIsReplacedNotAppendedTo verifies 792a4d1b (The list is replaced, not
// appended to): a note whose supersedes: names A and B, with a Supersedes: line for each, is amended with
// --supersedes "C|narrows|c"; its supersedes: then names only C and its body has one Supersedes: line, for C.
func Test792a4d1b_Amend_TheSupersessionListIsReplacedNotAppendedTo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learn("fact", "--situation", "testing", "--subject", "s", "--predicate", "p",
		"--object", "o", "--supersedes", "A|updates|a", "--supersedes", "B|refutes|b")

	g.Expect(nr1Supersedes(rig.fields(nr1BasenameA))).To(HaveLen(2), "the Given: A and B")
	g.Expect(strings.Count(rig.body(nr1BasenameA), "Supersedes: ")).To(Equal(2), "the Given: a line for each")

	_, stderr := rig.amend("--target", nr1BasenameA, "--supersedes", "C|narrows|c")

	g.Expect(stderr).To(BeEmpty())

	entries := nr1Supersedes(rig.fields(nr1BasenameA))
	g.Expect(entries).To(HaveLen(1))

	if len(entries) != 1 {
		return
	}

	g.Expect(entries[0]).To(HaveKeyWithValue("note", "C"))

	body := rig.body(nr1BasenameA)
	g.Expect(strings.Count(body, "Supersedes: ")).To(Equal(1))
	g.Expect(body).To(ContainSubstring("Supersedes: [[C]] — narrows: c"))
	g.Expect(body).NotTo(ContainSubstring("[[A]]"))
	g.Expect(body).NotTo(ContainSubstring("[[B]]"))
}

// Test792a4e1a_Amend_ChunkSourcesAreMergedWithoutDuplicates verifies 792a4e1a (Chunk sources are merged without
// duplicates): a note whose sources: lists a#1, amended with --chunk-source a#1 and --chunk-source b#2 against a
// chunk index holding both, lists a#1 then b#2. The second case orders the IDs the other way round, so a merge
// that sorts the IDs, or puts the given ones first, would not pass: a note listing b#2, amended with a#1 and b#2,
// lists b#2 then a#1, the IDs already there first.
func Test792a4e1a_Amend_ChunkSourcesAreMergedWithoutDuplicates(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		had  string
		want []any
	}{
		"the scenario":                     {had: "a#1", want: []any{"a#1", "b#2"}},
		"the IDs already there come first": {had: "b#2", want: []any{"b#2", "a#1"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			rig := nr1NewRig(t, nr1Alice)
			rig.learn("fact", "--situation", "testing", "--subject", "s", "--predicate", "p",
				"--object", "o", "--chunk-source", tc.had)
			rig.plantChunkIndex("a#1", "b#2")

			g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("sources", []any{tc.had}), "the Given")

			_, stderr := rig.amend("--target", nr1BasenameA, "--chunk-source", "a#1", "--chunk-source", "b#2")

			g.Expect(stderr).To(BeEmpty())
			g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("sources", tc.want))
		})
	}
}

// Test792a4f1a_Amend_AnInvalidArgumentIsRefusedBeforeAnyWrite verifies 792a4f1a (An invalid amend argument is
// refused before any write): each of the five invalid arguments makes `engram amend --target 1` fail with an
// error that says the problem, and the note and its sidecar are unchanged.
func Test792a4f1a_Amend_AnInvalidArgumentIsRefusedBeforeAnyWrite(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		args    []string
		problem string
	}{
		"a supersession of an unknown type": {
			args:    []string{"--supersedes", "a|bogus|c"},
			problem: `type must be updates|narrows|refutes: got "bogus"`,
		},
		"a supersession without a claim part": {
			args:    []string{"--supersedes", "a|updates"},
			problem: `format must be "<basename>|<type>|<claim>"`,
		},
		"a supersession with an empty claim": {
			args:    []string{"--supersedes", "a|updates|"},
			problem: "claim must be non-empty",
		},
		"a trigger shorter than three characters": {
			args:    []string{"--trigger", "ab"},
			problem: "trigger must be at least 3 characters after trimming",
		},
		"a chunk source the index does not hold": {
			args:    []string{"--chunk-source", "nope#x"},
			problem: `unresolved chunk-source id: "nope#x"`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			rig := nr1NewRig(t, nr1Alice)
			rig.learnFact()

			noteBefore, sidecarBefore := rig.note(nr1BasenameA), rig.sidecar()

			stdout, stderr := rig.amend(append([]string{"--target", "1"}, tc.args...)...)

			g.Expect(stderr).To(ContainSubstring(tc.problem))
			g.Expect(stdout).To(BeEmpty())
			g.Expect(rig.note(nr1BasenameA)).To(Equal(noteBefore), "the note is unchanged")
			g.Expect(rig.sidecar()).To(Equal(sidecarBefore), "the sidecar is unchanged")
			g.Expect(rig.writesAfterSetup()).To(BeZero(), "nothing was written")
		})
	}
}

// Test792a4g1a_Amend_AContentAmendReEmbeds verifies 792a4g1a (A content amend re-embeds): amending a fact note
// whose sidecar is current with --object new rebuilds the sidecar from the amended text, so `engram embed status`
// reports the note ok and not stale.
func Test792a4g1a_Amend_AContentAmendReEmbeds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learnFact()

	sidecarBefore := rig.sidecar()

	status, _ := rig.run("embed", "status")
	g.Expect(status).To(ContainSubstring("with-embeddings: 1"), "the Given: a current sidecar")

	_, stderr := rig.amend("--target", nr1BasenameA, "--object", "new")

	g.Expect(stderr).To(BeEmpty())
	g.Expect(rig.sidecar()).NotTo(Equal(sidecarBefore), "the sidecar was rebuilt")

	status, _ = rig.run("embed", "status")
	g.Expect(status).To(ContainSubstring("with-embeddings: 1"))
	g.Expect(status).To(ContainSubstring("stale:           0"))
}

// Test792a4g1b_Amend_AProvenanceOnlyAmendDoesNotReEmbed verifies 792a4g1b (A provenance-only amend does not
// re-embed): amending a fact note whose sidecar is current with only --supersedes, or only --chunk-source, leaves
// the sidecar file as it was. The technical sibling 792b14c6a1 keeps its test
// (Test792b14c6a1_RunAmend_SupersedesOnlyReStamp_DoesNotTriggerReEmbed).
func Test792a4g1b_Amend_AProvenanceOnlyAmendDoesNotReEmbed(t *testing.T) {
	t.Parallel()

	for name, args := range map[string][]string{
		"only --supersedes":   {"--supersedes", "B|updates|b"},
		"only --chunk-source": {"--chunk-source", "a#1"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			rig := nr1NewRig(t, nr1Alice)
			rig.learnFact()
			rig.plantChunkIndex("a#1")

			sidecarBefore := rig.sidecar()
			noteBefore := rig.note(nr1BasenameA)
			writesBefore := rig.fs.sidecarWrites()

			_, stderr := rig.amend(append([]string{"--target", nr1BasenameA}, args...)...)

			g.Expect(stderr).To(BeEmpty())
			g.Expect(rig.note(nr1BasenameA)).NotTo(Equal(noteBefore), "the amend did write the note")
			g.Expect(rig.sidecar()).To(Equal(sidecarBefore))
			g.Expect(rig.fs.sidecarWrites()).To(Equal(writesBefore), "no write touched the sidecar")
		})
	}
}

// Test792a4g1c_Amend_AFailedReEmbedKeepsTheAmend verifies 792a4g1c (A failed re-embed keeps the amend): when
// embedding fails for the note being amended, `engram amend --target <note> --object new` still amends the note,
// prints the path, warns on stderr naming the note and succeeds.
func Test792a4g1c_Amend_AFailedReEmbedKeepsTheAmend(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learnFact()

	rig.embedFails = true

	stdout, stderr := rig.amend("--target", nr1BasenameA, "--object", "new")

	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("object", "new"), "the note is amended")
	g.Expect(stdout).To(Equal(nr1Path(nr1BasenameA)+"\n"), "the command succeeds and prints the path")
	g.Expect(stderr).To(HavePrefix("warning: "))
	g.Expect(stderr).To(ContainSubstring("embed failed for " + nr1BasenameA + ".md"))
	g.Expect(strings.Count(stderr, "\n")).To(Equal(1), "one warning and no error")
}

// Test792a4h1a_Amend_FromAnotherEnvironmentReStamps verifies 792a4h1a (Amend from another environment
// re-stamps): a fact note made in alice's environment, amended from bob's with a content flag, --supersedes or
// --chunk-source, carries repo:, user: and vault: of bob's environment.
func Test792a4h1a_Amend_FromAnotherEnvironmentReStamps(t *testing.T) {
	t.Parallel()

	for name, args := range map[string][]string{
		"a content flag": {"--object", "new"},
		"--supersedes":   {"--supersedes", "B|updates|b"},
		"--chunk-source": {"--chunk-source", "a#1"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			rig := nr1NewRig(t, nr1Alice)
			rig.learnFact()
			rig.plantChunkIndex("a#1")

			made := rig.fields(nr1BasenameA)
			g.Expect(made).To(HaveKeyWithValue("repo", nr1Alice.remote), "the Given: made in alice's environment")

			rig.env = nr1Bob

			_, stderr := rig.amend(append([]string{"--target", nr1BasenameA}, args...)...)

			g.Expect(stderr).To(BeEmpty())
			g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("repo", nr1Bob.remote))
			g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("user", nr1Bob.email))
			g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("vault", nr1Bob.vaultName))
		})
	}
}

// Test792a4h1b_Amend_BackfillsMissingIdentity verifies 792a4h1b (Amend backfills missing identity): a fact note
// with no repo:, user: or vault: gains the current environment's three values when amended with --object.
func Test792a4h1b_Amend_BackfillsMissingIdentity(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learnFact()
	rig.dropLines(nr1BasenameA, "repo:", "user:", "vault:")

	g.Expect(rig.fields(nr1BasenameA)).NotTo(HaveKey("repo"), "the Given")
	g.Expect(rig.fields(nr1BasenameA)).NotTo(HaveKey("user"))
	g.Expect(rig.fields(nr1BasenameA)).NotTo(HaveKey("vault"))

	rig.env = nr1Bob

	_, stderr := rig.amend("--target", nr1BasenameA, "--object", "new")

	g.Expect(stderr).To(BeEmpty())
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("repo", nr1Bob.remote))
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("user", nr1Bob.email))
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("vault", nr1Bob.vaultName))
}

// Test792a4h1c_Amend_TheProjectFieldDoesNotStandInForTheRepo verifies 792a4h1c (The project field does not stand
// in for the repo): a note whose project: is another-project, amended from a working directory whose repository
// is bob's, gets repo: of that repository, not the project: value, and keeps its project:.
func Test792a4h1c_Amend_TheProjectFieldDoesNotStandInForTheRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const project = "another-project"

	rig := nr1NewRig(t, nr1Alice)
	rig.learn("fact", "--situation", "testing", "--subject", "s", "--predicate", "p",
		"--object", "o", "--project", project)

	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("project", project), "the Given")

	rig.env = nr1Bob

	_, stderr := rig.amend("--target", nr1BasenameA, "--object", "new")

	g.Expect(stderr).To(BeEmpty())
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("repo", nr1Bob.remote))
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("project", project), "project: is kept by every amend")
}

// Test792a4h1d_Amend_EmptyUserDetectionKeepsThePriorUser verifies 792a4h1d (Empty user detection keeps the prior
// user): a note whose user: is alice's, amended where neither git config user.email nor the OS username resolves,
// keeps alice's user:, warns on stderr naming the note and succeeds. The technical sibling 792b11c4a2 keeps its
// test (Test792b11c4a2_RunAmend_EmptyUserDetectionKeepsPriorUser).
func Test792a4h1d_Amend_EmptyUserDetectionKeepsThePriorUser(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learnFact()

	rig.env = nr1Env{remote: nr1Bob.remote, vaultName: nr1Bob.vaultName}

	stdout, stderr := rig.amend("--target", nr1BasenameA, "--object", "new")

	g.Expect(stdout).To(Equal(nr1Path(nr1BasenameA)+"\n"), "the amend succeeds")
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("user", nr1Alice.email))
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("repo", nr1Bob.remote), "repo: and vault: are re-stamped")
	g.Expect(stderr).To(HavePrefix("warning: "))
	g.Expect(stderr).To(ContainSubstring("user detection resolved empty"))
	g.Expect(stderr).To(ContainSubstring(nr1BasenameA))
}

// Test792a4h1e_Amend_EmptyUserDetectionNeverWritesAnEmptyUser verifies 792a4h1e (Empty user detection never
// writes an empty user): a note with no user:, amended where neither git config user.email nor the OS username
// resolves, still has no user: field. The technical sibling 792b11c4a3 keeps its test
// (Test792b11c4a3_RunAmend_EmptyUserDetectionNoPriorUserOmitsUser).
func Test792a4h1e_Amend_EmptyUserDetectionNeverWritesAnEmptyUser(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learnFact()
	rig.dropLines(nr1BasenameA, "user:")

	g.Expect(rig.fields(nr1BasenameA)).NotTo(HaveKey("user"), "the Given: a note with no user:")

	rig.env = nr1Env{remote: nr1Alice.remote, vaultName: nr1Alice.vaultName}

	stdout, _ := rig.amend("--target", nr1BasenameA, "--object", "new")

	g.Expect(stdout).To(Equal(nr1Path(nr1BasenameA) + "\n"))
	g.Expect(rig.note(nr1BasenameA)).NotTo(ContainSubstring("user:"), "no user: field, not even an empty one")
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("object", "new"))
}

// Test792a4h1f_Amend_WithoutAVaultNameTheDefaultIsStamped verifies 792a4h1f (Without a vault name the default is
// stamped): a note made under the vault name alice-vault, amended where neither --vault-name nor
// ENGRAM_VAULT_NAME is set, gets vault: personal and not an empty string.
func Test792a4h1f_Amend_WithoutAVaultNameTheDefaultIsStamped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learnFact()

	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("vault", nr1Alice.vaultName), "the Given")

	rig.env.vaultName = ""

	_, stderr := rig.amend("--target", nr1BasenameA, "--object", "new")

	g.Expect(stderr).To(BeEmpty())
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("vault", "personal"))
	g.Expect(rig.note(nr1BasenameA)).NotTo(ContainSubstring(`vault: ""`))
}

// Test792a4h1g_Amend_RecordingAUseKeepsTheDeclaredIdentity verifies 792a4h1g (Recording a use keeps the declared
// identity): amending a note made in alice's environment with --activate alone, from bob's, leaves repo:, user:
// and vault: as alice's.
func Test792a4h1g_Amend_RecordingAUseKeepsTheDeclaredIdentity(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learnFact()

	rig.env = nr1Bob

	stdout, stderr := rig.amend("--target", nr1BasenameA, "--activate")

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal(nr1Path(nr1BasenameA)+"\n"), "the amend ran and wrote the note")
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("repo", nr1Alice.remote))
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("user", nr1Alice.email))
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("vault", nr1Alice.vaultName))
}

// unexported constants.
const (
	nr1BasenameA = "1.2026-10-09.a"
	nr1Chunks    = "/chunks"
	nr1Vault     = "/vault"
)

// unexported variables.
var (
	errNr1EmbedDown   = errors.New("embedding backend unavailable")
	errNr1NoGit       = errors.New("git: nothing configured")
	errNr1Unsupported = errors.New("not modelled by the in-memory vault")
	nr1Alice          = nr1Env{
		remote: "github.com/alice/widgets", email: "alice@example.com", osUser: "alice", vaultName: "alice-vault",
	}
	nr1Bob = nr1Env{
		remote: "github.com/bob/gadgets", email: "bob@example.com", osUser: "bob", vaultName: "bob-vault",
	}
	nr1Now = time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
)

// nr1Embedder is the stub embedder with a switch that makes every embedding fail.
type nr1Embedder struct {
	stubEmbedder

	fails func() bool
}

func (e nr1Embedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if e.fails() {
		return nil, errNr1EmbedDown
	}

	return e.stubEmbedder.Embed(ctx, text)
}

// nr1Env is the environment a command runs in: the git remote of the working directory ("" = not a git
// repository), git config user.email ("" = unset), the OS username ("" = unresolvable) and ENGRAM_VAULT_NAME
// ("" = unset).
type nr1Env struct {
	remote    string
	email     string
	osUser    string
	vaultName string
}

// nr1FS is a complete in-memory cli.EdgeFS: files by absolute path, and the directories they sit in.
type nr1FS struct {
	mu     sync.Mutex
	files  map[string][]byte
	dirs   map[string]bool
	writes []string
}

func (f *nr1FS) Lstat(name string) (fs.FileInfo, error) { return f.Stat(name) }

func (f *nr1FS) MkdirAll(name string, _ fs.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for dir := path.Clean(name); dir != "/" && dir != "."; dir = path.Dir(dir) {
		f.dirs[dir] = true
	}

	return nil
}

func (f *nr1FS) MkdirTemp(string, string) (string, error) { return "", errNr1Unsupported }

func (f *nr1FS) ReadDir(name string) ([]fs.DirEntry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	dir := path.Clean(name)
	isDir := map[string]bool{}
	exists := f.dirs[dir]

	for file := range f.files {
		rest, below := strings.CutPrefix(file, dir+"/")
		if !below {
			continue
		}

		exists = true
		child, _, nested := strings.Cut(rest, "/")
		isDir[child] = isDir[child] || nested
	}

	for known := range f.dirs {
		if known != dir && path.Dir(known) == dir {
			exists = true
			isDir[path.Base(known)] = true
		}
	}

	if !exists {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}

	names := slices.Sorted(maps.Keys(isDir))
	entries := make([]fs.DirEntry, 0, len(names))

	for _, child := range names {
		entries = append(entries, fakeDirEntry{name: child, dir: isDir[child]})
	}

	return entries, nil
}

func (f *nr1FS) ReadFile(name string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, found := f.files[path.Clean(name)]
	if !found {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	return slices.Clone(data), nil
}

func (f *nr1FS) Readlink(string) (string, error) { return "", errNr1Unsupported }

func (f *nr1FS) Remove(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.files, path.Clean(name))

	return nil
}

func (f *nr1FS) RemoveAll(string) error { return errNr1Unsupported }

func (f *nr1FS) Rename(oldPath, newPath string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, found := f.files[path.Clean(oldPath)]
	if !found {
		return &fs.PathError{Op: "rename", Path: oldPath, Err: fs.ErrNotExist}
	}

	f.files[path.Clean(newPath)] = data
	delete(f.files, path.Clean(oldPath))

	return nil
}

func (f *nr1FS) Stat(name string) (fs.FileInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	clean := path.Clean(name)

	if data, found := f.files[clean]; found {
		return fakeFileInfo{size: int64(len(data)), mtime: nr1Now}, nil
	}

	if f.dirs[clean] {
		return fakeFileInfo{dir: true, mtime: nr1Now}, nil
	}

	for file := range f.files {
		if strings.HasPrefix(file, clean+"/") {
			return fakeFileInfo{dir: true, mtime: nr1Now}, nil
		}
	}

	return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
}

func (f *nr1FS) Symlink(string, string) error { return errNr1Unsupported }

func (f *nr1FS) WalkDir(string, fs.WalkDirFunc) error { return errNr1Unsupported }

func (f *nr1FS) WriteFile(name string, data []byte, _ fs.FileMode) error {
	return f.put(name, data, false)
}

func (f *nr1FS) WriteFileAtomic(name string, data []byte, _ fs.FileMode) error {
	return f.put(name, data, false)
}

func (f *nr1FS) WriteFileExcl(name string, data []byte, _ fs.FileMode) error {
	return f.put(name, data, true)
}

// put stores data at name; exclusive refuses a name that exists.
func (f *nr1FS) put(name string, data []byte, exclusive bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	clean := path.Clean(name)

	if _, found := f.files[clean]; found && exclusive {
		return &fs.PathError{Op: "open", Path: name, Err: fs.ErrExist}
	}

	f.files[clean] = slices.Clone(data)
	f.writes = append(f.writes, clean)

	return nil
}

// sidecarWrites counts the writes made to sidecar files.
func (f *nr1FS) sidecarWrites() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	count := 0

	for _, written := range f.writes {
		if strings.HasSuffix(written, ".vec.json") {
			count++
		}
	}

	return count
}

// nr1Rig is one scenario's vault, environment and command runner.
type nr1Rig struct {
	t          *testing.T
	fs         *nr1FS
	env        nr1Env
	embedFails bool
	setupDone  int
}

// amend runs `engram amend` with args against the rig's vault and chunk index directory.
func (r *nr1Rig) amend(args ...string) (string, string) {
	r.t.Helper()

	return r.run(append(append([]string{"amend"}, args...), "--chunks-dir", nr1Chunks)...)
}

// body returns the note's body without its surrounding blank lines.
func (r *nr1Rig) body(basename string) string {
	r.t.Helper()

	_, body := nr1Split(r.t, r.note(basename))

	return body
}

// commander is a git that answers for the rig's current environment.
func (r *nr1Rig) commander() fakeGitCommander {
	git := fakeGitCommander{
		remoteOut: r.env.remote, configOut: r.env.email, revParseErr: errNr1NoGit,
	}

	if r.env.remote == "" {
		git.remoteErr = errNr1NoGit
	}

	if r.env.email == "" {
		git.configErr = errNr1NoGit
	}

	return git
}

// dropLines removes from the note every frontmatter line that starts with one of the prefixes.
func (r *nr1Rig) dropLines(basename string, prefixes ...string) {
	r.t.Helper()

	lines := strings.Split(r.note(basename), "\n")
	kept := make([]string, 0, len(lines))

	for _, line := range lines {
		if !slices.ContainsFunc(prefixes, func(prefix string) bool { return strings.HasPrefix(line, prefix) }) {
			kept = append(kept, line)
		}
	}

	r.fs.files[nr1Path(basename)] = []byte(strings.Join(kept, "\n"))
	r.setupDone = len(r.fs.writes)
}

// fields returns the note's frontmatter as a map.
func (r *nr1Rig) fields(basename string) map[string]any {
	r.t.Helper()

	fields, _ := nr1Split(r.t, r.note(basename))

	return fields
}

// learn runs `engram learn <kind>` for note 1.2026-10-09.a.
func (r *nr1Rig) learn(kind string, args ...string) {
	r.t.Helper()

	argv := append([]string{"learn", kind, "--slug", "a", "--source", "test", "--position", "top"}, args...)

	stdout, stderr := r.run(argv...)
	if stdout != nr1Path(nr1BasenameA)+"\n" || strings.Contains(stderr, "rror") {
		r.t.Fatalf("learn %s: stdout %q, stderr %q", kind, stdout, stderr)
	}

	r.setupDone = len(r.fs.writes)
}

// learnFact runs `engram learn fact` for the fact of the scenarios: situation "testing", subject s, predicate p
// and object o.
func (r *nr1Rig) learnFact() {
	r.t.Helper()

	r.learn("fact", "--situation", "testing", "--subject", "s", "--predicate", "p", "--object", "o")
}

// note returns the note's file ("" when there is none).
func (r *nr1Rig) note(basename string) string {
	r.t.Helper()

	return string(r.fs.files[nr1Path(basename)])
}

// plantChunkIndex writes a chunk index, as ingest writes one, that holds the given chunk IDs (source#anchor).
func (r *nr1Rig) plantChunkIndex(ids ...string) {
	r.t.Helper()

	records := make([]chunk.Record, 0, len(ids))

	for _, id := range ids {
		source, anchor, _ := strings.Cut(id, "#")
		records = append(records, chunk.Record{
			Source: source, Anchor: anchor, ContentHash: chunk.HashText(id), Text: id, Vector: []float32{1},
		})
	}

	data, err := chunk.EncodeRecords(records)
	if err != nil {
		r.t.Fatalf("encoding the chunk index: %v", err)
	}

	r.fs.files[nr1Chunks+"/index.jsonl"] = data
}

// relabel renames a note, and its sidecar, to basename newName and gives it the Luhmann ID luhmann: the vault
// then holds a note of the name and ID the scenario names.
func (r *nr1Rig) relabel(oldName, newName, luhmann string) {
	r.t.Helper()

	renamed := strings.Replace(r.note(oldName), `luhmann: "1"`, `luhmann: "`+luhmann+`"`, 1)

	r.fs.files[nr1Path(newName)] = []byte(renamed)
	r.fs.files[nr1Sidecar(newName)] = r.fs.files[nr1Sidecar(oldName)]

	delete(r.fs.files, nr1Path(oldName))
	delete(r.fs.files, nr1Sidecar(oldName))

	r.setupDone = len(r.fs.writes)
}

// run runs `engram <args> --vault <vault>` over the real command wiring, returning stdout and stderr.
func (r *nr1Rig) run(args ...string) (string, string) {
	r.t.Helper()

	argv := append(append([]string{"engram"}, args...), "--vault", nr1Vault)

	return executeCapturingBoth(r.t, argv, func(deps *cli.Deps) {
		deps.FS = r.fs
		deps.Lock = fakeLocker{lock: func(string) (func() error, error) { return func() error { return nil }, nil }}
		deps.Embed = nr1Embedder{
			stubEmbedder: stubEmbedder{modelID: "m@4", dims: 4},
			fails:        func() bool { return r.embedFails },
		}
		deps.Now = func() time.Time { return nr1Now }
		deps.Getenv = func(key string) string {
			if key == "ENGRAM_VAULT_NAME" {
				return r.env.vaultName
			}

			return ""
		}
		deps.UserHomeDir = func() (string, error) { return "/nr1-home", nil }
		deps.Getwd = func() (string, error) { return "/work/repo", nil }
		deps.Commander = r.commander()
		deps.Username = func() (string, error) {
			if r.env.osUser == "" {
				return "", errNr1NoGit
			}

			return r.env.osUser, nil
		}
	})
}

// sidecar returns the sidecar file of note 1.2026-10-09.a ("" when there is none).
func (r *nr1Rig) sidecar() string {
	r.t.Helper()

	return string(r.fs.files[nr1Sidecar(nr1BasenameA)])
}

// writesAfterSetup counts the writes made since the Given was set up.
func (r *nr1Rig) writesAfterSetup() int {
	r.t.Helper()

	return len(r.fs.writes) - r.setupDone
}

// nr1NewRig returns a rig with an empty vault in the environment env.
func nr1NewRig(t *testing.T, env nr1Env) *nr1Rig {
	t.Helper()

	return &nr1Rig{
		t:   t,
		env: env,
		fs: &nr1FS{
			files: map[string][]byte{},
			dirs:  map[string]bool{nr1Vault: true, nr1Chunks: true},
		},
	}
}

// nr1Path is the path of the note's file.
func nr1Path(basename string) string { return nr1Vault + "/" + basename + ".md" }

// nr1Sidecar is the path of the note's sidecar file.
func nr1Sidecar(basename string) string { return nr1Vault + "/" + basename + ".vec.json" }

// nr1Split parses a note into its frontmatter and its body.
func nr1Split(t *testing.T, note string) (map[string]any, string) {
	t.Helper()

	rest, found := strings.CutPrefix(note, "---\n")
	if !found {
		t.Fatalf("the note has no frontmatter: %q", note)
	}

	frontmatter, body, found := strings.Cut(rest, "\n---\n")
	if !found {
		t.Fatalf("the frontmatter is not closed: %q", note)
	}

	fields := map[string]any{}

	err := yaml.Unmarshal([]byte(frontmatter), &fields)
	if err != nil {
		t.Fatalf("the frontmatter does not parse: %v", err)
	}

	return fields, strings.TrimSpace(body)
}

// nr1Supersedes returns the entries of the note's supersedes: list.
func nr1Supersedes(fields map[string]any) []map[string]any {
	list, _ := fields["supersedes"].([]any)
	entries := make([]map[string]any, 0, len(list))

	for _, item := range list {
		if entry, isMap := item.(map[string]any); isMap {
			entries = append(entries, entry)
		}
	}

	return entries
}
