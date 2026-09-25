package cli_test

import (
	"errors"
	"fmt"
	"maps"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/vaultgraph"
)

// TestExcludePendingOffers covers the query-time filter: a pending note is
// dropped from the returned slice and anyPending reports true; an
// all-normal vault returns every note with anyPending false; a note whose
// read fails is kept (fail-open, never silently drop a note query would
// otherwise return over an unrelated read glitch).
func TestExcludePendingOffers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// vaultgraph.Note.Basename never carries the .md suffix (ParseBasename
	// strips it) — the fake read below is keyed with pathOf's "+.md" shape
	// to catch exactly this mismatch (a prior version of the implementation
	// under test joined vaultPath+Basename directly, without pathOf,
	// silently fail-opening on every note).
	notes := []vaultgraph.Note{
		{Basename: "1.2026-01-01.pending"},
		{Basename: "2.2026-01-02.normal"},
		{Basename: "3.2026-01-03.unreadable"},
	}

	read := func(path string) ([]byte, error) {
		switch path {
		case "/vault/1.2026-01-01.pending.md":
			return []byte(pendingFactNote), nil
		case "/vault/2.2026-01-02.normal.md":
			return []byte(normalFactNote), nil
		default:
			return nil, errUnreachableRead
		}
	}

	kept, anyPending := cli.ExportExcludePendingOffers(notes, "/vault", read)

	g.Expect(anyPending).To(BeTrue())
	g.Expect(kept).To(HaveLen(2), "the pending note is dropped, the unreadable one is kept fail-open")

	basenames := make([]string, 0, len(kept))
	for _, n := range kept {
		basenames = append(basenames, n.Basename)
	}

	g.Expect(basenames).To(ConsistOf("2.2026-01-02.normal", "3.2026-01-03.unreadable"))
}

// TestExcludePendingOffers_NoneReportsFalse covers the all-normal case: no
// pending notes means every note survives and anyPending is false.
func TestExcludePendingOffers_NoneReportsFalse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notes := []vaultgraph.Note{{Basename: "2.2026-01-02.normal"}}

	var readPath string

	read := func(path string) ([]byte, error) {
		readPath = path

		return []byte(normalFactNote), nil
	}

	kept, anyPending := cli.ExportExcludePendingOffers(notes, "/vault", read)

	g.Expect(readPath).To(Equal("/vault/2.2026-01-02.normal.md"), "must join pathOf(Basename), not Basename directly")

	g.Expect(anyPending).To(BeFalse())
	g.Expect(kept).To(HaveLen(1))
}

// TestExcludePendingOffers_SkillRunbookNote proves excludePendingOffers (and
// therefore engram query's exclusion + pending_offers flag, since both route
// through noteHasPendingMarker) drops a pending skill-runbook note exactly
// like a pending fact/feedback note, while an ordinary runbook survives.
func TestExcludePendingOffers_SkillRunbookNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "4.2026-01-04.skill-demo"},
		{Basename: "7.2026-01-07.normal-runbook"},
	}

	read := func(path string) ([]byte, error) {
		switch path {
		case "/vault/4.2026-01-04.skill-demo.md":
			return []byte(pendingSkillRunbookNote), nil
		case "/vault/7.2026-01-07.normal-runbook.md":
			return []byte(nonPendingSkillRunbookNote), nil
		default:
			return nil, errUnreachableRead
		}
	}

	kept, anyPending := cli.ExportExcludePendingOffers(notes, "/vault", read)

	g.Expect(anyPending).To(BeTrue())
	g.Expect(kept).To(HaveLen(1))
	g.Expect(kept[0].Basename).To(Equal("7.2026-01-07.normal-runbook"))
}

// TestNoteHasPendingMarker covers noteHasPendingMarker: a pending fact/
// feedback note is flagged, a normal one isn't, and a non-fact/feedback
// note (e.g. a vocab definition) is never flagged even if its raw bytes
// happen to be unparseable as fact/feedback frontmatter.
func TestNoteHasPendingMarker(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(cli.ExportNoteHasPendingMarker([]byte(pendingFactNote))).To(BeTrue())
	g.Expect(cli.ExportNoteHasPendingMarker([]byte(pendingFeedbackNote))).To(BeTrue())
	g.Expect(cli.ExportNoteHasPendingMarker([]byte(normalFactNote))).To(BeFalse())
	g.Expect(cli.ExportNoteHasPendingMarker([]byte(vocabDefNoteWithPendingLikeContent))).To(BeFalse())
	g.Expect(cli.ExportNoteHasPendingMarker([]byte("not frontmatter at all"))).To(BeFalse())
}

// TestNoteHasPendingMarker_SkillRunbook covers the skill-runbook-registration
// extension (vault-offer-curation ADDED requirement "The pending-offer marker
// SHALL apply to skill runbook notes"): a runbook note is a pending offer
// only when it carries BOTH `pending: true` and a non-empty `skill_hash`. A
// runbook with `pending: true` but no `skill_hash` is unchanged behavior —
// not a pending offer — and neither is a `skill_hash`-carrying runbook that
// isn't pending.
func TestNoteHasPendingMarker_SkillRunbook(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(cli.ExportNoteHasPendingMarker([]byte(pendingSkillRunbookNote))).To(BeTrue())
	g.Expect(cli.ExportNoteHasPendingMarker([]byte(pendingRunbookNoSkillHashNote))).To(BeFalse())
	g.Expect(cli.ExportNoteHasPendingMarker([]byte(nonPendingSkillRunbookNote))).To(BeFalse())
}

