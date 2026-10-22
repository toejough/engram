package cli_test

import (
	"bytes"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792a9a1a_Targets_CheckACleanVaultPrintsThreePassLines verifies 792a9a1a (vault-health.md): a vault
// whose links resolve, whose fact notes all name a situation and whose sidecars are current prints a PASS
// line for G0, M5 and S1 and asks for no failing exit status.
func Test792a9a1a_Targets_CheckACleanVaultPrintsThreePassLines(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose links resolve, whose fact notes all have a situation, and whose sidecars are current
	out := vh1Check(t, vh1WithSidecars(map[string]string{
		"1.2026-05-30.alpha.md": vic1NoteText(vic1FactType, "situation: checking a clean vault",
			"body\n\nRelated to:\n- [[2.2026-05-30.beta]] — x.\n"),
		"2.2026-05-30.beta.md": vic1NoteText(vh1FeedbackType, "situation: reading the clean report", "body\n"),
	}))

	// Then stdout has a PASS line for G0, M5 and S1, and the exit status is zero
	g.Expect(out.stdout).To(Equal(vic1G0PassLine + vic1M5PassLine + vic1S1PassLine))
	g.Expect(out.stderr).To(BeEmpty())
	g.Expect(out.exits).To(BeEmpty())
}

// Test792a9a1b_Targets_CheckAFailLineFailsTheCommand verifies 792a9a1b (vault-health.md): a bare-ID link
// that should resolve gives a FAIL line naming G0, the failure message on stderr and a non-zero status.
func Test792a9a1b_Targets_CheckAFailLineFailsTheCommand(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault with a note whose link is a bare ID that should resolve
	out := vh1Check(t, map[string]string{
		"1.2026-05-30.foo.md": vic1NoteText(vic1FactType, "situation: x", "body\n\n- [[2]] — x.\n"),
		"2.2026-05-30.bar.md": vic1NoteText(vic1FactType, "situation: x", "body\n"),
	})

	// Then a line beginning FAIL names G0 graph-resolution, stderr has the failure message, and the exit
	// status is non-zero
	failLines := vh1LinesWithPrefix(out.stdout, "FAIL")

	g.Expect(failLines).To(ContainElement(HavePrefix("FAIL  G0 graph-resolution")))
	g.Expect(out.stderr).To(ContainSubstring("check: FAIL-class invariant violations found"))
	g.Expect(out.exits).To(Equal([]int{1}))
}

// Test792a9a1c_Targets_CheckAWarnLineDoesNotFailTheCommand verifies 792a9a1c (vault-health.md): a vault
// whose only finding is a link to a missing note prints a WARN line naming G3 and exits zero.
func Test792a9a1c_Targets_CheckAWarnLineDoesNotFailTheCommand(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose only finding is a link to a note that does not exist
	out := vh1Check(t, vh1WithSidecars(map[string]string{
		vic1NoteFile: vic1NoteText(vic1FactType, "situation: x", "body\n\n- [[ghost]] — x.\n"),
	}))

	// Then a line beginning WARN names G3 dangling, and the exit status is zero
	g.Expect(vh1LinesWithPrefix(out.stdout, "WARN")).To(ConsistOf(
		"WARN  G3 dangling: 1 authored links target no note"))
	g.Expect(vh1LinesWithPrefix(out.stdout, "FAIL")).To(BeEmpty())
	g.Expect(out.stderr).To(BeEmpty())
	g.Expect(out.exits).To(BeEmpty())
}

// Test792a9a1d_Targets_CheckOffendersAreCappedAtTen verifies 792a9a1d (vault-health.md): twelve links that
// target no note are reported by ten listed offenders followed by the line "… and 2 more".
func Test792a9a1d_Targets_CheckOffendersAreCappedAtTen(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault in which twelve links target no note
	const danglingCount = 12

	var body strings.Builder

	body.WriteString("body\n\n")

	for i := 1; i <= danglingCount; i++ {
		body.WriteString("- [[ghost-" + strconv.Itoa(i) + "]] — x.\n")
	}

	out := vh1Check(t, vh1WithSidecars(map[string]string{
		vic1NoteFile: vic1NoteText(vic1FactType, "situation: x", body.String()),
	}))

	// Then ten of them are listed, followed by the line "… and 2 more"
	lines := strings.Split(out.stdout, "\n")
	listed := vh1LinesContaining(lines, " → [[")

	g.Expect(lines).To(ContainElement("WARN  G3 dangling: 12 authored links target no note"))
	g.Expect(listed).To(HaveLen(vh1ListedCap))

	if len(listed) != vh1ListedCap {
		return
	}

	afterLast := slices.Index(lines, listed[vh1ListedCap-1]) + 1

	g.Expect(afterLast).To(BeNumerically("<", len(lines)))

	if afterLast >= len(lines) {
		return
	}

	g.Expect(lines[afterLast]).To(Equal("        … and 2 more"))
}

// Test792a9a1e_Targets_CheckAnUnreadableNoteStopsTheCheck verifies 792a9a1e (vault-health.md): a note the
// user cannot read fails the command with "check: scan:" naming the file, and no report line is printed.
func Test792a9a1e_Targets_CheckAnUnreadableNoteStopsTheCheck(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault with a note the user cannot read
	const lockedFile = "2.2026-05-30.locked.md"

	out := vh1Check(t, map[string]string{
		vic1NoteFile: vic1NoteText(vic1FactType, "situation: x", "body\n"),
		lockedFile:   vic1NoteText(vic1FactType, "situation: x", "body\n"),
	}, lockedFile)

	// Then it fails with an error beginning "check: scan:" and naming the file, and prints no PASS, WARN or
	// FAIL line
	g.Expect(out.stderr).To(HavePrefix("check: scan:"))
	g.Expect(out.stderr).To(ContainSubstring(lockedFile))
	g.Expect(out.exits).To(Equal([]int{1}))
	g.Expect(out.stdout).NotTo(ContainSubstring("PASS"))
	g.Expect(out.stdout).NotTo(ContainSubstring("WARN"))
	g.Expect(out.stdout).NotTo(ContainSubstring("FAIL"))
}

// Test792a9a1g_Targets_CheckANoteThatIsNotValidUTF8IsStillChecked verifies 792a9a1g (vault-health.md): a
// note with invalid UTF-8 bytes in its body that links a missing note has the dangling link reported, as
// the same note without the invalid bytes does.
func Test792a9a1g_Targets_CheckANoteThatIsNotValidUTF8IsStillChecked(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a note with invalid UTF-8 bytes in its body that links [[ghost]], and no such note
	const (
		invalidBytes = "\xff\xfe\xc3\x28"
		linkLines    = "\n\nRelated to:\n- [[ghost]] — x.\n"
	)

	withInvalid := vh1Check(t, vh1WithSidecars(map[string]string{
		vic1NoteFile: vic1NoteText(vic1FactType, "situation: x", "body "+invalidBytes+" before the link"+linkLines),
	}))
	without := vh1Check(t, vh1WithSidecars(map[string]string{
		vic1NoteFile: vic1NoteText(vic1FactType, "situation: x", "body before the link"+linkLines),
	}))

	// Then the dangling link is reported, as it would be without the invalid bytes
	const danglingReport = "WARN  G3 dangling: 1 authored links target no note\n" +
		"        " + vic1NoteBasename + " → [[ghost]]\n"

	g.Expect(withInvalid.stdout).To(ContainSubstring(danglingReport))
	g.Expect(withInvalid.stdout).To(Equal(without.stdout))
	g.Expect(withInvalid.exits).To(BeEmpty())
}

// Test792a9b1a_Targets_CheckABareIDLinkFails verifies 792a9b1a (vault-health.md): a note linking [[2]] when
// a note 2.2026-05-30.bar exists fails G0 with the count line and the source and link listed.
func Test792a9b1a_Targets_CheckABareIDLinkFails(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a note 1.2026-05-30.foo that links [[2]], and a note 2.2026-05-30.bar
	out := vh1Check(t, map[string]string{
		"1.2026-05-30.foo.md": vic1NoteText(vic1FactType, "situation: x", "body\n\n- [[2]] — x.\n"),
		"2.2026-05-30.bar.md": vic1NoteText(vic1FactType, "situation: x", "body\n"),
	})

	// Then stdout has the FAIL G0 line and the line "1.2026-05-30.foo → [[2]]"
	g.Expect(out.stdout).To(ContainSubstring(
		"FAIL  G0 graph-resolution: 1 bare-id links that should resolve\n        1.2026-05-30.foo → [[2]]\n"))
}

// Test792a9b1b_Targets_CheckABasenameLinkPasses verifies 792a9b1b (vault-health.md): a note A linking [[B]]
// with a note B present prints the G0 PASS line.
func Test792a9b1b_Targets_CheckABasenameLinkPasses(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a note A that links [[B]], and a note B
	out := vh1Check(t, map[string]string{
		"A.md": vic1NoteText(vic1FactType, "situation: x", "body\n\n- [[B]] — x.\n"),
		"B.md": vic1NoteText(vic1FactType, "situation: x", "body\n"),
	})

	// Then stdout has "PASS  G0 graph-resolution: every authored link resolves by form"
	g.Expect(out.stdout).To(HavePrefix(vic1G0PassLine))
	g.Expect(out.stderr).To(BeEmpty())
}

// Test792a9b1c_Targets_CheckALegacySuffixedLinkResolves verifies 792a9b1c (vault-health.md): a link
// "Supersedes: [[B.md]]" to a note B is reported neither under G0 nor as dangling under G3.
func Test792a9b1c_Targets_CheckALegacySuffixedLinkResolves(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a note body containing "Supersedes: [[B.md]]" and a note B
	out := vh1Check(t, map[string]string{
		"A.md": vic1NoteText(vic1FactType, "situation: x", "body\n\nSupersedes: [[B.md]]\n"),
		"B.md": vic1NoteText(vic1FactType, "situation: x", "body\n"),
	})

	// Then the link is not reported, neither under G0 nor as dangling under G3
	g.Expect(out.stdout).To(HavePrefix(vic1G0PassLine))
	g.Expect(out.stdout).NotTo(ContainSubstring("G3"))
	g.Expect(out.stdout).NotTo(ContainSubstring("→ [["))
	g.Expect(out.exits).To(BeEmpty())
}

// Test792a9c1a_Targets_CheckALinkToAMissingNoteWarns verifies 792a9c1a (vault-health.md): a fact note
// linking [[999]] when no note has ID 999 gets a WARN G3 line listing the link, G0 still passes, and the
// exit status is zero.
func Test792a9c1a_Targets_CheckALinkToAMissingNoteWarns(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a fact note that links [[999]] and no note has the ID 999
	out := vh1Check(t, map[string]string{
		vic1NoteFile: vic1NoteText(vic1FactType, "situation: x", "body\n\n- [[999]] — x.\n"),
	})

	// Then stdout has a WARN G3 dangling line listing the link, G0 still passes, and the exit status is zero
	g.Expect(out.stdout).To(HavePrefix(vic1G0PassLine +
		"WARN  G3 dangling: 1 authored links target no note\n        " + vic1NoteBasename + " → [[999]]\n"))
	g.Expect(out.exits).To(BeEmpty())
}

// Test792a9c1b_Targets_CheckAMissingNoteIsReportedOnceHoweverItIsSpelled verifies 792a9c1b
// (vault-health.md): links [[vocab.X]] and [[vocab.X.md]] to a note that does not exist are counted and
// listed once.
func Test792a9c1b_Targets_CheckAMissingNoteIsReportedOnceHoweverItIsSpelled(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a note that links [[vocab.X]] and [[vocab.X.md]], and no such note
	out := vh1Check(t, map[string]string{
		vic1NoteFile: vic1NoteText(vic1FactType, "situation: x", "body\n\n- [[vocab.X]] — x.\n- [[vocab.X.md]] — x.\n"),
	})

	// Then stdout has "WARN  G3 dangling: 1 authored links target no note"
	g.Expect(out.stdout).To(ContainSubstring("WARN  G3 dangling: 1 authored links target no note\n"))
	g.Expect(vh1LinesContaining(strings.Split(out.stdout, "\n"), " → [[")).To(ConsistOf(
		"        " + vic1NoteBasename + " → [[vocab.X]]"))
}

// Test792a9c1c_Targets_CheckANotesLinkToItselfIsNotReported verifies 792a9c1c (vault-health.md): a note
// linking its own basename produces no G3 line.
func Test792a9c1c_Targets_CheckANotesLinkToItselfIsNotReported(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a note that links its own basename
	out := vh1Check(t, map[string]string{
		vic1NoteFile: vic1NoteText(vic1FactType, "situation: x", "body\n\n- [["+vic1NoteBasename+"]] — me.\n"),
	})

	// Then no G3 line is printed
	g.Expect(out.stdout).To(HavePrefix(vic1G0PassLine))
	g.Expect(out.stdout).NotTo(ContainSubstring("G3"))
	g.Expect(out.exits).To(BeEmpty())
}

// Test792a9c1d_Targets_CheckALinkInsideAFencedCodeBlockIsNotReported verifies 792a9c1d (vault-health.md):
// a note whose only link to a missing note sits inside a fenced code block produces no G3 line.
func Test792a9c1d_Targets_CheckALinkInsideAFencedCodeBlockIsNotReported(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a note whose only link to a missing note is inside a fenced code block
	out := vh1Check(t, map[string]string{
		vic1NoteFile: vic1NoteText(vic1FactType, "situation: x", "body\n\n```\nsee [[ghost]] in the transcript\n```\n"),
	})

	// Then no G3 line is printed
	g.Expect(out.stdout).To(HavePrefix(vic1G0PassLine))
	g.Expect(out.stdout).NotTo(ContainSubstring("G3"))
	g.Expect(out.exits).To(BeEmpty())
}

// Test792a9c1e_Targets_CheckAVaultWithNoDanglingLinkPrintsNoG3Line verifies 792a9c1e (vault-health.md): a
// vault whose links all resolve has no line mentioning G3.
func Test792a9c1e_Targets_CheckAVaultWithNoDanglingLinkPrintsNoG3Line(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose links all resolve
	out := vh1Check(t, map[string]string{
		"A.md": vic1NoteText(vic1FactType, "situation: x", "body\n\n- [[B]] — x.\n"),
		"B.md": vic1NoteText(vic1FactType, "situation: x", "body\n\n- [[A]] — x.\n"),
	})

	// Then stdout has no line mentioning G3
	g.Expect(out.stdout).To(HavePrefix(vic1G0PassLine))
	g.Expect(out.stdout).NotTo(ContainSubstring("G3"))
}

// Test792a9d1a_Targets_CheckAFactWithoutASituationFails verifies 792a9d1a (vault-health.md): a fact note
// whose frontmatter has no situation key gets the M5 FAIL line naming the note and a non-zero status.
func Test792a9d1a_Targets_CheckAFactWithoutASituationFails(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a fact note whose frontmatter has no situation key
	out := vh1Check(t, map[string]string{
		vic1NoteFile: vic1NoteText(vic1FactType, "subject: x", "body\n"),
	})

	// Then stdout has the M5 FAIL line naming the note, and the exit status is non-zero
	g.Expect(out.stdout).To(ContainSubstring(
		"FAIL  M5 situation-presence: 1 note(s) missing a situation\n        " + vic1NoteBasename + "\n"))
	g.Expect(out.exits).To(Equal([]int{1}))
}

// Test792a9d1b_Targets_CheckABlankSituationFails verifies 792a9d1b (vault-health.md): a feedback note whose
// situation is only spaces fails M5, naming the note.
func Test792a9d1b_Targets_CheckABlankSituationFails(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a feedback note whose situation is only spaces
	out := vh1Check(t, map[string]string{
		vic1NoteFile: vic1NoteText(vh1FeedbackType, `situation: "   "`, "body\n"),
	})

	// Then M5 fails naming the note
	g.Expect(out.stdout).To(ContainSubstring(
		"FAIL  M5 situation-presence: 1 note(s) missing a situation\n        " + vic1NoteBasename + "\n"))
	g.Expect(out.exits).To(Equal([]int{1}))
}

// Test792a9d1c_Targets_CheckEveryPresentSituationPasses verifies 792a9d1c (vault-health.md): fact and
// feedback notes that each name a situation give the M5 PASS line.
func Test792a9d1c_Targets_CheckEveryPresentSituationPasses(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given fact and feedback notes that each name a situation
	out := vh1Check(t, map[string]string{
		"1.2026-05-30.fact.md":     vic1NoteText(vic1FactType, "situation: judging a fact", "body\n"),
		"2.2026-05-30.feedback.md": vic1NoteText(vh1FeedbackType, "situation: judging feedback", "body\n"),
	})

	// Then stdout has "PASS  M5 situation-presence: every fact/feedback names a situation"
	g.Expect(out.stdout).To(ContainSubstring(vic1M5PassLine))
}

// Test792a9d1d_Targets_CheckOtherNoteTypesAreNotJudged verifies 792a9d1d (vault-health.md): a note of type
// moc and a note of type runbook, neither with a situation, leave M5 passing.
func Test792a9d1d_Targets_CheckOtherNoteTypesAreNotJudged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a note of type moc and a note of type runbook, neither with a situation
	out := vh1Check(t, map[string]string{
		"1.2026-05-30.map.md": vic1NoteText("moc", "title: a map", "map of content\n"),
		"2.2026-05-30.how.md": vic1NoteText("runbook", "title: a procedure", "steps\n"),
	})

	// Then M5 passes
	g.Expect(out.stdout).To(ContainSubstring(vic1M5PassLine))
	g.Expect(out.stdout).NotTo(ContainSubstring("FAIL"))
}

