package cli_test

// Scenario tests for the note-revision product capability, part 2 (861c2zzw): the vault name on every amend
// (792a4i), untouched frontmatter (792a4j), anchored keys (792a4k), resituation (792a4l), CRLF notes (792a4m),
// activation (792a4n) and concurrent revisions (792a4o). Spec: docs/specs/product/note-revision.md.
//
// Every scenario runs `engram amend`, `engram resituate` or `engram activate` through the real command wiring
// (cli.Targets, flag parsing, the vault lock, the scan and the production dependencies) over the in-memory vault
// of part 1 (nr1Rig, a complete cli.EdgeFS), with the environment a scenario names pinned by nr1Env. The notes of
// a Given are written by `engram learn` in the same rig; a Given that needs another shape (an unmodeled key, an
// anchor, a CRLF note, a note without a sidecar) edits that output by one named helper (nr2Insert, nr2Replace,
// nr2AnchorKey, nr2CRLF, nr2SetLastUsed). The two concurrency scenarios give each command its own file system
// view (nr2HookFS) over the shared vault, and hold each command at the write whose loss the vault lock prevents
// until its peer has reached the same point or has been held up by the lock. No temporary directory, no real
// home, no git, no network.

import (
	"io/fs"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792a4i1a_Amend_AUseRecordedOnANoteWithoutVaultFillsItIn verifies 792a4i1a (A use recorded on a note without
// vault fills it in): a note with no user: and no vault:, amended with --activate where no vault name is
// configured (and where a user could be detected), gains vault: personal and gains no user: field.
func Test792a4i1a_Amend_AUseRecordedOnANoteWithoutVaultFillsItIn(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learnFact()
	rig.dropLines(nr1BasenameA, "user:", "vault:")

	g.Expect(rig.fields(nr1BasenameA)).NotTo(HaveKey("user"), "the Given: no user:")
	g.Expect(rig.fields(nr1BasenameA)).NotTo(HaveKey("vault"), "the Given: no vault:")

	rig.env.vaultName = ""

	stdout, stderr := rig.amend("--target", nr1BasenameA, "--activate")

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal(nr1Path(nr1BasenameA) + "\n"))
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("vault", "personal"))
	g.Expect(rig.fields(nr1BasenameA)).NotTo(HaveKey("user"))
	g.Expect(rig.note(nr1BasenameA)).NotTo(ContainSubstring("user:"))
}

// Test792a4j1a_Amend_KeepsAnUnmodeledKey verifies 792a4j1a (Amend keeps an unmodeled key): a fact, feedback or
// runbook note whose frontmatter carries luhmann_old: "12" and a nested map that engram does not define, amended
// with a content flag, --supersedes or --chunk-source, still carries luhmann_old: "12" and the nested map with
// the same values.
func Test792a4j1a_Amend_KeepsAnUnmodeledKey(t *testing.T) {
	t.Parallel()

	for _, kind := range nr2Kinds() {
		for form, flags := range nr2AmendForms(kind) {
			t.Run(kind.name+" with "+form, func(t *testing.T) {
				t.Parallel()
				g := NewWithT(t)

				rig := nr2NewNote(t, kind)
				nr2Insert(t, rig, nr1BasenameA, nr2Unmodeled)

				g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("luhmann_old", "12"), "the Given")

				_, stderr := rig.amend(append([]string{"--target", nr1BasenameA}, flags...)...)

				g.Expect(stderr).To(BeEmpty())
				g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("luhmann_old", "12"))
				g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("provenance", nr2UnmodeledMap))
			})
		}
	}
}

// Test792a4j1b_Amend_KeepsAnAnchorOnAKeyItDoesNotEdit verifies 792a4j1b (Amend keeps an anchor on a key it does
// not edit): a fact note whose source: &src test is aliased by another key, amended with --object new, succeeds,
// and source: and the aliasing key both still read test, the anchor and the alias still in the file.
func Test792a4j1b_Amend_KeepsAnAnchorOnAKeyItDoesNotEdit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learnFact()
	nr2AnchorKey(t, rig, nr1BasenameA, "source", "src")

	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("source", "test"), "the Given: source: &src test")
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("x_src_ref", "test"), "the Given: an alias of it")

	stdout, stderr := rig.amend("--target", nr1BasenameA, "--object", "new")

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal(nr1Path(nr1BasenameA) + "\n"))
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("object", "new"), "the amend wrote the note")
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("source", "test"))
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("x_src_ref", "test"))
	g.Expect(rig.note(nr1BasenameA)).To(ContainSubstring("source: &src test"))
	g.Expect(rig.note(nr1BasenameA)).To(ContainSubstring("x_src_ref: *src"))
}

