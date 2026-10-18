package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

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

// momentInput is the JSON object that the memory mod hands to the moment for a prompt of an origin.
func momentInput(origin string) *strings.Reader {
	return strings.NewReader(`{"session":"` + sessionID + `","cue":"prompt","origin":"` + origin +
		`","text":"Prepare a test run against the Zorblax staging database tonight."}`)
}