// Test792a9d1e_Targets_CheckANoteWithoutUsableFrontmatterIsNotJudged verifies 792a9d1e (vault-health.md):
// a note with no frontmatter and a note whose frontmatter is malformed YAML leave M5 passing.
func Test792a9d1e_Targets_CheckANoteWithoutUsableFrontmatterIsNotJudged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a note with no frontmatter and a note whose frontmatter is malformed YAML
	out := vh1Check(t, map[string]string{
		"1.2026-05-30.raw.md": "just prose, no frontmatter\n",
		"2.2026-05-30.bad.md": "---\ntype: fact\nsituation: [unterminated\n---\nbody\n",
	})

	// Then M5 passes
	g.Expect(out.stdout).To(ContainSubstring(vic1M5PassLine))
	g.Expect(out.stdout).NotTo(ContainSubstring("FAIL"))
}

// Test792a9d1f_Targets_CheckACRLFNoteIsJudged verifies 792a9d1f (vault-health.md): a fact note whose lines
// end in \r\n and whose situation is empty fails M5 naming the note, as the same note with LF endings does.
func Test792a9d1f_Targets_CheckACRLFNoteIsJudged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a fact note whose lines end in \r\n and whose situation is empty
	lfText := vic1NoteText(vic1FactType, `situation: ""`, "body\n")
	crlfText := strings.ReplaceAll(lfText, "\n", "\r\n")

	crlf := vh1Check(t, map[string]string{vic1NoteFile: crlfText})
	lf := vh1Check(t, map[string]string{vic1NoteFile: lfText})

	// Then M5 fails naming the note
	g.Expect(crlf.stdout).To(ContainSubstring(
		"FAIL  M5 situation-presence: 1 note(s) missing a situation\n        " + vic1NoteBasename + "\n"))
	g.Expect(crlf.stdout).To(Equal(lf.stdout))
	g.Expect(crlf.exits).To(Equal([]int{1}))
}