// Test792a4j1c_Amend_KeepsTheExtraKeysOfAKeptSupersession verifies 792a4j1c (Amend keeps the extra keys of a kept
// supersession): a note whose supersedes: names A with an extra key x_reason: kept, and names B, amended with
// --supersedes "A|refutes|new" --supersedes "C|narrows|c", has an A entry with type: refutes, claim: new and
// x_reason: kept, no B entry, and a C entry with only note, type and claim. The second case names A in the old
// entry with its .md, which the match ignores.
func Test792a4j1c_Amend_KeepsTheExtraKeysOfAKeptSupersession(t *testing.T) {
	t.Parallel()

	for name, given := range map[string]string{"A named as A": "A", "A named as A.md": "A.md"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			rig := nr1NewRig(t, nr1Alice)
			rig.learn("fact", "--situation", "testing", "--subject", "s", "--predicate", "p", "--object", "o",
				"--supersedes", given+"|narrows|old a", "--supersedes", "B|updates|old b")
			nr2Replace(t, rig, nr1BasenameA, "claim: old a\n", "claim: old a\n      x_reason: kept\n")

			before := nr1Supersedes(rig.fields(nr1BasenameA))
			g.Expect(before).To(HaveLen(2), "the Given: A and B")

			if len(before) != 2 {
				return
			}

			g.Expect(before[0]).To(HaveKeyWithValue("x_reason", "kept"), "the Given: A carries x_reason")

			_, stderr := rig.amend("--target", nr1BasenameA,
				"--supersedes", "A|refutes|new", "--supersedes", "C|narrows|c")

			g.Expect(stderr).To(BeEmpty())

			entries := nr1Supersedes(rig.fields(nr1BasenameA))
			g.Expect(entries).To(HaveLen(2), "A and C, and B is gone")

			if len(entries) != 2 {
				return
			}

			g.Expect(entries[0]).To(Equal(map[string]any{
				"note": "A", "type": "refutes", "claim": "new", "x_reason": "kept",
			}))
			g.Expect(entries[1]).To(Equal(map[string]any{"note": "C", "type": "narrows", "claim": "c"}))
			g.Expect(rig.note(nr1BasenameA)).NotTo(ContainSubstring("old b"))
		})
	}
}

// Test792a4j1d_Resituate_KeepsEveryUntouchedField verifies 792a4j1d (Resituate keeps every untouched field): a
// fact note carrying tags:, sources:, supersedes:, pending: true, parent:, aliases: and luhmann_old: "12",
// resituated with --situation new, has all of those unchanged, and only its situation: line and the first line of
// its body differ.
func Test792a4j1d_Resituate_KeepsEveryUntouchedField(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learn("fact", "--situation", "testing", "--subject", "s", "--predicate", "p", "--object", "o",
		"--tag", "topic/revision", "--chunk-source", "a#1", "--supersedes", "9.2026-01-01.old|narrows|old")
	nr2Insert(t, rig, nr1BasenameA, "pending: true\n"+nr2Unmodeled+nr2ExchangeBlock())

	before := rig.note(nr1BasenameA)
	beforeFields := rig.fields(nr1BasenameA)

	for _, key := range []string{"tags", "sources", "supersedes", "pending", "parent", "aliases", "luhmann_old"} {
		g.Expect(beforeFields).To(HaveKey(key), "the Given carries "+key)
	}

	stdout, stderr := rig.run("resituate", "--note", nr1BasenameA, "--situation", "new")

	g.Expect(stderr).To(HavePrefix("warning: vault holds pending offer(s)"), "the pending note is the vault's upkeep")
	g.Expect(strings.Count(stderr, "\n")).To(Equal(1), "that warning and no error")
	g.Expect(stdout).To(Equal(nr1Path(nr1BasenameA) + "\n"))

	afterFields := rig.fields(nr1BasenameA)
	g.Expect(afterFields).To(HaveKeyWithValue("situation", "new"))

	delete(beforeFields, "situation")
	delete(afterFields, "situation")
	g.Expect(afterFields).To(Equal(beforeFields), "every field but situation: is unchanged")
	g.Expect(nr2ChangedLines(before, rig.note(nr1BasenameA))).To(Equal([]string{
		"situation: testing -> situation: new",
		"Information learned: when in testing, s p o. -> Information learned: when in new, s p o.",
	}), "only situation: and the first line of the body differ")
}

// Test792a4k1a_Amend_RefusesAnAnchoredEditedKey verifies 792a4k1a (Amend refuses an anchored edited key): a fact
// note whose object: value carries an anchor that another key aliases, amended with --object new, fails naming
// object, and the note and its sidecar are unchanged.
func Test792a4k1a_Amend_RefusesAnAnchoredEditedKey(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learnFact()
	nr2AnchorKey(t, rig, nr1BasenameA, "object", "obj")

	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("x_obj_ref", "o"), "the Given: the alias reads the value")

	before := rig.note(nr1BasenameA)
	sidecarBefore := rig.sidecar()

	stdout, stderr := rig.amend("--target", nr1BasenameA, "--object", "new")

	g.Expect(stdout).To(BeEmpty())
	g.Expect(stderr).To(ContainSubstring("carries a YAML anchor: object"))
	g.Expect(rig.note(nr1BasenameA)).To(Equal(before), "the note is unchanged")
	g.Expect(rig.sidecar()).To(Equal(sidecarBefore), "the sidecar is unchanged")
	g.Expect(rig.writesAfterSetup()).To(BeZero(), "nothing was written")
}

