package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
	"github.com/toejough/engram/internal/cli"
)

func Test793d4p1c_RunAutoDeliver_ALineThatWasShownIsNotShownAgain(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	records := t.TempDir()
	dir := filepath.Join(records, sessionID)
	firstFile, secondFile := "20261017T203225.000000000Z-0-written.json", "20261017T203226.000000000Z-0-written.json"

	plantActivity(t, dir, firstFile, activityRecordAt(0, sessionID, auto.RecordWritten,
		func(r *auto.Record) { r.Note = "a.md" }))
	plantActivity(t, dir, secondFile, activityRecordAt(1, sessionID, auto.RecordWritten,
		func(r *auto.Record) { r.Note = "b.md" }))

	// the mark beside the first record: its line was shown
	g.Expect(os.WriteFile(filepath.Join(dir, firstFile+".told"), nil, 0o600)).To(Succeed())

	first := deliverOverRemoving(t, records, sessionID, keepTheRecords)
	second := deliverOverRemoving(t, records, sessionID, keepTheRecords)

	g.Expect(first.err).NotTo(HaveOccurred())
	g.Expect(second.err).NotTo(HaveOccurred())
	g.Expect(recordLines(first.shown)).To(Equal([]string{"20:32:26 wrote the note b.md"}), "the lines of the first prompt")
	g.Expect(recordLines(second.shown)).To(BeEmpty(), "the lines of the second prompt")
	g.Expect(filepath.Join(dir, secondFile+".told")).To(BeAnExistingFile(), "the mark of the line that was shown")
}

func Test793d4p1e_RunAutoDeliver_OneLineIsShownForEachKindOfRecord(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	records := t.TempDir()
	dir := filepath.Join(records, sessionID)
	files := make([]string, 0, 11)

	for index := range 7 {
		name := fmt.Sprintf("2026101720322%d.000000000Z-0-written.json", index)
		files = append(files, name)

		plantActivity(t, dir, name, activityRecordAt(index, sessionID, auto.RecordWritten, func(r *auto.Record) {
			r.Note = fmt.Sprintf("%d.2026-10-17.zorblax-note-%d.md", index+1, index)
		}))
	}

	for index := 7; index < 9; index++ {
		name := fmt.Sprintf("2026101720322%d.000000000Z-0-skipped.json", index)
		files = append(files, name)

		plantActivity(t, dir, name, activityRecordAt(index, sessionID, auto.RecordSkipped, func(r *auto.Record) {
			r.Slug, r.Reason = fmt.Sprintf("lesson-%d", index), "the vault already states it"
		}))
	}

	for index := 9; index < 11; index++ {
		name := fmt.Sprintf("202610172032%d.000000000Z-0-failed.json", index+15)
		files = append(files, name)

		plantActivity(t, dir, name, activityRecordAt(index, sessionID, auto.RecordFailed,
			func(r *auto.Record) { r.Reason = "the disk is full" }))
	}

	first := deliverOverRemoving(t, records, sessionID, keepTheRecords)
	second := deliverOverRemoving(t, records, sessionID, keepTheRecords)

	g.Expect(first.err).NotTo(HaveOccurred())
	g.Expect(second.err).NotTo(HaveOccurred())

	lines := recordLines(first.shown)

	g.Expect(lines).To(HaveLen(3), "the lines of the first prompt: one for each kind")

	if len(lines) == 3 {
		g.Expect(utf8.RuneCountInString(lines[0])).To(BeNumerically("<=", 240), "the line of the written")
		g.Expect(lines[0]).To(HavePrefix("20:32:25 wrote the notes 1.2026-10-17.zorblax-note-0.md, "))
		g.Expect(lines[0]).To(HaveSuffix("7.2026-10-17.zorblax-note-6.md"))
	}

	g.Expect(recordLines(second.shown)).To(BeEmpty(), "the lines of the second prompt")

	for _, name := range files {
		g.Expect(filepath.Join(dir, name+".told")).To(BeAnExistingFile(), "the mark of the record "+name)
	}
}