// Test792a9e1a_Targets_CheckAnOldSchemaSidecarFails verifies 792a9e1a (vault-health.md): a fact note whose
// sidecar has the old single-vector schema gets an S1 FAIL line naming the note and the force command, and
// a non-zero status.
func Test792a9e1a_Targets_CheckAnOldSchemaSidecarFails(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a fact note whose sidecar has the old single-vector schema
	out := vh1Check(t, map[string]string{
		vic1NoteFile: vic1NoteText(vic1FactType, "situation: x", "body\n"),
		vh1SidecarFile: `{"embedding_model_id":"m@4","dims":4,"vector":[1,0,0,0],` +
			`"content_hash":"sha256:x"}`,
	})

	// Then stdout has a FAIL line for S1 naming the note and the force command, and the exit status is
	// non-zero
	g.Expect(out.stdout).To(ContainSubstring(
		"FAIL  S1 sidecar-schema: 1 sidecar(s) on an old/invalid schema (run `engram embed apply --force`)\n" +
			"        " + vic1NoteBasename + "\n"))
	g.Expect(out.exits).To(Equal([]int{1}))
}

// Test792a9e1b_Targets_CheckAMalformedSidecarFails verifies 792a9e1b (vault-health.md): a note whose
// sidecar is not valid JSON fails S1, naming the note.
func Test792a9e1b_Targets_CheckAMalformedSidecarFails(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a note whose sidecar is not valid JSON
	out := vh1Check(t, map[string]string{
		vic1NoteFile:   vic1NoteText(vic1FactType, "situation: x", "body\n"),
		vh1SidecarFile: "{not json",
	})

	// Then S1 fails naming the note
	g.Expect(out.stdout).To(ContainSubstring("FAIL  S1 sidecar-schema: 1 sidecar(s) on an old/invalid schema"))
	g.Expect(out.stdout).To(ContainSubstring("        " + vic1NoteBasename + "\n"))
	g.Expect(out.exits).To(Equal([]int{1}))
}