// Test792a4k1b_Resituate_RefusesAnAnchoredSituation verifies 792a4k1b (Resituate refuses an anchored situation):
// a fact note whose situation: or created: carries an anchor, resituated with --situation new, fails naming that
// key, and the note and its sidecar are unchanged.
func Test792a4k1b_Resituate_RefusesAnAnchoredSituation(t *testing.T) {
	t.Parallel()

	for _, key := range []string{"situation", "created"} {
		t.Run(key, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			rig := nr1NewRig(t, nr1Alice)
			rig.learnFact()
			nr2AnchorKey(t, rig, nr1BasenameA, key, "anc")

			g.Expect(rig.fields(nr1BasenameA)).To(HaveKey("x_anc_ref"), "the Given: an alias of the anchor")

			before := rig.note(nr1BasenameA)
			sidecarBefore := rig.sidecar()

			stdout, stderr := rig.run("resituate", "--note", nr1BasenameA, "--situation", "new")

			g.Expect(stdout).To(BeEmpty())
			g.Expect(stderr).To(ContainSubstring("carries a YAML anchor: " + key))
			g.Expect(rig.note(nr1BasenameA)).To(Equal(before), "the note is unchanged")
			g.Expect(rig.sidecar()).To(Equal(sidecarBefore), "the sidecar is unchanged")
			g.Expect(rig.writesAfterSetup()).To(BeZero(), "nothing was written")
		})
	}
}

// Test792a4l1a_Resituate_AFactIsResituated verifies 792a4l1a (A fact is resituated): a fact whose situation is
// "old" and whose body reads "Information learned: when in old, s p o." followed by a related-notes section,
// resituated with --situation new, has situation: new, the first body line "Information learned: when in new,
// s p o.", the related-notes section unchanged, a rebuilt sidecar, and its path printed.
func Test792a4l1a_Resituate_AFactIsResituated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const related = "Related to:\n- [[9a.2026-05-01.nilaway-basics]] — foundational concept."

	rig := nr1NewRig(t, nr1Alice)
	rig.learn("fact", "--situation", "old", "--subject", "s", "--predicate", "p", "--object", "o")
	nr2Append(rig, nr1BasenameA, "\n"+related+"\n")

	sidecarBefore := nr2Sidecar(t, rig, nr1BasenameA)
	bodyBefore := rig.body(nr1BasenameA)

	g.Expect(rig.body(nr1BasenameA)).To(HavePrefix("Information learned: when in old, s p o."), "the Given")
	g.Expect(rig.body(nr1BasenameA)).To(HaveSuffix(related), "the Given: a related-notes section")

	stdout, stderr := rig.run("resituate", "--note", nr1BasenameA, "--situation", "new")

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal(nr1Path(nr1BasenameA) + "\n"))
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("situation", "new"))

	opener, rest, _ := strings.Cut(rig.body(nr1BasenameA), "\n")
	_, restBefore, _ := strings.Cut(bodyBefore, "\n")

	g.Expect(opener).To(Equal("Information learned: when in new, s p o."))
	g.Expect(rest).To(Equal(restBefore), "the rest of the body, the related-notes section included, is unchanged")
	g.Expect(rest).To(HaveSuffix(related))

	sidecarAfter := nr2Sidecar(t, rig, nr1BasenameA)
	g.Expect(sidecarAfter.ContentHash).NotTo(Equal(sidecarBefore.ContentHash), "the sidecar was rebuilt")
	nr2ExpectFreshSidecar(g, rig)
}

// Test792a4l1b_Resituate_ARunbookCannotBeResituated verifies 792a4l1b (A runbook cannot be resituated): in a vault
// holding runbook note 2, `engram resituate --note 2 --situation new` fails saying the note type runbook is
// unknown, and the note and its sidecar are unchanged.
func Test792a4l1b_Resituate_ARunbookCannotBeResituated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const runbook = "2.2026-10-09.a"

	rig := nr1NewRig(t, nr1Alice)
	rig.learn("runbook", "--situation", "x", "--done-when", "done", "--body", "1. step", "--red-flag", "A")
	rig.relabel(nr1BasenameA, runbook, "2")

	before := rig.note(runbook)
	sidecarBefore := string(rig.fs.files[nr1Sidecar(runbook)])

	g.Expect(rig.fields(runbook)).To(HaveKeyWithValue("type", "runbook"), "the Given: runbook note 2")

	stdout, stderr := rig.run("resituate", "--note", "2", "--situation", "new")

	g.Expect(stdout).To(BeEmpty())
	g.Expect(stderr).To(ContainSubstring(`resituate: unknown note type: "runbook"`))
	g.Expect(rig.note(runbook)).To(Equal(before), "the note is unchanged")
	g.Expect(string(rig.fs.files[nr1Sidecar(runbook)])).To(Equal(sidecarBefore), "the sidecar is unchanged")
	g.Expect(rig.writesAfterSetup()).To(BeZero(), "nothing was written")
}