func Test793d4p1f_RunAutoDeliver_TheRecordsOfAnotherSessionAnUnreadableRecordAndAMissingDirectoryMakeNoLine(
	t *testing.T,
) {
	t.Parallel()
	g := NewWithT(t)

	records, beside := t.TempDir(), t.TempDir()
	mine := filepath.Join(records, sessionID)
	theirs := filepath.Join(records, otherSessionID)

	// the directory of the session: two records of its own, a record of another session, a file that is not JSON, a
	// file of a record's content that is not named as a record is, and a directory with the name of a record
	plantActivity(t, mine, "20261017T203225.000000000Z-0-written.json", activityRecordAt(0, sessionID, auto.RecordWritten,
		func(r *auto.Record) { r.Note = "a.md" }))
	plantActivity(t, mine, "20261017T203226.000000000Z-0-written.json", activityRecordAt(1, otherSessionID,
		auto.RecordWritten, func(r *auto.Record) { r.Note = "intruder.md" }))
	g.Expect(os.WriteFile(filepath.Join(mine, "20261017T203227.000000000Z-0-written.json"), []byte("{ not json"),
		0o600)).To(Succeed())
	notRecord, marshalErr := json.Marshal(activityRecordAt(2, sessionID, auto.RecordWritten,
		func(r *auto.Record) { r.Note = "not-a-record.md" }))
	g.Expect(marshalErr).NotTo(HaveOccurred())
	g.Expect(os.WriteFile(filepath.Join(mine, "notes.txt"), notRecord, 0o600)).To(Succeed())
	g.Expect(os.Mkdir(filepath.Join(mine, "20261017T203228.000000000Z-0-written.json"), 0o750)).To(Succeed())
	plantActivity(t, mine, "20261017T203229.000000000Z-0-skipped.json", activityRecordAt(4, sessionID,
		auto.RecordSkipped, func(r *auto.Record) {
			r.Slug = "staging-reset"
			r.Reason = "the vault already states it"
		}))

	// the other session has its own directory, a third has none, and a fourth is named by a path to a directory beside
	// the records, which holds a record that claims to be the session's
	aside := "../" + filepath.Base(beside)
	plantActivity(t, beside, "20261017T203231.000000000Z-0-written.json", activityRecordAt(6, aside,
		auto.RecordWritten, func(r *auto.Record) { r.Note = "aside.md" }))
	plantActivity(t, theirs, "20261017T203230.000000000Z-0-failed.json", activityRecordAt(5, otherSessionID,
		auto.RecordFailed, func(r *auto.Record) { r.Reason = "the other session's failure" }))

	shown := func(session string) []string {
		result := deliverOver(t, records, session)
		g.Expect(result.err).NotTo(HaveOccurred(), "the delivery for "+session)
		g.Expect(result.block).To(ContainSubstring(resetNoteName), "the note for "+session)

		return recordLines(result.shown)
	}

	g.Expect(shown(sessionID)).To(Equal([]string{
		"20:32:25 wrote the note a.md",
		"20:32:29 left out: the lesson staging-reset: the vault already states it",
	}), "the lines of the session, oldest first")
	g.Expect(shown(otherSessionID)).To(Equal([]string{"20:32:30 failed: the other session's failure"}),
		"the lines of the other session")
	g.Expect(shown(thirdSessionID)).To(BeEmpty(), "the lines of a session with no directory")
	g.Expect(shown(aside)).To(BeEmpty(), "the lines of a session named by a path")
}