// Test792a9e1c_Targets_CheckACurrentSidecarPasses verifies 792a9e1c (vault-health.md): a note with a
// current sidecar gets the S1 PASS line.
func Test792a9e1c_Targets_CheckACurrentSidecarPasses(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a note with a current sidecar
	out := vh1Check(t, map[string]string{
		vic1NoteFile:   vic1NoteText(vic1FactType, "situation: x", "body\n"),
		vh1SidecarFile: vh1CurrentSidecar(),
	})

	// Then stdout has "PASS  S1 sidecar-schema: every sidecar parses under the current schema"
	g.Expect(out.stdout).To(ContainSubstring(vic1S1PassLine))
	g.Expect(out.stdout).NotTo(ContainSubstring("WARN"))
	g.Expect(out.exits).To(BeEmpty())
}

// Test792a9e1d_Targets_CheckAMissingSidecarWarns verifies 792a9e1d (vault-health.md): notes with no
// sidecar and no invalid sidecar get an S1 WARN line counting them and naming the apply command, and the
// exit status is zero.
func Test792a9e1d_Targets_CheckAMissingSidecarWarns(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given notes with no sidecar and no invalid sidecar
	out := vh1Check(t, map[string]string{
		"1.2026-05-30.one.md": vic1NoteText(vic1FactType, "situation: x", "body\n"),
		"2.2026-05-30.two.md": vic1NoteText(vic1FactType, "situation: x", "body\n"),
	})

	// Then stdout has a WARN line for S1 counting the notes and naming `engram embed apply`, and the exit
	// status is zero
	g.Expect(vh1LinesWithPrefix(out.stdout, "WARN")).To(ConsistOf(
		"WARN  S1 sidecar-schema: 2 note(s) missing a sidecar (run `engram embed apply`)"))
	g.Expect(vh1LinesWithPrefix(out.stdout, "FAIL")).To(BeEmpty())
	g.Expect(out.stderr).To(BeEmpty())
	g.Expect(out.exits).To(BeEmpty())
}

