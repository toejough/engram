package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/debuglog"
)

func Test855d4a1c_Moment_TheCommandReadsTheEventOfEachToolCue(t *testing.T) {
	t.Parallel()

	events := map[string]struct{ event, line string }{
		"before-tool": {`{"tool":"Bash","input":{"command":"make ci"}}`, `Agent used Bash with {"command":"make ci"}`},
		"failure": {
			`{"tool":"Bash","input":{"command":"make ci"},"error":"exit 1"}`,
			`Agent used Bash with {"command":"make ci"}` + "\nAgent used Bash failed: exit 1",
		},
		"batch-done": {
			`{"calls":[{"tool":"Bash","input":{"command":"make ci"},"response":"ok"}]}`,
			`Agent used Bash with {"command":"make ci"}` + "\nAgent used Bash returned: ok",
		},
	}

	for cue, spec := range events {
		t.Run(cue, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			var out bytes.Buffer

			err := cli.RunAutoMoment(context.Background(), cli.AutoMomentArgs{SessionsDir: "/sessions"},
				cli.AutoMomentDeps{
					Stdin: strings.NewReader(`{"session":"` + sessionID + `","cue":"` + cue + `","origin":"composer",` +
						`"text":"Run the build.","event":` + spec.event + `}`),
					ReadRecord: func(string, string) []byte { return nil },
				}, &out)
			g.Expect(err).NotTo(HaveOccurred())

			var step struct {
				Request struct {
					Prompt string `json:"prompt"`
				} `json:"request"`
			}

			g.Expect(json.Unmarshal(out.Bytes(), &step)).To(Succeed(), "the output is one JSON object: %q", out.String())
			g.Expect(step.Request.Prompt).To(Equal("<window>\nUser: Run the build.\n" + spec.line + "\n</window>"))
		})
	}
}

//go:generate impgen cli.RecordRead --dependency --import-path github.com/toejough/engram/internal/cli

func Test855d4a1e_Moment_APromptOfAnOriginTheUserDoesNotSendIsJudgedNothing(t *testing.T) {
	t.Parallel()

	for _, origin := range []string{"task-notification", "peer", ""} {
		t.Run("origin "+origin, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			read, _ := MockRecordRead(t)
			done := make(chan error, 1)

			var out bytes.Buffer

			go func() {
				done <- cli.RunAutoMoment(context.Background(), cli.AutoMomentArgs{SessionsDir: "/sessions"},
					cli.AutoMomentDeps{Stdin: momentInput(origin), ReadRecord: read}, &out)
			}()

			select {
			case err := <-done:
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(out.String()).To(BeEmpty())
			case <-time.After(mockTimeout):
				t.Fatal("the moment did not finish: it read the record of a session of an origin the user does not send")
			}
		})
	}

	t.Run("an origin the user sends is read for and judged", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		read, readCalls := MockRecordRead(t)
		done := make(chan error, 1)

		var out bytes.Buffer

		go func() {
			done <- cli.RunAutoMoment(context.Background(), cli.AutoMomentArgs{SessionsDir: "/sessions"},
				cli.AutoMomentDeps{Stdin: momentInput("composer"), ReadRecord: read}, &out)
		}()

		go func() { readCalls.ArgsShould(Equal("/sessions"), Equal(sessionID)).Return(nil) }()

		select {
		case err := <-done:
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(out.String()).To(HavePrefix(`{"request":`), "the mod knows a request by its first characters")
		case <-time.After(mockTimeout):
			t.Fatal("the moment did not finish: the record of the session was not read once from the directory")
		}
	})
}

func Test855d4a1f_Moment_ASessionWithNoTranscriptIsJudgedWithoutConversationLines(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var out bytes.Buffer

	err := cli.RunAutoMoment(context.Background(), cli.AutoMomentArgs{SessionsDir: "/sessions"},
		cli.AutoMomentDeps{
			Stdin:      momentInput("composer"),
			ReadRecord: func(string, string) []byte { return nil },
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())

	var step struct {
		Request struct {
			Prompt string `json:"prompt"`
		} `json:"request"`
	}

	g.Expect(json.Unmarshal(out.Bytes(), &step)).To(Succeed(), "the output is one JSON object: %q", out.String())
	g.Expect(step.Request.Prompt).To(Equal("<window>\n" +
		"User: Prepare a test run against the Zorblax staging database tonight.\n</window>"))
}

func Test855d4a1g_Moment_AnInputThatIsNotAJSONObjectFailsTheCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var out bytes.Buffer

	err := cli.RunAutoMoment(context.Background(), cli.AutoMomentArgs{SessionsDir: "/sessions"},
		cli.AutoMomentDeps{
			Stdin:      strings.NewReader("not a JSON object"),
			ReadRecord: func(string, string) []byte { return nil },
		}, &out)

	g.Expect(err).To(MatchError(ContainSubstring("auto moment: reading the moment")))
	g.Expect(out.String()).To(BeEmpty())
}

func Test855d4a1h_Moment_AMomentThatPrintsNothingForTheEndOfATurnLogsIt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var logged, out bytes.Buffer

	ctx := debuglog.WithLogger(context.Background(),
		debuglog.New(&logged, "engram", func() time.Time { return recordedAt }))

	err := cli.RunAutoMoment(ctx, cli.AutoMomentArgs{SessionsDir: "/sessions"},
		cli.AutoMomentDeps{
			Stdin: strings.NewReader(`{"session":"` + sessionID + `","cue":"turn-end","origin":"task-notification",` +
				`"turn":"` + turnID + `","text":"a notice","answer":"Understood."}`),
			ReadRecord: func(string, string) []byte { return nil },
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(BeEmpty())

	lines := strings.Split(strings.TrimSpace(logged.String()), "\n")

	g.Expect(lines).To(HaveLen(1), "a line for the moment")

	if len(lines) == 1 {
		g.Expect(lines[0]).To(HaveSuffix("[engram] auto.moment: session=" + sessionID + " nothing"))
	}
}

// momentInput is the JSON object that the memory mod hands to the moment for a prompt of an origin.
func momentInput(origin string) *strings.Reader {
	return strings.NewReader(`{"session":"` + sessionID + `","cue":"prompt","origin":"` + origin +
		`","text":"Prepare a test run against the Zorblax staging database tonight."}`)
}