// TestVaultHasPendingOffers covers the engram-update detector: a vault
// holding one pending note flags true, an all-normal vault flags false, and
// a missing vault directory self-silences to false (same convention as
// notesMissingIdentityFields).
func TestVaultHasPendingOffers(t *testing.T) {
	t.Parallel()

	table := []struct {
		name  string
		files map[string][]byte
		want  bool
	}{
		{
			name:  "pending note flags the vault",
			files: map[string][]byte{"/vault/1.2026-01-01.a.md": []byte(pendingFactNote)},
			want:  true,
		},
		{
			name:  "normal note does not flag",
			files: map[string][]byte{"/vault/2.2026-01-02.a.md": []byte(normalFactNote)},
			want:  false,
		},
		{
			name:  "missing vault dir self-silences",
			files: map[string][]byte{},
			want:  false,
		},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fileSystem := newU1FS()
			maps.Copy(fileSystem.files, tc.files)

			got := cli.ExportVaultHasPendingOffers("/vault", fileSystem)
			g.Expect(got).To(Equal(tc.want))
		})
	}
}

// TestWarnIfPendingOffers covers the write-path log-only nudge: fires when
// the vault holds a pending offer, silent otherwise, and a no-op when any
// dep is nil (mirrors checkAndPersistVocabRefitTrigger's nil-tolerance).
func TestWarnIfPendingOffers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// listMD returns full .md filenames (ADR-0021 convention); read is
	// joined against vault by the caller, so the fake is keyed by the full
	// vault-joined path, not the bare filename.
	files := map[string][]byte{"/vault/1.2026-01-01.a.md": []byte(pendingFactNote)}
	listMD := func(string) ([]string, error) { return []string{"1.2026-01-01.a.md"}, nil }
	read := func(path string) ([]byte, error) { return files[path], nil }

	var warnings []string

	logWarn := func(format string, args ...any) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}

	cli.ExportWarnIfPendingOffers("/vault", listMD, read, logWarn)
	g.Expect(warnings).To(HaveLen(1))

	if len(warnings) != 1 {
		return
	}

	// The nudge carries the curate runbook's trigger query, not a skill name.
	g.Expect(warnings[0]).To(ContainSubstring(`engram query --text "curate pending offers"`))
	g.Expect(warnings[0]).To(ContainSubstring("--phrase"))
	g.Expect(warnings[0]).NotTo(ContainSubstring("skill"))
	g.Expect(warnings[0]).To(ContainSubstring(cli.ExportPendingOfferCurateInstruction))

	warnings = nil
	files["/vault/1.2026-01-01.a.md"] = []byte(normalFactNote)

	cli.ExportWarnIfPendingOffers("/vault", listMD, read, logWarn)
	g.Expect(warnings).To(BeEmpty())

	// nil deps: must not panic.
	cli.ExportWarnIfPendingOffers("/vault", nil, read, logWarn)
}

// unexported constants.
const (
	nonPendingSkillRunbookNote = "---\ntype: runbook\nsituation: s\ndone_when: d\n" +
		"luhmann: \"6\"\ncreated: 2026-01-06\nsource: agent\nuser: u\nvault: personal\n" +
		"skill_hash: abc123\n---\n\n1. step\n"
	normalFactNote = "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\n" +
		"luhmann: \"2\"\ncreated: 2026-01-02\nsource: agent\nuser: u\nvault: personal\n---\n\nbody\n"
	pendingFactNote = "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\n" +
		"luhmann: \"1\"\ncreated: 2026-01-01\nsource: agent\nuser: u\nvault: personal\npending: true\n---\n\nbody\n"
	pendingFeedbackNote = "---\ntype: feedback\nsituation: s\nbehavior: b\nimpact: i\naction: act\n" +
		"luhmann: \"3\"\ncreated: 2026-01-03\nsource: agent\nuser: u\nvault: personal\npending: true\n---\n\nbody\n"
	pendingRunbookNoSkillHashNote = "---\ntype: runbook\nsituation: s\ndone_when: d\n" +
		"luhmann: \"5\"\ncreated: 2026-01-05\nsource: agent\nuser: u\nvault: personal\n" +
		"pending: true\n---\n\n1. step\n"
	pendingSkillRunbookNote = "---\ntype: runbook\nsituation: n/a\ndone_when: n/a\n" +
		"luhmann: \"4\"\ncreated: 2026-01-04\nsource: agent\nuser: u\nvault: personal\n" +
		"skill_hash: abc123\npending: true\n---\n\nSKILL body\n"
	vocabDefNoteWithPendingLikeContent = "---\ntype: term\nterm: recall\ndescription: recall the vault\n---\n\nbody\n"
)

// unexported variables.
var (
	errUnreachableRead = errors.New("offer_test: unexpected read")
)