// Test792a4l1c_Resituate_AFailedSidecarRebuildFailsTheCommand verifies 792a4l1c (A failed sidecar rebuild fails
// the command): when embedding fails for the note being resituated, `engram resituate --note <note> --situation
// new` fails naming the note, and the note's situation is already new.
func Test792a4l1c_Resituate_AFailedSidecarRebuildFailsTheCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learnFact()

	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("situation", "testing"), "the Given")

	rig.embedFails = true

	stdout, stderr := rig.run("resituate", "--note", nr1BasenameA, "--situation", "new")

	g.Expect(stdout).To(BeEmpty(), "no path is printed for a failed command")
	g.Expect(stderr).To(ContainSubstring("resituate: embedding " + nr1Path(nr1BasenameA)))
	g.Expect(stderr).To(ContainSubstring(errNr1EmbedDown.Error()))
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("situation", "new"), "the note is already rewritten")
}

// Test792a4m1a_Amend_ConvertsACRLFNote verifies 792a4m1a (Amend converts a CRLF note): a fact, feedback or
// runbook note whose lines end in \r\n, amended with a content flag, --supersedes, --chunk-source or --activate,
// is amended without complaint, and the written note contains no \r\n, is byte-identical to what the same amend
// writes for the note's LF form, and has a fresh sidecar, written by that amend.
func Test792a4m1a_Amend_ConvertsACRLFNote(t *testing.T) {
	t.Parallel()

	for _, kind := range nr2Kinds() {
		forms := nr2AmendForms(kind)
		forms["--activate"] = []string{"--activate"}

		for form, flags := range forms {
			t.Run(kind.name+" with "+form, func(t *testing.T) {
				t.Parallel()
				g := NewWithT(t)

				lfRig := nr2NewNote(t, kind)
				crlfRig := nr2NewNote(t, kind)
				nr2CRLF(t, crlfRig, nr1BasenameA)

				g.Expect(crlfRig.note(nr1BasenameA)).To(ContainSubstring("\r\n"), "the Given: lines end in \\r\\n")

				sidecarWrites := crlfRig.fs.sidecarWrites()
				argv := append([]string{"--target", nr1BasenameA}, flags...)

				lfStdout, lfStderr := lfRig.amend(argv...)
				crlfStdout, crlfStderr := crlfRig.amend(argv...)

				g.Expect(lfStderr).To(BeEmpty(), "the same amend of the LF form succeeds")
				g.Expect(lfStdout).To(Equal(nr1Path(nr1BasenameA) + "\n"))
				g.Expect(crlfStderr).To(BeEmpty())
				g.Expect(crlfStdout).To(Equal(nr1Path(nr1BasenameA) + "\n"))
				g.Expect(crlfRig.note(nr1BasenameA)).NotTo(ContainSubstring("\r\n"))
				g.Expect(crlfRig.note(nr1BasenameA)).To(Equal(lfRig.note(nr1BasenameA)))
				g.Expect(crlfRig.fs.sidecarWrites()).To(BeNumerically(">", sidecarWrites), "the sidecar was written")
				nr2ExpectFreshSidecar(g, crlfRig)
			})
		}
	}
}

// Test792a4m1b_Resituate_ConvertsACRLFNote verifies 792a4m1b (Resituate converts a CRLF note): a fact or feedback
// note whose lines end in \r\n, resituated with --situation new, is rewritten without complaint, the written note
// contains no \r, only its situation: line and the first line of its body changed, it is byte-identical to what
// the same resituation writes for the note's LF form, and its sidecar was rebuilt.
func Test792a4m1b_Resituate_ConvertsACRLFNote(t *testing.T) {
	t.Parallel()

	for _, kind := range nr2Kinds()[:2] {
		t.Run(kind.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			lfRig := nr2NewNote(t, kind)
			crlfRig := nr2NewNote(t, kind)
			nr2CRLF(t, crlfRig, nr1BasenameA)

			lfBefore := lfRig.note(nr1BasenameA)
			sidecarWrites := crlfRig.fs.sidecarWrites()

			g.Expect(crlfRig.note(nr1BasenameA)).To(ContainSubstring("\r\n"), "the Given: lines end in \\r\\n")

			_, lfStderr := lfRig.run("resituate", "--note", nr1BasenameA, "--situation", "new")
			stdout, stderr := crlfRig.run("resituate", "--note", nr1BasenameA, "--situation", "new")

			g.Expect(lfStderr).To(BeEmpty(), "the same resituation of the LF form succeeds")
			g.Expect(stderr).To(BeEmpty())
			g.Expect(stdout).To(Equal(nr1Path(nr1BasenameA) + "\n"))

			written := crlfRig.note(nr1BasenameA)
			g.Expect(written).NotTo(ContainSubstring("\r"))
			g.Expect(written).To(Equal(lfRig.note(nr1BasenameA)))
			g.Expect(nr2ChangedLines(lfBefore, written)).To(HaveLen(2), "only situation: and the body's first line")
			g.Expect(crlfRig.fields(nr1BasenameA)).To(HaveKeyWithValue("situation", "new"))
			g.Expect(crlfRig.fs.sidecarWrites()).To(BeNumerically(">", sidecarWrites), "the sidecar was rebuilt")
			nr2ExpectFreshSidecar(g, crlfRig)
		})
	}
}

