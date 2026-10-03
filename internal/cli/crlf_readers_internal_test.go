package cli

// Non-exchange readers read a CRLF note as its LF form (#789, design D8):
// query project filtering, recency's created:, query's vault metadata
// (tags, supersedes, triggers), vocab tagging, engram check and count.

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/vaultgraph"
)

func TestCapRedFlagsForPreview_CRLFNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var flags strings.Builder
	for index := range 40 {
		fmt.Fprintf(&flags, "    - a red flag long enough to push the preview over its budget, number %c\n",
			rune('a'+index%26))
	}

	lf := "---\ntype: runbook\nsituation: s\ndone_when: d\nred_flags:\n" + flags.String() +
		"luhmann: \"1\"\n---\n\n1. step\n"

	capped := capRedFlagsForPreview(strings.ReplaceAll(lf, "\n", "\r\n"), "1.2026-01-01.rb")
	g.Expect(capped).To(ContainSubstring("EARLIER RED_FLAGS OMITTED"))
	g.Expect(capped).To(Equal(capRedFlagsForPreview(lf, "1.2026-01-01.rb")))
}

func TestCheckSituationPresence_CRLFNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// A CRLF fact with an empty situation is caught, not skipped as if it
	// had no frontmatter.
	note := strings.Replace(crlfReaderNote(""), "situation: when reading CRLF", "situation: \"\"", 1)

	var out bytes.Buffer

	failed := checkSituationPresence([]vaultgraph.Note{{Basename: "1.2026-01-01.a"}},
		func(string) ([]byte, error) { return []byte(note), nil }, "/vault", &out)
	g.Expect(failed).To(BeTrue(), out.String())
	g.Expect(out.String()).To(ContainSubstring("1.2026-01-01.a"))
}

func TestItemMatchesProject_CRLFNote(t *testing.T) {
	t.Parallel()
	NewWithT(t).Expect(itemMatchesProject(resolvedItem{content: crlfReaderNote("")}, "widgets")).To(BeTrue())
}

func TestKindFromContent_CRLFNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(kindFromContent(crlfReaderNote(""))).To(Equal("fact"), "the kind label carries no \\r")
	g.Expect(isQueryExcludedKind(strings.Replace(crlfReaderNote(""), "type: fact", "type: qa-question", 1))).
		To(BeTrue(), "a CRLF qa-question note is excluded from the main set")
}

func TestParseCreatedFromNote_CRLFNote(t *testing.T) {
	t.Parallel()
	NewWithT(t).Expect(parseCreatedFromNote([]byte(crlfReaderNote("")))).To(Equal("2026-01-01"))
}

func TestParseNoteQueryFrontmatter_CRLFNote(t *testing.T) {
	t.Parallel()

	parsed := parseNoteQueryFrontmatter(crlfReaderNote("tags:\n    - vocab/widgets\ntriggers:\n    - turn it\n"))
	g := NewWithT(t)
	g.Expect(parsed.Tags).To(Equal([]string{"vocab/widgets"}))
	g.Expect(parsed.Triggers).To(Equal([]string{"turn it"}))
}

func TestReadNoteAttrs_CRLFNote(t *testing.T) {
	t.Parallel()

	attrs, ok := readNoteAttrs(func(string) ([]byte, error) { return []byte(crlfReaderNote("")), nil }, "/vault/a.md")
	g := NewWithT(t)
	g.Expect(ok).To(BeTrue())
	g.Expect(attrs).To(HaveKeyWithValue("type", "fact"))
}

func TestTriggerIndex_CRLFRunbook(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runbook := strings.ReplaceAll("---\ntype: runbook\nsituation: s\ndone_when: d\ntriggers:\n    - rotate the keys\n"+
		"luhmann: \"1\"\n---\n\n1. step\n", "\n", "\r\n")

	meta := loadAllVaultNotesMeta([]compatibleSidecar{{note: vaultgraph.Note{Basename: "1.2026-01-01.rb"}}}, "/vault",
		func(string) ([]byte, error) { return []byte(runbook), nil })
	g.Expect(meta.TriggerIndex).To(HaveKeyWithValue("1.2026-01-01.rb", []string{"rotate the keys"}))
	g.Expect(matchTriggers("please rotate the keys today", meta.TriggerIndex)).To(Equal([]string{"1.2026-01-01.rb"}))
}

func TestVocab_CRLFNote(t *testing.T) {
	t.Parallel()

	t.Run("assignment writes tags as LF", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		updated := WriteVocabAssignment(crlfReaderNote(""), []string{"widgets"})
		g.Expect(updated).To(ContainSubstring("tags:\n    - vocab/widgets\n"))
		g.Expect(updated).NotTo(ContainSubstring("\r\n"))
	})

	t.Run("definition note is recognised", func(t *testing.T) {
		t.Parallel()
		NewWithT(t).Expect(isVocabDefinitionNote(crlfReaderNote("tags:\n    - vocab\n"))).To(BeTrue())
	})

	t.Run("an unchanged CRLF note is not written", func(t *testing.T) {
		t.Parallel()

		note := crlfReaderNote("tags:\n    - vocab/widgets\n")
		NewWithT(t).Expect(vocabAssignmentUnchanged(note, WriteVocabAssignment(note, []string{"widgets"}))).
			To(BeTrue(), "an assignment that changes no tag must not write a note just to convert it")
	})
}

// crlfReaderNote is a fact note with extra frontmatter lines, all CRLF.
func crlfReaderNote(extra string) string {
	lf := "---\ntype: fact\nsituation: when reading CRLF\nsubject: a\npredicate: b\nobject: c\n" +
		"luhmann: \"1\"\ncreated: 2026-01-01\nsource: test\nproject: widgets\n" + extra +
		"---\n\nInformation learned: when reading CRLF, a b c.\n"

	return strings.ReplaceAll(lf, "\n", "\r\n")
}
