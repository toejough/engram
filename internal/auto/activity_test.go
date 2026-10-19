package auto_test

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
)

func Test793a2e1a_ActivityLines_ANoteThatWasAmendedIsToldAsAmended(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lines := auto.ActivityLines([]auto.Record{
		recordAt(0, auto.RecordWritten, func(r *auto.Record) {
			r.Note, r.Reason = "1.2026-10-17.zorblax.md", auto.ReasonAmended
		}),
	})

	g.Expect(textsOf(lines)).To(Equal([]string{"20:32:25 wrote the note 1.2026-10-17.zorblax.md (amended)"}))
}

func Test793d4p1a_ActivityLines_ARecordThatWasNotShownBecomesALine(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lines := auto.ActivityLines([]auto.Record{
		recordAt(0, auto.RecordWritten, func(r *auto.Record) { r.Note = "2.2026-10-17.zorblax-backup.md" }),
		recordAt(1, auto.RecordSkipped, func(r *auto.Record) {
			r.Slug, r.Reason = "zorblax-build", "the vault already states it: 1.2026-10-17.zorblax-build.md"
		}),
		recordAt(2, auto.RecordSkipped, func(r *auto.Record) { r.Reason = "the reply is not a list of lessons" }),
		recordAt(3, auto.RecordFailed, func(r *auto.Record) { r.Reason = "the model call failed: timed out" }),
		recordAt(4, auto.RecordFailed, func(r *auto.Record) {
			r.Slug, r.Reason = "zorblax-build", "learn: acquiring lock: permission denied"
		}),
	})

	g.Expect(textsOf(lines)).To(Equal([]string{
		"20:32:25 wrote the note 2.2026-10-17.zorblax-backup.md",
		"20:32:26 left out: the lesson zorblax-build: the vault already states it: 1.2026-10-17.zorblax-build.md; " +
			"the reply is not a list of lessons",
		"20:32:28 failed: the model call failed: timed out; writing zorblax-build: learn: acquiring lock: permission denied",
	}), "one line for the written, one for the left out and one for the failed")
	g.Expect(filesOf(lines)).To(Equal([][]string{
		{"record-0.json"}, {"record-1.json", "record-2.json"}, {"record-3.json", "record-4.json"},
	}), "the records that each line tells")
}

func Test793d4p1b_ActivityLines_ARecordOfAKindThatTellsNothingMakesNoLine(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// nothing is the kind that an earlier version made for a reply with no lesson
	g.Expect(auto.ActivityLines([]auto.Record{recordAt(0, "nothing", nil)})).To(BeEmpty())
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
	})

	g.Expect(lines).To(HaveLen(1))

	if len(lines) != 1 {
		return
	}

	line := lines[0].Text

	g.Expect(line).NotTo(ContainSubstring("\n"), "the line is one line")
	g.Expect(line).NotTo(ContainSubstring("ghp_"), "the credential, or part of it")
	g.Expect(utf8.RuneCountInString(line)).To(BeNumerically("<=", 240), "the line as shown")
	g.Expect(line).To(HavePrefix("20:32:25 failed: learn: writing xxx"), "the beginning of the line")
	g.Expect(line).To(HaveSuffix("yyy: permission denied"), "the end of the line, which holds the cause")
}

func Test793d4p1e_ActivityLines_OneLineIsShownForEachKindOfRecord(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	records := make([]auto.Record, 0, 11)

	for index := range 7 {
		records = append(records, recordAt(index, auto.RecordWritten, func(r *auto.Record) {
			r.Note = fmt.Sprintf("%d.2026-10-17.zorblax-note-%d.md", index+1, index)
		}))
	}

	for index := 7; index < 9; index++ {
		records = append(records, recordAt(index, auto.RecordSkipped, func(r *auto.Record) {
			r.Slug, r.Reason = fmt.Sprintf("lesson-%d", index), "the vault already states it"
		}))
	}

	for index := 9; index < 11; index++ {
		records = append(records, recordAt(index, auto.RecordFailed, func(r *auto.Record) { r.Reason = "the disk is full" }))
	}

	lines := auto.ActivityLines(records)

	g.Expect(lines).To(HaveLen(3), "one line for each kind")

	if len(lines) != 3 {
		return
	}

	written := lines[0].Text

	g.Expect(utf8.RuneCountInString(written)).To(BeNumerically("<=", 240), "the line of the written, cut")
	g.Expect(written).To(HavePrefix("20:32:25 wrote the notes 1.2026-10-17.zorblax-note-0.md, " +
		"2.2026-10-17.zorblax-note-1.md"))
	g.Expect(written).To(HaveSuffix("7.2026-10-17.zorblax-note-6.md"), "the first and the last of the file names stay")
	g.Expect(lines[0].Files).To(HaveLen(7), "every record is told, also those the cut left out of the text")
	g.Expect(lines[1].Text).To(Equal("20:32:32 left out: the lesson lesson-7: the vault already states it; " +
		"the lesson lesson-8: the vault already states it"))
	g.Expect(lines[2].Text).To(Equal("20:32:34 failed: the disk is full; the disk is full"))
}

// filesOf returns the files of the records that each line tells.
func filesOf(lines []auto.Line) [][]string {
	files := make([][]string, 0, len(lines))

	for _, line := range lines {
		files = append(files, line.Files)
	}

	return files
}

// recordAt makes a record of a kind, made a second after the one at the index before it, and lets the test fill it.
func recordAt(index int, kind string, fill func(*auto.Record)) auto.Record {
	zone := time.FixedZone("EDT", -4*60*60)
	record := auto.Record{
		At:      time.Date(2026, time.October, 17, 20, 32, 25+index, 0, zone),
		Session: "4f1c0a9e-6f1d-4f5e-9d58-0f3d9a3c8e11",
		Turn:    "a-turn",
		Kind:    kind,
		File:    fmt.Sprintf("record-%d.json", index),
	}

	if fill != nil {
		fill(&record)
	}

	return record
}

// textsOf returns the text of each line.
func textsOf(lines []auto.Line) []string {
	texts := make([]string, 0, len(lines))

	for _, line := range lines {
		texts = append(texts, line.Text)
	}

	return texts
}
