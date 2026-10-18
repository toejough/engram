package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/auto"
	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
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

			var delivered bytes.Buffer

			deliverErr := cli.RunAutoDeliver(context.Background(), cli.AutoDeliverArgs{VaultPath: vault},
				cli.AutoDeliverDeps{
					Stdin:    deliveryOfOrigin(origin),
					Activate: noActivation,
				}, &delivered, io.Discard)
			g.Expect(deliverErr).NotTo(HaveOccurred())
			g.Expect(delivered.String()).To(HavePrefix("[engram memory]"), "the block of the note")

			// the turn of a headless prompt is judged here because the user asked for it (793d4m1c)
			for _, cue := range []string{"prompt", "turn-end"} {
				var judged bytes.Buffer

				momentErr := cli.RunAutoMoment(context.Background(),
					cli.AutoMomentArgs{SessionsDir: recordOfAPrompt(t), CaptureSDK: true},
					cli.AutoMomentDeps{Stdin: momentOf(origin, cue), ReadRecord: readRecordFile}, &judged)
				g.Expect(momentErr).NotTo(HaveOccurred(), cue)
				g.Expect(judged.String()).To(HavePrefix(`{"request":`), "the request for the model, with the cue "+cue)
			}
		})
	}
}

func Test793d4m1b_Auto_APromptOfAnyOtherOriginIsDeliveredNothingAndItsTurnIsJudgedNothing(t *testing.T) {
	t.Parallel()

	sessions := recordOfAPrompt(t)

	// check dispatches the delivery and the moment for an origin, and reports what became of them
	check := func(fail func(string, ...any), origin string) {
		var delivered bytes.Buffer

		reads := 0
		deliverErr := cli.RunAutoDeliver(context.Background(),
			cli.AutoDeliverArgs{VaultPath: "/vault", SessionsDir: sessions},
			cli.AutoDeliverDeps{
				Stdin:    deliveryOfOrigin(origin),
				Activate: noActivation,
				ReadDir: func(path string) ([]fs.DirEntry, error) {
					reads++

					return os.ReadDir(path)
				},
				ReadFile: func(path string) ([]byte, error) {
					reads++

					return os.ReadFile(path)
				},
			}, &delivered, io.Discard)

		if deliverErr != nil || delivered.Len() != 0 {
			fail("the origin %q was delivered %d characters, failing with %v", origin, delivered.Len(), deliverErr)
		}

		for _, cue := range []string{"prompt", "turn-end"} {
			var judged bytes.Buffer

			momentErr := cli.RunAutoMoment(context.Background(),
				cli.AutoMomentArgs{SessionsDir: sessions, CaptureSDK: true},
				cli.AutoMomentDeps{
					Stdin: momentOf(origin, cue),
					ReadRecord: func(dir, session string) []byte {
						reads++

						return readRecordFile(dir, session)
					},
				}, &judged)

			if momentErr != nil || judged.Len() != 0 {
				fail("the origin %q with the cue %s was judged %d characters, failing with %v", origin, cue, judged.Len(),
					momentErr)
			}
		}

		if reads != 0 {
			fail("the origin %q made %d reads", origin, reads)
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

// backupCandidate is the candidate of a second note of the vault about the same database, as the carry holds it.
func backupCandidate() auto.Candidate {
	return auto.Candidate{
		ID: backupNoteName, Kind: auto.KindNote, Type: "fact", Date: "2026-10-17",
		Situation: "testing against the Zorblax staging database",
		Text: "Information learned: when in testing against the Zorblax staging database, " +
			"the Zorblax staging database is backed up every Sunday at 03:40 UTC.",
	}
}

// deliveryFields are the fields of the request the memory mod hands to the delivery for a prompt the user typed, in a
// session: its reply delivers each of the candidates the carry holds, a note as covered and a passage as evidence.
func deliveryFields(session string, candidates ...auto.Candidate) map[string]any {
	return deliveryFieldsWith(session, repliesFor(candidates), candidates...)
}

// deliveryFieldsWith is deliveryFields with the reply the candidate judgement gave.
func deliveryFieldsWith(session, reply string, candidates ...auto.Candidate) map[string]any {
	return map[string]any{
		"text": "Prepare a test run against the Zorblax staging database tonight.", "session": session,
		"origin": "composer", "cue": "prompt", "reply": reply,
		"carry": auto.Carry{Cue: "prompt", Depth: auto.DepthGlance, Window: "w", Candidates: candidates},
	}
}

// deliveryOfOrigin is the request of a delivery of a prompt of an origin, over the sessions recordOfAPrompt writes.
func deliveryOfOrigin(origin string) io.Reader {
	fields := deliveryFields(sessionID, resetCandidate())
	fields["origin"] = origin

	return requestJSON(fields)
}

// deliveryRequest is the request of a delivery, as a reader.
func deliveryRequest(session string, candidates ...auto.Candidate) io.Reader {
	return requestJSON(deliveryFields(session, candidates...))
}

// deliveryRequestWith is the fields of a delivery whose reply is the one given.
func deliveryRequestWith(session, reply string, candidates ...auto.Candidate) map[string]any {
	return deliveryFieldsWith(session, reply, candidates...)
}

// freezePassage is the candidate of a passage of an earlier conversation, as the carry holds it.
func freezePassage() auto.Candidate {
	return auto.Candidate{
		ID: "/chunks/an-earlier-session.jsonl#c3", Kind: auto.KindPassage, Date: "2026-10-17",
		Text: "the user said the production cutover is frozen on the last Friday of every month",
	}
}

// lastUsed returns the date a note of a vault was last marked as used, or nothing when it never was.
func lastUsed(tb testing.TB, vault, note string) string {
	tb.Helper()

	data, err := os.ReadFile(filepath.Join(vault, embed.SidecarPath(note)))
	if err != nil {
		tb.Fatalf("reading the sidecar of %s: %v", note, err)
	}

	var sidecar embed.Sidecar

	err = json.Unmarshal(data, &sidecar)
	if err != nil {
		tb.Fatalf("decoding the sidecar of %s: %v", note, err)
	}

	return sidecar.LastUsed
}

// momentOf is the JSON object that the memory mod hands to the moment for the turn of a prompt of an origin, at a cue,
// over the sessions recordOfAPrompt writes.
func momentOf(origin, cue string) io.Reader {
	return requestJSON(map[string]any{
		"session": sessionID, "cue": cue, "origin": origin, "turn": "a-turn", "answer": "On Sundays.",
		"text": "When is the Zorblax staging database reset?",
	})
}

// noActivation is an activation that marks nothing, for a test that does not look at the vault.
func noActivation(context.Context, string, auto.Candidate) {}

// originRequest is the JSON object that the memory mod hands to a command: the field given, the session of the record
// recordOfAPrompt writes, and the origin.
func originRequest(origin, field, value string) io.Reader {
	return requestOf(map[string]string{field: value, "session": sessionID, "origin": origin})
}

// readRecordFile reads the record of a session from a directory of records, or nothing.
func readRecordFile(dir, session string) []byte {
	record, err := os.ReadFile(filepath.Join(dir, session+".jsonl"))
	if err != nil {
		return nil
	}

	return record
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

// repliesFor is the reply of a candidate judgement that delivers each of the candidates: covered for a note and
// evidence for a passage.
func repliesFor(candidates []auto.Candidate) string {
	verdicts := make([]map[string]string, 0, len(candidates))

	for _, candidate := range candidates {
		verdict := "covered"
		if candidate.Kind == auto.KindPassage {
			verdict = "evidence"
		}

		verdicts = append(verdicts, map[string]string{"id": candidate.ID, "verdict": verdict})
	}

	encoded, err := json.Marshal(map[string]any{"candidates": verdicts})
	if err != nil {
		panic(err)
	}

	return string(encoded)
}

// requestJSON is a request to a command, with the fields given, as a reader.
func requestJSON(fields map[string]any) io.Reader {
	encoded, err := json.Marshal(fields)
	if err != nil {
		panic(err)
	}

	return bytes.NewReader(encoded)
}

// requestOf is the JSON object that the memory mod hands to a command, with the fields given, as a reader.
func requestOf(fields map[string]string) io.Reader {
	encoded, err := json.Marshal(fields)
	if err != nil {
		panic(err)
	}

	return bytes.NewReader(encoded)
}

// resetCandidate is the candidate of the note zorblaxResetNote, as the carry holds it.
func resetCandidate() auto.Candidate {
	return auto.Candidate{
		ID: resetNoteName, Kind: auto.KindNote, Type: "fact", Date: "2026-10-17",
		Situation: "testing against the Zorblax staging database",
		Text: "Information learned: when in testing against the Zorblax staging database, " +
			"the Zorblax staging database is reset every Sunday at 03:40 UTC.",
	}
}

// userPrompt is the request that the memory mod hands to the delivery for a prompt the user typed, in a session, whose
// reply delivers the note zorblaxResetNote.
func userPrompt(text, session string) io.Reader {
	fields := deliveryFields(session, resetCandidate())
	fields["text"] = text

	return requestJSON(fields)
}

// withoutReply takes the reply and the carry out of a request: the prompt was no recall moment.
func withoutReply(fields map[string]any) map[string]any {
	delete(fields, "reply")
	delete(fields, "carry")

	return fields
}