func Test793d4p1g_Targets_TheRecordsAreReadFromTheDirectoryTheFlagNamesElseFromTheEngramDataDirectory(
	t *testing.T,
) {
	t.Parallel()
	g := NewWithT(t)

	vault := plantTwoNotes(t)
	home, flagged := t.TempDir(), t.TempDir()
	fromFlag := activityRecordAt(0, sessionID, auto.RecordWritten, func(r *auto.Record) { r.Note = "flagged.md" })
	fromData := activityRecordAt(1, sessionID, auto.RecordWritten, func(r *auto.Record) { r.Note = "default.md" })

	plantActivity(t, filepath.Join(flagged, sessionID), "20261017T203225.000000000Z-0-written.json", fromFlag)
	plantActivity(t, filepath.Join(home, ".local", "share", "engram", "auto", sessionID),
		"20261017T203226.000000000Z-0-written.json", fromData)

	withFlag := deliverWith(t, []string{"engram", "auto", "deliver", "--vault", vault, "--records", flagged}, home, nil)
	without := deliverWith(t, []string{"engram", "auto", "deliver", "--vault", vault}, home, nil)

	g.Expect(recordLines(withFlag.shown)).To(Equal([]string{"20:32:25 wrote the note flagged.md"}), "with the flag")
	g.Expect(recordLines(without.shown)).To(Equal([]string{"20:32:26 wrote the note default.md"}), "without the flag")
}

func Test793d4p1h_RunAutoDeliver_ANoteDeliveredIsShownAsALine(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := deliverOver(t, t.TempDir(), sessionID)

	g.Expect(result.err).NotTo(HaveOccurred())
	g.Expect(result.block).To(ContainSubstring("- "+resetNoteName+" ("), "the block, on standard output")
	g.Expect(result.shown).To(Equal("delivered the note "+resetNoteName+"\n"), "what the user is shown")
}

func Test855d4p1b_RunAutoDeliver_ADeliveryWithinATurnReadsMarksAndRemovesNoRecord(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	records := t.TempDir()
	dir := filepath.Join(records, sessionID)
	shownFile, newFile := "20261017T203225.000000000Z-0-written.json", "20261017T203226.000000000Z-0-written.json"

	plantActivity(t, dir, shownFile, activityRecordAt(0, sessionID, auto.RecordWritten,
		func(r *auto.Record) { r.Note = "a.md" }))
	plantActivity(t, dir, newFile, activityRecordAt(1, sessionID, auto.RecordWritten,
		func(r *auto.Record) { r.Note = "b.md" }))
	g.Expect(os.WriteFile(filepath.Join(dir, shownFile+".told"), nil, 0o600)).To(Succeed())

	fields := deliveryFields(sessionID, resetCandidate())
	fields["cue"] = "before-tool"

	var block, shown bytes.Buffer

	err := cli.RunAutoDeliver(context.Background(),
		cli.AutoDeliverArgs{VaultPath: t.TempDir(), RecordsDir: records},
		cli.AutoDeliverDeps{
			Stdin: requestJSON(fields), Activate: noActivation, ReadDir: os.ReadDir, ReadFile: os.ReadFile,
			WriteRecord: writeFileExclusively, Remove: os.Remove,
		}, &block, &shown)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(shown.String()).To(Equal("delivered the note "+resetNoteName+"\n"), "the line of the note only")

	for _, kept := range []string{shownFile, shownFile + ".told", newFile} {
		g.Expect(filepath.Join(dir, kept)).To(BeAnExistingFile(), kept+" was left as it was")
	}

	g.Expect(filepath.Join(dir, newFile+".told")).NotTo(BeAnExistingFile(), "a record was marked")
}

func Test793d4p1j_RunAutoDeliver_AMarkThatCannotBeWrittenDoesNotFailTheCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	records := t.TempDir()

	plantActivity(t, filepath.Join(records, sessionID), "20261017T203225.000000000Z-0-written.json",
		activityRecordAt(0, sessionID, auto.RecordWritten, func(r *auto.Record) { r.Note = "a.md" }))

	var block, shown bytes.Buffer

	err := cli.RunAutoDeliver(context.Background(), cli.AutoDeliverArgs{VaultPath: t.TempDir(), RecordsDir: records},
		cli.AutoDeliverDeps{
			Stdin:       userPrompt("When is the Zorblax staging database reset?", sessionID),
			Activate:    noActivation,
			ReadDir:     os.ReadDir,
			ReadFile:    os.ReadFile,
			WriteRecord: func(string, []byte) error { return errors.New("the disk is full") },
			Remove:      os.Remove,
		}, &block, &shown)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(recordLines(shown.String())).To(Equal([]string{"20:32:25 wrote the note a.md"}), "the line is shown")
}

