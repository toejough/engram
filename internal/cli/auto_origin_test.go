package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

func Test793d4m1a_Auto_APromptOfAnOriginTheUserSendsIsDeliveredForAndItsTurnIsJudged(t *testing.T) {
	t.Parallel()

	for _, origin := range []string{"composer", "bridge", "slack-ping", "sdk"} {
		t.Run(origin, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := t.TempDir()
			memFS := newInMemoryFS()
			plantNoteWithSidecar(t, memFS, vault, "1.2026-10-17.zorblax-staging-reset.md", zorblaxResetNote)

			var delivered, judged bytes.Buffer

			deliverErr := cli.RunAutoDeliver(context.Background(), cli.AutoDeliverArgs{VaultPath: vault},
				cli.AutoDeliverDeps{
					Stdin: originRequest(origin, "text", "When is the Zorblax staging database reset?"),
					Query: queryOver(newQueryDeps(memFS)),
				}, &delivered)
			g.Expect(deliverErr).NotTo(HaveOccurred())
			g.Expect(delivered.String()).To(HavePrefix("[engram memory]"), "the block of the note")

			judgeErr := cli.RunAutoJudge(context.Background(),
				cli.AutoJudgeArgs{VaultPath: vault, SessionsDir: recordOfAPrompt(t)},
				cli.AutoJudgeDeps{
					Stdin:    originRequest(origin, "answer", "On Sundays."),
					Query:    queryOver(newQueryDeps(memFS)),
					ReadDir:  os.ReadDir,
					ReadFile: os.ReadFile,
				}, &judged)
			g.Expect(judgeErr).NotTo(HaveOccurred())
			g.Expect(judged.String()).To(HavePrefix(`{"model":`), "the request for the model")
		})
	}
}

func Test793d4m1b_Auto_APromptOfAnyOtherOriginIsDeliveredNothingAndItsTurnIsJudgedNothing(t *testing.T) {
	t.Parallel()

	sessions := recordOfAPrompt(t)

	// check dispatches the delivery and the judgement for an origin, and reports what became of them
	check := func(fail func(string, ...any), origin string) {
		lookups := 0
		lookup := func(context.Context, cli.QueryArgs, io.Writer) error {
			lookups++

			return nil
		}

		var delivered, judged bytes.Buffer

		deliverErr := cli.RunAutoDeliver(context.Background(),
			cli.AutoDeliverArgs{VaultPath: "/vault", SessionsDir: sessions},
			cli.AutoDeliverDeps{
				Stdin:    originRequest(origin, "text", "When is the Zorblax staging database reset?"),
				Query:    lookup,
				ReadDir:  os.ReadDir,
				ReadFile: os.ReadFile,
			}, &delivered)
		judgeErr := cli.RunAutoJudge(context.Background(), cli.AutoJudgeArgs{VaultPath: "/vault", SessionsDir: sessions},
			cli.AutoJudgeDeps{
				Stdin:    originRequest(origin, "answer", "On Sundays."),
				Query:    lookup,
				ReadDir:  os.ReadDir,
				ReadFile: os.ReadFile,
			}, &judged)

		if deliverErr != nil || judgeErr != nil {
			fail("the origin %q failed a command: delivery %v, judgement %v", origin, deliverErr, judgeErr)
		}

		if delivered.Len() != 0 || judged.Len() != 0 {
			fail("the origin %q was delivered %d characters and judged %d characters", origin, delivered.Len(), judged.Len())
		}

		if lookups != 0 {
			fail("the origin %q made %d lookups", origin, lookups)
		}
	}

	t.Run("the origins that Claude Code names", func(t *testing.T) {
		t.Parallel()

		for _, origin := range []string{
			"task-notification", "scheduled-trigger", "peer", "peer-send-message", "projects-relay", "channel",
			"coordinator", "observer", "observer-activity", "auto-continuation", "unclassified", "plugin", "",
		} {
			check(t.Fatalf, origin)
		}
	})

	t.Run("any origin", func(t *testing.T) {
		t.Parallel()

		rapid.Check(t, func(rt *rapid.T) {
			origin := rapid.String().
				Filter(func(text string) bool {
					return !slices.Contains([]string{"composer", "bridge", "slack-ping", "sdk"}, text)
				}).Draw(rt, "origin")

			check(rt.Fatalf, origin)
		})
	})
}

// originRequest is the JSON object that the memory mod hands to a command: the field given, the session of the record
// recordOfAPrompt writes, and the origin.
func originRequest(origin, field, value string) io.Reader {
	encoded, err := json.Marshal(map[string]string{field: value, "session": sessionID, "origin": origin})
	if err != nil {
		panic(err)
	}

	return bytes.NewReader(encoded)
}

// recordOfAPrompt writes the record of a session whose last turn began with a prompt, and returns the directory of the
// records.
func recordOfAPrompt(tb testing.TB) string {
	tb.Helper()

	dir := tb.TempDir()
	record := `{"type":"user","message":{"role":"user","content":"When is the Zorblax staging database reset?"}}` + "\n"

	err := os.WriteFile(filepath.Join(dir, sessionID+".jsonl"), []byte(record), 0o600)
	if err != nil {
		tb.Fatalf("writing the record: %v", err)
	}

	return dir
}

// userPrompt is the request that the memory mod hands to the delivery for a prompt the user typed, in a session.
func userPrompt(text, session string) io.Reader {
	encoded, err := json.Marshal(map[string]string{"text": text, "session": session, "origin": "composer"})
	if err != nil {
		panic(err)
	}

	return bytes.NewReader(encoded)
}
