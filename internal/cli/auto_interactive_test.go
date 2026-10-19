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

	"github.com/toejough/engram/internal/cli"
)

func Test855d4g1e_RunAutoMomentAndJudge_AnInteractiveSessionsTypedTurnIsInTheWindow(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// the excerpt is shaped as the record of an interactive session: its prompts are stamped human, and a notification
	// of the harness is stamped task-notification
	record, readErr := os.ReadFile(filepath.Join("testdata", "interactive_session.jsonl"))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	sessions := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(sessions, sessionID+".jsonl"), record, 0o600)).To(Succeed())

	const typed = "No - build with make zorblax-build, never make all."

	var moment bytes.Buffer

	momentErr := cli.RunAutoMoment(context.Background(), cli.AutoMomentArgs{SessionsDir: sessions, RecordsDir: t.TempDir()},
		cli.AutoMomentDeps{
			Stdin: bytes.NewReader([]byte(`{"origin":"composer","cue":"turn-end","session":"` + sessionID + `","turn":"t2",` +
				`"text":"` + typed + `","answer":"Built with make zorblax-build."}`)),
			ReadRecord: func(dir, session string) []byte {
				data, err := os.ReadFile(filepath.Join(dir, session+".jsonl"))
				if err != nil {
					return nil
				}

				return data
			},
			Now:         time.Now,
			WriteRecord: func(string, []byte) error { return nil },
		}, &moment)
	g.Expect(momentErr).NotTo(HaveOccurred())
	g.Expect(moment.String()).To(ContainSubstring(typed), "the moment's window holds the user's text")
	g.Expect(moment.String()).NotTo(ContainSubstring("Background task"), "a task notification is no prompt")

	var judged bytes.Buffer

	judgeErr := cli.RunAutoJudge(context.Background(), cli.AutoJudgeArgs{VaultPath: t.TempDir(), SessionsDir: sessions},
		cli.AutoJudgeDeps{
			Stdin: bytes.NewReader([]byte(`{"origin":"composer","session":"` + sessionID + `","turn":"t2","text":"` + typed +
				`",` + learnTrue + `"answer":"Built with make zorblax-build."}`)),
			Query:    queryOver(newQueryDeps(newInMemoryFS())),
			ReadDir:  os.ReadDir,
			ReadFile: os.ReadFile,
		}, &judged)
	g.Expect(judgeErr).NotTo(HaveOccurred())

	var step struct {
		Request struct {
			Prompt string `json:"prompt"`
		} `json:"request"`
	}

	g.Expect(json.Unmarshal(judged.Bytes(), &step)).To(Succeed(), "a learn moment prints the request: %q", judged.String())
	g.Expect(step.Request.Prompt).To(ContainSubstring("User: "+typed), "the window holds the user's text")
	g.Expect(step.Request.Prompt).NotTo(ContainSubstring("Background task"))
}
