package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	first := deliverOver(t, records, "", sessionID)
	second := deliverOver(t, records, "", sessionID)

	g.Expect(first.err).NotTo(HaveOccurred())
	g.Expect(second.err).NotTo(HaveOccurred())
	g.Expect(recordLines(first.shown)).To(Equal([]string{"20:32:26 wrote the note b.md"}), "the lines of the first prompt")
	g.Expect(recordLines(second.shown)).To(BeEmpty(), "the lines of the second prompt")
	g.Expect(filepath.Join(dir, secondFile+".told")).To(BeAnExistingFile(), "the mark of the line that was shown")
}

func Test793d4p1e_RunAutoDeliver_AtMostFiveLinesAreShownAtOnce(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	records := t.TempDir()
	dir := filepath.Join(records, sessionID)

	for index := range 7 {
		plantActivity(t, dir, fmt.Sprintf("2026101720322%d.000000000Z-0-written.json", index),
			activityRecordAt(index, sessionID, auto.RecordWritten, func(r *auto.Record) {
				r.Note = fmt.Sprintf("note-%d.md", index)
			}))
	}

	first := deliverOver(t, records, "", sessionID)
	second := deliverOver(t, records, "", sessionID)
	third := deliverOver(t, records, "", sessionID)

	g.Expect(recordLines(first.shown)).To(HaveLen(5), "the lines of the first prompt")
	g.Expect(recordLines(second.shown)).To(Equal([]string{
		"20:32:30 wrote the note note-5.md", "20:32:31 wrote the note note-6.md",
	}), "the lines of the second prompt")
	g.Expect(recordLines(third.shown)).To(BeEmpty(), "the lines of the third prompt")
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
		result := deliverOver(t, records, "", session)
		g.Expect(result.err).NotTo(HaveOccurred(), "the delivery for "+session)
		g.Expect(result.block).To(ContainSubstring(resetNoteName), "the note for "+session)

		return recordLines(result.shown)
	}

	g.Expect(shown(sessionID)).To(Equal([]string{
		"20:32:25 wrote the note a.md",
		"20:32:29 left out the lesson staging-reset: the vault already states it",
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

	result := deliverOver(t, t.TempDir(), "", sessionID)

	g.Expect(result.err).NotTo(HaveOccurred())
	g.Expect(result.block).To(ContainSubstring("- "+resetNoteName+" ("), "the block, on standard output")
	g.Expect(result.shown).To(Equal("delivered the note "+resetNoteName+"\n"), "what the user is shown")
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

// deliverOver runs the delivery for a prompt of a session, over the vault of one note and the directories of records of
// what automatic memory did and of sessions that it is given, and returns what it prints.
func deliverOver(t *testing.T, records, sessions, session string) shownBy {
	t.Helper()

	memFS := newInMemoryFS()
	vault := t.TempDir()
	plantNoteWithSidecar(t, memFS, vault, resetNoteName, zorblaxResetNote)

	var block, shown bytes.Buffer

	err := cli.RunAutoDeliver(context.Background(),
		cli.AutoDeliverArgs{VaultPath: vault, SessionsDir: sessions, RecordsDir: records},
		cli.AutoDeliverDeps{
			Stdin:       userPrompt("When is the Zorblax staging database reset?", session),
			Query:       queryOver(newQueryDeps(memFS)),
			ReadDir:     os.ReadDir,
			ReadFile:    os.ReadFile,
			WriteRecord: writeFileExclusively,
		}, &block, &shown)

	return shownBy{block: block.String(), shown: shown.String(), err: err}
}

// deliverWith dispatches `engram auto deliver`, as the arguments say, for a prompt of the session, with the home and
// the environment that are given and nothing else of the machine's, and returns what it prints.
func deliverWith(t *testing.T, args []string, home string, environment map[string]string) shownBy {
	t.Helper()

	var stdout bytes.Buffer

	stderr := executeForTestWithDeps(t, args, func(d *cli.Deps) {
		d.Embed = stubEmbedder{modelID: "m@4", dims: 4}
		d.Getenv = func(name string) string { return environment[name] }
		d.UserHomeDir = func() (string, error) { return home, nil }
		d.Stdin = userPrompt("When is the Zorblax staging database reset?", sessionID)
		d.Stdout = &stdout
	})

	return shownBy{block: stdout.String(), shown: stderr}
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
		if line != "" && !strings.HasPrefix(line, "delivered the note ") {
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