// Test792a4n1a_Activate_ANotesUseIsRecorded verifies 792a4n1a (A note's use is recorded): `engram activate --note
// <basename>` of a note with a current sidecar, last used before, makes the sidecar's last_used today, leaves its
// vectors and content hash unchanged and the note file unchanged, and succeeds.
func Test792a4n1a_Activate_ANotesUseIsRecorded(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learnFact()
	nr2SetLastUsed(t, rig, nr1BasenameA, nr2EarlierUse)

	noteBefore := rig.note(nr1BasenameA)
	before := nr2Sidecar(t, rig, nr1BasenameA)

	g.Expect(before.LastUsed).To(Equal(nr2EarlierUse), "the Given: last used before")

	stdout, stderr := rig.run("activate", "--note", nr1BasenameA)

	g.Expect(stderr).To(BeEmpty(), "the command succeeds")
	g.Expect(stdout).To(BeEmpty())

	after := nr2Sidecar(t, rig, nr1BasenameA)
	g.Expect(after.LastUsed).To(Equal(nr2Today))

	after.LastUsed = before.LastUsed
	g.Expect(after).To(Equal(before), "vectors, content hash and the rest of the sidecar are unchanged")
	g.Expect(rig.note(nr1BasenameA)).To(Equal(noteBefore), "the note file is unchanged")
}

// Test792a4n1b_Activate_ANoteWithoutASidecarStillCounts verifies 792a4n1b (A note without a sidecar still
// counts): `engram activate --note <basename>` of a note that has no sidecar succeeds and writes no sidecar.
func Test792a4n1b_Activate_ANoteWithoutASidecarStillCounts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learnFact()
	delete(rig.fs.files, nr1Sidecar(nr1BasenameA))

	noteBefore := rig.note(nr1BasenameA)
	rig.setupDone = len(rig.fs.writes)

	stdout, stderr := rig.run("activate", "--note", nr1BasenameA)

	g.Expect(stderr).To(BeEmpty(), "the command succeeds")
	g.Expect(stdout).To(BeEmpty())
	g.Expect(rig.fs.files).NotTo(HaveKey(nr1Sidecar(nr1BasenameA)), "no sidecar is written")
	g.Expect(rig.writesAfterSetup()).To(BeZero())
	g.Expect(rig.note(nr1BasenameA)).To(Equal(noteBefore))
}

// Test792a4n1d_Amend_RecordsAUse verifies 792a4n1d (Amend records a use): `engram amend --target <note>
// --activate` of a note with a current sidecar, last used before, makes the sidecar's last_used today.
func Test792a4n1d_Amend_RecordsAUse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learnFact()
	nr2SetLastUsed(t, rig, nr1BasenameA, nr2EarlierUse)

	g.Expect(nr2Sidecar(t, rig, nr1BasenameA).LastUsed).To(Equal(nr2EarlierUse), "the Given: last used before")

	_, stderr := rig.amend("--target", nr1BasenameA, "--activate")

	g.Expect(stderr).To(BeEmpty())
	g.Expect(nr2Sidecar(t, rig, nr1BasenameA).LastUsed).To(Equal(nr2Today))
}

