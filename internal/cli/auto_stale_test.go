package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
	"github.com/toejough/engram/internal/cli"
)

func Test855d4g1g_RunAutoJudge_AReplyThatIsEmptyIsStillAFailureAndWritesNoRecord(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sessions := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(sessions, sessionID+".jsonl"),
		[]byte(`{"type":"user","message":{"role":"user","content":"No - build with make zorblax-build."}}`+"\n"),
		0o600)).To(Succeed())

	written := map[string][]byte{}

	var printed bytes.Buffer

	err := cli.RunAutoJudge(context.Background(),
		cli.AutoJudgeArgs{VaultPath: t.TempDir(), SessionsDir: sessions, RecordsDir: "/data/engram/auto"},
		cli.AutoJudgeDeps{
			Stdin: bytes.NewReader([]byte(
				`{"origin":"composer","session":"` + sessionID + `","turn":"a-turn","reply":"","answer":"Understood."}`)),
			Query:    queryOver(newQueryDeps(newInMemoryFS())),
			ReadDir:  os.ReadDir,
			ReadFile: os.ReadFile,
			Now:      time.Now,
			WriteRecord: func(path string, data []byte) error {
				written[path] = data

				return nil
			},
		}, &printed)

	g.Expect(err).To(MatchError(auto.ErrLearnReply), "a reply the new mod sent that is no moment reply fails")
	g.Expect(written).To(BeEmpty())
}

func Test855d4g1g_RunAutoJudge_ATurnFromAnOlderModIsRecordedAsSkippedAndNotAFailure(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sessions := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(sessions, sessionID+".jsonl"),
		[]byte(`{"type":"user","message":{"role":"user","content":"No - build with make zorblax-build."}}`+"\n"),
		0o600)).To(Succeed())

	moment := time.Date(2026, time.October, 19, 9, 30, 0, 0, time.UTC)
	records := "/data/engram/auto"
	written := map[string][]byte{}

	var printed bytes.Buffer

	// the old mod hands over the turn without the reply of the learn-moment judgement
	err := cli.RunAutoJudge(context.Background(),
		cli.AutoJudgeArgs{VaultPath: t.TempDir(), SessionsDir: sessions, RecordsDir: records},
		cli.AutoJudgeDeps{
			Stdin: bytes.NewReader([]byte(
				`{"origin":"composer","session":"` + sessionID + `","turn":"a-turn","answer":"Understood."}`)),
			Query:    queryOver(newQueryDeps(newInMemoryFS())),
			ReadDir:  os.ReadDir,
			ReadFile: os.ReadFile,
			Now:      func() time.Time { return moment },
			WriteRecord: func(path string, data []byte) error {
				written[path] = data

				return nil
			},
		}, &printed)

	g.Expect(err).NotTo(HaveOccurred(), "an older mod's turn is no failure")
	g.Expect(printed.String()).To(BeEmpty(), "the model is not asked")
	g.Expect(written).To(HaveLen(1), "the records")

	for path, data := range written {
		g.Expect(path).To(Equal(filepath.Join(records, sessionID, "20261019T093000.000000000Z-0-skipped.json")))

		var record auto.Record

		g.Expect(json.Unmarshal(data, &record)).To(Succeed())
		g.Expect(record.Kind).To(Equal(auto.RecordSkipped))
		g.Expect(record.Session).To(Equal(sessionID))
		g.Expect(record.Turn).To(Equal("a-turn"))
		g.Expect(record.Reason).To(Equal(auto.StaleModSkipReason))
		g.Expect(record.Reason).To(ContainSubstring("restart the session"))
	}
}