func Test793d4p1l_Targets_TheNotesDeliveredAreOneLine(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := deliverWithRequest(t,
		[]string{"engram", "auto", "deliver", "--vault", plantTwoNotes(t), "--records", t.TempDir()}, t.TempDir(), nil,
		deliveryRequest(sessionID, resetCandidate(), backupCandidate()))

	g.Expect(result.block).To(ContainSubstring("- "+resetNoteName+" ("), "the block holds the first note")
	g.Expect(result.block).To(ContainSubstring("- "+backupNoteName+" ("), "the block holds the second note")

	lines := strings.Split(strings.TrimSpace(result.shown), "\n")

	g.Expect(lines).To(HaveLen(1), "what the user is shown")

	if len(lines) == 1 {
		g.Expect(lines[0]).To(HavePrefix("delivered the notes "))
		g.Expect(lines[0]).To(ContainSubstring(resetNoteName))
		g.Expect(lines[0]).To(ContainSubstring(backupNoteName))
	}
}

func Test793d4p1m_RunAutoDeliver_TheRecordsThatWereShownAreRemovedWithTheirMarks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	records := t.TempDir()
	dir := filepath.Join(records, sessionID)
	shownFile, newFile := "20261017T203225.000000000Z-0-written.json", "20261017T203226.000000000Z-0-written.json"

	plantActivity(t, dir, shownFile, activityRecordAt(0, sessionID, auto.RecordWritten,
		func(r *auto.Record) { r.Note = "a.md" }))
	plantActivity(t, dir, newFile, activityRecordAt(1, sessionID, auto.RecordWritten,
		func(r *auto.Record) { r.Note = "b.md" }))
	g.Expect(os.WriteFile(filepath.Join(dir, shownFile+".told"), nil, 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not a record"), 0o600)).To(Succeed())

	first := deliverOver(t, records, sessionID)
	second := deliverOver(t, records, sessionID)

	g.Expect(first.err).NotTo(HaveOccurred())
	g.Expect(recordLines(first.shown)).To(Equal([]string{"20:32:26 wrote the note b.md"}), "the lines of the first prompt")

	for _, gone := range []string{shownFile, shownFile + ".told", newFile, newFile + ".told"} {
		g.Expect(filepath.Join(dir, gone)).NotTo(BeAnExistingFile(), gone+" was shown")
	}

	g.Expect(filepath.Join(dir, "notes.txt")).To(BeAnExistingFile(), "a file that is no record")
	g.Expect(recordLines(second.shown)).To(BeEmpty(), "the lines of the second prompt")
}

func Test793d4p1n_RunAutoDeliver_TheDirectoryOfASessionWhoseRecordsWereShownIsRemoved(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	records := t.TempDir()
	dir := filepath.Join(records, sessionID)

	plantActivity(t, dir, "20261017T203225.000000000Z-0-written.json", activityRecordAt(0, sessionID,
		auto.RecordWritten, func(r *auto.Record) { r.Note = "a.md" }))

	first := deliverOver(t, records, sessionID)
	second := deliverOver(t, records, sessionID)

	g.Expect(first.err).NotTo(HaveOccurred())
	g.Expect(recordLines(first.shown)).To(Equal([]string{"20:32:25 wrote the note a.md"}), "the lines of the first prompt")
	g.Expect(dir).NotTo(BeADirectory(), "the directory of the session")
	g.Expect(recordLines(second.shown)).To(BeEmpty(), "the lines of the second prompt")
}

// unexported constants.
const (
	// otherSessionID and thirdSessionID are the IDs of other sessions in the tests.
	otherSessionID = "9a2b7c1d-0e3f-4a5b-8c6d-1e2f3a4b5c6d"
	thirdSessionID = "0b1c2d3e-4f50-4617-8899-aabbccddeeff"
)

// shownBy is what a delivery printed: the block on standard output and the lines for the user on standard error.
type shownBy struct {
	block string
	shown string
	err   error
}

// activityRecordAt makes a record of what automatic memory did, made a second after the one at the index before it, and
// lets the test fill it.
func activityRecordAt(index int, session, kind string, fill func(*auto.Record)) auto.Record {
	record := auto.Record{
		At:      time.Date(2026, time.October, 17, 20, 32, 25+index, 0, time.FixedZone("EDT", -4*60*60)),
		Session: session,
		Turn:    "a-turn",
		Kind:    kind,
	}

	fill(&record)

	return record
}

// deliverOver runs the delivery for a prompt of a session, over the vault of one note and the directory of records of
// what automatic memory did that it is given, and returns what it prints.
func deliverOver(t *testing.T, records, session string) shownBy {
	t.Helper()

	return deliverOverRemoving(t, records, session, os.Remove)
}

// deliverOverRemoving is deliverOver with the removal of the records that were shown given, so that a test of the marks
// can have a place that cannot remove them.
func deliverOverRemoving(t *testing.T, records, session string, remove func(path string) error) shownBy {
	t.Helper()

	var block, shown bytes.Buffer

	err := cli.RunAutoDeliver(context.Background(),
		cli.AutoDeliverArgs{VaultPath: t.TempDir(), RecordsDir: records},
		cli.AutoDeliverDeps{
			Stdin:       userPrompt("When is the Zorblax staging database reset?", session),
			Activate:    noActivation,
			ReadDir:     os.ReadDir,
			ReadFile:    os.ReadFile,
			WriteRecord: writeFileExclusively,
			Remove:      remove,
		}, &block, &shown)

	return shownBy{block: block.String(), shown: shown.String(), err: err}
}

// deliverWith dispatches `engram auto deliver`, as the arguments say, for a prompt of the session, with the home and
// the environment that are given and nothing else of the machine's, and returns what it prints.
func deliverWith(t *testing.T, args []string, home string, environment map[string]string) shownBy {
	t.Helper()

	return deliverWithRequest(t, args, home, environment, userPrompt("When is the Zorblax staging database reset?", sessionID))
}

// deliverWithRequest is deliverWith for the request given.
func deliverWithRequest(
	t *testing.T, args []string, home string, environment map[string]string, request io.Reader,
) shownBy {
	t.Helper()

	var stdout bytes.Buffer

	stderr := executeForTestWithDeps(t, args, func(d *cli.Deps) {
		d.Embed = stubEmbedder{modelID: "m@4", dims: 4}
		d.Getenv = func(name string) string { return environment[name] }
		d.UserHomeDir = func() (string, error) { return home, nil }
		d.Stdin = request
		d.Stdout = &stdout
	})

	return shownBy{block: stdout.String(), shown: stderr}
}

// keepTheRecords is a removal that removes nothing: a place that cannot remove a record.
func keepTheRecords(string) error {
	return errors.New("the place cannot remove a record")
}

// plantActivity writes a record of what automatic memory did as the writing makes it: a JSON file of its own, in a
// directory of the session.
func plantActivity(t *testing.T, dir, name string, record auto.Record) {
	t.Helper()

	mkdirErr := os.MkdirAll(dir, 0o750)
	if mkdirErr != nil {
		t.Fatalf("creating %s: %v", dir, mkdirErr)
	}

	data, marshalErr := json.Marshal(record)
	if marshalErr != nil {
		t.Fatalf("encoding the record: %v", marshalErr)
	}

	writeErr := os.WriteFile(filepath.Join(dir, name), data, 0o600)
	if writeErr != nil {
		t.Fatalf("writing the record: %v", writeErr)
	}
}

// recordLines returns the lines a delivery printed for the user that tell what became of a record: all but the lines of
// the notes it delivered.
func recordLines(shown string) []string {
	var lines []string

	for line := range strings.SplitSeq(shown, "\n") {
		if line != "" && !strings.HasPrefix(line, "delivered the note") {
			lines = append(lines, line)
		}
	}

	return lines
}

// writeFileExclusively creates a file that must not exist, as the file system of the command does.
func writeFileExclusively(path string, data []byte) error {
	file, openErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if openErr != nil {
		return fmt.Errorf("creating %s: %w", path, openErr)
	}

	_, writeErr := file.Write(data)

	return errors.Join(writeErr, file.Close())
}