// Test792a4o1a_Amend_ConcurrentAmendsBothTakeEffect verifies 792a4o1a (Concurrent amends both take effect):
// `engram amend --target <note> --subject s2` and `engram amend --target <note> --chunk-source a#1`, started at
// the same time, leave the note with both subject: s2 and a#1 in sources:. Each amend is held at its write of the
// note until the other has reached its own write or has been held up by the vault lock, so without the lock both
// read the note as it was and the later write overwrites the earlier one's change.
func Test792a4o1a_Amend_ConcurrentAmendsBothTakeEffect(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learnFact()
	rig.plantChunkIndex("a#1")

	contention := &nr2Latch{opened: make(chan struct{})}
	locker := &ci2Locker{held: map[string]*sync.Mutex{}, onContention: contention.open}
	meeting := &nr2Meeting{met: make(chan struct{})}

	var (
		running sync.WaitGroup
		mu      sync.Mutex
		stderrs []string
	)

	for _, flags := range [][]string{{"--subject", "s2"}, {"--chunk-source", "a#1"}} {
		running.Go(func() {
			view := &nr2HookFS{nr1FS: rig.fs, beforeWrite: func(path string) {
				if path == nr1Path(nr1BasenameA) {
					meeting.arrive(contention.opened)
				}
			}}

			_, stderr := nr2Run(rig, view, locker,
				append([]string{"amend", "--target", nr1BasenameA, "--chunks-dir", nr1Chunks}, flags...)...)

			mu.Lock()
			defer mu.Unlock()

			stderrs = append(stderrs, stderr)
		})
	}

	running.Wait()

	g.Expect(stderrs).To(ConsistOf(BeEmpty(), BeEmpty()), "both amends succeed")
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("subject", "s2"), "the --subject amend took effect")
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("sources", ContainElement("a#1")),
		"the --chunk-source amend took effect")
}

// Test792a4o1b_Activate_AnActivationDoesNotUndoAReEmbed verifies 792a4o1b (An activation does not undo a re-embed):
// `engram amend --target <note> --object new` and `engram activate --note <note>`, started at the same time on a
// note with a current sidecar, leave a sidecar that holds the vectors of the amended text. The activation is held
// between reading the sidecar and writing it until the amend has written the re-embedded sidecar (and the amend
// before that write until the activation has read the sidecar), or until the vault lock holds one of them up, so
// without the lock the activation writes back the vectors it read.
func Test792a4o1b_Activate_AnActivationDoesNotUndoAReEmbed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, nr1Alice)
	rig.learnFact()

	before := nr2Sidecar(t, rig, nr1BasenameA)
	sidecarPath := nr1Sidecar(nr1BasenameA)
	contention := &nr2Latch{opened: make(chan struct{})}
	locker := &ci2Locker{held: map[string]*sync.Mutex{}, onContention: contention.open}
	activationRead := &nr2Latch{opened: make(chan struct{})}
	amendWritten := &nr2Latch{opened: make(chan struct{})}

	amendView := &nr2HookFS{
		nr1FS: rig.fs,
		beforeWrite: func(path string) {
			if path == sidecarPath {
				activationRead.wait(contention.opened)
			}
		},
		afterWrite: func(path string) {
			if path == sidecarPath {
				amendWritten.open()
			}
		},
	}
	activateView := &nr2HookFS{
		nr1FS: rig.fs,
		afterRead: func(path string) {
			if path == sidecarPath {
				activationRead.open()
			}
		},
		beforeWrite: func(path string) {
			if path == sidecarPath {
				amendWritten.wait(contention.opened)
			}
		},
	}

	var (
		running                   sync.WaitGroup
		amendStderr, activeStderr string
	)

	running.Go(func() {
		_, amendStderr = nr2Run(rig, amendView, locker,
			"amend", "--target", nr1BasenameA, "--object", "new", "--chunks-dir", nr1Chunks)
	})
	running.Go(func() {
		_, activeStderr = nr2Run(rig, activateView, locker, "activate", "--note", nr1BasenameA)
	})

	running.Wait()

	g.Expect(amendStderr).To(BeEmpty())
	g.Expect(activeStderr).To(BeEmpty())
	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("object", "new"))

	nr2ExpectFreshSidecar(g, rig)

	after := nr2Sidecar(t, rig, nr1BasenameA)
	g.Expect(after.ContentHash).NotTo(Equal(before.ContentHash), "the sidecar describes the amended text, not the old")
}

// unexported constants.
const (
	nr2EarlierUse = "2026-09-01"
	nr2Today      = "2026-10-09"
	nr2Unmodeled  = "luhmann_old: \"12\"\nprovenance:\n    origin: import\n    steps:\n        - a\n        - b\n"
)

// unexported variables.
var (
	nr2UnmodeledMap = map[string]any{"origin": "import", "steps": []any{"a", "b"}}
)

// nr2HookFS is one command's view of the shared in-memory vault: every operation is the vault's, and the hooks
// are called around a read and a write so that a test can hold the command at that point.
type nr2HookFS struct {
	*nr1FS

	afterRead   func(path string)
	beforeWrite func(path string)
	afterWrite  func(path string)
}

func (v *nr2HookFS) ReadFile(name string) ([]byte, error) {
	data, err := v.nr1FS.ReadFile(name)

	if v.afterRead != nil {
		v.afterRead(name)
	}

	return data, err
}

func (v *nr2HookFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return v.write(name, func() error { return v.nr1FS.WriteFile(name, data, perm) })
}

func (v *nr2HookFS) WriteFileAtomic(name string, data []byte, perm fs.FileMode) error {
	return v.write(name, func() error { return v.nr1FS.WriteFileAtomic(name, data, perm) })
}

