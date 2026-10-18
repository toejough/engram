package auto_test

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
)

func Test793d4p1a_ActivityLines_ARecordThatWasNotToldBecomesALine(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lines := auto.ActivityLines([]auto.Record{
		recordAt(0, auto.RecordWritten, func(r *auto.Record) { r.Note = "2.2026-10-17.zorblax-backup.md" }),
		recordAt(1, auto.RecordSkipped, func(r *auto.Record) {
			r.Slug, r.Reason = "zorblax-build", "the vault already states it: 1.2026-10-17.zorblax-build.md"
		}),
		recordAt(2, auto.RecordSkipped, func(r *auto.Record) { r.Reason = "the reply is not a list of lessons" }),
		recordAt(3, auto.RecordFailed, func(r *auto.Record) { r.Reason = "learn: writing x: permission denied" }),
	}, nil)

	g.Expect(lines).To(Equal([]string{
		"20:32:25 wrote the note 2.2026-10-17.zorblax-backup.md",
		"20:32:26 left out the lesson zorblax-build: the vault already states it: 1.2026-10-17.zorblax-build.md",
		"20:32:27 left out: the reply is not a list of lessons",
		"20:32:28 failed: learn: writing x: permission denied",
	}))
}

func Test793d4p1b_ActivityLines_ARecordOfTheKindNothingMakesNoLine(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(auto.ActivityLines([]auto.Record{recordAt(0, auto.RecordNothing, nil)}, nil)).To(BeEmpty())
}

func Test793d4p1c_ActivityLines_ALineTheSessionWasToldIsNotToldAgain(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	records := []auto.Record{
		recordAt(0, auto.RecordWritten, func(r *auto.Record) { r.Note = "a.md" }),
		recordAt(1, auto.RecordWritten, func(r *auto.Record) { r.Note = "b.md" }),
	}

	g.Expect(auto.ActivityLines(records, []string{"20:32:25 wrote the note a.md"})).To(Equal([]string{
		"20:32:26 wrote the note b.md",
	}))
}

func Test793d4p1d_ActivityLines_AReasonIsOneLineRedactedAndCutInItsMiddle(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// the credential is built here, as the tests of the redaction build theirs; it begins before the place where the
	// reason is cut and ends after it, so that a reason cut before it is redacted leaves part of the credential. The
	// sentence names no secret, so that the redaction does not take the cause for a value
	token := "ghp" + "_" + strings.Repeat("a1", 18)
	reason := "learn: writing " + strings.Repeat("x", 80) + "\nthe value " + token + " leaked\n" +
		strings.Repeat("y", 150) + ": permission denied"

	lines := auto.ActivityLines([]auto.Record{
		recordAt(0, auto.RecordFailed, func(r *auto.Record) { r.Reason = reason }),
	}, nil)

	g.Expect(lines).To(HaveLen(1))

	if len(lines) != 1 {
		return
	}

	line := lines[0]
	shown := strings.TrimPrefix(line, "20:32:25 failed: ")

	g.Expect(line).NotTo(ContainSubstring("\n"), "the line is one line")
	g.Expect(line).NotTo(ContainSubstring("ghp_"), "the credential, or part of it")
	g.Expect(utf8.RuneCountInString(shown)).To(BeNumerically("<=", 240), "the reason as shown")
	g.Expect(shown).To(HavePrefix("learn: writing xxx"), "the beginning of the reason")
	g.Expect(shown).To(HaveSuffix("yyy: permission denied"), "the end of the reason, which holds the cause")
}

func Test793d4p1e_ActivityLines_AtMostFiveLinesAreToldAtOnce(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	records := make([]auto.Record, 0, 7)
	notes := []string{"a", "b", "c", "d", "e", "f", "g"}

	for index, note := range notes {
		records = append(records, recordAt(index, auto.RecordWritten, func(r *auto.Record) { r.Note = note + ".md" }))
	}

	first := auto.ActivityLines(records, nil)

	g.Expect(first).To(HaveLen(5), "the lines of the first block")
	g.Expect(auto.ActivityLines(records, first)).To(Equal([]string{
		"20:32:30 wrote the note f.md", "20:32:31 wrote the note g.md",
	}), "the lines of the second block")
}

// recordAt makes a record of a kind, made a second after the one at the index before it, and lets the test fill it.
func recordAt(index int, kind string, fill func(*auto.Record)) auto.Record {
	zone := time.FixedZone("EDT", -4*60*60)
	record := auto.Record{
		At:      time.Date(2026, time.October, 17, 20, 32, 25+index, 0, zone),
		Session: "4f1c0a9e-6f1d-4f5e-9d58-0f3d9a3c8e11",
		Turn:    "a-turn",
		Kind:    kind,
	}

	if fill != nil {
		fill(&record)
	}

	return record
}
