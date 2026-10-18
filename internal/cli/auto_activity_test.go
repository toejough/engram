package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
	"github.com/toejough/engram/internal/cli"
)

func Test793d4p1f_RunAutoDeliver_TheRecordsOfAnotherSessionAnUnreadableRecordAndAMissingDirectoryMakeNoLine(
	t *testing.T,
) {
	t.Parallel()
	g := NewWithT(t)

	records := t.TempDir()
	mine := filepath.Join(records, sessionID)
	theirs := filepath.Join(records, otherSessionID)

	// the directory of the session: two records of its own, a record of another session, a file that is not JSON, a
	// file that is not a record, and a directory with the name of a record
	plantActivity(t, mine, "20261017T203225.000000000Z-0-written.json", activityRecordAt(0, sessionID, auto.RecordWritten,
		func(r *auto.Record) { r.Note = "a.md" }))
	plantActivity(t, mine, "20261017T203226.000000000Z-0-written.json", activityRecordAt(1, otherSessionID,
		auto.RecordWritten, func(r *auto.Record) { r.Note = "intruder.md" }))
	g.Expect(os.WriteFile(filepath.Join(mine, "20261017T203227.000000000Z-0-written.json"), []byte("{ not json"),
		0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(mine, "notes.txt"), []byte("not a record"), 0o600)).To(Succeed())
	g.Expect(os.Mkdir(filepath.Join(mine, "20261017T203228.000000000Z-0-written.json"), 0o750)).To(Succeed())
	plantActivity(t, mine, "20261017T203229.000000000Z-0-skipped.json", activityRecordAt(4, sessionID,
		auto.RecordSkipped, func(r *auto.Record) {
			r.Slug = "staging-reset"
			r.Reason = "the vault already states it"
		}))

	// the other session has its own directory, and a third has none
	plantActivity(t, theirs, "20261017T203230.000000000Z-0-failed.json", activityRecordAt(5, otherSessionID,
		auto.RecordFailed, func(r *auto.Record) { r.Reason = "the other session's failure" }))

	shown := func(session string) []string {
		out, err := deliverOver(t, records, session)
		g.Expect(err).NotTo(HaveOccurred(), "the delivery for "+session)
		g.Expect(out).To(ContainSubstring(resetNoteName), "the note for "+session)

		return told(out)
	}

	g.Expect(shown(sessionID)).To(Equal([]string{
		"* 20:32:25 wrote the note a.md",
		"* 20:32:29 left out the lesson staging-reset: the vault already states it",
	}), "the lines of the session, oldest first")
	g.Expect(shown(otherSessionID)).To(Equal([]string{"* 20:32:30 failed: the other session's failure"}),
		"the lines of the other session")
	g.Expect(shown(thirdSessionID)).To(BeEmpty(), "the lines of a session with no directory")
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

	g.Expect(told(withFlag)).To(Equal([]string{"* 20:32:25 wrote the note flagged.md"}), "with the flag")
	g.Expect(told(without)).To(Equal([]string{"* 20:32:26 wrote the note default.md"}), "without the flag")
}

// unexported constants.
const (
	// otherSessionID and thirdSessionID are the IDs of other sessions in the tests.
	otherSessionID = "9a2b7c1d-0e3f-4a5b-8c6d-1e2f3a4b5c6d"
	thirdSessionID = "0b1c2d3e-4f50-4617-8899-aabbccddeeff"
)

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

// deliverOver runs the delivery for a prompt of a session, over the vault of one note and the directory of records that
// it is given, and returns what it prints.
func deliverOver(t *testing.T, records, session string) (string, error) {
	t.Helper()

	memFS := newInMemoryFS()
	vault := t.TempDir()
	plantNoteWithSidecar(t, memFS, vault, resetNoteName, zorblaxResetNote)

	var out bytes.Buffer

	err := cli.RunAutoDeliver(context.Background(), cli.AutoDeliverArgs{VaultPath: vault, RecordsDir: records},
		cli.AutoDeliverDeps{
			Stdin:    userPrompt("When is the Zorblax staging database reset?", session),
			Query:    queryOver(newQueryDeps(memFS)),
			ReadDir:  os.ReadDir,
			ReadFile: os.ReadFile,
		}, &out)

	return out.String(), err
}

// deliverWith dispatches `engram auto deliver`, as the arguments say, for a prompt of the session, with the home and
// the environment that are given and nothing else of the machine's, and returns what it prints.
func deliverWith(t *testing.T, args []string, home string, environment map[string]string) string {
	t.Helper()

	var stdout bytes.Buffer

	stderr := executeForTestWithDeps(t, args, func(d *cli.Deps) {
		d.Embed = stubEmbedder{modelID: "m@4", dims: 4}
		d.Getenv = func(name string) string { return environment[name] }
		d.UserHomeDir = func() (string, error) { return home, nil }
		d.Stdin = userPrompt("When is the Zorblax staging database reset?", sessionID)
		d.Stdout = &stdout
	})
	if stderr != "" {
		t.Fatalf("the dispatch printed to standard error: %s", stderr)
	}

	return stdout.String()
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

// told returns the lines of a block that tell what automatic memory did, as they were printed.
func told(block string) []string {
	var lines []string

	for _, line := range strings.Split(block, "\n") {
		if strings.HasPrefix(line, "* ") {
			lines = append(lines, line)
		}
	}

	return lines
}