func (v *nr2HookFS) WriteFileExcl(name string, data []byte, perm fs.FileMode) error {
	return v.write(name, func() error { return v.nr1FS.WriteFileExcl(name, data, perm) })
}

// write runs the vault's write between the hooks.
func (v *nr2HookFS) write(name string, perform func() error) error {
	if v.beforeWrite != nil {
		v.beforeWrite(name)
	}

	err := perform()

	if v.afterWrite != nil {
		v.afterWrite(name)
	}

	return err
}

// nr2Kind is one note kind of the scenarios: how `engram learn` makes the note, and a content flag of the kind.
type nr2Kind struct {
	name    string
	learn   []string
	content []string
}

// nr2Latch is opened once, by one event, and waited for by the other run; a wait ends too when the lock holds a
// run up (contended opens) or after the bound, which is a failure bound and not a synchronisation delay.
type nr2Latch struct {
	once   sync.Once
	opened chan struct{}
}

func (l *nr2Latch) open() { l.once.Do(func() { close(l.opened) }) }

func (l *nr2Latch) wait(contended <-chan struct{}) {
	select {
	case <-l.opened:
	case <-contended:
	case <-time.After(ci2WaitBound):
	}
}

// nr2Meeting holds each run that arrives until its peer has arrived too, the lock has held a run up, or the bound.
type nr2Meeting struct {
	arrivals atomic.Int32
	once     sync.Once
	met      chan struct{}
}

func (m *nr2Meeting) arrive(contended <-chan struct{}) {
	const peers = 2

	if m.arrivals.Add(1) >= peers {
		m.once.Do(func() { close(m.met) })
	}

	select {
	case <-m.met:
	case <-contended:
	case <-time.After(ci2WaitBound):
	}
}

// nr2AmendForms is the amends of 792a4j1a and 792a4m1a for kind: by name, the flags after --target <note>. The
// chunk index holding a#1 is planted by nr2NewNote.
func nr2AmendForms(kind nr2Kind) map[string][]string {
	return map[string][]string{
		"a content flag": kind.content,
		"--supersedes":   {"--supersedes", "B|updates|b"},
		"--chunk-source": {"--chunk-source", "a#1"},
	}
}

// nr2AnchorKey puts the anchor &<anchor> on the value of key in the note and adds the key x_<anchor>_ref that
// aliases it, the one-line edit of a note a human or Obsidian anchored.
func nr2AnchorKey(t *testing.T, rig *nr1Rig, basename, key, anchor string) {
	t.Helper()

	nr2Replace(t, rig, basename, "\n"+key+": ", "\n"+key+": &"+anchor+" ")
	nr2Insert(t, rig, basename, "x_"+anchor+"_ref: *"+anchor+"\n")
}

// nr2Append adds text to the end of the note's file.
func nr2Append(rig *nr1Rig, basename, text string) {
	rig.fs.files[nr1Path(basename)] = append(rig.fs.files[nr1Path(basename)], []byte(text)...)
	rig.setupDone = len(rig.fs.writes)
}

// nr2CRLF turns every line ending of the note into \r\n, leaving the sidecar the LF note was learned with.
func nr2CRLF(t *testing.T, rig *nr1Rig, basename string) {
	t.Helper()

	rig.fs.files[nr1Path(basename)] = []byte(toCRLF(rig.note(basename)))
	rig.setupDone = len(rig.fs.writes)
}

// nr2ChangedLines lists, as "before -> after", the lines of after that differ from the same line of before; the
// two notes must have as many lines.
func nr2ChangedLines(before, after string) []string {
	beforeLines := strings.Split(before, "\n")
	afterLines := strings.Split(after, "\n")

	if len(beforeLines) != len(afterLines) {
		return []string{"the line counts differ: " + before + " -> " + after}
	}

	changed := make([]string, 0, 2)

	for index, line := range beforeLines {
		if line != afterLines[index] {
			changed = append(changed, line+" -> "+afterLines[index])
		}
	}

	return changed
}

// nr2ExchangeBlock is the frontmatter lines of a note pulled down from a parent: xid, parent, aliases and offer.
func nr2ExchangeBlock() string {
	block, _ := yaml.Marshal(exchangeSurvivalFixture())

	return string(block)
}

// nr2ExpectFreshSidecar asserts the sidecar of note 1.2026-10-09.a holds the model, dimensions, vectors and
// content hash of the note as it is now.
func nr2ExpectFreshSidecar(g Gomega, rig *nr1Rig) {
	rig.t.Helper()

	got := nr2Sidecar(rig.t, rig, nr1BasenameA)

	want, err := embed.BuildSidecar(
		rig.t.Context(), stubEmbedder{modelID: "m@4", dims: 4}, []byte(rig.note(nr1BasenameA)))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got.ContentHash).To(Equal(want.ContentHash))
	g.Expect(got.SituationVector).To(Equal(want.SituationVector))
	g.Expect(got.BodyVector).To(Equal(want.BodyVector))
	g.Expect(got.EmbeddingModelID).To(Equal(want.EmbeddingModelID))
	g.Expect(got.Dims).To(Equal(want.Dims))
}

