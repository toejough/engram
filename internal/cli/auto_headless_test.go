package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
	"github.com/toejough/engram/internal/cli"
)

func Test793d4m1e_RunAutoJudge_AHeadlessTurnThatIsNotJudgedIsRecordedAsSkippedWithTheReason(t *testing.T) {
	t.Parallel()

	moment := time.Date(2026, time.October, 17, 20, 32, 25, 0, time.UTC)
	records := "/data/engram/auto"

	// judge gives the judgement a turn of an origin and returns what it printed, the records it wrote by the path of
	// each, and how it ended
	judge := func(t *testing.T, origin string, captureSDK bool, writeErr error) (string, map[string][]byte, error) {
		t.Helper()

		written := map[string][]byte{}

		var printed bytes.Buffer

		err := cli.RunAutoJudge(context.Background(),
			cli.AutoJudgeArgs{VaultPath: t.TempDir(), SessionsDir: recordOfAPrompt(t), RecordsDir: records,
				CaptureSDK: captureSDK},
			cli.AutoJudgeDeps{
				Stdin: requestOf(map[string]string{
					"answer": "On Sundays.", "session": sessionID, "turn": "a-turn", "origin": origin,
				}),
				Query:    queryOver(newQueryDeps(newInMemoryFS())),
				ReadDir:  os.ReadDir,
				ReadFile: os.ReadFile,
				Now:      func() time.Time { return moment },
				WriteRecord: func(path string, data []byte) error {
					written[path] = data

					return writeErr
				},
			}, &printed)

		return printed.String(), written, err
	}

	t.Run("the turn of a headless prompt, without the flag", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		printed, written, err := judge(t, "sdk", false, nil)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(printed).To(BeEmpty(), "the request for the model")
		g.Expect(written).To(HaveLen(1), "the records")

		for path, data := range written {
			g.Expect(path).To(Equal(filepath.Join(records, sessionID, "20261017T203225.000000000Z-0-skipped.json")))

			var record auto.Record

			g.Expect(json.Unmarshal(data, &record)).To(Succeed())
			g.Expect(record.At).To(BeTemporally("==", moment))
			g.Expect(record.Session).To(Equal(sessionID))
			g.Expect(record.Turn).To(Equal("a-turn"))
			g.Expect(record.Kind).To(Equal(auto.RecordSkipped))
			g.Expect(record.Note).To(BeEmpty())
			g.Expect(record.Slug).To(BeEmpty())
			g.Expect(record.Reason).To(And(ContainSubstring("not captured"), ContainSubstring("ENGRAM_AUTO_CAPTURE_SDK")))
		}
	})

	t.Run("the turn of a headless prompt, with the flag", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		printed, written, err := judge(t, "sdk", true, nil)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(printed).To(HavePrefix(`{"model":`), "the request for the model")
		g.Expect(written).To(BeEmpty(), "the records")
	})

	t.Run("the turns of the origins that are judged for nothing", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		for _, origin := range []string{"task-notification", "scheduled-trigger", ""} {
			printed, written, err := judge(t, origin, false, nil)

			g.Expect(err).NotTo(HaveOccurred(), origin)
			g.Expect(printed).To(BeEmpty(), origin)
			g.Expect(written).To(BeEmpty(), origin)
		}
	})

	t.Run("a record that cannot be written does not fail the command", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		printed, written, err := judge(t, "sdk", false, errNoWrite)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(printed).To(BeEmpty())
		g.Expect(written).To(HaveLen(1), "the record was tried")
	})
}

func Test793d4m1e_Targets_AHeadlessTurnThatIsNotJudgedIsRecordedUnderTheEngramDataDirectory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault, home := t.TempDir(), t.TempDir()
	sessions := recordOfAPrompt(t)

	var stdout bytes.Buffer

	stderr := executeForTestWithDeps(t, []string{"engram", "auto", "judge", "--vault", vault, "--sessions", sessions},
		func(d *cli.Deps) {
			d.Embed = stubEmbedder{modelID: "m@4", dims: 4}
			d.Getenv = func(string) string { return "" }
			d.UserHomeDir = func() (string, error) { return home, nil }
			d.Stdin = requestOf(map[string]string{
				"answer": "On Sundays.", "session": sessionID, "turn": "a-turn", "origin": "sdk",
			})
			d.Stdout = &stdout
		})

	recorded, globErr := filepath.Glob(
		filepath.Join(home, ".local", "share", "engram", "auto", sessionID, "*-skipped.json"))

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout.String()).To(BeEmpty(), "the turn of a headless prompt is judged nothing")
	g.Expect(globErr).NotTo(HaveOccurred())
	g.Expect(recorded).To(HaveLen(1), "the records of the turn")
}

// unexported variables.
var (
	errNoWrite = errors.New("permission denied")
)