// unexported constants.
const (
	vh1FeedbackType = "feedback"
	// vh1ListedCap is the number of offenders a report line lists before it counts the rest.
	vh1ListedCap   = 10
	vh1SidecarFile = vic1NoteBasename + ".vec.json"
)

// vh1Outcome is what one `engram check` printed and the exit codes it asked for.
type vh1Outcome struct {
	stdout string
	stderr string
	exits  []int
}

// vh1Check runs `engram check --vault v` through cli.Targets over an in-memory vault holding files (names
// relative to the vault directory). Each name in unreadable is a file whose read is denied. A zero exit
// status is the Exit hook never being called.
func vh1Check(t *testing.T, files map[string]string, unreadable ...string) vh1Outcome {
	t.Helper()

	edge := newVic1Vault(files).edgeFS()
	readFile := edge.readFile
	denied := make(map[string]struct{}, len(unreadable))

	for _, name := range unreadable {
		denied[filepath.Join(vic1VaultPath, name)] = struct{}{}
	}

	edge.readFile = func(path string) ([]byte, error) {
		_, blocked := denied[filepath.Clean(path)]
		if blocked {
			return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrPermission}
		}

		return readFile(path)
	}
	edge.stat = func(path string) (fs.FileInfo, error) {
		if filepath.Clean(path) == vic1VaultPath {
			return fakeFileInfo{dir: true}, nil
		}

		return nil, &fs.PathError{Op: "stat", Path: path, Err: fs.ErrNotExist}
	}

	var (
		stdout, stderr bytes.Buffer
		exits          []int
	)

	deps := newTestDeps(&stdout, &stderr)
	deps.FS = edge
	deps.DebugLog = nil
	deps.Getenv = func(string) string { return "" }
	deps.UserHomeDir = func() (string, error) { return vic1VaultPath, nil }
	deps.Now = func() time.Time { return time.Time{} }
	deps.Exit = func(code int) { exits = append(exits, code) }

	_, err := targ.Execute([]string{"engram", "check", "--vault", vic1VaultPath}, cli.Targets(deps)...)
	if err != nil {
		stderr.WriteString(err.Error() + "\n")
	}

	return vh1Outcome{stdout: stdout.String(), stderr: stderr.String(), exits: exits}
}

// vh1CurrentSidecar returns a sidecar under the current schema.
func vh1CurrentSidecar() string {
	const dims = 4

	return string(embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "m@4",
		Dims:             dims,
		SituationVector:  []float32{1, 0, 0, 0},
		BodyVector:       []float32{0, 1, 0, 0},
		ContentHash:      "sha256:x",
	}))
}

// vh1LinesContaining returns the lines that contain part.
func vh1LinesContaining(lines []string, part string) []string {
	return slices.DeleteFunc(slices.Clone(lines), func(line string) bool {
		return !strings.Contains(line, part)
	})
}

// vh1LinesWithPrefix returns the lines of text that begin with prefix.
func vh1LinesWithPrefix(text, prefix string) []string {
	return slices.DeleteFunc(strings.Split(text, "\n"), func(line string) bool {
		return !strings.HasPrefix(line, prefix)
	})
}

// vh1WithSidecars returns files plus a current sidecar beside every note in it.
func vh1WithSidecars(files map[string]string) map[string]string {
	withSidecars := make(map[string]string, len(files)*2)

	for name, content := range files {
		withSidecars[name] = content
		withSidecars[strings.TrimSuffix(name, ".md")+".vec.json"] = vh1CurrentSidecar()
	}

	return withSidecars
}