// nr2Insert adds the frontmatter lines (each ending in a newline) to the end of the note's frontmatter.
func nr2Insert(t *testing.T, rig *nr1Rig, basename, lines string) {
	t.Helper()

	note := rig.note(basename)

	closing := strings.Index(note[len("---\n"):], "\n---\n")
	if closing < 0 {
		t.Fatalf("the note has no closing frontmatter delimiter: %q", note)
	}

	closing += len("---\n") + 1

	rig.fs.files[nr1Path(basename)] = []byte(note[:closing] + lines + note[closing:])
	rig.setupDone = len(rig.fs.writes)
}

// nr2Kinds is the three kinds of note of the scenarios, each with the flags `engram learn` makes it with and a
// content flag of the kind.
func nr2Kinds() []nr2Kind {
	return []nr2Kind{
		{
			name: "fact",
			learn: []string{
				"fact", "--situation", "testing", "--subject", "s", "--predicate", "p", "--object", "o",
			},
			content: []string{"--object", "new"},
		},
		{
			name:    "feedback",
			learn:   []string{"feedback", "--situation", "x", "--action", "a"},
			content: []string{"--action", "new"},
		},
		{
			name: "runbook",
			learn: []string{
				"runbook", "--situation", "x", "--done-when", "done", "--body", "1. old", "--red-flag", "A",
			},
			content: []string{"--done-when", "new"},
		},
	}
}

// nr2NewNote returns a rig in alice's environment holding note 1.2026-10-09.a of the kind, made by `engram
// learn`, and a chunk index holding a#1.
func nr2NewNote(t *testing.T, kind nr2Kind) *nr1Rig {
	t.Helper()

	rig := nr1NewRig(t, nr1Alice)
	rig.learn(kind.learn[0], kind.learn[1:]...)
	rig.plantChunkIndex("a#1")

	return rig
}

// nr2Replace replaces the first occurrence of old in the note with replacement, the named one-line edit of a note.
func nr2Replace(t *testing.T, rig *nr1Rig, basename, old, replacement string) {
	t.Helper()

	note := rig.note(basename)
	if !strings.Contains(note, old) {
		t.Fatalf("the note has no %q: %q", old, note)
	}

	rig.fs.files[nr1Path(basename)] = []byte(strings.Replace(note, old, replacement, 1))
	rig.setupDone = len(rig.fs.writes)
}

// nr2Run runs `engram <args> --vault <vault>` as rig.run does, over the given view of the vault and lock, so that
// two commands can run at once against the rig's vault.
func nr2Run(rig *nr1Rig, view cli.EdgeFS, locker cli.FileLocker, args ...string) (string, string) {
	rig.t.Helper()

	argv := append(append([]string{"engram"}, args...), "--vault", nr1Vault)

	return executeCapturingBoth(rig.t, argv, func(deps *cli.Deps) {
		deps.FS = view
		deps.Lock = locker
		deps.Embed = nr1Embedder{
			stubEmbedder: stubEmbedder{modelID: "m@4", dims: 4},
			fails:        func() bool { return rig.embedFails },
		}
		deps.Now = func() time.Time { return nr1Now }
		deps.Getenv = func(key string) string {
			if key == "ENGRAM_VAULT_NAME" {
				return rig.env.vaultName
			}

			return ""
		}
		deps.UserHomeDir = func() (string, error) { return "/nr2-home", nil }
		deps.Getwd = func() (string, error) { return "/work/repo", nil }
		deps.Commander = rig.commander()
		deps.Username = func() (string, error) {
			if rig.env.osUser == "" {
				return "", errNr1NoGit
			}

			return rig.env.osUser, nil
		}
	})
}

// nr2SetLastUsed makes the note's sidecar record a use on date, as an earlier activation would have.
func nr2SetLastUsed(t *testing.T, rig *nr1Rig, basename, date string) {
	t.Helper()

	sidecar := nr2Sidecar(t, rig, basename)
	sidecar.LastUsed = date

	rig.fs.files[nr1Sidecar(basename)] = embed.MarshalSidecar(sidecar)
	rig.setupDone = len(rig.fs.writes)
}

// nr2Sidecar reads the note's sidecar.
func nr2Sidecar(t *testing.T, rig *nr1Rig, basename string) embed.Sidecar {
	t.Helper()

	sidecar, err := embed.UnmarshalSidecar(rig.fs.files[nr1Sidecar(basename)])
	if err != nil {
		t.Fatalf("the sidecar of %s does not parse: %v", basename, err)
	}

	return sidecar
}
